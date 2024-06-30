package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/ukane-philemon/scomp/graph"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	// Collections
	adminCollection   = "admin"
	classCollection   = "class"
	studentCollection = "students"

	// Keys
	dbIDKey                  = "_id"
	classIDKey               = "classID"
	usernameKey              = "username"
	nameKey                  = "name"
	subjectsKey              = "subjects"
	classReportKey           = "classReport"
	studentsSubjectRecordKey = "studentsSubjectRecord"
	studentReportKey         = "studentReport"
	lastUpdatedAtKey         = "lastUpdatedAt"

	// Actions
	actionSet = "$set"
)

// Check that *MongoDB implements graph.ClassDatabase.
var _ graph.ClassDatabase = (*MongoDB)(nil)

// MongoDB implements graph.ClassDatabase.
type MongoDB struct {
	ctx               context.Context
	db                *mongo.Database
	adminCollection   *mongo.Collection
	classCollection   *mongo.Collection
	studentCollection *mongo.Collection
}

// New connects to a mongo database and returns a new instance of *MongoDB.
func New(ctx context.Context, dbName string, connectionURL string) (*MongoDB, error) {
	if connectionURL == "" {
		return nil, errors.New("missing mongodb database connection URL")
	}

	if dbName == "" {
		return nil, errors.New("database name is required")
	}

	// Set server API version for the client.
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(connectionURL).SetServerAPIOptions(serverAPI)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongo.Connect error: %w", err)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("client.Ping error: %w", err)
	}

	log.Println("Database has been connected and pinged successfully...")

	db := client.Database(dbName)

	// Create a unique index on the admin collection.
	adminCollection := db.Collection(adminCollection)
	adminCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{
			Key:   usernameKey,
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	})

	// Create a unique index on the class collection.
	classCollection := db.Collection(classCollection)
	classCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{
			Key:   nameKey,
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	})

	return &MongoDB{
		ctx:               ctx,
		db:                db,
		adminCollection:   adminCollection,
		classCollection:   classCollection,
		studentCollection: db.Collection(studentCollection),
	}, nil
}

// Shutdown attempts to shutdown the database.
func (mdb *MongoDB) Shutdown(ctx context.Context) error {
	client := mdb.db.Client()
	err := client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("client.Disconnect error: %w", err)
	}

	log.Println("Database has been shutdown successfully...")

	return nil
}

// mapKey converts the provided keys to mongodb map key for easy retrieval of a
// specific map value.
func mapKey(keys ...string) string {
	return strings.Join(keys, ".")
}
