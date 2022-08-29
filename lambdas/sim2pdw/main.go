package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.eagleview.com/engineering/assess-platform-library/log"
	"github.eagleview.com/engineering/symphony-service/commons/common_handler"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/log_config"
)

const (
	loglevel    = "info"
	envS3Bucket = "PDO_BUCKET"
)

var (
	commonHandler common_handler.CommonHandler

	crs4326 = map[string]interface{}{
		"properties": map[string]string{
			"name": "epsg:4326",
		},
		"type": "name",
	}

	outlineTypeFootPrint = pdwAttributes2{
		Value: "Footprint",
	}

	tags = map[string]interface{}{
		"appID":  "SD",
		"domain": "PDO", // have to confirm
	}
)

type sim2pdwInput struct {
	SimOutput  string `json:"simOuput"`
	WorkflowId string `json:"workflowId"`
	Address    string `json:"address"`
	ParcelId   string `json:"parcelId"`
}

type SimOutput struct {
	Lat       float64     `json:"lat"`
	Long      float64     `json:"lon"`
	Image     imageSource `json:"image"`
	Structure []structure `json:"structure"`
}

type imageSource struct {
	ImageURN     string `json:"image_urn"`
	ImageSetURN  string `json:"image_set_urn"`
	ShotDateTime string `json:"shot_date_time"`
	Source       string `json:"source"`
}

type structure struct {
	Type       string                 `json:"type"`
	SubType    string                 `json:"sub_type"`
	Confidence float64                `json:"confidence"`
	Geometry   geometry               `json:"geometry"`
	Primary    bool                   `json:"primary"`
	Details    map[string]interface{} `json:"details"`
}

type geometry struct {
	Type        string                 `json:"type"`
	Coordinates [][][]float64          `json:"coordinates"`
	CRS         map[string]interface{} `json:"crs"`
}

type PDWPayload struct {
	Asset      pdwAsset                 `json:"asset"`
	Version    string                   `json:"version"`
	Addresses  []string                 `json:"addresses"`
	Date       string                   `json:"date"`
	Source     pdwSource                `json:"source"`
	Imagery    pdwImagery               `json:"imagery"`
	Attributes map[string]pdwAttributes `json:"attrs"`
	Tags       map[string]interface{}   `json:"tags"`
}

type pdwAsset struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Type string  `json:"type"`
	Id   string  `json:"id,omitempty"`
}

type pdwSource struct {
	Type        string                 `json:"type"`
	DateCreated string                 `json:"dateCreated"`
	Meta        map[string]interface{} `json:"meta"`
}

type pdwImagery struct {
	Source  string                 `json:"source"`
	UrnList []pdwUrn               `json:"urnList"`
	Date    string                 `json:"date"`
	Meta    map[string]interface{} `json:"meta"`
}

type pdwUrn struct {
	Urn   string `json:"urn"`
	Index int    `json:"index"`
	Date  string `json:"date"`
}

type pdwAttributes struct {
	Value      interface{}               `json:"value"`
	Meta       map[string]interface{}    `json:"meta"`
	Attributes map[string]pdwAttributes2 `json:"attributes"`
}

type pdwAttributes2 struct {
	Value      interface{}            `json:"value"`
	Confidence float64                `json:"confidence"`
	Meta       map[string]interface{} `json:"meta"`
}

func handler(ctx context.Context, eventData sim2pdwInput) (map[string]interface{}, error) {
	resp := make(map[string]interface{})
	host, path, err := commonHandler.AwsClient.FetchS3BucketPath(eventData.SimOutput)
	if err != nil {
		log.Error(ctx, "Error in fetching AWS path: ", err.Error())
		return resp, error_handler.NewServiceError(error_codes.ErrorFetchingS3BucketPath, err.Error())
	}

	data, err := commonHandler.AwsClient.GetDataFromS3(ctx, host, path)
	if err != nil {
		return resp, err
	}
	output := SimOutput{}
	err = json.Unmarshal(data, &output)
	if err != nil {
		return resp, err
	}

	pdwPayload, err := sim2Pdw(ctx, &output, eventData.ParcelId, eventData.Address)
	if err != nil {
		return resp, err
	}

	data, err = json.Marshal(pdwPayload)
	if err != nil {
		return resp, err
	}

	s3Bucket := os.Getenv("PDO_S3_BUCKET")
	err = commonHandler.AwsClient.StoreDataToS3(ctx, s3Bucket, "/sim/pdw_payload.json", data)
	if err != nil {
		return resp, err
	}
	log.Info(context.Background(), "Successfull")
	// Upload to s3
	return map[string]interface{}{"status": "success", "payload": pdwPayload}, nil
}

func sim2Pdw(ctx context.Context, simOutput *SimOutput, parcelId, address string) ([]PDWPayload, error) {
	resp := []PDWPayload{}
	buildingCount := 0
	poolCount := 0
	trampolineCount := 0
	timeStamp := simOutput.Image.ShotDateTime + "T00:00:00.000000+00:00"

	for _, v := range simOutput.Structure {
		var payload PDWPayload

		payload.Addresses = append(payload.Addresses, address)
		payload.Date = timeStamp
		payload.Asset.Lat = simOutput.Lat
		payload.Asset.Lon = simOutput.Long
		payload.Version = "v3"
		payload.Imagery = pdwImagery{
			Source: simOutput.Image.Source,
			UrnList: []pdwUrn{
				{
					Urn:  simOutput.Image.ImageURN,
					Date: timeStamp,
				},
			},
			Date: timeStamp,
			Meta: map[string]interface{}{
				"imageSetUrn": simOutput.Image.ImageSetURN,
			},
		}
		payload.Tags = tags
		payload.Source = pdwSource{
			Type:        "ML",
			DateCreated: time.Now().String(),
		}

		v.Geometry.CRS = crs4326
		payload.Attributes = make(map[string]pdwAttributes)
		payload.Attributes["outline"] = pdwAttributes{
			Value: v.Geometry,
			Attributes: map[string]pdwAttributes2{
				"outlineType": outlineTypeFootPrint,
				"confidence":  {Value: v.Confidence},
			},
		}

		switch v.Type {
		case "building":
			if v.SubType != "building" {
				continue
			}
			payload.Asset.Type = "Structure"
			buildingCount += 1
			if v.Primary {
				payload.Attributes["type"] = pdwAttributes{
					Value: "main",
				}
			}

		case "trampoline":
			if v.SubType != "building" {
				continue
			}
			payload.Asset.Type = "Trampoline"
			trampolineCount += 1

		case "swimming pool":
			if v.SubType != "swimming pool" {
				continue
			}
			payload.Asset.Type = "Pool"
			poolCount += 1

		default:
			log.Info(ctx, "unsupported type: "+v.Type)
		}
		resp = append(resp, payload)
	}

	parcel := PDWPayload{
		Asset: pdwAsset{
			Type: "Parcel",
			Id:   parcelId,
		},
		Version:   "v3",
		Addresses: []string{address},
		Date:      timeStamp,
		Source: pdwSource{
			Type:        "ML",
			DateCreated: time.Now().String(),
		},
	}

	return resp, nil
}

func main() {
	log_config.InitLogging(loglevel)
	commonHandler = common_handler.New(true, false, false, false)
	handler(context.Background(), sim2pdwInput{
		SimOutput: "s3://platform-evml-address-pool/dr5r0f9/metas/ae0e9570-13bd-4c83-87b3-16bdd7c411d0_dr5r0f9-s1jxcn-fdw1qn_output.json",
		Address:   "12 Houstan TX Bangalore",
		ParcelId:  "9172490-oiahsvlk-918yhkljh-alkjsdq4r",
	})
	lambda.Start(handler)
}
