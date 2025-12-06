# Contributing to Prisma for Go

Thank you for your interest in contributing to Prisma for Go! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/prisma.git
   cd prisma
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/carlosnayan/prisma-go-client.git
   ```
4. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

1. **Install dependencies**:

   ```bash
   go mod download
   ```

2. **Build the CLI**:

   ```bash
   cd cmd/prisma
   go build -o prisma .
   ```

3. **Run tests**:

   ```bash
   go test ./...
   ```

4. **Run linter**:
   ```bash
   golangci-lint run ./...
   ```

## Making Changes

### Code Style

- Follow Go conventions and best practices
- Use `gofmt` or `goimports` to format code
- Write clear, self-documenting code
- Add comments for exported functions and types
- Keep functions focused and small

### Commit Messages

- Use clear, descriptive commit messages
- Start with a verb in imperative mood (e.g., "Add", "Fix", "Update")
- Reference issue numbers when applicable (e.g., "Fix #123: ...")
- Keep the first line under 72 characters
- Add a detailed description if needed

Example:

```
Add support for MySQL JSON operations

- Implement GetJSONContainsQuery for MySQL dialect
- Add tests for JSON operations
- Update documentation

Fixes #456
```

### Testing

- Write tests for new features
- Ensure all tests pass before submitting
- Add integration tests for database operations when applicable
- Test with multiple database providers when possible

## Running Database Tests

### Setting Up Test Databases

Database tests require test database instances. You can set up test databases in several ways:

#### Option 1: Using Docker Compose (Recommended)

Create a `docker-compose.test.yml` file:

```yaml
version: "3.8"
services:
  postgres_test:
    image: postgres:15
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: postgres
    ports:
      - "5433:5432"

  mysql_test:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: prisma_test
    ports:
      - "3307:3306"
```

Start the databases:

```bash
docker-compose -f docker-compose.test.yml up -d
```

#### Option 2: Local Installation

Install PostgreSQL and MySQL locally, then configure test URLs in `.env.test` (copy from `.env.test.example`).

### Environment Variables

Copy `.env.test.example` to `.env.test` and configure:

```bash
cp .env.test.example .env.test
# Edit .env.test with your test database credentials
```

Required environment variables:

- `TEST_DATABASE_URL_POSTGRESQL` - PostgreSQL test database URL
- `TEST_DATABASE_URL_MYSQL` - MySQL test database URL
- `TEST_DATABASE_URL_SQLITE` - SQLite test database file path
- `TEST_DATABASE_URL` - Fallback URL if provider-specific not set

### Running Tests

#### Run all tests:

```bash
go test ./...
```

#### Run tests for specific provider:

```bash
# PostgreSQL
TEST_DATABASE_URL_POSTGRESQL="postgresql://..." go test ./...

# MySQL
TEST_DATABASE_URL_MYSQL="mysql://..." go test ./...

# SQLite
TEST_DATABASE_URL_SQLITE="file:./test.db" go test ./...
```

#### Run tests with pgx build tag:

```bash
go test -tags=pgx ./...
```

#### Skip database tests:

If test database URLs are not set, database tests will be automatically skipped.

### Test Helpers

The `internal/testing` package provides helpers for database tests:

- `testing.SetupTestDB(t, provider)` - Creates test database and returns connection + cleanup
- `testing.ApplyTestMigrations(t, db, path, provider)` - Applies migrations
- `testing.CleanTestData(t, db, provider)` - Cleans test data
- `testing.WithTestTransaction(t, db, fn)` - Executes test in transaction with rollback
- `testing.SkipIfNoDatabase(t, provider)` - Skips test if database not available

Example:

```go
func TestMyFeature(t *testing.T) {
    db, cleanup := testing.SetupTestDB(t, "postgresql")
    defer cleanup()

    testing.ApplyTestMigrations(t, db, "prisma/migrations", "postgresql")

    // Your test code here
}
```

### Documentation

- Update relevant documentation when adding features
- Add examples for new functionality
- Update CHANGELOG.md for user-facing changes

## Submitting Changes

1. **Ensure your code is formatted**:

   ```bash
   goimports -w .
   ```

2. **Run tests and linter**:

   ```bash
   go test ./...
   golangci-lint run ./...
   ```

3. **Commit your changes**:

   ```bash
   git add .
   git commit -m "Your commit message"
   ```

4. **Push to your fork**:

   ```bash
   git push origin feature/your-feature-name
   ```

5. **Create a Pull Request** on GitHub:
   - Provide a clear title and description
   - Reference any related issues
   - Include screenshots or examples if applicable

## Pull Request Guidelines

- Keep PRs focused and small when possible
- Provide a clear description of changes
- Reference related issues
- Ensure CI checks pass
- Request review from maintainers

## Project Structure

```
.
├── cmd/prisma/          # CLI executable
├── internal/            # Internal packages
│   ├── config/         # Configuration management
│   ├── parser/         # Schema parser
│   ├── migrations/     # Migration system
│   ├── dialect/        # Database dialects
│   └── logger/         # Logging utilities
├── builder/             # Query builder
├── generator/          # Code generation
├── docs/               # Documentation
└── docs/EXAMPLES.md    # Example code
```

## Areas for Contribution

- **Bug fixes**: Fix reported bugs
- **Features**: Implement new features from the roadmap
- **Documentation**: Improve or add documentation
- **Examples**: Add more practical examples
- **Tests**: Increase test coverage
- **Performance**: Optimize existing code
- **Database support**: Add support for new databases

## Questions?

If you have questions, feel free to:

- Open an issue for discussion
- Check existing issues and discussions
- Review the documentation

Thank you for contributing! 🎉
