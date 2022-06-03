package main

import (
	"context"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"go.mongodb.org/mongo-driver/bson"
)

var commonHandler common_handler.CommonHandler

const (
	Success  = "success"
	loglevel = "info"
	failed   = "failed"
	hipster  = "Hipster"
)

type eventData struct {
	ReportID   string `json:"reportId"`
	OrderID    string `json:"orderId"`
	WorkflowID string `json:"workflowId"`
}

const AllowedHipsterCount = "AllowedHipsterCount"

func handler(ctx context.Context, eventData *eventData) (map[string]interface{}, error) {

	ctx = log_config.SetTraceIdInContext(ctx, eventData.ReportID, eventData.WorkflowID)
	// Get the count of data from documnetDB for last 24 hours UTC
	var Path = hipster
	count, err := commonHandler.DBClient.GetHipsterCountPerDay(ctx)
	if err != nil {
		log.Errorf(ctx, "Unable to Fetch from DocumentDb error = %s", err)
		return map[string]interface{}{"status": failed}, err
	}
	// if totalcount > 50 return twister else Hipster
	threshold, err := strconv.ParseInt(os.Getenv(AllowedHipsterCount), 10, 64)
	if err != nil {
		log.Errorf(ctx, "Unable to convert string to int64 error = %s", err)
		return map[string]interface{}{"status": failed}, err
	}
	if count > threshold {
		Path = "Twister"
	}
	query := bson.M{"_id": eventData.WorkflowID}
	setrecord := bson.M{
		"$set": bson.M{
			"flowType": Path,
		}}

	err = commonHandler.DBClient.UpdateDocumentDB(ctx, query, setrecord, documentDB_client.WorkflowDataCollection)
	if err != nil {
		log.Errorf(ctx, "Unable to UpdateDocumentDB error = %s", err)
		return map[string]interface{}{"status": failed}, err
	}
	return map[string]interface{}{"Path": Path, "status": Success}, nil
}

func notifcationWrapper(ctx context.Context, eventData *eventData) (map[string]interface{}, error) {
	resp, err := handler(ctx, eventData)
	if err != nil {
		commonHandler.SlackClient.SendErrorMessage(eventData.ReportID, eventData.WorkflowID, "throttle", err.Error(), nil)
	}
	return resp, err
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(false, false, true, false)
	lambda.Start(notifcationWrapper)

}
