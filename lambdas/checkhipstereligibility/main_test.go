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
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false}, "status": "failure"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestIsHipsterEligibleInvokeFailed(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, struct {
		DetectedBuildingCount struct {
			Value interface{} "json:\"value\""
		} "json:\"_detectedBuildingCount\""
	}{DetectedBuildingCount: struct {
		Value interface{} "json:\"value\""
	}{Value: 0}})
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	slackClient.On("SendErrorMessage", 4041, "", "", "checkHipsterEligibility", "checkHipsterEligibility", mock.Anything, mock.Anything).Return(nil)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false}, "status": "success"}, false).Return(nil, errors.New("some error"))
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.Error(t, err)
}

func TestCheckHipsterEligibilityBuildingCount(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, struct {
		DetectedBuildingCount struct {
			Value interface{} "json:\"value\""
		} "json:\"_detectedBuildingCount\""
	}{DetectedBuildingCount: struct {
		Value interface{} "json:\"value\""
	}{Value: 2}})
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCount0(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, struct {
		DetectedBuildingCount struct {
			Value interface{} "json:\"value\""
		} "json:\"_detectedBuildingCount\""
	}{DetectedBuildingCount: struct {
		Value interface{} "json:\"value\""
	}{Value: 0}})
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCount1(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, struct {
		DetectedBuildingCount struct {
			Value interface{} "json:\"value\""
		} "json:\"_detectedBuildingCount\""
	}{DetectedBuildingCount: struct {
		Value interface{} "json:\"value\""
	}{Value: 1}})
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": true}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}

func TestCheckHipsterEligibilityBuildingCountNil(t *testing.T) {
	in := pdwOutput{
		Status: "success",
	}
	in.Response.Data.Parcels = append(in.Response.Data.Parcels, struct {
		DetectedBuildingCount struct {
			Value interface{} "json:\"value\""
		} "json:\"_detectedBuildingCount\""
	}{DetectedBuildingCount: struct {
		Value interface{} "json:\"value\""
	}{Value: nil}})
	awsClient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	awsClient.On("InvokeLambda", context.Background(), "", map[string]interface{}{"callbackId": "", "message": "", "messageCode": 0, "response": map[string]interface{}{"isHipsterCompatible": false}, "status": "success"}, false).Return(nil, nil)
	commonHandler.SlackClient = slackClient
	commonHandler.AwsClient = awsClient
	err := notificationWrapper(context.Background(), in)
	assert.NoError(t, err)
}
