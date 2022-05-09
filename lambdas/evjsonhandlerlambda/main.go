package main

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/lambdas/legacyupdate/status"
)

const (
	success  = "success"
	failure  = "failure"
	logLevel = "info"
)

var (
	legacyStatusMap      = map[string]string{}
	endpoint, authsecret string
	commonHandler        common_handler.CommonHandler
)

type eventData struct {
	WorkflowID            string `json:"workflowId"`
	ImageMetaDataLocation string `json:"imageMetaDataLocation"`
}

func handler(ctx context.Context, eventData eventData) (map[string]interface{}, error) {
	var (
		err                 error
		ok                  bool
		finalTaskStepID     string
		taskOutput          interface{}
		propertyModelS3Path string
	)
	statusObject := *status.New()
	if statusObject, ok = status.StatusMap["QCCompleted"]; !ok {
		return lambdaResponse(failure), errors.New("record not found in map")
	}

	workflowData, err := commonHandler.DBClient.FetchWorkflowExecutionData(eventData.WorkflowID)
	if err != nil {
		return lambdaResponse(failure), err
	}

	lastCompletedTask := workflowData.StepsPassedThrough[len(workflowData.StepsPassedThrough)-2]
	if lastCompletedTask.Status == success {
		finalTaskStepID = lastCompletedTask.StepId
		if workflowData.FlowType == "Twister" {
			if statusObject, ok = status.StatusMap["MACompleted"]; !ok {
				return lambdaResponse(failure), errors.New("record not found in map")
			}
		}

	} else {
		if failureOutput, ok := status.FailedTaskStatusMap[lastCompletedTask.TaskName]; !ok {
			return lambdaResponse(failure), errors.New("record not found in failureTaskOutputMap map")
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
	legacyStatus := statusObject.SubStatus
	taskData, err := commonHandler.DBClient.FetchStepExecutionData(finalTaskStepID)
	if err != nil {
		return lambdaResponse(failure), err
	}
	if taskOutput, ok = taskData.Output["propertyModelLocation"]; !ok {
		return lambdaResponse(failure), errors.New("propertyModelLocation missing from task output")
	}
	if propertyModelS3Path, ok = taskOutput.(string); !ok {
		return lambdaResponse(failure), errors.New("unable to cast propertyModelLocation to string")
	}

	evjsonS3Path, err := CovertPropertyModelToEVJson(ctx, propertyModelS3Path, eventData.ImageMetaDataLocation)
	if err != nil {
		return lambdaResponse(failure), err
	}

	host, path, err := commonHandler.AwsClient.FetchS3BucketPath(evjsonS3Path)
	if err != nil {
		return lambdaResponse(failure), err
	}
	propertyModelByteArray, err := commonHandler.AwsClient.GetDataFromS3(ctx, host, path)
	if err != nil {
		return lambdaResponse(failure), err
	}

	if err = commonHandler.LegacyClient.UploadMLJsonToEvoss(ctx, workflowData.OrderId, propertyModelByteArray); err != nil {
		return lambdaResponse(failure), err
	}

	return map[string]interface{}{
		"status":       success,
		"legacyStatus": legacyStatus,
	}, nil
}

func CovertPropertyModelToEVJson(ctx context.Context, PropertyModelS3Path, ImageMetaDataS3Path string) (string, error) {
	return "", nil
}

func lambdaResponse(status string) map[string]interface{} {
	return map[string]interface{}{
		"status": status,
	}
}

func initLogging(level string) {
	log.SetFormat("json")
	l := log.ParseLevel(level)
	log.SetLevel(l)
}

func main() {
	initLogging(logLevel)
	commonHandler = common_handler.New(true, true, true, true)
	lambda.Start(handler)
}
