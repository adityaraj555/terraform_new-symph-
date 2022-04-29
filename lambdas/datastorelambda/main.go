package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"go.mongodb.org/mongo-driver/bson"
)

var awsClient aws_client.AWSClient

const (
	Success    = "success"
	Inprogress = "inprogress"
	Finished   = "finished"
)

type RequestBody struct {
	Input      map[string]interface{} `json:"input"`
	OrderId    string                 `json:"orderId"`
	WorkflowId string                 `json:"workflowId"`
	Action     string                 `json:"action"`
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

	switch Request.Action {
	case "insert":
		var data documentDB_client.WorkflowExecutionDataBody
		data.CreatedAt = time.Now().Unix()
		data.OrderId = Request.OrderId
		data.WorkflowId = Request.WorkflowId
		data.Status = Inprogress
		data.InitialInput = Request.Input
		err = NewDBClient.InsertWorkflowExecution(data)
		if err != nil {
			return map[string]interface{}{"status": "failed"}, err
		}
	case "update":
		update := bson.M{
			"$set": bson.M{
				"finishedAt": time.Now().Unix(),
				"status":     Finished,
			},
		}
		query := bson.M{"_id": Request.WorkflowId}
		err = NewDBClient.UpdateDocumentDB(query, update, documentDB_client.WorkflowDataCollection)
		if err != nil {
			return map[string]interface{}{"status": "failed"}, err
		}
	}

	return map[string]interface{}{"status": Success}, nil
}

func main() {

	lambda.Start(Handler)
}
