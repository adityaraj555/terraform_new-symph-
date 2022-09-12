package documentDB_client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.eagleview.com/engineering/assess-platform-library/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CaFilePath                    = "/rds-combined-ca-bundle.pem"
	ConnectTimeout                = 5
	QueryTimeout                  = 30
	ConnectionStringTemplate      = "mongodb://%s:%s@%s/%s?replicaSet=rs0&readpreference=%s"
	Database                      = "test"
	WorkflowDataCollection        = "WorkflowData"
	StepsDataCollection           = "StepsData"
	success                       = "success"
	failure                       = "failure"
	Submitted                     = "submitted"
	running                       = "running"
	UpdateStepExecution           = "UpdateStepExecution"
	UpdateWorkflowExecutionSteps  = "UpdateWorkflowExecutionSteps"
	UpdateWorkflowExecutionStatus = "UpdateWorkflowExecutionStatus"
	PSTTimeZone                   = "America/Los_Angeles"
)

var (
	Username        string
	Password        string
	ClusterEndpoint string
	ReadPreference  string
)

type IDocDBClient interface {
	FetchStepExecutionData(ctx context.Context, StepId string) (StepExecutionDataBody, error)
	InsertStepExecutionData(ctx context.Context, StepExecutionData StepExecutionDataBody) error
	InsertWorkflowExecutionData(ctx context.Context, Data WorkflowExecutionDataBody) error
	UpdateDocumentDB(ctx context.Context, query, update interface{}, collectionName string) error
	FetchWorkflowExecutionData(ctx context.Context, workFlowId string) (WorkflowExecutionDataBody, error)
	BuildQueryForCallBack(ctx context.Context, event, status, workflowID, stepID, TaskName string, callbackResponse map[string]interface{}) (interface{}, interface{})
	BuildQueryForUpdateWorkflowDataCallout(ctx context.Context, TaskName, stepID, status string, starttime int64, IsWaitTask bool) interface{}
	CheckConnection(ctx context.Context) error
	GetHipsterCountPerDay(ctx context.Context) (int64, error)
	GetTimedoutTask(ctx context.Context, WorkflowId string) string
}

type DocDBClient struct {
	DBClient *mongo.Client
}

type WorkflowExecutionDataBody struct {
	WorkflowId         string                   `bson:"_id"`
	Status             string                   `bson:"status"`
	OrderId            string                   `bson:"orderId"`
	FlowType           string                   `bson:"flowType"`
	UpdatedAt          int64                    `bson:"updatedAt"`
	CreatedAt          int64                    `bson:"createdAt"`
	FinishedAt         int64                    `bson:"finishedAt"`
	RunningState       map[string]interface{}   `bson:"runningState"`
	InitialInput       map[string]interface{}   `bson:"initialInput"`
	FinalOutput        map[string]interface{}   `bson:"finalOutput"`
	StepsPassedThrough []StepsPassedThroughBody `bson:"stepsPassedThrough"`
}

type StepExecutionDataBody struct {
	StepId             string                 `bson:"_id"`
	StartTime          int64                  `bson:"startTime"`
	EndTime            int64                  `bson:"endTime"`
	Url                string                 `bson:"url"`
	Input              interface{}            `bson:"input"`
	Output             map[string]interface{} `bson:"output"`
	IntermediateOutput map[string]interface{} `bson:"intermediateOutput"`
	Status             string                 `bson:"status"`
	TaskToken          string                 `bson:"taskToken"`
	WorkflowId         string                 `bson:"workflowId"`
	TaskName           string                 `bson:"taskName"`
	ReportId           string                 `bson:"reportId"`
}

type StepsPassedThroughBody struct {
	TaskName  string `bson:"taskName"`
	StepId    string `bson:"stepId"`
	StartTime int64  `bson:"startTime"`
	Status    string `bson:"status"`
}

func NewDBClientService(secrets map[string]interface{}) *DocDBClient {
	Username = secrets["username"].(string)
	Password = secrets["password"].(string)
	ClusterEndpoint = fmt.Sprintf("%v:%v", secrets["host"], secrets["port"])
	connectionURI := fmt.Sprintf(ConnectionStringTemplate, Username, Password, ClusterEndpoint, Database, ReadPreference)
	tlsConfig, err := getCustomTLSConfig(CaFilePath)
	if err != nil {
		log.Errorf(context.Background(), "Failed getting TLS configuration: %v", err)
	}
	DBClient, err := mongo.NewClient(options.Client().ApplyURI(connectionURI).SetTLSConfig(tlsConfig))
	if err != nil {
		log.Errorf(context.Background(), "Error= %v", err)
	}
	return &DocDBClient{DBClient: DBClient}
}

func (DBClient *DocDBClient) CheckConnection(ctx context.Context) error {
	return DBClient.DBClient.Connect(ctx)
}
func (DBClient *DocDBClient) FetchStepExecutionData(ctx context.Context, StepId string) (StepExecutionDataBody, error) {
	collection := DBClient.DBClient.Database(Database).Collection(StepsDataCollection)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout*time.Second)
	defer cancel()
	var StepExecutionData StepExecutionDataBody
	err := collection.FindOne(ctx, bson.M{"_id": StepId}).Decode(&StepExecutionData)
	if err != nil {
		log.Errorf(ctx, "Failed to run find query: %v", err)
		return StepExecutionDataBody{}, err
	}
	return StepExecutionData, nil
}
func (DBClient *DocDBClient) InsertStepExecutionData(ctx context.Context, StepExecutionData StepExecutionDataBody) error {
	collection := DBClient.DBClient.Database(Database).Collection(StepsDataCollection)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout*time.Second)
	defer cancel()
	res, err := collection.InsertOne(ctx, StepExecutionData)
	if err != nil {
		log.Errorf(ctx, "Failed to insert document: %v", err)
		return err
	}
	id := res.InsertedID
	log.Infof(ctx, "Inserted document ID: %s", id)
	return nil
}
func (DBClient *DocDBClient) InsertWorkflowExecutionData(ctx context.Context, Data WorkflowExecutionDataBody) error {
	collection := DBClient.DBClient.Database(Database).Collection(WorkflowDataCollection)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout*time.Second)
	defer cancel()
	res, err := collection.InsertOne(ctx, Data)
	if err != nil {
		log.Errorf(ctx, "Failed to insert document: %v", err)
		return err
	}
	id := res.InsertedID
	log.Infof(ctx, "Inserted document ID: %s", id)
	return nil
}
func (DBClient *DocDBClient) UpdateDocumentDB(ctx context.Context, query, update interface{}, collectionName string) error {
	collection := DBClient.DBClient.Database(Database).Collection(collectionName)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout*time.Second)
	defer cancel()

	res, err := collection.UpdateMany(ctx, query, update)

	if err != nil {
		log.Errorf(ctx, "Failed to update document: %v", err)
		return err
	}
	if res.MatchedCount == 0 {
		log.Errorf(ctx, "Unable to update document as no such document exist")
	}
	log.Infof(ctx, "Updated document ID: %s", res.UpsertedID)
	return nil
}
func (DBClient *DocDBClient) FetchWorkflowExecutionData(ctx context.Context, workFlowId string) (WorkflowExecutionDataBody, error) {
	collection := DBClient.DBClient.Database(Database).Collection(WorkflowDataCollection)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout*time.Second)
	defer cancel()
	var WorkflowExecutionData WorkflowExecutionDataBody
	err := collection.FindOne(ctx, bson.M{"_id": workFlowId}).Decode(&WorkflowExecutionData)
	if err != nil {
		log.Errorf(ctx, "Failed to run find query: %v", err)
		return WorkflowExecutionDataBody{}, err
	}
	return WorkflowExecutionData, nil
}

func (DBClient *DocDBClient) BuildQueryForUpdateWorkflowDataCallout(ctx context.Context, TaskName, stepID, status string, starttime int64, IsWaitTask bool) interface{} {
	var setrecord interface{}
	var stepstatus string = failure
	updatedAt := time.Now().Unix()
	if IsWaitTask && status == success {
		stepstatus = running
		setrecord = bson.M{
			"updatedAt": updatedAt,
			"runningState": bson.M{
				TaskName: Submitted,
			},
		}
	} else {
		if !IsWaitTask && status == success {
			stepstatus = success
		}
		setrecord = bson.M{
			"updatedAt": updatedAt,
		}
	}
	return bson.M{
		"$push": bson.M{
			"stepsPassedThrough": StepsPassedThroughBody{
				TaskName:  TaskName,
				StepId:    stepID,
				StartTime: starttime,
				Status:    stepstatus,
			},
		},
		"$set": setrecord,
	}
}
func (DBClient *DocDBClient) BuildQueryForCallBack(ctx context.Context, event, status, workflowID, stepID, TaskName string, callbackResponse map[string]interface{}) (interface{}, interface{}) {
	var filter interface{}
	var query interface{}
	if event == UpdateStepExecution {
		filter = bson.M{
			"_id": stepID,
		}
		query = bson.M{
			"$set": bson.M{
				"output":  callbackResponse,
				"status":  status,
				"endTime": time.Now().Unix(),
			},
		}
	} else if event == UpdateWorkflowExecutionSteps {
		filter = bson.M{
			"_id":                       workflowID,
			"stepsPassedThrough.stepId": stepID,
		}

		query = bson.M{
			"$set": bson.M{
				"stepsPassedThrough.$.status": status,
			},
		}
	} else if event == UpdateWorkflowExecutionStatus {
		filter = bson.M{
			"_id": workflowID,
		}
		query = bson.M{
			"$set": bson.M{
				"updatedAt": time.Now().Unix(),
				"runningState": bson.M{
					TaskName: status,
				},
			},
		}
	}
	return filter, query
}
func getCustomTLSConfig(caFile string) (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	certs, err := ioutil.ReadFile(caFile)
	if err != nil {
		return tlsConfig, err
	}
	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(certs)
	if !ok {
		return tlsConfig, errors.New("Failed parsing pem file")
	}
	return tlsConfig, nil
}

func (DBClient *DocDBClient) GetHipsterCountPerDay(ctx context.Context) (int64, error) {
	collection := DBClient.DBClient.Database(Database).Collection(WorkflowDataCollection)

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout*time.Second)
	defer cancel()
	endedTime := time.Now().Unix()
	loc, err := time.LoadLocation(PSTTimeZone)
	if err != nil {
		log.Errorf(ctx, "Failed to load time location: %v", err)
		return 0, err
	}
	y, m, d := (time.Now().In(loc).Date())
	pst_midnight := time.Date(y, m, d, 0, 0, 1, 0, loc).Unix()
	startTime := pst_midnight

	count, err := collection.CountDocuments(ctx, bson.M{"createdAt": bson.M{"$gt": startTime, "$lt": endedTime}, "flowType": "Hipster"})
	log.Infof(ctx, "No of documents with flowtype as hipster = %v since  %+v = ", count, startTime)
	if err != nil {
		log.Errorf(ctx, "Failed to run find query: %v", err)
		return 0, err
	}
	return count, nil
}
func (DBClient *DocDBClient) GetTimedoutTask(ctx context.Context, WorkflowId string) string {
	wfExecData, err := DBClient.FetchWorkflowExecutionData(ctx, WorkflowId)
	if err != nil {
		log.Error(ctx, "error fetching data from db", err.Error())
		return ""
	}
	var timedOutStep *StepsPassedThroughBody
	for _, state := range wfExecData.StepsPassedThrough {
		if state.Status == running {
			timedOutStep = &state
			break
		}
	}
	if timedOutStep == nil {
		return ""
	}
	log.Info(ctx, "task timed out: %s", timedOutStep.TaskName)
	return timedOutStep.TaskName
}
