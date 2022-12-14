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
	"github.eagleview.com/engineering/symphony-service/commons/enums"
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
	assert.Equal(t, "{\"message\":\"workflowId is a required field\",\"messageCode\":4029}", err.Error())

	//CallType
	//1.Invalid
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "assess"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "{\"message\":\"unsupported calltype\",\"messageCode\":4029}", err.Error())

	//2.Hipster, Status missing
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, CallType: "hipster"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "{\"message\":\"status cannot be empty\",\"messageCode\":4029}", err.Error())

	//3.Eagleflow
	awsClient.Mock.On("InvokeLambda", context.Background(), "", map[string]interface{}{"notes": "", "reportId": "1241243", "status": "MAStarted", "taskName": "", "workflowId": "some-id"}, false).
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
	assert.Equal(t, "{\"message\":\"invalid http request method\",\"messageCode\":4029}", err.Error())

	//2.Empty URL
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, RequestMethod: "POST"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "{\"message\":\"invalid callout request\",\"messageCode\":4029}", err.Error())

	//3.Invalid URL
	req = MyEvent{ReportID: reportID, WorkflowID: workflowId, RequestMethod: "POST", URL: "asdfasd.net"}
	_, err = CallService(context.Background(), req, "")
	assert.Equal(t, "{\"message\":\"url must be a valid URL\",\"messageCode\":4029}", err.Error())

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

	awsClient.Mock.On("InvokeLambda", context.Background(), "", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"status\": \"success\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	awsClient.Mock.On("StoreDataToS3", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 3. Valid POST Call with  wait taask with hipster job
	req := MyEvent{ReportID: reportID, Timeout: 30, Status: "QCCompleted", IsWaitTask: true, CallType: "hipster", TaskToken: "taskToken", WorkflowID: workflowId, RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
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
	_, err = notifcationWrapper(context.Background(), req)
	assert.NoError(t, err)

}

func TestCompleteCalloutFailure(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	slackClient := new(mocks.ISlackClient)
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
	slackClient.On("SendErrorMessage", mock.Anything, reportID, workflowId, "callout", mock.Anything, mock.Anything, map[string]string(nil)).Return(nil)
	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, failure, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.SlackClient = slackClient
	_, err := notifcationWrapper(context.Background(), req)
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

func TestHandleBasicAuthKeyValSecret(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	awsClient.Mock.On("GetSecretString", context.Background(), "SecretManagerArn").Return("{\"ClientID\":\"ClientID\",\r\n\"Secret\":\"Secret\"}", nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	authData := AuthData{
		Type: enums.AuthBasic,
	}
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key_value"
	authData.RequiredAuthData.SecretManagerArn = "SecretManagerArn"
	authData.RequiredAuthData.ClientIDKey = "ClientID"
	authData.RequiredAuthData.ClientSecretKey = "Secret"
	err := handleAuth(context.Background(), authData, map[string]string{})
	assert.NoError(t, err)
}
func TestHandleAuthError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	awsClient.Mock.On("GetSecretString", context.Background(), mock.Anything).Return("", errors.New("some eror"))
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	authData := AuthData{
		Type: enums.AuthBasic,
	}
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key_value"
	authData.RequiredAuthData.SecretManagerArn = "SecretManagerArn"
	authData.RequiredAuthData.ClientIDKey = "ClientID"
	authData.RequiredAuthData.ClientSecretKey = "Secret"
	err := handleAuth(context.Background(), authData, map[string]string{})
	assert.Error(t, err)
	authData.Type = enums.AuthXApiKey
	err = handleAuth(context.Background(), authData, map[string]string{})
	assert.Error(t, err)
	authData.Type = enums.AuthBearer
	err = handleAuth(context.Background(), authData, map[string]string{})
	assert.Error(t, err)
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key"
	authData.Type = enums.AuthXApiKey
	err = handleAuth(context.Background(), authData, map[string]string{})
	assert.Error(t, err)
}
func TestFetchClientIDSecretEror(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	awsClient.Mock.On("GetSecretString", context.Background(), mock.Anything).Return("", errors.New("some error"))
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	authData := AuthData{
		Type: enums.AuthBasic,
	}
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key_value"
	authData.RequiredAuthData.SecretManagerArn = "SecretManagerArn"
	authData.RequiredAuthData.ClientIDKey = "ClientID"
	authData.RequiredAuthData.ClientSecretKey = "Secret"
	_, _, err := fetchClientIdSecret(context.Background(), authData)
	assert.Error(t, err)
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key"
	_, _, err = fetchClientIdSecret(context.Background(), authData)
	assert.Error(t, err)
}
func TestHandleBasicAuthStringSecret(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	awsClient.Mock.On("GetSecretString", context.Background(), mock.Anything).Return("secret", nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	authData := AuthData{
		Type: enums.AuthBasic,
	}
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key"
	authData.RequiredAuthData.ClientIDKey = "ClientID"
	authData.RequiredAuthData.ClientSecretKey = "Secret"
	err := handleAuth(context.Background(), authData, map[string]string{})
	assert.NoError(t, err)
}

func TestHandleX_API_KEY(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	awsClient.Mock.On("GetSecretString", context.Background(), mock.Anything).Return("{\"XAPIKeyKey\":\"XAPIKeyKey\"}", nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	authData := AuthData{
		Type: enums.AuthXApiKey,
	}
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key_value"
	authData.RequiredAuthData.SecretManagerArn = "SecretManagerArn"
	authData.RequiredAuthData.XAPIKeyKey = "XAPIKeyKey"
	err := handleAuth(context.Background(), authData, map[string]string{})
	assert.NoError(t, err)
}
func TestHandleX_API_KEY_stringsecret(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	awsClient.Mock.On("GetSecretString", context.Background(), "XAPIKeyKey").Return("XAPIKey", nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	authData := AuthData{
		Type: enums.AuthXApiKey,
	}
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key"
	authData.RequiredAuthData.XAPIKeyKey = "XAPIKeyKey"
	err := handleAuth(context.Background(), authData, map[string]string{})
	assert.NoError(t, err)
}
func TestHandleBearerAuth(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	awsClient.Mock.On("GetSecretString", context.Background(), "SecretManagerArn").Return("{\"ClientID\":\"ClientID\",\r\n\"Secret\":\"Secret\"}", nil)
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"access_token": "access_token"
		}`))),
	}, nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	authData := AuthData{
		Type: enums.AuthBearer,
	}
	authData.RequiredAuthData.SecretStoreType = "secret_manager_key_value"
	authData.RequiredAuthData.SecretManagerArn = "SecretManagerArn"
	authData.RequiredAuthData.ClientIDKey = "ClientID"
	authData.RequiredAuthData.ClientSecretKey = "Secret"
	authData.RequiredAuthData.URL = "URL"
	err := handleAuth(context.Background(), authData, map[string]string{})
	assert.NoError(t, err)
}
func TestFetchAuthTokenErrorinvalidStatusCode(t *testing.T) {
	httpClient := new(mocks.MockHTTPClient)
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusBadRequest,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"access_token": "access_token"
		}`))),
	}, nil)
	commonHandler.HttpClient = httpClient
	_, err := fetchAuthToken(context.Background(), "URL", "ClientID", "clientSecret", map[string]string{})
	assert.Error(t, err)
}
func TestFetchAuthTokenErrormakingPostCall(t *testing.T) {
	httpClient := new(mocks.MockHTTPClient)
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusAccepted,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"access_token": "access_token"
		}`))),
	}, errors.New("some error"))
	commonHandler.HttpClient = httpClient
	_, err := fetchAuthToken(context.Background(), "URL", "ClientID", "clientSecret", map[string]string{})
	assert.Error(t, err)
}
func TestFetchAuthTokenErrordecoding(t *testing.T) {
	httpClient := new(mocks.MockHTTPClient)
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusAccepted,
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
	}, nil)
	commonHandler.HttpClient = httpClient
	_, err := fetchAuthToken(context.Background(), "URL", "ClientID", "clientSecret", map[string]string{})
	assert.Error(t, err)
}
func TestCallServiceLegacyCallError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"

	awsClient.Mock.On("InvokeLambda", context.Background(), "", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"errorType\": \"RetriableError\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient

	// 3. Valid POST Call with  wait taask with hipster job
	req := MyEvent{ReportID: reportID, Timeout: 30, Status: "QCCompleted", IsWaitTask: true, CallType: "Eagleflow", TaskToken: "taskToken", WorkflowID: workflowId, RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}

	commonHandler.HttpClient = httpClient
	_, err := CallService(context.Background(), req, "1234")
	assert.Error(t, err)
}

func TestCallServiceErrorStoringDataToS3(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	awsClient.Mock.On("StoreDataToS3", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(errors.New("some error"))

	req := MyEvent{ReportID: reportID, Timeout: 30, StoreDataToS3: "s3://app-dev-1x0-s3-symphony-workflow/44823954/imageMetadata.json", Status: "QCCompleted", IsWaitTask: false, CallType: "", TaskToken: "taskToken", WorkflowID: workflowId, RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
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
	req1 := MyEvent{ReportID: reportID, Timeout: 30, StoreDataToS3: "some  location", Status: "QCCompleted", IsWaitTask: false, CallType: "", TaskToken: "taskToken", WorkflowID: workflowId, RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	_, err = CallService(context.Background(), req1, "1234")
	assert.Error(t, err)
}

func TestGetPostPutDeleteError(t *testing.T) {
	httpClient := new(mocks.MockHTTPClient)
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"status": "success"
		}`))),
	}, errors.New("some error"))
	httpClient.Mock.On("Put").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"status": "success"
		}`))),
	}, errors.New("some error"))
	httpClient.Mock.On("Delete").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusBadRequest,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"status": "success"
		}`))),
	}, nil)
	httpClient.Mock.On("Get").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusBadRequest,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"status": "success"
		}`))),
	}, nil)
	httpClient.Mock.On("Getwithbody").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusAccepted,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"jobId": "jobId",
			"status": "success"
		}`))),
	}, errors.New("some error"))
	commonHandler.HttpClient = httpClient
	_, _, err := makePutPostDeleteCall(context.Background(), enums.POST, "some_url", map[string]string{"hello": "world"}, []byte("some  payload"))
	assert.Error(t, err)
	_, _, err = makePutPostDeleteCall(context.Background(), enums.PUT, "some_url", map[string]string{"hello": "world"}, []byte("some  payload"))
	assert.Error(t, err)
	_, _, err = makePutPostDeleteCall(context.Background(), enums.DELETE, "some_url", map[string]string{"hello": "world"}, []byte("some  payload"))
	assert.Error(t, err)
	_, _, err = makeGetCall(context.Background(), "some_url", map[string]string{"hello": "world"}, nil, map[string]string{"hello": "world"})
	assert.Error(t, err)
	_, _, err = makeGetCall(context.Background(), "some_url", map[string]string{"hello": "world"}, []byte(""), map[string]string{"hello": "world"})
	assert.Error(t, err)
}
func TestCallServiceValidationHipsterJobError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	awsClient.Mock.On("StoreDataToS3", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(nil)
	req := MyEvent{ReportID: reportID, Timeout: 30, Status: "QCCompleted", IsWaitTask: true, CallType: "hipster", TaskToken: "taskToken", WorkflowID: workflowId, RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(``))),
	}, nil)
	commonHandler.HttpClient = httpClient
	_, err := CallService(context.Background(), req, "1234")
	assert.Error(t, err)
}
func TestCallServiceValidationHipsterJobErrorMissingJobID(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	awsClient.Mock.On("StoreDataToS3", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(nil)
	req := MyEvent{ReportID: reportID, Timeout: 30, Status: "QCCompleted", IsWaitTask: true, CallType: "hipster", TaskToken: "taskToken", WorkflowID: workflowId, RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
	httpClient.Mock.On("Post").Return(&http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(string(`{}`))),
	}, nil)
	commonHandler.HttpClient = httpClient
	_, err := CallService(context.Background(), req, "1234")
	assert.Error(t, err)
}
func TestCallServiceValidationHipsterJobErrorUpdatingLegacy(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"

	awsClient.Mock.On("InvokeLambda", context.Background(), "", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"status\": \"success\"\n}")}, errors.New(""))
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	awsClient.Mock.On("StoreDataToS3", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 3. Valid POST Call with  wait taask with hipster job
	req := MyEvent{ReportID: reportID, Timeout: 30, Status: "QCCompleted", IsWaitTask: true, CallType: "hipster", TaskToken: "taskToken", WorkflowID: workflowId, RequestMethod: "POST", URL: "http://google.com", Payload: map[string]interface{}{"key": "value"}}
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
	assert.Error(t, err)
}
func TestCallLegacyErrorType(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", context.Background(), "", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"errorType\": \"errorString\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	err := callLegacyStatusUpdate(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
}
func TestCallLegacyErrorUnmarshalling(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", context.Background(), "", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("")}, nil)
	commonHandler.AwsClient = awsClient
	err := callLegacyStatusUpdate(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
}
func TestCallLegacyErrorStatus(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", context.Background(), "", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"status\": \"failure\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	err := callLegacyStatusUpdate(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
}
func TestCallLegacyErrorMissingStatus(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", context.Background(), "", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"key\": \"value\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	err := callLegacyStatusUpdate(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
}

func TestCompleteCalloutSuccessLambdaCall(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", mock.Anything, "lambda function arn", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("")}, nil)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. Valid POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: false, TaskToken: "taskToken", WorkflowID: workflowId, ARN: "lambda function arn", CallType: "lambda", Payload: map[string]interface{}{"key": "value"}}

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, success, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.NoError(t, err)
}
func TestCompleteCalloutFailureLambdaCall(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", mock.Anything, "lambda function arn", mock.Anything, false).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"key\": \"value\"\n}")}, errors.New("error"))
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. Valid POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: false, TaskToken: "taskToken", WorkflowID: workflowId, ARN: "lambda function arn", CallType: "lambda", Payload: map[string]interface{}{"key": "value"}}

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, failure, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.Error(t, err)
}
func TestFailureLambdaCallRetriableError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", mock.Anything, "lambda function arn", mock.Anything, true).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"errorType\": \"RetriableError\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	_, err := callLambda(context.Background(), map[string]interface{}{"key": "value"}, "lambda function arn", true)
	assert.Error(t, err)
}
func TestFailureLambdaCallServiceError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", mock.Anything, "lambda function arn", mock.Anything, true).
		Return(&lambda.InvokeOutput{Payload: []byte("{\n  \"errorType\": \"ServiceError\"\n}")}, nil)
	commonHandler.AwsClient = awsClient
	_, err := callLambda(context.Background(), map[string]interface{}{"key": "value"}, "lambda function arn", true)
	assert.Error(t, err)
}
func TestFailureLambdaCallunmarshalError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("InvokeLambda", mock.Anything, "lambda function arn", mock.Anything, true).
		Return(&lambda.InvokeOutput{Payload: []byte("h")}, nil)
	commonHandler.AwsClient = awsClient
	_, err := callLambda(context.Background(), map[string]interface{}{"key": "value"}, "lambda function arn", true)
	assert.Error(t, err)
}

func TestCompleteCalloutSuccessSQSCall(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("PushMessageToSQS", mock.Anything, "Queue endpoint", mock.Anything).
		Return(nil)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. Valid POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, QueueUrl: "Queue endpoint", CallType: "sqs", Payload: map[string]interface{}{"key": "value"}}

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, success, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.NoError(t, err)
}

func TestCompleteCalloutFailureSQSCall(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("PushMessageToSQS", mock.Anything, "Queue endpoint", mock.Anything).
		Return(errors.New("some error"))
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. Valid POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, QueueUrl: "Queue endpoint", CallType: "sqs", Payload: map[string]interface{}{"key": "value"}}

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, failure, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.Error(t, err)
}
func TestCompleteCalloutSuccessSQSCallWithMeta(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.Mock.On("PushMessageToSQS", mock.Anything, "Queue endpoint", mock.Anything).
		Return(nil)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	reportID := "1241243"
	workflowId := "some-id"
	// 3. Valid POST Call with  wait taask
	req := MyEvent{ReportID: reportID, IsWaitTask: true, TaskToken: "taskToken", WorkflowID: workflowId, QueueUrl: "Queue endpoint", CallType: "sqs", Payload: map[string]interface{}{"key": "value", "meta": map[string]interface{}{"S3URI": ""}}}

	dBClient.Mock.On("InsertStepExecutionData", mock.Anything, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", mock.Anything, req.TaskName, mock.Anything, success, mock.Anything, req.IsWaitTask).Return("update")
	dBClient.Mock.On("UpdateDocumentDB", mock.Anything, mock.Anything, "update", mock.Anything).Return(nil)
	commonHandler.HttpClient = httpClient
	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	_, err := HandleRequest(context.Background(), req)
	assert.NoError(t, err)
}
