//go:build pgx

package driver

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxPoolAdapter adapts *pgxpool.Pool to the driver.DB interface
// This file is only compiled when the pgx build tag is present
type PgxPoolAdapter struct {
	pool *pgxpool.Pool
}

// NewPgxPool creates a new adapter from *pgxpool.Pool
func NewPgxPool(pool *pgxpool.Pool) DB {
	return &PgxPoolAdapter{pool: pool}
}

// Exec executes a query that doesn't return rows
func (a *PgxPoolAdapter) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := a.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxResult{result: result}, nil
}

// Query executes a query that returns multiple rows
func (a *PgxPoolAdapter) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := a.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRows{rows: rows}, nil
}

// QueryRow executes a query that returns a single row
func (a *PgxPoolAdapter) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	row := a.pool.QueryRow(ctx, query, args...)
	return &PgxRow{row: row}
}

// Begin starts a transaction
func (a *PgxPoolAdapter) Begin(ctx context.Context) (Tx, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PgxTx{tx: tx}, nil
}

// SQLDB returns nil as pgxpool.Pool doesn't provide *sql.DB directly
// For migrations, users should use database/sql with pgx stdlib driver
func (a *PgxPoolAdapter) SQLDB() *sql.DB {
	return nil
}

// PgxResult wraps pgconn.CommandTag
type PgxResult struct {
	result pgconn.CommandTag
}

// RowsAffected returns the number of rows affected
func (r *PgxResult) RowsAffected() int64 {
	return r.result.RowsAffected()
}

// PgxRows wraps pgx.Rows
type PgxRows struct {
	rows pgx.Rows
}

// Close closes the rows iterator
func (r *PgxRows) Close() {
	r.rows.Close()
}

// Err returns any error that occurred during iteration
func (r *PgxRows) Err() error {
	return r.rows.Err()
}

// Next prepares the next result row for reading
func (r *PgxRows) Next() bool {
	return r.rows.Next()
}

// Scan copies the columns in the current row into the values pointed at by dest
func (r *PgxRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

// PgxRow wraps pgx.Row
type PgxRow struct {
	row pgx.Row
}

// Scan copies the columns in the current row into the values pointed at by dest
func (r *PgxRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

// PgxTx wraps pgx.Tx
type PgxTx struct {
	tx pgx.Tx
}

// Commit commits the transaction
func (t *PgxTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *PgxTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// Exec executes a query that doesn't return rows
func (t *PgxTx) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxResult{result: result}, nil
}

// Query executes a query that returns multiple rows
func (t *PgxTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRows{rows: rows}, nil
}

// QueryRow executes a query that returns a single row
func (t *PgxTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	row := t.tx.QueryRow(ctx, query, args...)
	return &PgxRow{row: row}
}

