#!/bin/bash

# Payment Sim API Local Development Script
# Following Traffic Tacos MSA development standards

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print with color
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if required tools are installed
check_dependencies() {
    print_status "Checking dependencies..."

    local missing_deps=()

    if ! command -v go &> /dev/null; then
        missing_deps+=("go")
    fi

    if ! command -v make &> /dev/null; then
        missing_deps+=("make")
    fi

    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_error "Missing dependencies: ${missing_deps[*]}"
        print_error "Please install the missing dependencies and try again."
        exit 1
    fi

    print_success "All dependencies are installed"
}

# Load environment variables
load_env() {
    print_status "Loading environment variables..."

    if [ -f "$PROJECT_ROOT/.env.local" ]; then
        print_status "Loading .env.local"
        export $(grep -v '^#' "$PROJECT_ROOT/.env.local" | xargs)
    elif [ -f "$PROJECT_ROOT/.env" ]; then
        print_status "Loading .env"
        export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
    else
        print_warning "No .env file found, using defaults"
        print_status "Copy .env.template to .env.local and customize as needed"
    fi

    # Set defaults
    export GRPC_PORT=${GRPC_PORT:-8003}
    export REST_PORT=${REST_PORT:-8004}
    export ENVIRONMENT=${ENVIRONMENT:-development}
    export WEBHOOK_SECRET=${WEBHOOK_SECRET:-local-dev-secret}
    export AWS_PROFILE=${AWS_PROFILE:-tacos}
    export AWS_REGION=${AWS_REGION:-ap-northeast-2}
}

# Build the application
build_app() {
    print_status "Building payment-sim-api..."
    cd "$PROJECT_ROOT"

    if make build; then
        print_success "Build completed successfully"
    else
        print_error "Build failed"
        exit 1
    fi
}

# Start the application
start_app() {
    print_status "Starting payment-sim-api..."
    print_status "gRPC server will start on port $GRPC_PORT"
    print_status "REST server will start on port $REST_PORT"
    print_status "Swagger UI: http://localhost:$REST_PORT/swagger"

    if command -v grpcui &> /dev/null; then
        print_status "gRPC UI available at: grpcui -plaintext localhost:$GRPC_PORT"
    else
        print_warning "grpcui not found. Install with: go install github.com/fullstorydev/grpcui/cmd/grpcui@latest"
    fi

    echo
    print_status "Press Ctrl+C to stop the server"
    echo

    cd "$PROJECT_ROOT"
    exec ./bin/payment-sim-api
}

# Show help
show_help() {
    echo "Payment Sim API Local Development Script"
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  setup     - Check dependencies and load environment"
    echo "  build     - Build the application"
    echo "  start     - Setup, build, and start the application"
    echo "  help      - Show this help message"
    echo ""
    echo "Environment files (in order of precedence):"
    echo "  .env.local (recommended for local development)"
    echo "  .env"
    echo "  .env.template (template file)"
    echo ""
    echo "Default ports:"
    echo "  gRPC: 8003"
    echo "  REST: 8004"
}

# Main function
main() {
    local command=${1:-start}

    case $command in
        "setup")
            check_dependencies
            load_env
            print_success "Setup completed"
            ;;
        "build")
            load_env
            build_app
            ;;
        "start")
            check_dependencies
            load_env
            build_app
            start_app
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Handle script interruption
trap 'print_warning "Script interrupted"; exit 130' INT

# Run main function
main "$@"