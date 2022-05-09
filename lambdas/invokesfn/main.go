package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/validator"
)

type sfnInput struct {
	Address struct {
		City      string  `json:"city" validate:"required"`
		Country   string  `json:"country" validate:"required"`
		Longitude float64 `json:"longitude" validate:"required"`
		Latitude  float64 `json:"latitude" validate:"required"`
		State     string  `json:"state" validate:"required"`
		Street    string  `json:"street" validate:"required"`
		Zip       string  `json:"zip" validate:"required"`
	}
	OrderID  string `json:"orderId"`
	ReportID string `json:"reportId" validate:"required"`
}

var awsClient aws_client.AWSClient

const (
	StateMachineARN = "StateMachineARN"
	emptyString     = ""
)

func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context, sqsEvent events.SQSEvent) (err error) {
	SFNStateMachineARN := os.Getenv(StateMachineARN)
	for _, message := range sqsEvent.Records {
		if err = validateInput(ctx, message.Body); err != nil {
			return err
		}
		ExecutionArn, err := awsClient.InvokeSFN(&message.Body, &SFNStateMachineARN)
		fmt.Printf("executionARN of the  above execution  %s", ExecutionArn)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return err
}

func validateInput(ctx context.Context, input string) error {
	log.Info("input body:", input)
	req := sfnInput{}
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		log.Error("invalid input for sfn", input)
		return err
	}

	if err := validator.ValidateInvokeSfnRequest(ctx, req); err != nil {
		log.Error("error in validation: ", err)
		return err
	}

	return nil
}
