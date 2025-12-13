package repository

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// DynamoDBConfig holds configuration for DynamoDB client
type DynamoDBConfig struct {
	TableName string
	Region    string
}

// NewDynamoDBConfig creates a new DynamoDB configuration from environment variables
func NewDynamoDBConfig() (*DynamoDBConfig, error) {
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("DYNAMODB_TABLE_NAME environment variable is required")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // Default region
	}

	return &DynamoDBConfig{
		TableName: tableName,
		Region:    region,
	}, nil
}

// NewDynamoDBClient creates a new DynamoDB client with proper configuration
func NewDynamoDBClient(ctx context.Context, cfg *DynamoDBConfig) (*dynamodb.Client, error) {
	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(awsCfg)

	return client, nil
}

// ClientManager manages DynamoDB client lifecycle and provides connection utilities
type ClientManager struct {
	client *dynamodb.Client
	config *DynamoDBConfig
}

// NewClientManager creates a new client manager with proper error handling
func NewClientManager(ctx context.Context) (*ClientManager, error) {
	// Load configuration
	cfg, err := NewDynamoDBConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB config: %w", err)
	}

	// Create client
	client, err := NewDynamoDBClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB client: %w", err)
	}

	return &ClientManager{
		client: client,
		config: cfg,
	}, nil
}

// GetClient returns the DynamoDB client
func (cm *ClientManager) GetClient() *dynamodb.Client {
	return cm.client
}

// GetTableName returns the configured table name
func (cm *ClientManager) GetTableName() string {
	return cm.config.TableName
}

// GetRegion returns the configured AWS region
func (cm *ClientManager) GetRegion() string {
	return cm.config.Region
}

// HealthCheck performs a basic health check on the DynamoDB connection
func (cm *ClientManager) HealthCheck(ctx context.Context) error {
	// Perform a simple DescribeTable operation to verify connectivity
	_, err := cm.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(cm.config.TableName),
	})
	if err != nil {
		return fmt.Errorf("DynamoDB health check failed: %w", err)
	}

	return nil
}
