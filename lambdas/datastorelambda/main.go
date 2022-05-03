package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"go.mongodb.org/mongo-driver/bson"
)

var awsClient aws_client.AWSClient
var newDBClient *documentDB_client.DocDBClient

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
	var err error

	switch Request.Action {
	case "insert":
		var data documentDB_client.WorkflowExecutionDataBody
		data.CreatedAt = time.Now().Unix()
		data.OrderId = Request.OrderId
		data.WorkflowId = Request.WorkflowId
		data.Status = Inprogress
		data.InitialInput = Request.Input
		data.StepsPassedThrough = []documentDB_client.StepsPassedThroughBody{}
		err = newDBClient.InsertWorkflowExecution(data)
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
		err = newDBClient.UpdateDocumentDB(query, update, documentDB_client.WorkflowDataCollection)
		if err != nil {
			return map[string]interface{}{"status": "failed"}, err
		}
	}

	return map[string]interface{}{"status": Success}, nil
}

func main() {
	if newDBClient == nil {
		SecretARN := os.Getenv(DBSecretARN)
		fmt.Println("fetching db secrets")
		secrets, err := awsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
		if err != nil {
			fmt.Println("Unable to fetch DocumentDb in secret")
		}
		newDBClient = documentDB_client.NewDBClientService(secrets)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = newDBClient.DBClient.Connect(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}
	lambda.Start(Handler)
}
