package email

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/thenasky/go-framework/internal/database"
	"github.com/thenasky/go-framework/modules/email/models"
	"github.com/thenasky/go-framework/modules/email/providers"
	"github.com/thenasky/go-framework/modules/email/queue"
	"github.com/thenasky/go-framework/modules/email/workers"
)

// EmailService handles email business logic
type EmailService struct {
	queue       *queue.MongoQueue
	worker      *workers.EmailWorker
	providers   []providers.EmailProvider
	initialized bool
	mu          sync.Mutex
}

// NewEmailService creates a new email service
func NewEmailService() *EmailService {
	return &EmailService{
		initialized: false,
	}
}

// ensureInitialized ensures the service is initialized
func (s *EmailService) ensureInitialized() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	// Check if MongoDB is connected
	if database.MongoDB == nil {
		return fmt.Errorf("MongoDB not connected")
	}

	// Create queue
	queue := queue.NewMongoQueue()

	// Create providers
	providers := createProviders()

	// Create worker
	worker := workers.NewEmailWorker(queue, providers, nil)

	// Start worker
	worker.Start()

	s.queue = queue
	s.worker = worker
	s.providers = providers
	s.initialized = true

	return nil
}

// createProviders creates and configures email providers
func createProviders() []providers.EmailProvider {
	var emailProviders []providers.EmailProvider

	// Add SMTP provider if configured
	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		smtpPort := 587 // Default to 587
		if portStr := os.Getenv("SMTP_PORT"); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				smtpPort = port
			}
		}

		smtpConfig := &providers.ProviderConfig{
			SMTPHost:         smtpHost,
			SMTPPort:         smtpPort,
			SMTPUsername:     os.Getenv("SMTP_USERNAME"),
			SMTPPassword:     os.Getenv("SMTP_PASSWORD"),
			SMTPFrom:         os.Getenv("SMTP_FROM"),
			MaxEmailsPerHour: getEnvInt("SMTP_MAX_EMAILS_PER_HOUR", 1000),
			MaxEmailsPerDay:  getEnvInt("SMTP_MAX_EMAILS_PER_DAY", 10000),
		}

		smtpProvider := providers.NewSMTPProvider(smtpConfig)
		emailProviders = append(emailProviders, smtpProvider)
	}

	// Add SendGrid provider if configured
	if sendGridKey := os.Getenv("SENDGRID_API_KEY"); sendGridKey != "" {
		_ = &providers.ProviderConfig{
			SendGridAPIKey:   sendGridKey,
			SendGridFrom:     os.Getenv("SENDGRID_FROM"),
			MaxEmailsPerHour: getEnvInt("SENDGRID_MAX_EMAILS_PER_HOUR", 10000),
			MaxEmailsPerDay:  getEnvInt("SENDGRID_MAX_EMAILS_PER_DAY", 100000),
		}

		// TODO: Implement SendGrid provider
		// sendGridProvider := providers.NewSendGridProvider(sendGridConfig)
		// emailProviders = append(emailProviders, sendGridProvider)
	}

	// If no providers configured, create a dummy one for testing
	if len(emailProviders) == 0 {
		dummyProvider := &DummyProvider{}
		emailProviders = append(emailProviders, dummyProvider)
	}

	return emailProviders
}

// getEnvInt gets an environment variable as integer with fallback
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

// SendEmail queues an email for sending
func (s *EmailService) SendEmail(req *models.SendEmailRequest) (*models.EmailResponse, error) {
	// Ensure service is initialized
	if err := s.ensureInitialized(); err != nil {
		return nil, fmt.Errorf("service not ready: %w", err)
	}

	// Validate request
	if err := s.validateSendRequest(req); err != nil {
		return nil, err
	}

	// Check rate limiting
	if err := s.checkRateLimit(req.From); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Create email job
	job := &models.EmailJob{
		To:          req.To,
		Subject:     req.Subject,
		HTML:        req.HTML,
		From:        req.From,
		Priority:    req.Priority,
		Status:      models.StatusPending,
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
		MaxAttempts: 3,
	}

	// Enqueue the job
	if err := s.queue.Enqueue(job); err != nil {
		return nil, fmt.Errorf("failed to enqueue email: %w", err)
	}

	// Create response
	response := &models.EmailResponse{
		ID:                job.ID.Hex(),
		Status:            "queued",
		Message:           "Email queued successfully",
		QueuedAt:          job.CreatedAt,
		EstimatedDelivery: time.Now().Add(5 * time.Minute), // Estimate 5 minutes
	}

	return response, nil
}

// GetEmailStatus returns the status of an email
func (s *EmailService) GetEmailStatus(emailID string) (*models.EmailStatus, error) {
	// Ensure service is initialized
	if err := s.ensureInitialized(); err != nil {
		return nil, fmt.Errorf("service not ready: %w", err)
	}

	// Parse ObjectID
	objectID, err := parseObjectID(emailID)
	if err != nil {
		return nil, fmt.Errorf("invalid email ID: %w", err)
	}

	// Get job from queue
	job, err := s.queue.GetJobByID(objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email job: %w", err)
	}

	if job == nil {
		return nil, fmt.Errorf("email not found")
	}

	// Convert to status response
	status := &models.EmailStatus{
		ID:            job.ID.Hex(),
		Status:        job.Status,
		To:            job.To,
		Subject:       job.Subject,
		CreatedAt:     job.CreatedAt,
		ProcessedAt:   job.ProcessedAt,
		ErrorMessage:  job.ErrorMessage,
		Provider:      job.Provider,
		ProviderMsgID: job.ProviderMsgID,
	}

	return status, nil
}

// GetStats returns email statistics
func (s *EmailService) GetStats() (*models.EmailStats, error) {
	// Ensure service is initialized
	if err := s.ensureInitialized(); err != nil {
		return nil, fmt.Errorf("service not ready: %w", err)
	}

	return s.worker.GetStats()
}

// validateSendRequest validates the send email request
func (s *EmailService) validateSendRequest(req *models.SendEmailRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient email is required")
	}

	if req.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if req.HTML == "" {
		return fmt.Errorf("HTML content is required")
	}

	if req.From == "" {
		return fmt.Errorf("sender email is required")
	}

	// Validate email formats
	for _, provider := range s.providers {
		if err := provider.ValidateEmail(req.To); err != nil {
			return fmt.Errorf("invalid recipient email: %w", err)
		}
		if err := provider.ValidateEmail(req.From); err != nil {
			return fmt.Errorf("invalid sender email: %w", err)
		}
	}

	// Validate priority
	if req.Priority < 1 || req.Priority > 3 {
		return fmt.Errorf("priority must be between 1 and 3")
	}

	return nil
}

// checkRateLimit checks if the sender has exceeded rate limits
func (s *EmailService) checkRateLimit(sender string) error {
	// TODO: Implement proper rate limiting
	// For now, just return nil (no rate limiting)
	return nil
}

// parseObjectID parses a string to ObjectID
func parseObjectID(id string) (primitive.ObjectID, error) {
	// Parse the string to ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid ObjectID format: %w", err)
	}
	return objectID, nil
}

// Stop stops the email service
func (s *EmailService) Stop() {
	if s.worker != nil {
		s.worker.Stop()
	}
}

// DummyProvider is a dummy provider for testing when no real providers are configured
type DummyProvider struct{}

func (p *DummyProvider) Send(email *models.EmailJob) error {
	// Simulate successful send
	return nil
}

func (p *DummyProvider) GetName() string {
	return "dummy"
}

func (p *DummyProvider) GetQuota() (*providers.QuotaInfo, error) {
	return &providers.QuotaInfo{
		Provider:    "dummy",
		DailyLimit:  1000,
		DailyUsed:   0,
		HourlyLimit: 100,
		HourlyUsed:  0,
		Remaining:   100,
		ResetTime:   "N/A",
	}, nil
}

func (p *DummyProvider) ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email address is empty")
	}
	if !contains(email, "@") {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || contains(s[1:len(s)-1], substr)))
}
