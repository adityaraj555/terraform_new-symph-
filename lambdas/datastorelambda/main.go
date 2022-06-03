package main

import (
	"context"

	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
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
	Timedout   = "timedout"
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

	log.Info(ctx, "Datastorelambda reached...")
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
			log.Error(ctx, "Error while inserting workflowExecutionData, error: ", err.Error())
			return map[string]interface{}{"status": "failed"}, err
		}
	case "update":
		// handle timeout
		err := handleTimeout(ctx, Request)
		if err != nil {
			log.Error(ctx, "error handling timeout, error: ", err.Error())
			return map[string]interface{}{"status": "failed"}, err
		}
		update := bson.M{
			"$set": bson.M{
				"finishedAt": time.Now().Unix(),
				"status":     Finished,
			},
		}
		query := bson.M{"_id": Request.WorkflowId}
		err = commonHandler.DBClient.UpdateDocumentDB(ctx, query, update, documentDB_client.WorkflowDataCollection)
		if err != nil {
			log.Error(ctx, "Error while updating workflowExecutionData, error: ", err.Error())
			return map[string]interface{}{"status": "failed"}, err
		}
	}

	log.Info(ctx, "Datastorelambda successful...")
	return map[string]interface{}{"status": Success}, nil
}

func notificationWrapper(ctx context.Context, req RequestBody) (map[string]interface{}, error) {
	resp, err := Handler(ctx, req)
	if err != nil {
		commonHandler.SlackClient.SendErrorMessage(req.OrderId, req.WorkflowId, "datastore", err.Error(), nil)
	}
	return resp, err
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(false, false, true, true)
	lambda.Start(notificationWrapper)
}

func handleTimeout(ctx context.Context, req RequestBody) error {
	wfExecData, err := commonHandler.DBClient.FetchWorkflowExecutionData(ctx, req.WorkflowId)
	if err != nil {
		//running,
		return err
	}
	var timedOutStep *documentDB_client.StepsPassedThroughBody
	for _, state := range wfExecData.StepsPassedThrough {
		if state.Status == "running" {
			timedOutStep = &state
			break
		}
	}

	if timedOutStep == nil {
		return nil
	}
	log.Info(ctx, "task timed out: %s", timedOutStep.TaskName)

	//update stepsPassedThrough
	filter := bson.M{
		"_id":                       req.WorkflowId,
		"stepsPassedThrough.stepId": timedOutStep.StepId,
	}
	update := bson.M{
		"$set": bson.M{
			"stepsPassedThrough.$.status": "failure",
		},
	}

	err = commonHandler.DBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.WorkflowDataCollection)
	if err != nil {
		log.Error(ctx, "error updating db", err.Error())
		return err
	}

	//update StepExecutionDataBody
	filter = bson.M{
		"_id": timedOutStep.StepId,
	}
	update = bson.M{
		"$set": bson.M{
			"status":  "failure",
			"endtime": time.Now().Unix(),
		},
	}
	err = commonHandler.DBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.WorkflowDataCollection)
	if err != nil {
		log.Error(ctx, "error updating db", err.Error())
		return err
	}
	commonHandler.SlackClient.SendErrorMessage(req.OrderId, req.WorkflowId, "datastore", "Task Timed Out", map[string]string{
		"Task":   timedOutStep.TaskName,
		"StepId": timedOutStep.StepId,
	})
	return nil
}
