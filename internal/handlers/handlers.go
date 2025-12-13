package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"fis-playground/internal/models"
	"fis-playground/internal/repository"
)

// ItemHandler handles HTTP requests for item operations
type ItemHandler struct {
	repo repository.ItemRepository
}

// NewItemHandler creates a new item handler instance
func NewItemHandler(repo repository.ItemRepository) *ItemHandler {
	return &ItemHandler{
		repo: repo,
	}
}

// HealthCheck handles GET /health requests - simple API health check
func (h *ItemHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":  "healthy",
			"service": "FIS Playground API",
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// HealthCheckDB handles GET /health/db requests - checks DynamoDB connectivity
func (h *ItemHandler) HealthCheckDB(w http.ResponseWriter, r *http.Request) {
	dynamoDBStatus := "healthy"
	dynamoDBMessage := "Connected"

	// Try to verify DynamoDB connection
	if dynamoRepo, ok := h.repo.(*repository.DynamoDBRepository); ok {
		if err := dynamoRepo.HealthCheck(r.Context()); err != nil {
			dynamoDBStatus = "unhealthy"
			dynamoDBMessage = err.Error()
		}
	} else {
		dynamoDBStatus = "unknown"
		dynamoDBMessage = "Repository type not recognized"
	}

	success := dynamoDBStatus == "healthy"
	statusCode := http.StatusOK
	if !success {
		statusCode = http.StatusServiceUnavailable
	}

	response := models.APIResponse{
		Success: success,
		Data: map[string]interface{}{
			"status":    dynamoDBStatus,
			"service":   "DynamoDB",
			"message":   dynamoDBMessage,
			"tableName": h.getTableName(),
		},
	}

	writeJSONResponse(w, statusCode, response)
}

// getTableName returns the DynamoDB table name if available
func (h *ItemHandler) getTableName() string {
	if dynamoRepo, ok := h.repo.(*repository.DynamoDBRepository); ok {
		return dynamoRepo.GetTableName()
	}
	return "unknown"
}

// CreateItem handles POST /items requests
func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var createReq models.CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
		WriteJSONParseErrorResponse(w, r, err)
		return
	}

	// Validate request
	if err := createReq.Validate(); err != nil {
		WriteValidationErrorResponse(w, r, err)
		return
	}

	// Create new item
	item := models.NewItem(createReq.Name, createReq.Description)

	// Save to repository
	if err := h.repo.CreateItem(r.Context(), item); err != nil {
		WriteRepositoryErrorResponse(w, r, err)
		return
	}

	// Return success response
	response := models.APIResponse{
		Success: true,
		Data:    item,
	}

	writeJSONResponse(w, http.StatusCreated, response)
}

// GetItem handles GET /items/{id} requests
func (h *ItemHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	if itemID == "" {
		WriteMissingParameterErrorResponse(w, r, "Item ID")
		return
	}

	// Retrieve item from repository
	item, err := h.repo.GetItem(r.Context(), itemID)
	if err != nil {
		WriteRepositoryErrorResponse(w, r, err)
		return
	}

	// Return success response
	response := models.APIResponse{
		Success: true,
		Data:    item,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// ListItems handles GET /items requests
func (h *ItemHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for pagination
	options := &repository.ListItemsOptions{
		Limit: 50, // Default limit
	}

	// Parse limit parameter
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err != nil {
			apiErr := NewValidationError(CodeInvalidFormat, "Invalid limit parameter", "Limit must be a valid integer")
			WriteErrorResponse(w, r, apiErr)
			return
		} else if limit <= 0 || limit > 100 {
			apiErr := NewValidationError(CodeInvalidValue, "Invalid limit value", "Limit must be between 1 and 100")
			WriteErrorResponse(w, r, apiErr)
			return
		} else {
			options.Limit = int32(limit)
		}
	}

	// Parse pagination token (simplified - in real implementation would decode properly)
	if token := r.URL.Query().Get("next_token"); token != "" {
		// For now, we'll skip complex token parsing
		// In a real implementation, this would decode the pagination token
		log.Printf("Pagination token provided: %s", token)
	}

	// Retrieve items from repository
	result, err := h.repo.ListItems(r.Context(), options)
	if err != nil {
		WriteRepositoryErrorResponse(w, r, err)
		return
	}

	// Prepare response data
	responseData := map[string]interface{}{
		"items":    result.Items,
		"has_more": result.HasMore,
		"count":    len(result.Items),
	}

	// Add next token if there are more items (simplified)
	if result.HasMore {
		responseData["next_token"] = "pagination_token_placeholder"
	}

	// Return success response
	response := models.APIResponse{
		Success: true,
		Data:    responseData,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// UpdateItem handles PUT /items/{id} requests
func (h *ItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	if itemID == "" {
		WriteMissingParameterErrorResponse(w, r, "Item ID")
		return
	}

	// Parse request body
	var updateReq models.UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		WriteJSONParseErrorResponse(w, r, err)
		return
	}

	// Validate request
	if err := updateReq.Validate(); err != nil {
		WriteValidationErrorResponse(w, r, err)
		return
	}

	// Update item in repository
	item, err := h.repo.UpdateItem(r.Context(), itemID, &updateReq)
	if err != nil {
		WriteRepositoryErrorResponse(w, r, err)
		return
	}

	// Return success response
	response := models.APIResponse{
		Success: true,
		Data:    item,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// DeleteItem handles DELETE /items/{id} requests
func (h *ItemHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	if itemID == "" {
		WriteMissingParameterErrorResponse(w, r, "Item ID")
		return
	}

	// Delete item from repository
	if err := h.repo.DeleteItem(r.Context(), itemID); err != nil {
		WriteRepositoryErrorResponse(w, r, err)
		return
	}

	// Return success response
	response := models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Item deleted successfully",
			"id":      itemID,
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// Helper functions for response creation
