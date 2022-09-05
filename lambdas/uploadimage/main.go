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
	"sync"

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
	ReportID       string `json:"reportId"`
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

		return nil, err
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

		return nil, err
	}
	log.Info(ctx, "ImageMetadata uploaded successfully...")

	//Invoke callback lambda
	lambdaOutput = LambdaOutput{
		Status:      success,
		MessageCode: 200,
		Message:     "upload image to evoss and upload imagedatametadata successfully",
	}

	log.Info(ctx, "UpdateImaged lambda successful...")
	return &lambdaOutput, nil
}

func UploadImageToEvoss(ctx context.Context, paths []Path, reportId string) error {
	log.Infof(ctx, "UploadImageToEvoss Reached")
	var fileTypeId string
	var location string
	fileFormatId := 1
	var err error

	var wg sync.WaitGroup
	wg.Add(len(paths))
	errChan := make(chan error, len(paths))
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

		if len(path.S3Path) == 0 {
			return error_handler.NewServiceError(error_codes.ErrorMissingS3Path, err.Error())
		} else {
			endpoint := os.Getenv(legacyEndpoint)
			splittedS3Path := strings.SplitAfterN(path.S3Path, "/", -1)
			filename := strings.Split(splittedS3Path[len(splittedS3Path)-1], ".")[0]
			filename = strings.ReplaceAll(filename, "-", "_")
			url := fmt.Sprintf("%s/UploadReportFile?reportId=%s&fileTypeId=%s&fileFormatId=%s&fileName=%s", endpoint, reportId, fileTypeId, strconv.Itoa(fileFormatId), filename)
			log.Info(ctx, "Endpoint: "+url)

			go UploadData(ctx, reportId, location, url, false, errChan, &wg)
		}
	}
	wg.Wait()
	close(errChan)
	for i := 0; i < len(paths); i++ {
		ch := <-errChan
		if ch != nil {
			return ch
		}
	}

	log.Info(ctx, "Update Image successful...")
	return nil
}

func UploadImageMetadata(ctx context.Context, imageMetadata string, reportId string) error {
	var err error
	endpoint := os.Getenv(legacyEndpoint)
	url := fmt.Sprintf("%s/StoreImageMetadata", endpoint)
	log.Info(ctx, "Endpoint: "+url)
	errImageMetaDataChan := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	UploadData(ctx, reportId, imageMetadata, url, true, errImageMetaDataChan, &wg)
	err = <-errImageMetaDataChan
	close(errImageMetaDataChan)
	if err == nil {
		log.Info(ctx, "Upload ImageMetadata successful...")
	}
	return err
}

func UploadData(ctx context.Context, reportId string, location string, url string, isImageMetadata bool, errChan chan error, wg *sync.WaitGroup) {
	log.Infof(ctx, "Reached Upload Data with reportId = %s,location =%s, url=%s isImageMetadata=%v", reportId, location, url, isImageMetadata)
	defer wg.Done()
	host, loc, err := commonHandler.AwsClient.FetchS3BucketPath(location)
	if err != nil {
		log.Error(ctx, "Error in fetching AWS path: ", err.Error())
		errChan <- error_handler.NewServiceError(error_codes.ErrorFetchingS3BucketPath, err.Error())
		return
	}

	ByteArray, err := commonHandler.AwsClient.GetDataFromS3(ctx, host, loc)
	if err != nil {
		log.Error(ctx, "Error in getting downloading from s3: ", err.Error())
		errChan <- error_handler.NewServiceError(error_codes.ErrorFetchingDataFromS3, err.Error())
		return
	}

	secretMap := commonHandler.Secrets

	token, ok := secretMap[legacyAuthKey].(string)
	if !ok {
		log.Error(ctx, "Issue with parsing Auth Token: ", secretMap[legacyAuthKey])
		errChan <- error_handler.NewServiceError(error_codes.ErrorParsingLegacyAuthToken, fmt.Sprintf("Issue with parsing Auth Token: %+v", secretMap[legacyAuthKey]))
		return
	}

	headers := map[string]string{
		"Authorization": "Basic " + token,
	}
	if !isImageMetadata {
		base64EncodedString := base64.StdEncoding.EncodeToString(ByteArray)
		ByteArray, err = json.Marshal(base64EncodedString)
		if err != nil {
			errChan <- error_handler.NewServiceError(error_codes.ErrorWhileMarshlingData, "Issue while Marshling Data")
			return
		}
	}
	response, err := commonHandler.HttpClient.Post(ctx, url, bytes.NewReader(ByteArray), headers)

	if err != nil {
		log.Error(ctx, "Error while making http call for upload image to evoss, error: ", err)
		errChan <- error_handler.NewServiceError(error_codes.ErrorMakingPostPutOrDeleteCall, err.Error())
		return
	}

	if response.StatusCode == http.StatusInternalServerError || response.StatusCode == http.StatusServiceUnavailable {
		errChan <- error_handler.NewRetriableError(error_codes.ErrorWhileUpdatingLegacy, fmt.Sprintf("%d status code received", response.StatusCode))
		return
	}
	if !strings.HasPrefix(strconv.Itoa(response.StatusCode), "20") {
		log.Error(ctx, "response not ok: ", response.StatusCode)
		errChan <- error_handler.NewServiceError(error_codes.ErrorWhileUpdatingLegacy, fmt.Sprintf("response not ok got = %d", response.StatusCode))
		return
	}
	errChan <- nil
}

func main() {
	log_config.InitLogging(logLevel)
	commonHandler = common_handler.New(true, true, true, true, true)
	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		// APITimeout: 90,
	})
	lambda.Start(handler)
}
