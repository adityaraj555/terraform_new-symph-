package legacy_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
)

func New(endpoint, authtoken string, httpClient httpservice.IHTTPClientV2) *LegacyClient {
	return &LegacyClient{
		AuthToken:  authtoken,
		EndPoint:   endpoint,
		HTTPClient: httpClient,
	}
}

type LegacyClient struct {
	HTTPClient httpservice.IHTTPClientV2
	EndPoint   string
	AuthToken  string
}

type ILegacyClient interface {
	UpdateReportStatus(ctx context.Context, req *LegacyUpdateRequest) error
}

type LegacyUpdateRequest struct {
	ReportID     string
	Status       string
	SubStatus    string
	Notes        string
	HipsterJobId string
}

func (lc *LegacyClient) UpdateReportStatus(ctx context.Context, req *LegacyUpdateRequest) error {
	log.Info(ctx, "UpdateReportStatus reached...")

	if req.ReportID == "" || req.Status == "" || req.SubStatus == "" {
		log.Errorf(ctx, "invalid request body, req body: %+v", req)
		return errors.New("invalid request body")
	}

	payload, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/UpdateReportStatus", lc.EndPoint)
	log.Info(ctx, "Endpoint: "+url)
	headers := map[string]string{
		"Authorization": "Basic " + lc.AuthToken,
	}

	response, err := lc.HTTPClient.Post(ctx, url, bytes.NewReader(payload), headers)
	if err != nil {
		log.Error(ctx, "Error while making http call, error: ", err)
		return err
	}

	if response.StatusCode == http.StatusInternalServerError || response.StatusCode == http.StatusServiceUnavailable {
		return error_handler.NewRetriableError(error_codes.ErrorWhileUpdatingLegacy, fmt.Sprintf("%d status code received", response.StatusCode))
	}
	if !strings.HasPrefix(strconv.Itoa(response.StatusCode), "20") {
		log.Error(ctx, "response not ok: ", response.StatusCode)
		return errors.New("response not ok from legacy")
	}
	log.Info(ctx, "UpdateReportStatus successful...")
	return nil
}
