package main

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"io/ioutil"
// 	"net/http"
// 	"testing"

// 	"github.com/aws/aws-sdk-go/service/lambda"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// 	"github.eagleview.com/engineering/symphony-service/commons/mocks"
// )

// //var testContext = log_config.SetTraceIdInContext(context.Background(), "44825849", "9cabffdf-e980-0bbf-b481-0048f7a88bef")
// var (
// 	eventData1 = []byte(`{
//   "ImageMetadata": "S3 Link for the ImageMetadata Json",
//   "meta": {
//     "callbackId": "a2192b7d-a78f-4fa3-90fd-5da69860d464",
//     "callbackUrl": "arn:aws:lambda:us-east-2:356071200662:function:app-dev-1x0-lambda-callbacklambda"
//   },
//   "orderId": "44828269",
//   "selectedImages": [
//     {
//       "S3Path": "S3 Link for the Ortho image",
//       "View": "O"
//     },
//     {
//       "S3Path": "S3 Link for the North image",
//       "View": "N"
//     },
//     {
//       "S3Path": "S3 Link for the South image",
//       "View": "S"
//     },
//     {
//       "S3Path": "S3 Link for the East image",
//       "View": "E"
//     },
//     {
//       "S3Path": "S3 Link for the West image",
//       "View": "W"
//     }
//   ],
//   "workflowId": "45de094f-816a-f0b7-3e1f-b74402dfd379"
// }`)
// )

// // func TestHandler(t *testing.T) {

// // 	var eventDataReq *eventData
// // 	scannerErr := json.Unmarshal(eventData1, &eventDataReq)
// // 	assert.NoError(t, scannerErr)

// // 	aws_Client := new(mocks.IAWSClient)
// // 	http_Client := new(mocks.MockHTTPClient)

// // 	convertorOutput := lambda.InvokeOutput{
// // 		Payload: []byte(`{"errorType": "RetriableError"}`),
// // 	}

// // 	aws_Client.Mock.On("FetchS3BucketPath", "some s3 path").Return("", "", nil)
// // 	aws_Client.Mock.On("GetDataFromS3", mock.Anything, "", "").Return([]byte("dummy response"), nil)
// // 	aws_Client.Mock.On("InvokeLambda", mock.Anything, "", mock.Anything, false).Return(&convertorOutput, nil)
// // 	aws_Client.Mock.On("GetSecret", mock.Anything, "", region).Return(map[string]interface{}{legacyAuthKey: "token"}, nil)
// // 	http_Client.Mock.On("Post").Return(&http.Response{
// // 		StatusCode: http.StatusOK,
// // 		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
// // 			"Success": true,
// // 			"Message": "Report Status updated for ReportId: "
// // 		}`))),
// // 	}, nil)

// // 	expectedResp := &LambdaOutput{
// // 		Status:      "success",
// // 		MessageCode: 200,
// // 		Message:     "report status updated successfully",
// // 	}
// // 	http_Client.Mock.On("Post").Return(&http.Response{
// // 		StatusCode: http.StatusOK,
// // 		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
// // 			"Success": true,
// // 			"Message": "Report Status updated for ReportId: "
// // 		}`))),
// // 	}, nil)

// // 	resp, err := handler(context.Background(), eventDataReq)
// // 	assert.NoError(t, err)
// // 	assert.Equal(t, expectedResp, resp)

// // }
