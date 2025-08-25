package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ErrorType represents the type of error that occurred
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeForbidden    ErrorType = "forbidden"
	ErrorTypeConflict     ErrorType = "conflict"
	ErrorTypeRateLimit    ErrorType = "rate_limit"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeExternal     ErrorType = "external"
)

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// APIError represents a detailed API error
type APIError struct {
	Type       ErrorType         `json:"type"`
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Details    interface{}       `json:"details,omitempty"`
	Validation []ValidationError `json:"validation,omitempty"`
	InternalID string            `json:"internal_id,omitempty"` // For debugging/tracking
}

// StandardResponse represents the standardized API response structure
type StandardResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Payload interface{} `json:"payload,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// Response provides methods for building standardized responses (like Express.js res)
type Response struct {
	writer http.ResponseWriter
}

// NewResponse creates a new response wrapper
func NewResponse(w http.ResponseWriter) *Response {
	return &Response{writer: w}
}

// Success sends a successful response (200)
func (res *Response) Success(message string, payload interface{}) {
	res.sendResponse(http.StatusOK, "success", message, payload, nil)
}

// Created sends a created response (201)
func (res *Response) Created(message string, payload interface{}) {
	res.sendResponse(http.StatusCreated, "success", message, payload, nil)
}

// Fail sends a client error response (400)
func (res *Response) Fail(message string, payload interface{}) {
	res.sendResponse(http.StatusBadRequest, "fail", message, payload, nil)
}

// Unauthorized sends an unauthorized response (401)
func (res *Response) Unauthorized(message string, payload interface{}) {
	res.sendResponse(http.StatusUnauthorized, "fail", message, payload, nil)
}

// Forbidden sends a forbidden response (403)
func (res *Response) Forbidden(message string, payload interface{}) {
	res.sendResponse(http.StatusForbidden, "fail", message, payload, nil)
}

// NotFound sends a not found response (404)
func (res *Response) NotFound(message string, payload interface{}) {
	res.sendResponse(http.StatusNotFound, "fail", message, payload, nil)
}

// Error sends a server error response (500)
func (res *Response) Error(message string, payload interface{}) {
	res.sendResponse(http.StatusInternalServerError, "error", message, payload, nil)
}

// Custom allows sending a response with custom status code
func (res *Response) Custom(statusCode int, status, message string, payload interface{}) {
	res.sendResponse(statusCode, status, message, payload, nil)
}

// ===== Enhanced Error Handling Methods =====

// ValidationError sends a validation error response (422)
func (res *Response) ValidationError(message string, validationErrors []ValidationError) {
	apiError := &APIError{
		Type:       ErrorTypeValidation,
		Code:       "VALIDATION_ERROR",
		Message:    message,
		Validation: validationErrors,
	}
	res.sendResponse(http.StatusUnprocessableEntity, "fail", message, nil, apiError)
}

// ValidationErrorSingle sends a single validation error response (422)
func (res *Response) ValidationErrorSingle(field, message string, value ...string) {
	var val string
	if len(value) > 0 {
		val = value[0]
	}

	validationError := ValidationError{
		Field:   field,
		Message: message,
		Value:   val,
	}

	res.ValidationError("Validation failed", []ValidationError{validationError})
}

// Conflict sends a conflict error response (409)
func (res *Response) Conflict(message string, details interface{}) {
	apiError := &APIError{
		Type:    ErrorTypeConflict,
		Code:    "CONFLICT",
		Message: message,
		Details: details,
	}
	res.sendResponse(http.StatusConflict, "fail", message, nil, apiError)
}

// RateLimit sends a rate limit error response (429)
func (res *Response) RateLimit(message string, retryAfter int) {
	apiError := &APIError{
		Type:    ErrorTypeRateLimit,
		Code:    "RATE_LIMIT",
		Message: message,
		Details: map[string]interface{}{
			"retry_after": retryAfter,
		},
	}

	res.writer.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
	res.sendResponse(http.StatusTooManyRequests, "fail", message, nil, apiError)
}

// ExternalError sends an external service error response (502)
func (res *Response) ExternalError(message string, details interface{}) {
	apiError := &APIError{
		Type:    ErrorTypeExternal,
		Code:    "EXTERNAL_SERVICE_ERROR",
		Message: message,
		Details: details,
	}
	res.sendResponse(http.StatusBadGateway, "fail", message, nil, apiError)
}

// InternalError sends an internal server error with optional internal ID for tracking
func (res *Response) InternalError(message string, internalID string, details interface{}) {
	apiError := &APIError{
		Type:       ErrorTypeInternal,
		Code:       "INTERNAL_ERROR",
		Message:    message,
		Details:    details,
		InternalID: internalID,
	}
	res.sendResponse(http.StatusInternalServerError, "error", message, nil, apiError)
}

// ErrorWithCode sends an error response with custom error code and type
func (res *Response) ErrorWithCode(statusCode int, errorType ErrorType, code, message string, details interface{}) {
	apiError := &APIError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Details: details,
	}
	res.sendResponse(statusCode, "fail", message, nil, apiError)
}

// ===== Helper Methods for Common Error Patterns =====

// BadRequest sends a bad request error (400)
func (res *Response) BadRequest(message string, details interface{}) {
	res.ErrorWithCode(http.StatusBadRequest, ErrorTypeValidation, "BAD_REQUEST", message, details)
}

// UnprocessableEntity sends an unprocessable entity error (422)
func (res *Response) UnprocessableEntity(message string, details interface{}) {
	res.ErrorWithCode(http.StatusUnprocessableEntity, ErrorTypeValidation, "UNPROCESSABLE_ENTITY", message, details)
}

// MethodNotAllowed sends a method not allowed error (405)
func (res *Response) MethodNotAllowed(message string, allowedMethods []string) {
	details := map[string]interface{}{
		"allowed_methods": allowedMethods,
	}
	res.ErrorWithCode(http.StatusMethodNotAllowed, ErrorTypeValidation, "METHOD_NOT_ALLOWED", message, details)
}

// ===== Utility Methods =====

// AddHeader adds a custom header to the response
func (res *Response) AddHeader(key, value string) {
	res.writer.Header().Set(key, value)
}

// SetContentType sets the content type header
func (res *Response) SetContentType(contentType string) {
	res.writer.Header().Set("Content-Type", contentType)
}

// Redirect sends a redirect response
func (res *Response) Redirect(statusCode int, url string) {
	res.writer.Header().Set("Location", url)
	res.writer.WriteHeader(statusCode)
}

// sendResponse is the internal method that actually sends the response
func (res *Response) sendResponse(statusCode int, status, message string, payload interface{}, apiError *APIError) {
	response := StandardResponse{
		Status:  status,
		Message: message,
		Payload: payload,
		Error:   apiError,
	}

	res.writer.Header().Set("Content-Type", "application/json")
	res.writer.WriteHeader(statusCode)

	if err := json.NewEncoder(res.writer).Encode(response); err != nil {
		// Fallback to basic error response if JSON encoding fails
		res.writer.WriteHeader(http.StatusInternalServerError)
		res.writer.Write([]byte(`{"status":"error","message":"Failed to encode response"}`))
	}
}

// ===== Error Creation Helpers =====

// NewValidationError creates a new validation error
func NewValidationError(field, message string, value ...string) ValidationError {
	var val string
	if len(value) > 0 {
		val = value[0]
	}

	return ValidationError{
		Field:   field,
		Message: message,
		Value:   val,
	}
}

// NewAPIError creates a new API error
func NewAPIError(errorType ErrorType, code, message string, details interface{}) *APIError {
	return &APIError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Details: details,
	}
}

// IsValidationError checks if the given error is a validation error
func IsValidationError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "validation")
}
