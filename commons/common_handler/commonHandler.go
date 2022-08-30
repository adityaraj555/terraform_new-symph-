package common_handler

import (
	"context"
	"os"
	"time"

	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/slack"
)

const (
	DBSecretARN   = "DBSecretARN"
	legacyAuthKey = "TOKEN"
	region        = "us-east-2"
	slackKey      = "SLACK_TOKEN"
	slackChannel  = "SlackChannel"
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
	if secretsRequired {
		SecretARN := os.Getenv(DBSecretARN)
		log.Info("fetching db secrets")
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
	if awsClient {
		CommonHandlerObject.AwsClient = &aws_client.AWSClient{}
	}

	if dbClient {
		SecretARN := os.Getenv(DBSecretARN)
		log.Info("fetching db secrets")
		if CommonHandlerObject.AwsClient == nil {
			CommonHandlerObject.AwsClient = &aws_client.AWSClient{}
		}
		if secrets == nil {
			secrets, err = CommonHandlerObject.AwsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
			if err != nil {
				log.Error("Unable to fetch DocumentDb in secret")
				panic(err)
			}
		}
		CommonHandlerObject.DBClient = documentDB_client.NewDBClientService(secrets)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = CommonHandlerObject.DBClient.CheckConnection(ctx); err != nil {
			panic(err)
		}
	}

	if slackClient {
		secretarn := os.Getenv(DBSecretARN)
		slackErrChannel := os.Getenv(slackChannel)
		if secrets == nil {
			secrets, err = CommonHandlerObject.AwsClient.GetSecret(context.Background(), secretarn, region)
			if err != nil {
				log.Error(context.Background(), err)
				panic(err)
			}
		}
		CommonHandlerObject.SlackClient = slack.NewSlackClient(secrets[slackKey].(string), slackErrChannel)
	}

	return CommonHandlerObject
}
