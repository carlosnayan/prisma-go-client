# Running Tests

This document describes how to run tests in the Prisma Go Client project.

## Quick Start

### Run All Tests (Local)

```bash
./test-ci-local.sh
```

This will:

- Run linter
- Run all unit tests
- Build the CLI binary
- Check code formatting
- Start Docker containers for PostgreSQL and MySQL
- Run integration tests against all databases

### Run Tests Without Database

```bash
go test ./...
```

Most builder tests will be **skipped** if database environment variables are not set.

---

## Database Tests

### Prerequisites

To run database integration tests, you need:

1. **Docker** and **Docker Compose** installed
2. **Database containers** running
3. **Environment variables** configured

### Option 1: Use Docker Compose (Recommended)

Start the test databases:

```bash
docker-compose -f docker-compose.test.yml up -d
```

### Option 2: Use Local Databases

If you have local PostgreSQL, MySQL, or SQLite:

1. Create a `.env.test.local` file (copy from `.env.test`)
2. Update the database URLs to point to your local instances

```bash
cp .env.test .env.test.local
# Edit .env.test.local with your database URLs
```

### Environment Variables

Tests look for these environment variables:

```bash
# PostgreSQL
TEST_DATABASE_URL_postgresql=postgresql://postgres:postgres@localhost:5432/prisma_test?sslmode=disable
TEST_DATABASE_URL_POSTGRESQL=postgresql://postgres:postgres@localhost:5432/prisma_test?sslmode=disable

# MySQL
TEST_DATABASE_URL_mysql=mysql://root:root@localhost:3306/prisma_test
TEST_DATABASE_URL_MYSQL=mysql://root:root@localhost:3306/prisma_test

# SQLite
TEST_DATABASE_URL_sqlite=file:./test.db
TEST_DATABASE_URL_SQLITE=file:./test.db

# Generic fallback (some older tests)
TEST_DATABASE_URL=postgresql://postgres:postgres@localhost:5432/prisma_test?sslmode=disable
```

**Load environment variables:**

```bash
# Option 1: Export manually
export TEST_DATABASE_URL_postgresql="postgresql://postgres:postgres@localhost:5433/postgres?sslmode=disable"
export TEST_DATABASE_URL_mysql="mysql://root:password@localhost:3307/prisma_test"
export TEST_DATABASE_URL_sqlite="file:./test.db"

# Option 2: Use direnv (if installed)
echo 'dotenv .env.test.local' > .envrc
direnv allow

# Option 3: Source the file (bash/zsh)
set -a; source .env.test.local; set +a
```

---

## Running Specific Tests

### Run Tests for a Package

```bash
go test ./builder
go test ./cmd/prisma/cmd
go test ./internal/generator
```

### Run Specific Test Function

```bash
go test ./builder -run TestCreateMany
go test ./builder -run TestOrderBy_SingleField
```

### Run Tests with Verbose Output

```bash
go test -v ./builder
```

### Run Tests for Specific Provider

```bash
# Only PostgreSQL
export TEST_DATABASE_URL_postgresql="postgresql://..."
go test ./builder -run TestOrderBy

# Only MySQL
export TEST_DATABASE_URL_mysql="mysql://..."
go test ./builder -run TestOrderBy
```

---

## Test Categories

### 1. Unit Tests (No Database Required)

These tests run without any database:

```bash
go test ./cli
go test ./internal/dialect
go test ./internal/parser
go test ./internal/migrations
```

### 2. Builder Tests (Database Required)

These tests require database connections and will be **skipped** without env vars:

```bash
# ~90 tests total
go test ./builder -v
```

Tests covered:

- Query building (`TestQuery_*`)
- CRUD operations (`TestCreate_*`, `TestUpdate_*`)
- Ordering (`TestOrderBy_*`)
- Pagination (`TestQuery_Skip`, `TestQuery_Take`)
- Transactions (`TestTransaction_*`)
- Batch operations (`TestCreateMany_*`, `TestUpdateMany_*`)

### 3. CLI Tests (Database Optional)

```bash
go test ./cmd/prisma/cmd -v
```

Most CLI tests don't need a database, but migration tests do:

- `TestMigrate*` - needs database
- `TestDbPush*` - needs database
- `TestDbPull*` - needs database
- `TestGenerate*` - no database needed
- `TestFormat*` - no database needed

### 4. Generator Tests (No Database Required)

```bash
go test ./internal/generator -v
```

These test code generation without database.

---

## Skipped Tests

### Why Tests Are Skipped

Tests are automatically skipped when:

1. **Missing environment variables**: `TEST_DATABASE_URL_{provider}` not set
2. **Missing drivers**: PostgreSQL driver not available without build tags
3. **Safety**: Some destructive tests (like `TestMigrateReset`) are disabled by default

### Check How Many Tests Are Skipped

```bash
go test ./... -v 2>&1 | grep -c "SKIP:"
```

### Enable All Tests

1. Start databases:

   ```bash
   docker-compose -f docker-compose.test.yml up -d
   ```

2. Export environment variables:

   ```bash
   export TEST_DATABASE_URL_postgresql="postgresql://postgres:postgres@localhost:5433/postgres?sslmode=disable"
   export TEST_DATABASE_URL_mysql="mysql://root:password@localhost:3307/prisma_test"
   export TEST_DATABASE_URL_sqlite="file:./test.db"
   ```

3. Run tests:
   ```bash
   go test ./...
   ```

---

## CI Tests

The CI runs all tests automatically via `test-ci-local.sh`.

### Run What CI Runs

```bash
./test-ci-local.sh
```

### Skip Docker Tests

```bash
./test-ci-local.sh --skip-docker
```

### Skip Linter

```bash
./test-ci-local.sh --skip-linter
```

### Skip Format Check

```bash
./test-ci-local.sh --skip-format
```

---

## Troubleshooting

### "PostgreSQL driver not available"

Some tests require the `pgx` build tag:

```bash
go test -tags=pgx ./...
```

### "TEST_DATABASE_URL not set, skipping"

Export the required environment variable:

```bash
export TEST_DATABASE_URL_postgresql="postgresql://..."
```

### Database Connection Refused

Make sure databases are running:

```bash
docker ps  # Check containers are up
docker-compose -f docker-compose.test.yml logs  # Check logs
```

PostgreSQL should be on port `5433` (not 5432)  
MySQL should be on port `3307` (not 3306)

### Tests Hang or Timeout

Increase test timeout:

```bash
go test -timeout 60s ./builder
```

---

## Best Practices

1. **Always run builder tests with database** - they're integration tests
2. **Use `test-ci-local.sh` before pushing** - catches issues early
3. **Clean up test databases** - `docker-compose down -v` removes volumes
4. **Run specific tests during development** - faster feedback loop
5. **Check skipped tests count** - should be 0 in CI

---

## See Also

- [TESTING.md](./TESTING.md) - Detailed testing guide (if exists)
- [docker-compose.test.yml](./docker-compose.test.yml) - Database setup
- [.env.test](./.env.test) - Environment variables template
