package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.eagleview.com/engineering/assess-platform-library/log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/enums"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/validator"
	"go.mongodb.org/mongo-driver/bson"
)

type Meta struct {
	CallbackID  string `json:"callbackId"`
	CallbackURL string `json:"callbackUrl"`
}

type AuthData struct {
	Type             enums.AuthType `json:"type" validate:"omitempty,authType"`
	RequiredAuthData struct {
		SecretStoreType  string            `json:"secretStoreType"`
		URL              string            `json:"url,omitempty"`
		Headers          map[string]string `json:"Headers,omitempty"`
		Payload          struct{}          `json:"Payload,omitempty"`
		SecretManagerArn string            `json:"secretManagerArn,omitempty"`
		ClientIDKey      string            `json:"clientIdKey,omitempty"`
		ClientSecretKey  string            `json:"clientSecretKey,omitempty"`
		BearerTokenKey   string            `json:"bearerTokenKey,omitempty"`
		XAPIKeyKey       string            `json:"X-API-Key_Key,omitempty"`
	} `json:"authData,omitempty"`
}

type MyEvent struct {
	Payload              interface{}         `json:"requestData"`
	URL                  string              `json:"url" validate:"omitempty,url"`
	ARN                  string              `json:"arn"`
	QueueUrl             string              `json:"queueUrl"`
	RequestMethod        enums.RequestMethod `json:"requestMethod" validate:"omitempty,httpMethod"`
	Headers              map[string]string   `json:"headers"`
	IsWaitTask           bool                `json:"isWaitTask"`
	Timeout              int                 `json:"timeout"`
	GetRequestBodyFromS3 string              `json:"getRequestBodyFromS3"`
	S3RequestBodyType    string              `json:"s3RequestBodyType"`
	StoreDataToS3        string              `json:"storeDataToS3"`
	TaskName             string              `json:"taskName"`
	CallType             enums.CallType      `json:"callType" validate:"omitempty,callTypes"`
	OrderID              string              `json:"orderId"`
	ReportID             string              `json:"reportId"`
	WorkflowID           string              `json:"workflowId" validate:"required"`
	TaskToken            string              `json:"taskToken" validate:"required_if=IsWaitTask true"`
	HipsterJobID         string              `json:"hipsterJobId,omitempty"`
	QueryParam           map[string]string   `json:"queryParam,omitempty"`
	Auth                 AuthData            `json:"auth"`
	Status               string              `json:"status"`
	ErrorMessage         ErrorMessage        `json:"errorMessage"`
}

type ErrorMessage struct {
	Error string `json:"Error"`
	Cause string `json:"Cause"`
}

// Currently not using because do not know how to handle runtime error lmbda
type LegacyLambdaOutput struct {
	Status      string `json:"status"`
	MessageCode int    `json:"messageCode"`
	Message     string `json:"message"`
}

var commonHandler common_handler.CommonHandler

const DBSecretARN = "DBSecretARN"
const envLegacyUpdatefunction = "envLegacyUpdatefunction"
const envCallbackLambdaFunction = "envCallbackLambdaFunction"
const success = "success"
const running = "running"
const failure = "failure"
const loglevel = "info"
const RetriableError = "RetriableError"
const invalidHTTPStatusCodeError = "invalid http status code received"
const ContextDeadlineExceeded = "context deadline exceeded"
const base64 = "base64"
const Timeout = "States.Timeout"

func handleAuth(ctx context.Context, payoadAuthData AuthData, headers map[string]string) error {
	log.Info(ctx, "handleAuth reached...")
	authType := strings.ToLower(strings.TrimSpace(payoadAuthData.Type.String()))
	log.Info(ctx, "Auth type: ", authType)
	switch authType {
	case "", enums.AuthNone:
		return nil
	case enums.AuthBasic:
		cllientId, clientSecret, err := fetchClientIdSecret(ctx, payoadAuthData)
		if err != nil {
			return err
		}
		tempString := cllientId + ":" + clientSecret
		basicTokenEnc := b64.StdEncoding.EncodeToString([]byte(tempString))
		headers["Authorization"] = "Basic " + basicTokenEnc
		return nil
	case enums.AuthXApiKey:
		secretStoreType := strings.ToLower(payoadAuthData.RequiredAuthData.SecretStoreType)
		var XAPIKey string
		switch secretStoreType {
		case "secret_manager_key_value":
			secretManagerArn := payoadAuthData.RequiredAuthData.SecretManagerArn
			XAPIKeyKey := payoadAuthData.RequiredAuthData.XAPIKeyKey

			secretString, err := commonHandler.AwsClient.GetSecretString(ctx, secretManagerArn)
			if err != nil {
				return err
			}
			secretStringMap := make(map[string]json.RawMessage)
			json.Unmarshal([]byte(secretString), &secretStringMap)
			XAPIKey = strings.Trim(string(secretStringMap[XAPIKeyKey]), "\"")

		case "secret_manager_key":
			XAPIKeyKey := payoadAuthData.RequiredAuthData.XAPIKeyKey
			var err1 error
			XAPIKey, err1 = commonHandler.AwsClient.GetSecretString(ctx, XAPIKeyKey)

			if err1 != nil {
				return err1
			}

			XAPIKey = strings.Trim(string(XAPIKey), "\"")
		}
		headers["Authorization"] = "X-API-Key " + XAPIKey
		return nil
	case enums.AuthBearer:

		cllientId, clientSecret, err := fetchClientIdSecret(ctx, payoadAuthData)
		if err != nil {
			return err
		}

		authToken, err := fetchAuthToken(ctx, payoadAuthData.RequiredAuthData.URL, cllientId, clientSecret,
			payoadAuthData.RequiredAuthData.Headers)
		if err != nil {
			return err
		}
		headers["Authorization"] = "Bearer " + authToken

	case enums.AuthBearerSecret:
		secretStoreType := strings.ToLower(payoadAuthData.RequiredAuthData.SecretStoreType)
		var authToken string
		switch secretStoreType {
		case "secret_manager_key_value":
			secretManagerArn := payoadAuthData.RequiredAuthData.SecretManagerArn
			bearerTokenKey := payoadAuthData.RequiredAuthData.BearerTokenKey
			secretString, err := commonHandler.AwsClient.GetSecretString(ctx, secretManagerArn)
			if err != nil {
				return err
			}
			secretStringMap := make(map[string]json.RawMessage)
			json.Unmarshal([]byte(secretString), &secretStringMap)
			authToken = string(secretStringMap[bearerTokenKey])
		case "secret_manager_key":
			bearerTokenKey := payoadAuthData.RequiredAuthData.BearerTokenKey
			var err1 error
			authToken, err1 = commonHandler.AwsClient.GetSecretString(ctx, bearerTokenKey)
			if err1 != nil {
				return err1
			}

		case "pdo_secret_manager":
			bearerTokenKey := payoadAuthData.RequiredAuthData.BearerTokenKey
			authToken = commonHandler.Secrets[bearerTokenKey].(string)
		}
		authToken = ""
		headers["Authorization"] = "Bearer " + authToken
	}
	log.Info(ctx, "handleAuth successful...")
	return nil
}

func FetchS3BucketPath(s3Path string) (string, string, error) {
	if !(strings.HasPrefix(s3Path, "s3://") || strings.HasPrefix(s3Path, "S3://")) {
		s3Path = "s3://" + s3Path
	}
	u, err := url.Parse(s3Path)
	if err != nil {
		return "", "", err
	}
	return u.Host, u.Path, nil
}

func generateBasicToken(cllientId, clientSecret string) string {
	tempString := cllientId + ":" + clientSecret
	basicTokenEnc := b64.StdEncoding.EncodeToString([]byte(tempString))
	return basicTokenEnc
}

func makeGetCall(ctx context.Context, URL string, headers map[string]string, payload []byte, queryParam map[string]string) ([]byte, string, error) {
	log.Info(ctx, "makeGetCall reached...")
	u, err := url.Parse(URL)
	if err != nil {
		log.Error(ctx, err)
		return nil, "", error_handler.NewServiceError(error_codes.ErrorParsingURLCalloutLambda, err.Error())
	}
	q := u.Query()
	for key, element := range queryParam {
		q.Set(key, element)
	}
	u.RawQuery = q.Encode()
	URL = u.String()
	log.Info(ctx, "Endpoint: ", URL)
	var resp *http.Response
	if payload != nil {
		resp, err = commonHandler.HttpClient.Getwithbody(ctx, URL, bytes.NewReader(payload), headers)
	} else {
		resp, err = commonHandler.HttpClient.Get(ctx, URL, headers)
	}
	if err != nil {
		log.Error(ctx, "Error while making http call: ", err.Error())
		if strings.Contains(err.Error(), ContextDeadlineExceeded) {
			return nil, "", error_handler.NewRetriableError(error_codes.ErrorMakingGetCall, err.Error())
		}
		return nil, "", error_handler.NewServiceError(error_codes.ErrorMakingGetCall, err.Error())
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "Unable to read response body: ", err)
	}

	if resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusServiceUnavailable {
		return responseBody, resp.Status, error_handler.NewRetriableError(error_codes.ReceivedInternalServerErrorInCallout, fmt.Sprintf("%d status code received", resp.StatusCode))
	}
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "20") {
		log.Error(ctx, "invalid http status code received, statusCode: ", resp.StatusCode)
		return responseBody, resp.Status, error_handler.NewServiceError(error_codes.ReceivedInvalidHTTPStatusCodeInCallout, "received invalid http status code: "+strconv.Itoa(resp.StatusCode))
	}

	log.Info(ctx, "makeGetCall finished...")
	return responseBody, resp.Status, nil
}

func fetchAuthToken(ctx context.Context, URL, cllientId, clientSecret string, headers map[string]string) (string, error) {
	payload := strings.NewReader("grant_type=client_credentials")
	basicTokenEnc := generateBasicToken(cllientId, clientSecret)
	if headers == nil {
		headers = make(map[string]string)
	}
	if len(headers) == 0 {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
		headers["Accept"] = "application/json"
	}

	headers["Authorization"] = "Basic " + basicTokenEnc

	resp, err := commonHandler.HttpClient.Post(ctx, URL, payload, headers)
	if err != nil {
		log.Error(ctx, err)
		return "", error_handler.NewServiceError(error_codes.ErrorWhileFetchingAuthToken, err.Error())
	}
	var respJson map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&respJson)
	if err != nil {
		log.Error(ctx, err)
		return "", error_handler.NewServiceError(error_codes.ErrorUnableToDecodeAuthServiceResponse, err.Error())
	}

	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "20") {
		log.Error(ctx, errors.New(invalidHTTPStatusCodeError+strconv.Itoa(resp.StatusCode)))
		return "", error_handler.NewServiceError(error_codes.ErrorUnSuccessfullResponseFromAuthService, invalidHTTPStatusCodeError)
	}

	return fmt.Sprint(respJson["access_token"]), nil
}

func makePutPostDeleteCall(ctx context.Context, httpMethod, URL string, headers map[string]string, payload []byte) ([]byte, string, error) {
	log.Info(ctx, "makePutPostDeleteCall reached...")
	var resp *http.Response
	var err error
	log.Info(ctx, "Http Method: ", httpMethod)
	switch httpMethod {
	case enums.POST:
		resp, err = commonHandler.HttpClient.Post(ctx, URL, bytes.NewReader(payload), headers)
	case enums.PUT:
		resp, err = commonHandler.HttpClient.Put(ctx, URL, bytes.NewReader(payload), headers)
	case enums.DELETE:
		resp, err = commonHandler.HttpClient.Delete(ctx, URL, headers)
	}

	if err != nil {
		log.Error(ctx, "Error while making http request: ", err.Error())
		if strings.Contains(err.Error(), ContextDeadlineExceeded) {
			return nil, "", error_handler.NewRetriableError(error_codes.ErrorMakingPostPutOrDeleteCall, err.Error())
		}
		return nil, "", error_handler.NewServiceError(error_codes.ErrorMakingPostPutOrDeleteCall, err.Error())
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "Error while reading response body: ", err.Error())
	}
	if resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusServiceUnavailable {
		return responseBody, resp.Status, error_handler.NewRetriableError(error_codes.ReceivedInternalServerErrorInCallout, fmt.Sprintf("%d status code received", resp.StatusCode))
	}
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "20") {
		log.Error(ctx, "invalid http status code received, statusCode: ", resp.StatusCode)
		return responseBody, resp.Status, error_handler.NewServiceError(error_codes.ReceivedInvalidHTTPStatusCodeInCallout, "received invalid http status code: "+strconv.Itoa(resp.StatusCode))
	}

	log.Info(ctx, "makePutPostDeleteCall finished...")
	return responseBody, resp.Status, nil
}

func fetchClientIdSecret(ctx context.Context, payoadAuthData AuthData) (string, string, error) {
	secretStoreType := strings.ToLower(payoadAuthData.RequiredAuthData.SecretStoreType)
	var cllientId, clientSecret string
	switch secretStoreType {
	case "secret_manager_key_value":
		secretManagerArn := payoadAuthData.RequiredAuthData.SecretManagerArn
		cllientIdKey := payoadAuthData.RequiredAuthData.ClientIDKey
		clientSecretKey := payoadAuthData.RequiredAuthData.ClientSecretKey
		secretString, err := commonHandler.AwsClient.GetSecretString(ctx, secretManagerArn)
		if err != nil {
			return "", "", error_handler.NewServiceError(error_codes.ErrorFetchingSecretsFromSecretManager, err.Error())
		}
		secretStringMap := make(map[string]json.RawMessage)
		json.Unmarshal([]byte(secretString), &secretStringMap)
		cllientId = string(secretStringMap[cllientIdKey])
		clientSecret = string(secretStringMap[clientSecretKey])

	case "secret_manager_key":
		cllientIdKey := payoadAuthData.RequiredAuthData.ClientIDKey
		clientSecretKey := payoadAuthData.RequiredAuthData.ClientSecretKey
		var err1, err2 error
		cllientId, err1 = commonHandler.AwsClient.GetSecretString(ctx, cllientIdKey)
		clientSecret, err2 = commonHandler.AwsClient.GetSecretString(ctx, clientSecretKey)

		if err1 != nil {
			return "", "", error_handler.NewServiceError(error_codes.ErrorFetchingSecretsFromSecretManager, err1.Error())
		}
		if err2 != nil {
			return "", "", error_handler.NewServiceError(error_codes.ErrorFetchingSecretsFromSecretManager, err2.Error())
		}
	case "pdo_secret_manager":
		cllientIdKey := payoadAuthData.RequiredAuthData.ClientIDKey
		clientSecretKey := payoadAuthData.RequiredAuthData.ClientSecretKey
		cllientId = commonHandler.Secrets[cllientIdKey].(string)
		clientSecret = commonHandler.Secrets[clientSecretKey].(string)
	}
	cllientId = strings.Trim(string(cllientId), "\"")
	clientSecret = strings.Trim(string(clientSecret), "\"")
	return cllientId, clientSecret, nil
}

func storeDataToS3(ctx context.Context, s3Path string, responseBody []byte) error {

	bucketName, s3KeyPath, err := FetchS3BucketPath(s3Path)
	if err != nil {
		log.Error(ctx, "Error while parsing s3 path, error: ", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorFetchingS3BucketPath, err.Error())
	}
	err = commonHandler.AwsClient.StoreDataToS3(ctx, bucketName, s3KeyPath, responseBody)
	if err != nil {
		return error_handler.NewServiceError(error_codes.ErrorStoringDataToS3, err.Error())
	}
	return nil
}

func callLegacyStatusUpdate(ctx context.Context, payload map[string]interface{}) error {
	log.Infof(ctx, "callLegacyStatusUpdate reached...")
	legacyLambdaFunction := os.Getenv(envLegacyUpdatefunction)

	result, err := commonHandler.AwsClient.InvokeLambda(ctx, legacyLambdaFunction, payload, false)
	if err != nil {
		return error_handler.NewServiceError(error_codes.ErrorInvokingLambdaLegacyUpdateLambda, err.Error())
	}
	var resp map[string]interface{}
	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		log.Error(ctx, "Error while unmarshalling, errror: ", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorDecodingLambdaOutput, err.Error())
	}

	errorType, ok := resp["errorType"]
	log.Errorf(ctx, "Error returned from lambda: %+v", errorType)
	if ok {
		if errorType == RetriableError {
			return error_handler.NewRetriableError(error_codes.ErrorWhileUpdatingLegacy, fmt.Sprintf("received %s errorType while updating legacy", errorType))
		}
		return error_handler.NewServiceError(error_codes.ErrorWhileUpdatingLegacy, "error while executing update legacy lamdba")
	}

	legacyStatus, ok := resp["status"]
	if !ok {
		log.Errorf(ctx, "legacy Response should have status")
		return error_handler.NewServiceError(error_codes.StatusNotFoundInLegacyUpdateResponse, "legacy update lambda response doesnt have status")
	}
	legacyStatusString := strings.ToLower(fmt.Sprintf("%v", legacyStatus))

	if legacyStatusString == "failure" {
		log.Errorf(ctx, "legacy returned with status as failure")
		return error_handler.NewServiceError(error_codes.LegacyStatusFailed, "legacy returned with status as failure")
	}

	log.Info(ctx, "callLegacyStatusUpdate successful...")
	return nil
}

func callLambda(ctx context.Context, payload interface{}, LambdaFunction string, isWaitTask bool) (map[string]interface{}, error) {
	log.Infof(ctx, "callLambda reached...")
	substrings := strings.Split(LambdaFunction, ":")
	functionName := substrings[len(substrings)-1]
	result, err := commonHandler.AwsClient.InvokeLambda(ctx, LambdaFunction, payload.(map[string]interface{}), isWaitTask)
	if err != nil {
		return nil, error_handler.NewServiceError(error_codes.ErrorInvokingLambda, fmt.Sprintf("error invoking %s lambda : %s", functionName, err.Error()))
	}
	var resp map[string]interface{}
	if len(result.Payload) != 0 {
		err = json.Unmarshal(result.Payload, &resp)
		if err != nil {
			log.Error(ctx, "Error while unmarshalling, errror: ", err.Error())
			return resp, error_handler.NewServiceError(error_codes.ErrorDecodingLambdaOutput, fmt.Sprintf("error unmarshalling %s output : %s", functionName, err.Error()))
		}
	}
	errorType, ok := resp["errorType"]
	log.Errorf(ctx, "Error returned from lambda: %+v", errorType)
	if ok {
		var errorMessage string
		_, ok = resp["errorMessage"]
		if ok {
			errorMessage = resp["errorMessage"].(string)
		} else {
			errorMessage = fmt.Sprintf("received %s", errorType)
		}
		if errorType == RetriableError {
			return resp, error_handler.NewRetriableError(error_codes.ErrorInvokingLambda, errorMessage, fmt.Sprintf("from %s", functionName))
		}
		return resp, error_handler.NewServiceError(error_codes.ErrorInvokingLambda, errorMessage, fmt.Sprintf("from %s", functionName))
	}
	log.Info(ctx, "callLambda successful...")
	return resp, nil
}

func handleHipster(ctx context.Context, reportId, status, jobID string) error {
	legacyRequestPayload := map[string]interface{}{
		"status":       status,
		"hipsterJobId": jobID,
		"reportId":     reportId,
	}

	return callLegacyStatusUpdate(ctx, legacyRequestPayload)
}

func validate(ctx context.Context, data MyEvent) error {
	if err := validator.ValidateCallOutRequest(ctx, data); err != nil {
		return err
	}

	callType := data.CallType.String()

	if callType == "" && (data.RequestMethod == "" || data.URL == "") {
		return errors.New("invalid callout request")
	}
	if (callType == enums.HipsterCT || callType == enums.LegacyCT) && (data.Status == "") {
		return errors.New("status cannot be empty")
	}
	if (callType == enums.LambdaCT) && (data.ARN == "") {
		return errors.New("Lambda ARN cannot be empty")
	}
	return nil
}

func CallService(ctx context.Context, data MyEvent, stepID string) (map[string]interface{}, error) {
	log.Info(ctx, "CallService reached...")
	returnResponse := make(map[string]interface{})

	if err := validate(ctx, data); err != nil {
		log.Error(ctx, "Validation failed, error: ", err.Error())
		return returnResponse, error_handler.NewServiceError(error_codes.ErrorValidatingCallOutLambdaRequest, err.Error())
	}

	timeout := 45
	if data.Timeout != 0 {
		timeout = data.Timeout
	}

	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		APITimeout: timeout,
	})

	callType := data.CallType.String()
	log.Info(ctx, "CallType: ", callType)

	if callType == enums.LegacyCT {
		var notes string
		if data.ErrorMessage.Error == Timeout {
			timedoutTask := commonHandler.DBClient.GetTimedoutTask(ctx, data.WorkflowID)
			if timedoutTask != "" {
				notes = fmt.Sprintf("Task Timedout at %s", timedoutTask)
			} else {
				notes = Timeout
			}
		} else {
			if data.ErrorMessage.Error != "" || data.ErrorMessage.Cause != "" {
				notes = fmt.Sprintf("Error: %s :: Cause: %s", data.ErrorMessage.Error, data.ErrorMessage.Cause)
			}
		}
		req := map[string]interface{}{
			"reportId":   data.ReportID,
			"workflowId": data.WorkflowID,
			"status":     data.Status,
			"taskName":   data.TaskName,
			"notes":      notes,
		}
		err := callLegacyStatusUpdate(ctx, req)
		if err != nil {
			returnResponse["status"] = failure
			return returnResponse, err
		}
		returnResponse["status"] = "success"
		log.Info(ctx, "CallService successfull...")
		return returnResponse, err
	}

	if data.IsWaitTask {
		metaObj := Meta{
			CallbackID:  stepID,
			CallbackURL: os.Getenv(envCallbackLambdaFunction),
		}

		if body, ok := data.Payload.(map[string]interface{}); ok {
			if val, ok := body["meta"]; ok {
				if _, ok := val.(map[string]interface{})["callbackUrl"]; ok {
					val.(map[string]interface{})["callbackId"] = metaObj.CallbackID
				} else {
					val.(map[string]interface{})["callbackUrl"] = metaObj.CallbackURL
					val.(map[string]interface{})["callbackId"] = metaObj.CallbackID
				}
			} else {
				body["meta"] = metaObj
			}
			data.Payload = body
		}
	}

	if callType == enums.LambdaCT {
		req := data.Payload
		responseBody, err := callLambda(ctx, req, data.ARN, data.IsWaitTask)
		if err != nil {
			returnResponse["status"] = failure
			return returnResponse, err
		}
		if responseBody == nil {
			responseBody = make(map[string]interface{})
		}
		responseBody["status"] = success
		log.Info(ctx, "CallService successfull...")
		return responseBody, err
	}
	if callType == enums.SQSCT {
		sqsurl := data.QueueUrl
		bytearray, err := json.Marshal(data.Payload)
		if err != nil {
			log.Error(ctx, "Error while marshalling callout payload, error: ", err.Error())
			returnResponse["status"] = failure
			return returnResponse, error_handler.NewServiceError(error_codes.ErrorSerializingCallOutPayload, err.Error())
		}
		err = commonHandler.AwsClient.PushMessageToSQS(ctx, sqsurl, string(bytearray))
		if err != nil {
			returnResponse["status"] = failure
			return returnResponse, error_handler.NewServiceError(error_codes.ErrorPushingDataToSQS, err.Error())
		}
		returnResponse["status"] = success
		log.Info(ctx, "CallService successfull...")
		return returnResponse, err
	}
	json_data, err := json.Marshal(data.Payload)
	if err != nil {
		log.Error(ctx, "Error while marshalling callout payload, error: ", err.Error())
		returnResponse["status"] = failure
		return returnResponse, error_handler.NewServiceError(error_codes.ErrorSerializingCallOutPayload, err.Error())
	}
	if data.GetRequestBodyFromS3 != "" {
		host, path, err := commonHandler.AwsClient.FetchS3BucketPath(data.GetRequestBodyFromS3)
		if err != nil {
			log.Error(ctx, "Error in fetching AWS path: ", err.Error())
			return returnResponse, error_handler.NewServiceError(error_codes.ErrorFetchingS3BucketPath, err.Error())
		}
		json_data, err = commonHandler.AwsClient.GetDataFromS3(ctx, host, path)
		if err != nil {
			log.Error(ctx, "Error in getting downloading from s3: ", err.Error())
			return returnResponse, error_handler.NewServiceError(error_codes.ErrorFetchingDataFromS3, err.Error())
		}
		if data.S3RequestBodyType == base64 {
			json_data, err = json.Marshal(b64.StdEncoding.EncodeToString(json_data))
			if err != nil {
				log.Error(ctx, "Error in marshalling : ", err.Error())
				return returnResponse, error_handler.NewServiceError(error_codes.ErrorSerializingS3Data, err.Error())
			}
		}

	}
	headers := make(map[string]string)
	if data.Headers != nil {
		headers = data.Headers
	}

	handleAuth(ctx, data.Auth, headers)

	var responseStatus string
	var responseBody []byte
	var responseError error
	requestMethod := strings.ToUpper(data.RequestMethod.String())
	switch requestMethod {
	case enums.GET:
		responseBody, responseStatus, responseError = makeGetCall(ctx, data.URL, headers, json_data, data.QueryParam)
		log.Info(ctx, "http response:", string(responseBody))
		if responseError != nil {
			returnResponse["status"] = failure
			return returnResponse, responseError
		}

	case enums.POST, enums.PUT, enums.DELETE:
		responseBody, responseStatus, responseError = makePutPostDeleteCall(ctx, requestMethod, data.URL, headers, json_data)
		log.Info(ctx, "http response: ", string(responseBody))

		if responseError != nil {
			returnResponse["status"] = failure
			return returnResponse, responseError
		}

	default:
		log.Error(ctx, "Unknown request method, can not proceed, RequestMethod: ", requestMethod)
		returnResponse["status"] = failure
		return returnResponse, error_handler.NewServiceError(error_codes.UnsupportedRequestMethodCallOutLambda, "unknown request method, can not proceed, requestMethod: "+requestMethod)

	}

	if !strings.HasPrefix(responseStatus, "20") {
		returnResponse["status"] = failure
		log.Error(ctx, "Failure status code Received ", responseStatus)
		return returnResponse, error_handler.NewServiceError(error_codes.ReceivedInvalidHTTPStatusCodeInCallout, "received failure status code")
	}

	if len(responseBody) != 0 {
		err = json.Unmarshal(responseBody, &returnResponse)
		if err != nil {
			log.Error(ctx, "Unable to unmarshall response: ", err.Error())
			returnResponse["status"] = failure
			return returnResponse, error_handler.NewServiceError(error_codes.ErrorDecodingLambdaOutput, err.Error())
		}
	}

	if data.StoreDataToS3 != "" {
		returnResponse = make(map[string]interface{})
		err := storeDataToS3(ctx, data.StoreDataToS3, responseBody)
		if err != nil {
			returnResponse["status"] = failure
			return returnResponse, err
		}
		returnResponse["s3DataLocation"] = data.StoreDataToS3
	}

	if callType == enums.HipsterCT {
		jobID := data.HipsterJobID
		if jobID == "" {
			hipsterOutput := make(map[string]string)
			ok := false
			err := json.Unmarshal(responseBody, &hipsterOutput)
			if err != nil {
				returnResponse["status"] = failure
				log.Error(ctx, "Error while unmarshalling response, error: ", err.Error())
				return returnResponse, error_handler.NewServiceError(error_codes.ErrorDecodingHipsterOutput, err.Error())
			}
			if jobID, ok = hipsterOutput["jobId"]; !ok {
				returnResponse["status"] = failure
				log.Error(ctx, "Hipster JobId missing in hipster output")
				return returnResponse, error_handler.NewServiceError(error_codes.JobIDMissingInHipsterOutput, "hipster jobId missing in hipster output")
			}
		}
		log.Info(ctx, "hipster jobId: ", jobID)
		err := handleHipster(ctx, data.ReportID, data.Status, jobID)
		if err != nil {
			returnResponse["status"] = failure
			return returnResponse, err
		}
	}

	log.Info(ctx, returnResponse, responseError)

	return returnResponse, responseError
}

func HandleRequest(ctx context.Context, data MyEvent) (map[string]interface{}, error) {
	starttime := time.Now().Unix()
	stepID := uuid.New().String()
	ctx = log_config.SetTraceIdInContext(ctx, data.ReportID, data.WorkflowID)

	log.Info(ctx, "callout lambda reached...")

	response, serviceerr := CallService(ctx, data, stepID)
	StepExecutionData := documentDB_client.StepExecutionDataBody{
		StepId:     stepID,
		StartTime:  starttime,
		Url:        data.URL,
		Input:      data.Payload,
		TaskToken:  data.TaskToken,
		WorkflowId: data.WorkflowID,
		TaskName:   data.TaskName,
		ReportId:   data.ReportID,
	}
	if serviceerr != nil {
		StepExecutionData.Status = failure
		StepExecutionData.Output = response
	}
	if data.IsWaitTask {
		StepExecutionData.IntermediateOutput = response
		StepExecutionData.Status = running
	} else {
		StepExecutionData.Output = response
		StepExecutionData.EndTime = time.Now().Unix()
	}
	err := commonHandler.DBClient.InsertStepExecutionData(ctx, StepExecutionData)
	if err != nil {
		log.Error(ctx, "Unable to insert Step Data in DocumentDB")
		return response, error_handler.NewServiceError(error_codes.ErrorInsertingStepExecutionDataInDB, err.Error())
	}
	filter := bson.M{"_id": data.WorkflowID}
	if serviceerr != nil {
		update := commonHandler.DBClient.BuildQueryForUpdateWorkflowDataCallout(ctx, data.TaskName, stepID, failure, starttime, data.IsWaitTask)
		commonHandler.DBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.WorkflowDataCollection)
		return response, serviceerr
		// Have to handle this
	} else {
		update := commonHandler.DBClient.BuildQueryForUpdateWorkflowDataCallout(ctx, data.TaskName, stepID, success, starttime, data.IsWaitTask)
		err := commonHandler.DBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.WorkflowDataCollection)
		if err != nil {
			response["status"] = failure
			return response, error_handler.NewServiceError(error_codes.ErrorUpdatingWorkflowDataInDB, err.Error())
		}
		return response, nil
	}
}

func notifcationWrapper(ctx context.Context, req MyEvent) (map[string]interface{}, error) {
	resp, err := HandleRequest(ctx, req)
	if err != nil {
		errT := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(errT.GetErrorCode(), req.ReportID, req.WorkflowID, "callout", req.TaskName, err.Error(), nil)
	}
	return resp, err
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(true, true, true, true, true)
	lambda.Start(notifcationWrapper)
}
