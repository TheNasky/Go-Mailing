package demo

import (
	"github.com/thenasky/go-framework/internal/core"
	"github.com/thenasky/go-framework/internal/router"

	"github.com/gorilla/mux"
)

// Module represents the demo module
type Module struct{}

// RegisterRoutes implements the core.ModuleRegistrar interface
func (m *Module) RegisterRoutes(r *mux.Router) {

	router.Router(r, "/demo").
		// Success responses
		Get("/success", getSuccess).
		Get("/created", getCreated).
		Get("/data", getDataWithPayload).
		// Client error responses
		Get("/bad-request", getBadRequest).
		Get("/unauthorized", getUnauthorized).
		Get("/forbidden", getForbidden).
		Get("/not-found", getNotFound).
		Get("/method-not-allowed", getMethodNotAllowed).
		Get("/conflict", getConflict).
		Get("/unprocessable", getUnprocessableEntity).
		Get("/rate-limit", getRateLimit).
		// Server error responses
		Get("/internal-error", getInternalError).
		Get("/external-error", getExternalError).
		// Validation errors
		Get("/validation-single", getValidationErrorSingle).
		Get("/validation-multiple", getValidationErrorMultiple).
		// Custom errors
		Get("/custom-error", getCustomError).
		Get("/business-rule", getBusinessRuleViolation).
		// Middleware examples
		Post("/validate", getValidationWithMiddleware).
		Get("/panic", getPanicExample).
		Get("/cors", getCORSExample).
		// Query parameter examples
		Get("/query-params", getQueryParamsExample).
		// JSON body examples
		Post("/json-body", getJSONBodyExample)
}

// init automatically registers this module when the package is imported
func init() {
	core.RegisterModule("demo", &Module{})
}
