package main

import (
	"context"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/enums"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"go.mongodb.org/mongo-driver/bson"
)

var commonHandler common_handler.CommonHandler

const (
	Success  = "success"
	loglevel = "info"
	failed   = "failed"
	hipster  = "Hipster"
	twister  = "Twister"
)

type eventData struct {
	ReportID      string `json:"reportId"`
	OrderID       string `json:"orderId"`
	WorkflowID    string `json:"workflowId"`
	OrderType     string `json:"orderType"`
	IsPenetration bool   `json:"isPenetration"`
}

const AllowedHipsterCount = "AllowedHipsterCount"

func handler(ctx context.Context, eventData *eventData) (map[string]interface{}, error) {

	log.Infof(ctx, "Reached throttle logic handler")
	ctx = log_config.SetTraceIdInContext(ctx, eventData.ReportID, eventData.WorkflowID)

	// set the execution to twister or hipster
	Path, err := getWorkflowExecutionPath(ctx, eventData)
	if err != nil {
		return map[string]interface{}{"status": failed}, err
	}

	query := bson.M{"_id": eventData.WorkflowID}
	setrecord := bson.M{
		"$set": bson.M{
			"flowType": Path,
		}}

	err = commonHandler.DBClient.UpdateDocumentDB(ctx, query, setrecord, documentDB_client.WorkflowDataCollection)
	if err != nil {
		log.Errorf(ctx, "Unable to UpdateDocumentDB error = %s", err)
		return map[string]interface{}{"status": failed}, error_handler.NewServiceError(error_codes.ErrorUpdatingWorkflowDataInDB, err.Error())
	}
	return map[string]interface{}{"Path": Path, "status": Success}, nil
}

func getWorkflowExecutionPath(ctx context.Context, eventData *eventData) (string, error) {

	// Get the count of data from documnetDB for last 24 hours UTC
	count, err := commonHandler.DBClient.GetHipsterCountPerDay(ctx)
	if err != nil {
		log.Errorf(ctx, "Unable to Fetch from DocumentDb error = %s", err)
		return "", error_handler.NewServiceError(error_codes.ErrorFetchingHipsterCountFromDB, err.Error())
	}

	threshold, err := strconv.ParseInt(os.Getenv(AllowedHipsterCount), 10, 64)
	if err != nil {
		log.Errorf(ctx, "Unable to convert string to int64 error = %s", err)
		return "", error_handler.NewServiceError(error_codes.ErrorConvertingAllowedHipsterCountToInteger, err.Error())
	}

	// Move to hipster only when count is less than threshold and product matches
	if (count < threshold) && hipsterAllowed(ctx, eventData) {
		log.Infof(ctx, "Path is set as hipster", err)
		return hipster, nil
	}
	log.Infof(ctx, "Path is set as twister", err)
	return twister, nil
}

func hipsterAllowed(ctx context.Context, eventData *eventData) bool {
	log.Infof(ctx, "Checking Hipster is allowed or not")
	if eventData.IsPenetration || !enums.IsHipsterCompatible(eventData.OrderType) {
		return false
	}
	return true
}

func notifcationWrapper(ctx context.Context, eventData *eventData) (map[string]interface{}, error) {
	resp, err := handler(ctx, eventData)
	if err != nil {
		cerr := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(cerr.GetErrorCode(), eventData.ReportID, eventData.WorkflowID, "throttleService", "throttle", err.Error(), nil)
	}
	return resp, err
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(false, false, true, false, false)
	lambda.Start(notifcationWrapper)

}
