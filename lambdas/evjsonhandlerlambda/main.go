package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/fatih/structs"
	"github.com/google/uuid"
	logconst "github.eagleview.com/engineering/assess-platform-library/constants"
	ctxlog "github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/lambdas/legacyupdate/status"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	success                    = "success"
	failure                    = "failure"
	logLevel                   = "info"
	taskName                   = "EVMLJsonConverter_UploadToEvoss"
	envCalloutLambdaFunction   = "CALLOUT_LAMBDA_FUNCTION"
	envEvJsonConvertorEndpoint = "EVJSON_CONVERTOR_ENDPOINT"
)

var (
	legacyStatusMap                       = map[string]string{}
	commonHandler                         common_handler.CommonHandler
	logKeyForOrderID, logKeyForWorkflowID ctxlog.TrackID
)

type eventData struct {
	WorkflowID            string `json:"workflowId"`
	ImageMetaDataLocation string `json:"imageMetaDataLocation"`
}

func handler(ctx context.Context, eventData eventData) (map[string]interface{}, error) {
	ctx = context.WithValue(ctx, logKeyForWorkflowID, eventData.WorkflowID)
	ctxlog.Info(ctx, "EVMLConverter Lambda Reached")

	var (
		err                                               error
		ok                                                bool
		finalTaskStepID                                   string
		taskOutput                                        interface{}
		propertyModelS3Path, legacyStatus, evJsonLocation string
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
	statusObject := *status.New()
	if statusObject, ok = status.StatusMap["QCCompleted"]; !ok {
		ctxlog.Error(ctx, "QCCompleted record not found in StatusMap map")
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), errors.New("QCCompleted record not found in StatusMap map")
	}

	workflowData, err := commonHandler.DBClient.FetchWorkflowExecutionData(eventData.WorkflowID)
	if err != nil {
		ctxlog.Error(ctx, "Error in fetching workflow data from DocumentDb: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), err
	}

	ctx = context.WithValue(ctx, logKeyForOrderID, workflowData.OrderId)
	ctxlog.Info(ctx, "Workflow Data Fetched from DocumentDb...")

	lastCompletedTask := workflowData.StepsPassedThrough[len(workflowData.StepsPassedThrough)-1]
	ctxlog.Info(ctx, fmt.Sprintf("Last executed task: %s, status: %s", lastCompletedTask.TaskName, lastCompletedTask.Status))

	if lastCompletedTask.Status == success {
		finalTaskStepID = lastCompletedTask.StepId
		if workflowData.FlowType == "Twister" {
			ctxlog.Info(ctx, "Job being pushed to Twister...")
			if statusObject, ok = status.StatusMap["MACompleted"]; !ok {
				ctxlog.Error(ctx, "MACompleted record not found in StatusMap map")
				return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), errors.New("MACompleted record not found in StatusMap map")
			}
		}
	} else {
		if failureOutput, ok := status.FailedTaskStatusMap[lastCompletedTask.TaskName]; !ok {
			ctxlog.Error(ctx, lastCompletedTask.TaskName+" record not found in failureTaskOutputMap map")
			return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), errors.New(lastCompletedTask.TaskName + " record not found in failureTaskOutputMap map")
		} else {
			statusObject = failureOutput.Status
			for _, val := range workflowData.StepsPassedThrough {
				if val.TaskName == failureOutput.FallbackTaskName {
					finalTaskStepID = val.StepId
					break
				}
			}
		}
	}

	legacyStatus = statusObject.SubStatus
	taskData, err := commonHandler.DBClient.FetchStepExecutionData(finalTaskStepID)
	if err != nil {
		ctxlog.Error(ctx, "Error in fetching steo data from DocumentDb: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), err
	}
	if taskOutput, ok = taskData.Output["propertyModelLocation"]; !ok {
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), errors.New("propertyModelLocation missing from task output")
	}
	if propertyModelS3Path, ok = taskOutput.(string); !ok {
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), err
	}

	evjsonS3Path, err := CovertPropertyModelToEVJson(ctx, workflowData.OrderId, eventData.WorkflowID, propertyModelS3Path, eventData.ImageMetaDataLocation)
	if err != nil {
		ctxlog.Error(ctx, "Error in calling EVJson convertor service: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), err
	}

	if evJsonLocation, ok = evjsonS3Path["evJsonLocation"]; !ok {
		ctxlog.Error(ctx, "evJsonLocation not returned")
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), errors.New("evJsonLocation not returned")
	}

	ctxlog.Info(ctx, "EVJsonLocation: ", evJsonLocation)

	//get s3path from map
	host, path, err := commonHandler.AwsClient.FetchS3BucketPath(evJsonLocation)
	if err != nil {
		ctxlog.Error(ctx, "Error in fetching AWS path: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), err
	}
	propertyModelByteArray, err := commonHandler.AwsClient.GetDataFromS3(ctx, host, path)
	if err != nil {
		ctxlog.Error(ctx, "Error in getting downloading from s3: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), err
	}

	if _, err = UploadMLJsonToEvoss(ctx, workflowData.OrderId, eventData.WorkflowID, propertyModelByteArray); err != nil {
		ctxlog.Error(ctx, "Error while uploading file to EVOSS: ", err.Error())
		return updateDocumentDbAndGetResponse(ctx, failure, legacyStatus, eventData.WorkflowID, StepExecutionData), err
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
		"taskName":      "CovertPropertyModelToEVJson",
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
		return resp, errors.New(fmt.Sprintf("error occured while executing lambda: %+v", errorType))
	}

	return resp, nil
}

func UploadMLJsonToEvoss(ctx context.Context, reportId, workflowId string, mlJson []byte) (map[string]string, error) {
	calloutLambdaFunction := os.Getenv(envCalloutLambdaFunction)

	requestBody := make(map[string]interface{})
	json.Unmarshal(mlJson, requestBody)

	endpoint, token := commonHandler.LegacyClient.GetLegacyBaseUrlAndAuthToken(ctx)
	payload := map[string]interface{}{
		"requestData":   requestBody,
		"url":           fmt.Sprintf("%s/UploadMLJson?reportId=%s", endpoint, reportId),
		"requestMethod": "POST",
		"header": map[string]string{
			"Authorization": "Basic " + token,
		},
		"IsWaitTask": false,
		"taskName":   "UploadMLJsonToEvoss",
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

	err := commonHandler.DBClient.InsertStepExecutionData(stepExecutionData)
	if err != nil {
		ctxlog.Error(ctx, "Unable to insert Step Data in DocumentDB")
	}
	filter := bson.M{"_id": workflowId}
	update := commonHandler.DBClient.BuildQueryForUpdateWorkflowDataCallout(taskName, stepExecutionData.StepId, status, stepExecutionData.StartTime, false)
	err = commonHandler.DBClient.UpdateDocumentDB(filter, update, documentDB_client.WorkflowDataCollection)
	if err != nil {
		ctxlog.Error(ctx, "Unable to update DocumentDb")
	}
	return response
}

func initLogging(level string) {
	log.SetFormat("json")
	l := log.ParseLevel(level)
	log.SetLevel(l)
	logKeyForOrderID = ctxlog.TrackID(logconst.TrackIDOrderID)
	logKeyForWorkflowID = ctxlog.TrackID(logconst.TrackIDWfTracerID)
	ctxlog.SetContextKeys(logKeyForOrderID, logKeyForWorkflowID)
}

func main() {
	initLogging(logLevel)
	commonHandler = common_handler.New(true, true, true, true)
	lambda.Start(handler)
}
