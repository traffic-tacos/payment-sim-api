# Payment Simulator API Makefile

.PHONY: help build test clean docker-build docker-run lint fmt vet mod-tidy run-local stop-local

# Default target
help: ## Show this help message
	@echo "Payment Simulator API"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the application
	@echo "Building payment-sim-api..."
	@go build -o bin/payment-sim-api ./cmd/payment-sim-api

build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build -o bin/payment-sim-api-linux ./cmd/payment-sim-api

build-arm64: ## Build for ARM64
	@echo "Building for ARM64..."
	@GOOS=linux GOARCH=arm64 go build -o bin/payment-sim-api-arm64 ./cmd/payment-sim-api

# Test targets
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v ./internal/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v ./test/...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Development targets
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@golangci-lint run

mod-tidy: ## Tidy Go modules
	@echo "Tidying modules..."
	@go mod tidy

# Local development
run-local: ## Run locally with default config
	@echo "Starting payment-sim-api locally..."
	@WEBHOOK_SECRET=test-secret ./bin/payment-sim-api

run-local-bg: ## Run locally in background
	@echo "Starting payment-sim-api locally in background..."
	@WEBHOOK_SECRET=test-secret ./bin/payment-sim-api &

stop-local: ## Stop local instance
	@echo "Stopping local payment-sim-api..."
	@pkill -f payment-sim-api || true

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t traffic-tacos/payment-sim-api:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 -e WEBHOOK_SECRET=test-secret traffic-tacos/payment-sim-api:latest

# Deployment artifacts
artifacts: build-linux build-arm64 ## Build deployment artifacts

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# CI/CD targets
ci: mod-tidy fmt vet lint test build ## Run full CI pipeline

# Performance testing
perf-test: ## Run performance tests
	@echo "Running performance tests..."
	@k6 run test/performance/load_test.js

# Development setup
setup: ## Setup development environment
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install go.k6.io/k6@latest

# All targets
all: clean mod-tidy fmt vet lint test build artifacts ## Run everything
