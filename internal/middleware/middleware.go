package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/thenasky/go-framework/internal/router"
)

// ValidationRule represents a validation rule for a field
type ValidationRule struct {
	Field    string
	Required bool
	Min      int
	Max      int
	Pattern  string
	Custom   func(value interface{}) error
}

// ValidationMiddleware provides request validation
type ValidationMiddleware struct {
	Rules map[string][]ValidationRule // endpoint -> rules
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware() *ValidationMiddleware {
	return &ValidationMiddleware{
		Rules: make(map[string][]ValidationRule),
	}
}

// AddRule adds a validation rule for an endpoint
func (vm *ValidationMiddleware) AddRule(endpoint string, rules []ValidationRule) {
	vm.Rules[endpoint] = rules
}

// Validate validates the request based on the defined rules
func (vm *ValidationMiddleware) Validate(endpoint string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rules, exists := vm.Rules[endpoint]
			if !exists {
				next(w, r)
				return
			}

			// Parse request body if it's JSON
			var body map[string]interface{}
			if r.Header.Get("Content-Type") == "application/json" {
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					res := router.NewResponse(w)
					res.BadRequest("Invalid JSON body", map[string]string{"error": err.Error()})
					return
				}
			}

			// Parse query parameters
			query := r.URL.Query()

			// Validate according to rules
			var validationErrors []router.ValidationError
			for _, rule := range rules {
				if err := vm.validateField(rule, body, query); err != nil {
					validationErrors = append(validationErrors, router.ValidationError{
						Field:   rule.Field,
						Message: err.Error(),
					})
				}
			}

			// If validation failed, return error
			if len(validationErrors) > 0 {
				res := router.NewResponse(w)
				res.ValidationError("Validation failed", validationErrors)
				return
			}

			// Validation passed, continue to next handler
			next(w, r)
		}
	}
}

// validateField validates a single field according to its rules
func (vm *ValidationMiddleware) validateField(rule ValidationRule, body map[string]interface{}, query map[string][]string) error {
	// Check if field exists in body or query
	var value interface{}
	var exists bool

	// Check body first, then query parameters
	if body != nil {
		value, exists = body[rule.Field]
	}
	if !exists && query != nil {
		if queryValues, ok := query[rule.Field]; ok && len(queryValues) > 0 {
			value = queryValues[0]
			exists = true
		}
	}

	// Required field check
	if rule.Required && !exists {
		return fmt.Errorf("Field '%s' is required", rule.Field)
	}

	// If field doesn't exist and isn't required, skip validation
	if !exists {
		return nil
	}

	// Type-specific validation
	if err := vm.validateValue(rule, value); err != nil {
		return err
	}

	// Custom validation
	if rule.Custom != nil {
		if err := rule.Custom(value); err != nil {
			return err
		}
	}

	return nil
}

// validateValue performs type-specific validation
func (vm *ValidationMiddleware) validateValue(rule ValidationRule, value interface{}) error {
	if value == nil {
		return nil
	}

	// String validation
	if str, ok := value.(string); ok {
		if rule.Min > 0 && len(str) < rule.Min {
			return fmt.Errorf("Field '%s' must be at least %d characters long", rule.Field, rule.Min)
		}
		if rule.Max > 0 && len(str) > rule.Max {
			return fmt.Errorf("Field '%s' must be no more than %d characters long", rule.Field, rule.Max)
		}
	}

	// Number validation (int/float)
	if num, ok := value.(float64); ok {
		if rule.Min > 0 && num < float64(rule.Min) {
			return fmt.Errorf("Field '%s' must be at least %d", rule.Field, rule.Min)
		}
		if rule.Max > 0 && num > float64(rule.Max) {
			return fmt.Errorf("Field '%s' must be no more than %d", rule.Field, rule.Max)
		}
	}

	return nil
}

// ===== Common Validation Rules =====

// Required creates a required field rule
func Required(field string) ValidationRule {
	return ValidationRule{
		Field:    field,
		Required: true,
	}
}

// MinLength creates a minimum length rule for strings
func MinLength(field string, min int) ValidationRule {
	return ValidationRule{
		Field: field,
		Min:   min,
	}
}

// MaxLength creates a maximum length rule for strings
func MaxLength(field string, max int) ValidationRule {
	return ValidationRule{
		Field: field,
		Max:   max,
	}
}

// Range creates a numeric range rule
func Range(field string, min, max int) ValidationRule {
	return ValidationRule{
		Field: field,
		Min:   min,
		Max:   max,
	}
}

// Custom creates a custom validation rule
func Custom(field string, validator func(value interface{}) error) ValidationRule {
	return ValidationRule{
		Field:  field,
		Custom: validator,
	}
}

// ===== Error Recovery Middleware =====

// RecoveryMiddleware recovers from panics and returns proper error responses
func RecoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic (you might want to use your logger here)
				// logger.LogError(fmt.Sprintf("Panic recovered: %v", err))

				// Generate a unique ID for tracking
				internalID := generateInternalID()

				// Return a proper error response
				res := router.NewResponse(w)
				res.InternalError(
					"An unexpected error occurred",
					internalID,
					map[string]interface{}{
						"error": fmt.Sprintf("%v", err),
					},
				)
			}
		}()

		next(w, r)
	}
}

// generateInternalID generates a simple internal ID for error tracking
func generateInternalID() string {
	return fmt.Sprintf("ERR_%d", time.Now().Unix())
}

// ===== CORS Middleware =====

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORSMiddleware adds CORS headers to responses
func CORSMiddleware(config *CORSConfig) func(http.HandlerFunc) http.HandlerFunc {
	if config == nil {
		config = DefaultCORSConfig()
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			if len(config.AllowedOrigins) > 0 {
				origin := r.Header.Get("Origin")
				if origin != "" {
					allowed := false
					for _, allowedOrigin := range config.AllowedOrigins {
						if allowedOrigin == "*" || allowedOrigin == origin {
							w.Header().Set("Access-Control-Allow-Origin", origin)
							allowed = true
							break
						}
					}
					if !allowed {
						w.Header().Set("Access-Control-Allow-Origin", config.AllowedOrigins[0])
					}
				}
			}

			if len(config.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			}

			if len(config.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}

			// Handle preflight request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}
}
