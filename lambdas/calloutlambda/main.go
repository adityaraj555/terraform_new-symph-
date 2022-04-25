package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"

	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"

	"github.com/google/uuid"
)

type Meta struct {
	CallbackID  string `json:"callbackId"`
	CallbackURL string `json:"callbackUrl"`
}

type AuthData struct {
	Type             string `json:"type"`
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
	Payload                map[string]interface{} `json:"requestData"`
	URL                    string                 `json:"url"`
	RequestMethod          string                 `json:"requestMethod"`
	Headers                map[string]string      `json:"headers"`
	IsWaitTask             bool                   `json:"isWaitTask"`
	Timeout                int                    `json:"timeout"`
	StoreDataToS3          string                 `json:"storeDataToS3"`
	TaskName               string                 `json:"taskName"`
	CallType               string                 `json:"callType"`
	OrderID                string                 `json:"orderId"`
	ReportID               string                 `json:"reportId"`
	WorkflowID             string                 `json:"workflowId"`
	TaskToken              string                 `json:"taskToken"`
	HipsterLegacySubStatus string                 `json:"hipsterLegacySubStatus,omitempty"`
	HipsterJobID           string                 `json:"hipsterJobId,omitempty"`
	QueryParam             map[string]string      `json:"queryParam,omitempty"`
	Auth                   AuthData               `json:"auth"`
}

// Currently not using because do not know how to handle runtime error lmbda
type LegacyLambdaOutput struct {
	Status      string `json:"status"`
	MessageCode int    `json:"messageCode"`
	Message     string `json:"message"`
}

var awsClient aws_client.IAWSClient
var httpClient httpservice.IHTTPClientV2

func handleAuth(ctx context.Context, payoadAuthData AuthData, headers map[string]string) error {
	authType := strings.ToLower(strings.TrimSpace(payoadAuthData.Type))
	switch authType {
	case "", "none":
		return nil
	case "basic":
		cllientId, clientSecret, err := fetchClientIdSecret(ctx, payoadAuthData)
		if err != nil {
			return err
		}
		tempString := cllientId + ":" + clientSecret
		basicTokenEnc := b64.StdEncoding.EncodeToString([]byte(tempString))
		headers["Authorization"] = "Basic " + basicTokenEnc
		return nil
	case "x-api-key":
		secretStoreType := strings.ToLower(payoadAuthData.RequiredAuthData.SecretStoreType)
		var XAPIKey string
		switch secretStoreType {
		case "secret_manager_key_value":
			secretManagerArn := payoadAuthData.RequiredAuthData.SecretManagerArn
			XAPIKeyKey := payoadAuthData.RequiredAuthData.XAPIKeyKey

			secretString, err := awsClient.GetSecretString(ctx, secretManagerArn)
			if err != nil {
				return err
			}
			secretStringMap := make(map[string]json.RawMessage)
			json.Unmarshal([]byte(secretString), &secretStringMap)
			XAPIKey = strings.Trim(string(secretStringMap[XAPIKeyKey]), "\"")

		case "secret_manager_key":
			XAPIKeyKey := payoadAuthData.RequiredAuthData.XAPIKeyKey
			var err1 error
			XAPIKey, err1 = awsClient.GetSecretString(ctx, XAPIKeyKey)

			if err1 != nil {
				return err1
			}

			XAPIKey = strings.Trim(string(XAPIKey), "\"")
		}
		headers["Authorization"] = "X-API-Key " + XAPIKey
		return nil
	case "bearer":
		cllientId, clientSecret, err := fetchClientIdSecret(ctx, payoadAuthData)
		if err != nil {
			fmt.Println("unable to fetch cllientId, clientSecret")
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
		log.Fatal(err)
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
		log.Fatal(err)
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
		log.Fatal(err)
		return "", err
	}
	var respJson map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&respJson)
	if err != nil {
		log.Fatal(err)
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
	case "POST":
		resp, err = httpClient.Post(ctx, URL, bytes.NewReader(payload), headers)
	case "PUT":
		resp, err = httpClient.Put(ctx, URL, bytes.NewReader(payload), headers)
	case "DELETE":
		resp, err = httpClient.Delete(ctx, URL, headers)
	}

	if err != nil {
		log.Fatal(err)
		return nil, "", err
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
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
		secretString, err := awsClient.GetSecretString(ctx, secretManagerArn)
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
		cllientId, err1 = awsClient.GetSecretString(ctx, cllientIdKey)
		clientSecret, err2 = awsClient.GetSecretString(ctx, clientSecretKey)

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
	err = awsClient.StoreDataToS3(ctx, bucketName, s3KeyPath, responseBody)

	if err != nil {
		return err
	}
	return nil
}

func callLegacyStatusUpdate(ctx context.Context, payload map[string]interface{}) error {
	legacyLambdaFunction := os.Getenv("envLegacyUpdatefunction")

	result, err := awsClient.InvokeLambda(ctx, legacyLambdaFunction, payload)

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
		fmt.Println(errorType)
		return errors.New("error occured while executing lambda ")
	}

	legacyStatus, ok := resp["Status"]
	if !ok {
		return errors.New("legacy Response should have status")
	}
	legacyStatusString := strings.ToLower(fmt.Sprintf("%v", legacyStatus))

	if legacyStatusString == "failure" {
		return errors.New("legacy returned with status as failure")
	}

	return nil
}

func handleHipster(ctx context.Context, reportId, hipsterLegacySubStatus, jobID string) error {

	legacyRequestPayload := map[string]interface{}{
		"ReportId":     reportId,
		"Status":       "InProcess",
		"SubStatus":    hipsterLegacySubStatus,
		"HipsterJobId": jobID,
	}

	return callLegacyStatusUpdate(ctx, legacyRequestPayload)
}

func HandleRequest(ctx context.Context, data MyEvent) (string, error) {

	timeout := 30
	if data.Timeout != 0 {
		timeout = data.Timeout
	}

	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		APITimeout: timeout,
	})

	callType := strings.ToLower(data.CallType)

	if callType == "eagleflow" {
		err := callLegacyStatusUpdate(ctx, data.Payload)
		if err != nil {
			fmt.Println(err)
			return "faiure", err
		}
		return "success", err
	}

	if data.IsWaitTask {
		callbackId := uuid.New()
		metaObj := Meta{
			CallbackID:  callbackId.String(),
			CallbackURL: os.Getenv("envCallbackLambdaFunction"),
		}
		data.Payload["meta"] = metaObj
	}

	json_data, _ := json.Marshal(data.Payload)
	fmt.Println(json_data)

	headers := make(map[string]string)
	headers = data.Headers

	handleAuth(ctx, data.Auth, headers)

	var responseStatus string
	var responseBody []byte
	var responseError error
	requestMethod := strings.ToUpper(data.RequestMethod)
	switch requestMethod {
	case "GET":
		responseBody, responseStatus, responseError = makeGetCall(ctx, data.URL, headers, json_data, data.QueryParam)
		fmt.Println(string(responseBody))
		if responseError != nil {
			return responseStatus, responseError
		}
	case "POST", "PUT", "DELETE":
		responseBody, responseStatus, responseError = makePutPostDeleteCall(ctx, requestMethod, data.URL, headers, json_data)
		fmt.Println(string(responseBody))
		if responseError != nil {
			return responseStatus, responseError
		}
	}
	if data.StoreDataToS3 != "" {
		storeDataToS3(ctx, data.StoreDataToS3, responseBody)
	}

	if callType == "hipster" {
		jobID := data.HipsterJobID
		if jobID == "" {
			hipsterOutput := make(map[string]string)
			ok := false
			err := json.Unmarshal(responseBody, &hipsterOutput)
			if err != nil {
				return "", err
			}
			if jobID, ok = hipsterOutput["jobId"]; !ok {
				return "", errors.New("Hipster JobId missing in hipster output")
			}
		}
		err := handleHipster(ctx, data.ReportID, data.HipsterLegacySubStatus, jobID)
		if err != nil {
			return "", err
		}
	}

	fmt.Println(responseStatus, responseError)
	return responseStatus, responseError
}

func main() {
	httpClient = &httpservice.HTTPClientV2{}
	awsClient = &aws_client.AWSClient{}
	lambda.Start(HandleRequest)
}
