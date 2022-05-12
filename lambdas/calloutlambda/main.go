package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.eagleview.com/engineering/assess-platform-library/log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/enums"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/validator"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/google/uuid"
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
		XAPIKeyKey       string            `json:"X-API-Key_Key,omitempty"`
	} `json:"authData,omitempty"`
}

type MyEvent struct {
	Payload       interface{}         `json:"requestData"`
	URL           string              `json:"url" validate:"omitempty,url"`
	RequestMethod enums.RequestMethod `json:"requestMethod" validate:"omitempty,httpMethod"`
	Headers       map[string]string   `json:"headers"`
	IsWaitTask    bool                `json:"isWaitTask"`
	Timeout       int                 `json:"timeout"`
	StoreDataToS3 string              `json:"storeDataToS3"`
	TaskName      string              `json:"taskName"`
	CallType      enums.CallType      `json:"callType" validate:"omitempty,callTypes"`
	OrderID       string              `json:"orderId"`
	ReportID      string              `json:"reportId" validate:"required"`
	WorkflowID    string              `json:"workflowId" validate:"required"`
	TaskToken     string              `json:"taskToken" validate:"required_if=IsWaitTask true"`
	HipsterJobID  string              `json:"hipsterJobId,omitempty"`
	QueryParam    map[string]string   `json:"queryParam,omitempty"`
	Auth          AuthData            `json:"auth"`
	Status        string              `json:"status"`
}

// Currently not using because do not know how to handle runtime error lmbda
type LegacyLambdaOutput struct {
	Status      string `json:"status"`
	MessageCode int    `json:"messageCode"`
	Message     string `json:"message"`
}

var AwsClient aws_client.IAWSClient
var httpClient httpservice.IHTTPClientV2
var newDBClient *documentDB_client.DocDBClient

const DBSecretARN = "DBSecretARN"
const envLegacyUpdatefunction = "envLegacyUpdatefunction"
const envCallbackLambdaFunction = "envCallbackLambdaFunction"
const success = "success"
const failure = "failure"
const loglevel = "info"

func handleAuth(ctx context.Context, payoadAuthData AuthData, headers map[string]string) error {
	authType := strings.ToLower(strings.TrimSpace(payoadAuthData.Type.String()))
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

			secretString, err := AwsClient.GetSecretString(ctx, secretManagerArn)
			if err != nil {
				return err
			}
			secretStringMap := make(map[string]json.RawMessage)
			json.Unmarshal([]byte(secretString), &secretStringMap)
			XAPIKey = strings.Trim(string(secretStringMap[XAPIKeyKey]), "\"")

		case "secret_manager_key":
			XAPIKeyKey := payoadAuthData.RequiredAuthData.XAPIKeyKey
			var err1 error
			XAPIKey, err1 = AwsClient.GetSecretString(ctx, XAPIKeyKey)

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
			log.Error(ctx, "unable to fetch cllientId, clientSecret")
			return err
		}

		authToken, err := fetchAuthToken(ctx, payoadAuthData.RequiredAuthData.URL, cllientId, clientSecret,
			payoadAuthData.RequiredAuthData.Headers)
		if err != nil {
			return err
		}
		headers["Authorization"] = "Bearer " + authToken
	}
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
	u, err := url.Parse(URL)
	if err != nil {
		log.Error(ctx, err)
		return nil, "", err
	}
	q := u.Query()
	for key, element := range queryParam {
		q.Set(key, element)
	}
	u.RawQuery = q.Encode()
	URL = u.String()
	var resp *http.Response
	if payload != nil {
		resp, err = httpClient.Getwithbody(ctx, URL, bytes.NewReader(payload), headers)
	} else {
		resp, err = httpClient.Get(ctx, URL, headers)
	}
	if err != nil {
		return nil, "", err
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, err)
	}

	if resp.StatusCode != 200 {
		return responseBody, resp.Status, errors.New("invalid http status code received")
	}

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

	resp, err := httpClient.Post(ctx, URL, payload, headers)
	if err != nil {
		log.Error(ctx, err)
		return "", err
	}
	var respJson map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&respJson)
	if err != nil {
		log.Error(ctx, err)
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", errors.New("invalid http status code received")
	}

	return fmt.Sprint(respJson["access_token"]), nil
}

func makePutPostDeleteCall(ctx context.Context, httpMethod, URL string, headers map[string]string, payload []byte) ([]byte, string, error) {

	var resp *http.Response
	var err error
	switch httpMethod {
	case enums.POST:
		resp, err = httpClient.Post(ctx, URL, bytes.NewReader(payload), headers)
	case enums.PUT:
		resp, err = httpClient.Put(ctx, URL, bytes.NewReader(payload), headers)
	case enums.DELETE:
		resp, err = httpClient.Delete(ctx, URL, headers)
	}

	if err != nil {
		log.Error(ctx, err)
		return nil, "", err
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Error(ctx, err)
	}
	if resp.StatusCode != 200 {
		return responseBody, resp.Status, errors.New("invalid http status code received")
	}
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
		secretString, err := AwsClient.GetSecretString(ctx, secretManagerArn)
		if err != nil {
			return "", "", err
		}
		secretStringMap := make(map[string]json.RawMessage)
		json.Unmarshal([]byte(secretString), &secretStringMap)
		cllientId = string(secretStringMap[cllientIdKey])
		clientSecret = string(secretStringMap[clientSecretKey])

	case "secret_manager_key":
		cllientIdKey := payoadAuthData.RequiredAuthData.ClientIDKey
		clientSecretKey := payoadAuthData.RequiredAuthData.ClientSecretKey
		var err1, err2 error
		cllientId, err1 = AwsClient.GetSecretString(ctx, cllientIdKey)
		clientSecret, err2 = AwsClient.GetSecretString(ctx, clientSecretKey)

		if err1 != nil {
			return "", "", err1
		}
		if err2 != nil {
			return "", "", err2
		}

	}
	cllientId = strings.Trim(string(cllientId), "\"")
	clientSecret = strings.Trim(string(clientSecret), "\"")
	return cllientId, clientSecret, nil
}
func storeDataToS3(ctx context.Context, s3Path string, responseBody []byte) error {

	bucketName, s3KeyPath, err := FetchS3BucketPath(s3Path)
	if err != nil {
		return err
	}
	err = AwsClient.StoreDataToS3(ctx, bucketName, s3KeyPath, responseBody)

	if err != nil {
		return err
	}
	return nil
}

func callLegacyStatusUpdate(ctx context.Context, payload map[string]interface{}) error {
	legacyLambdaFunction := os.Getenv(envLegacyUpdatefunction)

	result, err := AwsClient.InvokeLambda(ctx, legacyLambdaFunction, payload)

	if err != nil {
		return err
	}
	var resp map[string]interface{}
	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		return err
	}

	// Do not know how to handle error result.FunctionError

	errorType, ok := resp["errorType"]
	if ok {
		log.Info(ctx, errorType)
		return errors.New("error occured while executing lambda ")
	}

	legacyStatus, ok := resp["status"]
	if !ok {
		return errors.New("legacy Response should have status")
	}
	legacyStatusString := strings.ToLower(fmt.Sprintf("%v", legacyStatus))

	if legacyStatusString == "failure" {
		return errors.New("legacy returned with status as failure")
	}

	return nil
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
	return nil
}

func CallService(ctx context.Context, data MyEvent, stepID string) (map[string]interface{}, error) {

	returnResponse := make(map[string]interface{})

	if err := validate(ctx, data); err != nil {
		return returnResponse, err
	}

	timeout := 30
	if data.Timeout != 0 {
		timeout = data.Timeout
	}

	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		APITimeout: timeout,
	})

	callType := data.CallType.String()

	if callType == enums.LegacyCT {
		req := map[string]interface{}{
			"reportId":   data.ReportID,
			"workflowId": data.WorkflowID,
			"status":     data.Status,
			"taskName":   data.TaskName,
		}
		err := callLegacyStatusUpdate(ctx, req)
		if err != nil {
			log.Error(ctx, err)
			returnResponse["status"] = failure
			return returnResponse, err
		}
		returnResponse["status"] = "success"
		return returnResponse, err
	}

	if data.IsWaitTask {

		metaObj := Meta{
			CallbackID:  stepID,
			CallbackURL: os.Getenv(envCallbackLambdaFunction),
		}

		if body, ok := data.Payload.(map[string]interface{}); ok {
			body["meta"] = metaObj
			data.Payload = body
		}
	}

	json_data, err := json.Marshal(data.Payload)
	if err != nil {
		returnResponse["status"] = failure
		return returnResponse, err
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
		log.Info(ctx, string(responseBody))
		if responseError != nil {
			returnResponse["status"] = failure
			return returnResponse, responseError
		}

	case enums.POST, enums.PUT, enums.DELETE:
		responseBody, responseStatus, responseError = makePutPostDeleteCall(ctx, requestMethod, data.URL, headers, json_data)
		log.Info(ctx, string(responseBody))

		if responseError != nil {
			returnResponse["status"] = failure
			return returnResponse, responseError
		}

	default:
		log.Info(ctx, "Unknown request method, can not proceed")
		returnResponse["status"] = failure
		return returnResponse, responseError

	}
	if !strings.HasPrefix(responseStatus, "20") {
		returnResponse["status"] = failure
		return returnResponse, errors.New("Failure status code Received " + responseStatus)
	}
	if len(responseBody) != 0 {
		err = json.Unmarshal(responseBody, &returnResponse)
		if err != nil {
			returnResponse["status"] = failure
			return returnResponse, err
		}
	}

	if data.StoreDataToS3 != "" {
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
				return returnResponse, err
			}
			if jobID, ok = hipsterOutput["jobId"]; !ok {
				returnResponse["status"] = failure
				return returnResponse, errors.New("Hipster JobId missing in hipster output")
			}
		}
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
	response, serviceerr := CallService(ctx, data, stepID)
	StepExecutionData := documentDB_client.StepExecutionDataBody{
		StepId:     stepID,
		StartTime:  starttime,
		Url:        data.URL,
		Input:      data.Payload,
		TaskToken:  data.TaskToken,
		WorkflowId: data.WorkflowID,
		TaskName:   data.TaskName,
	}
	if serviceerr != nil {
		StepExecutionData.Status = failure
		StepExecutionData.Output = response
	}
	if data.IsWaitTask {
		StepExecutionData.IntermediateOutput = response
	} else {
		StepExecutionData.Output = response
		StepExecutionData.EndTime = time.Now().Unix()
	}
	err := newDBClient.InsertStepExecutionData(ctx, StepExecutionData)
	if err != nil {
		log.Error(ctx, "Unable to insert Step Data in DocumentDB")
		return response, err
	}
	filter := bson.M{"_id": data.WorkflowID}
	if serviceerr != nil {
		update := newDBClient.BuildQueryForUpdateWorkflowDataCallout(ctx, data.TaskName, stepID, failure, starttime, data.IsWaitTask)
		newDBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.WorkflowDataCollection)
		return response, serviceerr
	} else {
		update := newDBClient.BuildQueryForUpdateWorkflowDataCallout(ctx, data.TaskName, stepID, success, starttime, data.IsWaitTask)
		err := newDBClient.UpdateDocumentDB(ctx, filter, update, documentDB_client.WorkflowDataCollection)
		if err != nil {
			response["status"] = failure
		}
		return response, err
	}

}

func main() {
	log_config.InitLogging(loglevel)
	httpClient = &httpservice.HTTPClientV2{}
	AwsClient = &aws_client.AWSClient{}
	if newDBClient == nil {
		SecretARN := os.Getenv(DBSecretARN)
		log.Error(context.Background(), "fetching db secrets")
		secrets, err := AwsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
		if err != nil {
			log.Error(context.Background(), "Unable to fetch DocumentDb in secret")
		}
		newDBClient = documentDB_client.NewDBClientService(secrets)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = newDBClient.DBClient.Connect(ctx)
		if err != nil {
			log.Error(ctx, err)
		}
	}
	lambda.Start(HandleRequest)
}
