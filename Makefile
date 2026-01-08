# FIS Playground Makefile

.PHONY: build deploy test clean help check clean-build

# Build configuration
BUILD_DIR=build
LAMBDA_SRC=lambda-nodejs
PACKAGE_NAME=lambda-deployment.zip

# Default target
help:
	@echo "FIS Playground - Available commands:"
	@echo "  build            - Build the Lambda function"
	@echo "  deploy           - Deploy the CloudFormation stack"
	@echo "  test             - Run unit tests"
	@echo "  clean            - Clean up AWS resources"
	@echo "  help             - Show this help message"

# Build the Lambda function
build:
	@echo "Building Lambda function (Node.js)..."
	@mkdir -p $(BUILD_DIR)
	@cd $(LAMBDA_SRC) && npm ci --omit=dev
	@cd $(LAMBDA_SRC) && zip -r ../$(BUILD_DIR)/$(PACKAGE_NAME) .
	@echo "Build completed: $(BUILD_DIR)/$(PACKAGE_NAME)"

# Deploy the CloudFormation stack
deploy: build
	@echo "Deploying infrastructure..."
	./scripts/deploy.sh

# Run tests
test:
	@echo "Running tests..."
	./scripts/test.sh

# Clean build artifacts
clean-build:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

# Clean up AWS resources
clean:
	@echo "Cleaning up resources..."
	./scripts/cleanup.sh

# Combined development check
check: test
	@echo "All checks passed!"
