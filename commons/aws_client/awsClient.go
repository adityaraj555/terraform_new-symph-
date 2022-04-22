package aws_client

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type IAWSClient interface {
	GetSecret(ctx context.Context, secretName, region string) (map[string]interface{}, error)
}

type AWSClient struct{}

func (ac *AWSClient) GetSecret(ctx context.Context, secretName, region string) (map[string]interface{}, error) {
	resp := make(map[string]interface{})
	svc := secretsmanager.New(
		session.Must(session.NewSession()),
		aws.NewConfig().WithRegion(region),
	)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		return resp, err
	}

	var secretString string
	if result.SecretString != nil {
		secretString = *result.SecretString
	}

	err = json.Unmarshal([]byte(secretString), &resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
