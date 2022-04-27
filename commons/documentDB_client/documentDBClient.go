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
	MetaCollection           = "callBackData"
	DataStoreCollection      = "datastore"
)

var (
	DBClient        *mongo.Client
	Username        string
	Password        string
	ClusterEndpoint string
	ReadPreference  string
)

type IDocDBClient interface {
	FetchMetaData(CallbackID string) (MetaData, error)
	DeleteMetaData(CallbackID string) error
	UpdateDocumentDB(Data DataStoreBody) bool
}
type MetaData struct {
	ID   string `bson:"_id"`
	Data struct {
		OrderID    string `bson:"order_id"`
		TaskToken  string `bson:"task_token"`
		WorkflowID string `bson:"workflow_id"`
		TaskName   string `bson:"task_name"`
	} `bson:"data"`
}
type DocDBClient struct {
	DBClient *mongo.Client
}
type DataStoreBody struct {
	WorkflowId         string                 `bson:"_id" `
	OrderId            string                 `bson:"orderId" `
	FlowType           string                 `bson:"flowType"`
	UpdatedAt          time.Time              `bson:"updatedAt" `
	CreatedAt          time.Time              `bson:"createdAt" `
	EndAt              time.Time              `bson:"endAt" `
	RunningState       map[string]interface{} `bson:"runningState" `
	InitialInput       map[string]interface{} `bson:"initialInput" `
	StepsPassedThrough []string               `bson:"stepsPassedThrough" `
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

func (DBClient *DocDBClient) FetchMetaData(CallbackID string) (MetaData, error) {
	collection := DBClient.DBClient.Database(Database).Collection(MetaCollection)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()
	var DBMetaData MetaData
	err := collection.FindOne(ctx, bson.M{"_id": CallbackID}).Decode(&DBMetaData)
	if err != nil {
		log.Fatalf("Failed to run find query: %v", err)
		return MetaData{}, err
	}
	return DBMetaData, nil
}
func (DBClient *DocDBClient) DeleteMetaData(CallbackID string) error {
	collection := DBClient.DBClient.Database(Database).Collection(MetaCollection)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	_, err := collection.DeleteMany(ctx, bson.M{"_id": CallbackID})
	if err != nil {
		log.Fatalf("Failed to run delete query: %v", err)
		return err
	}
	return nil
}
func (DBClient *DocDBClient) InsertMetaData(MetaData MetaData) error {
	collection := DBClient.DBClient.Database(Database).Collection(MetaCollection)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()
	res, err := collection.InsertOne(ctx, MetaData)
	if err != nil {
		log.Fatalf("Failed to insert document: %v", err)
		return err
	}
	id := res.InsertedID
	log.Printf("Inserted document ID: %s", id)
	return nil
}
func (DBClient *DocDBClient) DataStoreInsertion(Data DataStoreBody) error {
	collection := DBClient.DBClient.Database(Database).Collection(DataStoreCollection)

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
func (DBClient *DocDBClient) UpdateEndTimeInDocumentDB(workFlowId string) error {
	collection := DBClient.DBClient.Database(Database).Collection(DataStoreCollection)

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	res, err := collection.UpdateMany(ctx, bson.M{"_id": workFlowId}, bson.D{
		{"$set", bson.D{{"endAt", time.Now()}}},
	})

	if err != nil {
		log.Fatalf("Failed to update document: %v", err)
		return err
	}
	if res.MatchedCount == 0 {
		log.Fatalf("Unable to update document as no such document exist")
	}
	log.Printf("Updated document ID: %s", workFlowId)
	return nil
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
