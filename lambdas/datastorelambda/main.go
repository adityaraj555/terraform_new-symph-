package main

import (
	"context"

	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"

	"go.mongodb.org/mongo-driver/bson"
)

var awsClient aws_client.AWSClient
var newDBClient *documentDB_client.DocDBClient

const (
	Success    = "success"
	Inprogress = "inprogress"
	Finished   = "finished"
	loglevel   = "info"
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
	ctx = log_config.SetTraceIdInContext(ctx, Request.OrderId, Request.WorkflowId)
	switch Request.Action {
	case "insert":
		var data documentDB_client.WorkflowExecutionDataBody
		data.CreatedAt = time.Now().Unix()
		data.OrderId = Request.OrderId
		data.WorkflowId = Request.WorkflowId
		data.Status = Inprogress
		data.InitialInput = Request.Input
		data.StepsPassedThrough = []documentDB_client.StepsPassedThroughBody{}
		err = newDBClient.InsertWorkflowExecutionData(ctx, data)
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
		err = newDBClient.UpdateDocumentDB(ctx, query, update, documentDB_client.WorkflowDataCollection)
		if err != nil {
			return map[string]interface{}{"status": "failed"}, err
		}
	}

	return map[string]interface{}{"status": Success}, nil
}

func main() {
	log_config.InitLogging(loglevel)
	if newDBClient == nil {
		SecretARN := os.Getenv(DBSecretARN)
		log.Info(context.Background(), "fetching db secrets")
		secrets, err := awsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
		if err != nil {
			log.Error(context.Background(), "Unable to fetch DocumentDb in secret")
		}
		newDBClient = documentDB_client.NewDBClientService(secrets)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = newDBClient.DBClient.Connect(ctx)
		if err != nil {
			log.Error(context.Background(), err)
		}
	}
	lambda.Start(Handler)
}
