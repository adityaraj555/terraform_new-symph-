package main

import (
	"context"

	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
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
	FlowType   string                 `json:"flowType"`
}

const DBSecretARN = "DBSecretARN"

func Handler(ctx context.Context, Request RequestBody) (interface{}, error) {
	var err error
	ctx = log_config.SetTraceIdInContext(ctx, Request.OrderId, Request.WorkflowId)

	log.Infof(ctx, "Datastorelambda reached... %+v", Request)
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
			return map[string]interface{}{"status": "failed"}, error_handler.NewServiceError(error_codes.ErrorInsertingWorkflowDataInDB, err.Error())
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
			return map[string]interface{}{"status": "failed"}, error_handler.NewServiceError(error_codes.ErrorUpdatingWorkflowDataInDB, err.Error())
		}
	case "updateFlowType":
		query := bson.M{"_id": Request.WorkflowId}
		setrecord := bson.M{
			"$set": bson.M{
				"flowType": Request.FlowType,
			}}

		err = commonHandler.DBClient.UpdateDocumentDB(ctx, query, setrecord, documentDB_client.WorkflowDataCollection)
		if err != nil {
			log.Errorf(ctx, "Unable to UpdateDocumentDB error = %s", err)
			return map[string]interface{}{"status": "failed"}, error_handler.NewServiceError(error_codes.ErrorUpdatingWorkflowDataInDB, err.Error())
		}
	case "sfnSummary":
		var listOfOrderIds, listOfWorkflowIds []string
		if list, ok := Request.Input["orderIds"]; ok {
			if value, ok := list.([]string); ok {
				listOfOrderIds = value
			}
		}

		if list, ok := Request.Input["workflowIds"]; ok {
			if value, ok := list.([]string); ok {
				listOfWorkflowIds = value
			}
		}
		log.Infof(ctx, "Filter: %+v, %+v, %s", listOfOrderIds, listOfWorkflowIds, Request.Input["source"].(string))
		response, err := commonHandler.DBClient.FetchWorkflowExecutionDataByListOfWorkflows(ctx, Request.Input["source"].(string), listOfWorkflowIds, listOfOrderIds)
		if err != nil {
			log.Errorf(ctx, "Unable to UpdateDocumentDB error = %s", err)
			return map[string]interface{}{"status": "failed"}, error_handler.NewServiceError(error_codes.ErrorUpdatingWorkflowDataInDB, err.Error())
		}
		log.Infof(ctx, "Response: %+v", response)
		return response, nil
	}

	log.Info(ctx, "Datastorelambda successful...")
	return map[string]interface{}{"status": Success}, nil
}

func notificationWrapper(ctx context.Context, req RequestBody) (interface{}, error) {
	resp, err := Handler(ctx, req)
	if err != nil {
		cerr := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(cerr.GetErrorCode(), req.OrderId, req.WorkflowId, "", "datastore", err.Error(), nil)
	}
	return resp, err
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(false, false, true, true, false)
	lambda.Start(notificationWrapper)
}

func handleTimeout(ctx context.Context, req RequestBody) error {
	wfExecData, err := commonHandler.DBClient.FetchWorkflowExecutionData(ctx, req.WorkflowId)
	if err != nil {
		//running,
		return error_handler.NewServiceError(error_codes.ErrorFetchingWorkflowExecutionDataFromDB, err.Error())
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
	filter, update := commonHandler.DBClient.BuildQueryForCallBack(ctx, documentDB_client.UpdateWorkflowExecutionSteps, "failure", req.WorkflowId, timedOutStep.StepId, timedOutStep.TaskName, nil)
	err = commonHandler.DBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.WorkflowDataCollection)
	if err != nil {
		log.Error(ctx, "error updating db", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorUpdatingWorkflowDataInDB, err.Error())
	}

	//update StepExecutionDataBody
	filter, update = commonHandler.DBClient.BuildQueryForCallBack(ctx, documentDB_client.UpdateStepExecution, "failure", req.WorkflowId, timedOutStep.StepId, timedOutStep.TaskName, nil)
	err = commonHandler.DBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.StepsDataCollection)
	if err != nil {
		log.Error(ctx, "error updating db", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorUpdatingStepsDataInDB, err.Error())
	}
	commonHandler.SlackClient.SendErrorMessage(error_codes.StepFunctionTaskTimedOut, req.OrderId, req.WorkflowId, "datastore", timedOutStep.TaskName, "Task Timed Out", map[string]string{
		"Task":   timedOutStep.TaskName,
		"StepId": timedOutStep.StepId,
	})
	return nil
}
