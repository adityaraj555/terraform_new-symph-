package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/legacy_client"
	"github.eagleview.com/engineering/symphony-service/lambdas/legacyupdate/status"
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
	DBSecretARN                    = "DBSecretARN"
)

var (
	failureTaskOutputMap = map[string]string{
		"CreateHipsterJobAndWaitForMeasurement": "3DModellingService",
		"UpdateHipsterJobAndWaitForQC":          "CreateHipsterJobAndWaitForMeasurement",
	}
	legacyStatusMap      = map[string]string{}
	AwsClient            aws_client.IAWSClient
	HttpClient           httpservice.IHTTPClientV2
	DBClient             documentDB_client.IDocDBClient
	endpoint, authsecret string
	LegacyClient         *legacy_client.LegacyClient
)

type eventData struct {
	WorkflowID          string `json:"workflowId"`
	ImageMetaDataS3Path string `json:"imageMetaDataS3Path"`
}

func Handler(ctx context.Context, eventData eventData) (map[string]interface{}, error) {
	var (
		err                    error
		requiredOutputTaskName string
		ok                     bool
		finalTaskStepID        string
		taskOutput             interface{}
		propertyModelS3Path    string
		legacyStatus           string = "HipsterQCCompleted"
	)
	statusObject := *status.New()
	if statusObject, ok = status.StatusMap["QCCompleted"]; !ok {
		return lambdaResponse(failure), errors.New("record not found in map")
	}

	if endpoint == "" || authsecret == "" {
		return lambdaResponse(failure), errors.New("Unable to read env variables")
	}

	workflowData, err := DBClient.FetchWorkflowExecutionData(eventData.WorkflowID)
	if err != nil {
		return lambdaResponse(failure), err
	}

	finalTask := workflowData.StepsPassedThrough[len(workflowData.StepsPassedThrough)-1]

	if finalTask.Status == success {
		finalTaskStepID = finalTask.StepId
		if workflowData.FlowType == "Twister" {
			if statusObject, ok = status.StatusMap["MACompleted"]; !ok {
				return lambdaResponse(failure), errors.New("record not found in map")
			}
		}

	} else {
		if requiredOutputTaskName, ok = failureTaskOutputMap[finalTask.TaskName]; !ok {
			return lambdaResponse(failure), errors.New("record not found in map")
		}

		if statusObject, ok = status.StatusMap[finalTask.TaskName]; !ok {
			return lambdaResponse(failure), errors.New("record not found in map")
		}
		legacyStatus = statusObject.SubStatus

		for _, val := range workflowData.StepsPassedThrough {
			if val.TaskName == requiredOutputTaskName {
				finalTaskStepID = val.StepId
				break
			}
		}
	}
	taskData, err := DBClient.FetchStepExecutionData(finalTaskStepID)
	if err != nil {
		return lambdaResponse(failure), err
	}
	if taskOutput, ok = taskData.Output["propertyModelLocation"]; !ok {
		return lambdaResponse(failure), err
	}
	if propertyModelS3Path, ok = taskOutput.(string); !ok {
		return lambdaResponse(failure), err
	}

	evjsonS3Path, err := CovertPropertyModelToEVJson(ctx, propertyModelS3Path, eventData.ImageMetaDataS3Path)
	if err != nil {
		return lambdaResponse(failure), err
	}

	host, path, err := AwsClient.FetchS3BucketPath(evjsonS3Path)
	if err != nil {
		return lambdaResponse(failure), err
	}
	propertyModelByteArray, err := AwsClient.GetDataFromS3(ctx, host, path)
	if err != nil {
		return lambdaResponse(failure), err
	}

	if err = LegacyClient.UploadMLJsonToEvoss(ctx, workflowData.OrderId, propertyModelByteArray); err != nil {
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
	HttpClient = &httpservice.HTTPClientV2{}
	AwsClient = &aws_client.AWSClient{}
	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		APITimeout: 90,
	})
	if DBClient == nil {
		SecretARN := os.Getenv(DBSecretARN)
		fmt.Println("fetching db secrets")
		secrets, err := AwsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
		if err != nil {
			fmt.Println("Unable to fetch DocumentDb in secret")
		}
		DBClient = documentDB_client.NewDBClientService(secrets)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = DBClient.CheckConnection(ctx); err != nil {
			panic(err)
		}
	}
	endpoint = os.Getenv(envLegacyUploadToEvossEndpoint)
	authsecret = os.Getenv(envLegacyAuthSecret)
	secretMap, err := AwsClient.GetSecret(context.Background(), authsecret, region)
	if err != nil {
		panic(err)
	}
	LegacyClient = legacy_client.New(endpoint, secretMap[legacyAuthKey].(string), HttpClient)
	lambda.Start(Handler)
}
