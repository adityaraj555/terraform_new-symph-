package main_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
	main "github.eagleview.com/engineering/symphony-service/lambdas/calloutlambda"
)

func TestRequestValidation(t *testing.T) {
	reportID := "1241243"
	workflowId := "some-id"
	//Empty Request
	req := main.MyEvent{}
	_, err := main.CallService(context.Background(), req, "")
	assert.Equal(t, "reportId is a required field,workflowId is a required field", err.Error())

	//CallType
	//1.Invalid
	req = main.MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "assess"}
	_, err = main.CallService(context.Background(), req, "")
	assert.Equal(t, "unsupported calltype", err.Error())

	//2.Hipster, Status missing
	req = main.MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "hipster"}
	_, err = main.CallService(context.Background(), req, "")
	assert.Equal(t, "status cannot be empty", err.Error())

	//3.Eagleflow
	mockAWS := &mocks.IAWSClient{}
	mockAWS.
		On("InvokeLambda", context.Background(), "", map[string]interface{}{"reportId": "1241243", "status": "MAStarted", "taskName": "", "workflowId": "some-id"}).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"status\": \"success\"\n}")}, nil)
	main.AwsClient = mockAWS
	req = main.MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "Eagleflow", Status: "MAStarted"}
	_, err = main.CallService(context.Background(), req, "")
	assert.NoError(t, err)

	//RequestMethod
	//1.Invalid
	req = main.MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "Eagleflow", RequestMethod: "PATCH"}
	_, err = main.CallService(context.Background(), req, "")
	assert.Equal(t, "invalid http request method", err.Error())

	//2.Empty URL
	req = main.MyEvent{ReportID: reportID, WorkflowID: workflowId, RequestMethod: "POST"}
	_, err = main.CallService(context.Background(), req, "")
	assert.Equal(t, "invalid callout request", err.Error())

	//3.Invalid URL
	req = main.MyEvent{ReportID: reportID, WorkflowID: workflowId, RequestMethod: "POST", URL: "asdfasd.net"}
	_, err = main.CallService(context.Background(), req, "")
	assert.Equal(t, "url must be a valid URL", err.Error())

	//3. Valid, need http mocking
	// req = main.MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "", RequestMethod: "GET", URL: "http://google.com"}
	// _, err = main.CallService(context.Background(), req, "")
	// assert.NoError(t, err)

}
