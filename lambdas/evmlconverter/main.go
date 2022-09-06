package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/fatih/structs"
	"github.com/google/uuid"
	ctxlog "github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
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
	EvossLocationUrl                     = "envEvossUrl"
	UpdateHipsterJobAndWaitForQCTaskName = "UpdateHipsterJobAndWaitForQC"
)

var (
	legacyStatusMap     = map[string]string{}
	commonHandler       common_handler.CommonHandler
	lambdaExecutonError = "error occured while executing lambda: %+v"
)

type eventData struct {
	ReportID   string `json:"reportId"`
	WorkflowID string `json:"workflowId"`
}

func handler(ctx context.Context, eventData eventData) (map[string]interface{}, error) {
	ctx = log_config.SetTraceIdInContext(ctx, eventData.ReportID, eventData.WorkflowID)
	ctxlog.Info(ctx, "EVMLConverter Lambda Reached")

	var (
		err                 error
		ok                  bool
		finalTaskStepID     string
		taskOutput          interface{}
		propertyModelS3Path string
		legacyStatus        string = "QCCompleted"
	)
	// evossLocationUrl := os.Getenv(EvossLocationUrl)
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
		return updateDocumentDbAndGetResponse(ctx, failure, "", "", eventData.WorkflowID, StepExecutionData), error_handler.NewServiceError(error_codes.ErrorFetchingWorkflowExecutionDataFromDB, err.Error())
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
	}
	if lastCompletedTask.Status != success || (lastCompletedTask.Status == success && lastCompletedTask.TaskName != UpdateHipsterJobAndWaitForQCTaskName && workflowData.FlowType != "Twister") {
		if failureOutput, ok := status.FailedTaskStatusMap[lastCompletedTask.TaskName]; !ok {
			ctxlog.Error(ctx, lastCompletedTask.TaskName+" record not found in failureTaskOutputMap map")
			return updateDocumentDbAndGetResponse(ctx, failure, "", "", eventData.WorkflowID, StepExecutionData), error_handler.NewServiceError(error_codes.TaskRecordNotFoundInFailureTaskOutputMap, lastCompletedTask.TaskName+" record not found in failureTaskOutputMap map")
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
		return updateDocumentDbAndGetResponse(ctx, failure, "", "", eventData.WorkflowID, StepExecutionData), error_handler.NewServiceError(error_codes.ErrorFetchingStepExecutionDataFromDB, err.Error())
	}
	if taskOutput, ok = taskData.Output["propertyModelLocation"]; !ok {
		return updateDocumentDbAndGetResponse(ctx, failure, "", "", eventData.WorkflowID, StepExecutionData), error_handler.NewServiceError(error_codes.PropertyModelLocationMissingInTaskOutput, "propertyModelLocation missing from task output")
	}
	if propertyModelS3Path, ok = taskOutput.(string); !ok {
		return updateDocumentDbAndGetResponse(ctx, failure, "", "", eventData.WorkflowID, StepExecutionData), error_handler.NewServiceError(error_codes.InvalidTypeForPropertyModelLocation, "propertyModelLocation should be a string")
	}
	lambdaResp := updateDocumentDbAndGetResponse(ctx, success, legacyStatus, propertyModelS3Path, eventData.WorkflowID, StepExecutionData)
	return lambdaResp, nil
}

func updateDocumentDbAndGetResponse(ctx context.Context, status, legacyStatus, propertyModelS3Path, workflowId string, stepExecutionData documentDB_client.StepExecutionDataBody) map[string]interface{} {
	stepExecutionData.EndTime = time.Now().Unix()
	response := map[string]interface{}{
		"status": status,
	}
	if status == failure {
		stepExecutionData.Status = failure
		stepExecutionData.Output = response
	} else {
		response["legacyStatus"] = legacyStatus
		response["propertyModelS3Path"] = propertyModelS3Path
		stepExecutionData.Status = success
		stepExecutionData.Output = response
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
		cerr := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(cerr.GetErrorCode(), req.ReportID, req.WorkflowID, "evmlconverter", "evmlconverter", err.Error(), map[string]string(nil))
	}
	return resp, err
}

func main() {
	log_config.InitLogging(logLevel)
	commonHandler = common_handler.New(true, true, true, true, false)
	lambda.Start(notificationWrapper)
}
