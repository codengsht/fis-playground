package models

import (
	"errors"
	"strings"
	"time"
)

// Item represents an item in the FIS playground system
type Item struct {
	ID          string    `json:"id" dynamodbav:"id"`
	Name        string    `json:"name" dynamodbav:"name"`
	Description string    `json:"description" dynamodbav:"description"`
	CreatedAt   time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" dynamodbav:"updated_at"`
	Status      string    `json:"status" dynamodbav:"status"`
}

// CreateItemRequest represents the request payload for creating an item
type CreateItemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateItemRequest represents the request payload for updating an item
type UpdateItemRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

// APIResponse represents the standard API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo provides detailed error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Details string `json:"details,omitempty"`
}

// Validation errors
var (
	ErrEmptyName          = errors.New("name cannot be empty")
	ErrEmptyDescription   = errors.New("description cannot be empty")
	ErrInvalidStatus      = errors.New("status must be one of: active, inactive, pending")
	ErrNameTooLong        = errors.New("name cannot exceed 100 characters")
	ErrDescriptionTooLong = errors.New("description cannot exceed 500 characters")
)

// Validate validates a CreateItemRequest
func (r *CreateItemRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return ErrEmptyName
	}
	if len(r.Name) > 100 {
		return ErrNameTooLong
	}
	if strings.TrimSpace(r.Description) == "" {
		return ErrEmptyDescription
	}
	if len(r.Description) > 500 {
		return ErrDescriptionTooLong
	}
	return nil
}

// Validate validates an UpdateItemRequest
func (r *UpdateItemRequest) Validate() error {
	// For updates, fields are optional but if provided must be valid
	if r.Name != "" {
		if strings.TrimSpace(r.Name) == "" {
			return ErrEmptyName
		}
		if len(r.Name) > 100 {
			return ErrNameTooLong
		}
	}
	if r.Description != "" {
		if strings.TrimSpace(r.Description) == "" {
			return ErrEmptyDescription
		}
		if len(r.Description) > 500 {
			return ErrDescriptionTooLong
		}
	}
	if r.Status != "" {
		if !isValidStatus(r.Status) {
			return ErrInvalidStatus
		}
	}
	return nil
}

// Validate validates a complete Item struct
func (i *Item) Validate() error {
	if strings.TrimSpace(i.Name) == "" {
		return ErrEmptyName
	}
	if len(i.Name) > 100 {
		return ErrNameTooLong
	}
	if strings.TrimSpace(i.Description) == "" {
		return ErrEmptyDescription
	}
	if len(i.Description) > 500 {
		return ErrDescriptionTooLong
	}
	if !isValidStatus(i.Status) {
		return ErrInvalidStatus
	}
	return nil
}

// isValidStatus checks if the status is one of the allowed values
func isValidStatus(status string) bool {
	validStatuses := []string{"active", "inactive", "pending"}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

// NewItem creates a new Item with default values
func NewItem(name, description string) *Item {
	now := time.Now()
	return &Item{
		Name:        name,
		Description: description,
		Status:      "active", // default status
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// UpdateFields updates the item with new values from UpdateItemRequest
func (i *Item) UpdateFields(req *UpdateItemRequest) {
	if req.Name != "" {
		i.Name = req.Name
	}
	if req.Description != "" {
		i.Description = req.Description
	}
	if req.Status != "" {
		i.Status = req.Status
	}
	i.UpdatedAt = time.Now()
}
