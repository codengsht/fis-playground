# Integration Tests

This directory contains end-to-end integration tests for the FIS Playground API.

## Overview

The integration tests validate the complete CRUD workflows through the deployed API, including:

- **Complete CRUD workflows**: Create, Read, Update, Delete operations
- **Error scenarios**: Invalid inputs, missing resources, malformed requests
- **API response format validation**: JSON structure, content types, status codes
- **Edge cases**: Maximum field lengths, pagination, concurrent operations

## Requirements

- Go 1.23 or later
- AWS CLI configured with appropriate credentials
- Deployed FIS Playground CloudFormation stack

## Prerequisites

**IMPORTANT**: These are true integration tests that require a deployed FIS Playground stack. You must deploy the infrastructure before running these tests.

```bash
# Deploy the stack first
make deploy

# Then run integration tests
make integration-test
```

## Running Tests

### Using the Test Runner Script

The recommended way to run integration tests is using the provided script:

```bash
# Run tests against default stack
./run_tests.sh

# Run tests against specific stack and region
./run_tests.sh --stack-name my-fis-playground --region us-west-2

# Run with verbose output
./run_tests.sh --verbose --timeout 60s
```

### Manual Test Execution

You can also run tests manually if you have the API endpoint:

```bash
# Set the API endpoint environment variable
export API_ENDPOINT="https://your-api-gateway-url.amazonaws.com/prod"

# Run the tests
go test -v -timeout 30s ./...
```

## Test Structure

### TestEndToEndCRUDWorkflow

Tests the complete CRUD workflow:
1. Create a new item
2. Retrieve the created item
3. List items (verify item is included)
4. Update the item
5. Delete the item
6. Verify item is deleted

**Validates Requirements**: 1.2, 3.1, 3.2, 3.3, 3.4

### TestErrorScenarios

Tests various error conditions:
- Invalid request data
- Malformed JSON
- Nonexistent resources
- Invalid parameters
- Invalid field values

**Validates Requirements**: 3.5

### TestAPIResponseFormat

Validates that all API responses follow the expected JSON format:
- Proper content types
- Required response fields
- Error response structure
- Item data structure

**Validates Requirements**: 1.2, 3.1, 3.2, 3.3, 3.4

### TestEdgeCases

Tests boundary conditions and edge cases:
- Maximum field lengths
- Pagination parameters
- Concurrent operations

**Validates Requirements**: 3.1, 3.2, 3.3, 3.4

## Configuration

The tests can be configured using environment variables:

- `API_ENDPOINT`: The base URL of the deployed API (required)
- `AWS_REGION`: AWS region for CloudFormation operations (optional)

## Test Data Management

The tests create and clean up their own test data against the real deployed API. Each test:
1. Creates any required test items in the deployed DynamoDB table
2. Performs the test operations through the real API Gateway and Lambda
3. Cleans up created resources from the deployed infrastructure

This ensures tests are isolated and don't interfere with each other while testing the actual deployed system.

## Troubleshooting

### API Endpoint Not Found

If you get an error about the API endpoint not being found:

1. Verify your CloudFormation stack is deployed and in a healthy state
2. Check that the stack has an output named `ApiEndpoint`
3. Ensure your AWS credentials have permission to describe CloudFormation stacks

### Tests Timing Out

If tests are timing out:

1. Increase the timeout using `--timeout` flag
2. Check if the API is responding to health checks
3. Verify network connectivity to the API endpoint

### Authentication Errors

If you get AWS authentication errors:

1. Verify AWS CLI is configured: `aws sts get-caller-identity`
2. Check that your credentials have the necessary permissions
3. Ensure you're using the correct AWS region

## Adding New Tests

When adding new integration tests:

1. Follow the existing test structure and naming conventions
2. Include proper cleanup of any created resources
3. Add appropriate error handling and validation
4. Update this README with information about new test cases
5. Reference the specific requirements being validated

## Performance Considerations

These tests make real HTTP requests to a deployed API, so they:

- Take longer to run than unit tests
- Require network connectivity
- May be affected by API latency and AWS service performance
- Should be run less frequently than unit tests

Consider running these tests:
- Before deploying to production
- As part of a CI/CD pipeline
- When validating infrastructure changes
- During fault injection experiments