package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	scomDB = "scomp"

	classCollection = "class"

	// Keys
	dbIDKey    = "_id"
	classIDKey = "classID"
)

type MongoDB struct {
	ctx             context.Context
	db              *mongo.Database
	classCollection *mongo.Collection
	log             *slog.Logger
}

// New connects to a mongo database and returns a new instance of *MongoDB.
func New(ctx context.Context, connectionURL string, logger *slog.Logger) (*MongoDB, error) {
	if connectionURL == "" {
		return nil, errors.New("missing mongodb database connection URL")
	}

	if logger == nil {
		return nil, errors.New("mongodb logger is required")
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

	logger.Info("Database has been connected and pinged successfully...")

	db := client.Database(scomDB)

	// Create a unique index on the users collection.
	classCollection := db.Collection(classCollection)
	classCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{
			Key:   classIDKey,
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	})

	return &MongoDB{
		ctx:             ctx,
		db:              db,
		classCollection: classCollection,
		log:             logger,
	}, nil
}

// Shutdown attempts to shutdown the database.
func (mdb *MongoDB) Shutdown(ctx context.Context) error {
	client := mdb.db.Client()
	err := client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("client.Disconnect error: %w", err)
	}

	mdb.log.Info("Database has been shutdown successfully...")

	return nil
}
