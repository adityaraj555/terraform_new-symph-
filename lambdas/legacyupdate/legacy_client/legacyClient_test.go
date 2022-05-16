package legacy_client_test

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
	main "github.eagleview.com/engineering/symphony-service/lambdas/legacyupdate/legacy_client"
)

func TestLegacyClient(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)

	LegacyClient := main.New("legacyURL", "AuthToken", http_Client)
	LegacyRequest := main.LegacyUpdateRequest{
		ReportID:     "12345",
		Status:       "inProcess",
		SubStatus:    "HipsterQCCompleted",
		HipsterJobId: "HipsterJobId",
	}
	http_Client.Mock.On("Post").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)
	err := LegacyClient.UpdateReportStatus(context.Background(), &LegacyRequest)
	assert.NoError(t, err)
}
func TestLegacyClientValidationErrorr(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)

	LegacyClient := main.New("legacyURL", "AuthToken", http_Client)
	LegacyRequest := main.LegacyUpdateRequest{
		ReportID:     "",
		Status:       "inProcess",
		SubStatus:    "HipsterQCCompleted",
		HipsterJobId: "HipsterJobId",
	}
	err := LegacyClient.UpdateReportStatus(context.Background(), &LegacyRequest)
	assert.Error(t, err)
}
func TestLegacyClientinvalidStatusCode(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)

	LegacyClient := main.New("legacyURL", "AuthToken", http_Client)
	LegacyRequest := main.LegacyUpdateRequest{
		ReportID:     "12345",
		Status:       "inProcess",
		SubStatus:    "HipsterQCCompleted",
		HipsterJobId: "HipsterJobId",
	}
	http_Client.Mock.On("Post").Return(&http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)
	err := LegacyClient.UpdateReportStatus(context.Background(), &LegacyRequest)
	assert.Error(t, err)
}
func TestLegacyClientinvalidStatusCode400(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)

	LegacyClient := main.New("legacyURL", "AuthToken", http_Client)
	LegacyRequest := main.LegacyUpdateRequest{
		ReportID:     "12345",
		Status:       "inProcess",
		SubStatus:    "HipsterQCCompleted",
		HipsterJobId: "HipsterJobId",
	}
	http_Client.Mock.On("Post").Return(&http.Response{
		StatusCode: 400,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, nil)
	err := LegacyClient.UpdateReportStatus(context.Background(), &LegacyRequest)
	assert.Error(t, err)
}
func TestLegacyClientErrorMakingAppiCall(t *testing.T) {
	http_Client := new(mocks.MockHTTPClient)

	LegacyClient := main.New("legacyURL", "AuthToken", http_Client)
	LegacyRequest := main.LegacyUpdateRequest{
		ReportID:     "12345",
		Status:       "inProcess",
		SubStatus:    "HipsterQCCompleted",
		HipsterJobId: "HipsterJobId",
	}
	http_Client.Mock.On("Post").Return(&http.Response{
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(string(`{
			"Success": true,
			"Message": "Report Status updated for ReportId: "
		}`))),
	}, errors.New("some error"))
	err := LegacyClient.UpdateReportStatus(context.Background(), &LegacyRequest)
	assert.Error(t, err)
}
