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

type LegacyUpdateRequest struct {
	ReportID     string
	Status       string
	SubStatus    string
	Notes        string
	HipsterJobId string
}

func (lc *LegacyClient) UpdateReportStatus(ctx context.Context, req *LegacyUpdateRequest) error {

	if req.ReportID == "" || req.Status == "" || req.SubStatus == "" {
		return errors.New("invalid request body")
	}

	payload, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/UpdateReportStatus", lc.EndPoint)
	headers := map[string]string{
		"Authorization": "Basic " + lc.AuthToken,
	}

	response, err := lc.HTTPClient.Post(ctx, url, bytes.NewReader(payload), headers)
	if err != nil {
		log.Error(ctx, "error : ", err)
		return err
	}

	if response.StatusCode != http.StatusOK {
		log.Error(ctx, "response not ok: ", response.StatusCode)
		return errors.New("response not ok from legacy")
	}
	return nil
}

func (lc *LegacyClient) UploadMLJsonToEvoss(ctx context.Context, reportId string, mlJson []byte) error {

	if reportId == "" || len(mlJson) == 0 {
		return errors.New("invalid request body")
	}

	url := fmt.Sprintf("%s/UploadMLJson?reportId=%s", lc.EndPoint, reportId)
	headers := map[string]string{
		"Authorization": "Basic " + lc.AuthToken,
	}

	response, err := lc.HTTPClient.Post(ctx, url, bytes.NewReader(mlJson), headers)
	if err != nil {
		log.Error(ctx, "error : ", err)
		return err
	}

	if response.StatusCode != http.StatusOK {
		log.Error(ctx, "response not ok: ", response.StatusCode)
		return errors.New("response not ok from legacy")
	}
	return nil
}
