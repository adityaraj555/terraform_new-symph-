package aws_client

import (
	"context"
	"encoding/json"
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
	"github.eagleview.com/engineering/assess-platform-library/log"
)

const success = "success"

type IAWSClient interface {
	GetSecret(ctx context.Context, secretName, region string) (map[string]interface{}, error)
	GetSecretString(ctx context.Context, secretManagerNameArn string) (string, error)
	InvokeLambda(ctx context.Context, lambdafunctionArn string, payload map[string]interface{}, isWaitTask bool) (*lambda.InvokeOutput, error)
	StoreDataToS3(ctx context.Context, bucketName, s3KeyPath string, responseBody []byte) error
	InvokeSFN(Input, StateMachineArn, Name *string) (string, error)
	GetDataFromS3(ctx context.Context, bucketName, s3KeyPath string) ([]byte, error)
	FetchS3BucketPath(s3Path string) (string, string, error)
	CloseWaitTask(ctx context.Context, status, TaskToken, Output, Cause, Error string) error
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
				log.Error(ctx, secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			case secretsmanager.ErrCodeInvalidParameterException:
				log.Error(ctx, secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
			case secretsmanager.ErrCodeInvalidRequestException:
				log.Error(ctx, secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
			case secretsmanager.ErrCodeDecryptionFailure:
				log.Error(ctx, secretsmanager.ErrCodeDecryptionFailure, aerr.Error())
			case secretsmanager.ErrCodeInternalServiceError:
				log.Error(ctx, secretsmanager.ErrCodeInternalServiceError, aerr.Error())
			default:
				log.Error(ctx, aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Error(ctx, err.Error())
		}
		return "", err
	}

	return *result.SecretString, nil
}

func (ac *AWSClient) InvokeLambda(ctx context.Context, lambdafunctionArn string, payload map[string]interface{}, isWaitTask bool) (*lambda.InvokeOutput, error) {
	region := "us-east-2"
	var InvocationType string
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambda.New(sess, &aws.Config{Region: aws.String(region)})

	lambdaPayload, err := json.Marshal(payload)
	if err != nil {
		log.Error(ctx, "Error marshalling payload request")
		return nil, err
	}
	if isWaitTask {
		InvocationType = "Event"
	} else {
		InvocationType = "RequestResponse"
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(lambdafunctionArn), Payload: lambdaPayload, InvocationType: aws.String(InvocationType)})
	if err != nil {
		log.Error(ctx, "Error calling "+lambdafunctionArn)
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
				log.Error(ctx, aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Error(ctx, err.Error())
		}
		return err
	}
	return nil
}

func (ac *AWSClient) InvokeSFN(Input, StateMachineArn, Name *string) (string, error) {
	mySession := session.Must(session.NewSession())
	svc := sfn.New(mySession)
	out, err := svc.StartExecution(&sfn.StartExecutionInput{
		Input:           Input,
		StateMachineArn: StateMachineArn,
		Name:            Name,
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

func (ac *AWSClient) CloseWaitTask(ctx context.Context, status, TaskToken, Output, Cause, Error string) error {
	mySession := session.Must(session.NewSession())
	svc := sfn.New(mySession)
	if status == success {
		taskoutput, err := svc.SendTaskSuccess(&sfn.SendTaskSuccessInput{
			TaskToken: &TaskToken,
			Output:    &Output,
		})
		if err != nil {
			log.Error(ctx, "Unable to Mark Task as Success", taskoutput, err)
		}
		return err
	} else {
		taskoutput, err := svc.SendTaskFailure(&sfn.SendTaskFailureInput{
			TaskToken: &TaskToken,
			Cause:     &Cause,
			Error:     &Error,
		})
		if err != nil {
			log.Error(ctx, "Unable to Mark Task as Failure", taskoutput, err)
		}
		return err
	}
}
