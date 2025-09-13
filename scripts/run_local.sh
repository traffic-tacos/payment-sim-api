#!/bin/bash

# Payment Simulator API Local Development Script

set -e

# Default values
PORT=${PORT:-8080}
WEBHOOK_SECRET=${WEBHOOK_SECRET:-"dev-secret-key-change-in-production"}
LOG_LEVEL=${LOG_LEVEL:-"info"}
OTEL_ENDPOINT=${OTEL_EXPORTER_OTLP_ENDPOINT:-"http://localhost:4317"}

echo "🚀 Starting Payment Simulator API (Local Development)"
echo "=================================================="
echo "Port: $PORT"
echo "Webhook Secret: $WEBHOOK_SECRET"
echo "Log Level: $LOG_LEVEL"
echo "OTEL Endpoint: $OTEL_ENDPOINT"
echo "=================================================="

# Build the application
echo "📦 Building application..."
make build

# Set environment variables
export PORT=$PORT
export WEBHOOK_SECRET=$WEBHOOK_SECRET
export LOG_LEVEL=$LOG_LEVEL
export OTEL_EXPORTER_OTLP_ENDPOINT=$OTEL_ENDPOINT

# Default webhook settings (can be overridden)
export DEFAULT_APPROVE_DELAY_MS=${DEFAULT_APPROVE_DELAY_MS:-200}
export DEFAULT_FAIL_DELAY_MS=${DEFAULT_FAIL_DELAY_MS:-100}
export DEFAULT_DELAY_DELAY_MS=${DEFAULT_DELAY_DELAY_MS:-3000}
export RANDOM_APPROVE_RATE=${RANDOM_APPROVE_RATE:-0.8}
export WEBHOOK_TIMEOUT_MS=${WEBHOOK_TIMEOUT_MS:-1000}
export WEBHOOK_MAX_RETRIES=${WEBHOOK_MAX_RETRIES:-5}
export WEBHOOK_BACKOFF_MS=${WEBHOOK_BACKOFF_MS:-1000}
export WEBHOOK_MAX_RPS=${WEBHOOK_MAX_RPS:-500}

echo "🔧 Environment variables set"
echo "💡 API Documentation: http://localhost:$PORT/openapi/payment-sim.yaml"
echo "📊 Metrics: http://localhost:$PORT/metrics"
echo "🏥 Health Check: http://localhost:$PORT/healthz"
echo ""

# Run the application
echo "▶️  Starting server..."
./bin/payment-sim-api
