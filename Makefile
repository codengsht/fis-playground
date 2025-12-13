# FIS Playground Makefile

.PHONY: build deploy test clean help fmt vet mod-tidy check clean-build integration-test

# Build configuration
GOOS=linux
GOARCH=amd64
CGO_ENABLED=0
BUILD_DIR=build
BINARY_NAME=main
PACKAGE_NAME=lambda-deployment.zip

# Default target
help:
	@echo "FIS Playground - Available commands:"
	@echo "  build            - Build the Lambda function"
	@echo "  deploy           - Deploy the CloudFormation stack"
	@echo "  test             - Run unit tests and code quality checks"
	@echo "  integration-test - Run end-to-end integration tests"
	@echo "  clean            - Clean up AWS resources"
	@echo "  help             - Show this help message"

# Build the Lambda function
build:
	@echo "Building Lambda function for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	@cd cmd/lambda && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -ldflags="-s -w" -trimpath -o ../../$(BUILD_DIR)/$(BINARY_NAME) .
	@chmod 755 $(BUILD_DIR)/$(BINARY_NAME)
	@cd $(BUILD_DIR) && zip -r $(PACKAGE_NAME) $(BINARY_NAME)
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

# Development targets
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

mod-tidy:
	@echo "Tidying Go modules..."
	go mod tidy

# Run integration tests
integration-test:
	@echo "Running integration tests..."
	./test/integration/run_tests.sh

# Combined development check
check: fmt vet test
	@echo "All checks passed!"