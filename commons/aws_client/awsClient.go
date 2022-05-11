package aws_client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sfn"
)

type IAWSClient interface {
	GetSecret(ctx context.Context, secretName, region string) (map[string]interface{}, error)
	GetSecretString(ctx context.Context, secretManagerNameArn string) (string, error)
	InvokeLambda(ctx context.Context, lambdafunctionArn string, payload map[string]interface{}) (*lambda.InvokeOutput, error)
	StoreDataToS3(ctx context.Context, bucketName, s3KeyPath string, responseBody []byte) error
	InvokeSFN(Input, StateMachineArn *string) (string, error)
	GetDataFromS3(ctx context.Context, bucketName, s3KeyPath string) ([]byte, error)
	FetchS3BucketPath(s3Path string) (string, string, error)
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

func (ac *AWSClient) GetSecretString(ctx context.Context, secretManagerNameArn string) (string, error) {
	region := "us-east-2"
	sess, _ := session.NewSession()
	svc := secretsmanager.New(sess,
		aws.NewConfig().WithRegion(region))

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretManagerNameArn),
		VersionStage: aws.String("AWSCURRENT"),
	}
	result, err := svc.GetSecretValue(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeResourceNotFoundException:
				fmt.Println(secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			case secretsmanager.ErrCodeInvalidParameterException:
				fmt.Println(secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
			case secretsmanager.ErrCodeInvalidRequestException:
				fmt.Println(secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
			case secretsmanager.ErrCodeDecryptionFailure:
				fmt.Println(secretsmanager.ErrCodeDecryptionFailure, aerr.Error())
			case secretsmanager.ErrCodeInternalServiceError:
				fmt.Println(secretsmanager.ErrCodeInternalServiceError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return "", err
	}

	return *result.SecretString, nil
}

func (ac *AWSClient) InvokeLambda(ctx context.Context, lambdafunctionArn string, payload map[string]interface{}) (*lambda.InvokeOutput, error) {
	region := "us-east-2"

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambda.New(sess, &aws.Config{Region: aws.String(region)})

	lambdaPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling payload request")
		return nil, err
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(lambdafunctionArn), Payload: lambdaPayload})
	if err != nil {
		fmt.Println("Error calling " + lambdafunctionArn)
	}

	return result, err
}

func (ac *AWSClient) StoreDataToS3(ctx context.Context, bucketName, s3KeyPath string, responseBody []byte) error {

	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	region := "us-east-2"
	svc := s3.New(sess, aws.NewConfig().WithRegion(region))

	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(string(responseBody))),
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3KeyPath),
	}

	_, err = svc.PutObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}
	return nil
}

func (ac *AWSClient) InvokeSFN(Input, StateMachineArn *string) (string, error) {
	mySession := session.Must(session.NewSession())
	svc := sfn.New(mySession)
	out, err := svc.StartExecution(&sfn.StartExecutionInput{
		Input:           Input,
		StateMachineArn: StateMachineArn,
	})
	if err != nil {
		return "", err
	}
	return *out.ExecutionArn, nil
}

func (ac *AWSClient) GetDataFromS3(ctx context.Context, bucketName, s3KeyPath string) ([]byte, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	region := "us-east-2"
	svc := s3.New(sess, aws.NewConfig().WithRegion(region))
	requestInput := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3KeyPath),
	}
	result, err := svc.GetObject(requestInput)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return ioutil.ReadAll(result.Body)
}

func (ac *AWSClient) FetchS3BucketPath(s3Path string) (string, string, error) {
	if !(strings.HasPrefix(s3Path, "s3://") || strings.HasPrefix(s3Path, "S3://")) {
		s3Path = "s3://" + s3Path
	}
	u, err := url.Parse(s3Path)
	if err != nil {
		return "", "", err
	}
	return u.Host, u.Path, nil
}
