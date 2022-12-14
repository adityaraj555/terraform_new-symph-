package main

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/lambdas/legacyupdate/legacy_client"
	"github.eagleview.com/engineering/symphony-service/lambdas/legacyupdate/status"
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
	OrderID      string `json:"orderId"`
	ReportID     string `json:"reportId"`
	WorkflowID   string `json:"workflowId"`
	TaskName     string `json:"taskName"`
	Status       string `json:"status"`
	HipsterJobID string `json:"hipsterJobId"`
	Notes        string `json:"notes"`
}

type LambdaOutput struct {
	Status      string `json:"status"`
	MessageCode int    `json:"messageCode"`
	Message     string `json:"message"`
}

/*
Input:
{
	"status": "wf-status",
	"hipsterJobId": "613498-kjhvcdlo87234",
	"taskName": "facet-key-point",
	"orderId": "",
	"reportId": "",
	"workflowId": "",
}

Output:
{
	"status": "success/failure",
	"messageCode": 200/500,
	"message": ""
}

*/
func handler(ctx context.Context, eventData *eventData) (*LambdaOutput, error) {

	if eventData.ReportID == "" {
		log.Errorf(ctx, "reportId cannot be empty, body: %+v", eventData)
		return nil, errors.New("reportId cannot be empty")
	}

	ctx = log_config.SetTraceIdInContext(ctx, eventData.ReportID, eventData.WorkflowID)
	log.Info(ctx, "LegacyUpdate lambda reached...")

	status, ok := status.StatusMap[eventData.Status]
	if !ok {
		log.Error(ctx, "Invalid status: ", eventData.Status)
		return nil, errors.New("invalid status: " + eventData.Status)
	}

	endpoint := os.Getenv(envLegacyEndpoint)
	authsecret := os.Getenv(envLegacyAuthSecret)

	secretMap, err := awsClient.GetSecret(ctx, authsecret, region)
	if err != nil {
		log.Error(ctx, "error while fetching auth token from secret manager, error: ", err.Error())
		return nil, err
	}

	client := legacy_client.New(endpoint, secretMap[legacyAuthKey].(string), httpClient)
	payload := legacy_client.LegacyUpdateRequest{
		Status:       status.Status,
		SubStatus:    status.SubStatus,
		ReportID:     eventData.ReportID,
		HipsterJobId: eventData.HipsterJobID,
		Notes:        eventData.Notes,
	}

	err = client.UpdateReportStatus(ctx, &payload)
	if err != nil {
		log.Error(ctx, "Error while updating status in Legacy, error: ", err.Error())
		return &LambdaOutput{
			Status: failure,
			//MessageCode: code,
			Message: err.Error(),
		}, err
	}

	log.Info(ctx, "LegacyUpdate lambda successful...")
	return &LambdaOutput{
		Status: success,
		//MessageCode: code,
		Message: "report status updated successfully",
	}, nil
}

func main() {
	log_config.InitLogging(logLevel)
	httpClient = &httpservice.HTTPClientV2{}
	awsClient = &aws_client.AWSClient{}
	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		// APITimeout: 90,
	})
	lambda.Start(handler)
}
