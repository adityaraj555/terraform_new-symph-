package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	b64 "encoding/base64"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/fatih/structs"
	"github.com/google/uuid"
	ctxlog "github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/lambdas/legacyupdate/status"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	success                              = "success"
	failure                              = "failure"
	logLevel                             = "info"
	region                               = "us-east-2"
	taskName                             = "EVMLJsonConverter_UploadToEvoss"
	envCalloutLambdaFunction             = "envCalloutLambdaFunction"
	envEvJsonConvertorEndpoint           = "envEvJsonConvertorEndpoint"
	envLegacyEndpoint                    = "envLegacyEndpoint"
	DBSecretARN                          = "DBSecretARN"
	legacyAuthKey                        = "TOKEN"
	RetriableError                       = "RetriableError"
	ConvertPropertyModelToEVJsonTaskName = "ConvertPropertyModelToEVJson"
	UploadMLJsonToEvossTaskName          = "UploadMLJsonToEvoss"
)

var (
	legacyStatusMap = map[string]string{}
	commonHandler   common_handler.CommonHandler
)

type eventData struct {
	ReportID              string `json:"reportId"`
	WorkflowID            string `json:"workflowId"`
	ImageMetaDataLocation string `json:"imageMetaDataLocation"`
}

func handler(ctx context.Context, eventData eventData) (map[string]interface{}, error) {
	ctx = log_config.SetTraceIdInContext(ctx, eventData.ReportID, eventData.WorkflowID)
	ctxlog.Info(ctx, "EVMLConverter Lambda Reached")

	var (
		err                                 error
		ok                                  bool
		finalTaskStepID                     string
		taskOutput                          interface{}
		propertyModelS3Path, evJsonLocation string
		legacyStatus                        string = "QCCompleted"
	)
	starttime := time.Now().Unix()
	stepID := uuid.New().String()
	StepExecutionData := documentDB_client.StepExecutionDataBody{
		StepId:     stepID,
		StartTime:  starttime,
		Input:      structs.Map(eventData),
		WorkflowId: eventData.WorkflowID,
		TaskName:   "EVMLJsonConverter_UploadToEvoss",
	}

	workflowData, err := commonHandler.DBClient.FetchWorkflowExecutionData(ctx, eventData.WorkflowID)
	if err != nil {
		ctxlog.Error(ctx, "Error in fetching workflow data from DocumentDb: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), err
	}

	ctxlog.Info(ctx, "Workflow Data Fetched from DocumentDb...")
	stepscount := len(workflowData.StepsPassedThrough)
	var lastCompletedTask documentDB_client.StepsPassedThroughBody
	for i := stepscount - 1; i >= 0; i-- {
		if workflowData.StepsPassedThrough[i].TaskName != taskName &&
			workflowData.StepsPassedThrough[i].TaskName != UploadMLJsonToEvossTaskName &&
			workflowData.StepsPassedThrough[i].TaskName != ConvertPropertyModelToEVJsonTaskName {
			lastCompletedTask = workflowData.StepsPassedThrough[i]
			break
		}
	}
	ctxlog.Info(ctx, fmt.Sprintf("Last executed task: %s, status: %s", lastCompletedTask.TaskName, lastCompletedTask.Status))

	ctxlog.Info(ctx, "FLow type: ", workflowData.FlowType)
	if lastCompletedTask.Status == success {
		finalTaskStepID = lastCompletedTask.StepId
		if workflowData.FlowType == "Twister" {
			ctxlog.Info(ctx, "Job being pushed to Twister...")
			legacyStatus = "MACompleted"
		}
	} else {
		if failureOutput, ok := status.FailedTaskStatusMap[lastCompletedTask.TaskName]; !ok {
			ctxlog.Error(ctx, lastCompletedTask.TaskName+" record not found in failureTaskOutputMap map")
			return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), errors.New(lastCompletedTask.TaskName + " record not found in failureTaskOutputMap map")
		} else {
			legacyStatus = failureOutput.StatusKey
			for i := stepscount - 1; i >= 0; i-- {
				if workflowData.StepsPassedThrough[i].TaskName == failureOutput.FallbackTaskName {
					finalTaskStepID = workflowData.StepsPassedThrough[i].StepId
					break
				}
			}
		}
	}

	taskData, err := commonHandler.DBClient.FetchStepExecutionData(ctx, finalTaskStepID)
	if err != nil {
		ctxlog.Error(ctx, "Error in fetching steo data from DocumentDb: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), err
	}
	if taskOutput, ok = taskData.Output["propertyModelLocation"]; !ok {
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), errors.New("propertyModelLocation missing from task output")
	}
	if propertyModelS3Path, ok = taskOutput.(string); !ok {
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), err
	}

	evjsonS3Path, err := CovertPropertyModelToEVJson(ctx, workflowData.OrderId, eventData.WorkflowID, propertyModelS3Path, eventData.ImageMetaDataLocation)
	if err != nil {
		ctxlog.Error(ctx, "Error in calling EVJson convertor service: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), err
	}

	if evJsonLocation, ok = evjsonS3Path["evJsonLocation"]; !ok {
		ctxlog.Error(ctx, "evJsonLocation not returned")
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), errors.New("evJsonLocation not returned")
	}

	ctxlog.Info(ctx, "EVJsonLocation: ", evJsonLocation)

	//get s3path from map
	host, path, err := commonHandler.AwsClient.FetchS3BucketPath(evJsonLocation)
	if err != nil {
		ctxlog.Error(ctx, "Error in fetching AWS path: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), err
	}
	propertyModelByteArray, err := commonHandler.AwsClient.GetDataFromS3(ctx, host, path)
	if err != nil {
		ctxlog.Error(ctx, "Error in getting downloading from s3: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), err
	}

	if _, err = UploadMLJsonToEvoss(ctx, workflowData.OrderId, eventData.WorkflowID, propertyModelByteArray); err != nil {
		ctxlog.Error(ctx, "Error while uploading file to EVOSS: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, "", eventData.WorkflowID, StepExecutionData), err
	}
	ctxlog.Info(ctx, "EVJson successfully uploaded to EVOSS...")
	return updateDocumentDbAndGetResponse(ctx, success, legacyStatus, eventData.WorkflowID, StepExecutionData), nil
}

func CovertPropertyModelToEVJson(ctx context.Context, reportId, workflowId, PropertyModelS3Path, ImageMetaDataS3Path string) (map[string]string, error) {
	calloutLambdaFunction := os.Getenv(envCalloutLambdaFunction)
	evJsonConvertorEndpoint := os.Getenv(envEvJsonConvertorEndpoint)

	payload := map[string]interface{}{
		"requestData": map[string]string{
			"reportId":              reportId,
			"propertyModelLocation": PropertyModelS3Path,
			"imageMetaDataLocation": ImageMetaDataS3Path,
		},
		"url":           evJsonConvertorEndpoint,
		"requestMethod": "POST",
		"IsWaitTask":    false,
		"taskName":      ConvertPropertyModelToEVJsonTaskName,
		"orderId":       reportId,
		"reportId":      reportId,
		"workflowId":    workflowId,
	}
	result, err := commonHandler.AwsClient.InvokeLambda(ctx, calloutLambdaFunction, payload)
	if err != nil {
		return nil, err
	}
	var resp map[string]string
	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		return nil, err
	}
	errorType, ok := resp["errorType"]
	if ok {
		ctxlog.Errorf(ctx, "error occured while executing lambda: %+v", errorType)
		if errorType == RetriableError {
			return resp, &error_handler.RetriableError{Message: fmt.Sprintf("received %s errorType while executing lambda", errorType)}
		}
		return resp, errors.New(fmt.Sprintf("error occured while executing lambda: %+v", errorType))
	}

	return resp, nil
}

func UploadMLJsonToEvoss(ctx context.Context, reportId, workflowId string, mlJson []byte) (map[string]string, error) {
	calloutLambdaFunction := os.Getenv(envCalloutLambdaFunction)
	authsecret := os.Getenv(DBSecretARN)
	endpoint := os.Getenv(envLegacyEndpoint)

	secretMap, err := commonHandler.AwsClient.GetSecret(ctx, authsecret, region)
	if err != nil {
		ctxlog.Error(ctx, "error while fetching auth token from secret manager", err.Error())
		return nil, err
	}

	token, ok := secretMap[legacyAuthKey].(string)
	if !ok {
		ctxlog.Error(ctx, "Issue with parsing Auth Token: ", secretMap[legacyAuthKey])
		return nil, errors.New(fmt.Sprintf("Issue with parsing Auth Token: %+v", secretMap[legacyAuthKey]))
	}

	payload := map[string]interface{}{
		"requestData":   b64.StdEncoding.EncodeToString(mlJson),
		"url":           fmt.Sprintf("%s/UploadMLJson?reportId=%s", endpoint, reportId),
		"requestMethod": "POST",
		"headers": map[string]string{
			"Content-Type":  "application/json",
			"Accept":        "application/json",
			"Authorization": "Basic " + token,
		},
		"IsWaitTask": false,
		"taskName":   UploadMLJsonToEvossTaskName,
		"orderId":    reportId,
		"reportId":   reportId,
		"workflowId": workflowId,
	}

	result, err := commonHandler.AwsClient.InvokeLambda(ctx, calloutLambdaFunction, payload)
	if err != nil {
		return nil, err
	}
	var resp map[string]string
	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		return resp, err
	}

	errorType, ok := resp["errorType"]
	if ok {
		ctxlog.Errorf(ctx, "error occured while executing lambda: %+v", errorType)
		if errorType == RetriableError {
			return resp, &error_handler.RetriableError{Message: fmt.Sprintf("received %s errorType while executing lambda", errorType)}
		}
		return resp, errors.New(fmt.Sprintf("error occured while executing lambda: %+v", errorType))
	}

	return resp, nil
}

func updateDocumentDbAndGetResponse(ctx context.Context, status, legacyStatus, workflowId string, stepExecutionData documentDB_client.StepExecutionDataBody) map[string]interface{} {
	stepExecutionData.EndTime = time.Now().Unix()
	response := map[string]interface{}{
		"status": status,
	}
	if status == failure {
		stepExecutionData.Status = failure
		stepExecutionData.Output = response
	} else {
		stepExecutionData.Status = success
		stepExecutionData.Output = response
		response["legacyStatus"] = legacyStatus
	}

	err := commonHandler.DBClient.InsertStepExecutionData(ctx, stepExecutionData)
	if err != nil {
		ctxlog.Error(ctx, "Unable to insert Step Data in DocumentDB")
	}
	filter := bson.M{"_id": workflowId}
	update := commonHandler.DBClient.BuildQueryForUpdateWorkflowDataCallout(ctx, taskName, stepExecutionData.StepId, status, stepExecutionData.StartTime, false)
	err = commonHandler.DBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.WorkflowDataCollection)
	if err != nil {
		ctxlog.Error(ctx, "Unable to update DocumentDb")
	}
	return response
}

func notificationWrapper(ctx context.Context, req eventData) (map[string]interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		commonHandler.SlackClient.SendErrorMessage(req.ReportID, req.WorkflowID, "evmlconverter", err.Error())
	}
	return resp, err
}

func main() {
	log_config.InitLogging(logLevel)
	commonHandler = common_handler.New(true, true, true, true)
	lambda.Start(notificationWrapper)
}
