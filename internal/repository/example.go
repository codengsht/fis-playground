package repository

import (
	"context"
	"fmt"
	"log"
)

// ExampleUsage demonstrates how to use the DynamoDB client and repository
func ExampleUsage() {
	ctx := context.Background()

	// Create client manager
	clientManager, err := NewClientManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create client manager: %v", err)
	}

	// Perform health check
	if err := clientManager.HealthCheck(ctx); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		log.Println("DynamoDB connection is healthy")
	}

	// Create repository
	repo := NewDynamoDBRepositoryFromManager(clientManager)

	// Display configuration
	fmt.Printf("Table Name: %s\n", repo.GetTableName())
	fmt.Printf("Region: %s\n", clientManager.GetRegion())

	// Repository health check
	if err := repo.HealthCheck(ctx); err != nil {
		log.Printf("Repository health check failed: %v", err)
	} else {
		log.Println("Repository is ready for operations")
	}
}
