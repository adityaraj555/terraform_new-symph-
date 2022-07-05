package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

var (
	commonHandler        common_handler.CommonHandler
	reportId, workflowId string
)

const (
	StateMachineARN    = "StateMachineARN"
	AISStateMachineARN = "AISStateMachineARN"
	loglevel           = "info"
)

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(true, false, false, true)
	lambda.Start(notificationWrapper)
}

func notificationWrapper(ctx context.Context, sqsEvent events.SQSEvent) error {
	req, err := Handler(ctx, sqsEvent)
	if err != nil {
		cerr := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(cerr.GetErrorCode(), reportId, workflowId, "invokesfn", err.Error(), map[string]string{
			"request": strings.Join(req, " : "),
		})
	}
	return err
}

func Handler(ctx context.Context, sqsEvent events.SQSEvent) (req []string, err error) {
	log.Infof(ctx, "Invokesfn Lambda reached...")
	var SFNStateMachineARN string

	for _, message := range sqsEvent.Records {
		log.Info(ctx, "SQS Message: %+v", message)
		req = append(req, message.Body)
		if err = validateInput(ctx, message.Body); err != nil {
			return req, err
		}
		sfnreq := sfnInput{}
		err := json.Unmarshal([]byte(message.Body), &sfnreq)
		if err != nil {
			log.Error(ctx, err)
			return req, error_handler.NewServiceError(error_codes.ErrorDecodingInvokeSFNInput, err.Error())
		}

		sfnName := fmt.Sprintf("%s-%s-%s", sfnreq.ReportID, sfnreq.WorkflowId, sfnreq.Source.String())

		switch sfnreq.Source {
		case enums.AutoImageSelection:
			SFNStateMachineARN = os.Getenv(AISStateMachineARN)
		default:
			SFNStateMachineARN = os.Getenv(StateMachineARN)
		}

		ExecutionArn, err := commonHandler.AwsClient.InvokeSFN(&message.Body, &SFNStateMachineARN, &sfnName)
		log.Infof(ctx, "executionARN of Step function:  %s", ExecutionArn)
		if err != nil {
			log.Error(ctx, err)
			return req, error_handler.NewServiceError(error_codes.ErrorInvokingStepFunction, err.Error())
		}
	}
	log.Infof(ctx, "Invokesfn Lambda successful...")
	return req, err
}

func validateInput(ctx context.Context, input string) error {
	log.Info(ctx, "input body:", input)
	req := sfnInput{}
	err := json.Unmarshal([]byte(input), &req)
	log.Info(ctx, req)
	if err != nil {
		log.Error(ctx, "invalid input for sfn", input)
		return error_handler.NewServiceError(error_codes.ErrorDecodingInvokeSFNInput, err.Error())
	}

	reportId = req.ReportID
	workflowId = req.WorkflowId

	if err := validator.ValidateInvokeSfnRequest(ctx, req); err != nil {
		log.Error(ctx, "error in validation: ", err)
		return error_handler.NewServiceError(error_codes.ErrorValidatingCallOutLambdaRequest, err.Error())
	}

	log_config.SetTraceIdInContext(ctx, req.ReportID, "")
	return nil
}
