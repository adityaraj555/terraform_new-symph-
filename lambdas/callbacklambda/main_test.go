package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

var RequestBodyString string = `{
    "status": "success",
    "message": "",
    "messageCode": "",
    "callbackId": "callbackId",
    "response": {
        "facetKeyPointLocation": "S3 link for facet_key_point_detection "
    }
}`

func TestCallbacksuccess(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	aws_client := new(mocks.IAWSClient)
	RequestBodyObj := RequestBody{}
	mydata := []byte(RequestBodyString)
	json.Unmarshal(mydata, &RequestBodyObj)

	expectedResp := map[string]interface{}{"status": "success"}
	dBClient.Mock.On("FetchStepExecutionData", context.Background(), "callbackId").Return(documentDB_client.StepExecutionDataBody{TaskToken: "TaskToken"}, nil)
	aws_client.Mock.On("CloseWaitTask", context.Background(), "success", "TaskToken", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForCallBack", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("filter", "query")
	dBClient.Mock.On("UpdateDocumentDB", context.Background(), "filter", "query", mock.Anything).Return(nil)
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = aws_client
	resp, err := notificationWrapper(context.Background(), RequestBodyObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestCallbackvalidation(t *testing.T) {

	RequestBodyObj := RequestBody{}
	slackClient := &mocks.ISlackClient{}
	slackClient.On("SendErrorMessage", mock.Anything, "", "", "callback", "invalid status", map[string]string(nil)).Return(nil)
	mydata := []byte(RequestBodyString)
	json.Unmarshal(mydata, &RequestBodyObj)
	RequestBodyObj.Status = "random"
	expectedResp := map[string]interface{}{"status": failure}
	commonHandler.SlackClient = slackClient
	resp, err := notificationWrapper(context.Background(), RequestBodyObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestCallbackErrorFetching(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	aws_client := new(mocks.IAWSClient)
	RequestBodyObj := RequestBody{}
	mydata := []byte(RequestBodyString)
	json.Unmarshal(mydata, &RequestBodyObj)

	expectedResp := map[string]interface{}{"status": failure}
	dBClient.Mock.On("FetchStepExecutionData", context.Background(), "callbackId").Return(documentDB_client.StepExecutionDataBody{TaskToken: "TaskToken"}, errors.New("error while fetching"))
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = aws_client
	resp, _, _, err := Handler(context.Background(), RequestBodyObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestCallbackRework(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	aws_client := new(mocks.IAWSClient)
	RequestBodyObj := RequestBody{}
	mydata := []byte(RequestBodyString)
	json.Unmarshal(mydata, &RequestBodyObj)
	RequestBodyObj.Status = rework
	expectedResp := map[string]interface{}{"status": "success"}
	dBClient.Mock.On("FetchStepExecutionData", context.Background(), "callbackId").Return(documentDB_client.StepExecutionDataBody{TaskToken: "TaskToken"}, nil)
	aws_client.Mock.On("CloseWaitTask", context.Background(), "success", "TaskToken", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForCallBack", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("filter", "query")
	dBClient.Mock.On("UpdateDocumentDB", context.Background(), "filter", "query", mock.Anything).Return(nil)
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = aws_client
	resp, _, _, err := Handler(context.Background(), RequestBodyObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestCallbackFailure(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	aws_client := new(mocks.IAWSClient)
	RequestBodyObj := RequestBody{}
	mydata := []byte(RequestBodyString)
	json.Unmarshal(mydata, &RequestBodyObj)
	RequestBodyObj.Status = failure
	expectedResp := map[string]interface{}{"status": success}
	dBClient.Mock.On("FetchStepExecutionData", context.Background(), "callbackId").Return(documentDB_client.StepExecutionDataBody{TaskToken: "TaskToken"}, nil)
	aws_client.Mock.On("CloseWaitTask", context.Background(), "failure", "TaskToken", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForCallBack", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("filter", "query")
	dBClient.Mock.On("UpdateDocumentDB", context.Background(), "filter", "query", mock.Anything).Return(nil)
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = aws_client
	resp, _, _, err := Handler(context.Background(), RequestBodyObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestCallbackFailureClosingWaitTaks(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	aws_client := new(mocks.IAWSClient)
	RequestBodyObj := RequestBody{}
	mydata := []byte(RequestBodyString)
	json.Unmarshal(mydata, &RequestBodyObj)
	RequestBodyObj.Status = success
	expectedResp := map[string]interface{}{"status": failure}
	dBClient.Mock.On("FetchStepExecutionData", context.Background(), "callbackId").Return(documentDB_client.StepExecutionDataBody{TaskToken: "TaskToken"}, nil)
	aws_client.Mock.On("CloseWaitTask", context.Background(), success, "TaskToken", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("some error"))
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = aws_client
	resp, _, _, err := Handler(context.Background(), RequestBodyObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestCallbackFailedUpdatingDB(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	aws_client := new(mocks.IAWSClient)
	RequestBodyObj := RequestBody{}
	mydata := []byte(RequestBodyString)
	json.Unmarshal(mydata, &RequestBodyObj)

	expectedResp := map[string]interface{}{"status": failure}
	dBClient.Mock.On("FetchStepExecutionData", context.Background(), "callbackId").Return(documentDB_client.StepExecutionDataBody{TaskToken: "TaskToken"}, nil)
	aws_client.Mock.On("CloseWaitTask", context.Background(), "success", "TaskToken", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForCallBack", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("filter", "query")
	dBClient.Mock.On("UpdateDocumentDB", context.Background(), "filter", "query", mock.Anything).Return(errors.New("error updating DocDB"))
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = aws_client
	resp, _, _, err := Handler(context.Background(), RequestBodyObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}
