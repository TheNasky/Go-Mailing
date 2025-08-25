package providers

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"github.com/thenasky/go-framework/modules/email/models"
)

// SMTPProvider implements EmailProvider for SMTP
type SMTPProvider struct {
	config *ProviderConfig
}

// extractEmailAddress extracts just the email address from a "Display Name <email@domain.com>" format
func extractEmailAddress(from string) string {
	// If it contains < and >, extract the email part
	if strings.Contains(from, "<") && strings.Contains(from, ">") {
		start := strings.Index(from, "<")
		end := strings.Index(from, ">")
		if start != -1 && end != -1 && end > start {
			return from[start+1 : end]
		}
	}
	// Otherwise return as-is (already just an email address)
	return from
}

// NewSMTPProvider creates a new SMTP provider
func NewSMTPProvider(config *ProviderConfig) *SMTPProvider {
	return &SMTPProvider{
		config: config,
	}
}

// Send sends an email via SMTP
func (p *SMTPProvider) Send(email *models.EmailJob) error {
	// Set default values if not provided
	if p.config.SMTPFrom == "" {
		p.config.SMTPFrom = p.config.SMTPUsername
	}

	// Create email message
	message := p.createEmailMessage(email)

	// Connect to SMTP server
	auth := smtp.PlainAuth("", p.config.SMTPUsername, p.config.SMTPPassword, p.config.SMTPHost)

	// Determine if we need TLS
	var err error
	if p.config.SMTPPort == 587 {
		// Use STARTTLS
		err = p.sendWithSTARTTLS(auth, message, email)
	} else if p.config.SMTPPort == 465 {
		// Use SSL/TLS
		err = p.sendWithTLS(auth, message, email)
	} else {
		// Use plain SMTP
		err = p.sendPlain(auth, message, email)
	}

	if err != nil {
		// Log the email message for debugging
		log.Printf("SMTP send failed for email to %s: %v", email.To, err)
		log.Printf("Email message content: %s", string(message))
		return fmt.Errorf("SMTP send failed: %w", err)
	}

	return nil
}

// createEmailMessage creates the email message in proper format
func (p *SMTPProvider) createEmailMessage(email *models.EmailJob) []byte {
	// Create headers with proper RFC 5322 format in consistent order
	type header struct {
		key   string
		value string
	}

	headers := []header{
		{"From", p.config.SMTPFrom},
		{"To", email.To},
		{"Subject", email.Subject},
		{"Date", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")},
		{"Message-ID", fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), email.ID.Hex(), p.config.SMTPHost)},
		{"MIME-Version", "1.0"},
		{"Content-Type", "text/html; charset=UTF-8"},
		{"Content-Transfer-Encoding", "8bit"},
	}

	// Build message
	var message strings.Builder

	// Add headers in consistent order
	for _, h := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", h.key, h.value))
	}

	// Add blank line between headers and body (RFC 5322 requirement)
	// This creates the required separation: \r\n\r\n
	message.WriteString("\r\n")

	// Add body with proper line ending handling
	// Ensure HTML content doesn't break SMTP formatting
	body := strings.ReplaceAll(email.HTML, "\n", "\r\n")
	// Remove any carriage returns that might cause issues
	body = strings.ReplaceAll(body, "\r\r", "\r")

	// Write the body content
	message.WriteString(body)

	// Ensure message ends with proper line ending
	if !strings.HasSuffix(body, "\r\n") {
		message.WriteString("\r\n")
	}

	// Log the message for debugging (remove in production)
	messageStr := message.String()
	log.Printf("Generated email message for %s:\n%s", email.To, messageStr)

	// Validate the message format
	if !strings.Contains(messageStr, "\r\n\r\n") {
		log.Printf("WARNING: Message missing proper header-body separator")
	} else {
		log.Printf("✓ Message has proper header-body separator")
	}

	// Show the exact structure for debugging
	parts := strings.Split(messageStr, "\r\n\r\n")
	if len(parts) >= 2 {
		log.Printf("✓ Headers section:\n%s", parts[0])
		log.Printf("✓ Body section:\n%s", parts[1])
	}

	return []byte(messageStr)
}

// sendWithSTARTTLS sends email using STARTTLS
func (p *SMTPProvider) sendWithSTARTTLS(auth smtp.Auth, message []byte, email *models.EmailJob) error {
	// Connect to server
	host := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)
	client, err := smtp.Dial(host)
	if err != nil {
		return err
	}
	defer client.Close()

	// Start TLS
	if err = client.StartTLS(&tls.Config{ServerName: p.config.SMTPHost}); err != nil {
		return err
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return err
	}

	// Send email - FIXED: Extract email address from display name format
	fromEmail := extractEmailAddress(p.config.SMTPFrom)
	if err = client.Mail(fromEmail); err != nil {
		return err
	}
	if err = client.Rcpt(email.To); err != nil {
		return err
	}

	// Write message
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(message)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// sendWithTLS sends email using SSL/TLS
func (p *SMTPProvider) sendWithTLS(auth smtp.Auth, message []byte, email *models.EmailJob) error {
	host := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)

	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: p.config.SMTPHost,
	}

	// Connect with TLS
	conn, err := tls.Dial("tcp", host, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, p.config.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return err
	}

	// Send email - FIXED: Extract email address from display name format
	fromEmail := extractEmailAddress(p.config.SMTPFrom)
	if err = client.Mail(fromEmail); err != nil {
		return err
	}
	if err = client.Rcpt(email.To); err != nil {
		return err
	}

	// Write message
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(message)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// sendPlain sends email using plain SMTP
func (p *SMTPProvider) sendPlain(auth smtp.Auth, message []byte, email *models.EmailJob) error {
	host := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)
	// FIXED: Extract email address from display name format
	fromEmail := extractEmailAddress(p.config.SMTPFrom)
	log.Printf("SMTP MAIL FROM: %s (extracted from: %s)", fromEmail, p.config.SMTPFrom)
	return smtp.SendMail(host, auth, fromEmail, []string{email.To}, message)
}

// GetName returns the provider name
func (p *SMTPProvider) GetName() string {
	return "smtp"
}

// GetQuota returns quota information (SMTP doesn't have built-in quotas)
func (p *SMTPProvider) GetQuota() (*QuotaInfo, error) {
	return &QuotaInfo{
		Provider:    "smtp",
		DailyLimit:  p.config.MaxEmailsPerDay,
		DailyUsed:   0, // SMTP doesn't track this
		HourlyLimit: p.config.MaxEmailsPerHour,
		HourlyUsed:  0, // SMTP doesn't track this
		Remaining:   p.config.MaxEmailsPerHour,
		ResetTime:   "N/A",
	}, nil
}

// ValidateEmail validates an email address format
func (p *SMTPProvider) ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email address is empty")
	}

	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format: missing @ symbol")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format: multiple @ symbols")
	}

	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email format: empty local or domain part")
	}

	if !strings.Contains(parts[1], ".") {
		return fmt.Errorf("invalid email format: domain must contain a dot")
	}

	return nil
}
