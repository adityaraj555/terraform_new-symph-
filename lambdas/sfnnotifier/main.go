package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/labstack/gommon/log"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
)

type eventData struct {
	CallbackID   string       `json:"callbackId"`
	ErrorMessage ErrorMessage `json:"errorMessage"`
	CallbackURL  string       `json:"callbackUrl"`
	WorkflowID   string       `json:"workflowId"`
}

type ErrorMessage struct {
	Error string `json:"Error"`
	Cause string `json:"Cause"`
}
type CauseMessage struct {
	ErrorMessage string `json:"errorMessage"`
	ErrorType    string `json:"errorType"`
}
type Message struct {
	Message     string      `json:"message"`
	MessageCode interface{} `json:"messageCode"`
}

const (
	failure = "failure"
)

var commonHandler common_handler.CommonHandler

func handler(ctx context.Context, eventData eventData) error {
	ctx = log_config.SetTraceIdInContext(ctx, "", eventData.WorkflowID)
	var CauseMessage CauseMessage
	var Message Message
	err := json.Unmarshal([]byte(eventData.ErrorMessage.Cause), &CauseMessage)
	if err != nil {
		log.Error(ctx, "Error while unmarshalling CauseMessage, error: ", err.Error())
		// make callback with empty message and messagecode
		return makeCallBack(ctx, eventData.ErrorMessage.Cause, eventData.CallbackID, eventData.CallbackURL, nil)
	}
	err = json.Unmarshal([]byte(CauseMessage.ErrorMessage), &Message)
	if err != nil {
		log.Error(ctx, "Error while unmarshalling ErrorMessage, error: ", err.Error())
		// make callback with empty message and messagecode
		return makeCallBack(ctx, eventData.ErrorMessage.Cause, eventData.CallbackID, eventData.CallbackURL, nil)
	}
	err = makeCallBack(ctx, Message.Message, eventData.CallbackID, eventData.CallbackURL, Message.MessageCode)
	return err
}

func makeCallBack(ctx context.Context, message, callbackId, callbackUrl string, messageCode interface{}) error {
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	callbackRequest := map[string]interface{}{
		"callbackId":  callbackId,
		"status":      failure,
		"message":     message,
		"messageCode": messageCode,
	}

	ByteArray, err := json.Marshal(callbackRequest)
	if err != nil {
		log.Error(ctx, "Error while marshalling callbackRequest, error: ", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorSerializingCallOutPayload, err.Error())
	}
	_, err = commonHandler.MakePostCall(ctx, callbackUrl, ByteArray, headers)
	if err != nil {
		log.Error(ctx, "Error while making callbackRequest, error: ", err.Error())
		return err
	}
	return nil
}
func notificationWrapper(ctx context.Context, req eventData) error {
	err := handler(ctx, req)
	if err != nil {
		errT := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(errT.GetErrorCode(), "", req.WorkflowID, "sfnnotifier", "sfnnotifier", err.Error(), nil)
	}
	return err
}
func main() {
	log_config.InitLogging("info")
	commonHandler = common_handler.New(false, true, false, true, false)
	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		// APITimeout: 90,
	})
	lambda.Start(notificationWrapper)
}
