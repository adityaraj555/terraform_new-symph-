package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

var (
	testevent = []byte(`{
		"callbackId": "mycallbackid",
		"errorMessage": {
		  "Error": "ServiceError",
		  "Cause": "{\"errorMessage\":\"{\\\"message\\\":\\\"workflowId is a required field\\\",\\\"messageCode\\\":4029}\",\"errorType\":\"ServiceError\"}"
		},
		"callbackUrl": "https://simcallback.free.beeceptor.com/callback"
	  }`)
)

func TestNotifier(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = awsClient
	commonHandler.HttpClient = httpClient
	httpClient.Mock.On("Post").Return(&http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: http.StatusOK,
	}, nil).Once()
	eventDataRequestObj := &eventData{}
	mydata := []byte(testevent)
	json.Unmarshal(mydata, &eventDataRequestObj)

	err := notificationWrapper(context.Background(), *eventDataRequestObj)
	assert.NoError(t, err)

}
func TestTmeoutNotifier(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = awsClient
	commonHandler.HttpClient = httpClient
	httpClient.Mock.On("Post").Return(&http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: http.StatusOK,
	}, nil).Once()
	dBClient.Mock.On("GetTimedoutTask", mock.Anything, mock.Anything).Return("some task")
	eventDataRequestObj := &eventData{}
	mydata := []byte(testevent)
	json.Unmarshal(mydata, &eventDataRequestObj)
	eventDataRequestObj.ErrorMessage.Error = Timeout
	err := notificationWrapper(context.Background(), *eventDataRequestObj)
	assert.NoError(t, err)

}
func TestCallbackError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = awsClient
	commonHandler.HttpClient = httpClient
	httpClient.Mock.On("Post").Return(&http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: http.StatusOK,
	}, errors.New("some error")).Once()
	dBClient.Mock.On("GetTimedoutTask", mock.Anything, mock.Anything).Return("some task")
	eventDataRequestObj := &eventData{}
	mydata := []byte(testevent)
	json.Unmarshal(mydata, &eventDataRequestObj)
	eventDataRequestObj.ErrorMessage.Error = Timeout
	err := handler(context.Background(), *eventDataRequestObj)
	assert.Error(t, err)

}
func TestErrorCodesUnavailability(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = awsClient
	commonHandler.HttpClient = httpClient
	httpClient.Mock.On("Post").Return(&http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: http.StatusOK,
	}, nil).Once()
	dBClient.Mock.On("GetTimedoutTask", mock.Anything, mock.Anything).Return("some task")
	eventDataRequestObj := &eventData{}
	sample := []byte(`{
		"callbackId": "mycallbackid",
		"errorMessage": {
		  "Error": "ServiceError",
		  "Cause": "some cause"
		},
		"callbackUrl": "https://simcallback.free.beeceptor.com/callback"
	  }`)
	mydata := []byte(sample)
	json.Unmarshal(mydata, &eventDataRequestObj)
	err := handler(context.Background(), *eventDataRequestObj)
	assert.NoError(t, err)
}
func TestErrorCodesUnavailability2(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	commonHandler.DBClient = dBClient
	commonHandler.AwsClient = awsClient
	commonHandler.HttpClient = httpClient
	httpClient.Mock.On("Post").Return(&http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: http.StatusOK,
	}, nil).Once()
	dBClient.Mock.On("GetTimedoutTask", mock.Anything, mock.Anything).Return("some task")
	eventDataRequestObj := &eventData{}
	sample := []byte(`{
		"callbackId": "mycallbackid",
		"errorMessage": {
		  "Error": "ServiceError",
		  "Cause": "{\"errorMessage\":\"some error\",\"errorType\":\"ServiceError\"}"
		},
		"callbackUrl": "https://simcallback.free.beeceptor.com/callback"
	  }`)
	mydata := []byte(sample)
	json.Unmarshal(mydata, &eventDataRequestObj)
	err := handler(context.Background(), *eventDataRequestObj)
	assert.NoError(t, err)
}
