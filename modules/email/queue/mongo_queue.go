package queue

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/thenasky/go-framework/internal/database"
	"github.com/thenasky/go-framework/modules/email/models"
)

// MongoQueue implements email queue using MongoDB
type MongoQueue struct {
	collection *mongo.Collection
	ctx        context.Context
}

// NewMongoQueue creates a new MongoDB-based email queue
func NewMongoQueue() *MongoQueue {
	// Check if MongoDB is connected
	if database.MongoDB == nil {
		panic("MongoDB not connected. Call database.ConnectMongoDB() first.")
	}

	collection := database.MongoDB.Collection("emails_queue")

	// Create indexes for performance
	createIndexes(collection)

	return &MongoQueue{
		collection: collection,
		ctx:        context.Background(),
	}
}

// createIndexes creates necessary indexes for the queue
func createIndexes(collection *mongo.Collection) {
	// Index for finding next job (status + priority + scheduled_at)
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "status", Value: 1},
			{Key: "priority", Value: 1},
			{Key: "scheduled_at", Value: 1},
		},
		Options: options.Index().SetName("status_priority_scheduled"),
	}
	collection.Indexes().CreateOne(context.Background(), indexModel)

	// TTL index to automatically clean up old jobs (24 hours)
	ttlIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "created_at", Value: 1},
		},
		Options: options.Index().SetExpireAfterSeconds(86400).SetName("ttl_created_at"),
	}
	collection.Indexes().CreateOne(context.Background(), ttlIndex)

	// Index for status queries
	statusIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "status", Value: 1},
		},
		Options: options.Index().SetName("status_index"),
	}
	collection.Indexes().CreateOne(context.Background(), statusIndex)
}

// Enqueue adds an email job to the queue
func (q *MongoQueue) Enqueue(job *models.EmailJob) error {
	// Set default values
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.ScheduledAt.IsZero() {
		job.ScheduledAt = time.Now()
	}
	if job.Status == "" {
		job.Status = models.StatusPending
	}
	if job.Priority == 0 {
		job.Priority = models.PriorityNormal
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}

	// Insert the job
	result, err := q.collection.InsertOne(q.ctx, job)
	if err != nil {
		return fmt.Errorf("failed to enqueue email: %w", err)
	}

	// Set the generated ID
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		job.ID = oid
	}

	return nil
}

// Dequeue gets the next available job from the queue
func (q *MongoQueue) Dequeue() (*models.EmailJob, error) {
	// Use findOneAndUpdate for atomic operation
	filter := bson.M{
		"status":       bson.M{"$in": []string{models.StatusPending, models.StatusFailed}},
		"scheduled_at": bson.M{"$lte": time.Now()},
	}

	update := bson.M{
		"$set": bson.M{
			"status": models.StatusProcessing,
		},
		"$inc": bson.M{
			"attempts": 1,
		},
	}

	opts := options.FindOneAndUpdate().SetSort(bson.D{
		{Key: "priority", Value: 1},
		{Key: "created_at", Value: 1},
	}).SetReturnDocument(options.After)

	var job models.EmailJob
	err := q.collection.FindOneAndUpdate(q.ctx, filter, update, opts).Decode(&job)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	return &job, nil
}

// MarkComplete marks a job as successfully completed
func (q *MongoQueue) MarkComplete(jobID primitive.ObjectID, provider, providerMsgID string) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":          models.StatusSent,
			"processed_at":    now,
			"provider":        provider,
			"provider_msg_id": providerMsgID,
		},
	}

	_, err := q.collection.UpdateOne(
		q.ctx,
		bson.M{"_id": jobID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to mark job complete: %w", err)
	}

	return nil
}

// MarkFailed marks a job as failed
func (q *MongoQueue) MarkFailed(jobID primitive.ObjectID, errorMessage string) error {
	update := bson.M{
		"$set": bson.M{
			"status":        models.StatusFailed,
			"error_message": errorMessage,
		},
	}

	_, err := q.collection.UpdateOne(
		q.ctx,
		bson.M{"_id": jobID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to mark job failed: %w", err)
	}

	return nil
}

// GetJobByID retrieves a job by its ID
func (q *MongoQueue) GetJobByID(jobID primitive.ObjectID) (*models.EmailJob, error) {
	var job models.EmailJob
	err := q.collection.FindOne(q.ctx, bson.M{"_id": jobID}).Decode(&job)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// GetQueueStats returns queue statistics
func (q *MongoQueue) GetQueueStats() (*models.EmailStats, error) {
	stats := &models.EmailStats{}

	// Count by status
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$status",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := q.collection.Aggregate(q.ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	defer cursor.Close(q.ctx)

	for cursor.Next(q.ctx) {
		var result struct {
			Status string `bson:"_id"`
			Count  int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		switch result.Status {
		case models.StatusPending:
			stats.PendingCount = result.Count
		case models.StatusProcessing:
			stats.ProcessingCount = result.Count
		case models.StatusSent:
			stats.TotalSent = result.Count
		case models.StatusFailed:
			stats.TotalFailed = result.Count
		}
	}

	// Total queued (pending + processing)
	stats.TotalQueued = stats.PendingCount + stats.ProcessingCount
	stats.QueueSize = stats.PendingCount

	return stats, nil
}

// CleanupOldJobs removes old completed/failed jobs
func (q *MongoQueue) CleanupOldJobs(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// Delete old completed/failed jobs
	filter := bson.M{
		"status":       bson.M{"$in": []string{models.StatusSent, models.StatusFailed}},
		"processed_at": bson.M{"$lt": cutoff},
	}

	_, err := q.collection.DeleteMany(q.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to cleanup old jobs: %w", err)
	}

	return nil
}

// GetPendingJobsCount returns the count of pending jobs
func (q *MongoQueue) GetPendingJobsCount() (int64, error) {
	count, err := q.collection.CountDocuments(q.ctx, bson.M{"status": models.StatusPending})
	if err != nil {
		return 0, fmt.Errorf("failed to count pending jobs: %w", err)
	}
	return count, nil
}
