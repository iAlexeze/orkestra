#!/bin/bash
# tests/e2e/run.sh
set -e

TEST_NAME=$1
TIMEOUT=${2:-60}

echo "=== Running E2E Test: $TEST_NAME ==="

# Setup
kind create cluster --name orkestra-test
trap "kind delete cluster --name orkestra-test" EXIT

# Build Orkestra
go build -o /tmp/ork ./cmd/ork

# Run test
./tests/e2e/$TEST_NAME/test.sh

echo "=== E2E Test Passed ==="