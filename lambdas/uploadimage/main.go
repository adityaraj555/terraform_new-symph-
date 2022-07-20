package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/enums"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
)

const (
	legacyEndpoint = "LEGACY_ENDPOINT"
	DBSecretARN    = "DBSecretARN"
	legacyAuthKey  = "TOKEN"
	region         = "us-east-2"
	success        = "success"
	failure        = "failure"
	logLevel       = "info"
	RetriableError = "RetriableError"
)

type eventData struct {
	ReportID       string `json:"orderId"`
	WorkflowID     string `json:"workflowId"`
	ImageMetadata  string `json:"ImageMetadata"`
	Meta           Meta   `json:"meta"`
	SelectedImages []Path `json:"selectedImages"`
}

type Meta struct {
	CallbackID  string `json:"callbackId"`
	CallbackURL string `json:"callbackUrl"`
}

type Path struct {
	S3Path string `json:"S3Path"`
	View   string `json:"View"`
}

type LambdaOutput struct {
	Status      string `json:"status"`
	MessageCode int    `json:"messageCode"`
	Message     string `json:"message"`
}

//var awsClient aws_client.IAWSClient
//var httpClient httpservice.IHTTPClientV2
var commonHandler common_handler.CommonHandler
var lambdaExecutonError = "error occured while executing lambda: %+v"

func handler(ctx context.Context, eventData *eventData) (*LambdaOutput, error) {

	ctx = log_config.SetTraceIdInContext(ctx, eventData.ReportID, eventData.WorkflowID)
	log.Info(ctx, "UpdateImaged reached...")
	var err error
	var lambdaOutput LambdaOutput

	//validation of attributes
	if eventData.ReportID == "" || eventData.ImageMetadata == "" || len(eventData.SelectedImages) == 0 {
		log.Errorf(ctx, "ReportId or ImageMetadata or SelectedImages  cannot be empty, body: %+v", eventData)
		err = errors.New("error validating input missing fields")
		return &LambdaOutput{
			Status:      failure,
			MessageCode: error_codes.ErrorValidationCheck,
			Message:     err.Error(),
		}, err
	}

	//Upload images to evoss
	err = UploadImageToEvoss(ctx, eventData.SelectedImages, eventData.ReportID)
	if err != nil {
		log.Error(ctx, "Error while uploading images to EVOSS: ", err.Error())
		errT := err.(error_handler.ICodedError)
		lambdaOutput = LambdaOutput{
			Status:      failure,
			MessageCode: errT.GetErrorCode(),
			Message:     err.Error(),
		}
		res, callBackErr := InvokeLambdaforCallback(ctx, eventData.Meta, eventData.ReportID, eventData.WorkflowID, lambdaOutput)
		if callBackErr != nil {
			log.Error(ctx, "Error while calling callback lambda, error: ", callBackErr.Error(), res)
		}
		return nil, error_handler.NewServiceError(error_codes.ErrorWhileUploadImageToEVOSS, err.Error())
	}
	log.Info(ctx, "Images successfully uploaded to EVOSS...")

	//Upload imagemetadata to legacy
	err = UploadImageMetadata(ctx, eventData.ImageMetadata, eventData.ReportID)
	if err != nil {
		log.Error(ctx, "Error while uploading images to EVOSS: ", err.Error())
		errT := err.(error_handler.ICodedError)
		lambdaOutput = LambdaOutput{
			Status:      failure,
			MessageCode: errT.GetErrorCode(),
			Message:     err.Error(),
		}
		res, callBackErr := InvokeLambdaforCallback(ctx, eventData.Meta, eventData.ReportID, eventData.WorkflowID, lambdaOutput)
		if callBackErr != nil {
			log.Error(ctx, "Error while calling callback lambda, error: ", callBackErr.Error(), res)
		}
		return nil, error_handler.NewServiceError(error_codes.ErrorWhileUploadImageMetaDataEVOSS, err.Error())
	}
	log.Info(ctx, "ImageMetadata uploaded successfully...")

	//Invoke callback lambda
	lambdaOutput = LambdaOutput{
		Status:      success,
		MessageCode: 200,
		Message:     "upload image to evoss and upload imagedatametadata successfully",
	}
	res, err := InvokeLambdaforCallback(ctx, eventData.Meta, eventData.ReportID, eventData.WorkflowID, lambdaOutput)
	if err != nil {
		log.Error(ctx, "Error while calling callback lambda, error: ", err.Error(), res)
		return nil, error_handler.NewServiceError(error_codes.ErrorInvokingLambda, err.Error())
	}
	log.Info(ctx, "Callback lambda successful for UpdateImage...")
	log.Info(ctx, "UpdateImaged lambda successful...")
	return &lambdaOutput, nil
}

func InvokeLambdaforCallback(ctx context.Context, meta Meta, reportId, workflowId string, lambdaOutput LambdaOutput) (map[string]string, error) {

	payload := map[string]interface{}{
		"status":      lambdaOutput.Status,
		"message":     lambdaOutput.Message,
		"messageCode": "",
		"callbackId":  meta.CallbackID,
		"response":    map[string]interface{}{},
	}

	result, err := commonHandler.AwsClient.InvokeLambda(ctx, meta.CallbackURL, payload, false)
	if err != nil {
		return nil, error_handler.NewServiceError(error_codes.ErrorInvokingCalloutLambdaFromEVMLConverter, err.Error())
	}
	var resp map[string]string
	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		return nil, error_handler.NewServiceError(error_codes.ErrorDecodingLambdaOutput, err.Error())
	}
	errorType, ok := resp["errorType"]
	if ok {
		log.Errorf(ctx, lambdaExecutonError, errorType)
		return resp, error_handler.NewServiceError(error_codes.LambdaExecutionError, fmt.Sprintf(lambdaExecutonError, errorType))
	}

	return resp, nil
}

func UploadImageToEvoss(ctx context.Context, paths []Path, reportId string) error {
	log.Infof(ctx, "UploadImageToEvoss Reached")
	var fileTypeId string
	var location string
	fileFormatId := 1
	var err error

	for _, path := range paths {
		if path.View == "O" {
			fileTypeId = enums.TopImage
			location = path.S3Path
		} else if path.View == "E" {
			fileTypeId = enums.EastImage
			location = path.S3Path
		} else if path.View == "W" {
			fileTypeId = enums.WestImage
			location = path.S3Path
		} else if path.View == "N" {
			fileTypeId = enums.NorthImage
			location = path.S3Path
		} else if path.View == "S" {
			fileTypeId = enums.SouthImage
			location = path.S3Path
		}
		endpoint := os.Getenv(legacyEndpoint)
		//https://intranetrest.cmh.reportsprod.evinternal.net/UploadReportFile?reportId={reportId}&fileTypeId={fileTypeId}&fileFormatId={fileFormatId}
		url := fmt.Sprintf("%s/UploadReportFile?reportId=%s&fileTypeId=%s&fileFormatId=%s", endpoint, reportId, fileTypeId, strconv.Itoa(fileFormatId))
		log.Info(ctx, "Endpoint: "+url)
		err = UploadData(ctx, reportId, location, url, false)
		if err != nil {
			return error_handler.NewServiceError(error_codes.ErrorWhileUpdatingLegacy, err.Error())

		}
		log.Info(ctx, "Update Image successful...")
	}
	return err
}

func UploadImageMetadata(ctx context.Context, imageMetadata string, reportId string) error {
	var err error
	endpoint := os.Getenv(legacyEndpoint)
	url := fmt.Sprintf("%s/StoreImageMetadata", endpoint)
	log.Info(ctx, "Endpoint: "+url)
	err = UploadData(ctx, reportId, imageMetadata, url, true)
	if err == nil {
		log.Info(ctx, "Upload ImageMetadata successful...")
	}
	return err
}

func UploadData(ctx context.Context, reportId string, location string, url string, isImageMetadata bool) error {
	log.Infof(ctx, "Reached Upload Data with reportId = %s,location =%s, url=%s isImageMetadata=%v", reportId, location, url, isImageMetadata)
	fmt.Println(commonHandler, commonHandler.AwsClient)
	host, loc, err := commonHandler.AwsClient.FetchS3BucketPath(location)
	if err != nil {
		log.Error(ctx, "Error in fetching AWS path: ", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorFetchingS3BucketPath, err.Error())
	}

	ByteArray, err := commonHandler.AwsClient.GetDataFromS3(ctx, host, loc)
	if err != nil {
		log.Error(ctx, "Error in getting downloading from s3: ", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorFetchingDataFromS3, err.Error())
	}

	authsecret := os.Getenv(DBSecretARN)
	secretMap, err := commonHandler.AwsClient.GetSecret(ctx, authsecret, region)
	if err != nil {
		log.Error(ctx, "error while fetching auth token from secret manager", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorFetchingSecretsFromSecretManager, err.Error())
	}

	token, ok := secretMap[legacyAuthKey].(string)
	if !ok {
		log.Error(ctx, "Issue with parsing Auth Token: ", secretMap[legacyAuthKey])
		return error_handler.NewServiceError(error_codes.ErrorParsingLegacyAuthToken, fmt.Sprintf("Issue with parsing Auth Token: %+v", secretMap[legacyAuthKey]))
	}

	headers := map[string]string{
		"Authorization": "Basic " + token,
	}
	if !isImageMetadata {
		base64EncodedString := base64.StdEncoding.EncodeToString(ByteArray)
		ByteArray, err = json.Marshal(base64EncodedString)
		if err != nil {
			return error_handler.NewServiceError(error_codes.ErrorWhileMarshlingData, "Issue while Marshling Data")
		}
	}
	response, err := commonHandler.HttpClient.Post(ctx, url, bytes.NewReader(ByteArray), headers)

	if err != nil {
		log.Error(ctx, "Error while making http call for upload image to evoss, error: ", err)
		// return error_handler.NewServiceError(error_codes.ErrorMakingPostPutOrDeleteCall, err.Error())
	}

	if response.StatusCode == http.StatusInternalServerError || response.StatusCode == http.StatusServiceUnavailable {
		return error_handler.NewRetriableError(error_codes.ErrorWhileUpdatingLegacy, fmt.Sprintf("%d status code received", response.StatusCode))
	}
	if !strings.HasPrefix(strconv.Itoa(response.StatusCode), "20") {
		log.Error(ctx, "response not ok: ", response.StatusCode)
		return error_handler.NewServiceError(error_codes.ErrorWhileUpdatingLegacy, fmt.Sprintf("response not ok got = %d", response.StatusCode))
	}
	return nil
}

func main() {
	log_config.InitLogging(logLevel)
	commonHandler = common_handler.New(true, true, true, true)
	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		// APITimeout: 90,
	})
	lambda.Start(handler)
}
