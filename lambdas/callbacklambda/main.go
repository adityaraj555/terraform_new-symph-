package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"go.mongodb.org/mongo-driver/bson"
)

var awsClient aws_client.AWSClient
var newDBClient *documentDB_client.DocDBClient

type RequestBody struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message"`
	MessageCode int                    `json:"messageCode"`
	CallbackID  string                 `json:"callbackId"`
	Response    map[string]interface{} `json:"response"`
}

const DBSecretARN = "DBSecretARN"
const success = "success"
const failure = "failure"

func Handler(ctx context.Context, CallbackRequest map[string]interface{}) (map[string]interface{}, error) {
	var err error
	mySession := session.Must(session.NewSession())
	svc := sfn.New(mySession)
	var body string
	if requestbody, ok := CallbackRequest["body"]; ok {
		body = requestbody.(string)
	} else {
		return map[string]interface{}{"status": failure}, errors.New("body is empty in request body")
	}
	var requestBody = RequestBody{}
	err = json.Unmarshal([]byte(body), &requestBody)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	StepExecutionData, err := newDBClient.FetchStepExecution(requestBody.CallbackID)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	var stepstatus string = failure
	if requestBody.Status == success {
		stepstatus = success
		byteData, _ := json.Marshal(requestBody.Response)
		jsonResponse := string(byteData)
		taskoutput, err := svc.SendTaskSuccess(&sfn.SendTaskSuccessInput{
			TaskToken: &StepExecutionData.TaskToken,
			Output:    &jsonResponse,
		})
		fmt.Println(&taskoutput, err)
	} else {
		messageCode := strconv.Itoa(requestBody.MessageCode)
		taskoutput, err := svc.SendTaskFailure(&sfn.SendTaskFailureInput{
			TaskToken: &StepExecutionData.TaskToken,
			Cause:     &requestBody.Message,
			Error:     &messageCode,
		})
		fmt.Println(&taskoutput, err)
	}
	query := bson.M{
		"_id": StepExecutionData.StepId,
	}
	update := bson.M{
		"$set": bson.M{
			"output":  requestBody.Response,
			"status":  requestBody.Status,
			"endTime": time.Now().Unix(),
		},
	}
	err = newDBClient.UpdateDocumentDB(query, update, documentDB_client.StepsDataCollection)
	query = bson.M{
		"_id":                       StepExecutionData.WorkflowId,
		"stepsPassedThrough.stepId": requestBody.CallbackID,
	}

	update = bson.M{
		"$set": bson.M{
			"stepsPassedThrough.$.status": stepstatus,
		},
	}
	err = newDBClient.UpdateDocumentDB(query, update, documentDB_client.WorkflowDataCollection)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	query = bson.M{
		"_id": StepExecutionData.WorkflowId,
	}
	update = bson.M{
		"$set": bson.M{
			"updatedAt": time.Now().Unix(),
			"runningState": bson.M{
				StepExecutionData.TaskName: stepstatus,
			},
		},
	}
	err = newDBClient.UpdateDocumentDB(query, update, documentDB_client.WorkflowDataCollection)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	return map[string]interface{}{"status": success}, nil
}

func main() {
	if newDBClient == nil {
		SecretARN := os.Getenv(DBSecretARN)
		fmt.Println("fetching db secrets")
		secrets, err := awsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
		if err != nil {
			fmt.Println("Unable to fetch DocumentDb in secret")
		}
		newDBClient = documentDB_client.NewDBClientService(secrets)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = newDBClient.DBClient.Connect(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}
	lambda.Start(Handler)
}
