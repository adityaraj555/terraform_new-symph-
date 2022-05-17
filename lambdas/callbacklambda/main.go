package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/enums"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/validator"
)

var commonHandler common_handler.CommonHandler

type RequestBody struct {
	Status      enums.TaskStatus       `json:"status" validate:"required,taskStatus"`
	Message     string                 `json:"message"`
	MessageCode interface{}            `json:"messageCode"`
	CallbackID  string                 `json:"callbackId" validate:"required"`
	Response    map[string]interface{} `json:"response"`
}

const DBSecretARN = "DBSecretARN"
const success = "success"
const failure = "failure"
const rework = "rework"
const isReworkRequired = "isReworkRequired"
const loglevel = "info"

func Handler(ctx context.Context, CallbackRequest RequestBody) (map[string]interface{}, error) {
	var err error

	if err := validator.ValidateCallBackRequest(ctx, CallbackRequest); err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	StepExecutionData, err := commonHandler.DBClient.FetchStepExecutionData(ctx, CallbackRequest.CallbackID)

	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	var stepstatus string = failure
	if CallbackRequest.Status.String() == rework {
		CallbackRequest.Response[isReworkRequired] = true
	} else {
		CallbackRequest.Response[isReworkRequired] = false
	}
	if CallbackRequest.Status.String() == success || CallbackRequest.Status.String() == rework {
		stepstatus = success
		byteData, _ := json.Marshal(CallbackRequest.Response)
		jsonResponse := string(byteData)
		err = commonHandler.AwsClient.CloseWaitTask(ctx, success, StepExecutionData.TaskToken, jsonResponse, "", "")
	} else {
		messageCode := CallbackRequest.MessageCode.(string)
		err = commonHandler.AwsClient.CloseWaitTask(ctx, failure, StepExecutionData.TaskToken, "", CallbackRequest.Message, messageCode)
	}
	if err != nil {
		log.Error(ctx, "Error Calling CloseWaitTask", err)
		return map[string]interface{}{"status": failure}, err
	}
	filter, query := commonHandler.DBClient.BuildQueryForCallBack(ctx, documentDB_client.UpdateStepExecution, stepstatus, StepExecutionData.WorkflowId, StepExecutionData.StepId, StepExecutionData.TaskName, CallbackRequest.Response)
	err = commonHandler.DBClient.UpdateDocumentDB(ctx, filter, query, documentDB_client.StepsDataCollection)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	filter, query = commonHandler.DBClient.BuildQueryForCallBack(ctx, documentDB_client.UpdateWorkflowExecutionSteps, stepstatus, StepExecutionData.WorkflowId, StepExecutionData.StepId, StepExecutionData.TaskName, CallbackRequest.Response)
	err = commonHandler.DBClient.UpdateDocumentDB(ctx, filter, query, documentDB_client.WorkflowDataCollection)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	filter, query = commonHandler.DBClient.BuildQueryForCallBack(ctx, documentDB_client.UpdateWorkflowExecutionStatus, stepstatus, StepExecutionData.WorkflowId, StepExecutionData.StepId, StepExecutionData.TaskName, CallbackRequest.Response)
	err = commonHandler.DBClient.UpdateDocumentDB(ctx, filter, query, documentDB_client.WorkflowDataCollection)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	return map[string]interface{}{"status": success}, nil
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(true, false, true, true)
	lambda.Start(notificationWrapper)
}

func notificationWrapper(ctx context.Context, req RequestBody) (map[string]interface{}, error) {
	resp, err := Handler(ctx, req)
	if err != nil {
		commonHandler.SlackClient.SendErrorMessage("", req.CallbackID, "callback", err.Error())
	}
	return resp, err
}
