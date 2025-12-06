# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed (Breaking)

- **Driver Abstraction**: Removed direct dependency on `pgx`
  - `NewClient()` now accepts `driver.DB` instead of `*pgxpool.Pool`
  - Users must install their own database drivers (pgx, mysql, sqlite3)
  - Query builder interfaces changed from pgx-specific to driver-agnostic
  - All database operations now use abstract `driver.DB` interface

### Added

- **Driver Abstraction Package**: `internal/driver` with abstract interfaces

  - `DB`, `Result`, `Rows`, `Row`, `Tx` interfaces
  - `PgxPoolAdapter` for PostgreSQL (with build tag)
  - `SQLDBAdapter` for MySQL/SQLite via `database/sql`
  - Support for multiple database drivers without core dependencies

- **Documentation Improvements**:
  - Complete beginner-friendly Quick Start guide
  - Simple examples for Query Builder and Raw SQL
  - Database-specific installation and usage examples
  - Removed complex examples folder

### Removed

- `examples/` folder - replaced with simple examples in documentation
- Direct `pgx` dependency from core library

## [0.1.0] - 2024-12-19

### Added

#### Core Features

- **CLI**: Complete Prisma-like CLI with Cobra framework
  - `prisma init`: Initialize new projects
  - `prisma generate`: Generate type-safe Go code from schema.prisma
  - `prisma validate`: Validate schema.prisma syntax and consistency
  - `prisma format`: Format schema.prisma files
  - `prisma migrate dev`: Create and apply migrations in development
  - `prisma migrate deploy`: Apply pending migrations in production
  - `prisma migrate reset`: Reset database and reapply all migrations
  - `prisma migrate status`: Check migration status
  - `prisma migrate resolve`: Manually resolve migration conflicts
  - `prisma migrate diff`: Compare schemas and generate migration SQL
  - `prisma db push`: Apply schema changes directly to database
  - `prisma db pull`: Introspect database and generate schema.prisma
  - `prisma db seed`: Execute database seed scripts
  - `prisma db execute`: Execute arbitrary SQL

#### Schema Management

- **Parser**: Complete lexer and AST parser for schema.prisma
  - Support for models, fields, enums, relations
  - Type system: String, Int, Boolean, DateTime, Float, Decimal, Json, Bytes, BigInt
  - Attributes: @id, @unique, @default, @relation, @map, @updatedAt
  - Model attributes: @@id, @@unique, @@index, @@map
  - Function support: env(), autoincrement(), now(), uuid(), cuid()

#### Configuration

- **prisma.conf**: TOML-based configuration file
  - Environment variable expansion: `env('DATABASE_URL')`
  - Automatic .env file loading with godotenv
  - Migration path configuration
  - Seed script configuration
  - Connection pool settings

#### Query Builder

- **Fluent API**: 100% type-safe query builder with Prisma-like syntax
  - **Create**: `client.User.Create().Data(inputs.UserCreateInput{...}).Exec(ctx)`
  - **FindMany**: `client.User.FindMany().Select(...).Where(...).Exec(ctx)`
  - **FindFirst**: `client.User.FindFirst().Select(...).Where(...).Exec(ctx)`
  - **Update**: `client.User.Update().Where(...).Data(...).Exec(ctx)`
  - **Delete**: `client.User.Delete().Where(...).Exec(ctx)`
  - **Type-safe WhereInput**: All fields use Filter types (StringFilter, IntFilter, etc.)
  - **Type-safe Select**: Field selection with compile-time safety
  - Advanced WHERE operators: Gt, Gte, Lt, Lte, In, NotIn, Contains, StartsWith, EndsWith
  - Logical operators: AND, OR, NOT in WhereInput
  - Null checks: IsNull, IsNotNull
  - Automatic timestamps: created_at, updated_at

#### Database Support

- **Multi-database**: Dialect abstraction for database-specific SQL
  - PostgreSQL: Full support with JSONB, full-text search
  - MySQL: Full support with JSON, full-text search
  - SQLite: Full support with JSON functions
  - SQL Server: Planned for future release

#### Migrations

- **Migration System**: Complete migration management
  - Automatic migration generation from schema changes
  - Database introspection and comparison
  - Migration history tracking
  - Rollback support
  - Migration status checking
  - Schema diff generation

#### Documentation

- **Comprehensive Documentation**:
  - Quick Start Guide
  - Complete API Reference
  - Migrations Guide
  - Relationships Guide
  - Best Practices
  - Examples Overview

#### Examples

- **Practical Examples**:
  - REST API example with gorilla/mux
  - GraphQL example with graphql-go
  - Microservices architecture example
  - Multi-tenancy example

#### Utilities

- **Logging**: Structured logging with levels (query, info, warn, error)
- **Connection Pooling**: Configurable connection pool with health checks
- **Error Handling**: Comprehensive error handling and validation

### Changed

- Initial release

### Fixed

- N/A (initial release)

### Security

- Parameterized queries to prevent SQL injection
- Environment variable support for sensitive credentials

---

## [Unreleased]

### Planned

- SQL Server dialect support
- Prisma Studio equivalent (web interface)
- Performance metrics
- Comprehensive test suite
- Additional database drivers
