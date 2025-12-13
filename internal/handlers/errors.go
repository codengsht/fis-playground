package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"fis-playground/internal/models"
	"fis-playground/internal/repository"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeConflict   ErrorType = "conflict"
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeSystem     ErrorType = "system"
	ErrorTypeAuth       ErrorType = "authentication"
	ErrorTypeRate       ErrorType = "rate_limit"
)

// ErrorCode represents specific error codes for different scenarios
type ErrorCode string

const (
	// Validation errors
	CodeInvalidRequest     ErrorCode = "INVALID_REQUEST"
	CodeMissingField       ErrorCode = "MISSING_FIELD"
	CodeInvalidFormat      ErrorCode = "INVALID_FORMAT"
	CodeValueTooLong       ErrorCode = "VALUE_TOO_LONG"
	CodeInvalidValue       ErrorCode = "INVALID_VALUE"

	// Resource errors
	CodeNotFound           ErrorCode = "NOT_FOUND"
	CodeAlreadyExists      ErrorCode = "ALREADY_EXISTS"

	// Database errors
	CodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	CodeConnectionError    ErrorCode = "CONNECTION_ERROR"
	CodeOperationFailed    ErrorCode = "OPERATION_FAILED"
	CodeThroughputExceeded ErrorCode = "THROUGHPUT_EXCEEDED"

	// System errors
	CodeInternalError      ErrorCode = "INTERNAL_ERROR"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	CodeTimeout            ErrorCode = "TIMEOUT"

	// Authentication errors
	CodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	CodeForbidden          ErrorCode = "FORBIDDEN"

	// Rate limiting errors
	CodeRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
)

// APIError represents a structured error with HTTP status code mapping
type APIError struct {
	Type       ErrorType
	Code       ErrorCode
	Message    string
	Details    string
	StatusCode int
	Cause      error
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(code ErrorCode, message string, details ...string) *APIError {
	detail := ""
	if len(details) > 0 {
		detail = details[0]
	}
	return &APIError{
		Type:       ErrorTypeValidation,
		Code:       code,
		Message:    message,
		Details:    detail,
		StatusCode: http.StatusBadRequest,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource string, id string) *APIError {
	return &APIError{
		Type:       ErrorTypeNotFound,
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		Details:    fmt.Sprintf("No %s found with ID: %s", resource, id),
		StatusCode: http.StatusNotFound,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(resource string, details string) *APIError {
	return &APIError{
		Type:       ErrorTypeConflict,
		Code:       CodeAlreadyExists,
		Message:    fmt.Sprintf("%s already exists", resource),
		Details:    details,
		StatusCode: http.StatusConflict,
	}
}

// NewDatabaseError creates a new database error
func NewDatabaseError(code ErrorCode, message string, cause error) *APIError {
	statusCode := http.StatusInternalServerError
	if code == CodeThroughputExceeded {
		statusCode = http.StatusTooManyRequests
	}
	
	return &APIError{
		Type:       ErrorTypeDatabase,
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Cause:      cause,
	}
}

// NewSystemError creates a new system error
func NewSystemError(code ErrorCode, message string, cause error) *APIError {
	statusCode := http.StatusInternalServerError
	if code == CodeServiceUnavailable {
		statusCode = http.StatusServiceUnavailable
	} else if code == CodeTimeout {
		statusCode = http.StatusRequestTimeout
	}
	
	return &APIError{
		Type:       ErrorTypeSystem,
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Cause:      cause,
	}
}

// MapRepositoryError converts repository errors to API errors
func MapRepositoryError(err error) *APIError {
	if err == nil {
		return nil
	}

	switch {
	case repository.IsNotFoundError(err):
		return &APIError{
			Type:       ErrorTypeNotFound,
			Code:       CodeNotFound,
			Message:    "Resource not found",
			Details:    err.Error(),
			StatusCode: http.StatusNotFound,
			Cause:      err,
		}
	case repository.IsConflictError(err):
		return &APIError{
			Type:       ErrorTypeConflict,
			Code:       CodeAlreadyExists,
			Message:    "Resource already exists",
			Details:    err.Error(),
			StatusCode: http.StatusConflict,
			Cause:      err,
		}
	case repository.IsValidationError(err):
		return &APIError{
			Type:       ErrorTypeValidation,
			Code:       CodeInvalidRequest,
			Message:    "Invalid input provided",
			Details:    err.Error(),
			StatusCode: http.StatusBadRequest,
			Cause:      err,
		}
	case repository.IsConnectionError(err):
		return &APIError{
			Type:       ErrorTypeDatabase,
			Code:       CodeConnectionError,
			Message:    "Database connection failed",
			Details:    "Unable to connect to the database",
			StatusCode: http.StatusServiceUnavailable,
			Cause:      err,
		}
	case repository.IsOperationError(err):
		// Check if it's a throughput error based on error message
		if containsThroughputError(err.Error()) {
			return &APIError{
				Type:       ErrorTypeDatabase,
				Code:       CodeThroughputExceeded,
				Message:    "Database throughput exceeded",
				Details:    "Please retry your request after a brief delay",
				StatusCode: http.StatusTooManyRequests,
				Cause:      err,
			}
		}
		return &APIError{
			Type:       ErrorTypeDatabase,
			Code:       CodeDatabaseError,
			Message:    "Database operation failed",
			Details:    err.Error(),
			StatusCode: http.StatusInternalServerError,
			Cause:      err,
		}
	default:
		return &APIError{
			Type:       ErrorTypeSystem,
			Code:       CodeInternalError,
			Message:    "An unexpected error occurred",
			Details:    "Please try again later",
			StatusCode: http.StatusInternalServerError,
			Cause:      err,
		}
	}
}

// MapValidationError converts validation errors to API errors
func MapValidationError(err error) *APIError {
	if err == nil {
		return nil
	}

	message := err.Error()
	code := CodeInvalidRequest

	// Map specific validation errors to appropriate codes
	switch {
	case containsError(message, "empty", "required"):
		code = CodeMissingField
	case containsError(message, "too long", "exceed"):
		code = CodeValueTooLong
	case containsError(message, "invalid", "format"):
		code = CodeInvalidFormat
	case containsError(message, "status", "one of"):
		code = CodeInvalidValue
	}

	return &APIError{
		Type:       ErrorTypeValidation,
		Code:       code,
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Cause:      err,
	}
}

// WriteErrorResponse writes a standardized error response
func WriteErrorResponse(w http.ResponseWriter, r *http.Request, apiErr *APIError) {
	// Log the error for debugging
	if apiErr.Cause != nil {
		log.Printf("API Error [%s %s]: %s - %s (caused by: %v)", 
			r.Method, r.URL.Path, apiErr.Code, apiErr.Message, apiErr.Cause)
	} else {
		log.Printf("API Error [%s %s]: %s - %s", 
			r.Method, r.URL.Path, apiErr.Code, apiErr.Message)
	}

	// Create error info
	errorInfo := &models.ErrorInfo{
		Code:    string(apiErr.Code),
		Message: apiErr.Message,
		Type:    string(apiErr.Type),
		Details: apiErr.Details,
	}

	// Create response
	response := models.APIResponse{
		Success: false,
		Error:   errorInfo,
	}

	// Write response
	writeJSONResponse(w, apiErr.StatusCode, response)
}

// WriteValidationErrorResponse writes a validation error response
func WriteValidationErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := MapValidationError(err)
	WriteErrorResponse(w, r, apiErr)
}

// WriteRepositoryErrorResponse writes a repository error response
func WriteRepositoryErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := MapRepositoryError(err)
	WriteErrorResponse(w, r, apiErr)
}

// WriteJSONParseErrorResponse writes a JSON parsing error response
func WriteJSONParseErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := &APIError{
		Type:       ErrorTypeValidation,
		Code:       CodeInvalidFormat,
		Message:    "Invalid JSON format",
		Details:    "Request body contains malformed JSON",
		StatusCode: http.StatusBadRequest,
		Cause:      err,
	}
	WriteErrorResponse(w, r, apiErr)
}

// WriteMissingParameterErrorResponse writes a missing parameter error response
func WriteMissingParameterErrorResponse(w http.ResponseWriter, r *http.Request, parameter string) {
	apiErr := &APIError{
		Type:       ErrorTypeValidation,
		Code:       CodeMissingField,
		Message:    fmt.Sprintf("%s is required", parameter),
		Details:    fmt.Sprintf("The required parameter '%s' is missing from the request", parameter),
		StatusCode: http.StatusBadRequest,
	}
	WriteErrorResponse(w, r, apiErr)
}

// WriteInternalErrorResponse writes an internal server error response
func WriteInternalErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := &APIError{
		Type:       ErrorTypeSystem,
		Code:       CodeInternalError,
		Message:    "Internal server error",
		Details:    "An unexpected error occurred. Please try again later.",
		StatusCode: http.StatusInternalServerError,
		Cause:      err,
	}
	WriteErrorResponse(w, r, apiErr)
}

// Helper functions

// containsThroughputError checks if error message indicates throughput exceeded
func containsThroughputError(errMsg string) bool {
	keywords := []string{
		"throughput",
		"capacity",
		"throttling",
		"rate",
	}
	return containsAnyKeyword(errMsg, keywords)
}

// containsError checks if error message contains any of the specified keywords
func containsError(errMsg string, keywords ...string) bool {
	return containsAnyKeyword(errMsg, keywords)
}

// containsAnyKeyword checks if the message contains any of the keywords (case-insensitive)
func containsAnyKeyword(message string, keywords []string) bool {
	msgLower := toLower(message)
	for _, keyword := range keywords {
		if contains(msgLower, toLower(keyword)) {
			return true
		}
	}
	return false
}

// toLower converts string to lowercase (simple implementation)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}

// contains checks if string contains substring (case-sensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

// findSubstring finds the index of substring in string, returns -1 if not found
func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// writeJSONResponse writes a JSON response to the HTTP response writer
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		// Fallback to plain text error
		http.Error(w, `{"success": false, "error": {"code": "INTERNAL_ERROR", "message": "Failed to encode response", "type": "system"}}`, http.StatusInternalServerError)
	}
}