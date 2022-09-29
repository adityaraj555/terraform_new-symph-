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
        "details": {
          "facets": [
                    {
                        "bbox": [
                            50,
                            317,
                            150,
                            135
                        ],
                        "confindence": 0.3034,
                        "geometry": {
                            "type": "Polygon",
                            "coordinates": [
                                [
                                    [
                                        -98.55593999999999,
                                        29.467476612954187
                                    ],
                                    [
                                        -98.55594709433961,
                                        29.467437908372826
                                    ],
                                    [
                                        -98.55594886792453,
                                        29.467447197472353
                                    ],
                                    [
                                        -98.55598345283019,
                                        29.467444875197472
                                    ],
                                    [
                                        -98.55598966037735,
                                        29.467403074249603
                                    ],
                                    [
                                        -98.55598079245283,
                                        29.46739842969984
                                    ],
                                    [
                                        -98.55598256603773,
                                        29.467382947867296
                                    ],
                                    [
                                        -98.55588679245282,
                                        29.467372884676145
                                    ],
                                    [
                                        -98.5558796981132,
                                        29.46739068878357
                                    ],
                                    [
                                        -98.55586994339622,
                                        29.467394559241704
                                    ],
                                    [
                                        -98.55587881132075,
                                        29.46740384834123
                                    ],
                                    [
                                        -98.55586462264151,
                                        29.467407718799368
                                    ],
                                    [
                                        -98.55585752830189,
                                        29.46746732385466
                                    ],
                                    [
                                        -98.55593999999999,
                                        29.467476612954187
                                    ]
                                ]
                            ]
                        },
                        "centroid": {
                            "longitude": -98.5559208,
                            "latitude": 29.4674233
                        }
                    },
                    {
                        "bbox": [
                            49,
                            353,
                            53,
                            65
                        ],
                        "confindence": 0.353,
                        "geometry": {
                            "type": "Polygon",
                            "coordinates": [
                                [
                                    [
                                        -98.55594443396225,
                                        29.467447197472353
                                    ],
                                    [
                                        -98.55598345283019,
                                        29.46744797156398
                                    ],
                                    [
                                        -98.55598966037735,
                                        29.46742087835703
                                    ],
                                    [
                                        -98.55599054716981,
                                        29.467405396524484
                                    ],
                                    [
                                        -98.555987,
                                        29.467399977883094
                                    ],
                                    [
                                        -98.55598079245283,
                                        29.46739920379147
                                    ],
                                    [
                                        -98.55597901886792,
                                        29.46740384834123
                                    ],
                                    [
                                        -98.55594620754717,
                                        29.467430941548184
                                    ],
                                    [
                                        -98.55594443396225,
                                        29.467447197472353
                                    ]
                                ]
                            ]
                        },
                        "centroid": {
                            "longitude": -98.5559705,
                            "latitude": 29.4674285
                        }
                    },
                    {
                        "bbox": [
                            101,
                            317,
                            100,
                            92
                        ],
                        "confindence": 0.576,
                        "geometry": {
                            "type": "Polygon",
                            "coordinates": [
                                [
                                    [
                                        -98.55593911320754,
                                        29.467476612954187
                                    ],
                                    [
                                        -98.55594443396225,
                                        29.467430167456556
                                    ],
                                    [
                                        -98.55588058490565,
                                        29.467421652448657
                                    ],
                                    [
                                        -98.55587526415094,
                                        29.467416233807267
                                    ],
                                    [
                                        -98.5558779245283,
                                        29.4674115892575
                                    ],
                                    [
                                        -98.55586816981132,
                                        29.46740694470774
                                    ],
                                    [
                                        -98.55587171698113,
                                        29.467418556082148
                                    ],
                                    [
                                        -98.55586462264151,
                                        29.4674201042654
                                    ],
                                    [
                                        -98.55585664150942,
                                        29.46746732385466
                                    ],
                                    [
                                        -98.55593911320754,
                                        29.467476612954187
                                    ]
                                ]
                            ]
                        },
                        "centroid": {
                            "longitude": -98.5559004,
                            "latitude": 29.4674478
                        }
                    },
                    {
                        "bbox": [
                            48,
                            415,
                            14,
                            44
                        ],
                        "confindence": 0.5795,
                        "geometry": {
                            "type": "Polygon",
                            "coordinates": [
                                [
                                    [
                                        -98.55598167924528,
                                        29.467400751974722
                                    ],
                                    [
                                        -98.55598788679245,
                                        29.467399977883094
                                    ],
                                    [
                                        -98.55598966037735,
                                        29.46739842969984
                                    ],
                                    [
                                        -98.55599143396226,
                                        29.46738527014218
                                    ],
                                    [
                                        -98.55599143396226,
                                        29.467369788309636
                                    ],
                                    [
                                        -98.55598966037735,
                                        29.46736824012638
                                    ],
                                    [
                                        -98.55598522641509,
                                        29.467367466034755
                                    ],
                                    [
                                        -98.55598256603773,
                                        29.46738759241706
                                    ],
                                    [
                                        -98.55597990566038,
                                        29.467395333333332
                                    ],
                                    [
                                        -98.55597990566038,
                                        29.467399977883094
                                    ],
                                    [
                                        -98.55598167924528,
                                        29.467400751974722
                                    ]
                                ]
                            ]
                        },
                        "centroid": {
                            "longitude": -98.5559857,
                            "latitude": 29.4673963
                        }
                    },
                    {
                        "bbox": [
                            173,
                            408,
                            18,
                            26
                        ],
                        "confindence": 0.6385,
                        "geometry": {
                            "type": "Polygon",
                            "coordinates": [
                                [
                                    [
                                        -98.5558779245283,
                                        29.467406170616112
                                    ],
                                    [
                                        -98.5558796981132,
                                        29.46740152606635
                                    ],
                                    [
                                        -98.55587881132075,
                                        29.467400751974722
                                    ],
                                    [
                                        -98.55587881132075,
                                        29.467396881516585
                                    ],
                                    [
                                        -98.55588058490565,
                                        29.467392236966823
                                    ],
                                    [
                                        -98.5558796981132,
                                        29.467388366508686
                                    ],
                                    [
                                        -98.55587881132075,
                                        29.46738759241706
                                    ],
                                    [
                                        -98.55586728301886,
                                        29.467386818325433
                                    ],
                                    [
                                        -98.5558663962264,
                                        29.46738759241706
                                    ],
                                    [
                                        -98.55586550943396,
                                        29.46740462243286
                                    ],
                                    [
                                        -98.55587171698113,
                                        29.46740462243286
                                    ],
                                    [
                                        -98.5558779245283,
                                        29.467406170616112
                                    ]
                                ]
                            ]
                        },
                        "centroid": {
                            "longitude": -98.555873,
                            "latitude": 29.4673944
                        }
                    },
                    {
                        "bbox": [
                            58,
                            376,
                            135,
                            77
                        ],
                        "confindence": 0.6958,
                        "geometry": {
                            "type": "Polygon",
                            "coordinates": [
                                [
                                    [
                                        -98.55598256603773,
                                        29.46741004107425
                                    ],
                                    [
                                        -98.55598079245283,
                                        29.467382173775672
                                    ],
                                    [
                                        -98.55588501886793,
                                        29.467372110584517
                                    ],
                                    [
                                        -98.5558796981132,
                                        29.467391462875195
                                    ],
                                    [
                                        -98.55586905660377,
                                        29.467395333333332
                                    ],
                                    [
                                        -98.55587881132075,
                                        29.46740462243286
                                    ],
                                    [
                                        -98.55586550943396,
                                        29.46740462243286
                                    ],
                                    [
                                        -98.55586373584904,
                                        29.467424748815166
                                    ],
                                    [
                                        -98.55586550943396,
                                        29.4674286192733
                                    ],
                                    [
                                        -98.55588147169811,
                                        29.467423974723538
                                    ],
                                    [
                                        -98.55594620754717,
                                        29.467430941548184
                                    ],
                                    [
                                        -98.55598256603773,
                                        29.46741004107425
                                    ]
                                ]
                            ]
                        },
                        "centroid": {
                            "longitude": -98.5559235,
                            "latitude": 29.4674023
                        }
                    },
                    {
                        "bbox": [
                            55,
                            440,
                            115,
                            74
                        ],
                        "confindence": 0.8428,
                        "geometry": {
                            "type": "Polygon",
                            "coordinates": [
                                [
                                    [
                                        -98.55598167924528,
                                        29.467381399684044
                                    ],
                                    [
                                        -98.55598433962264,
                                        29.467334954186413
                                    ],
                                    [
                                        -98.55597458490566,
                                        29.46733882464455
                                    ],
                                    [
                                        -98.55593999999999,
                                        29.467337276461294
                                    ],
                                    [
                                        -98.55592403773585,
                                        29.467328761453395
                                    ],
                                    [
                                        -98.55588679245282,
                                        29.467325665086886
                                    ],
                                    [
                                        -98.55588413207546,
                                        29.467370562401264
                                    ],
                                    [
                                        -98.55598167924528,
                                        29.467381399684044
                                    ]
                                ]
                            ]
                        },
                        "centroid": {
                            "longitude": -98.5559332,
                            "latitude": 29.4673543
                        }
                    }
                ]
        }
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
      },
      {
        "type": "extension",
        "sub_type": "deck",
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
	awsClient.On("StoreDataToS3", context.Background(), "", "/sim-pipeline/1/sim2pdw/pdw_payload.json", mock.Anything).Return(nil)
	commonHandler.AwsClient = awsClient

	resp, err := notificationWrapper(context.Background(), sim2pdwInput{SimOutput: "s3path", WorkflowId: "1", Address: "some address", ParcelId: "some id"})
	assert.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
	assert.Equal(t, "s3:///sim-pipeline/1/sim2pdw/pdw_payload.json", resp["pdwPayload"])
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
