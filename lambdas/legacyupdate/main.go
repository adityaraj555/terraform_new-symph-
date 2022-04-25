package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/lambdas/legacyupdate/legacy_client"
)

const (
	envLegacyEndpoint   = "LEGACY_ENDPOINT"
	envLegacyAuthSecret = "LEGACY_AUTH_SECRET"
	legacyAuthKey       = "TOKEN"
	region              = "us-east-2"
	success             = "success"
	failure             = "failure"
	logLevel            = "info"
)

var awsClient aws_client.IAWSClient
var httpClient httpservice.IHTTPClientV2

type eventData struct {
	OrderID    string                            `json:"orderId"`
	ReportID   string                            `json:"reportId"`
	WorkflowID string                            `json:"workflowId"`
	TaskName   string                            `json:"taskName"`
	Payload    legacy_client.LegacyUpdateRequest `json:"payload"`
}

type LambdaOutput struct {
	Status      string `json:"status"`
	MessageCode int    `json:"messageCode"`
	Message     string `json:"message"`
}

/*
Input:
{
	"payload": {
		"ReportId": 12345,
  		"Status": "InProcess",
  		"SubStatus": "MLAutomationCompleted|MeasurementPending|MeasurementCompleted|QCPending|QCCompleted",
  		"Notes": "some notes",
  		"IsRecapture": false,
  		"HipsterJobId": "afaa627a-727a-4e1d-a5d2-9ef16471759b"
	},
	"task_name": "facet-key-point",
	"order_id": "",
	"report_id": "",
	"workflow_id": "",
}

Output:
{
	"status": "success/failure",
	"messageCode": 200/500,
	"message": ""
}

*/
func handler(ctx context.Context, eventData *eventData) (*LambdaOutput, error) {
	endpoint := os.Getenv(envLegacyEndpoint)
	authsecret := os.Getenv(envLegacyAuthSecret)

	secretMap, err := awsClient.GetSecret(ctx, authsecret, region)
	if err != nil {
		log.Error(ctx, "error while fetching auth token from secret manager", err.Error())
		return nil, err
	}

	client := legacy_client.New(endpoint, secretMap[legacyAuthKey].(string), httpClient)
	err = client.UpdateReportStatus(ctx, &eventData.Payload)
	if err != nil {
		return &LambdaOutput{
			Status: failure,
			//MessageCode: code,
			Message: err.Error(),
		}, err
	}
	return &LambdaOutput{
		Status: success,
		//MessageCode: code,
		Message: "report status updated successfully",
	}, nil
}

func initLogging(level string) {
	log.SetFormat("json")
	l := log.ParseLevel(level)
	log.SetLevel(l)
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
