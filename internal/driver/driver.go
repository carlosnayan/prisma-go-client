package driver

import (
	"context"
	"database/sql"
)

// DB is the main database interface that abstracts different database drivers
type DB interface {
	// Exec executes a query that doesn't return rows
	Exec(ctx context.Context, sql string, args ...interface{}) (Result, error)

	// Query executes a query that returns multiple rows
	Query(ctx context.Context, sql string, args ...interface{}) (Rows, error)

	// QueryRow executes a query that returns a single row
	QueryRow(ctx context.Context, sql string, args ...interface{}) Row

	// Begin starts a transaction
	Begin(ctx context.Context) (Tx, error)

	// SQLDB returns the underlying *sql.DB for migrations and introspection
	// Returns nil if not available (e.g., for pgx pool)
	SQLDB() *sql.DB
}

// Result represents the result of an Exec operation
type Result interface {
	// RowsAffected returns the number of rows affected
	RowsAffected() int64
}

// Rows represents a set of query results
type Rows interface {
	// Close closes the rows iterator
	Close()

	// Err returns any error that occurred during iteration
	Err() error

	// Next prepares the next result row for reading
	Next() bool

	// Scan copies the columns in the current row into the values pointed at by dest
	Scan(dest ...interface{}) error
}

// Row represents a single row result
type Row interface {
	// Scan copies the columns in the current row into the values pointed at by dest
	Scan(dest ...interface{}) error
}

// Tx represents a database transaction
type Tx interface {
	// Commit commits the transaction
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction
	Rollback(ctx context.Context) error

	// Exec executes a query that doesn't return rows
	Exec(ctx context.Context, sql string, args ...interface{}) (Result, error)

	// Query executes a query that returns multiple rows
	Query(ctx context.Context, sql string, args ...interface{}) (Rows, error)

	// QueryRow executes a query that returns a single row
	QueryRow(ctx context.Context, sql string, args ...interface{}) Row
}
