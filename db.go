package main

import (
	"context"

	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type Msg struct {
	Id      string `json:"id" dynamodbav:"id"`
	Name    string `json:"name" dynamodbav:"name"`
	Email   string `json:"email" dynamodbav:"email"`
	Message string `json:"message" dynamodbav:"message"`
}

const TableName = "Msgs"

var db dynamodb.Client

func init() {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	db = *dynamodb.NewFromConfig(sdkConfig)
}

func getItem(ctx context.Context, id string) (*Msg, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
	}

	log.Printf("Calling Dynamodb with input: %v", input)
	result, err := db.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}
	log.Printf("Executed GetItem DynamoDb successfully. Result: %#v", result)

	if result.Item == nil {
		return nil, nil
	}

	msg := new(Msg)
	err = attributevalue.UnmarshalMap(result.Item, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func insertItem(ctx context.Context, createMsg CreateMsg) (*Msg, error) {
	msg := Msg{
		Name:    createMsg.Name,
		Email:   createMsg.Email,
		Message: createMsg.Message,
		Id:      uuid.NewString(),
	}

	item, err := attributevalue.MarshalMap(msg)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	}

	res, err := db.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &msg)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

func deleteItem(ctx context.Context, id string) (*Msg, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		ReturnValues: types.ReturnValue(*aws.String("ALL_OLD")),
	}

	res, err := db.DeleteItem(ctx, input)
	if err != nil {
		return nil, err
	}

	if res.Attributes == nil {
		return nil, nil
	}

	msg := new(Msg)
	err = attributevalue.UnmarshalMap(res.Attributes, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func listItems(ctx context.Context) ([]Msg, error) {
	msgs := make([]Msg, 0)
	var token map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(TableName),
			ExclusiveStartKey: token,
		}

		result, err := db.Scan(ctx, input)
		if err != nil {
			return nil, err
		}

		var fetchedmsgs []Msg
		err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedmsgs)
		if err != nil {
			return nil, err
		}

		msgs = append(msgs, fetchedmsgs...)
		token = result.LastEvaluatedKey
		if token == nil {
			break
		}
	}

	return msgs, nil
}
