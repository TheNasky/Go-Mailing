package providers

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/thenasky/go-framework/modules/email/models"
)

// SMTPProvider implements EmailProvider for SMTP
type SMTPProvider struct {
	config *ProviderConfig
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
		return fmt.Errorf("SMTP send failed: %w", err)
	}

	return nil
}

// createEmailMessage creates the email message in proper format
func (p *SMTPProvider) createEmailMessage(email *models.EmailJob) []byte {
	// Create headers
	headers := make(map[string]string)
	headers["From"] = p.config.SMTPFrom
	headers["To"] = email.To
	headers["Subject"] = email.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// Build message
	var message strings.Builder

	// Add headers
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")

	// Add body
	message.WriteString(email.HTML)

	return []byte(message.String())
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

	// Send email - FIXED: Use email.To instead of p.config.SMTPFrom
	if err = client.Mail(p.config.SMTPFrom); err != nil {
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

	// Send email - FIXED: Use email.To instead of p.config.SMTPFrom
	if err = client.Mail(p.config.SMTPFrom); err != nil {
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
	// FIXED: Use email.To instead of p.config.SMTPFrom
	return smtp.SendMail(host, auth, p.config.SMTPFrom, []string{email.To}, message)
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
