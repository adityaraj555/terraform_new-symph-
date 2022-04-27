package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
)

var awsClient aws_client.AWSClient

const StateMachineARN = "StateMachineARN"

func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context, sqsEvent events.SQSEvent) (err error) {
	SFNStateMachineARN := os.Getenv(StateMachineARN)
	for _, message := range sqsEvent.Records {
		ExecutionArn, err := awsClient.InvokeSFN(&message.Body, &SFNStateMachineARN)
		fmt.Printf("executionARN of the  above execution  %s", ExecutionArn)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return err
}
