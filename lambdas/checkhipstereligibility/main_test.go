package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

func TestIsHipsterCompatibleSIMfailed(t *testing.T) {
	in := pdwOutput{
		Status: "failure",
	}
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false, "facetCount": 0, "buildingCount": 0}, "status": "failure"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestIsHipsterEligibleInvokeFailed(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	pa := parcel{}
	val := 2
	pa.DetectedBuildingCount.Value = &val
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	slackClient.On("SendErrorMessage", 4041, "", "", "checkHipsterEligibility", "checkHipsterEligibility", mock.Anything, mock.Anything).Return(nil)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false, "facetCount": 0, "buildingCount": 2}, "status": "success"}, false).Return(nil, errors.New("some error"))
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.Error(t, err)
}

func TestCheckHipsterEligibilityBuildingCount(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	pa := parcel{}
	val := 2
	pa.DetectedBuildingCount.Value = &val
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false, "facetCount": 0, "buildingCount": 2}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCount0(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}

	pa := parcel{}
	val := 0
	pa.DetectedBuildingCount.Value = &val
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false, "facetCount": 0, "buildingCount": 0}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCount1FC1(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	pa := parcel{}
	val := 1
	main := "main"
	pa.DetectedBuildingCount.Value = &val
	pa.Structures = []structure{{Type: struct {
		Value *string "json:\"value\""
	}{
		Value: &main,
	}, Roof: struct {
		CountRoofFacets struct {
			Value *int "json:\"value\""
		} "json:\"_countRoofFacets\""
	}{CountRoofFacets: struct {
		Value *int "json:\"value\""
	}{
		Value: &val,
	}},
	}}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": true, "facetCount": 1, "buildingCount": 1}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCount1FC2(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	pa := parcel{}
	val := 1
	fc := 2
	main := "main"
	pa.DetectedBuildingCount.Value = &val
	pa.Structures = []structure{{Type: struct {
		Value *string "json:\"value\""
	}{
		Value: &main,
	}, Roof: struct {
		CountRoofFacets struct {
			Value *int "json:\"value\""
		} "json:\"_countRoofFacets\""
	}{CountRoofFacets: struct {
		Value *int "json:\"value\""
	}{
		Value: &fc,
	}},
	}}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": true, "facetCount": 2, "buildingCount": 1}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCount1FC4(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	pa := parcel{}
	val := 1
	fc := 4
	main := "main"
	pa.DetectedBuildingCount.Value = &val
	pa.Structures = []structure{{Type: struct {
		Value *string "json:\"value\""
	}{
		Value: &main,
	}, Roof: struct {
		CountRoofFacets struct {
			Value *int "json:\"value\""
		} "json:\"_countRoofFacets\""
	}{CountRoofFacets: struct {
		Value *int "json:\"value\""
	}{
		Value: &fc,
	}},
	}}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": true, "facetCount": 4, "buildingCount": 1}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCount1FC4NotMain(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	pa := parcel{}
	val := 1
	fc := 4
	main := "Barn"
	pa.DetectedBuildingCount.Value = &val
	pa.Structures = []structure{{Type: struct {
		Value *string "json:\"value\""
	}{
		Value: &main,
	}, Roof: struct {
		CountRoofFacets struct {
			Value *int "json:\"value\""
		} "json:\"_countRoofFacets\""
	}{CountRoofFacets: struct {
		Value *int "json:\"value\""
	}{
		Value: &fc,
	}},
	}}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false, "facetCount": 0, "buildingCount": 1}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCount1FC5(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	pa := parcel{}
	val := 1
	fc := 5
	main := "main"
	pa.DetectedBuildingCount.Value = &val
	pa.Structures = []structure{{Type: struct {
		Value *string "json:\"value\""
	}{
		Value: &main,
	}, Roof: struct {
		CountRoofFacets struct {
			Value *int "json:\"value\""
		} "json:\"_countRoofFacets\""
	}{CountRoofFacets: struct {
		Value *int "json:\"value\""
	}{
		Value: &fc,
	}},
	}}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false, "facetCount": 5, "buildingCount": 1}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCountNil(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	pa := parcel{}
	pa.DetectedBuildingCount.Value = nil
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, pa)
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false, "facetCount": 0, "buildingCount": 0}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}
