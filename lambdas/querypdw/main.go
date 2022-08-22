package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, eventData map[string]interface{}) (map[string]interface{}, error) {

	return map[string]interface{}{"status": "success"}, nil
}

func main() {

	lambda.Start(handler)

}
