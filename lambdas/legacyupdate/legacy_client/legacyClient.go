package legacy_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/assess-platform-library/log"
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
	log.Debug(ctx, "Endpoint: "+url)
	headers := map[string]string{
		"Authorization": "Basic " + lc.AuthToken,
	}

	response, err := lc.HTTPClient.Post(ctx, url, bytes.NewReader(payload), headers)
	if err != nil {
		log.Error(ctx, "Error while making http call, error: ", err)
		return err
	}

	if response.StatusCode != http.StatusOK {
		log.Error(ctx, "response not ok: ", response.StatusCode)
		return errors.New("response not ok from legacy")
	}
	log.Info(ctx, "UpdateReportStatus successful...")
	return nil
}
