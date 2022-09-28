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
	commonHandler   common_handler.CommonHandler
	validFacetCount = []int{1, 2, 4}
)

type pdwOutput struct {
	Status      string `json:"status"`
	MessageCode int    `json:"messageCode"`
	Message     string `json:"message"`
	CallbackId  string `json:"callbackId"`
	Response    struct {
		Data struct {
			Parcels []parcel `json:"parcels"`
		} `json:"data"`
	} `json:"response"`
}

type parcel struct {
	DetectedBuildingCount struct {
		Value *int `json:"value"`
	} `json:"_detectedBuildingCount"`
	Structures []structure `json:"structures"`
}
type structure struct {
	Type struct {
		Value *string `json:"value"`
	} `json:"_type"`
	Roof struct {
		CountRoofFacets struct {
			Value *int `json:"value"`
		} `json:"_countRoofFacets"`
	} `json:"roof"`
}

func handler(ctx context.Context, input pdwOutput) error {
	notMultiStructure := false
	isValidFacetCount := false
	status := "success"
	bCount := 0
	facetCount := 0

	/*
		case 1 : bc = 1, fC = [1, 2, 4] , true
		case 2 : bc = 0, >2, fc = [1,2,4], false
		case 3,  bc = 1, fc != [1,2,4] : false
		case 4 : bc = 1, main structure not found, false
		case 5 : bc = 0, >2, fc != [1,2,4], false
	*/

	if input.Status == "success" && len(input.Response.Data.Parcels) > 0 && input.Response.Data.Parcels[0].DetectedBuildingCount.Value != nil {
		bCount = *input.Response.Data.Parcels[0].DetectedBuildingCount.Value
		if *input.Response.Data.Parcels[0].DetectedBuildingCount.Value == 1 {
			notMultiStructure = true
		}
	}

	if input.Status == "success" && len(input.Response.Data.Parcels) > 0 && len(input.Response.Data.Parcels[0].Structures) > 0 {
		for _, s := range input.Response.Data.Parcels[0].Structures {
			if s.Type.Value != nil && *s.Type.Value == "main" {
				if s.Roof.CountRoofFacets.Value != nil {
					isValidFacetCount = findInIntArray(validFacetCount, *s.Roof.CountRoofFacets.Value)
					facetCount = *s.Roof.CountRoofFacets.Value
				}
				break
			}
		}
	}

	if input.Status == "failure" {
		status = "failure"
	}

	response := map[string]interface{}{
		"status":      status,
		"messageCode": input.MessageCode,
		"message":     input.Message,
		"callbackId":  input.CallbackId,
		"response": map[string]interface{}{
			"isHipsterCompatible": notMultiStructure && isValidFacetCount,
			"buildingCount":       bCount,
			"facetCount":          facetCount,
			"isValidFacetCount":   isValidFacetCount,
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

func findInIntArray(arr []int, val int) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
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
