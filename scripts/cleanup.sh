#!/bin/bash

# Cleanup script for FIS Playground CloudFormation stack
set -e

# Default values
STACK_NAME="fis-playground"
REGION="us-east-1"
FORCE_DELETE="false"
VERIFY_CLEANUP="true"

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
    
    log_success "AWS prerequisites check completed"
}

# Get stack resources before deletion for verification
get_stack_resources() {
    log_info "Collecting stack resources for cleanup verification..."
    
    local resources_file="/tmp/stack-resources-$STACK_NAME.json"
    
    if aws cloudformation list-stack-resources \
        --stack-name "$STACK_NAME" \
        --region "$REGION" \
        --output json > "$resources_file" 2>/dev/null; then
        
        # Extract resource information
        STACK_RESOURCES=$(jq -r '.StackResourceSummaries[] | "\(.ResourceType):\(.PhysicalResourceId)"' "$resources_file" 2>/dev/null || echo "")
        
        if [[ -n "$STACK_RESOURCES" ]]; then
            log_info "Found the following resources to be deleted:"
            echo "$STACK_RESOURCES" | while read -r resource; do
                log_info "  - $resource"
            done
        fi
        
        rm -f "$resources_file"
    else
        log_warning "Could not retrieve stack resources list"
        STACK_RESOURCES=""
    fi
}

# Verify resources are actually deleted
verify_resource_cleanup() {
    if [[ "$VERIFY_CLEANUP" != "true" ]] || [[ -z "$STACK_RESOURCES" ]]; then
        return 0
    fi
    
    log_info "Verifying resource cleanup..."
    
    local cleanup_issues=0
    
    # Check each resource type that was in the stack
    echo "$STACK_RESOURCES" | while read -r resource; do
        if [[ -z "$resource" ]]; then
            continue
        fi
        
        local resource_type=$(echo "$resource" | cut -d':' -f1)
        local resource_id=$(echo "$resource" | cut -d':' -f2)
        
        case "$resource_type" in
            "AWS::DynamoDB::Table")
                if aws dynamodb describe-table --table-name "$resource_id" --region "$REGION" >/dev/null 2>&1; then
                    log_warning "DynamoDB table still exists: $resource_id"
                    cleanup_issues=$((cleanup_issues + 1))
                fi
                ;;
            "AWS::Lambda::Function")
                if aws lambda get-function --function-name "$resource_id" --region "$REGION" >/dev/null 2>&1; then
                    log_warning "Lambda function still exists: $resource_id"
                    cleanup_issues=$((cleanup_issues + 1))
                fi
                ;;
            "AWS::ApiGateway::RestApi")
                if aws apigateway get-rest-api --rest-api-id "$resource_id" --region "$REGION" >/dev/null 2>&1; then
                    log_warning "API Gateway still exists: $resource_id"
                    cleanup_issues=$((cleanup_issues + 1))
                fi
                ;;
        esac
    done
    
    if [[ $cleanup_issues -gt 0 ]]; then
        log_warning "Found $cleanup_issues resources that may not have been properly cleaned up"
        log_warning "This could be due to eventual consistency or dependencies"
        log_info "You may want to check the AWS console to verify complete cleanup"
    else
        log_success "Resource cleanup verification completed - no issues found"
    fi
}

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
    --force)
      FORCE_DELETE="true"
      shift
      ;;
    --no-verify)
      VERIFY_CLEANUP="false"
      shift
      ;;
    --help)
      echo "Usage: $0 [OPTIONS]"
      echo "Options:"
      echo "  --stack-name    CloudFormation stack name (default: fis-playground)"
      echo "  --region        AWS region (default: us-east-1)"
      echo "  --force         Skip confirmation prompt"
      echo "  --no-verify     Skip resource cleanup verification"
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

log_info "Cleaning up FIS Playground..."
log_info "Stack Name: $STACK_NAME"
log_info "Region: $REGION"

# Check if stack exists
STACK_STATUS=""
if aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" >/dev/null 2>&1; then
    STACK_STATUS=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" --query 'Stacks[0].StackStatus' --output text)
    log_info "Stack exists with status: $STACK_STATUS"
    
    # Check if stack is in a state that prevents deletion
    case "$STACK_STATUS" in
        "DELETE_IN_PROGRESS")
            log_info "Stack deletion already in progress"
            log_info "Waiting for deletion to complete..."
            ;;
        "DELETE_FAILED")
            log_warning "Stack is in DELETE_FAILED state"
            log_warning "You may need to manually resolve issues before deletion can proceed"
            if [[ "$FORCE_DELETE" != "true" ]]; then
                read -p "Do you want to attempt deletion anyway? (y/N): " -n 1 -r
                echo
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    log_info "Cleanup cancelled by user"
                    exit 0
                fi
            fi
            ;;
    esac
else
    log_info "Stack '$STACK_NAME' does not exist in region '$REGION'"
    log_success "No cleanup required"
    exit 0
fi

# Get stack resources before deletion
get_stack_resources

# Confirmation prompt (unless forced)
if [[ "$FORCE_DELETE" != "true" ]]; then
    echo
    log_warning "This will permanently delete the CloudFormation stack and all its resources!"
    read -p "Are you sure you want to proceed? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Cleanup cancelled by user"
        exit 0
    fi
fi

# Delete the CloudFormation stack
if [[ "$STACK_STATUS" != "DELETE_IN_PROGRESS" ]]; then
    log_info "Initiating CloudFormation stack deletion..."
    if ! aws cloudformation delete-stack \
        --stack-name "$STACK_NAME" \
        --region "$REGION"; then
        log_error "Failed to initiate stack deletion"
        exit 1
    fi
fi

# Wait for deletion to complete with timeout and monitoring
log_info "Waiting for stack deletion to complete..."
WAIT_START_TIME=$(date +%s)
TIMEOUT_SECONDS=1800  # 30 minutes

# Monitor deletion progress
monitor_deletion() {
    local last_event_time=""
    while true; do
        # Check current stack status
        local current_status=""
        if aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" >/dev/null 2>&1; then
            current_status=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" --query 'Stacks[0].StackStatus' --output text 2>/dev/null || echo "")
        else
            # Stack no longer exists - deletion completed
            log_success "Stack deletion completed successfully"
            return 0
        fi
        
        case "$current_status" in
            "DELETE_COMPLETE")
                log_success "Stack deletion completed successfully"
                return 0
                ;;
            "DELETE_FAILED")
                log_error "Stack deletion failed"
                
                # Show recent stack events for debugging
                log_info "Recent stack events:"
                aws cloudformation describe-stack-events \
                    --stack-name "$STACK_NAME" \
                    --region "$REGION" \
                    --query 'StackEvents[0:10].[Timestamp,LogicalResourceId,ResourceStatus,ResourceStatusReason]' \
                    --output table 2>/dev/null || true
                
                return 1
                ;;
        esac
        
        # Show recent deletion events
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
        
        # Check timeout
        local current_time=$(date +%s)
        if (( current_time - WAIT_START_TIME > TIMEOUT_SECONDS )); then
            log_error "Deletion timeout after $((TIMEOUT_SECONDS/60)) minutes"
            return 1
        fi
        
        sleep 10
    done
}

# Start monitoring and wait for completion
if ! monitor_deletion; then
    log_error "Stack deletion failed or timed out"
    exit 1
fi

# Verify cleanup
verify_resource_cleanup

log_success "Cleanup completed successfully!"
log_success "All resources have been removed."