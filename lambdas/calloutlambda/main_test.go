package main

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

func TestRequestValidation(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	//Empty Request
	req := MyEvent{}
	_, err := CallService(context.Background(), req, "")
	assert.Equal(t, "reportId is a required field,workflowId is a required field", err.Error())

	//CallType
	//1.Invalid
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "assess"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "unsupported calltype", err.Error())

	//2.Hipster, Status missing
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "hipster"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "status cannot be empty", err.Error())

	//3.Eagleflow
	awsClient.Mock.On("InvokeLambda", context.Background(), "", map[string]interface{}{"reportId": "1241243", "status": "MAStarted", "taskName": "", "workflowId": "some-id"}).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"status\": \"success\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient

	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "Eagleflow", Status: "MAStarted"}
	_, err = CallService(context.Background(), req, "")
	assert.NoError(t, err)

	//RequestMethod
	//1.Invalid
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "Eagleflow", RequestMethod: "PATCH"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "invalid http request method", err.Error())

	//2.Empty URL
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, RequestMethod: "POST"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "invalid callout request", err.Error())

	//3.Invalid URL
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, RequestMethod: "POST", URL: "asdfasd.net"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "url must be a valid URL", err.Error())

	// 3. Valid, need http mocking
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "", RequestMethod: "GET", URL: "http://google.com"}
	httpClient.Mock.On("Getwithbody").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)
	commonHandler.HttpClient = httpClient
	_, err = CallService(context.Background(), req, "")
	assert.NoError(t, err)

	// valid get request with sttoe datata to s3
	awsClient.Mock.On("StoreDataToS3", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(nil)
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "", RequestMethod: "GET", URL: "http://google.com", StoreDataToS3: "s3://app-dev-1x0-s3-symphony-workflow/44823954/imageMetadata.json"}
	commonHandler.AwsClient = awsClient
	_, err = CallService(context.Background(), req, "")
	assert.NoError(t, err)

	// 3. Valid POST Call not  wait taask
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "", RequestMethod: "POST", URL: "http://google.com", Payload: "some payload"}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)
	commonHandler.HttpClient = httpClient
	_, err = CallService(context.Background(), req, "")
	assert.NoError(t, err)

	// headers passed
	req = MyEvent{ReportID: reportID, Headers: map[string]string{"contentType": "application/json"}, WorkflowID: workflowId, CallType: "", RequestMethod: "POST", URL: "http://google.com", Payload: "some payload"}
	_, err = CallService(context.Background(), req, "")
	assert.NoError(t, err)
}

func TestCallServiceValidationHipsterJob(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"

	awsClient.Mock.On("InvokeLambda", context.Background(), "", mock.Anything).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"status\": \"success\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	awsClient.Mock.On("StoreDataToS3", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 3. Valid POST Call with  wait taask with hipster job
	req := MyEvent{ReportID: reportID, Status: "QCCompleted", IsWaitTask: true, CallType: "hipster", TaskToken: "taskToken", WorkflowID: workflowId, RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"status": "success"
		}`))),
	}, nil)
	commonHandler.HttpClient = httpClient
	_, err := CallService(context.Background(), req, "1234")
	assert.NoError(t, err)
}

func TestCompleteCalloutSuccess(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. Valid POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, CallType: "", RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, success, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.NoError(t, err)

	// handle sync task
	req = MyEvent{ReportID: reportID, IsWaitTask: false, TaskToken: "taskToken", WorkflowID: workflowId, CallType: "", RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, success, mock.Anything, req.IsWaitTask).Return("update")
	commonHandler.DBClient = dBClient
	_, err = HandleRequest(context.Background(), req)
	assert.NoError(t, err)

}

func TestCompleteCalloutFailure(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. failed POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, CallType: "", RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusInternalServerError,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, failure, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.Error(t, err)
}
func TestCompleteCalloutFailureInsertinginDB(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. Valid POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, CallType: "", RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(errors.New("some error"))
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.Error(t, err)
}

func TestCompleteCalloutFailureGetRequest(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. failed POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, CallType: "", RequestMethod: "GET", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Getwithbody").Return(&http.Response{
		Status:     "500 Error",
		StatusCode: http.StatusInternalServerError,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, failure, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.Error(t, err)
}

func TestCompleteCalloutFailureStatus(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. failed POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, CallType: "", RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "500 ERROR",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, failure, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.Error(t, err)
}

func TestCompleteCalloutWrongResponse(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. failed POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, CallType: "", RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(`some random response`))),
	}, nil)

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, failure, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.Error(t, err)
}
