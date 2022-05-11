package common_handler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/legacy_client"
)

const (
	DBSecretARN                    = "DBSecretARN"
	envLegacyUploadToEvossEndpoint = "LEGACY_EVOSS_ENDPOINT"
	envLegacyAuthSecret            = "LEGACY_AUTH_SECRET"
	legacyAuthKey                  = "TOKEN"
	region                         = "us-east-2"
)

type CommonHandler struct {
	AwsClient    aws_client.IAWSClient
	HttpClient   httpservice.IHTTPClientV2
	DBClient     documentDB_client.IDocDBClient
	LegacyClient legacy_client.ILegacyClient
}

func New(awsClient, httpClient, dbClient, legacyClient bool) CommonHandler {
	CommonHandlerObject := CommonHandler{}
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
		fmt.Println("fetching db secrets")
		if CommonHandlerObject.AwsClient == nil {
			CommonHandlerObject.AwsClient = &aws_client.AWSClient{}
		}
		secrets, err := CommonHandlerObject.AwsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
		if err != nil {
			fmt.Println("Unable to fetch DocumentDb in secret")
			panic(err)
		}
		CommonHandlerObject.DBClient = documentDB_client.NewDBClientService(secrets)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = CommonHandlerObject.DBClient.CheckConnection(ctx); err != nil {
			panic(err)
		}
	}
	if legacyClient {
		endpoint := os.Getenv(envLegacyUploadToEvossEndpoint)
		authsecret := os.Getenv(envLegacyAuthSecret)
		if CommonHandlerObject.AwsClient == nil {
			CommonHandlerObject.AwsClient = &aws_client.AWSClient{}
		}
		secretMap, err := CommonHandlerObject.AwsClient.GetSecret(context.Background(), authsecret, region)
		if err != nil {
			panic(err)
		}
		CommonHandlerObject.LegacyClient = legacy_client.New(endpoint, secretMap[legacyAuthKey].(string), CommonHandlerObject.HttpClient)
	}
	return CommonHandlerObject
}
