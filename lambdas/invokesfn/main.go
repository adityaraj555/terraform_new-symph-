package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/enums"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/validator"
)

type sfnInput struct {
	Address struct {
		City      string  `json:"city" validate:"required"`
		Country   string  `json:"country" validate:"required"`
		Longitude float64 `json:"longitude" validate:"required"`
		Latitude  float64 `json:"latitude" validate:"required"`
		State     string  `json:"state" validate:"required"`
		Street    string  `json:"street" validate:"required"`
		Zip       string  `json:"zip" validate:"required"`
	}
	OrderID    string        `json:"orderId"`
	ReportID   string        `json:"reportId" validate:"required"`
	WorkflowId string        `json:"workflowId"`
	Source     enums.Sources `json:"source" validate:"source"`
}

type Address struct {
	ParcelAddress string  `json:"parcelAddress"`
	Lat           float64 `json:"lat"`
	Long          float64 `json:"long"`
}

type sfnSIMInput struct {
	Address Address       `json:"address"`
	Meta    *Meta         `json:"meta" validate:"required"`
	Vintage time.Time     `json:"vintage"`
	Source  enums.Sources `json:"source" validate:"source"`
}
type Meta struct {
	CallbackID  string `json:"callbackId" validate:"required"`
	CallbackURL string `json:"callbackUrl" validate:"required"`
}

type NotifyRequest struct {
	CallbackID   string                    `json:"callbackId"`
	ErrorMessage NotifyRequestErrorMessage `json:"errorMessage"`
	CallbackURL  string                    `json:"callbackUrl"`
	WorkflowID   string                    `json:"workflowId"`
}

type NotifyRequestErrorMessage struct {
	Error string `json:"Error"`
	Cause string `json:"Cause"`
}

var (
	commonHandler        common_handler.CommonHandler
	reportId, workflowId string
)

const (
	StateMachineARN      = "StateMachineARN"
	AISStateMachineARN   = "AISStateMachineARN"
	SIMStateMachineARN   = "SIMStateMachineARN"
	SFNNotifierLambdaARN = "SFNNotifierLambdaARN"
	loglevel             = "info"
)

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(true, false, false, true, false)
	lambda.Start(notificationWrapper)
}

func notificationWrapper(ctx context.Context, sqsEvent events.SQSEvent) {
	req, err := Handler(ctx, sqsEvent)
	if err != nil {
		cerr := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(cerr.GetErrorCode(), reportId, workflowId, "", "invokesfn", err.Error(), map[string]string{
			"request": strings.Join(req, " : "),
		})
	}
}

func Handler(ctx context.Context, sqsEvent events.SQSEvent) (req []string, err error) {
	log.Infof(ctx, "Invokesfn Lambda reached...")

	for _, message := range sqsEvent.Records {
		log.Info(ctx, "SQS Message: %+v", message)
		req = append(req, message.Body)

		var sfnreq map[string]interface{}
		err := json.Unmarshal([]byte(message.Body), &sfnreq)
		if err != nil {
			log.Error(ctx, err)
			return req, error_handler.NewServiceError(error_codes.ErrorDecodingInvokeSFNInput, err.Error())
		}
		src, ok := sfnreq["source"]
		if !ok {
			return req, error_handler.NewServiceError(error_codes.ErrorUnknownSource, "Unknown Source")
		}
		err, SFNStateMachineARN, sfnName := GetSfnDataBySource(ctx, message.Body, src.(string))
		if err != nil {
			log.Error(ctx, err)
			return req, err
		}

		ExecutionArn, err := commonHandler.AwsClient.InvokeSFN(&message.Body, &SFNStateMachineARN, &sfnName)
		log.Infof(ctx, "executionARN of Step function:  %s", ExecutionArn)
		if err != nil {
			log.Error(ctx, err)
			// NOTIFY Lambda
			notifyError := NotifyRequestErrorMessage{
				Error: err.Error(),
				Cause: fmt.Sprintf("Unable to trigger workflow as callback Id %s is not unique", sfnreq["callbackId"].(string)),
			}

			payload := map[string]interface{}{
				"CallbackID":   sfnreq["meta"].(map[string]interface{})["callbackId"].(string),
				"CallbackURL":  sfnreq["meta"].(map[string]interface{})["callbackUrl"].(string),
				"ErrorMessage": notifyError,
			}

			sfnnotifierlambdaarn := os.Getenv("SFNNotifierLambdaARN")
			_, invokeErr := commonHandler.AwsClient.InvokeLambda(ctx, sfnnotifierlambdaarn, payload, false)
			log.Error(ctx, invokeErr)
			return req, error_handler.NewServiceError(error_codes.ErrorInvokingStepFunction, err.Error())
		}

	}
	log.Infof(ctx, "Invokesfn Lambda successful...")
	return req, nil
}

func GetSfnDataBySource(ctx context.Context, input string, source string) (error, string, string) {
	log.Info(ctx, "input body:", input)

	switch source {
	case enums.AutoImageSelection, enums.MeasurementAutomation:
		var SFNStateMachineARN string
		sfnreq := sfnInput{}
		err := json.Unmarshal([]byte(input), &sfnreq)
		if err != nil {
			log.Error(ctx, err)
			return error_handler.NewServiceError(error_codes.ErrorDecodingInvokeSFNInput, err.Error()), "", ""
		}
		if err := validator.ValidateInvokeSfnRequest(ctx, sfnreq); err != nil {
			log.Error(ctx, "error in validation: ", err)
			return error_handler.NewServiceError(error_codes.ErrorValidatingCallOutLambdaRequest, err.Error()), "", ""
		}
		if source == enums.AutoImageSelection {
			SFNStateMachineARN = os.Getenv(AISStateMachineARN)
		} else {
			SFNStateMachineARN = os.Getenv(StateMachineARN)
		}
		sfnName := fmt.Sprintf("%s-%s-%s", sfnreq.ReportID, sfnreq.WorkflowId, sfnreq.Source)
		log_config.SetTraceIdInContext(ctx, sfnreq.ReportID, "")
		return nil, SFNStateMachineARN, sfnName

	case enums.SIM:
		sfnsimreq := sfnSIMInput{}
		err := json.Unmarshal([]byte(input), &sfnsimreq)
		if err != nil {
			log.Error(ctx, err)
			return error_handler.NewServiceError(error_codes.ErrorDecodingInvokeSFNInput, err.Error()), "", ""
		}
		if err := validator.ValidateInvokeSfnRequest(ctx, sfnsimreq); err != nil {
			log.Error(ctx, "error in validation: ", err)
			return error_handler.NewServiceError(error_codes.ErrorValidatingCallOutLambdaRequest, err.Error()), "", ""
		}
		SFNStateMachineARN := os.Getenv(SIMStateMachineARN)
		sfnName := fmt.Sprintf("%s-%s", sfnsimreq.Meta.CallbackID, sfnsimreq.Source)
		log_config.SetTraceIdInContext(ctx, sfnsimreq.Meta.CallbackID, "")
		return nil, SFNStateMachineARN, sfnName

	default:
		return error_handler.NewServiceError(error_codes.ErrorUnknownSource, "Unknown Source"), "", ""
	}

}
