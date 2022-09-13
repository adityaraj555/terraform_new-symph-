package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/labstack/gommon/log"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/utils"
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

func (Message *Message) getMessageCode() int {
	if reflect.TypeOf(Message.MessageCode) == reflect.TypeOf("") {
		a, _ := strconv.Atoi(Message.MessageCode.(string))
		return a
	}
	if reflect.TypeOf(Message.MessageCode) == reflect.TypeOf(0.0) {
		floatval := (Message.MessageCode.(float64))
		return int(floatval)
	}
	return Message.MessageCode.(int)
}

const (
	failure = "failure"
	running = "running"
	Timeout = "States.Timeout"
	appCode = "O2"
)

var auth_client utils.AuthTokenInterface = &utils.AuthTokenUtil{}

var commonHandler common_handler.CommonHandler

func handler(ctx context.Context, eventData eventData) error {
	ctx = log_config.SetTraceIdInContext(ctx, "", eventData.WorkflowID)
	var causeMessage CauseMessage
	var message Message
	if eventData.ErrorMessage.Error == Timeout {
		timedouttask := commonHandler.DBClient.GetTimedoutTask(ctx, eventData.WorkflowID)
		return makeCallBack(ctx, fmt.Sprintf("%s Task TimedOut", timedouttask), eventData.CallbackID, eventData.CallbackURL, error_codes.TaskTimedOutError)
	}
	err := json.Unmarshal([]byte(eventData.ErrorMessage.Cause), &causeMessage)
	if err != nil {
		log.Error(ctx, "Error while unmarshalling CauseMessage, error: ", err.Error())
		// make callback with empty message and messagecode
		return makeCallBack(ctx, eventData.ErrorMessage.Cause, eventData.CallbackID, eventData.CallbackURL, error_codes.ErrorRetrievingMsgCode)
	}
	err = json.Unmarshal([]byte(causeMessage.ErrorMessage), &message)
	if err != nil {
		log.Error(ctx, "Error while unmarshalling ErrorMessage, error: ", err.Error())
		// make callback with empty message and messagecode
		return makeCallBack(ctx, causeMessage.ErrorMessage, eventData.CallbackID, eventData.CallbackURL, error_codes.ErrorRetrievingMsgCode)
	}
	err = makeCallBack(ctx, message.Message, eventData.CallbackID, eventData.CallbackURL, message.getMessageCode())
	return err
}

func makeCallBack(ctx context.Context, message, callbackId, callbackUrl string, messageCode int) error {
	callbackRequest := map[string]interface{}{
		"callbackId":  callbackId,
		"status":      failure,
		"message":     message,
		"messageCode": messageCode,
	}
	if strings.HasPrefix(callbackUrl, "arn") {
		_, err := commonHandler.AwsClient.InvokeLambda(ctx, callbackUrl, callbackRequest, false)
		if err != nil {
			log.Error(ctx, "Error while making callbackRequest, error: ", err.Error())
			return err
		}
		return nil
	}
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	secretMap := commonHandler.Secrets
	clientID := secretMap["ClientID"].(string)
	clientSecret := secretMap["ClientSecret"].(string)
	err := auth_client.AddAuthorizationTokenHeader(ctx, commonHandler.HttpClient, headers, appCode, clientID, clientSecret)
	if err != nil {
		log.Error(ctx, "Error while adding token to header, error: ", err.Error())
		return err
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
	commonHandler = common_handler.New(true, true, true, true, true)
	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		// APITimeout: 90,
	})
	lambda.Start(notificationWrapper)
}
