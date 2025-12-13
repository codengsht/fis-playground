# FIS Playground

A serverless playground application designed for learning and experimenting with AWS Fault Injection Service (FIS). The system provides a realistic microservice architecture using API Gateway, Lambda functions written in Go, and DynamoDB storage, all deployed via CloudFormation for easy provisioning and teardown.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [AWS Setup](#aws-setup)
- [Deployment Guide](#deployment-guide)
- [API Documentation](#api-documentation)
- [Usage Examples](#usage-examples)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Cleanup](#cleanup)

## Architecture Overview

The application follows a three-tier serverless architecture designed for fault injection experimentation:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   API Gateway   │───▶│   Lambda (Go)    │───▶│    DynamoDB     │
│                 │    │                  │    │                 │
│ - HTTP Routing  │    │ - Business Logic │    │ - Data Storage  │
│ - Request/Resp  │    │ - Validation     │    │ - Persistence   │
│ - CORS          │    │ - Error Handling │    │ - Querying      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Components

- **API Gateway**: RESTful API with CORS support, handles HTTP routing and request/response transformation
- **Lambda Function**: Go-based serverless compute processing business logic with comprehensive error handling
- **DynamoDB**: NoSQL database providing fast, scalable data persistence with on-demand billing
- **CloudFormation**: Infrastructure as Code ensuring consistent, reproducible deployments

## Prerequisites

### Required Software

- **Go 1.21 or later**: [Download Go](https://golang.org/dl/)
- **AWS CLI v2**: [Installation Guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- **Make**: Build automation (usually pre-installed on macOS/Linux)
- **curl**: For API testing (usually pre-installed)

### Optional Tools

- **jq**: JSON processor for parsing API responses
- **AWS SAM CLI**: For local testing and debugging
- **Postman**: GUI-based API testing

### System Requirements

- **Operating System**: macOS, Linux, or Windows with WSL2
- **Memory**: Minimum 4GB RAM for development
- **Disk Space**: At least 1GB free space
- **Network**: Internet connection for AWS API calls

## AWS Setup

### 1. AWS Account Setup

Ensure you have an active AWS account with appropriate permissions. The deployment requires the following AWS services:
- CloudFormation
- Lambda
- API Gateway
- DynamoDB
- IAM (for role creation)

### 2. AWS CLI Configuration

Configure your AWS CLI with credentials that have sufficient permissions:

```bash
aws configure
```

You'll need to provide:
- **AWS Access Key ID**: Your access key
- **AWS Secret Access Key**: Your secret key
- **Default region**: e.g., `us-east-1`
- **Default output format**: `json` (recommended)

### 3. Required IAM Permissions

Your AWS user/role needs the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "cloudformation:*",
                "lambda:*",
                "apigateway:*",
                "dynamodb:*",
                "iam:CreateRole",
                "iam:DeleteRole",
                "iam:AttachRolePolicy",
                "iam:DetachRolePolicy",
                "iam:PassRole",
                "iam:GetRole",
                "logs:*"
            ],
            "Resource": "*"
        }
    ]
}
```

### 4. Verify AWS Setup

Test your AWS configuration:

```bash
# Verify AWS CLI is working
aws sts get-caller-identity

# Check available regions
aws ec2 describe-regions --query 'Regions[].RegionName' --output table
```

## Deployment Guide

### Step 1: Clone and Setup

```bash
# Clone the repository
git clone <repository-url>
cd fis-playground

# Verify Go installation
go version

# Download dependencies
go mod download
```

### Step 2: Build the Application

```bash
# Build the Lambda function
make build

# Or use the script directly
./scripts/build.sh
```

The build process:
1. Compiles Go code for Linux/AMD64 architecture
2. Creates a deployment package
3. Validates the binary

### Step 3: Deploy Infrastructure

#### Basic Deployment

```bash
# Deploy with default settings
./scripts/deploy.sh
```

This creates a stack named `fis-playground` in the `dev` environment using `us-east-1` region.

#### Advanced Deployment Options

```bash
# Deploy with custom parameters
./scripts/deploy.sh \
  --stack-name my-fis-playground \
  --environment prod \
  --region us-west-2

# Deploy without rollback on failure (for debugging)
./scripts/deploy.sh --no-rollback
```

#### Deployment Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--stack-name` | `fis-playground` | CloudFormation stack name |
| `--environment` | `dev` | Environment tag for resources |
| `--region` | `us-east-1` | AWS region for deployment |
| `--no-rollback` | `false` | Disable rollback on failure |

### Step 4: Verify Deployment

After successful deployment, you'll see output similar to:

```
[SUCCESS] Deployment completed successfully!
Stack outputs:
---------------------------------------------------------
|                    DescribeStacks                     |
+------------------+----------------------------------+
|  ApiEndpoint     | https://abc123.execute-api.us-east-1.amazonaws.com/dev |
|  DynamoDBTableName| fis-playground-items-dev        |
|  LambdaFunctionName| fis-playground-dev             |
---------------------------------------------------------

API endpoint available: https://abc123.execute-api.us-east-1.amazonaws.com/dev
You can test the API using: curl https://abc123.execute-api.us-east-1.amazonaws.com/dev/items
```

### Step 5: Test the Deployment

```bash
# Quick health check
curl https://your-api-endpoint.amazonaws.com/dev/items

# Run integration tests
make test-integration
```

## Alternative: AWS Console Deployment

If you prefer using the AWS Console instead of command-line tools, you can deploy the CloudFormation stack through the web interface.

### Prerequisites for Console Deployment

1. **Build the Lambda Function First:**
   ```bash
   # You still need to build the Lambda function locally
   make build
   ```

2. **Upload Lambda Code to S3 (Required):**
   ```bash
   # Create an S3 bucket for deployment artifacts
   aws s3 mb s3://your-deployment-bucket-name --region us-east-1
   
   # Upload the Lambda deployment package
   aws s3 cp lambda s3://your-deployment-bucket-name/fis-playground-lambda.zip
   ```

3. **Create CloudFormation Service Role (Optional but Recommended):**
   
   For console deployment, you can either use your user permissions directly or create a dedicated CloudFormation service role. Using a service role is recommended for better security and audit trails.

   **Option A: Use Your User Permissions (Simpler)**
   - Your AWS user needs the permissions listed in [Required IAM Permissions](#3-required-iam-permissions)
   - Leave the "IAM role" field empty during stack creation
   - CloudFormation will use your user's permissions

   **Option B: Create CloudFormation Service Role (Recommended)**
   
   Create a dedicated IAM role for CloudFormation to use:

   ```bash
   # Create the CloudFormation service role
   aws iam create-role \
     --role-name CloudFormation-FISPlayground-Role \
     --assume-role-policy-document '{
       "Version": "2012-10-17",
       "Statement": [
         {
           "Effect": "Allow",
           "Principal": {
             "Service": "cloudformation.amazonaws.com"
           },
           "Action": "sts:AssumeRole"
         }
       ]
     }'

   # Create and attach the policy
   aws iam put-role-policy \
     --role-name CloudFormation-FISPlayground-Role \
     --policy-name FISPlaygroundDeploymentPolicy \
     --policy-document '{
       "Version": "2012-10-17",
       "Statement": [
         {
           "Effect": "Allow",
           "Action": [
             "lambda:*",
             "apigateway:*",
             "dynamodb:*",
             "iam:CreateRole",
             "iam:DeleteRole",
             "iam:AttachRolePolicy",
             "iam:DetachRolePolicy",
             "iam:PassRole",
             "iam:GetRole",
             "iam:CreatePolicy",
             "iam:DeletePolicy",
             "iam:GetPolicy",
             "iam:ListAttachedRolePolicies",
             "logs:CreateLogGroup",
             "logs:DeleteLogGroup",
             "logs:PutRetentionPolicy",
             "logs:DescribeLogGroups",
             "s3:GetObject"
           ],
           "Resource": "*"
         }
       ]
     }'
   ```

   **Or create via AWS Console:**
   
   1. Go to **IAM** → **Roles** → **Create role**
   2. Select **AWS service** → **CloudFormation**
   3. Create a custom policy with the permissions above
   4. Name the role: `CloudFormation-FISPlayground-Role`

### Console Deployment Steps

#### Step 1: Access CloudFormation Console

1. Sign in to the [AWS Management Console](https://console.aws.amazon.com/)
2. Navigate to **CloudFormation** service
3. Ensure you're in the correct AWS region (e.g., us-east-1)

#### Step 2: Create New Stack

1. Click **"Create stack"** → **"With new resources (standard)"**
2. In **"Specify template"** section:
   - Select **"Upload a template file"**
   - Click **"Choose file"** and select `deployments/cloudformation/template.yaml`
   - Click **"Next"**

#### Step 3: Configure Stack Details

1. **Stack name**: Enter a name (e.g., `fis-playground`)
2. **Parameters**:
   - **Environment**: Enter environment name (e.g., `dev`, `prod`)
3. Click **"Next"**

#### Step 4: Configure Stack Options

1. **Tags** (Optional but recommended):
   - Key: `Project`, Value: `FISPlayground`
   - Key: `Environment`, Value: `dev` (or your environment)
   - Key: `Purpose`, Value: `FaultInjectionTesting`

2. **Permissions - IAM Role Selection**:
   
   **Option A: Use Your User Permissions**
   - Leave **"IAM role"** field empty
   - CloudFormation will use your current user's permissions
   - Ensure your user has all required permissions from [Required IAM Permissions](#3-required-iam-permissions)

   **Option B: Use CloudFormation Service Role (Recommended)**
   - In **"IAM role"** dropdown, select: `CloudFormation-FISPlayground-Role`
   - If you don't see the role, ensure it was created properly in the prerequisites
   - This provides better security isolation and audit trails

   **IAM Role Requirements Summary:**
   The selected role (or your user) needs permissions for:
   - **Lambda**: Create, update, delete functions and permissions
   - **API Gateway**: Create, configure, and manage REST APIs
   - **DynamoDB**: Create, configure, and manage tables
   - **IAM**: Create and manage service roles for Lambda
   - **CloudWatch Logs**: Create and manage log groups
   - **S3**: Read access to Lambda deployment packages

3. **Stack failure options**:
   - Select **"Roll back all stack resources"** (recommended)

4. **Advanced options**:
   - Leave as default

5. Click **"Next"**

#### Step 5: Review and Deploy

1. **Review all settings** carefully
2. **Capabilities**: Check the box for **"I acknowledge that AWS CloudFormation might create IAM resources"**
3. Click **"Submit"**

#### Step 6: Monitor Deployment

1. The stack will appear with status **"CREATE_IN_PROGRESS"**
2. Click on the stack name to view details
3. Monitor the **"Events"** tab for deployment progress
4. Deployment typically takes 2-5 minutes

#### Step 7: Retrieve Outputs

Once deployment completes (status: **CREATE_COMPLETE**):

1. Click on the **"Outputs"** tab
2. Note the following important values:
   - **ApiEndpoint**: Your API Gateway URL
   - **DynamoDBTableName**: DynamoDB table name
   - **LambdaFunctionName**: Lambda function name

### Console Deployment Troubleshooting

#### Common Console Issues

1. **Template Upload Fails**
   - Ensure the template file is valid YAML
   - Check file size (must be < 1MB for direct upload)
   - Validate template syntax locally first

2. **Lambda Code Not Found**
   - Verify the Lambda code was built and uploaded to S3
   - Check S3 bucket permissions
   - Ensure CodeUri in template points to correct S3 location

3. **IAM Permission Errors**
   - Verify your AWS user has CloudFormation permissions
   - Check that you acknowledged IAM resource creation
   - Review the IAM permissions in the [AWS Setup](#aws-setup) section

4. **CloudFormation Service Role Issues**
   
   **Error**: `User: arn:aws:iam::123456789012:user/username is not authorized to perform: iam:PassRole on resource: arn:aws:iam::123456789012:role/CloudFormation-FISPlayground-Role`
   
   **Solution**: Your user needs `iam:PassRole` permission for the CloudFormation service role:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": "iam:PassRole",
         "Resource": "arn:aws:iam::*:role/CloudFormation-FISPlayground-Role"
       }
     ]
   }
   ```

   **Error**: `Insufficient permissions to create stack`
   
   **Solutions**:
   - **Option 1**: Use your user permissions (leave IAM role empty)
   - **Option 2**: Ensure the CloudFormation service role has all required permissions
   - **Option 3**: Add missing permissions to the service role policy

   **Error**: `Role does not exist or cannot be assumed`
   
   **Solution**: Verify the CloudFormation service role was created correctly:
   ```bash
   # Check if role exists
   aws iam get-role --role-name CloudFormation-FISPlayground-Role
   
   # Check role's trust policy allows CloudFormation
   aws iam get-role --role-name CloudFormation-FISPlayground-Role --query 'Role.AssumeRolePolicyDocument'
   ```

4. **Stack Creation Fails**
   - Check the **Events** tab for specific error messages
   - Common issues: resource limits, naming conflicts, region availability
   - Delete failed stack and retry with different parameters

#### Updating Stack via Console

To update an existing stack:

1. Select your stack in the CloudFormation console
2. Click **"Update"**
3. Choose **"Replace current template"**
4. Upload the updated template file
5. Follow the same configuration steps
6. Review changes in the **"Change set preview"**
7. Execute the update

#### Deleting Stack via Console

To clean up resources:

1. Select your stack in the CloudFormation console
2. Click **"Delete"**
3. Confirm deletion
4. Monitor the **Events** tab for deletion progress
5. Verify all resources are removed

### IAM Role Quick Reference

| Deployment Method | IAM Role Requirement | Permissions Needed |
|-------------------|---------------------|-------------------|
| **Console - User Permissions** | None (leave empty) | Your user needs all CloudFormation + service permissions |
| **Console - Service Role** | `CloudFormation-FISPlayground-Role` | Service role needs service permissions, user needs `iam:PassRole` |
| **Command Line** | None (uses user permissions) | Your user needs all CloudFormation + service permissions |

**Required Service Permissions** (for user or service role):
- `lambda:*` - Lambda function management
- `apigateway:*` - API Gateway management  
- `dynamodb:*` - DynamoDB table management
- `iam:CreateRole`, `iam:AttachRolePolicy`, `iam:PassRole` - IAM role management
- `logs:*` - CloudWatch Logs management
- `s3:GetObject` - S3 access for Lambda code

### Console vs Command Line Comparison

| Feature | Console | Command Line |
|---------|---------|--------------|
| **Ease of Use** | Visual interface, guided process | Requires CLI knowledge |
| **Automation** | Manual process | Fully automated scripts |
| **Customization** | Limited to UI options | Full scripting flexibility |
| **Monitoring** | Real-time visual feedback | Text-based output |
| **Repeatability** | Manual steps each time | Scriptable and repeatable |
| **Advanced Options** | All CloudFormation features | All CloudFormation features |
| **Learning Curve** | Beginner-friendly | Requires CLI familiarity |
| **IAM Role Options** | Can use service role or user permissions | Uses user permissions only |

### When to Use Console Deployment

**Use Console when:**
- You're new to AWS CloudFormation
- You prefer visual interfaces
- You need to explore CloudFormation features
- You're doing one-time deployments
- You want to see detailed deployment progress

**Use Command Line when:**
- You need automated deployments
- You're integrating with CI/CD pipelines
- You prefer scriptable solutions
- You're doing frequent deployments
- You want version-controlled deployment processes

Both methods create identical infrastructure and provide the same functionality.

## API Documentation

The FIS Playground provides a RESTful API for managing items with full CRUD operations.

### Base URL

```
https://{api-id}.execute-api.{region}.amazonaws.com/{environment}
```

### Authentication

Currently, the API does not require authentication. All endpoints are publicly accessible.

### Response Format

All API responses follow a consistent JSON format:

```json
{
  "success": true,
  "data": {
    // Response data here
  },
  "error": null
}
```

Error responses:

```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "ValidationError",
    "code": "INVALID_INPUT",
    "message": "Name is required and cannot be empty",
    "details": {}
  }
}
```

### Endpoints

#### 1. Create Item

**POST** `/items`

Creates a new item in the system.

**Request Body:**
```json
{
  "name": "My Item",
  "description": "A description of my item"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My Item",
    "description": "A description of my item",
    "status": "active",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "error": null
}
```

**Validation Rules:**
- `name`: Required, 1-100 characters
- `description`: Optional, max 500 characters

#### 2. Get Item

**GET** `/items/{id}`

Retrieves a specific item by ID.

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My Item",
    "description": "A description of my item",
    "status": "active",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "error": null
}
```

**Response (404 Not Found):**
```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "NotFoundError",
    "code": "ITEM_NOT_FOUND",
    "message": "Item with ID '550e8400-e29b-41d4-a716-446655440000' not found"
  }
}
```

#### 3. List Items

**GET** `/items`

Retrieves a paginated list of all items.

**Query Parameters:**
- `limit`: Number of items to return (default: 20, max: 100)
- `cursor`: Pagination cursor for next page

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Item 1",
        "description": "First item",
        "status": "active",
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      },
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "name": "Item 2",
        "description": "Second item",
        "status": "inactive",
        "created_at": "2024-01-15T11:00:00Z",
        "updated_at": "2024-01-15T11:00:00Z"
      }
    ],
    "has_more": false,
    "next_cursor": null
  },
  "error": null
}
```

#### 4. Update Item

**PUT** `/items/{id}`

Updates an existing item. Only provided fields will be updated.

**Request Body:**
```json
{
  "name": "Updated Item Name",
  "description": "Updated description",
  "status": "inactive"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Updated Item Name",
    "description": "Updated description",
    "status": "inactive",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T12:00:00Z"
  },
  "error": null
}
```

**Validation Rules:**
- `name`: Optional, 1-100 characters if provided
- `description`: Optional, max 500 characters if provided
- `status`: Optional, must be "active" or "inactive" if provided

#### 5. Delete Item

**DELETE** `/items/{id}`

Deletes an item from the system.

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Item deleted successfully",
    "deleted_id": "550e8400-e29b-41d4-a716-446655440000"
  },
  "error": null
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | OK - Request successful |
| 201 | Created - Item created successfully |
| 400 | Bad Request - Invalid input or malformed request |
| 404 | Not Found - Item not found |
| 500 | Internal Server Error - Server-side error |

### CORS Support

The API includes CORS headers for browser-based applications:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Headers: Content-Type,Authorization`
- `Access-Control-Allow-Methods: GET,POST,PUT,DELETE,OPTIONS`

## Usage Examples

### Using curl

#### Create an Item
```bash
curl -X POST https://your-api-endpoint.amazonaws.com/dev/items \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Item",
    "description": "This is a test item"
  }'
```

#### Get All Items
```bash
curl https://your-api-endpoint.amazonaws.com/dev/items
```

#### Get Specific Item
```bash
curl https://your-api-endpoint.amazonaws.com/dev/items/550e8400-e29b-41d4-a716-446655440000
```

#### Update an Item
```bash
curl -X PUT https://your-api-endpoint.amazonaws.com/dev/items/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Test Item",
    "status": "inactive"
  }'
```

#### Delete an Item
```bash
curl -X DELETE https://your-api-endpoint.amazonaws.com/dev/items/550e8400-e29b-41d4-a716-446655440000
```

### Using JavaScript/Node.js

```javascript
const API_BASE = 'https://your-api-endpoint.amazonaws.com/dev';

// Create item
async function createItem(name, description) {
  const response = await fetch(`${API_BASE}/items`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ name, description })
  });
  return response.json();
}

// Get all items
async function getItems() {
  const response = await fetch(`${API_BASE}/items`);
  return response.json();
}

// Usage
createItem('My Item', 'Item description')
  .then(result => console.log('Created:', result))
  .catch(error => console.error('Error:', error));
```

### Using Python

```python
import requests
import json

API_BASE = 'https://your-api-endpoint.amazonaws.com/dev'

# Create item
def create_item(name, description):
    response = requests.post(
        f'{API_BASE}/items',
        headers={'Content-Type': 'application/json'},
        json={'name': name, 'description': description}
    )
    return response.json()

# Get all items
def get_items():
    response = requests.get(f'{API_BASE}/items')
    return response.json()

# Usage
result = create_item('My Item', 'Item description')
print(f"Created: {result}")
```

## Testing

### Unit Tests

Run unit tests for individual components:

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/handlers/...
```

### Integration Tests

Integration tests require a deployed API endpoint:

```bash
# Set API endpoint (from deployment output)
export API_ENDPOINT=https://your-api-endpoint.amazonaws.com/dev

# Run integration tests
make test-integration

# Or run directly
cd test/integration && go test -v
```

### Manual Testing

Use the provided test script for quick manual verification:

```bash
# Run comprehensive API tests
./scripts/test.sh
```

This script performs:
1. Health check
2. CRUD operations workflow
3. Error scenario testing
4. Response format validation

## Troubleshooting

### Common Deployment Issues

#### 1. AWS Credentials Not Configured

**Error:** `Unable to locate credentials`

**Solution:**
```bash
aws configure
# Or set environment variables
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
export AWS_DEFAULT_REGION=us-east-1
```

#### 2. Insufficient IAM Permissions

**Error:** `User: arn:aws:iam::123456789012:user/username is not authorized to perform: cloudformation:CreateStack`

**Solution:** Ensure your AWS user has the required IAM permissions listed in the [AWS Setup](#aws-setup) section.

#### 3. Stack Already Exists

**Error:** `Stack [fis-playground] already exists`

**Solution:**
```bash
# Update existing stack
./scripts/deploy.sh

# Or use different stack name
./scripts/deploy.sh --stack-name fis-playground-2
```

#### 4. Build Failures

**Error:** `go: command not found`

**Solution:**
- Install Go 1.21 or later
- Ensure Go is in your PATH
- Verify with `go version`

#### 5. Lambda Function Timeout

**Error:** API requests timing out or returning 502 errors

**Solution:**
- Check CloudWatch logs for the Lambda function
- Increase timeout in CloudFormation template if needed
- Verify DynamoDB table exists and is accessible

### API Testing Issues

#### 1. CORS Errors in Browser

**Error:** `Access to fetch at 'https://...' from origin 'http://localhost:3000' has been blocked by CORS policy`

**Solution:** The API includes CORS headers. Ensure you're making requests with proper headers:
```javascript
fetch(url, {
  headers: {
    'Content-Type': 'application/json'
  }
})
```

#### 2. 404 Errors on Valid Endpoints

**Error:** `404 Not Found` on `/items`

**Solution:**
- Verify the API endpoint URL from deployment output
- Ensure the environment path is included (e.g., `/dev/items`)
- Check API Gateway deployment status in AWS Console

### Debugging Tips

1. **Check CloudWatch Logs:**
   ```bash
   aws logs describe-log-groups --log-group-name-prefix "/aws/lambda/fis-playground"
   ```

2. **Validate CloudFormation Template:**
   ```bash
   aws cloudformation validate-template --template-body file://deployments/cloudformation/template.yaml
   ```

3. **Test Lambda Function Directly:**
   ```bash
   aws lambda invoke --function-name fis-playground-dev --payload '{}' response.json
   ```

4. **Monitor API Gateway:**
   - Enable CloudWatch logging in API Gateway
   - Check execution logs for request/response details

## Cleanup

### Complete Resource Cleanup

Remove all AWS resources created by the deployment:

```bash
# Clean up with default stack name
./scripts/cleanup.sh

# Clean up specific stack
./scripts/cleanup.sh --stack-name my-fis-playground --region us-west-2

# Force cleanup without confirmation
./scripts/cleanup.sh --force
```

### Cleanup Options

| Parameter | Description |
|-----------|-------------|
| `--stack-name` | CloudFormation stack name to delete |
| `--region` | AWS region where stack is deployed |
| `--force` | Skip confirmation prompt |
| `--no-verify` | Skip resource cleanup verification |

### Verification

The cleanup script automatically verifies that all resources are removed:
- DynamoDB tables
- Lambda functions
- API Gateway APIs
- IAM roles and policies

### Manual Cleanup

If automatic cleanup fails, manually delete resources in this order:

1. **CloudFormation Stack:**
   ```bash
   aws cloudformation delete-stack --stack-name fis-playground
   ```

2. **Verify Deletion:**
   ```bash
   aws cloudformation describe-stacks --stack-name fis-playground
   ```

3. **Check for Orphaned Resources:**
   - DynamoDB tables with prefix `fis-playground-items-`
   - Lambda functions with prefix `fis-playground-`
   - API Gateway APIs with name `fis-playground-api-`

### Cost Considerations

The FIS Playground uses on-demand pricing for all services:
- **DynamoDB**: Pay per request (very low cost for testing)
- **Lambda**: Pay per invocation and duration (free tier available)
- **API Gateway**: Pay per API call (free tier available)

Typical costs for light testing: **< $1 per month**

---

## Project Structure

```
fis-playground/
├── cmd/
│   └── lambda/                 # Lambda function entry point
│       └── main.go
├── internal/
│   ├── handlers/               # HTTP request handlers
│   │   ├── handlers.go
│   │   ├── handlers_test.go
│   │   └── errors.go
│   ├── models/                 # Data models and types
│   │   └── item.go
│   └── repository/             # Data access layer
│       ├── dynamodb.go
│       ├── client.go
│       ├── errors.go
│       └── example.go
├── deployments/
│   └── cloudformation/         # Infrastructure as Code
│       ├── template.yaml
│       └── template-ui.yaml
├── scripts/                    # Build and deployment automation
│   ├── build.sh               # Build Lambda function
│   ├── deploy.sh              # Deploy CloudFormation stack
│   ├── cleanup.sh             # Clean up resources
│   └── test.sh                # Run tests
├── test/
│   └── integration/           # Integration tests
│       ├── api_test.go
│       ├── go.mod
│       ├── run_tests.sh
│       └── README.md
├── go.mod                     # Go module definition
├── go.sum                     # Go module checksums
├── Makefile                   # Build automation
└── README.md                  # This documentation
```

## Contributing

This project is designed for learning and experimentation. Feel free to:
- Add new endpoints
- Implement additional fault injection scenarios
- Enhance error handling
- Add monitoring and observability features

## License

This project is provided as-is for educational purposes.