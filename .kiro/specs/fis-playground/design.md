# FIS Playground Design Document

## Overview

The FIS Playground is a serverless microservice built on AWS that demonstrates realistic patterns for fault injection testing. The system uses a three-tier architecture with API Gateway handling HTTP requests, Go Lambda functions processing business logic, and DynamoDB providing persistent storage. All infrastructure is defined as CloudFormation templates for reproducible deployments.

The application implements a simple item management system with CRUD operations, providing meaningful business logic that can be disrupted during fault injection experiments. The design emphasizes realistic error handling, proper AWS service integration patterns, and clean separation of concerns.

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   API Gateway   │───▶│   Lambda (Go)    │───▶│    DynamoDB     │
│                 │    │                  │    │                 │
│ - HTTP Routing  │    │ - Business Logic │    │ - Data Storage  │
│ - Request/Resp  │    │ - Validation     │    │ - Persistence   │
│ - CORS          │    │ - Error Handling │    │ - Querying      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

The architecture follows AWS Well-Architected principles with:
- **API Gateway**: Handles HTTP protocol, routing, and request/response transformation
- **Lambda Function**: Stateless compute layer implementing business logic in Go
- **DynamoDB**: NoSQL database providing fast, scalable data persistence
- **CloudFormation**: Infrastructure as Code for consistent deployments

## Components and Interfaces

### API Gateway Component
- **REST API**: Exposes HTTP endpoints for CRUD operations
- **Resource Paths**: `/items`, `/items/{id}` for resource management
- **HTTP Methods**: GET, POST, PUT, DELETE mapped to Lambda integration
- **Request Validation**: Basic input validation and content-type checking
- **Response Transformation**: Standardized JSON response format

### Lambda Function Component
- **Runtime**: Go 1.x with AWS SDK v2
- **Handler Functions**: Separate handlers for each CRUD operation
- **Input Processing**: JSON unmarshaling and request validation
- **Business Logic**: Item creation, retrieval, updating, and deletion
- **Error Handling**: Structured error responses with appropriate HTTP status codes
- **DynamoDB Integration**: AWS SDK operations with proper error handling

### DynamoDB Component
- **Table Design**: Single table with partition key for item identification
- **Attributes**: Flexible schema supporting item metadata and content
- **Indexes**: Primary key access patterns for efficient queries
- **Consistency**: Eventually consistent reads with strong consistency options
- **Capacity**: On-demand billing for variable workloads

## Data Models

### Item Model
```go
type Item struct {
    ID          string    `json:"id" dynamodbav:"id"`
    Name        string    `json:"name" dynamodbav:"name"`
    Description string    `json:"description" dynamodbav:"description"`
    CreatedAt   time.Time `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" dynamodbav:"updated_at"`
    Status      string    `json:"status" dynamodbav:"status"`
}
```

### API Request/Response Models
```go
type CreateItemRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

type UpdateItemRequest struct {
    Name        string `json:"name,omitempty"`
    Description string `json:"description,omitempty"`
    Status      string `json:"status,omitempty"`
}

type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
```

### DynamoDB Schema
- **Table Name**: `fis-playground-items`
- **Partition Key**: `id` (String)
- **Attributes**: All item fields stored as DynamoDB attributes
- **Billing Mode**: On-demand for cost efficiency during experimentation

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: CloudFormation deployment creates all required resources
*For any* valid CloudFormation template deployment, the resulting stack should contain API Gateway, Lambda function, and DynamoDB table resources
**Validates: Requirements 1.1, 1.3**

### Property 2: Successful deployment provides functional API endpoint
*For any* successfully deployed stack, the API endpoint should be accessible and respond to HTTP requests
**Validates: Requirements 1.2, 1.5**

### Property 3: Stack deletion removes all resources
*For any* deployed CloudFormation stack, deleting the stack should remove all created AWS resources without orphans
**Validates: Requirements 1.4, 4.5**

### Property 4: CRUD operations maintain data consistency
*For any* valid item data, performing create, read, update, and delete operations should maintain data integrity in DynamoDB
**Validates: Requirements 3.1, 3.2, 3.3, 3.4**

### Property 5: Lambda-DynamoDB integration preserves data
*For any* Lambda function operation on DynamoDB, the data changes should be correctly persisted and retrievable
**Validates: Requirements 2.2, 4.3**

### Property 6: Error handling returns appropriate responses
*For any* invalid request or system error, the API should return proper HTTP status codes and meaningful error messages
**Validates: Requirements 2.3, 3.5, 5.2**

### Property 7: API responses follow JSON format standards
*For any* API response, the content should be valid JSON with proper content-type headers and HTTP status codes
**Validates: Requirements 2.4, 5.4**

### Property 8: Infrastructure components integrate correctly
*For any* deployed system, API Gateway should successfully route requests to Lambda functions with proper permissions
**Validates: Requirements 4.1, 4.2**

## Error Handling

### Lambda Function Error Handling
- **Input Validation**: Validate all incoming requests against expected schemas
- **DynamoDB Errors**: Handle throttling, capacity exceeded, and network errors with exponential backoff
- **Business Logic Errors**: Return appropriate HTTP status codes (400 for client errors, 500 for server errors)
- **Timeout Handling**: Configure appropriate Lambda timeout values and handle timeout scenarios gracefully

### API Gateway Error Handling
- **Request Validation**: Validate request format, content-type, and required parameters
- **Integration Errors**: Handle Lambda function errors and timeouts with proper error responses
- **Rate Limiting**: Configure throttling to prevent abuse and handle rate limit exceeded scenarios
- **CORS Errors**: Properly handle cross-origin requests and preflight OPTIONS requests

### DynamoDB Error Handling
- **Conditional Check Failures**: Handle item conflicts during updates with appropriate error messages
- **Capacity Errors**: Handle read/write capacity exceeded with retry logic
- **Item Not Found**: Return 404 status codes when requested items don't exist
- **Validation Errors**: Handle schema validation failures with descriptive error messages

### Infrastructure Error Handling
- **CloudFormation Rollback**: Handle deployment failures with automatic rollback
- **IAM Permission Errors**: Ensure proper error messages for permission-related failures
- **Resource Limits**: Handle AWS service limits and quota exceeded scenarios
- **Network Connectivity**: Handle VPC and network-related configuration errors

## Testing Strategy

### Dual Testing Approach
The testing strategy employs both unit testing and property-based testing to ensure comprehensive coverage:

- **Unit tests** verify specific examples, edge cases, and error conditions
- **Property tests** verify universal properties that should hold across all inputs
- Together they provide comprehensive coverage: unit tests catch concrete bugs, property tests verify general correctness

### Unit Testing
Unit tests will cover:
- Individual Lambda handler functions with specific input/output examples
- DynamoDB operations with known test data
- Error handling scenarios with specific error conditions
- API Gateway integration points with sample requests
- CloudFormation template validation with specific resource configurations

### Property-Based Testing
Property-based testing will use **Testify** and **go-fuzz** libraries for Go, configured to run a minimum of 100 iterations per property test. Each property-based test will be tagged with comments explicitly referencing the correctness property from this design document using the format: '**Feature: fis-playground, Property {number}: {property_text}**'

Property tests will verify:
- CRUD operations maintain data consistency across all valid inputs
- Error handling behaves correctly for all types of invalid inputs
- API responses maintain proper format across all request types
- Infrastructure deployment succeeds for all valid configurations
- Resource cleanup completes successfully for all deployment scenarios

### Integration Testing
- End-to-end API testing through deployed CloudFormation stacks
- Cross-service integration validation between API Gateway, Lambda, and DynamoDB
- Infrastructure deployment and teardown testing
- Error propagation testing across service boundaries

### Test Environment Management
- Use separate AWS accounts or regions for testing to avoid conflicts
- Implement test data cleanup procedures to prevent test interference
- Use CloudFormation stack naming conventions to identify test resources
- Implement automated test environment provisioning and teardown