package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

//var testContext = log_config.SetTraceIdInContext(context.Background(), "44825849", "9cabffdf-e980-0bbf-b481-0048f7a88bef")
var (
	eventData1 = []byte(`{
  "ImageMetadata": "S3 Link",
  "meta": {
    "callbackId": "a2192b7d-a78f-4fa3-90fd-5da69860d464",
    "callbackUrl": "arn:aws:lambda:us-east-2:356071200662:function:app-dev-1x0-lambda-callbacklambda"
  },
  "orderId": "44828269",
  "selectedImages": [
    {
      "S3Path": "S3 Link",
      "View": "O"
    },
    {
      "S3Path": "S3 Link",
      "View": "N"
    },
    {
      "S3Path": "S3 Link",
      "View": "S"
    },
    {
      "S3Path": "S3 Link",
      "View": "E"
    },
    {
      "S3Path": "S3 Link",
      "View": "W"
    }
  ],
  "workflowId": "45de094f-816a-f0b7-3e1f-b74402dfd379"
}`)
)

func TestHandler(t *testing.T) {

	var eventDataReq *eventData
	scannerErr := json.Unmarshal(eventData1, &eventDataReq)
	assert.NoError(t, scannerErr)

	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)

	convertorOutput := lambda.InvokeOutput{
		Payload: []byte(`{"status": "success"}`),
	}
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	aws_Client.Mock.On("FetchS3BucketPath", mock.Anything).Return("", "", nil)
	aws_Client.Mock.On("GetDataFromS3", mock.Anything, "", "").Return([]byte("dummy response"), nil)
	aws_Client.Mock.On("InvokeLambda", mock.Anything, mock.Anything, mock.Anything, false).Return(&convertorOutput, nil)
	aws_Client.Mock.On("GetSecret", mock.Anything, mock.Anything, region).Return(map[string]interface{}{legacyAuthKey: "token"}, nil)
	http_Client.Mock.On("Post").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)

	expectedResp := &LambdaOutput{
		Status:      "success",
		MessageCode: 200,
		Message:     "upload image to evoss and upload imagedatametadata successfully",
	}
	resp, err := handler(context.Background(), eventDataReq)
	fmt.Println(resp, err, expectedResp)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)

}

func TestErrorFetchS3BucketPath(t *testing.T) {

	var eventDataReq *eventData
	scannerErr := json.Unmarshal(eventData1, &eventDataReq)
	assert.NoError(t, scannerErr)

	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)

	convertorOutput := lambda.InvokeOutput{
		Payload: []byte(`{"status": "success"}`),
	}
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	aws_Client.Mock.On("FetchS3BucketPath", mock.Anything).Return("", "", errors.New("some error"))
	aws_Client.Mock.On("InvokeLambda", mock.Anything, mock.Anything, mock.Anything, false).Return(&convertorOutput, nil)

	resp, err := handler(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, (*LambdaOutput)(nil), resp)

}

func TestErrorS3GetData(t *testing.T) {

	var eventDataReq *eventData
	scannerErr := json.Unmarshal(eventData1, &eventDataReq)
	assert.NoError(t, scannerErr)

	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)

	convertorOutput := lambda.InvokeOutput{
		Payload: []byte(`{"status": "success"}`),
	}
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	aws_Client.Mock.On("FetchS3BucketPath", mock.Anything).Return("", "", nil)
	aws_Client.Mock.On("GetDataFromS3", mock.Anything, "", "").Return([]byte(""), errors.New("some error"))
	aws_Client.Mock.On("InvokeLambda", mock.Anything, mock.Anything, mock.Anything, false).Return(&convertorOutput, nil)

	resp, err := handler(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, (*LambdaOutput)(nil), resp)

}

func TestErrorLambdaInvoke(t *testing.T) {

	var eventDataReq *eventData
	scannerErr := json.Unmarshal(eventData1, &eventDataReq)
	assert.NoError(t, scannerErr)

	aws_Client := new(mocks.IAWSClient)
	http_Client := new(mocks.MockHTTPClient)

	convertorOutput := lambda.InvokeOutput{
		Payload: []byte(`{"errorType": "some lambda error"}`),
	}
	commonHandler.AwsClient = aws_Client
	commonHandler.HttpClient = http_Client
	aws_Client.Mock.On("FetchS3BucketPath", mock.Anything).Return("", "", nil)
	aws_Client.Mock.On("GetDataFromS3", mock.Anything, "", "").Return([]byte("dummy response"), nil)
	aws_Client.Mock.On("GetSecret", mock.Anything, mock.Anything, region).Return(map[string]interface{}{legacyAuthKey: "token"}, nil)
	aws_Client.Mock.On("InvokeLambda", mock.Anything, mock.Anything, mock.Anything, false).Return(&convertorOutput, nil)
	http_Client.Mock.On("Post").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)

	resp, err := handler(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, (*LambdaOutput)(nil), resp)

}

func TestErrorValidationdata(t *testing.T) {

	var eventDataReq *eventData
	scannerErr := json.Unmarshal(eventData1, &eventDataReq)
	assert.NoError(t, scannerErr)

	eventDataReq.ReportID = ""
	expectedResp := &LambdaOutput{
		Status:      failure,
		MessageCode: 4047,
		Message:     "error validating input missing fields",
	}
	resp, err := handler(context.Background(), eventDataReq)
	assert.Error(t, err)
	assert.Equal(t, expectedResp, resp)

}
