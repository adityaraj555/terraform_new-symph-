package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

var InvokeSFNRequest string = "{ \"address\": { \"city\": \"Gilroy\", \"country\": \"UnitedStates\", \"latitude\": 37.024966, \"longitude\": -121.583003, \"state\": \"CA\", \"street\": \"270 Ronan Ave\", \"zip\": \"95020\" }, \"reportId\": \"44825849\", \"orderId\": \"44825849\", \"customerNotes\": \"\", \"measurementInstructions\": {}, \"orderType\": \"\" }"

func TestInvokeSFN(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	commonHandler.AwsClient = awsclient
	InvokeSFNRequestObj := events.SQSEvent{}
	InvokeSFNRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: InvokeSFNRequest}}
	awsclient.Mock.On("InvokeSFN", mock.Anything, mock.Anything).Return("ExecutionARN", nil)
	err := Handler(context.Background(), InvokeSFNRequestObj)
	assert.NoError(t, err)
}

func TestInvokeSFNerrorNoBody(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	commonHandler.AwsClient = awsclient
	InvokeSFNRequestObj := events.SQSEvent{}
	InvokeSFNRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: ""}}
	err := Handler(context.Background(), InvokeSFNRequestObj)
	assert.Error(t, err)
}
func TestInvokeSFNerror(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	commonHandler.AwsClient = awsclient
	InvokeSFNRequestObj := events.SQSEvent{}
	InvokeSFNRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: InvokeSFNRequest}}
	awsclient.Mock.On("InvokeSFN", mock.Anything, mock.Anything).Return("ExecutionARN", errors.New("some error"))
	err := Handler(context.Background(), InvokeSFNRequestObj)
	assert.Error(t, err)
}

var InvokeSFNRequestaddressmissing string = "{ \"address\": { \"country\": \"UnitedStates\", \"latitude\": 37.024966, \"longitude\": -121.583003, \"state\": \"CA\", \"street\": \"270 Ronan Ave\", \"zip\": \"95020\" }, \"reportId\": \"44825849\", \"orderId\": \"44825849\", \"customerNotes\": \"\", \"measurementInstructions\": {}, \"orderType\": \"\" }"

func TestInvokeSFNerrorValidation(t *testing.T) {
	awsclient := new(mocks.IAWSClient)
	commonHandler.AwsClient = awsclient
	InvokeSFNRequestObj := events.SQSEvent{}
	InvokeSFNRequestObj.Records = []events.SQSMessage{events.SQSMessage{Body: InvokeSFNRequestaddressmissing}}
	awsclient.Mock.On("InvokeSFN", mock.Anything, mock.Anything).Return("ExecutionARN", nil)
	err := Handler(context.Background(), InvokeSFNRequestObj)
	assert.Error(t, err)
}
