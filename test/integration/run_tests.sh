#!/bin/bash

# Integration test runner for FIS Playground API
set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Default values
STACK_NAME="fis-playground"
REGION="us-east-1"
TIMEOUT="30s"
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --stack-name)
      STACK_NAME="$2"
      shift 2
      ;;
    --region)
      REGION="$2"
      shift 2
      ;;
    --timeout)
      TIMEOUT="$2"
      shift 2
      ;;
    --verbose)
      VERBOSE=true
      shift
      ;;
    --help)
      echo "Usage: $0 [OPTIONS]"
      echo "Options:"
      echo "  --stack-name    CloudFormation stack name (default: fis-playground)"
      echo "  --region        AWS region (default: us-east-1)"
      echo "  --timeout       Test timeout (default: 30s)"
      echo "  --verbose       Enable verbose output"
      echo "  --help          Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

log_info "Starting integration tests for FIS Playground API"
log_info "Stack Name: $STACK_NAME"
log_info "Region: $REGION"
log_info "Timeout: $TIMEOUT"

# Check prerequisites
log_info "Checking prerequisites..."

# Check if AWS CLI is available
if ! command -v aws &> /dev/null; then
    log_error "AWS CLI is not installed or not in PATH"
    exit 1
fi

# Check if Go is available
if ! command -v go &> /dev/null; then
    log_error "Go is not installed or not in PATH"
    exit 1
fi

# Check AWS credentials
if ! aws sts get-caller-identity --region "$REGION" >/dev/null 2>&1; then
    log_error "AWS credentials not configured or invalid for region $REGION"
    exit 1
fi

# Get API endpoint from CloudFormation stack
log_info "Retrieving API endpoint from CloudFormation stack..."
API_ENDPOINT=""

if aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" >/dev/null 2>&1; then
    API_ENDPOINT=$(aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$REGION" \
        --query 'Stacks[0].Outputs[?OutputKey==`ApiEndpoint`].OutputValue' \
        --output text 2>/dev/null || echo "")
    
    if [[ -n "$API_ENDPOINT" && "$API_ENDPOINT" != "None" ]]; then
        log_success "API endpoint found: $API_ENDPOINT"
    else
        log_error "API endpoint not found in stack outputs"
        log_info "Available stack outputs:"
        aws cloudformation describe-stacks \
            --stack-name "$STACK_NAME" \
            --region "$REGION" \
            --query 'Stacks[0].Outputs[*].[OutputKey,OutputValue]' \
            --output table
        exit 1
    fi
else
    log_error "CloudFormation stack '$STACK_NAME' not found in region '$REGION'"
    log_info "Available stacks:"
    aws cloudformation list-stacks \
        --region "$REGION" \
        --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE \
        --query 'StackSummaries[*].[StackName,StackStatus]' \
        --output table
    exit 1
fi

# Verify API endpoint is accessible
log_info "Verifying API endpoint accessibility..."
if ! curl -s --max-time 10 "$API_ENDPOINT/health" >/dev/null 2>&1; then
    log_warning "API endpoint may not be accessible or healthy"
    log_info "Attempting to get health check response..."
    curl -s --max-time 10 "$API_ENDPOINT/health" || true
    echo ""
    log_warning "Continuing with tests anyway..."
else
    log_success "API endpoint is accessible"
fi

# Set environment variables for tests
export API_ENDPOINT="$API_ENDPOINT"
export AWS_REGION="$REGION"

# Change to test directory
cd "$(dirname "$0")"

# Run the integration tests
log_info "Running integration tests..."

# Set Go test flags
GO_TEST_FLAGS="-v -timeout=$TIMEOUT"
if [ "$VERBOSE" = true ]; then
    GO_TEST_FLAGS="$GO_TEST_FLAGS -test.v"
fi

# Run tests with proper error handling
if go test $GO_TEST_FLAGS ./...; then
    log_success "All integration tests passed!"
    exit 0
else
    TEST_EXIT_CODE=$?
    log_error "Integration tests failed with exit code $TEST_EXIT_CODE"
    
    # Provide debugging information
    log_info "Debugging information:"
    log_info "API Endpoint: $API_ENDPOINT"
    log_info "AWS Region: $REGION"
    log_info "Stack Name: $STACK_NAME"
    
    # Check if stack is still healthy
    STACK_STATUS=$(aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$REGION" \
        --query 'Stacks[0].StackStatus' \
        --output text 2>/dev/null || echo "UNKNOWN")
    
    log_info "Stack Status: $STACK_STATUS"
    
    if [[ "$STACK_STATUS" != "CREATE_COMPLETE" && "$STACK_STATUS" != "UPDATE_COMPLETE" ]]; then
        log_warning "Stack is not in a healthy state. This may be causing test failures."
    fi
    
    # Try a simple health check
    log_info "Attempting health check..."
    if curl -s --max-time 10 "$API_ENDPOINT/health"; then
        echo ""
        log_info "Health check succeeded - API is responding"
    else
        echo ""
        log_warning "Health check failed - API may be down"
    fi
    
    exit $TEST_EXIT_CODE
fi