package main

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

var testContext = log_config.SetTraceIdInContext(context.Background(), "44825849", "9cabffdf-e980-0bbf-b481-0048f7a88bef")

func TestLegacyStatusUpdate(t *testing.T) {
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	eventDataObj := eventData{
		Status:       "QCCompleted",
		HipsterJobID: "HipsterJobID",
		OrderID:      "44825849",
		ReportID:     "44825849",
		WorkflowID:   "9cabffdf-e980-0bbf-b481-0048f7a88bef",
	}
	expectedResp := &LambdaOutput{
		Status:  success,
		Message: "report status updated successfully",
	}
	aws_Client.Mock.On("GetSecret", testContext, mock.Anything, mock.Anything).Return(map[string]interface{}{"TOKEN": "authToken"}, nil)
	http_Client.Mock.On("Post").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)
	awsClient = aws_Client
	httpClient = http_Client
	resp, err := handler(context.Background(), &eventDataObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
}
func TestLegacyStatusUpdateValidationError(t *testing.T) {

	eventDataObj := eventData{
		Status:       "QCCompleted",
		HipsterJobID: "HipsterJobID",
		OrderID:      "44825849",
		ReportID:     "",
		WorkflowID:   "9cabffdf-e980-0bbf-b481-0048f7a88bef",
	}
	expectedResp := (*LambdaOutput)(nil)

	resp, err := handler(context.Background(), &eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestLegacyStatusUpdateinvalidstatus(t *testing.T) {

	eventDataObj := eventData{
		Status:       "random status",
		HipsterJobID: "HipsterJobID",
		OrderID:      "44825849",
		ReportID:     "44825849",
		WorkflowID:   "9cabffdf-e980-0bbf-b481-0048f7a88bef",
	}
	expectedResp := (*LambdaOutput)(nil)

	resp, err := handler(context.Background(), &eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestLegacyStatusUpdateUnableToFetchToken(t *testing.T) {
	aws_Client := new(mocks.IAWSClient)
	eventDataObj := eventData{
		Status:       "QCCompleted",
		HipsterJobID: "HipsterJobID",
		OrderID:      "44825849",
		ReportID:     "44825849",
		WorkflowID:   "9cabffdf-e980-0bbf-b481-0048f7a88bef",
	}
	expectedResp := (*LambdaOutput)(nil)
	aws_Client.Mock.On("GetSecret", testContext, mock.Anything, mock.Anything).Return(nil, errors.New("unable to fetch secrets"))
	awsClient = aws_Client
	resp, err := handler(context.Background(), &eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)
}
func TestLegacyStatusUpdateErrorMakingApiCall(t *testing.T) {
	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)
	eventDataObj := eventData{
		Status:       "QCCompleted",
		HipsterJobID: "HipsterJobID",
		OrderID:      "44825849",
		ReportID:     "44825849",
		WorkflowID:   "9cabffdf-e980-0bbf-b481-0048f7a88bef",
	}
	expectedResp := &LambdaOutput{
		Status:  failure,
		Message: "response not ok from legacy",
	}
	aws_Client.Mock.On("GetSecret", testContext, mock.Anything, mock.Anything).Return(map[string]interface{}{"TOKEN": "authToken"}, nil)
	http_Client.Mock.On("Post").Return(&http.Response{
		StatusCode: 400,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)
	awsClient = aws_Client
	httpClient = http_Client
	resp, err := handler(context.Background(), &eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)
}
