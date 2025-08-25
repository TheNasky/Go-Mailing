package demo

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thenasky/go-framework/internal/router"
)

type Controller struct{}

func NewController() *Controller {
	return &Controller{}
}

// ===== Success Response Examples =====

func getSuccess(req *router.Req, res *router.Res) {
	res.Success("Operation completed successfully", map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"status":    "active",
	})
}

func getCreated(req *router.Req, res *router.Res) {
	res.Created("Resource created successfully", map[string]interface{}{
		"id":         "12345",
		"created_at": time.Now().Format(time.RFC3339),
		"type":       "demo_resource",
	})
}

func getDataWithPayload(req *router.Req, res *router.Res) {
	res.Success("Data retrieved successfully", map[string]interface{}{
		"items": []map[string]interface{}{
			{"id": 1, "name": "Item 1", "active": true},
			{"id": 2, "name": "Item 2", "active": false},
			{"id": 3, "name": "Item 3", "active": true},
		},
		"total": 3,
		"page":  1,
	})
}

// ===== Client Error Response Examples =====

func getBadRequest(req *router.Req, res *router.Res) {
	res.BadRequest("Invalid request parameters", map[string]interface{}{
		"missing_fields": []string{"name", "email"},
		"suggestion":     "Please provide all required fields",
	})
}

func getUnauthorized(req *router.Req, res *router.Res) {
	res.Unauthorized("Authentication required", map[string]interface{}{
		"auth_type": "Bearer token",
		"scope":     "read:users",
	})
}

func getForbidden(req *router.Req, res *router.Res) {
	res.Forbidden("Access denied", map[string]interface{}{
		"required_role": "admin",
		"current_role":  "user",
		"resource":      "admin_panel",
	})
}

func getNotFound(req *router.Req, res *router.Res) {
	res.NotFound("Resource not found", map[string]interface{}{
		"resource_id":   "99999",
		"resource_type": "user",
		"suggestion":    "Check if the ID is correct",
	})
}

func getMethodNotAllowed(req *router.Req, res *router.Res) {
	res.MethodNotAllowed("Method not allowed for this endpoint", []string{"GET", "POST"})
}

func getConflict(req *router.Req, res *router.Res) {
	res.Conflict("Resource already exists", map[string]interface{}{
		"conflict_field": "email",
		"conflict_value": "user@example.com",
		"suggestion":     "Try using a different email or reset your password",
	})
}

func getUnprocessableEntity(req *router.Req, res *router.Res) {
	res.UnprocessableEntity("Validation failed", map[string]interface{}{
		"validation_errors": []string{
			"Email format is invalid",
			"Password is too weak",
			"Age must be positive",
		},
	})
}

func getRateLimit(req *router.Req, res *router.Res) {
	res.RateLimit("Too many requests", 300) // Retry after 5 minutes
}

// ===== Server Error Response Examples =====

func getInternalError(req *router.Req, res *router.Res) {
	internalID := fmt.Sprintf("ERR_%d", time.Now().Unix())
	res.InternalError("An unexpected error occurred", internalID, map[string]interface{}{
		"operation": "user_creation",
		"timestamp": time.Now().Format(time.RFC3339),
		"component": "database_service",
	})
}

func getExternalError(req *router.Req, res *router.Res) {
	res.ExternalError("External service temporarily unavailable", map[string]interface{}{
		"service":     "payment_gateway",
		"status_code": 503,
		"retry_after": "5 minutes",
		"error_code":  "SERVICE_UNAVAILABLE",
	})
}

// ===== Validation Error Examples =====

func getValidationErrorSingle(req *router.Req, res *router.Res) {
	res.ValidationErrorSingle("email", "Invalid email format", "invalid-email")
}

func getValidationErrorMultiple(req *router.Req, res *router.Res) {
	validationErrors := []router.ValidationError{
		router.NewValidationError("name", "Name is required"),
		router.NewValidationError("age", "Age must be between 13 and 120", "150"),
		router.NewValidationError("email", "Invalid email format", "test@"),
		router.NewValidationError("password", "Password must be at least 8 characters long", "123"),
	}

	res.ValidationError("Multiple validation errors occurred", validationErrors)
}

// ===== Custom Error Examples =====

func getCustomError(req *router.Req, res *router.Res) {
	res.ErrorWithCode(
		http.StatusBadRequest,
		router.ErrorTypeValidation,
		"CUSTOM_VALIDATION_ERROR",
		"Custom business rule validation failed",
		map[string]interface{}{
			"rule":          "business_hours",
			"current_time":  time.Now().Format("15:04"),
			"allowed_hours": "09:00-17:00",
			"timezone":      "UTC",
		},
	)
}

func getBusinessRuleViolation(req *router.Req, res *router.Res) {
	res.ErrorWithCode(
		http.StatusUnprocessableEntity,
		router.ErrorTypeValidation,
		"BUSINESS_RULE_VIOLATION",
		"Action not allowed in current state",
		map[string]interface{}{
			"current_state":    "pending_approval",
			"requested_action": "publish",
			"allowed_actions":  []string{"approve", "reject", "request_changes"},
			"business_rule":    "content_must_be_approved_before_publishing",
		},
	)
}

// ===== Middleware Examples =====

func getValidationWithMiddleware(req *router.Req, res *router.Res) {
	// This would normally be handled by validation middleware
	// For demo purposes, we'll show what the response would look like

	// Simulate validation failure
	validationErrors := []router.ValidationError{
		router.NewValidationError("email", "Invalid email format", "test"),
		router.NewValidationError("age", "Age must be positive", "-5"),
	}

	res.ValidationError("Validation failed", validationErrors)
}

func getPanicExample(req *router.Req, res *router.Res) {
	// This will trigger the recovery middleware
	panic("This is a test panic to demonstrate recovery middleware")
}

func getCORSExample(req *router.Req, res *router.Res) {
	// Add custom headers to demonstrate CORS
	res.AddHeader("X-Custom-Header", "Demo-Value")
	res.Success("CORS headers applied", map[string]interface{}{
		"cors_enabled": true,
		"origin":       req.GetHeader("Origin"),
		"method":       req.Method,
	})
}

// ===== Query Parameter Examples =====

func getQueryParamsExample(req *router.Req, res *router.Res) {
	name := req.QueryParam("name")
	age := req.QueryInt("age", 0)
	active := req.QueryBool("active", false)

	if name == "" {
		res.BadRequest("Name parameter is required", map[string]interface{}{
			"required_params": []string{"name"},
			"optional_params": []string{"age", "active"},
		})
		return
	}

	res.Success("Query parameters processed successfully", map[string]interface{}{
		"name":   name,
		"age":    age,
		"active": active,
		"params_received": map[string]interface{}{
			"name":   name,
			"age":    age,
			"active": active,
		},
	})
}

// ===== JSON Body Examples =====

func getJSONBodyExample(req *router.Req, res *router.Res) {
	var requestData map[string]interface{}
	if err := req.JSON(&requestData); err != nil {
		res.BadRequest("Invalid JSON body", map[string]string{"error": err.Error()})
		return
	}

	// Validate required fields
	var validationErrors []router.ValidationError

	if name, ok := requestData["name"].(string); !ok || strings.TrimSpace(name) == "" {
		validationErrors = append(validationErrors, router.ValidationError{
			Field:   "name",
			Message: "Name is required",
		})
	}

	if email, ok := requestData["email"].(string); ok && email != "" {
		if !strings.Contains(email, "@") {
			validationErrors = append(validationErrors, router.ValidationError{
				Field:   "email",
				Message: "Invalid email format",
				Value:   email,
			})
		}
	}

	if age, ok := requestData["age"].(float64); ok {
		if age < 0 || age > 150 {
			validationErrors = append(validationErrors, router.ValidationError{
				Field:   "age",
				Message: "Age must be between 0 and 150",
				Value:   strconv.FormatFloat(age, 'f', 0, 64),
			})
		}
	}

	if len(validationErrors) > 0 {
		res.ValidationError("JSON validation failed", validationErrors)
		return
	}

	// Success case
	res.Created("JSON data processed successfully", map[string]interface{}{
		"received_data": requestData,
		"processed_at":  time.Now().Format(time.RFC3339),
		"validation":    "passed",
	})
}
