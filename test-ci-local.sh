#!/bin/bash

# Script to test all CI flows locally
# Usage: ./test-ci-local.sh [--skip-docker] [--skip-linter] [--skip-format]

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Flags
SKIP_DOCKER=false
SKIP_LINTER=false
SKIP_FORMAT=false

# Parse arguments
for arg in "$@"; do
    case $arg in
        --skip-docker)
            SKIP_DOCKER=true
            shift
            ;;
        --skip-linter)
            SKIP_LINTER=true
            shift
            ;;
        --skip-format)
            SKIP_FORMAT=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $arg${NC}"
            echo "Usage: $0 [--skip-docker] [--skip-linter] [--skip-format]"
            exit 1
            ;;
    esac
done

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

ERRORS_FILE="ci_errors.txt"
echo "=== Errors captured during CI execution ===" > "$ERRORS_FILE"
echo "Date: $(date)" >> "$ERRORS_FILE"
echo "" >> "$ERRORS_FILE"

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
    echo "[ERROR] $1" >> "$ERRORS_FILE"
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
    local temp_output=$(mktemp)
    if eval "$test_cmd" > "$temp_output" 2>&1; then
        log_success "$test_name"
        rm -f "$temp_output"
        return 0
    else
        log_error "$test_name"
        echo "" >> "$ERRORS_FILE"
        echo "=== Error in test: $test_name ===" >> "$ERRORS_FILE"
        echo "Command: $test_cmd" >> "$ERRORS_FILE"
        echo "---" >> "$ERRORS_FILE"
        cat "$temp_output" >> "$ERRORS_FILE"
        echo "---" >> "$ERRORS_FILE"
        echo "" >> "$ERRORS_FILE"
        rm -f "$temp_output"
        FAILED_TESTS+=("$test_name")
        return 1
    fi
}

# Header
echo "=========================================="
echo "  Prisma Go Client - CI Local Tests"
echo "=========================================="
echo ""

# 1. Check prerequisites
log_info "Checking prerequisites..."

if ! command -v go &> /dev/null; then
    log_error "Go is not installed"
    exit 1
fi
GO_VERSION=$(go version | awk '{print $3}')
log_success "Go found: $GO_VERSION"

if ! command -v docker &> /dev/null; then
    log_warning "Docker not found. Database tests will be skipped."
    SKIP_DOCKER=true
else
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
    log_success "Docker found: $DOCKER_VERSION"
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    log_warning "Docker Compose not found. Database tests will be skipped."
    SKIP_DOCKER=true
else
    log_success "Docker Compose found"
fi

echo ""

# 2. Start Docker containers (before any tests)
if [ "$SKIP_DOCKER" = false ]; then
    log_info "=== Starting Docker Containers ==="
    log_info "Starting Docker containers..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose -f docker-compose.test.yml up -d
    else
        docker compose -f docker-compose.test.yml up -d
    fi
    
    log_success "Containers started"
    
    # Wait for databases to be ready
    log_info "Waiting for databases to be ready..."
    
    # Check PostgreSQL
    log_info "Checking PostgreSQL..."
    POSTGRES_CONTAINER=$(docker ps -q -f name=postgres_test)
    if [ -z "$POSTGRES_CONTAINER" ]; then
        POSTGRES_CONTAINER=$(docker ps -q -f ancestor=postgres:15)
    fi
    
    if [ -n "$POSTGRES_CONTAINER" ]; then
        for i in {1..30}; do
            if docker exec "$POSTGRES_CONTAINER" pg_isready -U postgres 2>/dev/null | grep -q "accepting connections"; then
                log_success "PostgreSQL ready"
                break
            fi
            if [ $i -eq 30 ]; then
                log_warning "PostgreSQL did not become ready in time"
            fi
            sleep 2
        done
    else
        log_warning "PostgreSQL container not found"
    fi
    
    # Check MySQL
    log_info "Checking MySQL..."
    MYSQL_CONTAINER=$(docker ps -q -f name=mysql_test)
    if [ -z "$MYSQL_CONTAINER" ]; then
        MYSQL_CONTAINER=$(docker ps -q -f ancestor=mysql:8)
    fi
    
    if [ -n "$MYSQL_CONTAINER" ]; then
        for i in {1..30}; do
            if docker exec "$MYSQL_CONTAINER" mysqladmin ping -h localhost -u root -ppassword 2>/dev/null; then
                log_success "MySQL ready"
                break
            fi
            if [ $i -eq 30 ]; then
                log_warning "MySQL did not become ready in time"
            fi
            sleep 2
        done
    else
        log_warning "MySQL container not found"
    fi
    
    # Export database URLs for all tests
    export TEST_DATABASE_URL="postgresql://postgres:postgres@localhost:5433/postgres?sslmode=disable"
    export TEST_DATABASE_URL_POSTGRESQL="postgresql://postgres:postgres@localhost:5433/postgres?sslmode=disable"
    export TEST_DATABASE_URL_MYSQL="mysql://root:password@localhost:3307/prisma_test"
    export TEST_DATABASE_URL_SQLITE="file:./test.db"
    
    log_success "Database URLs exported"
    echo ""
fi

# 3. Download dependencies
log_info "Downloading Go dependencies..."
if go mod download; then
    log_success "Dependencies downloaded"
else
    log_error "Failed to download dependencies"
    exit 1
fi
echo ""

# 4. Install tools
log_info "Installing development tools..."

if [ "$SKIP_LINTER" = false ]; then
    log_info "Installing golangci-lint..."
    if ! command -v golangci-lint &> /dev/null; then
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest || log_warning "Failed to install golangci-lint"
    fi
    
    log_info "Installing goimports..."
    if ! command -v goimports &> /dev/null; then
        go install golang.org/x/tools/cmd/goimports@latest || log_warning "Failed to install goimports"
    fi
fi

echo ""

# 4.5. Generate Prisma Client
log_info "Generating Prisma Client..."
if go run cmd/prisma/main.go generate; then
    log_success "Prisma Client generated"
    
    # Format generated code
    log_info "Formatting generated code..."
    if command -v goimports &> /dev/null; then
        goimports -w prisma/db/ > /dev/null 2>&1
        log_success "Generated code formatted"
    fi
else
    log_error "Failed to generate Prisma Client"
    exit 1
fi

echo ""

# 4. CI Workflow tests (ci.yml)

## 4.1 Linter
if [ "$SKIP_LINTER" = false ]; then
    log_info "=== Test: Linter ==="
    if command -v golangci-lint &> /dev/null; then
        temp_output=$(mktemp)
        if golangci-lint run ./... > "$temp_output" 2>&1; then
            log_success "Linter (golangci-lint)"
            rm -f "$temp_output"
        else
            log_error "Linter (golangci-lint)"
            echo "" >> "$ERRORS_FILE"
            echo "=== Error in test: Linter (golangci-lint) ===" >> "$ERRORS_FILE"
            echo "---" >> "$ERRORS_FILE"
            cat "$temp_output" >> "$ERRORS_FILE"
            echo "---" >> "$ERRORS_FILE"
            echo "" >> "$ERRORS_FILE"
            rm -f "$temp_output"
            FAILED_TESTS+=("Linter (golangci-lint)")
        fi
    else
        log_warning "golangci-lint not found, skipping linter test"
    fi
    echo ""
fi

## 4.2 General tests
log_info "=== Test: General Tests ==="
run_test "General tests (go test)" "go test -v ./..."
echo ""

## 4.3 Build CLI
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

## 4.4 Format check
if [ "$SKIP_FORMAT" = false ]; then
    log_info "=== Test: Format Check ==="
    if command -v goimports &> /dev/null; then
        temp_output=$(mktemp)
        UNFORMATTED=$(goimports -l . > "$temp_output" 2>&1; cat "$temp_output" | wc -l)
        if [ "$UNFORMATTED" -gt 0 ]; then
            log_error "Code is not formatted. Run 'goimports -w .'"
            echo "" >> "$ERRORS_FILE"
            echo "=== Error in test: Format Check ===" >> "$ERRORS_FILE"
            echo "Unformatted files:" >> "$ERRORS_FILE"
            echo "---" >> "$ERRORS_FILE"
            cat "$temp_output" >> "$ERRORS_FILE"
            echo "---" >> "$ERRORS_FILE"
            echo "" >> "$ERRORS_FILE"
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
fi

# 5. Database tests
if [ "$SKIP_DOCKER" = false ]; then
    log_info "=== Database Tests ==="
    
    # 5.1 PostgreSQL tests (database/sql with pgx driver)
    log_info "=== Test: PostgreSQL (database/sql) ==="
    if go list -m github.com/jackc/pgx/v5/pgxpool &> /dev/null || go get github.com/jackc/pgx/v5/pgxpool 2>/dev/null; then
        run_test "PostgreSQL tests (database/sql)" "go test -tags=pgx -v ./..."
    else
        log_warning "pgx not available, skipping PostgreSQL tests"
    fi
    echo ""
    
    # 5.2 PostgreSQL (pgx) tests  
    log_info "=== Test: PostgreSQL (pgx) ==="
    if go list -m github.com/jackc/pgx/v5/pgxpool &> /dev/null || go get github.com/jackc/pgx/v5/pgxpool 2>/dev/null; then
        run_test "PostgreSQL tests (pgx)" "go test -tags=pgx -v ./..."
    else
        log_warning "pgx not available, skipping pgx tests"
    fi
    echo ""
    
    # 5.3 MySQL tests
    log_info "=== Test: MySQL ==="
    if go list -m github.com/go-sql-driver/mysql &> /dev/null || go get github.com/go-sql-driver/mysql 2>/dev/null; then
        run_test "MySQL tests" "go test -tags=mysql -v ./..."
    else
        log_warning "MySQL driver not available, skipping MySQL tests"
    fi
    echo ""
    
    # 5.4 SQLite tests
    log_info "=== Test: SQLite ==="
    if go list -m github.com/mattn/go-sqlite3 &> /dev/null || go get github.com/mattn/go-sqlite3 2>/dev/null; then
        run_test "SQLite tests" "go test -tags=sqlite -v ./..."
        rm -f test.db
    else
        log_warning "SQLite driver not available, skipping SQLite tests"
    fi
    echo ""
    
    # 5.5 Stop containers
    log_info "Stopping Docker containers..."
    if docker-compose -f docker-compose.test.yml down 2>/dev/null || docker compose -f docker-compose.test.yml down 2>/dev/null; then
        log_success "Containers stopped"
    else
        log_warning "Failed to stop containers"
    fi

    echo ""
else
    log_warning "Database tests skipped (--skip-docker or Docker not available)"
    echo ""
fi

# 6. Final summary
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
    # Check if there is content in the error file (besides the header)
    if [ -f "$ERRORS_FILE" ] && [ $(wc -l < "$ERRORS_FILE") -gt 3 ]; then
        echo -e "${YELLOW}Detailed errors saved in: $ERRORS_FILE${NC}"
        echo -e "${YELLOW}Total error lines: $(($(wc -l < "$ERRORS_FILE") - 3))${NC}"
    else
        # If there are no detailed errors, remove the file
        rm -f "$ERRORS_FILE"
    fi
    echo ""
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    # Remove error file if there are no errors
    if [ -f "$ERRORS_FILE" ]; then
        # Check if there is only the header
        if [ $(wc -l < "$ERRORS_FILE") -le 3 ]; then
            rm -f "$ERRORS_FILE"
        fi
    fi
    echo ""
    exit 0
fi

