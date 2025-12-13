#!/bin/bash

# Simple build script wrapper - delegates to Makefile
# This script exists for backward compatibility

set -e

echo "Delegating to Makefile build target..."
make build