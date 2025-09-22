# Payment Sim API Makefile
# Following Traffic Tacos MSA development standards

.PHONY: help build test lint docker-build ci run-local perf-test clean generate

# Default goal
.DEFAULT_GOAL := help

# Variables
APP_NAME = payment-sim-api
BINARY_NAME = bin/$(APP_NAME)
DOCKER_IMAGE = $(APP_NAME):latest
GRPC_PORT = 8030
HEALTH_PORT = 8031

# Build settings
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS = -ldflags="-w -s"

help: ## Show this help message
	@echo "Payment Sim API - Traffic Tacos MSA"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application binary
	@echo "Building $(APP_NAME)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/$(APP_NAME)
	@echo "✓ Build complete: $(BINARY_NAME)"

test: ## Run all tests
	@echo "Running tests..."
	go test -v -race -cover ./...
	@echo "✓ Tests complete"

lint: ## Run linting
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not found, running go vet..."; \
		go vet ./...; \
	fi
	@echo "✓ Linting complete"

generate: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@if command -v buf >/dev/null 2>&1; then \
		buf generate; \
	else \
		echo "buf not found, skipping proto generation"; \
	fi
	@echo "✓ Code generation complete"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✓ Docker image built: $(DOCKER_IMAGE)"

ci: lint test build docker-build ## Run full CI pipeline

run-local: build ## Run the application locally
	@echo "Starting $(APP_NAME) locally..."
	@echo "gRPC server: localhost:$(GRPC_PORT)"
	@echo "Health/Metrics server: http://localhost:$(HEALTH_PORT)"
	@if [ -z "$(WEBHOOK_SECRET)" ]; then \
		echo "Warning: WEBHOOK_SECRET not set, using default"; \
		WEBHOOK_SECRET=local-dev-secret ./$(BINARY_NAME); \
	else \
		./$(BINARY_NAME); \
	fi

perf-test: ## Run performance tests
	@echo "Running performance tests..."
	@if command -v ghz >/dev/null 2>&1; then \
		echo "Testing gRPC performance..."; \
		ghz --insecure --proto proto/payment/v1/payment.proto \
			--call payment.v1.PaymentService.CreatePaymentIntent \
			--data '{"reservation_id":"test","user_id":"user123","amount":{"amount":100000,"currency":"KRW"},"scenario":"approve"}' \
			--duration=30s --concurrency=10 \
			localhost:$(GRPC_PORT); \
	else \
		echo "ghz not found, install with: go install github.com/bojand/ghz/cmd/ghz@latest"; \
	fi
	@if command -v hey >/dev/null 2>&1; then \
		echo "Testing REST performance..."; \
		hey -z 30s -c 10 -H "Content-Type: application/json" \
			-d '{"reservation_id":"test","user_id":"user123","amount":100000,"currency":"KRW","scenario":"approve"}' \
			http://localhost:$(REST_PORT)/v1/sim/intent; \
	else \
		echo "hey not found, install with: go install github.com/rakyll/hey@latest"; \
	fi
	@echo "✓ Performance tests complete"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf dist/
	docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "✓ Clean complete"

# Development helpers
dev-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/bojand/ghz/cmd/ghz@latest
	go install github.com/rakyll/hey@latest
	@if ! command -v buf >/dev/null 2>&1; then \
		echo "Installing buf..."; \
		curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(shell uname -s)-$(shell uname -m)" -o "/usr/local/bin/buf"; \
		chmod +x "/usr/local/bin/buf"; \
	fi
	@echo "✓ Development dependencies installed"

grpcui: ## Start grpcui for gRPC testing
	@echo "Starting grpcui on http://localhost:8080..."
	@echo "Make sure the gRPC server is running on port $(GRPC_PORT)"
	grpcui -plaintext localhost:$(GRPC_PORT)

# Docker helpers
docker-run: docker-build ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p $(GRPC_PORT):$(GRPC_PORT) -p $(HEALTH_PORT):$(HEALTH_PORT) \
		-e ENVIRONMENT=development \
		-e WEBHOOK_SECRET=docker-dev-secret \
		$(DOCKER_IMAGE)

docker-shell: ## Get shell access to Docker container
	docker run --rm -it --entrypoint /bin/sh $(DOCKER_IMAGE)

# AWS and deployment helpers
env-template: ## Generate .env template file
	@echo "Creating .env.template..."
	@echo "✓ .env.template created"

check-env: ## Check required environment variables
	@echo "Checking environment configuration..."
	@if [ -z "$(AWS_PROFILE)" ]; then echo "❌ AWS_PROFILE not set (should be 'tacos')"; else echo "✓ AWS_PROFILE: $(AWS_PROFILE)"; fi
	@if [ -z "$(AWS_REGION)" ]; then echo "❌ AWS_REGION not set"; else echo "✓ AWS_REGION: $(AWS_REGION)"; fi
	@if [ -z "$(WEBHOOK_SECRET)" ]; then echo "⚠️  WEBHOOK_SECRET not set (will use default)"; else echo "✓ WEBHOOK_SECRET configured"; fi

# Status and info
status: ## Show service status and information
	@echo "Payment Sim API Status"
	@echo "====================="
	@echo "Binary: $(BINARY_NAME)"
	@echo "Docker Image: $(DOCKER_IMAGE)"
	@echo "gRPC Port: $(GRPC_PORT)"
	@echo "Health/Metrics Port: $(HEALTH_PORT)"
	@echo ""
	@if [ -f "$(BINARY_NAME)" ]; then echo "✓ Binary exists"; else echo "❌ Binary not found (run 'make build')"; fi
	@if docker images $(DOCKER_IMAGE) --format "table {{.Repository}}:{{.Tag}}" | grep -q $(APP_NAME); then echo "✓ Docker image exists"; else echo "❌ Docker image not found (run 'make docker-build')"; fi
	@echo ""
	@echo "Quick start:"
	@echo "  make run-local    # Start locally"
	@echo "  make docker-run   # Start in Docker"
	@echo "  make grpcui      # Open gRPC UI"

# Combined workflows
all: generate ci ## Generate code and run full CI pipeline

dev: clean generate build test ## Development workflow

prod: clean generate ci ## Production build workflow