package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
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

var commonHandler common_handler.CommonHandler

const (
	StateMachineARN = "StateMachineARN"
	loglevel        = "info"
)

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(true, false, false, true)
	lambda.Start(notificationWrapper)
}

func notificationWrapper(ctx context.Context, sqsEvent events.SQSEvent) error {
	err := Handler(ctx, sqsEvent)
	if err != nil {
		commonHandler.SlackClient.SendErrorMessage("", "", "invokesfn", err.Error())
	}
	return err
}

func Handler(ctx context.Context, sqsEvent events.SQSEvent) (err error) {
	SFNStateMachineARN := os.Getenv(StateMachineARN)
	for _, message := range sqsEvent.Records {
		if err = validateInput(ctx, message.Body); err != nil {
			return err
		}
		ExecutionArn, err := commonHandler.AwsClient.InvokeSFN(&message.Body, &SFNStateMachineARN)
		log.Infof(ctx, "executionARN of the  above execution  %s", ExecutionArn)
		if err != nil {
			log.Error(ctx, err)
			return err
		}
	}
	return err
}

func validateInput(ctx context.Context, input string) error {
	log.Info(ctx, "input body:", input)
	req := sfnInput{}
	err := json.Unmarshal([]byte(input), &req)
	fmt.Println(req)
	if err != nil {
		log.Error(ctx, "invalid input for sfn", input)
		return err
	}

	if err := validator.ValidateInvokeSfnRequest(ctx, req); err != nil {
		log.Error(ctx, "error in validation: ", err)
		return err
	}

	return nil
}
