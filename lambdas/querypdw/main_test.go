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
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
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

	testevent2 = []byte(`{
		"vintage": "2020-12-13",
		"action": "querydata",
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
				  "_detectedBuildingCount": {
                    "marker": "2019-08-29",
                    "value": 1
                   },
				  "structures": [
					{
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "geocoder": {
					  "lat": 43.172988,
					  "lon": -77.501957
				  },
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	http_Client.On("Get").Return(nil, nil)
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	expectedResp := eventResponse{
		Address:    "23 HAVENSHIRE RD, ROCHESTER, NY, 14625",
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

func TestQueryFailed2(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent2, &eventDataReq)
	assert.NoError(t, scannerErr)
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	slackClient := new(mocks.ISlackClient)
	slackClient.On("SendErrorMessage", 4056, "", "", "querypdw", "querypdw", "{\"message\":\"unable to query data after ingestion\",\"messageCode\":4056}", map[string]string(nil)).Return(nil)
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
				  "_detectedBuildingCount": {
                    "marker": "2019-08-29",
                    "value": 1
                   },
				  "structures": [
					{
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "geocoder": {
					  "lat": 43.172988,
					  "lon": -77.501957
				  },
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	http_Client.On("Get").Return(nil, nil)
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.SlackClient = slackClient
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
	_, err := notificationWrapper(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, err.(error_handler.ICodedError).GetErrorCode(), error_codes.ErrorQueryingPDWAfterIngestion)

}

func TestHandlerTriggerSIM2(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	eventDataReq.Address.ParcelAddress = ""
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
				  "_detectedBuildingCount": {
                    "marker": "2019-08-29",
                    "value": 1
                   },
				  "structures": [
					{
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "geocoder": {
					  "lat": 43.172988,
					  "lon": -77.501957
				  },
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	http_Client.On("Get").Return(&http.Response{

		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"address": "31 Havenshire Rd, Rochester, NY 14625, United States",
			"geocoder": "bing",
			"input": {
				"lat": 43.172733,
				"lon": -77.501619,
				"query": "lat=43.172733&lon=-77.501619&parcelID=true"
			},
			"lat": 43.172704,
			"lon": -77.501657,
			"parcelID": "c7a80489-1693-427e-b1df-869cf985e063",
			"status": {
				"code": 1104,
				"message": "reverse geocoding successful"
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
		Address:    "31 Havenshire Rd, Rochester, NY 14625, United States",
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

func TestHandlerGeocodingError503(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	eventDataReq.Address.ParcelAddress = ""
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	slackClient := new(mocks.ISlackClient)
	slackClient.On("SendErrorMessage", 4051, "", "", "querypdw", "querypdw", "{\"message\":\"503 status code received\",\"messageCode\":4051}", map[string]string(nil)).Return(nil)
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
				  "_detectedBuildingCount": {
                    "marker": "2019-08-29",
                    "value": 1
                   },
				  "structures": [
					{
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "geocoder": {
					  "lat": 43.172988,
					  "lon": -77.501957
				  },
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	http_Client.On("Get").Return(&http.Response{
		StatusCode: 503,
	}, nil)
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.SlackClient = slackClient
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}

	_, err := notificationWrapper(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, error_handler.NewRetriableError(4051, "{\"message\":\"503 status code received\",\"messageCode\":4051}"), err)

}

func TestHandlerGeocodingErrorGetError(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	eventDataReq.Address.ParcelAddress = ""
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	slackClient := new(mocks.ISlackClient)
	slackClient.On("SendErrorMessage", 4042, "", "", "querypdw", "querypdw", "{\"message\":\"error calling EGS : some error\",\"messageCode\":4042}", map[string]string(nil)).Return(nil)
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
				  "_detectedBuildingCount": {
                    "marker": "2019-08-29",
                    "value": 1
                   },
				  "structures": [
					{
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "geocoder": {
					  "lat": 43.172988,
					  "lon": -77.501957
				  },
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	http_Client.On("Get").Return(&http.Response{
		StatusCode: 503,
	}, errors.New("some error"))
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.SlackClient = slackClient
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}

	_, err := notificationWrapper(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, error_handler.NewServiceError(4042, "{\"message\":\"error calling EGS : some error\",\"messageCode\":4042}"), err)

}

func TestHandlerGeocodingError400(t *testing.T) {

	var eventDataReq eventData
	scannerErr := json.Unmarshal(testevent, &eventDataReq)
	assert.NoError(t, scannerErr)
	eventDataReq.Address.ParcelAddress = ""
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	mock_auth_client := new(mocks.AuthTokenInterface)
	slackClient := new(mocks.ISlackClient)
	slackClient.On("SendErrorMessage", 4052, "", "", "querypdw", "querypdw", "{\"message\":\"received invalid http status code: 400\",\"messageCode\":4052}", map[string]string(nil)).Return(nil)
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
				  "_detectedBuildingCount": {
                    "marker": "2019-08-29",
                    "value": 1
                   },
				  "structures": [
					{
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "geocoder": {
					  "lat": 43.172988,
					  "lon": -77.501957
				  },
				  "address": "23 HAVENSHIRE RD",
				  "city": "ROCHESTER"
				}
			  ]
			}
		  }`))),
		StatusCode: http.StatusOK,
	}, nil)
	http_Client.On("Get").Return(&http.Response{
		StatusCode: 400,
	}, nil)
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	commonHandler.SlackClient = slackClient
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}

	_, err := notificationWrapper(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, error_handler.NewServiceError(4052, "{\"message\":\"received invalid http status code: 400\",\"messageCode\":4052}"), err)

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
						"roof": {
							"_countRoofFacets": {
								"marker": null,
								"value": null
							}
						},
					  "_outline": {
						"marker": "2021-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "_detectedBuildingCount": {
                    "marker": "2021-08-29",
                    "value": 1
                   },
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
	expectedResp := eventResponse{Address: "23 HAVENSHIRE RD, ROCHESTER, NY, 14625", Latitude: 43.172988, Longitude: -77.501957, ParcelID: "9a3a3f3b-8ba1-468b-8102-3b3e6ee5d8c1", TriggerSIM: true, Message: "Structures does not exist in the graph response"}
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
						"roof": {
							"_countRoofFacets": {
								"marker": null,
								"value": null
							}
						},
					  "_outline": {
						"marker": "2019-08-29"
					  },
					  "id": "5085a802-89fa-48a8-8c3c-bd8480f0378a"
					}
				  ],
				  "_detectedBuildingCount": {
                    "marker": "2019-08-29",
                    "value": 1
                   },
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

func TestCallbackError(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)
	http_Client.Mock.On("Post").Return(&http.Response{

		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
		StatusCode: 500,
	}, nil)
	commonHandler.Secrets = map[string]interface{}{
		"ClientID":     "id",
		"ClientSecret": "secret"}
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
