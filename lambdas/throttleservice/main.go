package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
)

type eventData struct {
	ReportID   string `json:"reportId"`
	OrderID    string `json:"orderId"`
	WorkflowID string `json:"workflowId"`
}

func handler(ctx context.Context, eventData *eventData) (map[string]interface{}, error) {
	return map[string]interface{}{"Path": "Hipster"}, nil
}

func main() {

	lambda.Start(handler)
}
