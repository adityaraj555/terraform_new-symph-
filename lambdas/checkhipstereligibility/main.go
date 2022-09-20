package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
)

const (
	loglevel    = "info"
	callBackEnv = "callBackLambdaARN"
)

var (
	commonHandler common_handler.CommonHandler
)

type pdwOutput struct {
	Status      string `json:"status"`
	MessageCode int    `json:"messageCode"`
	Message     string `json:"message"`
	CallbackId  string `json:"callbackId"`
	Response    struct {
		Data struct {
			Parcels []struct {
				DetectedBuildingCount struct {
					Value interface{} `json:"value"`
				} `json:"_detectedBuildingCount"`
			} `json:"parcels"`
		} `json:"data"`
	} `json:"response"`
}

func handler(ctx context.Context, input pdwOutput) error {
	isHipsterCompatible := false
	status := "success"

	if input.Status == "success" && len(input.Response.Data.Parcels) > 0 && input.Response.Data.Parcels[0].DetectedBuildingCount.Value != nil {
		if buildingCount := input.Response.Data.Parcels[0].DetectedBuildingCount.Value.(int); buildingCount == 1 {
			isHipsterCompatible = true
		}
	} else if input.Status == "failure" {
		status = "failure"
	}

	response := map[string]interface{}{
		"status":      status,
		"messageCode": input.MessageCode,
		"message":     input.Message,
		"callbackId":  input.CallbackId,
		"response": map[string]interface{}{
			"isHipsterCompatible": isHipsterCompatible,
		},
	}
	callBackLambdaArn := os.Getenv(callBackEnv)

	invokeOut, err := commonHandler.AwsClient.InvokeLambda(ctx, callBackLambdaArn, response, false)
	if err != nil {
		log.Error(ctx, "error invoking callback lambda", invokeOut, err)
		return error_handler.NewServiceError(error_codes.ErrorInvokingLambda, err.Error())
	}

	return nil
}

func notificationWrapper(ctx context.Context, req pdwOutput) error {
	err := handler(ctx, req)
	if err != nil {
		errT := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(errT.GetErrorCode(), "", "", "checkHipsterEligibility", "checkHipsterEligibility", err.Error(), map[string]string{
			"callbackId": req.CallbackId,
		})
	}
	return err
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(true, false, false, true, false)
	lambda.Start(notificationWrapper)
}
