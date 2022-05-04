package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
)

type sfnInput struct {
	Address struct {
		City      string  `json:"city"`
		Country   string  `json:"country"`
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
		State     string  `json:"state"`
		Street    string  `json:"street"`
		Zip       string  `json:"zip"`
	}
	OrderID  string `json:"orderId"`
	ReportID string `json:"reportId"`
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
		if err = validateInput(message.Body); err != nil {
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

func validateInput(input string) error {
	fmt.Print("input body:", input)
	req := sfnInput{}
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		log.Error("invalid input for sfn", input)
		return err
	}
	fmt.Printf("request input: %#v", req)
	if req.ReportID == emptyString {
		return errors.New("reportId cannot be empty")
	}

	if req.Address.City == emptyString || req.Address.Country == emptyString || req.Address.State == emptyString || req.Address.Zip == emptyString || req.Address.Street == emptyString {
		return errors.New("address fields(street, city, state, country, zip) missing")
	}

	return nil
}
