package main

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	tasksWithPMFOutputString := os.Getenv("tasksWithPMFOutput")
	tasksWithPMFOutputArray := strings.Split(tasksWithPMFOutputString, ",")
	var (
		err                 error
		ok                  bool
		finalTaskStepID     string
		taskOutput          interface{}
		propertyModelS3Path string
		legacyStatus        string = "QCCompleted"
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
		return updateDocumentDbAndGetResponse(ctx, failure, "", "", eventData.WorkflowID, StepExecutionData), error_handler.NewServiceError(error_codes.ErrorFetchingWorkflowExecutionDataFromDB, err.Error())
	}

	ctxlog.Info(ctx, "Workflow Data Fetched from DocumentDb...")
	stepscount := len(workflowData.StepsPassedThrough)
	var lastCompletedTask documentDB_client.StepsPassedThroughBody
	var isQCTaskCompleted, isHipsterTaskCompleted bool
	ctxlog.Info(ctx, "WorkflowID: %s and Flow type: %s", eventData.WorkflowID, workflowData.FlowType)
	ctxlog.Infof(ctx, "Workflow tasks list: %+v", workflowData.StepsPassedThrough)
	//iterate in reverse over list of tasks
out:
	for i := stepscount - 1; i >= 0; i-- {
		for _, task := range tasksWithPMFOutputArray {
			//check if workflow reached both hipster measure and qc in case of faliure in between.
			switch workflowData.StepsPassedThrough[i].TaskName {
			case "UpdateHipsterJobAndWaitForQC":
				isQCTaskCompleted = true
			case "CreateHipsterJobAndWaitForMeasurement":
				isHipsterTaskCompleted = true
			}
			//finding first task from set of tasks with PMF output
			if workflowData.StepsPassedThrough[i].TaskName == task {
				//if status==success, return PMF from this task
				if workflowData.StepsPassedThrough[i].Status == success {
					lastCompletedTask = workflowData.StepsPassedThrough[i]
					if workflowData.FlowType == "Twister" {
						ctxlog.Info(ctx, "Job being pushed to Twister...")
						legacyStatus = "MACompleted"
					}
					break out
					//if status == failure, get status to update back to legacy
				} else if workflowData.StepsPassedThrough[i].Status == failure {
					legacyStatus = status.FailedTaskStatusMap[task].StatusKey
				}
			}
		}
	}

	//check if Hipster Measure is completed which Hipster QC hasn't been reached
	if !isQCTaskCompleted && isHipsterTaskCompleted {
		legacyStatus = "MeasurementFailed"
	}

	ctxlog.Info(ctx, fmt.Sprintf("Last executed taskwith PMF Output: %s, status: %s", lastCompletedTask.TaskName, lastCompletedTask.Status))
	finalTaskStepID = lastCompletedTask.StepId

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
		response["path"] = "Twister"
		if legacyStatus == "QCCompleted" {
			response["path"] = "Hipster"
		}
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
