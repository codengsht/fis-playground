#!/bin/bash

# Deployment script for FIS Playground CloudFormation stack
set -e

# Default values
STACK_NAME="fis-playground"
ENVIRONMENT="dev"
REGION="us-east-1"
TEMPLATE_FILE="deployments/cloudformation/template.yaml"
ROLLBACK_ON_FAILURE="true"

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

# Validation functions
validate_parameters() {
    log_info "Validating deployment parameters..."
    
    # Validate stack name
    if [[ ! "$STACK_NAME" =~ ^[a-zA-Z][a-zA-Z0-9-]*$ ]]; then
        log_error "Invalid stack name: $STACK_NAME. Must start with a letter and contain only alphanumeric characters and hyphens."
        exit 1
    fi
    
    # Validate environment
    if [[ ! "$ENVIRONMENT" =~ ^[a-zA-Z0-9-]+$ ]]; then
        log_error "Invalid environment name: $ENVIRONMENT. Must contain only alphanumeric characters and hyphens."
        exit 1
    fi
    
    # Validate region format
    if [[ ! "$REGION" =~ ^[a-z0-9-]+$ ]]; then
        log_error "Invalid region format: $REGION"
        exit 1
    fi
    
    # Check if template file exists
    if [[ ! -f "$TEMPLATE_FILE" ]]; then
        log_error "CloudFormation template file not found: $TEMPLATE_FILE"
        exit 1
    fi
    
    # Validate template syntax
    log_info "Validating CloudFormation template syntax..."
    if ! aws cloudformation validate-template --template-body file://"$TEMPLATE_FILE" --region "$REGION" >/dev/null 2>&1; then
        log_error "CloudFormation template validation failed"
        aws cloudformation validate-template --template-body file://"$TEMPLATE_FILE" --region "$REGION"
        exit 1
    fi
    
    log_success "Parameter validation completed"
}

# Check AWS CLI and credentials
check_aws_prerequisites() {
    log_info "Checking AWS prerequisites..."
    
    # Check if AWS CLI is installed
    if ! command -v aws &> /dev/null; then
        log_error "AWS CLI is not installed or not in PATH"
        exit 1
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity --region "$REGION" >/dev/null 2>&1; then
        log_error "AWS credentials not configured or invalid for region $REGION"
        exit 1
    fi
    
    # Check if region is valid
    if ! aws ec2 describe-regions --region-names "$REGION" >/dev/null 2>&1; then
        log_error "Invalid AWS region: $REGION"
        exit 1
    fi
    
    log_success "AWS prerequisites check completed"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --stack-name)
      STACK_NAME="$2"
      shift 2
      ;;
    --environment)
      ENVIRONMENT="$2"
      shift 2
      ;;
    --region)
      REGION="$2"
      shift 2
      ;;
    --no-rollback)
      ROLLBACK_ON_FAILURE="false"
      shift
      ;;
    --help)
      echo "Usage: $0 [OPTIONS]"
      echo "Options:"
      echo "  --stack-name    CloudFormation stack name (default: fis-playground)"
      echo "  --environment   Environment name (default: dev)"
      echo "  --region        AWS region (default: us-east-1)"
      echo "  --no-rollback   Disable rollback on failure (default: enabled)"
      echo "  --help          Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Run prerequisite checks
check_aws_prerequisites
validate_parameters

log_info "Deploying FIS Playground..."
log_info "Stack Name: $STACK_NAME"
log_info "Environment: $ENVIRONMENT"
log_info "Region: $REGION"
log_info "Rollback on failure: $ROLLBACK_ON_FAILURE"

# Build the Lambda function first
log_info "Building Lambda function..."
if ! ./scripts/build.sh; then
    log_error "Lambda function build failed"
    exit 1
fi

# Check if stack exists and get current status
STACK_EXISTS=false
STACK_STATUS=""
if aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" >/dev/null 2>&1; then
    STACK_EXISTS=true
    STACK_STATUS=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" --query 'Stacks[0].StackStatus' --output text)
    log_info "Stack exists with status: $STACK_STATUS"
    
    # Check if stack is in a failed state that requires cleanup
    case "$STACK_STATUS" in
        "ROLLBACK_COMPLETE"|"CREATE_FAILED"|"DELETE_FAILED")
            log_warning "Stack is in a failed state: $STACK_STATUS"
            log_warning "You may need to delete the stack manually before redeploying"
            exit 1
            ;;
        "UPDATE_ROLLBACK_COMPLETE"|"UPDATE_ROLLBACK_FAILED")
            log_warning "Stack is in rollback state: $STACK_STATUS"
            log_warning "Previous update failed. Proceeding with new update attempt..."
            ;;
    esac
else
    log_info "Stack does not exist, will create new stack"
fi

# Determine operation type
if [ "$STACK_EXISTS" = true ]; then
    OPERATION="update-stack"
    WAIT_CONDITION="stack-update-complete"
    log_info "Updating existing stack..."
else
    OPERATION="create-stack"
    WAIT_CONDITION="stack-create-complete"
    log_info "Creating new stack..."
fi

# Prepare CloudFormation parameters
CF_PARAMS=(
    --stack-name "$STACK_NAME"
    --template-body file://"$TEMPLATE_FILE"
    --parameters ParameterKey=Environment,ParameterValue="$ENVIRONMENT"
    --capabilities CAPABILITY_IAM
    --region "$REGION"
)

# Add rollback configuration for create operations
if [ "$OPERATION" = "create-stack" ] && [ "$ROLLBACK_ON_FAILURE" = "false" ]; then
    CF_PARAMS+=(--disable-rollback)
fi

# Deploy the CloudFormation stack with error handling
log_info "Executing CloudFormation $OPERATION..."
if ! aws cloudformation "$OPERATION" "${CF_PARAMS[@]}"; then
    log_error "CloudFormation $OPERATION command failed"
    exit 1
fi

# Wait for deployment to complete with timeout and error handling
log_info "Waiting for deployment to complete..."
WAIT_START_TIME=$(date +%s)
TIMEOUT_SECONDS=1800  # 30 minutes

# Monitor stack events during deployment
monitor_stack_events() {
    local last_event_time=""
    while true; do
        # Get recent stack events
        local events=$(aws cloudformation describe-stack-events \
            --stack-name "$STACK_NAME" \
            --region "$REGION" \
            --query 'StackEvents[?Timestamp>`'"$last_event_time"'`].[Timestamp,LogicalResourceId,ResourceStatus,ResourceStatusReason]' \
            --output text 2>/dev/null || echo "")
        
        if [[ -n "$events" ]]; then
            echo "$events" | while IFS=$'\t' read -r timestamp resource_id status reason; do
                if [[ -n "$timestamp" ]]; then
                    log_info "[$timestamp] $resource_id: $status ${reason:+- $reason}"
                    last_event_time="$timestamp"
                fi
            done
        fi
        
        # Check if operation completed
        local current_status=$(aws cloudformation describe-stacks \
            --stack-name "$STACK_NAME" \
            --region "$REGION" \
            --query 'Stacks[0].StackStatus' \
            --output text 2>/dev/null || echo "")
        
        case "$current_status" in
            "CREATE_COMPLETE"|"UPDATE_COMPLETE")
                log_success "Stack operation completed successfully"
                return 0
                ;;
            "CREATE_FAILED"|"UPDATE_FAILED"|"ROLLBACK_COMPLETE"|"UPDATE_ROLLBACK_COMPLETE")
                log_error "Stack operation failed with status: $current_status"
                return 1
                ;;
        esac
        
        # Check timeout
        local current_time=$(date +%s)
        if (( current_time - WAIT_START_TIME > TIMEOUT_SECONDS )); then
            log_error "Deployment timeout after $((TIMEOUT_SECONDS/60)) minutes"
            return 1
        fi
        
        sleep 10
    done
}

# Start monitoring in background and wait for completion
if ! monitor_stack_events; then
    log_error "Stack deployment failed"
    
    # Show recent stack events for debugging
    log_info "Recent stack events:"
    aws cloudformation describe-stack-events \
        --stack-name "$STACK_NAME" \
        --region "$REGION" \
        --query 'StackEvents[0:10].[Timestamp,LogicalResourceId,ResourceStatus,ResourceStatusReason]' \
        --output table
    
    exit 1
fi

# Get and display stack outputs
log_success "Deployment completed successfully!"
log_info "Stack outputs:"
aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[*].[OutputKey,OutputValue]' \
    --output table

# Verify deployment by checking key resources
log_info "Verifying deployment..."
API_ENDPOINT=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`ApiEndpoint`].OutputValue' \
    --output text)

if [[ -n "$API_ENDPOINT" ]]; then
    log_success "API endpoint available: $API_ENDPOINT"
    log_info "You can test the API using: curl $API_ENDPOINT/items"
else
    log_warning "API endpoint not found in stack outputs"
fi

log_success "Deployment verification completed"