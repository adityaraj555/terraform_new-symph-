package common_handler

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/slack"
)

const (
	DBSecretARN             = "DBSecretARN"
	legacyAuthKey           = "TOKEN"
	region                  = "us-east-2"
	slackKey                = "SLACK_TOKEN"
	slackChannel            = "SlackChannel"
	ContextDeadlineExceeded = "context deadline exceeded"
)

type CommonHandler struct {
	AwsClient   aws_client.IAWSClient
	HttpClient  httpservice.IHTTPClientV2
	DBClient    documentDB_client.IDocDBClient
	SlackClient slack.ISlackClient
	Secrets     map[string]interface{}
}

func New(awsClient, httpClient, dbClient, slackClient, secretsRequired bool) CommonHandler {
	CommonHandlerObject := CommonHandler{}
	var secrets map[string]interface{}
	var err error
	if secretsRequired || awsClient || dbClient || slackClient {
		SecretARN := os.Getenv(DBSecretARN)
		log.Info("fetching db secrets")
		if CommonHandlerObject.AwsClient == nil {
			CommonHandlerObject.AwsClient = &aws_client.AWSClient{}
		}
		secrets, err = CommonHandlerObject.AwsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
		if err != nil {
			log.Error(context.Background(), err)
			panic(err)
		}
		CommonHandlerObject.Secrets = secrets
	}
	if httpClient {
		CommonHandlerObject.HttpClient = &httpservice.HTTPClientV2{}
		httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
			APITimeout: 90,
		})
	}

	if dbClient {
		CommonHandlerObject.DBClient = documentDB_client.NewDBClientService(secrets)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = CommonHandlerObject.DBClient.CheckConnection(ctx); err != nil {
			panic(err)
		}
	}

	if slackClient {
		slackErrChannel := os.Getenv(slackChannel)
		CommonHandlerObject.SlackClient = slack.NewSlackClient(secrets[slackKey].(string), slackErrChannel)
	}

	return CommonHandlerObject
}

func (CommonHandler *CommonHandler) MakePostCall(ctx context.Context, URL string, payload []byte, headers map[string]string) ([]byte, error) {
	log.Info(ctx, "makePostCall reached...")
	var resp *http.Response
	var err error
	resp, err = CommonHandler.HttpClient.Post(ctx, URL, bytes.NewReader(payload), headers)

	if err != nil {
		log.Error(ctx, "Error while making http request: ", err.Error())
		if strings.Contains(err.Error(), ContextDeadlineExceeded) {
			return nil, error_handler.NewRetriableError(error_codes.ErrorMakingPostPutOrDeleteCall, err.Error())
		}
		return nil, error_handler.NewServiceError(error_codes.ErrorMakingPostPutOrDeleteCall, err.Error())
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "Error while reading response body: ", err.Error())
	}
	if resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusServiceUnavailable {
		return responseBody, error_handler.NewRetriableError(error_codes.ReceivedInternalServerError, fmt.Sprintf("%d status code received", resp.StatusCode))
	}
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "20") {
		log.Error(ctx, "invalid http status code received, statusCode: ", resp.StatusCode)
		return responseBody, error_handler.NewServiceError(error_codes.ReceivedInvalidHTTPStatusCode, "received invalid http status code: "+strconv.Itoa(resp.StatusCode))
	}
	log.Info(ctx, "makePostCall finished...")
	return responseBody, nil
}
