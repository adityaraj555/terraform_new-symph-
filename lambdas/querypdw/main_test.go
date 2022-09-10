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
		"vintage": "2020-12-13",
		"action": "validatedata",
		"address": {
		  "parcelAddress": "23 HAVENSHIRE RD, ROCHESTER, NY, 14625",
		  "lat": 43.172988,
		  "long": -77.501957
		},
		"callbackId": "mycallbackid",
		"callbackUrl": "https://simcallback.free.beeceptor.com/callback"
	  }`)
)

func TestHandlerTriggerSIM(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	mock_auth_client.Mock.On("AddAuthorizationTokenHeader", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auth_client = mock_auth_client
	http_Client.Mock.On("Post").Return(&http.Response{

		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"data": {
			  "parcels": [
				{
				  "state": "NY",
				  "zip": "14625",
				  "id": "9a3a3f3b-8ba1-468b-8102-3b3e6ee5d8c1",
				  "structures": [
					{
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "lat": 43.172988,
				  "lon": -77.501957,
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	expectedResp := eventResponse{
		Address:    "23 HAVENSHIRE RD ROCHESTER NY 14625",
		Latitude:   43.172988,
		Longitude:  -77.501957,
		TriggerSIM: true,
		ParcelID:   "9a3a3f3b-8ba1-468b-8102-3b3e6ee5d8c1",
		Message:    NoStructureMessage,
	}
	resp, err := notificationWrapper(context.Background(), eventDataReq)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}
func TestHandlerParcelDoesnotExist(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	mock_auth_client.Mock.On("AddAuthorizationTokenHeader", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auth_client = mock_auth_client
	http_Client.Mock.On("Post").Return(&http.Response{

		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"data": {
			  "parcels": [
				
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	expectedResp := eventResponse{Message: NoParcelMessage}
	resp, err := notificationWrapper(context.Background(), eventDataReq)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestHandlerStructureExists(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	mock_auth_client.Mock.On("AddAuthorizationTokenHeader", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auth_client = mock_auth_client

	http_Client.Mock.On("Post").Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"data": {
			  "parcels": [
				{
				  "state": "NY",
				  "zip": "14625",
				  "id": "9a3a3f3b-8ba1-468b-8102-3b3e6ee5d8c1",
				  "structures": [
					{
					  "_outline": {
						"marker": "2021-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "lat": 43.172988,
				  "lon": -77.501957,
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil).Once()
	http_Client.Mock.On("Post").Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"data":{
				"parcels":[]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil).Once()
	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: http.StatusOK,
	}, nil).Once()
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	expectedResp := eventResponse{Message: StructurePresentMessage}
	resp, err := notificationWrapper(context.Background(), eventDataReq)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestHandlerNoStructuresAfterIngestion(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	mock_auth_client.Mock.On("AddAuthorizationTokenHeader", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auth_client = mock_auth_client

	http_Client.Mock.On("Post").Return(&http.Response{

		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"data": {
			  "parcels": [
				{
				  "state": "NY",
				  "zip": "14625",
				  "id": "9a3a3f3b-8ba1-468b-8102-3b3e6ee5d8c1",
				  "structures": [
					{
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "lat": 43.172988,
				  "lon": -77.501957,
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	expectedResp := eventResponse{}
	eventDataReq.Action = querydata
	resp, err := handler(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestHandlerUnmarshallingGraphResponseError(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	mock_auth_client.Mock.On("AddAuthorizationTokenHeader", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auth_client = mock_auth_client

	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: http.StatusOK,
	}, nil)
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	expectedResp := eventResponse{}
	eventDataReq.Action = querydata
	resp, err := handler(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestMakePostCallError(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)
	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: http.StatusOK,
	}, errors.New("some error"))
	commonHandler.HttpClient = http_Client
	_, err := makePostCall(context.Background(), "", []byte(""), map[string]string{})
	assert.Error(t, err)
}
func TestMakePostCallInternalError(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)
	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: 500,
	}, nil)
	commonHandler.HttpClient = http_Client
	_, err := makePostCall(context.Background(), "", []byte(""), map[string]string{})
	assert.Error(t, err)
}
func TestMakePostCallForbiddenError(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)
	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: 403,
	}, nil)
	commonHandler.HttpClient = http_Client
	_, err := makePostCall(context.Background(), "", []byte(""), map[string]string{})
	assert.Error(t, err)
}
func TestCallbackError(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)
	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: 500,
	}, nil)
	commonHandler.HttpClient = http_Client
	err := makeCallBack(context.Background(), success, "", "", "", 0, map[string]interface{}{})
	assert.Error(t, err)
}

func TestFetchPDWDataError(t *testing.T) {
	aws_Client := new(mocks.IAWSClient)

	http_Client := new(mocks.MockHTTPClient)
	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: 500,
	}, nil)
	commonHandler.HttpClient = http_Client
	commonHandler.AwsClient = aws_Client
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	mock_auth_client := new(mocks.AuthTokenInterface)
	mock_auth_client.Mock.On("AddAuthorizationTokenHeader", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auth_client = mock_auth_client
	_, err := fetchDataFromPDW(context.Background(), "")
	assert.Error(t, err)
}

func TestFetchPDWDataErrorAddingTokenToHeaders(t *testing.T) {
	aws_Client := new(mocks.IAWSClient)

	http_Client := new(mocks.MockHTTPClient)
	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: 200,
	}, nil)
	commonHandler.HttpClient = http_Client
	commonHandler.AwsClient = aws_Client
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	mock_auth_client := new(mocks.AuthTokenInterface)
	mock_auth_client.Mock.On("AddAuthorizationTokenHeader", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("some error"))
	auth_client = mock_auth_client
	_, err := fetchDataFromPDW(context.Background(), "")
	assert.Error(t, err)
}
