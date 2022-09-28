package error_codes

const (
	// AWS Client Errors
	ErrorWhileClosingWaitTaskInSFN              = 4001
	ErrorFetchingSecretsFromSecretManager       = 4002
	ErrorFetchingS3BucketPath                   = 4003
	ErrorStoringDataToS3                        = 4004
	ErrorInvokingLambdaLegacyUpdateLambda       = 4005
	ErrorDecodingLambdaOutput                   = 4006
	ErrorInvokingCalloutLambdaFromEVMLConverter = 4007
	RetriableCallOutHTTPError                   = 4008
	LambdaExecutionError                        = 4009
	ErrorInvokingStepFunction                   = 4010
	ErrorInvokingLambda                         = 4041
	ErrorFetchingDataFromS3                     = 4042
	ErrorPushingDataToSQS                       = 4050

	// DocumentDB Errors
	ErrorFetchingStepExecutionDataFromDB     = 4011
	ErrorFetchingWorkflowExecutionDataFromDB = 4012
	ErrorUpdatingStepsDataInDB               = 4013
	ErrorUpdatingWorkflowDataInDB            = 4014
	ErrorInsertingStepExecutionDataInDB      = 4015
	ErrorInsertingWorkflowDataInDB           = 4016
	ErrorFetchingHipsterCountFromDB          = 4017

	//Service Errors
	ErrorWhileUpdatingLegacy                    = 4018
	StatusNotFoundInLegacyUpdateResponse        = 4019
	LegacyStatusFailed                          = 4020
	UnsupportedRequestMethodCallOutLambda       = 4021
	ErrorDecodingHipsterOutput                  = 4022
	JobIDMissingInHipsterOutput                 = 4023
	StepFunctionTaskTimedOut                    = 4024
	TaskRecordNotFoundInFailureTaskOutputMap    = 4025
	ErrorParsingLegacyAuthToken                 = 4026
	ErrorConvertingAllowedHipsterCountToInteger = 4027

	// Validation Errors
	ErrorValidatingCallBackLambdaRequest          = 4028
	ErrorValidatingCallOutLambdaRequest           = 4029
	ErrorSerializingCallOutPayload                = 4030
	ErrorDecodingCallOutResponse                  = 4031
	PropertyModelLocationMissingInTaskOutput      = 4032
	InvalidTypeForPropertyModelLocation           = 4033
	ErrorDecodingInvokeSFNInput                   = 4034
	ErrorValidatingInvokeSFNInput                 = 4035
	ErrorEvossObjectIdMissingInEVMLUploadResponse = 4048
	ErrorSerializingS3Data                        = 4049
	// HTTP Errors
	ReceivedInternalServerErrorInCallout      = 4036
	ReceivedInvalidHTTPStatusCodeInCallout    = 4037
	ErrorWhileFetchingAuthToken               = 4038
	ErrorUnableToDecodeAuthServiceResponse    = 4039
	ErrorUnSuccessfullResponseFromAuthService = 4040
	ReceivedInternalServerError               = 4051
	ReceivedInvalidHTTPStatusCode             = 4052
	ErrorDecodingServiceResponse              = 4053

	ErrorParsingURLCalloutLambda   = 4041
	ErrorMakingGetCall             = 4042
	ErrorMakingPostPutOrDeleteCall = 4043

	ErrorWhileUploadImageToEVOSS       = 4044
	ErrorWhileUploadImageMetaDataEVOSS = 4045
	ErrorWhileMarshlingData            = 4046
	ErrorValidationCheck               = 4047
	ErrorMissingS3Path                 = 4048

	// PDW Errors
	ParcelIDDoesnotExist           = 4054
	ErrorReadingQueryFile          = 4055
	ErrorQueryingPDWAfterIngestion = 4056
	ErrorGettingAccessToken        = 4057
	Success                        = 4058

	ErrorUnmarshallingSimOutput   = 4061
	ErrorTransformingSim2PDW      = 4062
	ErrorValidatingSim2PDWRequest = 4063
	ErrorSentToCallbackLambda     = 4064
	TaskTimedOutError             = 4065
	ErrorRetrievingMsgCode        = 4066
	ErrorUnknownSource            = 4067
	ErrorFromGeocodingService     = 4068
)

// Messagecodes map for async tasks from callback range 4080-4100
var AsyncTaskMsgCodeMap = map[string]interface{}{
	"InvokeGraphPublisher":                  4080,
	"InvokeSIMModel":                        4081,
	"ImageryCheck":                          4082,
	"BuildingDetection":                     4083,
	"ImageSelection":                        4084,
	"FacetKeyPointDetection":                4085,
	"3DModellingService":                    4086,
	"CreateHipsterJobAndWaitForMeasurement": 4087,
	"UpdateHipsterJobAndWaitForQC":          4088,
	"ConvertPropertyModelToEVJson":          4089,
}
