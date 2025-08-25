package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmailJob represents an email job in the queue
type EmailJob struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	To            string             `json:"to" bson:"to" validate:"required,email"`
	Subject       string             `json:"subject" bson:"subject" validate:"required"`
	HTML          string             `json:"html" bson:"html" validate:"required"`
	From          string             `json:"from" bson:"from" validate:"required,email"`
	Status        string             `json:"status" bson:"status"`             // pending, processing, sent, failed
	Priority      int                `json:"priority" bson:"priority"`         // 1=high, 2=normal, 3=low
	Attempts      int                `json:"attempts" bson:"attempts"`         // Number of attempts made
	MaxAttempts   int                `json:"max_attempts" bson:"max_attempts"` // Maximum attempts allowed
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	ScheduledAt   time.Time          `json:"scheduled_at" bson:"scheduled_at"`
	ProcessedAt   *time.Time         `json:"processed_at,omitempty" bson:"processed_at,omitempty"`
	ErrorMessage  *string            `json:"error_message,omitempty" bson:"error_message,omitempty"`
	Provider      string             `json:"provider,omitempty" bson:"provider,omitempty"`               // Which provider was used
	ProviderMsgID string             `json:"provider_msg_id,omitempty" bson:"provider_msg_id,omitempty"` // Provider's message ID
}

// SendEmailRequest represents the API request for sending an email
type SendEmailRequest struct {
	To       string `json:"to" validate:"required,email"`
	Subject  string `json:"subject" validate:"required"`
	HTML     string `json:"html" validate:"required"`
	From     string `json:"from" validate:"required,email"`
	Priority int    `json:"priority" validate:"min=1,max=3"` // 1=high, 2=normal, 3=low
}

// EmailResponse represents the API response
type EmailResponse struct {
	ID                string    `json:"id"`
	Status            string    `json:"status"`
	Message           string    `json:"message"`
	QueuedAt          time.Time `json:"queued_at"`
	EstimatedDelivery time.Time `json:"estimated_delivery"`
}

// EmailStatus represents the current status of an email
type EmailStatus struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	To            string     `json:"to"`
	Subject       string     `json:"subject"`
	CreatedAt     time.Time  `json:"created_at"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	Provider      string     `json:"provider,omitempty"`
	ProviderMsgID string     `json:"provider_msg_id,omitempty"`
}

// RateLimit represents rate limiting information
type RateLimit struct {
	Key       string    `json:"key" bson:"key"`
	Count     int       `json:"count" bson:"count"`
	Limit     int       `json:"limit" bson:"limit"`
	ResetAt   time.Time `json:"reset_at" bson:"reset_at"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// EmailStats represents basic email statistics
type EmailStats struct {
	TotalQueued     int64 `json:"total_queued"`
	TotalSent       int64 `json:"total_sent"`
	TotalFailed     int64 `json:"total_failed"`
	PendingCount    int64 `json:"pending_count"`
	ProcessingCount int64 `json:"processing_count"`
	QueueSize       int64 `json:"queue_size"`
}

// Constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSent       = "sent"
	StatusFailed     = "failed"

	PriorityHigh   = 1
	PriorityNormal = 2
	PriorityLow    = 3
)
