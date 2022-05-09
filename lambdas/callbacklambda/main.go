package main

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.eagleview.com/engineering/platform-gosdk/log"
	"github.eagleview.com/engineering/symphony-service/commons/aws_client"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/enums"
	"github.eagleview.com/engineering/symphony-service/commons/validator"
)

var awsClient aws_client.AWSClient
var newDBClient *documentDB_client.DocDBClient

type RequestBody struct {
	Status      enums.TaskStatus       `json:"status" validate:"required,taskStatus"`
	Message     string                 `json:"message"`
	MessageCode int                    `json:"messageCode"`
	CallbackID  string                 `json:"callbackId" validate:"required"`
	Response    map[string]interface{} `json:"response"`
}

const DBSecretARN = "DBSecretARN"
const success = "success"
const failure = "failure"
const rework = "rework"
const isReworkRequired = "isReworkRequired"

func Handler(ctx context.Context, CallbackRequest RequestBody) (map[string]interface{}, error) {
	var err error

	if err := validator.ValidateCallBackRequest(ctx, CallbackRequest); err != nil {
		return map[string]interface{}{"status": "failed"}, err
	}

	mySession := session.Must(session.NewSession())
	svc := sfn.New(mySession)
	StepExecutionData, err := newDBClient.FetchStepExecutionData(CallbackRequest.CallbackID)

	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	var stepstatus string = failure
	if CallbackRequest.Status.String() == rework {
		CallbackRequest.Response[isReworkRequired] = true
	} else {
		CallbackRequest.Response[isReworkRequired] = false
	}
	if CallbackRequest.Status.String() == success || CallbackRequest.Status.String() == rework {
		stepstatus = success
		byteData, _ := json.Marshal(CallbackRequest.Response)
		jsonResponse := string(byteData)
		taskoutput, err := svc.SendTaskSuccess(&sfn.SendTaskSuccessInput{
			TaskToken: &StepExecutionData.TaskToken,
			Output:    &jsonResponse,
		})
		fmt.Println(&taskoutput, err)
	} else {
		messageCode := strconv.Itoa(CallbackRequest.MessageCode)
		taskoutput, err := svc.SendTaskFailure(&sfn.SendTaskFailureInput{
			TaskToken: &StepExecutionData.TaskToken,
			Cause:     &CallbackRequest.Message,
			Error:     &messageCode,
		})
		fmt.Println(&taskoutput, err)
	}
	filter, query := newDBClient.BuildQueryForCallBack(documentDB_client.UpdateStepExecution, stepstatus, StepExecutionData.WorkflowId, StepExecutionData.StepId, StepExecutionData.TaskName, CallbackRequest.Response)
	err = newDBClient.UpdateDocumentDB(filter, query, documentDB_client.StepsDataCollection)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	filter, query = newDBClient.BuildQueryForCallBack(documentDB_client.UpdateWorkflowExecutionSteps, stepstatus, StepExecutionData.WorkflowId, StepExecutionData.StepId, StepExecutionData.TaskName, CallbackRequest.Response)
	err = newDBClient.UpdateDocumentDB(filter, query, documentDB_client.WorkflowDataCollection)
	if err != nil {
		return map[string]interface{}{"status": failure}, err
	}
	filter, query = newDBClient.BuildQueryForCallBack(documentDB_client.UpdateWorkflowExecutionStatus, stepstatus, StepExecutionData.WorkflowId, StepExecutionData.StepId, StepExecutionData.TaskName, CallbackRequest.Response)
	err = newDBClient.UpdateDocumentDB(filter, query, documentDB_client.WorkflowDataCollection)
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
			log.Error(err)
		}
	}
	lambda.Start(Handler)

}
