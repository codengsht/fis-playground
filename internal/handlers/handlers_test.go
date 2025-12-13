package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"fis-playground/internal/models"
	"fis-playground/internal/repository"
)

// MockRepository is a simple mock for testing
type MockRepository struct {
	ShouldReturnError error
}

func (m *MockRepository) CreateItem(ctx context.Context, item *models.Item) error {
	if m.ShouldReturnError != nil {
		return m.ShouldReturnError
	}
	item.ID = "test-id"
	return nil
}

func (m *MockRepository) GetItem(ctx context.Context, id string) (*models.Item, error) {
	if m.ShouldReturnError != nil {
		return nil, m.ShouldReturnError
	}
	return &models.Item{
		ID:          id,
		Name:        "Test Item",
		Description: "Test Description",
		Status:      "active",
	}, nil
}

func (m *MockRepository) ListItems(ctx context.Context, options *repository.ListItemsOptions) (*repository.ListItemsResult, error) {
	if m.ShouldReturnError != nil {
		return nil, m.ShouldReturnError
	}
	return &repository.ListItemsResult{
		Items: []models.Item{
			{
				ID:          "1",
				Name:        "Item 1",
				Description: "Description 1",
				Status:      "active",
			},
		},
		HasMore: false,
	}, nil
}

func (m *MockRepository) UpdateItem(ctx context.Context, id string, updates *models.UpdateItemRequest) (*models.Item, error) {
	if m.ShouldReturnError != nil {
		return nil, m.ShouldReturnError
	}
	return &models.Item{
		ID:          id,
		Name:        "Updated Item",
		Description: "Updated Description",
		Status:      "active",
	}, nil
}

func (m *MockRepository) DeleteItem(ctx context.Context, id string) error {
	return m.ShouldReturnError
}

func TestHealthCheck(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestCreateItem(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	createReq := models.CreateItemRequest{
		Name:        "Test Item",
		Description: "Test Description",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateItem(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestGetItem(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	req := httptest.NewRequest("GET", "/items/test-id", nil)
	w := httptest.NewRecorder()

	// Set up Chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetItem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
}

// Test error handling scenarios

func TestCreateItem_ValidationError(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	// Test with empty name
	createReq := models.CreateItemRequest{
		Name:        "",
		Description: "Test Description",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error == nil {
		t.Error("Expected error to be present")
	}

	if response.Error.Type != "validation" {
		t.Errorf("Expected error type 'validation', got '%s'", response.Error.Type)
	}

	if response.Error.Code != "MISSING_FIELD" {
		t.Errorf("Expected error code 'MISSING_FIELD', got '%s'", response.Error.Code)
	}
}

func TestCreateItem_InvalidJSON(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	req := httptest.NewRequest("POST", "/items", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Type != "validation" {
		t.Errorf("Expected error type 'validation', got '%s'", response.Error.Type)
	}

	if response.Error.Code != "INVALID_FORMAT" {
		t.Errorf("Expected error code 'INVALID_FORMAT', got '%s'", response.Error.Code)
	}
}

func TestCreateItem_RepositoryError(t *testing.T) {
	mockRepo := &MockRepository{
		ShouldReturnError: repository.ErrItemAlreadyExists,
	}
	handler := NewItemHandler(mockRepo)

	createReq := models.CreateItemRequest{
		Name:        "Test Item",
		Description: "Test Description",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateItem(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Type != "conflict" {
		t.Errorf("Expected error type 'conflict', got '%s'", response.Error.Type)
	}
}

func TestGetItem_MissingID(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	req := httptest.NewRequest("GET", "/items/", nil)
	w := httptest.NewRecorder()

	// Set up Chi URL params with empty ID
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Code != "MISSING_FIELD" {
		t.Errorf("Expected error code 'MISSING_FIELD', got '%s'", response.Error.Code)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	mockRepo := &MockRepository{
		ShouldReturnError: repository.ErrItemNotFound,
	}
	handler := NewItemHandler(mockRepo)

	req := httptest.NewRequest("GET", "/items/nonexistent", nil)
	w := httptest.NewRecorder()

	// Set up Chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetItem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Type != "not_found" {
		t.Errorf("Expected error type 'not_found', got '%s'", response.Error.Type)
	}
}

func TestListItems_InvalidLimit(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	req := httptest.NewRequest("GET", "/items?limit=invalid", nil)
	w := httptest.NewRecorder()

	handler.ListItems(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Code != "INVALID_FORMAT" {
		t.Errorf("Expected error code 'INVALID_FORMAT', got '%s'", response.Error.Code)
	}
}

func TestListItems_LimitOutOfRange(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	req := httptest.NewRequest("GET", "/items?limit=200", nil)
	w := httptest.NewRecorder()

	handler.ListItems(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Code != "INVALID_VALUE" {
		t.Errorf("Expected error code 'INVALID_VALUE', got '%s'", response.Error.Code)
	}
}

func TestUpdateItem_ValidationError(t *testing.T) {
	handler := NewItemHandler(&MockRepository{})

	// Test with invalid status
	updateReq := models.UpdateItemRequest{
		Status: "invalid_status",
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/items/test-id", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up Chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.UpdateItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Type != "validation" {
		t.Errorf("Expected error type 'validation', got '%s'", response.Error.Type)
	}
}

func TestDeleteItem_NotFound(t *testing.T) {
	mockRepo := &MockRepository{
		ShouldReturnError: repository.ErrItemNotFound,
	}
	handler := NewItemHandler(mockRepo)

	req := httptest.NewRequest("DELETE", "/items/nonexistent", nil)
	w := httptest.NewRecorder()

	// Set up Chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.DeleteItem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Type != "not_found" {
		t.Errorf("Expected error type 'not_found', got '%s'", response.Error.Type)
	}
}

// Test error mapping functions

func TestMapRepositoryError(t *testing.T) {
	tests := []struct {
		name           string
		inputError     error
		expectedType   ErrorType
		expectedCode   ErrorCode
		expectedStatus int
	}{
		{
			name:           "Not found error",
			inputError:     repository.ErrItemNotFound,
			expectedType:   ErrorTypeNotFound,
			expectedCode:   CodeNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Conflict error",
			inputError:     repository.ErrItemAlreadyExists,
			expectedType:   ErrorTypeConflict,
			expectedCode:   CodeAlreadyExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "Validation error",
			inputError:     repository.ErrInvalidInput,
			expectedType:   ErrorTypeValidation,
			expectedCode:   CodeInvalidRequest,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Connection error",
			inputError:     repository.ErrConnectionFailed,
			expectedType:   ErrorTypeDatabase,
			expectedCode:   CodeConnectionError,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Operation error",
			inputError:     repository.ErrOperationFailed,
			expectedType:   ErrorTypeDatabase,
			expectedCode:   CodeDatabaseError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Unknown error",
			inputError:     errors.New("unknown error"),
			expectedType:   ErrorTypeSystem,
			expectedCode:   CodeInternalError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := MapRepositoryError(tt.inputError)

			if apiErr.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, apiErr.Type)
			}

			if apiErr.Code != tt.expectedCode {
				t.Errorf("Expected code %s, got %s", tt.expectedCode, apiErr.Code)
			}

			if apiErr.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, apiErr.StatusCode)
			}
		})
	}
}

func TestMapValidationError(t *testing.T) {
	tests := []struct {
		name         string
		inputError   error
		expectedCode ErrorCode
	}{
		{
			name:         "Empty field error",
			inputError:   errors.New("name cannot be empty"),
			expectedCode: CodeMissingField,
		},
		{
			name:         "Too long error",
			inputError:   errors.New("name cannot exceed 100 characters"),
			expectedCode: CodeValueTooLong,
		},
		{
			name:         "Invalid status error",
			inputError:   errors.New("status must be one of: active, inactive, pending"),
			expectedCode: CodeInvalidValue,
		},
		{
			name:         "Generic validation error",
			inputError:   errors.New("some validation error"),
			expectedCode: CodeInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := MapValidationError(tt.inputError)

			if apiErr.Type != ErrorTypeValidation {
				t.Errorf("Expected type %s, got %s", ErrorTypeValidation, apiErr.Type)
			}

			if apiErr.Code != tt.expectedCode {
				t.Errorf("Expected code %s, got %s", tt.expectedCode, apiErr.Code)
			}

			if apiErr.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, apiErr.StatusCode)
			}
		})
	}
}