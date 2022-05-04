package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/legacy_client"
)

const (
	envLegacyUploadToEvossEndpoint = "LEGACY_EVOSS_ENDPOINT"
	envLegacyAuthSecret            = "LEGACY_AUTH_SECRET"
	legacyAuthKey                  = "TOKEN"
	region                         = "us-east-2"
	success                        = "success"
	failure                        = "failure"
	logLevel                       = "info"
	legacyLambdaFunction           = "envLegacyUpdatefunction"
)

var (
	failureTaskOutputMap = map[string]string{
		"CreateHipsterJobAndWaitForMeasurement": "3DModellingService",
		"UpdateHipsterJobAndWaitForQC":          "CreateHipsterJobAndWaitForMeasurement",
	}
	legacyStatusMap = map[string]string{}
	awsClient       aws_client.IAWSClient
	httpClient      httpservice.IHTTPClientV2
)

type eventData struct {
	WorkflowID string `json:"workflowId"`
}

func handler(ctx context.Context, eventData eventData) (map[string]interface{}, error) {
	var (
		err                    error
		requiredOutputTaskName string
		ok                     bool
		taskData               documentDB_client.StepExecutionDataBody
		taskOutput             interface{}
		s3Location             string
		legacyStatus           string = "HipsterQCCompleted"
	)

	endpoint := os.Getenv(envLegacyUploadToEvossEndpoint)
	authsecret := os.Getenv(envLegacyAuthSecret)

	//Get data from documentDb using workflowId
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	finalTask := workflowData.StepsPassedThrough[len(workflowData.StepsPassedThrough)-1]

	if finalTask.Status == "success" {
		//get task output
		if workflowData.FlowType == "Twister" {
			legacyStatus = "MLAutomationCompleted"
		}
	} else {
		if requiredOutputTaskName, ok = failureTaskOutputMap[finalTask.TaskName]; !ok {
			return lambdaResponse(failure), errors.New("record not found in map")
		}
		if legacyStatus, ok = legacyStatusMap[finalTask.TaskName]; !ok {
			return lambdaResponse(failure), errors.New("record not found in map")
		}

		for _, val := range workflowData.StepsPassedThrough {
			if val.TaskName == requiredOutputTaskName {
				//get task output
			}
		}
	}

	if taskOutput, ok = taskData.Output["propertyModelLocation"]; !ok {
		return lambdaResponse(failure), err
	}
	if s3Location, ok = taskOutput.(string); !ok {
		return lambdaResponse(failure), err
	}
	host, path, err := awsClient.FetchS3BucketPath(s3Location)
	if err != nil {
		return lambdaResponse(failure), err
	}
	propertyModelByteArray, err := awsClient.GetDataFromS3(ctx, host, path)
	if err != nil {
		return lambdaResponse(failure), err
	}

	err = callLegacyStatusUpdate(ctx, workflowData.OrderId, legacyStatus)

	secretMap, err := awsClient.GetSecret(ctx, authsecret, region)
	if err != nil {
		log.Error(ctx, "error while fetching auth token from secret manager", err.Error())
		return nil, err
	}

	client := legacy_client.New(endpoint, secretMap[legacyAuthKey].(string), httpClient)
	err = client.UploadMLJsonToEvoss(ctx, workflowData.OrderId, propertyModelByteArray)
	if err != nil {
		return nil, err
	}
	return lambdaResponse(success), nil
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

func callLegacyStatusUpdate(ctx context.Context, reportId, subStatus string) error {

	legacyRequestPayload := map[string]interface{}{
		"ReportId":  reportId,
		"Status":    "InProcess",
		"SubStatus": subStatus,
	}

	result, err := awsClient.InvokeLambda(ctx, legacyLambdaFunction, legacyRequestPayload)
	if err != nil {
		return err
	}

	var resp map[string]interface{}
	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		return err
	}

	// Do not know how to handle error result.FunctionError

	errorType, ok := resp["errorType"]
	if ok {
		fmt.Println(errorType)
		return errors.New("error occured while executing lambda ")
	}

	legacyStatus, ok := resp["Status"]
	if !ok {
		return errors.New("legacy Response should have status")
	}
	legacyStatusString := strings.ToLower(fmt.Sprintf("%v", legacyStatus))

	if legacyStatusString == failure {
		return errors.New("legacy returned with status as failure")
	}

	return nil
}

func main() {
	initLogging(logLevel)
	httpClient = &httpservice.HTTPClientV2{}
	awsClient = &aws_client.AWSClient{}
	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		APITimeout: 90,
	})
	lambda.Start(handler)
}
