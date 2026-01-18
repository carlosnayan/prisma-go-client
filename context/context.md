# Project Context: Prisma Go Client

This document serves as a context guide for the development and maintenance of the `prisma-go-client` project.

## Overview

`prisma-go-client` is a native Go implementation of a Prisma client. Unlike other clients that may just be wrappers for Prisma's Rust binaries, this project implements its own schema parser, migration engine, and query builder.

## CLI Component Architecture

### Entry Point

- **Location:** `cmd/prisma/main.go`
- **Framework:** Uses a custom CLI framework defined in `cli/command.go`.

### Main Commands

- `init`: Initializes a new Prisma Go project.
- `generate`: Generates Go code (models, client, and query builders) from `schema.prisma`.
- `validate`: Explicitly validates the Prisma schema (relationships, types).
- `format`: Formats the `schema.prisma` file.
- `db`: Database management commands (`push`, `pull`, `seed`, `execute`).
- `migrate`: Migration management (`dev`, `deploy`, `reset`, `status`).

### Internal Logic

- **Parser (`internal/parser`):** Manual parser for `.prisma` files (ast, lexer, parser, validator).
- **Generator (`internal/generator`):** Uses Go templates (`.tmpl`) to generate the final code.
- **Migrations (`internal/migrations`):** Manages connections, database introspection, SQL diff generation, and migration execution.

## Query Builder Architecture

### Core Component (`builder/builder.go`)

- **`TableQueryBuilder`:** Base component that handles fundamental operations (`FindFirst`, `FindMany`, `Create`, `Update`, `Delete`, `Count`).
- **Database Abstraction:** Uses the `driver.DB` interface (`internal/driver`) to support multiple SQL drivers (PostgreSQL, MySQL, SQLite).
- **Dialect Support:** The `dialect.Dialect` interface allows handling database-specific particularities (placeholders, quoting, JSON, Full-text search).

### Fluent API (`builder/fluent.go`)

- **`Query` Struct:** The modern core of the query engine. It maintains internal state (`whereConditions`, `orderBy`, `take`, `skip`, `joins`, etc.).
- **Chaining Logic:** Methods like `.Where()`, `.Order()`, and `.Take()` mutate the `Query` state and return the pointer to the same `Query` instance, allowing for a fluent API.
- **State Management:**
  - `Reset()`: Essential for clearing the query state. This is particularly important when reusing query objects or within transactions to prevent criteria from leaking between operations.
  - `Clone()`: Creates a deep copy of the `Query` state, useful for creating variations of a base query.
- **Execution and Result Scanning:**
  - Uses internal methods like `scanRowIntoModel` and `scanRowsIntoModel` to map database rows to Go structs using reflection and field-to-column mapping.
  - Handles dialect-specific behaviors, such as the `RETURNING` clause in PostgreSQL via `UpdatesReturning`.

## Query Builder Generation and Evolution

### Generation Process

1. **Base Packages:** `internal/generator/builder.go` generates the foundational `builder` package in the output directory, including `builder.go` (legacy/utility) and `fluent.go` (modern).
2. **Type-Safe Filters:** `internal/generator/table_package.go` generates per-table packages (e.g., `db/tables/user`) that contain type-safe field definitions (e.g., `user.ID.Equals(1)`), moving logic away from raw strings.
3. **Model Queries:** `internal/generator/queries.go` generates the final `{{Model}}Query` structs (e.g., `UserQuery`) that wrap `builder.Query` and provide "Ent-style" builders for CRUD operations.

### Relationship between `TableQueryBuilder` and `Query`

- **`TableQueryBuilder` (Low-Level):** Originally the primary builder. Now primarily serves as a heavy-lifting internal helper for `Create` and `CreateMany` operations because it contains robust logic for retrieving fully materialized models after an insert (handling auto-increments and defaults across different dialects).
- **`Query` (High-Level):** The preferred API for `Find`, `Update`, and `Delete` operations. It provides the chaining logic and integrates directly with the type-safe filter system.

## Deprecated Patterns and Future Improvements

### Deprecated

- **Map-based `Updates()`:** The per-model `Updates(map[string]interface{})` method is deprecated/removed in favor of the fluent `Update().SetField(val).Exec()` builders.
- **Row Strings in Filters:** While still supported, using raw strings for field names in `.Where("field = ?", val)` is being replaced by type-safe table package filters.

### Future Roadmap

- **Phasing out `TableQueryBuilder`:** Aim to move the advanced insertion/materialization logic from `TableQueryBuilder` into `Query` to unify the core engine.
- **JSON/Full-Text Search Parity:** Ensure all dialects implement these advanced features consistently within the `Query` struct.
- **Reflection Optimization:** Explore code generation for result scanning to reduce reliance on reflection and improve performance.

## Key Components Summary

- **Configuration (`internal/config`):** Centralized TOML configuration (`prisma.conf`) with automatic `.env` loading and environment variable expansion (`${VAR}`, `env("VAR")`).
- **Safety Limits (`internal/limits`):** Hard limits to prevent OOM and DoS (e.g., 100k rows scan limit, 10MB raw query size).
- **Logging & Redaction (`internal/logger`):** Intelligent logger that redacts sensitive data (JWTs, passwords, API keys) from SQL query logs and tracks execution duration.
- **N+1 Query Detection (`internal/query`):** Built-in monitoring that identifies repetitive query patterns within a time window to alert developers of optimization opportunities.
- **Error Mapping & Sanitization (`internal/errors`):** Translates database errors to standard Prisma codes (P2XXX). In production mode, it sanitizes error messages to protect internal schema details.
- **CLI Commands (`cmd/prisma/cmd`):** Comprehensive toolset for schema validation, database introspection (`pull`), synchronisation (`push`), migration diffing, and native execution.
- **Query Builder (`builder/`):** A type-safe, fluent API generated from the schema, featuring automatic result scanning via reflection.
- **Transactions (`builder/transaction.go`):** Robust transaction management with automated rollback and context-aware timeouts.
- **Drivers & Dialects (`internal/driver`, `internal/dialect`):** Abstraction layers that handle dialect-specific SQL (Postgres, MySQL, SQLite) and underlying driver differences (e.g., `pgx` vs `database/sql`).

## Transaction Management (`builder/transaction.go`)

- **`Transaction` Struct:** Wraps a `driver.Tx` and provides an adapter (`txDBAdapter`) that implements the `driver.DB` interface. This allows reusing the same `Query` and `TableQueryBuilder` logic within a transaction.
- **Execution Patterns:**
  - `ExecuteTransaction`: A high-level helper that manages the boilerplate of `Begin`, `Commit`, and automated `Rollback` on errors or panics.
  - `ExecuteSequentialTransactions`: Allows running multiple independent transaction functions in a single database transaction.
- **Isolation:** Transaction contexts are managed by the caller, ensuring that timeouts and cancellations are respected throughout the operation.

## Abstraction Layers: Drivers and Dialects

### Database Drivers (`internal/driver`)

The project uses a custom `DB` interface to abstract different database connection libraries:

- **`sql.DB` Adapter:** Used for SQLite (`go-sqlite3`) and MySQL (`go-sql-driver/mysql`).
- **`pgx` Adapter:** A native adapter for PostgreSQL using the `pgx` library, supporting connection pooling via `pgxpool`.
- **Result & Rows Interfaces:** Abstract the differences between how various drivers return metadata and results.

### SQL Dialects (`internal/dialect`)

The `Dialect` interface defines how each database engine behaves:

- **Identifiers & Logic:** Handles quoting (backticks vs double quotes), placeholders ($1 vs ?), and specific SQL syntax for LIMIT/OFFSET/RETURNING.
- **Feature Support:** Detects support for advanced features like JSON operations and Full-Text Search.
- **Type Mapping:** Translates Prisma types (e.g., `DateTime`) to the appropriate database-specific type (e.g., `TIMESTAMP` vs `DATETIME`).

## Raw SQL Support

- **Interface:** Allows users to execute arbitrary SQL queries with optional result scanning.
- **Fluent API:** `Query().Scan(&dest).Exec()` for SELECT queries, `Query().Exec()` for DDL/DML without results.
- **Scanning:** Unified `scanInto()` function automatically detects and scans primitives, single structs, or slices of structs using field reflection.
- **Safety:** Encourages the use of parameterized queries through the `driver.DB` interface to prevent SQL injection.

## Configuration System (`internal/config`)

- **TOML Based:** Uses `prisma.conf` as the main configuration file.
- **Environment Integration:**
  - Automatically looks for `.env` or `env` files in parent directories.
  - Supports `env("VAR")`, `${VAR}`, and `$VAR` syntax for dynamic values.
- **Diagnostics:** Includes a dedicated diagnostic system to help users configure the `DATABASE_URL` correctly.

## Memory & Safety Limits (`internal/limits`)

To ensure stability and protect against OOM (Out of Memory):

- **Scan Limit:** `MaxScanRows = 100,000` rows.
- **Query Complexity:** `MaxQueryConditions = 1,000`, `MaxJoins = 50`.
- **Field Limits:** `MaxOrderByFields = 20`, `MaxGroupByFields = 20`, `MaxSelectFields = 100`.
- **Raw SQL:** `MaxRawQuerySize = 10MB`.

## Observability & Security (`internal/logger`, `internal/query`)

- **Query Logging:** Logs full SQL queries with duration.
- **Sensitive Data Redaction:** Automatically detects and obscures passwords, JWTs, API keys, and other credentials in logs.
- **N+1 Detection:** Tracks query patterns (e.g., same SELECT executed multiple times in a window) and alerts the developer using a `callback` mechanism.

## Error Handling (`internal/errors`)

- **Sentinel Errors:** Replicates Prisma's P-series error codes (P2002, P2025, etc.).
- **Smart Mapping:** Intelligently maps SQL/Driver errors to Prisma sentinels based on error codes and message regex.
- **Production Sanitization:** In `production` mode, error messages are stripped of table and column names to prevent schema leaking.

## CLI Deep-Dive (`cmd/prisma/cmd`)

- **`validate`**: Performs relationship consistency checks (e.g., matching field counts in `@relation`) and type verification beyond simple syntax parsing.
- **`db push`**: Synchronizes the database schema directly without migrations, including data loss warnings and optional code generation.
- **`db pull`**: Introspects a live database and generates a `schema.prisma` file, preserving existing comments and block structures.
- **`migrate diff`**: Extremely flexible tool for comparing any two states: File-to-File, File-to-DB, or DB-to-File.

### Fluent API (Generated Builders)

This is the modern, type-safe API used by developers. It uses specialized builders for different operations.

| Builder               | Methods                                                          | Returns            |
| :-------------------- | :--------------------------------------------------------------- | :----------------- |
| **`FindFirst()`**     | `Where()`, `OrderBy()`, `Select()`, `Exec()`                     | `(*Model, error)`  |
| **`FindMany()`**      | `Where()`, `OrderBy()`, `Take()`, `Skip()`, `Select()`, `Exec()` | `([]Model, error)` |
| **`Create()`**        | `Set[Field]()`, `Exec()`                                         | `(*Model, error)`  |
| **`Update()`**        | `Where()`, `Set[Field]()`, `Exec()`                              | `(*Model, error)`  |
| **`UpdateOneID(id)`** | `Set[Field]()`, `Exec()`                                         | `error`            |
| **`UpdateMany()`**    | `Where()`, `Set[Field]()`, `Exec()`                              | `error`            |
| **`Delete()`**        | `Where()`, `Exec()`                                              | `error`            |

> [!NOTE]
> All builders support `WithContext(ctx)` to explicitly pass a context. The standard `Exec()` uses the context stored in the query (usually `context.Background()` unless set via `WithContext`).

> [!IMPORTANT]
> Although the internal engine (`builder.Query`) supports `Group`, `Having`, and `Join`, these are **not exposed** in the current public Fluent API to maintain Prisma compatibility and type safety.

### Table Package (`TableQueryBuilder` struct)

Low-level operations usually called by generated code.

- **`FindFirst(ctx, where)`**: Retrieves one record using a map-based filter.
- **`FindMany(ctx, opts)`**: Retrieves multiple records using a `QueryOptions` struct.
- **`Count(ctx, where)`**: Simple count based on map filters.
- **`Create(ctx, data)`**: Direct insertion of a model struct.
- **`Update(ctx, id, data)`**: Updates a record by ID using a model struct.
- **`Delete(ctx, id)`**: Deletes a single record by ID.
- **`CreateMany(ctx, data)`**: Batch insertion logic.
- **`UpdateMany(ctx, where, data)`**: Batch update logic.

## Implementation Details

### Type Handling

- Supports primitive types, enums, JSON, UUID, DateTime, etc.
- Handles nullability through pointers or optional types.

### Safety and Performance

- Uses Go context for query timeouts.
- Protection against SQL Injection using placeholders.
- Database error sanitization.
- **Connection Pool Configuration:** Programmatically configured via `SetupClientWithOptions()` using `ClientOptions` and `PoolOptions` structs. This allows type-safe pool configuration with validation for MaxConns, MinConns, MaxConnLifetime, and PostgreSQL-specific settings (MaxIdleTime, HealthCheckPeriod, ConnectTimeout). The legacy `[datasource.pool]` configuration in `prisma.conf` has been removed.

### Extensibility

- New Prisma features (like `raw` queries or new operators) can be added by extending `TableQueryBuilder` or adding new templates to the generator.

## Notes for the Future

- **Query Builder:** Maintain parity with the official Prisma API.
- **Performance:** Evaluate reflection overhead in result scanning.
- **Drivers:** Expand support to other databases if necessary.
- **CLI:** Improve error messages and visual feedback during `generate` and `migrate`.
