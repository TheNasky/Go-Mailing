package email

import (
	"github.com/thenasky/go-framework/internal/core"
	"github.com/thenasky/go-framework/internal/router"

	"github.com/gorilla/mux"
)

// Module represents the email module
type Module struct {
	controller *Controller
}

// NewModule creates a new email module
func NewModule() *Module {
	return &Module{
		controller: NewController(),
	}
}

// RegisterRoutes implements the core.ModuleRegistrar interface
func (m *Module) RegisterRoutes(r *mux.Router) {
	// Create email routes
	router.Router(r, "/api/v1/emails").
		// Main email sending endpoint
		Post("/send", m.controller.SendEmail).
		// Email status and management
		Get("/{id}/status", m.controller.GetEmailStatus).
		Get("/stats", m.controller.GetStats).
		Get("/health", m.controller.Health)
}

// init automatically registers this module when the package is imported
func init() {
	core.RegisterModule("email", NewModule())
}
