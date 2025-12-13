package repository

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Common repository errors
var (
	ErrItemNotFound      = errors.New("item not found")
	ErrItemAlreadyExists = errors.New("item already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrConnectionFailed  = errors.New("database connection failed")
	ErrOperationFailed   = errors.New("database operation failed")
)

// HandleDynamoDBError converts DynamoDB-specific errors to repository errors
func HandleDynamoDBError(err error) error {
	if err == nil {
		return nil
	}

	// Handle specific DynamoDB error types
	var resourceNotFound *types.ResourceNotFoundException
	var conditionalCheckFailed *types.ConditionalCheckFailedException
	var provisionedThroughputExceeded *types.ProvisionedThroughputExceededException
	var resourceInUse *types.ResourceInUseException
	var internalServerError *types.InternalServerError

	switch {
	case errors.As(err, &resourceNotFound):
		return fmt.Errorf("%w: %s", ErrItemNotFound, err.Error())
	case errors.As(err, &conditionalCheckFailed):
		return fmt.Errorf("%w: conditional check failed", ErrItemAlreadyExists)
	case errors.As(err, &provisionedThroughputExceeded):
		return fmt.Errorf("%w: throughput exceeded", ErrOperationFailed)
	case errors.As(err, &resourceInUse):
		return fmt.Errorf("%w: resource in use", ErrOperationFailed)
	case errors.As(err, &internalServerError):
		return fmt.Errorf("%w: internal server error", ErrOperationFailed)
	default:
		// For validation and other errors, check error message patterns
		errMsg := err.Error()
		if containsValidationError(errMsg) {
			return fmt.Errorf("%w: %s", ErrInvalidInput, errMsg)
		}
		return fmt.Errorf("%w: %s", ErrOperationFailed, errMsg)
	}
}

// containsValidationError checks if error message indicates validation failure
func containsValidationError(errMsg string) bool {
	validationKeywords := []string{
		"ValidationException",
		"validation",
		"invalid",
		"malformed",
		"bad request",
	}

	for _, keyword := range validationKeywords {
		if len(errMsg) > 0 && contains(errMsg, keyword) {
			return true
		}
	}
	return false
}

// contains is a simple string contains check (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

// containsSubstring performs a simple substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	var provisionedThroughputExceeded *types.ProvisionedThroughputExceededException
	var internalServerError *types.InternalServerError

	return errors.As(err, &provisionedThroughputExceeded) ||
		errors.As(err, &internalServerError)
}

// IsNotFoundError checks if the error indicates an item was not found
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrItemNotFound)
}

// IsConflictError checks if the error indicates a conflict (item already exists)
func IsConflictError(err error) bool {
	return errors.Is(err, ErrItemAlreadyExists)
}

// IsValidationError checks if the error indicates invalid input
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsConnectionError checks if the error indicates a connection failure
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnectionFailed)
}

// IsOperationError checks if the error indicates a general operation failure
func IsOperationError(err error) bool {
	return errors.Is(err, ErrOperationFailed)
}
