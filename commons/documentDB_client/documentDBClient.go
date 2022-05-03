package documentDB_client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CaFilePath               = "rds-combined-ca-bundle.pem"
	ConnectTimeout           = 5
	QueryTimeout             = 30
	ConnectionStringTemplate = "mongodb://%s:%s@%s/%s?replicaSet=rs0&readpreference=%s"
	Database                 = "test"
	WorkflowDataCollection   = "WorkflowData"
	StepsDataCollection      = "StepsData"
)

var (
	Username        string
	Password        string
	ClusterEndpoint string
	ReadPreference  string
)

type IDocDBClient interface {
	FetchStepExecution(StepId string) (StepExecutionDataBody, error)
	InsertStepExecution(StepExecutionData StepExecutionDataBody) error
	InsertWorkflowExecution(Data WorkflowExecutionDataBody) error
	UpdateDocumentDB(query, update interface{}, collectionName string) error
	FetchWorkflowExecution(workFlowId string) (WorkflowExecutionDataBody, error)
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
	Input              map[string]interface{} `bson:"input"`
	Output             map[string]interface{} `bson:"output"`
	IntermediateOutput map[string]interface{} `bson:"intermediateOutput"`
	Status             string                 `bson:"status"`
	TaskToken          string                 `bson:"taskToken"`
	WorkflowId         string                 `bson:"workflowId"`
	TaskName           string                 `bson:"taskName"`
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
		log.Fatalf("Failed getting TLS configuration: %v", err)
	}
	DBClient, err := mongo.NewClient(options.Client().ApplyURI(connectionURI).SetTLSConfig(tlsConfig))
	if err != nil {
		log.Fatal(err)
	}
	return &DocDBClient{DBClient: DBClient}
}

func (DBClient *DocDBClient) FetchStepExecution(StepId string) (StepExecutionDataBody, error) {
	collection := DBClient.DBClient.Database(Database).Collection(StepsDataCollection)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()
	var StepExecutionData StepExecutionDataBody
	err := collection.FindOne(ctx, bson.M{"_id": StepId}).Decode(&StepExecutionData)
	if err != nil {
		log.Fatalf("Failed to run find query: %v", err)
		return StepExecutionDataBody{}, err
	}
	return StepExecutionData, nil
}
func (DBClient *DocDBClient) InsertStepExecution(StepExecutionData StepExecutionDataBody) error {
	collection := DBClient.DBClient.Database(Database).Collection(StepsDataCollection)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()
	res, err := collection.InsertOne(ctx, StepExecutionData)
	if err != nil {
		log.Fatalf("Failed to insert document: %v", err)
		return err
	}
	id := res.InsertedID
	log.Printf("Inserted document ID: %s", id)
	return nil
}
func (DBClient *DocDBClient) InsertWorkflowExecution(Data WorkflowExecutionDataBody) error {
	collection := DBClient.DBClient.Database(Database).Collection(WorkflowDataCollection)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()
	res, err := collection.InsertOne(ctx, Data)
	if err != nil {
		log.Fatalf("Failed to insert document: %v", err)
		return err
	}
	id := res.InsertedID
	log.Printf("Inserted document ID: %s", id)
	return nil
}
func (DBClient *DocDBClient) UpdateDocumentDB(query, update interface{}, collectionName string) error {
	collection := DBClient.DBClient.Database(Database).Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	res, err := collection.UpdateMany(ctx, query, update)

	if err != nil {
		log.Fatalf("Failed to update document: %v", err)
		return err
	}
	if res.MatchedCount == 0 {
		log.Fatalf("Unable to update document as no such document exist")
	}
	log.Printf("Updated document ID: %s", res.UpsertedID)
	return nil
}
func (DBClient *DocDBClient) FetchWorkflowExecution(workFlowId string) (WorkflowExecutionDataBody, error) {
	collection := DBClient.DBClient.Database(Database).Collection(WorkflowDataCollection)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()
	var WorkflowExecutionData WorkflowExecutionDataBody
	err := collection.FindOne(ctx, bson.M{"_id": workFlowId}).Decode(&WorkflowExecutionData)
	if err != nil {
		log.Fatalf("Failed to run find query: %v", err)
		return WorkflowExecutionDataBody{}, err
	}
	return WorkflowExecutionData, nil
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
