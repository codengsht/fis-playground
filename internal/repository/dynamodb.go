package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"fis-playground/internal/models"
)

// ListItemsOptions contains options for listing items with pagination
type ListItemsOptions struct {
	Limit            int32
	LastEvaluatedKey map[string]types.AttributeValue
}

// ListItemsResult contains the result of listing items with pagination info
type ListItemsResult struct {
	Items            []models.Item
	LastEvaluatedKey map[string]types.AttributeValue
	HasMore          bool
}

// ItemRepository defines the interface for item data operations
type ItemRepository interface {
	CreateItem(ctx context.Context, item *models.Item) error
	GetItem(ctx context.Context, id string) (*models.Item, error)
	ListItems(ctx context.Context, options *ListItemsOptions) (*ListItemsResult, error)
	UpdateItem(ctx context.Context, id string, updates *models.UpdateItemRequest) (*models.Item, error)
	DeleteItem(ctx context.Context, id string) error
}

// DynamoDBRepository implements ItemRepository using DynamoDB
type DynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoDBRepository creates a new DynamoDB repository instance
func NewDynamoDBRepository(client *dynamodb.Client, tableName string) *DynamoDBRepository {
	return &DynamoDBRepository{
		client:    client,
		tableName: tableName,
	}
}

// NewDynamoDBRepositoryFromManager creates a new DynamoDB repository using ClientManager
func NewDynamoDBRepositoryFromManager(clientManager *ClientManager) *DynamoDBRepository {
	return &DynamoDBRepository{
		client:    clientManager.GetClient(),
		tableName: clientManager.GetTableName(),
	}
}

// GetClient returns the DynamoDB client (for testing purposes)
func (r *DynamoDBRepository) GetClient() *dynamodb.Client {
	return r.client
}

// GetTableName returns the table name (for testing purposes)
func (r *DynamoDBRepository) GetTableName() string {
	return r.tableName
}

// HealthCheck verifies the repository can connect to DynamoDB
func (r *DynamoDBRepository) HealthCheck(ctx context.Context) error {
	cm := &ClientManager{
		client: r.client,
		config: &DynamoDBConfig{
			TableName: r.tableName,
		},
	}
	return cm.HealthCheck(ctx)
}

// CreateItem creates a new item in DynamoDB with proper error handling
func (r *DynamoDBRepository) CreateItem(ctx context.Context, item *models.Item) error {
	// Generate ID if not provided
	if item.ID == "" {
		item.ID = uuid.New().String()
	}

	// Validate the item
	if err := item.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	// Convert item to DynamoDB attribute values
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	// Create the item with conditional check to prevent duplicates
	input := &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		// For create, ConditionalCheckFailedException means item already exists
		var conditionalCheckFailed *types.ConditionalCheckFailedException
		if errors.As(err, &conditionalCheckFailed) {
			return fmt.Errorf("%w: item with this ID already exists", ErrItemAlreadyExists)
		}
		return HandleDynamoDBError(err)
	}

	return nil
}

// GetItem retrieves a single item from DynamoDB by ID
func (r *DynamoDBRepository) GetItem(ctx context.Context, id string) (*models.Item, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: item ID cannot be empty", ErrInvalidInput)
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, HandleDynamoDBError(err)
	}

	// Check if item was found
	if result.Item == nil {
		return nil, ErrItemNotFound
	}

	// Unmarshal the item
	var item models.Item
	err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	return &item, nil
}

// ListItems retrieves items with pagination support
func (r *DynamoDBRepository) ListItems(ctx context.Context, options *ListItemsOptions) (*ListItemsResult, error) {
	if options == nil {
		options = &ListItemsOptions{
			Limit: 50, // Default limit
		}
	}

	// Ensure limit is within reasonable bounds
	if options.Limit <= 0 {
		options.Limit = 50
	}
	if options.Limit > 100 {
		options.Limit = 100
	}

	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
		Limit:     aws.Int32(options.Limit),
	}

	// Add pagination token if provided
	if options.LastEvaluatedKey != nil {
		input.ExclusiveStartKey = options.LastEvaluatedKey
	}

	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, HandleDynamoDBError(err)
	}

	// Unmarshal items
	var items []models.Item
	err = attributevalue.UnmarshalListOfMaps(result.Items, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal items: %w", err)
	}

	return &ListItemsResult{
		Items:            items,
		LastEvaluatedKey: result.LastEvaluatedKey,
		HasMore:          result.LastEvaluatedKey != nil,
	}, nil
}

// UpdateItem updates an existing item with conditional checks
func (r *DynamoDBRepository) UpdateItem(ctx context.Context, id string, updates *models.UpdateItemRequest) (*models.Item, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: item ID cannot be empty", ErrInvalidInput)
	}

	if updates == nil {
		return nil, fmt.Errorf("%w: updates cannot be nil", ErrInvalidInput)
	}

	// Validate the update request
	if err := updates.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	// Build update expression and attribute values
	updateExpression := "SET updated_at = :updated_at"

	// Create a timestamp for the update
	now := time.Now()
	timestampAV, err := attributevalue.Marshal(now)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal timestamp: %w", err)
	}

	expressionAttributeValues := map[string]types.AttributeValue{
		":updated_at": timestampAV,
	}

	// Add fields to update if they are provided
	if updates.Name != "" {
		updateExpression += ", #name = :name"
		expressionAttributeValues[":name"] = &types.AttributeValueMemberS{Value: updates.Name}
	}
	if updates.Description != "" {
		updateExpression += ", description = :description"
		expressionAttributeValues[":description"] = &types.AttributeValueMemberS{Value: updates.Description}
	}
	if updates.Status != "" {
		updateExpression += ", #status = :status"
		expressionAttributeValues[":status"] = &types.AttributeValueMemberS{Value: updates.Status}
	}

	// Expression attribute names for reserved keywords
	expressionAttributeNames := map[string]string{}
	if updates.Name != "" {
		expressionAttributeNames["#name"] = "name"
	}
	if updates.Status != "" {
		expressionAttributeNames["#status"] = "status"
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ConditionExpression:       aws.String("attribute_exists(id)"), // Ensure item exists
		ReturnValues:              types.ReturnValueAllNew,
	}

	// Add expression attribute names if needed
	if len(expressionAttributeNames) > 0 {
		input.ExpressionAttributeNames = expressionAttributeNames
	}

	result, err := r.client.UpdateItem(ctx, input)
	if err != nil {
		return nil, HandleDynamoDBError(err)
	}

	// Unmarshal the updated item
	var item models.Item
	err = attributevalue.UnmarshalMap(result.Attributes, &item)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated item: %w", err)
	}

	return &item, nil
}

// DeleteItem deletes an item with existence validation
func (r *DynamoDBRepository) DeleteItem(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: item ID cannot be empty", ErrInvalidInput)
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		ConditionExpression: aws.String("attribute_exists(id)"), // Ensure item exists before deletion
	}

	_, err := r.client.DeleteItem(ctx, input)
	if err != nil {
		return HandleDynamoDBError(err)
	}

	return nil
}
