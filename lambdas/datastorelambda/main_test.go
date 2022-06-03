package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

var testContext = log_config.SetTraceIdInContext(context.Background(), "44825849", "9cabffdf-e980-0bbf-b481-0048f7a88bef")

var DataStoreRequest string = `{
	"action": "insert",
	"input": {
	  "address": {
		"city": "Gilroy",
		"country": "UnitedStates",
		"latitude": 37.024966,
		"longitude": -121.583003,
		"state": "CA",
		"street": "270 Ronan Ave",
		"zip": "95020"
	  },
	  "reportId": "44825849",
	  "orderId": "44825849",
	  "customerNotes": "",
	  "measurementInstructions": {},
	  "orderType": ""
	},
	"orderId": "44825849",
	"workflowId": "9cabffdf-e980-0bbf-b481-0048f7a88bef"
  }`

func TestDatastoreLambdainsert(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	slackClient := new(mocks.ISlackClient)

	DataStoreRequestObj := RequestBody{}
	mydata := []byte(DataStoreRequest)
	json.Unmarshal(mydata, &DataStoreRequestObj)

	expectedResp := map[string]interface{}{"status": "success"}

	dBClient.Mock.On("InsertWorkflowExecutionData", testContext, mock.Anything).Return(nil)
	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, mock.Anything).Return(documentDB_client.WorkflowExecutionDataBody{}, nil)
	commonHandler.DBClient = dBClient
	commonHandler.SlackClient = slackClient
	resp, err := notificationWrapper(context.Background(), DataStoreRequestObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}
func TestDatastoreLambdainserterror(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	slackClient := new(mocks.ISlackClient)

	DataStoreRequestObj := RequestBody{}
	mydata := []byte(DataStoreRequest)
	json.Unmarshal(mydata, &DataStoreRequestObj)

	expectedResp := map[string]interface{}{"status": "failed"}

	dBClient.Mock.On("InsertWorkflowExecutionData", testContext, mock.Anything).Return(errors.New("some error"))
	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, mock.Anything).Return(documentDB_client.WorkflowExecutionDataBody{}, nil)
	slackClient.On("SendErrorMessage", DataStoreRequestObj.OrderId, DataStoreRequestObj.WorkflowId, "datastore", mock.Anything, mock.Anything).Return(nil)
	commonHandler.DBClient = dBClient
	commonHandler.SlackClient = slackClient
	resp, err := notificationWrapper(context.Background(), DataStoreRequestObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}
func TestDatastoreLambdaupdate(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)

	DataStoreRequestObj := RequestBody{}
	mydata := []byte(DataStoreRequest)
	json.Unmarshal(mydata, &DataStoreRequestObj)
	DataStoreRequestObj.Action = "update"
	expectedResp := map[string]interface{}{"status": "success"}

	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, mock.Anything).Return(documentDB_client.WorkflowExecutionDataBody{}, nil)
	commonHandler.DBClient = dBClient
	resp, err := Handler(context.Background(), DataStoreRequestObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}
func TestDatastoreLambdaupdateerror(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)

	DataStoreRequestObj := RequestBody{}
	mydata := []byte(DataStoreRequest)
	json.Unmarshal(mydata, &DataStoreRequestObj)
	DataStoreRequestObj.Action = "update"
	expectedResp := map[string]interface{}{"status": "failed"}

	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("some error"))
	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, mock.Anything).Return(documentDB_client.WorkflowExecutionDataBody{}, nil)
	commonHandler.DBClient = dBClient
	resp, err := Handler(context.Background(), DataStoreRequestObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestDatastoreLambdaupdateStepTimeOut(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)

	DataStoreRequestObj := RequestBody{}
	mydata := []byte(DataStoreRequest)
	json.Unmarshal(mydata, &DataStoreRequestObj)
	DataStoreRequestObj.Action = "update"
	expectedResp := map[string]interface{}{"status": "success"}
	stepData := documentDB_client.WorkflowExecutionDataBody{
		StepsPassedThrough: []documentDB_client.StepsPassedThroughBody{
			{
				Status: "running",
				StepId: "1234",
			},
		},
	}

	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(3)
	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, mock.Anything).Return(stepData, nil)
	dBClient.Mock.On("BuildQueryForCallBack", testContext, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("filter", "query")
	commonHandler.DBClient = dBClient
	resp, err := Handler(context.Background(), DataStoreRequestObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}
