package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
	"github.eagleview.com/engineering/symphony-service/commons/mocks"
)

const sampleSimOutput = `{
    "lat": 40.6198009,
    "lon": -74.144398,
    "S3Uri": "",
    "ParcelGeometryUri": "",
    "image": {
        "image_urn": "urn:eagleview.com:v1:spatial-data:raster:singleframe:ae0e9570-13bd-4c83-87b3-16bdd7c411d0",
        "image_set_urn": "urn:eagleview.com:v1:spatial-data:raster:imageset:bf49bd70-f2c6-5c11-8da9-68a84debb0c4",
        "shot_date_time": "2019-04-29",
        "UL": [
            40.619918,
            -74.144615
        ],
        "RL": [
            40.619664,
            -74.144216
        ],
        "GSD": 0.1027,
        "source": "gserve",
        "MaskedImageUri": ""
    },
    "structure": [
        {
            "type": "building",
            "sub_type": "building",
            "confidence": 0.595,
            "pixel_box": [
                74,
                66,
                128,
                105
            ],
            "centroid": {
                "longitude": -121.5663409,
                "latitude": 37.0097997
            },
            "geometry": {
                "type": "Polygon",
                "coordinates": [
                    [
                        [
                            -74.1445128,
                            40.619855
                        ],
                        [
                            -74.144525,
                            40.6197734
                        ],
                        [
                            -74.1444082,
                            40.6197604
                        ],
                        [
                            -74.1443705,
                            40.6197632
                        ],
                        [
                            -74.1443717,
                            40.6198401
                        ],
                        [
                            -74.1444447,
                            40.6198457
                        ],
                        [
                            -74.1444775,
                            40.6198568
                        ],
                        [
                            -74.1444836,
                            40.6198503
                        ],
                        [
                            -74.1445128,
                            40.619855
                        ]
                    ]
                ]
            },
            "primary": true,
            "details": {}
        },
        {
            "type": "building",
            "sub_type": "barn",
            "confidence": 0.4352,
            "pixel_box": [
                238,
                41,
                83,
                90
            ],
            "centroid": {
                "longitude": -121.5663409,
                "latitude": 37.0097997
            },
            "geometry": {
                "type": "Polygon",
                "coordinates": [
                    [
                        [
                            -74.144317,
                            40.61988
                        ],
                        [
                            -74.1443255,
                            40.6198114
                        ],
                        [
                            -74.1442513,
                            40.619804
                        ],
                        [
                            -74.1442477,
                            40.619817
                        ],
                        [
                            -74.1442416,
                            40.6198142
                        ],
                        [
                            -74.1442464,
                            40.6197975
                        ],
                        [
                            -74.1442306,
                            40.6197975
                        ],
                        [
                            -74.1442258,
                            40.6198142
                        ],
                        [
                            -74.1442477,
                            40.6198188
                        ],
                        [
                            -74.1442416,
                            40.6198735
                        ],
                        [
                            -74.144317,
                            40.61988
                        ]
                    ]
                ]
            },
            "primary": false,
            "details": {}
        },
        {
            "type": "swimming pool",
            "sub_type": "swimming pool",
            "confidence": 0.4513,
            "pixel_box": [
                232,
                129,
                50,
                79
            ],
            "centroid": {
                "longitude": -121.5663409,
                "latitude": 37.0097997
            },
            "geometry": {
                "type": "Polygon",
                "coordinates": [
                    [
                        [
                            -74.1443194,
                            40.6197984
                        ],
                        [
                            -74.1443328,
                            40.6197929
                        ],
                        [
                            -74.1443316,
                            40.6197891
                        ],
                        [
                            -74.1443267,
                            40.619791
                        ],
                        [
                            -74.1443243,
                            40.6197808
                        ],
                        [
                            -74.1443304,
                            40.6197317
                        ],
                        [
                            -74.1442866,
                            40.6197261
                        ],
                        [
                            -74.1442829,
                            40.6197289
                        ],
                        [
                            -74.1442732,
                            40.6197929
                        ],
                        [
                            -74.1442829,
                            40.6197966
                        ],
                        [
                            -74.1443194,
                            40.6197984
                        ]
                    ]
                ]
            },
            "primary": false,
            "details": {}
        },
        {
            "type": "trampoline",
            "sub_type": "trampoline",
            "confidence": 0.5008,
            "pixel_box": [
                74,
                157,
                38,
                40
            ],
            "centroid": {
                "longitude": -121.5663409,
                "latitude": 37.0097997
            },
            "geometry": {
                "type": "Polygon",
                "coordinates": [
                    [
                        [
                            -74.1445177,
                            40.6197725
                        ],
                        [
                            -74.1445225,
                            40.6197697
                        ],
                        [
                            -74.144525,
                            40.6197586
                        ],
                        [
                            -74.1445238,
                            40.6197382
                        ],
                        [
                            -74.1445128,
                            40.6197363
                        ],
                        [
                            -74.1444958,
                            40.6197382
                        ],
                        [
                            -74.1444897,
                            40.6197502
                        ],
                        [
                            -74.1444848,
                            40.6197493
                        ],
                        [
                            -74.14448,
                            40.6197511
                        ],
                        [
                            -74.1444824,
                            40.6197567
                        ],
                        [
                            -74.1444873,
                            40.6197586
                        ],
                        [
                            -74.1444861,
                            40.6197623
                        ],
                        [
                            -74.1444824,
                            40.6197632
                        ],
                        [
                            -74.1444873,
                            40.6197641
                        ],
                        [
                            -74.1444921,
                            40.6197706
                        ],
                        [
                            -74.1445177,
                            40.6197725
                        ]
                    ]
                ]
            },
            "details": {}
        }
    ],
    "model": {
        "name": "structure-identifier",
        "arch": "HRNet OCR with AUGS",
        "version": "1.0",
        "docker_image": "",
        "input_size": 520
    }
}`

func TestSim2Pdw(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.On("FetchS3BucketPath", "s3path").Return("bucket", "path", nil)
	awsClient.On("GetDataFromS3", context.Background(), "bucket", "path").Return([]byte(sampleSimOutput), nil)
	awsClient.On("StoreDataToS3", context.Background(), "", "/sim/1/pdw_payload.json", mock.Anything).Return(nil)
	commonHandler.AwsClient = awsClient

	resp, err := notificationWrapper(context.Background(), sim2pdwInput{SimOutput: "s3path", WorkflowId: "1", Address: "some address", ParcelId: "some id"})
	assert.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
	assert.Equal(t, "s3:///sim/1/pdw_payload.json", resp["pdwPayload"])
}

func TestSim2PdwWrongData(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.On("FetchS3BucketPath", "s3path").Return("bucket", "path", nil)
	awsClient.On("GetDataFromS3", context.Background(), "bucket", "path").Return([]byte(`{"lat": "wrong type"}`), nil)
	commonHandler.AwsClient = awsClient

	resp, err := handler(context.Background(), sim2pdwInput{SimOutput: "s3path", WorkflowId: "1", Address: "some address", ParcelId: "some id"})
	assert.Error(t, err)
	assert.Equal(t, "failure", resp["status"])
}

func TestSim2PdwWrongInput(t *testing.T) {
	awsClient := new(mocks.IAWSClient)
	awsClient.On("FetchS3BucketPath", "s3path").Return("bucket", "path", nil)
	awsClient.On("GetDataFromS3", context.Background(), "bucket", "path").Return([]byte(`{"lat": "wrong type"}`), nil)
	commonHandler.AwsClient = awsClient

	resp, err := handler(context.Background(), sim2pdwInput{SimOutput: "s3path", WorkflowId: "1"})
	assert.Error(t, err)
	err2 := err.(error_handler.ICodedError)
	assert.Equal(t, error_codes.ErrorValidatingSim2PDWRequest, err2.GetErrorCode())
	assert.Equal(t, "failure", resp["status"])
}
