package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
)

type CreateMsg struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required"`
	Message string `json:"message" validate:"required"`
}

var validate *validator.Validate = validator.New()

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received req %#v", req)

	switch req.HTTPMethod {
	case "GET":
		return processGet(ctx, req)
	case "POST":
		return processPost(ctx, req)
	case "DELETE":
		return processDelete(ctx, req)
	default:
		return clientError(http.StatusMethodNotAllowed)
	}
}

func processGet(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return processGetMsgs(ctx)
	} else {
		return processGetMsg(ctx, id)
	}
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {

	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	log.Println(err.Error())

	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func processGetMsg(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received GET todo request with id = %s", id)

	todo, err := getItem(ctx, id)
	if err != nil {
		return serverError(err)
	}

	if todo == nil {
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(todo)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched todo item %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func processGetMsgs(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET todos request")

	msgs, err := listItems(ctx)
	if err != nil {
		return serverError(err)
	}

	json, err := json.Marshal(msgs)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched todos: %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func processPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var createMsg CreateMsg
	err := json.Unmarshal([]byte(req.Body), &createMsg)
	if err != nil {
		log.Printf("Can't unmarshal body: %v", err)
		return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&createMsg)
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received POST request with item: %+v", createMsg)

	res, err := insertItem(ctx, createMsg)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Inserted new todo: %+v", res)

	json, err := json.Marshal(res)
	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       string(json),
		Headers: map[string]string{
			"Location": fmt.Sprintf("/todo/%s", res.Id),
		},
	}, nil
}

func processDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received DELETE request with id = %s", id)

	msg, err := deleteItem(ctx, id)
	if err != nil {
		return serverError(err)
	}

	if msg == nil {
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(msg)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully deleted todo item %+v", msg)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}
