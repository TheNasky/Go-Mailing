package database

import (
	"context"
	"os"
	"time"

	"github.com/thenasky/go-framework/internal/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
)

// ConnectMongoDB attempts to connect to MongoDB if MONGODB_URI is present
func ConnectMongoDB() {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		// No logging when MongoDB URI is not found - as requested
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.LogMongoError("Failed to connect to MongoDB: " + err.Error())
		return
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		logger.LogMongoError("Failed to connect to MongoDB")
		return
	}

	MongoClient = client

	// Get database name from environment variable or use default
	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "go_db" // fallback default
	}

	MongoDB = client.Database(dbName)

	logger.LogMongo("Successfully connected to MongoDB database: " + dbName)
}

// DisconnectMongoDB disconnects from MongoDB if connected
func DisconnectMongoDB() {
	if MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := MongoClient.Disconnect(ctx); err != nil {
			logger.LogMongoError("Error disconnecting from MongoDB: " + err.Error())
		} else {
			logger.LogMongo("Disconnected from MongoDB")
		}
	}
}
