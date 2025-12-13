#!/bin/bash

# Test script for FIS Playground
set -e

echo "Running FIS Playground tests..."

# Run Go tests
echo "Running unit tests..."
go test -v ./...

# Run Go vet for static analysis
echo "Running go vet..."
go vet ./...

# Run Go fmt check
echo "Checking code formatting..."
if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then
    echo "Code is not properly formatted. Run 'go fmt ./...' to fix."
    gofmt -l .
    exit 1
fi

echo "All tests passed successfully!"