package main

import (
	"context"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/documentDB_client"
	"go.mongodb.org/mongo-driver/bson"
)

var commonHandler common_handler.CommonHandler

type eventData struct {
	ReportID   string `json:"reportId"`
	OrderID    string `json:"orderId"`
	WorkflowID string `json:"workflowId"`
}

const AllowedHipsterCount = "AllowedHipsterCount"

func handler(ctx context.Context, eventData *eventData) (map[string]interface{}, error) {
	// Get the count of data from documnetDB for last 24 hours
	count, err := commonHandler.DBClient.GetHipsterCountPerDay(ctx)
	if err != nil {
		log.Errorf(ctx, "Unable to Fetch from DocumentDb error = %s", err)
		return map[string]interface{}{"status": "failed"}, err
	}
	// if totalcount>50 return twister else Hipster
	threshold, err := strconv.ParseInt(os.Getenv(AllowedHipsterCount), 10, 64)
	if err != nil {
		log.Errorf(ctx, "Unable to convert string to int64 error = %s", err)
		return map[string]interface{}{"status": "failed"}, err
	}
	if count <= threshold {
		query := bson.M{"_id": eventData.WorkflowID}
		setrecord := bson.M{
			"$set": bson.M{
				"flowType": "Hipster",
			}}

		commonHandler.DBClient.UpdateDocumentDB(ctx, query, setrecord, documentDB_client.WorkflowDataCollection)
		return map[string]interface{}{"Path": "Hipster"}, nil
	} else {
		query := bson.M{"_id": eventData.WorkflowID}
		setrecord := bson.M{
			"$set": bson.M{
				"flowType": "Twister",
			}}

		commonHandler.DBClient.UpdateDocumentDB(ctx, query, setrecord, documentDB_client.WorkflowDataCollection)
		return map[string]interface{}{"Path": "Twister"}, nil
	}

}

func main() {

	lambda.Start(handler)
}
