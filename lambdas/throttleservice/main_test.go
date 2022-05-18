package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

var eventTestData string = `{
	"reportId": "44825849",
	"orderId": "44825849",
	"workflowId": "9cabffdf-e980-0bbf-b481-0048f7a88bef"
  }`

var testContext = log_config.SetTraceIdInContext(context.Background(), "44825849", "9cabffdf-e980-0bbf-b481-0048f7a88bef")

func TestThrottleLambdaTwisterFlow(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)

	eventDataRequestObj := &eventData{}
	mydata := []byte(eventTestData)
	json.Unmarshal(mydata, &eventDataRequestObj)

	expectedResp := map[string]interface{}{"Path": "Twister", "status": Success}
	commonHandler.DBClient = dBClient
	var count int64 = 52
	t.Setenv("AllowedHipsterCount", "50")
	dBClient.Mock.On("GetHipsterCountPerDay", testContext).Return(count, nil)
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	resp, err := notifcationWrapper(context.Background(), eventDataRequestObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestThrottleLambdaGetHipsterCountPerDayError(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	slackclient := new(mocks.ISlackClient)
	eventDataRequestObj := &eventData{}
	mydata := []byte(eventTestData)
	json.Unmarshal(mydata, &eventDataRequestObj)

	expectedResp := map[string]interface{}{"status": failed}
	commonHandler.DBClient = dBClient
	commonHandler.SlackClient = slackclient
	var count int64 = 52
	t.Setenv("AllowedHipsterCount", "50")
	dBClient.Mock.On("GetHipsterCountPerDay", testContext).Return(count, errors.New("some error"))
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	slackclient.Mock.On("SendErrorMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	resp, err := notifcationWrapper(context.Background(), eventDataRequestObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}
func TestThrottleLambdaErrorStringParse(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	slackclient := new(mocks.ISlackClient)
	eventDataRequestObj := &eventData{}
	mydata := []byte(eventTestData)
	json.Unmarshal(mydata, &eventDataRequestObj)

	expectedResp := map[string]interface{}{"status": failed}
	commonHandler.DBClient = dBClient
	commonHandler.SlackClient = slackclient
	var count int64 = 52
	t.Setenv("AllowedHipsterCount", "abc")
	dBClient.Mock.On("GetHipsterCountPerDay", testContext).Return(count, nil)
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("some error"))
	slackclient.Mock.On("SendErrorMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	resp, err := notifcationWrapper(context.Background(), eventDataRequestObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestThrottleLambdaUpdateDocumentDBError(t *testing.T) {
	dBClient := new(mocks.IDocDBClient)
	slackclient := new(mocks.ISlackClient)
	eventDataRequestObj := &eventData{}
	mydata := []byte(eventTestData)
	json.Unmarshal(mydata, &eventDataRequestObj)

	expectedResp := map[string]interface{}{"status": failed}
	commonHandler.DBClient = dBClient
	commonHandler.SlackClient = slackclient
	var count int64 = 52
	t.Setenv("AllowedHipsterCount", "50")
	dBClient.Mock.On("GetHipsterCountPerDay", testContext).Return(count, nil)
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("some error"))
	slackclient.Mock.On("SendErrorMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	resp, err := notifcationWrapper(context.Background(), eventDataRequestObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}
