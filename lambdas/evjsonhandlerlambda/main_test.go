package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

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
            "status": "running",
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
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{
			"propertyModelLocation": "s3Location",
		},
	}

	expectedResp := map[string]interface{}{
		"status":       success,
		"legacyStatus": "HipsterQCCompleted",
	}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	awsClient.Mock.On("FetchS3BucketPath", "").Return("", "", nil)
	awsClient.Mock.On("GetDataFromS3", ctx, "", "").Return([]byte("dummy response"), nil)
	legacyClient.Mock.On("UploadMLJsonToEvoss", ctx, workflowData.OrderId, []byte("dummy response")).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerTwisterFlow(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{
			"propertyModelLocation": "s3Location",
		},
	}

	expectedResp := map[string]interface{}{
		"status":       success,
		"legacyStatus": "MLAutomationCompleted",
	}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)
	workflowData.FlowType = "Twister"
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	awsClient.Mock.On("FetchS3BucketPath", "").Return("", "", nil)
	awsClient.Mock.On("GetDataFromS3", ctx, "", "").Return([]byte("dummy response"), nil)
	legacyClient.Mock.On("UploadMLJsonToEvoss", ctx, workflowData.OrderId, []byte("dummy response")).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerFailureCase(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{
			"propertyModelLocation": "s3Location",
		},
	}

	expectedResp := map[string]interface{}{
		"status":       success,
		"legacyStatus": "HipsterQCRejected",
	}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)
	workflowData.StepsPassedThrough[len(workflowData.StepsPassedThrough)-2] = documentDB_client.StepsPassedThroughBody{
		StartTime: 1651826230,
		Status:    failure,
		StepId:    "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		TaskName:  "UpdateHipsterJobAndWaitForQC",
	}
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	awsClient.Mock.On("FetchS3BucketPath", "").Return("", "", nil)
	awsClient.Mock.On("GetDataFromS3", ctx, "", "").Return([]byte("dummy response"), nil)
	legacyClient.Mock.On("UploadMLJsonToEvoss", ctx, workflowData.OrderId, []byte("dummy response")).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerFailureCaseErrorUnknownTask(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{
			"propertyModelLocation": "s3Location",
		},
	}

	expectedResp := map[string]interface{}{"status": "failure"}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)
	workflowData.StepsPassedThrough[len(workflowData.StepsPassedThrough)-2] = documentDB_client.StepsPassedThroughBody{
		StartTime: 1651826230,
		Status:    failure,
		StepId:    "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		TaskName:  "wrong task name",
	}
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	awsClient.Mock.On("FetchS3BucketPath", "").Return("", "", nil)
	awsClient.Mock.On("GetDataFromS3", ctx, "", "").Return([]byte("dummy response"), nil)
	legacyClient.Mock.On("UploadMLJsonToEvoss", ctx, workflowData.OrderId, []byte("dummy response")).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, "record not found in failureTaskOutputMap map", err.Error())
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerDocDbWorkflowDataError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
	}

	expectedResp := map[string]interface{}{
		"status": failure,
	}
	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, errors.New("error here"))

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "error here")
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerFetchStepExecutionDataError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
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
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, errors.New("error"))

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, "error", err.Error())
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerFetchS3BucketPathError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
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
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	awsClient.Mock.On("FetchS3BucketPath", "").Return("", "", errors.New("error"))

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, "error", err.Error())
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerGetDataFromS3Error(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
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
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	awsClient.Mock.On("FetchS3BucketPath", "").Return("", "", nil)
	awsClient.Mock.On("GetDataFromS3", ctx, "", "").Return([]byte("dummy response"), errors.New("error"))

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, "error", err.Error())
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerUploadMLJsonToEvossError(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
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
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	awsClient.Mock.On("FetchS3BucketPath", "").Return("", "", nil)
	awsClient.Mock.On("GetDataFromS3", ctx, "", "").Return([]byte("dummy response"), nil)
	legacyClient.Mock.On("UploadMLJsonToEvoss", ctx, workflowData.OrderId, []byte("dummy response")).Return(errors.New("error"))

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, "error", err.Error())
	assert.Equal(t, expectedResp, resp)
}

func TestHandlerPropertyModelLocationNotPresent(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	httpClient := new(mocks.MockHTTPClient)
	dBClient := new(mocks.IDocDBClient)
	legacyClient := new(mocks.ILegacyClient)

	eventDataObj := eventData{
		WorkflowID:            "",
		ImageMetaDataLocation: "",
	}

	taskdata := documentDB_client.StepExecutionDataBody{
		StepId: "03caaccc-cca9-4f7a-9dee-2d72d6a6a944",
		Output: map[string]interface{}{},
	}

	expectedResp := map[string]interface{}{"status": "failure"}

	workflowData := documentDB_client.WorkflowExecutionDataBody{}
	json.Unmarshal(mockWorkflowDetails, &workflowData)
	ctx := context.Background()

	dBClient.Mock.On("FetchWorkflowExecutionData", eventDataObj.WorkflowID).Return(workflowData, nil)
	dBClient.Mock.On("FetchStepExecutionData", "03caaccc-cca9-4f7a-9dee-2d72d6a6a944").Return(taskdata, nil)
	awsClient.Mock.On("FetchS3BucketPath", "").Return("", "", nil)
	awsClient.Mock.On("GetDataFromS3", ctx, "", "").Return([]byte("dummy response"), nil)
	legacyClient.Mock.On("UploadMLJsonToEvoss", ctx, workflowData.OrderId, []byte("dummy response")).Return(nil)

	commonHandler.AwsClient = awsClient
	commonHandler.DBClient = dBClient
	commonHandler.HttpClient = httpClient
	commonHandler.LegacyClient = legacyClient

	resp, err := handler(ctx, eventDataObj)
	assert.Error(t, err)
	assert.Equal(t, "propertyModelLocation missing from task output", err.Error())
	assert.Equal(t, expectedResp, resp)
}
