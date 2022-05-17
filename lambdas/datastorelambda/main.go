package main

import (
	"context"

	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"

	"go.mongodb.org/mongo-driver/bson"
)

var commonHandler common_handler.CommonHandler

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
		err = commonHandler.DBClient.InsertWorkflowExecutionData(ctx, data)
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
		err := commonHandler.DBClient.UpdateDocumentDB(ctx, query, update, documentDB_client.WorkflowDataCollection)
		if err != nil {
			return map[string]interface{}{"status": "failed"}, err
		}
	}

	return map[string]interface{}{"status": Success}, nil
}

func notificationWrapper(ctx context.Context, req RequestBody) (map[string]interface{}, error) {
	resp, err := Handler(ctx, req)
	if err != nil {
		commonHandler.SlackClient.SendErrorMessage("datastore", err.Error())
	}
	return resp, err
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(false, false, true, true)
	lambda.Start(notificationWrapper)
}
