package status

type status struct {
	Status    string
	SubStatus string
}

type failedTaskMetaData struct {
	Status           status
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
}

var FailedTaskStatusMap = map[string]failedTaskMetaData{
	"CreateHipsterJobAndWaitForMeasurement": {
		Status: status{
			Status:    "InProcess",
			SubStatus: "HipsterMeasurementRejected",
		},
		FallbackTaskName: "3DModellingService",
	},
	"UpdateHipsterJobAndWaitForQC": {
		Status: status{
			Status:    "InProcess",
			SubStatus: "HipsterQCRejected",
		},
		FallbackTaskName: "CreateHipsterJobAndWaitForMeasurement",
	},
	"UpdateHipsterJobAndWaitForMeasurement": {
		Status: status{
			Status:    "InProcess",
			SubStatus: "HipsterMeasurementRejected",
		},
		FallbackTaskName: "3DModellingService",
	},
}
