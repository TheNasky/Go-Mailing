package providers

import (
	"github.com/thenasky/go-framework/modules/email/models"
)

// EmailProvider defines the interface for email service providers
type EmailProvider interface {
	// Send sends a single email
	Send(email *models.EmailJob) error

	// GetName returns the provider name
	GetName() string

	// GetQuota returns current quota information
	GetQuota() (*QuotaInfo, error)

	// ValidateEmail validates an email address
	ValidateEmail(email string) error
}

// QuotaInfo represents provider quota information
type QuotaInfo struct {
	Provider    string `json:"provider"`
	DailyLimit  int    `json:"daily_limit"`
	DailyUsed   int    `json:"daily_used"`
	HourlyLimit int    `json:"hourly_limit"`
	HourlyUsed  int    `json:"hourly_used"`
	Remaining   int    `json:"remaining"`
	ResetTime   string `json:"reset_time"`
}

// ProviderConfig holds configuration for email providers
type ProviderConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from"`

	SendGridAPIKey string `json:"sendgrid_api_key"`
	SendGridFrom   string `json:"sendgrid_from"`

	// Rate limiting per provider
	MaxEmailsPerHour int `json:"max_emails_per_hour"`
	MaxEmailsPerDay  int `json:"max_emails_per_day"`
}
