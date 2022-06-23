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

	// DocumentDB Errors
	ErrorFetchingStepExecutionDataFromDB     = 4010
	ErrorFetchingWorkflowExecutionDataFromDB = 4011
	ErrorUpdatingStepsDataInDB               = 4012
	ErrorUpdatingWorkflowDataInDB            = 4013
	ErrorInsertingStepExecutionDataInDB      = 4014
	ErrorInsertingWorkflowDataInDB           = 4015
	ErrorFetchingHipsterCountFromDB          = 4016

	//Service Errors
	ErrorWhileUpdatingLegacy                    = 4017
	StatusNotFoundInLegacyUpdateResponse        = 4018
	LegacyStatusFailed                          = 4019
	UnsupportedRequestMethodCallOutLambda       = 4020
	ErrorDecodingHipsterOutput                  = 4021
	JobIDMissingInHipsterOutput                 = 4022
	StepFunctionTaskTimedOut                    = 4023
	TaskRecordNotFoundInFailureTaskOutputMap    = 4024
	ErrorParsingLegacyAuthToken                 = 4025
	ErrorConvertingAllowedHipsterCountToInteger = 4026

	// Validation Errors
	ErrorValidatingCallBackLambdaRequest     = 4027
	ErrorValidatingCallOutLambdaRequest      = 4028
	ErrorSerializingCallOutPayload           = 4029
	ErrorDecodingCallOutResponse             = 4030
	PropertyModelLocationMissingInTaskOutput = 4031
	InvalidTypeForPropertyModelLocation      = 4032
	ErrorDecodingInvokeSFNInput              = 4033
	ErrorValidatingInvokeSFNInput            = 4034

	// HTTP Errors
	ReceivedInternalServerErrorInCallout      = 4035
	ReceivedInvalidHTTPStatusCodeInCallout    = 4036
	ErrorWhileFetchingAuthToken               = 4037
	ErrorUnableToDecodeAuthServiceResponse    = 4038
	ErrorUnSuccessfullResponseFromAuthService = 4039
)
