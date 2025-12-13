package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"fis-playground/internal/models"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	APIEndpoint string
	Timeout     time.Duration
}

// getTestConfig retrieves test configuration from environment variables
func getTestConfig() *TestConfig {
	endpoint := os.Getenv("API_ENDPOINT")
	if endpoint == "" {
		// Try to get from CloudFormation stack outputs if not set
		endpoint = getAPIEndpointFromStack()
	}
	
	return &TestConfig{
		APIEndpoint: endpoint,
		Timeout:     30 * time.Second,
	}
}

// getAPIEndpointFromStack attempts to get API endpoint from CloudFormation stack
func getAPIEndpointFromStack() string {
	// This would typically use AWS SDK to get stack outputs
	// For now, return empty string to indicate endpoint not available
	return ""
}

// HTTPClient wraps http.Client with test-specific configuration
type HTTPClient struct {
	client   *http.Client
	baseURL  string
	timeout  time.Duration
}

// NewHTTPClient creates a new HTTP client for testing
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: strings.TrimSuffix(baseURL, "/"),
		timeout: timeout,
	}
}

// makeRequest performs HTTP request and returns response
func (c *HTTPClient) makeRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}
	
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	
	return c.client.Do(req)
}

// parseResponse parses HTTP response into APIResponse
func parseResponse(resp *http.Response) (*models.APIResponse, error) {
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	var apiResp models.APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return &apiResp, nil
}

// TestEndToEndCRUDWorkflow tests complete CRUD operations through deployed API
func TestEndToEndCRUDWorkflow(t *testing.T) {
	config := getTestConfig()
	if config.APIEndpoint == "" {
		t.Fatal("API_ENDPOINT not set. Integration tests require a deployed API endpoint. Deploy the stack first with 'make deploy'")
	}
	
	client := NewHTTPClient(config.APIEndpoint, config.Timeout)
	
	// Test data
	createReq := models.CreateItemRequest{
		Name:        "Integration Test Item",
		Description: "Created during end-to-end testing",
	}
	
	// Step 1: Create item
	t.Run("CreateItem", func(t *testing.T) {
		resp, err := client.makeRequest("POST", "/items", createReq)
		if err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}
		
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if !apiResp.Success {
			t.Errorf("Expected success=true, got success=%v, error=%v", apiResp.Success, apiResp.Error)
		}
		
		// Verify response contains item data
		if apiResp.Data == nil {
			t.Fatal("Expected data in response, got nil")
		}
		
		// Extract item ID for subsequent tests
		itemData, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected item data to be object")
		}
		
		itemID, ok := itemData["id"].(string)
		if !ok || itemID == "" {
			t.Fatal("Expected item ID in response")
		}
		
		// Store item ID for other tests
		os.Setenv("TEST_ITEM_ID", itemID)
	})
	
	// Step 2: Get created item
	t.Run("GetItem", func(t *testing.T) {
		itemID := os.Getenv("TEST_ITEM_ID")
		if itemID == "" {
			t.Skip("No item ID available from create test")
		}
		
		resp, err := client.makeRequest("GET", "/items/"+itemID, nil)
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if !apiResp.Success {
			t.Errorf("Expected success=true, got success=%v, error=%v", apiResp.Success, apiResp.Error)
		}
		
		// Verify item data matches what we created
		if apiResp.Data == nil {
			t.Fatal("Expected data in response, got nil")
		}
		
		itemData, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected item data to be object")
		}
		
		if itemData["name"] != createReq.Name {
			t.Errorf("Expected name %s, got %s", createReq.Name, itemData["name"])
		}
		
		if itemData["description"] != createReq.Description {
			t.Errorf("Expected description %s, got %s", createReq.Description, itemData["description"])
		}
	})
	
	// Step 3: List items (should include our created item)
	t.Run("ListItems", func(t *testing.T) {
		resp, err := client.makeRequest("GET", "/items", nil)
		if err != nil {
			t.Fatalf("Failed to list items: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if !apiResp.Success {
			t.Errorf("Expected success=true, got success=%v, error=%v", apiResp.Success, apiResp.Error)
		}
		
		// Verify response contains items array
		if apiResp.Data == nil {
			t.Fatal("Expected data in response, got nil")
		}
		
		listData, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected list data to be object")
		}
		
		items, ok := listData["items"].([]interface{})
		if !ok {
			t.Fatal("Expected items array in response")
		}
		
		// Verify our item is in the list
		itemID := os.Getenv("TEST_ITEM_ID")
		found := false
		for _, item := range items {
			itemObj, ok := item.(map[string]interface{})
			if ok && itemObj["id"] == itemID {
				found = true
				break
			}
		}
		
		if !found && itemID != "" {
			t.Error("Created item not found in list")
		}
	})
	
	// Step 4: Update item
	t.Run("UpdateItem", func(t *testing.T) {
		itemID := os.Getenv("TEST_ITEM_ID")
		if itemID == "" {
			t.Skip("No item ID available from create test")
		}
		
		updateReq := models.UpdateItemRequest{
			Name:        "Updated Integration Test Item",
			Description: "Updated during end-to-end testing",
			Status:      "inactive",
		}
		
		resp, err := client.makeRequest("PUT", "/items/"+itemID, updateReq)
		if err != nil {
			t.Fatalf("Failed to update item: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if !apiResp.Success {
			t.Errorf("Expected success=true, got success=%v, error=%v", apiResp.Success, apiResp.Error)
		}
		
		// Verify updated data
		if apiResp.Data == nil {
			t.Fatal("Expected data in response, got nil")
		}
		
		itemData, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected item data to be object")
		}
		
		if itemData["name"] != updateReq.Name {
			t.Errorf("Expected updated name %s, got %s", updateReq.Name, itemData["name"])
		}
		
		if itemData["status"] != updateReq.Status {
			t.Errorf("Expected updated status %s, got %s", updateReq.Status, itemData["status"])
		}
	})
	
	// Step 5: Delete item
	t.Run("DeleteItem", func(t *testing.T) {
		itemID := os.Getenv("TEST_ITEM_ID")
		if itemID == "" {
			t.Skip("No item ID available from create test")
		}
		
		resp, err := client.makeRequest("DELETE", "/items/"+itemID, nil)
		if err != nil {
			t.Fatalf("Failed to delete item: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if !apiResp.Success {
			t.Errorf("Expected success=true, got success=%v, error=%v", apiResp.Success, apiResp.Error)
		}
	})
	
	// Step 6: Verify item is deleted
	t.Run("VerifyItemDeleted", func(t *testing.T) {
		itemID := os.Getenv("TEST_ITEM_ID")
		if itemID == "" {
			t.Skip("No item ID available from create test")
		}
		
		resp, err := client.makeRequest("GET", "/items/"+itemID, nil)
		if err != nil {
			t.Fatalf("Failed to get deleted item: %v", err)
		}
		
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status %d for deleted item, got %d", http.StatusNotFound, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if apiResp.Success {
			t.Error("Expected success=false for deleted item")
		}
		
		if apiResp.Error == nil {
			t.Error("Expected error for deleted item")
		}
	})
}

// TestErrorScenarios tests various error conditions
func TestErrorScenarios(t *testing.T) {
	config := getTestConfig()
	if config.APIEndpoint == "" {
		t.Fatal("API_ENDPOINT not set. Integration tests require a deployed API endpoint. Deploy the stack first with 'make deploy'")
	}
	
	client := NewHTTPClient(config.APIEndpoint, config.Timeout)
	
	t.Run("CreateItemWithInvalidData", func(t *testing.T) {
		// Test with empty name
		invalidReq := models.CreateItemRequest{
			Name:        "",
			Description: "Test description",
		}
		
		resp, err := client.makeRequest("POST", "/items", invalidReq)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if apiResp.Success {
			t.Error("Expected success=false for invalid data")
		}
		
		if apiResp.Error == nil {
			t.Error("Expected error for invalid data")
		}
	})
	
	t.Run("CreateItemWithInvalidJSON", func(t *testing.T) {
		// Send invalid JSON
		req, err := http.NewRequest("POST", config.APIEndpoint+"/items", strings.NewReader("invalid json"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if apiResp.Success {
			t.Error("Expected success=false for invalid JSON")
		}
	})
	
	t.Run("GetNonexistentItem", func(t *testing.T) {
		resp, err := client.makeRequest("GET", "/items/nonexistent-id", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if apiResp.Success {
			t.Error("Expected success=false for nonexistent item")
		}
		
		if apiResp.Error == nil {
			t.Error("Expected error for nonexistent item")
		}
	})
	
	t.Run("UpdateNonexistentItem", func(t *testing.T) {
		updateReq := models.UpdateItemRequest{
			Name: "Updated name",
		}
		
		resp, err := client.makeRequest("PUT", "/items/nonexistent-id", updateReq)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if apiResp.Success {
			t.Error("Expected success=false for nonexistent item")
		}
	})
	
	t.Run("DeleteNonexistentItem", func(t *testing.T) {
		resp, err := client.makeRequest("DELETE", "/items/nonexistent-id", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if apiResp.Success {
			t.Error("Expected success=false for nonexistent item")
		}
	})
	
	t.Run("ListItemsWithInvalidParameters", func(t *testing.T) {
		// Test with invalid limit
		resp, err := client.makeRequest("GET", "/items?limit=invalid", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if apiResp.Success {
			t.Error("Expected success=false for invalid parameters")
		}
	})
	
	t.Run("UpdateItemWithInvalidStatus", func(t *testing.T) {
		// First create an item to update
		createReq := models.CreateItemRequest{
			Name:        "Test Item for Invalid Update",
			Description: "Test description",
		}
		
		createResp, err := client.makeRequest("POST", "/items", createReq)
		if err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}
		
		if createResp.StatusCode != http.StatusCreated {
			t.Skip("Failed to create item for update test")
		}
		
		createAPIResp, err := parseResponse(createResp)
		if err != nil {
			t.Fatalf("Failed to parse create response: %v", err)
		}
		
		itemData, ok := createAPIResp.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected item data to be object")
		}
		
		itemID, ok := itemData["id"].(string)
		if !ok || itemID == "" {
			t.Fatal("Expected item ID in response")
		}
		
		// Now try to update with invalid status
		updateReq := models.UpdateItemRequest{
			Status: "invalid_status",
		}
		
		resp, err := client.makeRequest("PUT", "/items/"+itemID, updateReq)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if apiResp.Success {
			t.Error("Expected success=false for invalid status")
		}
		
		// Clean up - delete the created item
		client.makeRequest("DELETE", "/items/"+itemID, nil)
	})
}

// TestAPIResponseFormat validates that all API responses follow the expected JSON format
func TestAPIResponseFormat(t *testing.T) {
	config := getTestConfig()
	if config.APIEndpoint == "" {
		t.Fatal("API_ENDPOINT not set. Integration tests require a deployed API endpoint. Deploy the stack first with 'make deploy'")
	}
	
	client := NewHTTPClient(config.APIEndpoint, config.Timeout)
	
	t.Run("HealthCheckResponseFormat", func(t *testing.T) {
		resp, err := client.makeRequest("GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}
		
		// Verify response structure
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		// Verify required fields are present
		if apiResp.Success != true {
			t.Error("Expected success field to be true")
		}
		
		// Health check should have data
		if apiResp.Data == nil {
			t.Error("Expected data field in health check response")
		}
	})
	
	t.Run("CreateItemResponseFormat", func(t *testing.T) {
		createReq := models.CreateItemRequest{
			Name:        "Format Test Item",
			Description: "Testing response format",
		}
		
		resp, err := client.makeRequest("POST", "/items", createReq)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}
		
		// Verify response structure
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		// Verify required fields
		if !apiResp.Success {
			t.Error("Expected success field to be true")
		}
		
		if apiResp.Data == nil {
			t.Error("Expected data field in create response")
		}
		
		// Verify item structure
		itemData, ok := apiResp.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected item data to be object")
		}
		
		requiredFields := []string{"id", "name", "description", "status", "created_at", "updated_at"}
		for _, field := range requiredFields {
			if _, exists := itemData[field]; !exists {
				t.Errorf("Expected field %s in item data", field)
			}
		}
		
		// Clean up
		if itemID, ok := itemData["id"].(string); ok {
			client.makeRequest("DELETE", "/items/"+itemID, nil)
		}
	})
	
	t.Run("ErrorResponseFormat", func(t *testing.T) {
		// Make request that should return error
		resp, err := client.makeRequest("GET", "/items/nonexistent", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}
		
		// Verify response structure
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		// Verify error response structure
		if apiResp.Success {
			t.Error("Expected success field to be false for error")
		}
		
		if apiResp.Error == nil {
			t.Error("Expected error field in error response")
		}
		
		// Verify error structure
		if apiResp.Error.Type == "" {
			t.Error("Expected error type to be set")
		}
		
		if apiResp.Error.Code == "" {
			t.Error("Expected error code to be set")
		}
		
		if apiResp.Error.Message == "" {
			t.Error("Expected error message to be set")
		}
	})
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	config := getTestConfig()
	if config.APIEndpoint == "" {
		t.Fatal("API_ENDPOINT not set. Integration tests require a deployed API endpoint. Deploy the stack first with 'make deploy'")
	}
	
	client := NewHTTPClient(config.APIEndpoint, config.Timeout)
	
	t.Run("CreateItemWithMaxLengthFields", func(t *testing.T) {
		// Test with maximum allowed field lengths
		longName := strings.Repeat("a", 100) // Assuming 100 is the max
		longDesc := strings.Repeat("b", 500) // Assuming 500 is the max
		
		createReq := models.CreateItemRequest{
			Name:        longName,
			Description: longDesc,
		}
		
		resp, err := client.makeRequest("POST", "/items", createReq)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		// Should succeed with max length fields
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if !apiResp.Success {
			t.Errorf("Expected success=true, got success=%v, error=%v", apiResp.Success, apiResp.Error)
		}
		
		// Clean up
		if apiResp.Data != nil {
			if itemData, ok := apiResp.Data.(map[string]interface{}); ok {
				if itemID, ok := itemData["id"].(string); ok {
					client.makeRequest("DELETE", "/items/"+itemID, nil)
				}
			}
		}
	})
	
	t.Run("ListItemsWithPagination", func(t *testing.T) {
		// Test pagination parameters
		resp, err := client.makeRequest("GET", "/items?limit=5&cursor=test", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		apiResp, err := parseResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if !apiResp.Success {
			t.Errorf("Expected success=true, got success=%v, error=%v", apiResp.Success, apiResp.Error)
		}
		
		// Verify pagination structure
		if apiResp.Data != nil {
			listData, ok := apiResp.Data.(map[string]interface{})
			if ok {
				if _, exists := listData["items"]; !exists {
					t.Error("Expected items array in paginated response")
				}
				if _, exists := listData["has_more"]; !exists {
					t.Error("Expected has_more field in paginated response")
				}
			}
		}
	})
	
	t.Run("ConcurrentOperations", func(t *testing.T) {
		// Test concurrent creation of items
		const numConcurrent = 5
		results := make(chan error, numConcurrent)
		
		for i := 0; i < numConcurrent; i++ {
			go func(index int) {
				createReq := models.CreateItemRequest{
					Name:        fmt.Sprintf("Concurrent Item %d", index),
					Description: fmt.Sprintf("Created concurrently %d", index),
				}
				
				resp, err := client.makeRequest("POST", "/items", createReq)
				if err != nil {
					results <- err
					return
				}
				
				if resp.StatusCode != http.StatusCreated {
					results <- fmt.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
					return
				}
				
				// Clean up created item
				apiResp, err := parseResponse(resp)
				if err == nil && apiResp.Data != nil {
					if itemData, ok := apiResp.Data.(map[string]interface{}); ok {
						if itemID, ok := itemData["id"].(string); ok {
							client.makeRequest("DELETE", "/items/"+itemID, nil)
						}
					}
				}
				
				results <- nil
			}(i)
		}
		
		// Wait for all operations to complete
		for i := 0; i < numConcurrent; i++ {
			if err := <-results; err != nil {
				t.Errorf("Concurrent operation failed: %v", err)
			}
		}
	})
}