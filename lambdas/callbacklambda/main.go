package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
)

var awsClient aws_client.AWSClient

const Success = "success"

type RequestBody struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message"`
	MessageCode int                    `json:"messageCode"`
	CallbackID  string                 `json:"callbackId"`
	Response    map[string]interface{} `json:"response"`
}

var SecretARN string

func Handler(ctx context.Context, CallbackRequest map[string]interface{}) (map[string]interface{}, error) {
	secrets, err := awsClient.GetSecret(context.Background(), SecretARN, "us-east-2")
	documentDB_client.Username = secrets["username"].(string)
	documentDB_client.Password = secrets["password"].(string)
	documentDB_client.ClusterEndpoint = fmt.Sprintf("%v:%v", secrets["host"], secrets["port"])
	connectionURI := fmt.Sprintf(documentDB_client.ConnectionStringTemplate, documentDB_client.Username, documentDB_client.Password, documentDB_client.ClusterEndpoint, documentDB_client.ReadPreference)
	NewDBClient := documentDB_client.NewDBClientService(documentDB_client.CaFilePath, connectionURI)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	documentDB_client.DBClient = NewDBClient.DBClient
	err = documentDB_client.DBClient.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	mySession := session.Must(session.NewSession())
	svc := sfn.New(mySession)
	body := CallbackRequest["body"].(string)
	var requestBody = RequestBody{}
	err = json.Unmarshal([]byte(body), &requestBody)
	if err != nil {
		return map[string]interface{}{"status": "failed"}, err
	}
	MetaData, err := NewDBClient.FetchMetaData(requestBody.CallbackID)
	if err != nil {
		return map[string]interface{}{"status": "failed"}, err
	}
	if requestBody.Status == Success {
		byteData, _ := json.Marshal(requestBody.Response)
		jsonResponse := string(byteData)
		taskoutput, err := svc.SendTaskSuccess(&sfn.SendTaskSuccessInput{
			TaskToken: &MetaData.Data.TaskToken,
			Output:    &jsonResponse,
		})
		fmt.Println(&taskoutput, err)
	} else {
		messageCode := strconv.Itoa(requestBody.MessageCode)
		taskoutput, err := svc.SendTaskFailure(&sfn.SendTaskFailureInput{
			TaskToken: &MetaData.Data.TaskToken,
			Cause:     &requestBody.Message,
			Error:     &messageCode,
		})
		fmt.Println(&taskoutput, err)
	}
	err = NewDBClient.DeleteMetaData(requestBody.CallbackID)
	if err != nil {
		return map[string]interface{}{"status": "failed"}, err
	}
	return map[string]interface{}{"status": Success}, nil
}

func main() {

	lambda.Start(Handler)
}
