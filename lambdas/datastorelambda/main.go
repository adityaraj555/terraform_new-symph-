package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
)

var awsClient aws_client.AWSClient

const Success = "success"

type RequestBody struct {
	OrderId    string `json:"orderId"`
	WorkflowId string ` json:"workflowId"`
}

const DBSecretARN = "DBSecretARN"

func Handler(ctx context.Context, Request RequestBody) (map[string]interface{}, error) {
	SecretARN := os.Getenv(DBSecretARN)
	secrets, err := awsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
	NewDBClient := documentDB_client.NewDBClientService(secrets)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = NewDBClient.DBClient.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var data documentDB_client.DataStoreBody
	data.CreatedAt = time.Now()
	data.OrderId = Request.OrderId
	data.WorkflowId = Request.WorkflowId

	err = NewDBClient.DataStoreInsertion(data)
	if err != nil {
		return map[string]interface{}{"status": "failed"}, err
	}
	return map[string]interface{}{"status": Success}, nil
}

func main() {

	lambda.Start(Handler)
}
