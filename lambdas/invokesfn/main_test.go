package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

var InvokeSFNRequest string = "{ \"address\": { \"city\": \"Gilroy\", \"country\": \"UnitedStates\", \"latitude\": 37.024966, \"longitude\": -121.583003, \"state\": \"CA\", \"street\": \"270 Ronan Ave\", \"zip\": \"95020\" }, \"reportId\": \"44825849\", \"orderId\": \"44825849\", \"customerNotes\": \"\", \"measurementInstructions\": {}, \"orderType\": \"\", \"source\": \"AIS\" }"

func TestInvokeSFN(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	commonHandler.AwsClient = awsclient
	InvokeSFNRequestObj := events.SQSEvent{}
	InvokeSFNRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: InvokeSFNRequest}}
	awsclient.Mock.On("InvokeSFN", mock.Anything, mock.Anything, mock.Anything).Return("ExecutionARN", nil)
	notificationWrapper(context.Background(), InvokeSFNRequestObj)
}

var InvokeSFNSIMRequest string = "{\"address\": { \"parcelAddress\":\"23 HAVENSHIRE RD, ROCHESTER, NY, 14625\",        \"lat\":  43.172988,        \"long\":  -77.501957    },    \"meta\":{        \"callbackId\":\"callback-test-00001\",        \"callbackUrl\":\"callback\"    },    \"vintage\":\"2017-08-16T09:19:47.051096+00:00\"  }"

func TestInvokeSFNSIM(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	slackClient.On("SendErrorMessage", mock.Anything, mock.Anything, "", mock.Anything, "invokesfn", mock.Anything, mock.Anything).Return(nil)
	commonHandler.AwsClient = awsclient
	commonHandler.SlackClient = slackClient
	InvokeSFNSIMRequestObj := events.SQSEvent{}
	InvokeSFNSIMRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: InvokeSFNSIMRequest}}
	awsclient.Mock.On("InvokeSFN", mock.Anything, mock.Anything, mock.Anything).Return("ExecutionARN", nil)
	notificationWrapper(context.Background(), InvokeSFNSIMRequestObj)
}
func TestInvokeSFNerrorNoBody(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	slackClient := new(mocks.ISlackClient)
	slackClient.On("SendErrorMessage", mock.Anything, mock.Anything, "", mock.Anything, "invokesfn", mock.Anything, mock.Anything).Return(nil)
	commonHandler.AwsClient = awsclient
	commonHandler.SlackClient = slackClient
	InvokeSFNRequestObj := events.SQSEvent{}
	InvokeSFNRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: ""}}
	_, err := Handler(context.Background(), InvokeSFNRequestObj)
	assert.Error(t, err)
}
func TestInvokeSFNerror(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	commonHandler.AwsClient = awsclient
	InvokeSFNRequestObj := events.SQSEvent{}
	InvokeSFNRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: InvokeSFNRequest}}
	awsclient.Mock.On("InvokeSFN", mock.Anything, mock.Anything, mock.Anything).Return("ExecutionARN", errors.New("some error"))
	awsclient.Mock.On("InvokeLambda", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&lambda.InvokeOutput{Payload: []byte("")}, nil)
	_, err := Handler(context.Background(), InvokeSFNRequestObj)
	assert.Error(t, err)
}

var InvokeSFNRequestaddressmissing string = "{ \"address\": { \"country\": \"UnitedStates\", \"latitude\": 37.024966, \"longitude\": -121.583003, \"state\": \"CA\", \"street\": \"270 Ronan Ave\", \"zip\": \"95020\" }, \"reportId\": \"44825849\", \"orderId\": \"44825849\", \"customerNotes\": \"\", \"measurementInstructions\": {}, \"orderType\": \"\" }"

func TestInvokeSFNerrorValidation(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	commonHandler.AwsClient = awsclient
	InvokeSFNRequestObj := events.SQSEvent{}
	InvokeSFNRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: InvokeSFNRequestaddressmissing}}
	awsclient.Mock.On("InvokeSFN", mock.Anything, mock.Anything, mock.Anything).Return("ExecutionARN", nil)
	_, err := Handler(context.Background(), InvokeSFNRequestObj)
	assert.Error(t, err)
}
