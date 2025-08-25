package email

import (
	"github.com/thenasky/go-framework/internal/router"
	"github.com/thenasky/go-framework/modules/email/models"
)

// Controller handles HTTP requests for email operations
type Controller struct {
	service *EmailService
}

// NewController creates a new email controller
func NewController() *Controller {
	return &Controller{
		service: NewEmailService(),
	}
}

// SendEmail handles POST /api/v1/emails/send
func (c *Controller) SendEmail(req *router.Req, res *router.Res) {
	// Parse request body
	var sendReq models.SendEmailRequest
	if err := req.JSON(&sendReq); err != nil {
		res.BadRequest("Invalid request body", map[string]string{"error": err.Error()})
		return
	}

	// Set default priority if not provided
	if sendReq.Priority == 0 {
		sendReq.Priority = models.PriorityNormal
	}

	// Send email
	response, err := c.service.SendEmail(&sendReq)
	if err != nil {
		res.Error("Failed to send email", map[string]string{"error": err.Error()})
		return
	}

	// Return success response
	res.Created("Email queued successfully", response)
}

// GetEmailStatus handles GET /api/v1/emails/{id}/status
func (c *Controller) GetEmailStatus(req *router.Req, res *router.Res) {
	// Get email ID from URL parameters
	emailID := req.Param("id")
	if emailID == "" {
		res.BadRequest("Email ID is required", nil)
		return
	}

	// Get email status
	status, err := c.service.GetEmailStatus(emailID)
	if err != nil {
		res.NotFound("Email not found", map[string]string{"error": err.Error()})
		return
	}

	// Return status
	res.Success("Email status retrieved successfully", status)
}

// GetStats handles GET /api/v1/emails/stats
func (c *Controller) GetStats(req *router.Req, res *router.Res) {
	// Get email statistics
	stats, err := c.service.GetStats()
	if err != nil {
		res.Error("Failed to get statistics", map[string]string{"error": err.Error()})
		return
	}

	// Return statistics
	res.Success("Statistics retrieved successfully", stats)
}

// Health handles GET /api/v1/emails/health
func (c *Controller) Health(req *router.Req, res *router.Res) {
	// Check if service is running
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "email",
		"timestamp": "2024-01-01T00:00:00Z",
		"version":   "1.0.0",
	}

	res.Success("Email service is healthy", health)
}
