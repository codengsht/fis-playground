#!/bin/bash

# Test script for FIS Playground
set -e

echo "Running FIS Playground tests..."

if ! command -v node >/dev/null 2>&1; then
    echo "Node.js is required to run tests."
    exit 1
fi

echo "Running Node.js unit tests..."
node --test test/lambda/handler.test.js

echo "All tests passed successfully!"
