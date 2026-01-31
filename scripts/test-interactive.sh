#!/bin/bash

# Interactive test runner for Prisma Go Client
# Usage: 
#   Interactive mode: ./scripts/test-interactive.sh
#   With flags: ./scripts/test-interactive.sh --unit | --pg | --mysql | --sqlite
#   Verbose mode: ./scripts/test-interactive.sh --pg --verbose (or -v)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

# Error log file (use absolute path to avoid permission issues)
ERROR_LOG="$(pwd)/test_errors.log"
VERBOSE_MODE=false

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

run_test() {
    local test_name="$1"
    shift
    local test_cmd="$@"
    
    log_info "Running: $test_name"
    
    if [ "$VERBOSE_MODE" = true ]; then
        # Verbose mode: show output directly
        if eval "$test_cmd"; then
            log_success "$test_name"
            return 0
        else
            log_error "$test_name"
            FAILED_TESTS+=("$test_name")
            return 1
        fi
    else
        # Normal mode: capture output and log to file
        local temp_output=$(mktemp)
        if eval "$test_cmd" > "$temp_output" 2>&1; then
            log_success "$test_name"
            rm -f "$temp_output"
            return 0
        else
            log_error "$test_name"
            echo "" >> "$ERROR_LOG"
            echo "=== Error in test: $test_name ===" >> "$ERROR_LOG"
            echo "Command: $test_cmd" >> "$ERROR_LOG"
            echo "---" >> "$ERROR_LOG"
            cat "$temp_output" >> "$ERROR_LOG"
            echo "---" >> "$ERROR_LOG"
            echo "" >> "$ERROR_LOG"
            cat "$temp_output"
            rm -f "$temp_output"
            FAILED_TESTS+=("$test_name")
            return 1
        fi
    fi
}

# Internal function to run unit tests (shared logic)
run_unit_tests_internal() {
    # Check prerequisites
    log_info "Checking prerequisites..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go not found. Please install Go 1.22 or later."
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}')
    log_success "Go found: $GO_VERSION"
    echo ""
    
    # Download dependencies
    log_info "Downloading Go dependencies..."
    run_test "Dependencies downloaded" "go mod download"
    echo ""
    
    # Install development tools
    log_info "Installing development tools..."
    if ! command -v goimports &> /dev/null; then
        log_info "Installing goimports..."
        go install golang.org/x/tools/cmd/goimports@latest
    fi
    if ! command -v golangci-lint &> /dev/null; then
        log_info "Installing golangci-lint..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    fi
    echo ""
    
    # Generate Prisma Client
    log_info "Generating Prisma Client..."
    if go run ./cmd/prisma generate; then
        log_success "Prisma Client generated"
    else
        log_error "Failed to generate Prisma Client"
        exit 1
    fi
    
    # Format generated code
    log_info "Formatting generated code..."
    if [ -d "prisma/db" ]; then
        goimports -w prisma/db/
        log_success "Generated code formatted"
    else
        log_warning "prisma/db directory not found, skipping format"
    fi
    echo ""
    
    # Run linter
    log_info "=== Test: Linter ==="
    if command -v golangci-lint &> /dev/null; then
        run_test "Linter (golangci-lint)" "golangci-lint run --timeout=5m ./..."
    else
        log_warning "golangci-lint not found, skipping linter test"
    fi
    echo ""
    
    # Run general tests (without database tags)
    log_info "=== Test: General Unit Tests ==="
    run_test "General unit tests (go test)" "go test -v ./..."
    echo ""
    
    # Build CLI
    log_info "=== Test: Build CLI ==="
    run_test "Build CLI" "mkdir -p bin && go build -o bin/prisma ./cmd/prisma"
    if [ -f "bin/prisma" ]; then
        log_info "Testing compiled binary..."
        if ./bin/prisma --help > /dev/null 2>&1; then
            log_success "Binary works correctly"
        else
            log_warning "Binary compiled but may have issues"
        fi
    fi
    echo ""
    
    # Format check
    log_info "=== Test: Format Check ==="
    if command -v goimports &> /dev/null; then
        temp_output=$(mktemp)
        UNFORMATTED=$(goimports -l . > "$temp_output" 2>&1; cat "$temp_output" | wc -l)
        if [ "$UNFORMATTED" -gt 0 ]; then
            log_error "Code is not formatted. Run 'goimports -w .'"
            cat "$temp_output"
            rm -f "$temp_output"
            FAILED_TESTS+=("Format Check")
        else
            log_success "Format OK"
            rm -f "$temp_output"
        fi
    else
        log_warning "goimports not found, skipping format check"
    fi
    echo ""
}

# Function to run unit tests (no database required)
run_unit_tests() {
    # Initialize error log (only if not in verbose mode)
    if [ "$VERBOSE_MODE" = false ]; then
        echo "=== Errors captured during test execution ===" > "$ERROR_LOG"
        echo "Date: $(date)" >> "$ERROR_LOG"
        echo "" >> "$ERROR_LOG"
    fi
    
    echo ""
    echo "=========================================="
    echo "  Running Unit Tests (No Database)"
    echo "=========================================="
    echo ""
    
    run_unit_tests_internal
    
    # Summary
    print_summary
}

# Function to run PostgreSQL tests
run_postgres_tests() {
    # Initialize error log (only if not in verbose mode)
    if [ "$VERBOSE_MODE" = false ]; then
        echo "=== Errors captured during test execution ===" > "$ERROR_LOG"
        echo "Date: $(date)" >> "$ERROR_LOG"
        echo "" >> "$ERROR_LOG"
    fi
    
    echo ""
    echo "=========================================="
    echo "  Running PostgreSQL Tests"
    echo "=========================================="
    echo ""
    
    # Run unit tests first to ensure prisma/db is generated
    log_info "Running unit tests first..."
    run_unit_tests_internal
    echo ""
    
    # Check prerequisites
    log_info "Checking prerequisites..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}')
    log_success "Go found: $GO_VERSION"
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    log_success "Docker found"
    
    echo ""
    
    # Start PostgreSQL container
    log_info "Starting PostgreSQL container..."
    if command -v docker-compose &> /dev/null; then
        docker-compose -f docker-compose.test.yml up -d postgres_test
    else
        docker compose -f docker-compose.test.yml up -d postgres_test
    fi
    log_success "PostgreSQL container started"
    
    # Wait for PostgreSQL to be ready
    log_info "Waiting for PostgreSQL to be ready..."
    POSTGRES_CONTAINER=$(docker ps -q -f name=postgres_test)
    if [ -z "$POSTGRES_CONTAINER" ]; then
        POSTGRES_CONTAINER=$(docker ps -q -f ancestor=postgres:15)
    fi
    
    for i in {1..30}; do
        if docker exec "$POSTGRES_CONTAINER" pg_isready -U postgres 2>/dev/null | grep -q "accepting connections"; then
            log_success "PostgreSQL ready"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "PostgreSQL did not become ready in time"
            exit 1
        fi
        sleep 2
    done
    
    # Export database URL
    export TEST_DATABASE_URL_POSTGRESQL="postgresql://postgres:postgres@localhost:5433/postgres?sslmode=disable"
    log_success "Database URL exported"
    echo ""
    
    # Download dependencies
    log_info "Downloading Go dependencies..."
    if go mod download; then
        log_success "Dependencies downloaded"
    else
        log_error "Failed to download dependencies"
        exit 1
    fi
    echo ""
    
    # Generate Prisma Client
    log_info "Generating Prisma Client..."
    if go run cmd/prisma/main.go generate; then
        log_success "Prisma Client generated"
    else
        log_error "Failed to generate Prisma Client"
        exit 1
    fi
    echo ""
    
    # Run PostgreSQL tests
    log_info "=== Running PostgreSQL Tests ==="
    run_test "PostgreSQL tests (tests/query-builder)" "go test -tags=postgresql -v ./tests/query-builder/..."
    echo ""
    
    # Stop container
    log_info "Stopping PostgreSQL container..."
    if docker-compose -f docker-compose.test.yml down 2>/dev/null || docker compose -f docker-compose.test.yml down 2>/dev/null; then
        log_success "Container stopped"
    else
        log_warning "Failed to stop container"
    fi
    echo ""
    
    # Summary
    print_summary
}

# Function to run MySQL tests
run_mysql_tests() {
    # Initialize error log (only if not in verbose mode)
    if [ "$VERBOSE_MODE" = false ]; then
        echo "=== Errors captured during test execution ===" > "$ERROR_LOG"
        echo "Date: $(date)" >> "$ERROR_LOG"
        echo "" >> "$ERROR_LOG"
    fi
    
    echo ""
    echo "=========================================="
    echo "  Running MySQL Tests"
    echo "=========================================="
    echo ""
    
    # Run unit tests first to ensure prisma/db is generated
    log_info "Running unit tests first..."
    run_unit_tests_internal
    echo ""
    
    # Check prerequisites
    log_info "Checking prerequisites..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}')
    log_success "Go found: $GO_VERSION"
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    log_success "Docker found"
    
    echo ""
    
    # Start MySQL container
    log_info "Starting MySQL container..."
    if command -v docker-compose &> /dev/null; then
        docker-compose -f docker-compose.test.yml up -d mysql_test
    else
        docker compose -f docker-compose.test.yml up -d mysql_test
    fi
    log_success "MySQL container started"
    
    # Wait for MySQL to be ready
    log_info "Waiting for MySQL to be ready..."
    MYSQL_CONTAINER=$(docker ps -q -f name=mysql_test)
    if [ -z "$MYSQL_CONTAINER" ]; then
        MYSQL_CONTAINER=$(docker ps -q -f ancestor=mysql:8)
    fi
    
    for i in {1..30}; do
        if docker exec "$MYSQL_CONTAINER" mysqladmin ping -h localhost -u root -ppassword 2>/dev/null; then
            log_success "MySQL ready"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "MySQL did not become ready in time"
            exit 1
        fi
        sleep 2
    done
    
    # Export database URL
    export TEST_DATABASE_URL_MYSQL="mysql://root:password@localhost:3307/prisma_test"
    log_success "Database URL exported"
    echo ""
    
    # Download dependencies
    log_info "Downloading Go dependencies..."
    if go mod download; then
        log_success "Dependencies downloaded"
    else
        log_error "Failed to download dependencies"
        exit 1
    fi
    echo ""
    
    # Generate Prisma Client
    log_info "Generating Prisma Client..."
    if go run cmd/prisma/main.go generate; then
        log_success "Prisma Client generated"
    else
        log_error "Failed to generate Prisma Client"
        exit 1
    fi
    echo ""
    
    # Run MySQL tests
    log_info "=== Running MySQL Tests ==="
    run_test "MySQL tests" "go test -tags=mysql -v ./..."
    echo ""
    
    # Stop container
    log_info "Stopping MySQL container..."
    if docker-compose -f docker-compose.test.yml down 2>/dev/null || docker compose -f docker-compose.test.yml down 2>/dev/null; then
        log_success "Container stopped"
    else
        log_warning "Failed to stop container"
    fi
    echo ""
    
    # Summary
    print_summary
}

# Function to run SQLite tests
run_sqlite_tests() {
    # Initialize error log (only if not in verbose mode)
    if [ "$VERBOSE_MODE" = false ]; then
        echo "=== Errors captured during test execution ===" > "$ERROR_LOG"
        echo "Date: $(date)" >> "$ERROR_LOG"
        echo "" >> "$ERROR_LOG"
    fi
    
    echo ""
    echo "=========================================="
    echo "  Running SQLite Tests"
    echo "=========================================="
    echo ""
    
    # Run unit tests first to ensure prisma/db is generated
    log_info "Running unit tests first..."
    run_unit_tests_internal
    echo ""
    
    # Check prerequisites
    log_info "Checking prerequisites..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}')
    log_success "Go found: $GO_VERSION"
    echo ""
    
    # Export database URL
    export TEST_DATABASE_URL_SQLITE="file:./test.db"
    log_success "Database URL exported"
    echo ""
    
    # Download dependencies
    log_info "Downloading Go dependencies..."
    if go mod download; then
        log_success "Dependencies downloaded"
    else
        log_error "Failed to download dependencies"
        exit 1
    fi
    echo ""
    
    # Generate Prisma Client
    log_info "Generating Prisma Client..."
    if go run cmd/prisma/main.go generate; then
        log_success "Prisma Client generated"
    else
        log_error "Failed to generate Prisma Client"
        exit 1
    fi
    echo ""
    
    # Run SQLite tests
    log_info "=== Running SQLite Tests ==="
    run_test "SQLite tests" "go test -tags=sqlite -v ./..."
    
    # Cleanup SQLite database
    if [ -f "test.db" ]; then
        rm -f test.db
        log_success "SQLite database cleaned up"
    fi
    echo ""
    
    # Summary
    print_summary
}


# Function to print test summary
print_summary() {
    echo "=========================================="
    echo "  Test Summary"
    echo "=========================================="
    echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
    echo ""
    
    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "${RED}Failed tests:${NC}"
        for test in "${FAILED_TESTS[@]}"; do
            echo -e "  ${RED}✗${NC} $test"
        done
        echo ""
        if [ "$VERBOSE_MODE" = false ]; then
            echo -e "${YELLOW}Error log saved to: $ERROR_LOG${NC}"
        fi
        exit 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        # Clean up error log on success (only if not in verbose mode)
        if [ "$VERBOSE_MODE" = false ] && [ -f "$ERROR_LOG" ]; then
            rm -f "$ERROR_LOG"
            log_info "Error log file removed (all tests passed)"
        fi
        echo ""
        exit 0
    fi
}

# Initialize control variables
VERBOSE_MODE=false
RAN_COMMAND=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE_MODE=true
            shift
            ;;
        --unit)
            RAN_COMMAND=true
            run_unit_tests
            print_summary
            shift
            ;;
        --pg|--postgres)
            RAN_COMMAND=true
            run_postgres_tests
            print_summary
            shift
            ;;
        --mysql)
            RAN_COMMAND=true
            run_mysql_tests
            print_summary
            shift
            ;;
        --sqlite)
            RAN_COMMAND=true
            run_sqlite_tests
            print_summary
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS] [--unit|--pg|--mysql|--sqlite]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose   Show test output in console (no error log file)"
            echo "  --unit          Run unit tests (no database)"
            echo "  --pg, --postgres Run PostgreSQL integration tests"
            echo "  --mysql         Run MySQL integration tests"
            echo "  --sqlite        Run SQLite integration tests"
            echo "  -h, --help      Show this help message"
            echo ""
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Usage: $0 [OPTIONS] [--unit|--pg|--mysql|--sqlite]"
            exit 1
            ;;
    esac
done

# If no command was run, start interactive mode
if [ "$RAN_COMMAND" = false ]; then
    # Interactive mode
    clear
    echo "=========================================="
    echo "  Prisma Go Client - Interactive Tests"
    echo "=========================================="
    echo ""
    
    # Menu options
    echo "Select which tests to run:"
    echo ""
    
    PS3="Choose an option (number): "
    options=("Units" "Postgres" "MySQL" "SQLite" "Exit")
    
    select opt in "${options[@]}"
    do
        case $REPLY in
            1)
                run_unit_tests
                ;;
            2)
                run_postgres_tests
                ;;
            3)
                run_mysql_tests
                ;;
            4)
                run_sqlite_tests
                ;;
            5)
                echo ""
                echo -e "${GREEN}Exiting...${NC}"
                echo ""
                exit 0
                ;;
            *)
                echo ""
                echo -e "${RED}Invalid option! Please choose a number from the menu.${NC}"
                echo ""
                ;;
        esac
    done
fi
