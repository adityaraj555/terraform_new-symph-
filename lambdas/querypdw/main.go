package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
	"github.eagleview.com/engineering/symphony-service/commons/utils"
)

type eventData struct {
	Vintage string `json:"vintage"`
	Action  string `json:"action"`
	Address struct {
		ParcelAddress string  `json:"parcelAddress"`
		Lat           float64 `json:"lat"`
		Long          float64 `json:"long"`
	} `json:"address"`
	CallbackID  string `json:"callbackId"`
	CallbackURL string `json:"callbackUrl"`
	ParcelID    string `json:"parcelId"`
	WorkflowID  string `json:"workflowId"`
}

type pdwValidationResponse struct {
	Data struct {
		Parcels []struct {
			ID         string `json:"id"`
			Structures []struct {
				Outline struct {
					Marker string `json:"marker"`
				} `json:"_outline"`
				ID string `json:"id"`
			} `json:"structures"`
			Lat     float64 `json:"lat"`
			Lon     float64 `json:"lon"`
			Address string  `json:"address"`
			City    string  `json:"city"`
			State   string  `json:"state"`
			Zip     string  `json:"zip"`
		} `json:"parcels"`
	} `json:"data"`
}

type eventResponse struct {
	Address    string  `json:"address,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
	ParcelID   string  `json:"parcelId,omitempty"`
	TriggerSIM bool    `json:"triggerSIM"`
	Message    string  `json:"message,omitempty"`
}

var commonHandler common_handler.CommonHandler
var auth_client utils.AuthTokenInterface = &utils.AuthTokenUtil{}

const (
	queryfilepath           = "query.gql"
	primary                 = "primary"
	ContextDeadlineExceeded = "context deadline exceeded"
	success                 = "success"
	failure                 = "failure"
	validatedata            = "validatedata"
	querydata               = "querydata"
	GraphEndpoint           = "GraphEndpoint"
	DBSecretARN             = "DBSecretARN"
	region                  = "us-east-2"
	NoParcelMessage         = "ParcelID does not exist in the graph response"
	NoStructureMessage      = "Structures does not exist in the graph response"
	StructurePresentMessage = "Structures exist in the graph response"
	appCode                 = "O2"
)

func handler(ctx context.Context, eventData eventData) (eventResponse, error) {
	ctx = log_config.SetTraceIdInContext(ctx, "", eventData.WorkflowID)

	log.Info(ctx, "querypdw reached...")

	// build the validation graph query
	query := generateValidationQuery(eventData)
	log.Info(ctx, "validation query generated...")
	// fetch the validation graph response
	response, err := fetchDataFromPDW(ctx, query)
	if err != nil {
		return eventResponse{}, err
	}
	var validationgraphResponse pdwValidationResponse
	err = json.Unmarshal(response, &validationgraphResponse)
	if err != nil {
		log.Error(ctx, "Error while unmarshalling graphresponse, error: ", err.Error())
		return eventResponse{}, error_handler.NewServiceError(error_codes.ErrorDecodingServiceResponse, err.Error())
	}
	// make callback if the parcel doesn't exist for the given address in graph
	if len(validationgraphResponse.Data.Parcels) == 0 || validationgraphResponse.Data.Parcels[0].ID == "" {
		log.Info(ctx, NoParcelMessage)
		err = makeCallBack(ctx, failure, NoParcelMessage, eventData.CallbackID, eventData.CallbackURL, error_codes.ParcelIDDoesnotExist, nil)
		return eventResponse{Message: NoParcelMessage}, err
	}
	parcelid := validationgraphResponse.Data.Parcels[0].ID
	isValid := isValidPDWResponse(validationgraphResponse, eventData.Vintage)
	// if structures doesnot exist
	if !isValid {
		// make callback if structures doesn't exist after ingestion
		if eventData.Action == querydata {
			err = error_handler.NewRetriableError(error_codes.ErrorQueryingPDWAfterIngestion, "unable to query data after ingestion")
			return eventResponse{}, err
		}
		Address := fmt.Sprintf("%s %s %s %s", validationgraphResponse.Data.Parcels[0].Address, validationgraphResponse.Data.Parcels[0].City, validationgraphResponse.Data.Parcels[0].State, validationgraphResponse.Data.Parcels[0].Zip)
		// Trigger SIM
		triggerSIMResponse := eventResponse{
			Latitude:   validationgraphResponse.Data.Parcels[0].Lat,
			Longitude:  validationgraphResponse.Data.Parcels[0].Lon,
			ParcelID:   validationgraphResponse.Data.Parcels[0].ID,
			TriggerSIM: true,
			Address:    Address,
			Message:    NoStructureMessage,
		}
		return triggerSIMResponse, nil
	} else {
		gqlbytearray, err := ioutil.ReadFile(queryfilepath)
		if err != nil {
			log.Error(ctx, "Unable to read query file: ", err)
			return eventResponse{}, error_handler.NewServiceError(error_codes.ErrorReadingQueryFile, err.Error())
		}
		graphquery := string(gqlbytearray)
		graphquery = fmt.Sprintf(graphquery, parcelid)
		response, err = fetchDataFromPDW(ctx, graphquery)
		if err != nil {
			return eventResponse{}, err
		}
		var graphResponse map[string]interface{}
		err = json.Unmarshal(response, &graphResponse)
		if err != nil {
			log.Error(ctx, "Error while unmarshalling graphresponse, error: ", err.Error())
			return eventResponse{}, error_handler.NewServiceError(error_codes.ErrorDecodingServiceResponse, err.Error())
		}
		err = makeCallBack(ctx, success, "", eventData.CallbackID, eventData.CallbackURL, error_codes.Success, graphResponse["data"].(map[string]interface{}))
		return eventResponse{Message: StructurePresentMessage}, err
	}
}

func generateValidationQuery(eventData eventData) string {
	commonattributelist := []string{"lat", "lon", "address", "city", "state", "zip", "id"}
	validationattributelist := []string{"structures%s._outline.marker", "structures%s.id"}
	validationattributelist = append(validationattributelist, commonattributelist...)
	query := GenerateGQL(validationattributelist, eventData.Address.Lat, eventData.Address.Long, eventData.ParcelID, eventData.Address.ParcelAddress, "")
	return query
}
func fetchDataFromPDW(ctx context.Context, query string) ([]byte, error) {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	secretMap := commonHandler.Secrets
	log.Info(ctx, "fetched secrets from secrets manager...")
	clientID := secretMap["clientID"].(string)
	clientSecret := secretMap["clientSecret"].(string)
	err := auth_client.AddAuthorizationTokenHeader(ctx, commonHandler.HttpClient, headers, appCode, clientID, clientSecret)
	if err != nil {
		log.Error(ctx, "Error while adding token to header, error: ", err.Error())
		return nil, err
	}
	log.Info(ctx, "added authtoken to headers...")
	graphrequest := map[string]interface{}{
		"query": query,
	}
	bytearray, err := json.Marshal(graphrequest)
	if err != nil {
		log.Error(ctx, "Error while marshalling graph request, error: ", err.Error())
		return nil, error_handler.NewServiceError(error_codes.ErrorSerializingCallOutPayload, err.Error())
	}
	responseBody, err := makePostCall(ctx, os.Getenv(GraphEndpoint), bytearray, headers)
	if err != nil {
		log.Error(ctx, "Error while making graph request, error: ", err.Error())
		return nil, err
	}
	return responseBody, nil
}
func makePostCall(ctx context.Context, URL string, payload []byte, headers map[string]string) ([]byte, error) {
	log.Info(ctx, "makePostCall reached...")
	var resp *http.Response
	var err error
	resp, err = commonHandler.HttpClient.Post(ctx, URL, bytes.NewReader(payload), headers)

	if err != nil {
		log.Error(ctx, "Error while making http request: ", err.Error())
		if strings.Contains(err.Error(), ContextDeadlineExceeded) {
			return nil, error_handler.NewRetriableError(error_codes.ErrorMakingPostPutOrDeleteCall, err.Error())
		}
		return nil, error_handler.NewServiceError(error_codes.ErrorMakingPostPutOrDeleteCall, err.Error())
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "Error while reading response body: ", err.Error())
	}
	if resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusServiceUnavailable {
		return responseBody, error_handler.NewRetriableError(error_codes.ReceivedInternalServerError, fmt.Sprintf("%d status code received", resp.StatusCode))
	}
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "20") {
		log.Error(ctx, "invalid http status code received, statusCode: ", resp.StatusCode)
		return responseBody, error_handler.NewServiceError(error_codes.ReceivedInvalidHTTPStatusCode, "received invalid http status code: "+strconv.Itoa(resp.StatusCode))
	}
	log.Info(ctx, "makePostCall finished...")
	return responseBody, nil
}

func makeCallBack(ctx context.Context, status, message, callbackId, callbackUrl string, messageCode int, graphresponse map[string]interface{}) error {
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	callbackRequest := map[string]interface{}{
		"callbackId":  callbackId,
		"status":      status,
		"message":     message,
		"messageCode": messageCode,
	}
	if status == success {
		callbackRequest["data"] = graphresponse
	}
	ByteArray, err := json.Marshal(callbackRequest)
	if err != nil {
		log.Error(ctx, "Error while marshalling callbackRequest, error: ", err.Error())
		return error_handler.NewServiceError(error_codes.ErrorSerializingCallOutPayload, err.Error())
	}
	_, err = makePostCall(ctx, callbackUrl, ByteArray, headers)
	if err != nil {
		log.Error(ctx, "Error while making callbackRequest, error: ", err.Error())
		return err
	}
	return nil
}
func isValidPDWResponse(pdwResponse pdwValidationResponse, minDate string) bool {
	if len(pdwResponse.Data.Parcels[0].Structures) == 0 {
		return false
	}
	for _, structure := range pdwResponse.Data.Parcels[0].Structures {
		structuremarker := structure.Outline.Marker
		if structuremarker == "" || (minDate != "" && structuremarker < minDate) {
			return false
		}
	}
	return true
}
func GenerateGQL(attributes []string, lat, long float64, parcelID, address, structureType string) string {
	topnode := ""
	if parcelID != "" {
		topnode = fmt.Sprintf(`parcels(ids: ["%s"])`, parcelID)
	} else if address != "" {
		topnode = fmt.Sprintf(`parcels(addresses: ["%s"])`, address)
	} else {
		topnode = fmt.Sprintf(`parcels(points:[{lat:%f,lon:%f}])`, lat, long)
	}
	parentChildAttributesMap := make(map[string][]string)
	for _, attr := range attributes {
		allattr := strings.Split(attr, ".")
		l := len(allattr)
		parentChildAttributesMap = inserttomap(parentChildAttributesMap, topnode, allattr[0])
		for i := range allattr {
			if i == l-1 {
				continue
			}
			parentChildAttributesMap = inserttomap(parentChildAttributesMap, allattr[i], allattr[i+1])
		}
	}
	query := "{ " + generatequery(parentChildAttributesMap, topnode) + " }"
	if structureType == primary {
		query = fmt.Sprintf(query, `(type: "main")`)
	} else {
		query = fmt.Sprintf(query, "")
	}
	return query
}

func generatequery(parentChildAttributesMap map[string][]string, node string) string {
	var graphql string
	if len(parentChildAttributesMap[node]) == 0 {
		graphql = node
	} else {
		for i := range parentChildAttributesMap[node] {
			graphql = graphql + "\n" + generatequery(parentChildAttributesMap, parentChildAttributesMap[node][i])
		}
		graphql = node + "{ " + graphql + " }"
	}
	return graphql
}

func inserttomap(parentChildAttributesMap map[string][]string, key, value string) map[string][]string {
	if _, ok := parentChildAttributesMap[key]; ok {
		if contains(parentChildAttributesMap[key], value) == false {
			parentChildAttributesMap[key] = append(parentChildAttributesMap[key], value)
		}
	} else {
		parentChildAttributesMap[key] = []string{value}
	}
	return parentChildAttributesMap
}
func contains(myarray []string, checkval string) bool {
	for _, value := range myarray {
		if value == checkval {
			return true
		}
	}
	return false
}
func notificationWrapper(ctx context.Context, req eventData) (eventResponse, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		errT := err.(error_handler.ICodedError)
		commonHandler.SlackClient.SendErrorMessage(errT.GetErrorCode(), "", req.WorkflowID, "querypdw", "querypdw", err.Error(), nil)
	}
	return resp, err
}
func main() {
	log_config.InitLogging("info")
	commonHandler = common_handler.New(true, true, false, true, true)
	httpservice.ConfigureHTTPClient(&httpservice.HTTPClientConfiguration{
		// APITimeout: 90,
	})
	lambda.Start(notificationWrapper)
}
