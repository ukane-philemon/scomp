package db

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// NewMongoDB connects to a mongo database and returns a new instance of
// *mongo.Databas.
func NewMongoDB(ctx context.Context, dbName string, connectionURL string) (*mongo.Database, error) {
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

	return client.Database(dbName), nil
}

// ShutdownMongoDB attempts to shutdown db.
func ShutdownMongoDB(ctx context.Context, db *mongo.Database) error {
	err := db.Client().Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("client.Disconnect error: %w", err)
	}

	log.Println("Database has been shutdown successfully...")

	return nil
}
