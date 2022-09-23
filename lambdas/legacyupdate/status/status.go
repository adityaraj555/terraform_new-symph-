package status

type status struct {
	Status    string
	SubStatus string
}

type failedTaskMetaData struct {
	StatusKey        string
	FallbackTaskName string
}

func New() *status {
	return new(status)
}

var StatusMap = map[string]status{
	"MAStarted": {
		Status:    "InProcess",
		SubStatus: "MLAutomationStarted",
	},
	"MAFailed": {
		Status:    "InProcess",
		SubStatus: "MLAutomationRejected",
	},
	"MACompleted": {
		Status:    "InProcess",
		SubStatus: "MLAutomationCompleted",
	},
	"MASymphonyCompleted": {
		Status:    "InProcess",
		SubStatus: "MLSFNAutomationCompleted",
	},
	"MeasurementStarted": {
		Status:    "InProcess",
		SubStatus: "HipsterMeasurementPending",
	},
	"MeasurementFailed": {
		Status:    "InProcess",
		SubStatus: "HipsterMeasurementRejected",
	},
	"MeasurementCompleted": {
		Status:    "InProcess",
		SubStatus: "HipsterMeasurementCompleted",
	},
	"QCStarted": {
		Status:    "InProcess",
		SubStatus: "HipsterQCPending",
	},
	"QCFailed": {
		Status:    "InProcess",
		SubStatus: "HipsterQCRejected",
	},
	"QCCompleted": {
		Status:    "InProcess",
		SubStatus: "HipsterQCCompleted",
	},
	"AISCompleted": {
		Status:    "InProcess",
		SubStatus: "AutoImageSelectionCompleted",
	},
	"AISFailed": {
		Status:    "InProcess",
		SubStatus: "AutoImageSelectionFailed",
	},
}

var FailedTaskStatusMap = map[string]failedTaskMetaData{
	"CreateHipsterJobAndWaitForMeasurement": {
		StatusKey:        "MeasurementFailed",
		FallbackTaskName: "3DModellingService",
	},
	"UpdateHipsterJobAndWaitForQC": {
		StatusKey:        "QCFailed",
		FallbackTaskName: "CreateHipsterJobAndWaitForMeasurement",
	},
	"UpdateHipsterJobAndWaitForMeasurement": {
		StatusKey:        "MeasurementFailed",
		FallbackTaskName: "3DModellingService",
	},
	"UpdateHipsterMeasurementCompleteInLegacy": {
		StatusKey:        "MeasurementFailed",
		FallbackTaskName: "3DModellingService",
	},
	"CheckIsMultiStructure": {
		StatusKey:        "MACompleted",
		FallbackTaskName: "3DModellingService",
	},
}
