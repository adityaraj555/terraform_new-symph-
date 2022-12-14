package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

func init() {
	os.Setenv("tasksWithPMFOutput", "3DModellingService,CreateHipsterJobAndWaitForMeasurement,UpdateHipsterJobAndWaitForQC")
}

var testContext = log_config.SetTraceIdInContext(context.Background(), "", "")

var mockWorkflowDetails = []byte(`{
    "_id": "a9c9c1d6-3afb-a119-4f8f-34a66461a7db",
    "createdAt": 1651826229,
    "finishedAt": 1651826268,
    "initialInput": {
        "address": {
            "city": "Grand Junction",
            "country": "UnitedStates",
            "latitude": 39.097281,
            "longitude": -108.591335,
            "state": "CO",
            "street": "2489 Apex Ave",
            "zip": "81505"
        },
        "customerNotes": "",
        "measurementInstructions": {},
        "orderId": "28741229",
        "reportId": "28741229"
    },
    "orderId": "28741229",
    "runningState": {
        "UpdateHipsterJobAndWaitForQC": "success"
    },
    "status": "running",
    "stepsPassedThrough": [
        {
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "UpdateLegacyMLAutomationStart"
        },
        {
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "GetImageMetaData"
        },
        {
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "ImageSelection"
        },
        {
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "FacetKeyPointDetection"
        },
        {
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "3DModellingService"
        },
        {
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "UpdateLegacyMLAutomationComplete"
        },{
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "CreateHipsterJobAndWaitForMeasurement"
        },{
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "UpdateHipsterMeasurementCompleteInLegacy"
        },{
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "UpdateHipsterJobAndWaitForQC"
        },
        {
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "UpdateHipsterJobAndWaitForQC"
        },
		{
            "startTime": 1651826230,
            "status": "success",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "ConvertPropertyModelToEVJson"
        },
		{
            "startTime": 1651826230,
            "status": "failure",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "UploadMLJsonToEvoss"
        },
		{
            "startTime": 1651826230,
            "status": "failure",
            "stepId": "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
            "taskName": "EVMLJsonConverter_UploadToEvoss"
        }
    ],
    "updatedAt": 1651826267
}`)

func TestHandler(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)

	eventDataObj := eventData{
		WorkflowID: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{
			"propertyModelLocation": "s3Location",
		},
	}

	expectedResp := map[string]interface{}{
		"status":              success,
		"legacyStatus":        "QCCompleted",
		"propertyModelS3Path": "s3Location",
		"path":                "Hipster",
	}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)

	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", testContext, "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	dBClient.Mock.On("InsertStepExecutionData", testContext, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", testContext, taskName, mock.Anything, success, mock.Anything, false).Return(nil)
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, nil, mock.Anything).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient

	resp, err := notificationWrapper(context.Background(), eventDataObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerTwisterFlow(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)

	eventDataObj := eventData{
		WorkflowID: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{
			"propertyModelLocation": "s3Location",
		},
	}

	expectedResp := map[string]interface{}{
		"status":              success,
		"legacyStatus":        "MACompleted",
		"propertyModelS3Path": "s3Location",
		"path":                "Twister",
	}

	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)
	workflowData.FlowType = "Twister"

	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", testContext, "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	dBClient.Mock.On("InsertStepExecutionData", testContext, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", testContext, taskName, mock.Anything, success, mock.Anything, false).Return(nil)
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, nil, mock.Anything).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient

	resp, err := handler(context.Background(), eventDataObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerFailureCase(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)

	eventDataObj := eventData{
		WorkflowID: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{
			"propertyModelLocation": "s3Location",
		},
	}

	expectedResp := map[string]interface{}{
		"status":              success,
		"legacyStatus":        "QCFailed",
		"propertyModelS3Path": "s3Location",
		"path":                "Twister",
	}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)
	workflowData.StepsPassedThrough[len(workflowData.StepsPassedThrough)-1] = documentDB_client.StepsPassedThroughBody{
		StartTime: 1651826230,
		Status:    failure,
		StepId:    "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		TaskName:  "UpdateHipsterJobAndWaitForQC",
	}

	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", testContext, "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	dBClient.Mock.On("InsertStepExecutionData", testContext, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", testContext, taskName, mock.Anything, success, mock.Anything, false).Return(nil)
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, nil, mock.Anything).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient

	resp, err := handler(context.Background(), eventDataObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerDocDbWorkflowDataError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)

	eventDataObj := eventData{
		WorkflowID: "",
	}

	expectedResp := map[string]interface{}{
		"status": failure,
	}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)

	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, eventDataObj.WorkflowID).Return(workflowData, errors.New("error here"))
	dBClient.Mock.On("InsertStepExecutionData", testContext, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", testContext, taskName, mock.Anything, failure, mock.Anything, false).Return(nil)
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, nil, mock.Anything).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient

	resp, err := handler(context.Background(), eventDataObj)
	assert.Error(t, err)
	// assert.Equal(t, err.Error(), "error here")
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerFetchStepExecutionDataError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)

	eventDataObj := eventData{
		WorkflowID: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{
			"propertyModelLocation": "s3Location",
		},
	}

	expectedResp := map[string]interface{}{
		"status": failure,
	}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)

	dBClient.Mock.On("FetchWorkflowExecutionData", testContext, eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", testContext, "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, errors.New("error"))
	dBClient.Mock.On("InsertStepExecutionData", testContext, mock.Anything).Return(nil)
	dBClient.Mock.On("BuildQueryForUpdateWorkflowDataCallout", testContext, taskName, mock.Anything, failure, mock.Anything, false).Return(nil)
	dBClient.Mock.On("UpdateDocumentDB", testContext, mock.Anything, nil, mock.Anything).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient

	resp, err := handler(context.Background(), eventDataObj)
	assert.Error(t, err)
	// assert.Equal(t, "error", err.Error())
	assert.Equal(t, expectedResp, resp)
}
