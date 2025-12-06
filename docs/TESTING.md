# Testing Guide

This guide explains how to write and run tests for Prisma for Go, including database tests for all supported providers (PostgreSQL, MySQL, SQLite).

## Table of Contents

- [Test Infrastructure](#test-infrastructure)
- [Setting Up Test Databases](#setting-up-test-databases)
- [Writing Database Tests](#writing-database-tests)
- [Running Tests](#running-tests)
- [CI/CD Testing](#cicd-testing)
- [Best Practices](#best-practices)

## Test Infrastructure

The `internal/testing` package provides comprehensive helpers for database testing:

### Available Helpers

- **`SetupTestDB(t *testing.T, provider string) (driver.DB, func())`** - Creates a test database and returns connection + cleanup function
- **`SetupPostgreSQLTestDB(t *testing.T) (driver.DB, func())`** - PostgreSQL-specific setup
- **`SetupMySQLTestDB(t *testing.T) (driver.DB, func())`** - MySQL-specific setup
- **`SetupSQLiteTestDB(t *testing.T) (driver.DB, func())`** - SQLite-specific setup (uses temp file)
- **`ApplyTestMigrations(t *testing.T, db driver.DB, migrationsPath string, provider string)`** - Apply all migrations
- **`RollbackTestMigrations(t *testing.T, db driver.DB, provider string)`** - Rollback all migrations
- **`ResetTestDatabase(t *testing.T, db driver.DB, migrationsPath string, provider string)`** - Drop all tables and reapply migrations
- **`CleanTestData(t *testing.T, db driver.DB, provider string)`** - Clean all test data (TRUNCATE or DELETE)
- **`WithTestTransaction(t *testing.T, db driver.DB, fn func(tx driver.Tx))`** - Execute test in transaction with rollback
- **`SkipIfNoDatabase(t *testing.T, provider string)`** - Skip test if database not available
- **`RequireDatabase(t *testing.T, provider string)`** - Fail test if database not available

## Setting Up Test Databases

### Option 1: Docker Compose (Recommended)

Use the provided `docker-compose.test.yml`:

```bash
docker-compose -f docker-compose.test.yml up -d
```

This starts:

- PostgreSQL on port `5433`
- MySQL on port `3307`

### Option 2: Local Installation

Install PostgreSQL and MySQL locally, then configure test URLs.

### Environment Variables

Copy `.env.test.example` to `.env.test` and configure:

```bash
cp .env.test.example .env.test
```

Required environment variables:

```env
# PostgreSQL
TEST_DATABASE_URL_POSTGRESQL=postgresql://postgres:postgres@localhost:5432/prisma_test?sslmode=disable

# MySQL
TEST_DATABASE_URL_MYSQL=mysql://root:password@localhost:3306/prisma_test

# SQLite (file path)
TEST_DATABASE_URL_SQLITE=file:./test.db

# Fallback (used if provider-specific URL is not set)
TEST_DATABASE_URL=postgresql://postgres:postgres@localhost:5432/prisma_test?sslmode=disable
```

**Note:** SQLite tests don't require external setup - they use temporary files automatically.

## Writing Database Tests

### Basic Test Example

```go
package mypackage

import (
    "context"
    "testing"

    "github.com/carlosnayan/prisma-go-client/builder"
    "github.com/carlosnayan/prisma-go-client/internal/dialect"
    testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

func TestMyFeature(t *testing.T) {
    // Setup test database
    db, cleanup := testutil.SetupTestDB(t, "postgresql")
    defer cleanup()

    // Apply migrations
    testutil.ApplyTestMigrations(t, db, "prisma/migrations", "postgresql")

    // Your test code here
    builder := builder.NewTableQueryBuilder(db, "users", []string{"id", "email"})
    builder.SetDialect(dialect.GetDialect("postgresql"))

    ctx := context.Background()
    // ... test operations
}
```

### Testing with Multiple Providers

```go
func TestFeature_AllProviders(t *testing.T) {
    providers := []string{"postgresql", "mysql", "sqlite"}

    for _, provider := range providers {
        t.Run(provider, func(t *testing.T) {
            testutil.SkipIfNoDatabase(t, provider)
            db, cleanup := testutil.SetupTestDB(t, provider)
            defer cleanup()

            // Test implementation
        })
    }
}
```

### Using Transactions for Isolation

```go
func TestWithTransaction(t *testing.T) {
    db, cleanup := testutil.SetupTestDB(t, "postgresql")
    defer cleanup()

    testutil.WithTestTransaction(t, db, func(tx driver.Tx) {
        // All operations in this function will be rolled back
        // This ensures test isolation
        ctx := context.Background()
        // ... test operations
    })
}
```

### Provider-Specific Tests

```go
// PostgreSQL-specific test
func TestPostgreSQLFeature(t *testing.T) {
    testutil.SkipIfNoDatabase(t, "postgresql")
    db, cleanup := testutil.SetupPostgreSQLTestDB(t)
    defer cleanup()

    // PostgreSQL-specific test code
}

// MySQL-specific test
func TestMySQLFeature(t *testing.T) {
    testutil.SkipIfNoDatabase(t, "mysql")
    db, cleanup := testutil.SetupMySQLTestDB(t)
    defer cleanup()

    // MySQL-specific test code
}

// SQLite-specific test
func TestSQLiteFeature(t *testing.T) {
    db, cleanup := testutil.SetupSQLiteTestDB(t)
    defer cleanup()

    // SQLite-specific test code
}
```

### Cleaning Test Data

```go
func TestWithCleanData(t *testing.T) {
    db, cleanup := testutil.SetupTestDB(t, "postgresql")
    defer cleanup()

    testutil.ApplyTestMigrations(t, db, "prisma/migrations", "postgresql")

    // Insert test data
    // ... insert operations

    // Clean all test data
    testutil.CleanTestData(t, db, "postgresql")

    // Verify data is cleaned
    // ... assertions
}
```

## Running Tests

### Run All Tests

```bash
go get github.com/jackc/pgx/v5/stdlib
go get github.com/go-sql-driver/mysql
go get github.com/mattn/go-sqlite3

# Run all tests
go test -tags="pgx,mysql,sqlite" ./...
```

### Run Tests for Specific Package

```bash
go test ./builder/...
go test ./internal/driver/...
```

### Run Tests for Specific Provider

```bash
# PostgreSQL
TEST_DATABASE_URL_POSTGRESQL="postgresql://..." go test ./...

# MySQL
TEST_DATABASE_URL_MYSQL="mysql://..." go test ./...

# SQLite
TEST_DATABASE_URL_SQLITE="file:./test.db" go test ./...
```

### Run Tests with pgx Build Tag

```bash
go test -tags=pgx ./...
```

### Run Specific Test

```bash
go test -run TestMyFeature ./...
```

### Verbose Output

```bash
go test -v ./...
```

### Skip Database Tests

If test database URLs are not set, database tests will be automatically skipped. To explicitly skip:

```go
func TestMyFeature(t *testing.T) {
    testutil.SkipIfNoDatabase(t, "postgresql")
    // Test will be skipped if database not available
}
```

## CI/CD Testing

### GitHub Actions Workflows

The project includes three GitHub Actions workflows that run sequentially on push to `main`:

1. **`test-postgresql.yml`** - Tests PostgreSQL/pgx driver
2. **`test-mysql.yml`** - Tests MySQL driver (runs after PostgreSQL)
3. **`test-sqlite.yml`** - Tests SQLite driver (runs after MySQL)

All workflows:

- Use `ubuntu-latest` (Ubuntu slim)
- Install the appropriate database driver
- Set up database services (PostgreSQL/MySQL) or use temp files (SQLite)
- Run all tests with provider-specific environment variables

### Local CI Simulation

To simulate CI locally:

```bash
# PostgreSQL
TEST_DATABASE_URL_POSTGRESQL="postgresql://postgres:postgres@localhost:5432/prisma_test?sslmode=disable" \
go test -tags=pgx -v ./...

# MySQL
TEST_DATABASE_URL_MYSQL="mysql://root:password@localhost:3306/prisma_test" \
go test -v ./...

# SQLite
TEST_DATABASE_URL_SQLITE="file:./test.db" \
go test -v ./...
```

## Best Practices

### 1. Always Clean Up

Always use `defer cleanup()` to ensure test databases are properly cleaned up:

```go
db, cleanup := testutil.SetupTestDB(t, "postgresql")
defer cleanup() // Always defer cleanup
```

### 2. Use Transactions for Isolation

Use `WithTestTransaction` for tests that modify data:

```go
testutil.WithTestTransaction(t, db, func(tx driver.Tx) {
    // Test operations - automatically rolled back
})
```

### 3. Skip Tests When Database Unavailable

Use `SkipIfNoDatabase` to gracefully skip tests when databases aren't available:

```go
testutil.SkipIfNoDatabase(t, "postgresql")
```

### 4. Test Across All Providers

When possible, test features across all providers:

```go
providers := []string{"postgresql", "mysql", "sqlite"}
for _, provider := range providers {
    t.Run(provider, func(t *testing.T) {
        // Test implementation
    })
}
```

### 5. Use Descriptive Test Names

Use clear, descriptive test names that indicate what's being tested:

```go
// Good
func TestBuilder_Create_WithValidData_ReturnsCreatedRecord(t *testing.T) {}

// Bad
func Test1(t *testing.T) {}
```

### 6. Keep Tests Independent

Each test should be independent and not rely on state from other tests:

```go
// Good - each test sets up its own data
func TestCreate(t *testing.T) {
    db, cleanup := testutil.SetupTestDB(t, "postgresql")
    defer cleanup()
    // Test implementation
}

// Bad - relies on previous test
func TestUpdate(t *testing.T) {
    // Assumes data from TestCreate exists
}
```

### 7. Use Table-Driven Tests

For testing multiple scenarios:

```go
func TestMapType(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"String", "TEXT"},
        {"Int", "INTEGER"},
        {"Boolean", "BOOLEAN"},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 8. Test Error Cases

Don't just test happy paths - also test error cases:

```go
func TestCreate_WithInvalidData_ReturnsError(t *testing.T) {
    // Test error handling
}
```

## Troubleshooting

### Tests Skip Automatically

If tests are skipping, check:

1. Environment variables are set correctly
2. Database services are running (for PostgreSQL/MySQL)
3. Connection strings are valid

### Connection Errors

If you see connection errors:

1. Verify database is running: `docker ps` or check service status
2. Check connection string format
3. Verify network connectivity (for remote databases)
4. Check firewall settings

### Migration Errors

If migrations fail:

1. Verify migration SQL is valid for the provider
2. Check for provider-specific syntax differences
3. Ensure migrations are in correct order

### SQLite Tests Fail

SQLite tests use temporary files and should work without setup. If they fail:

1. Check file permissions in temp directory
2. Verify SQLite driver is installed: `go get github.com/mattn/go-sqlite3`

## Examples

See the following test files for examples:

- `builder/db_test.go` - Query builder tests
- `internal/driver/driver_test.go` - Driver adapter tests
- `internal/migrations/migrations_test.go` - Migration tests
- `internal/dialect/dialect_test.go` - Dialect-specific tests

## Additional Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Database Testing Best Practices](https://go.dev/doc/database/testing)
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines including testing
