package raw

import (
	"context"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
)

// Executor provides methods for executing raw SQL queries
type Executor struct {
	db driver.DB
}

// DBTX is an alias for driver.DB for backward compatibility
type DBTX = driver.DB

// New creates a new raw query executor
func New(db driver.DB) *Executor {
	return &Executor{db: db}
}

// Query executes a raw SQL query that returns multiple rows
//
// Example:
//
//	rows, err := executor.Query(ctx, `
//	    SELECT u.*, t.tenant_name
//	    FROM users u
//	    JOIN tenants t ON u.id_tenant = t.id_tenant
//	    WHERE u.deleted_at IS NULL AND t.id_tenant = $1
//	`, tenantId)
func (e *Executor) Query(ctx context.Context, sql string, args ...interface{}) (driver.Rows, error) {
	return e.db.Query(ctx, sql, args...)
}

// QueryRow executes a raw SQL query that returns a single row
//
// Example:
//
//	row := executor.QueryRow(ctx, `
//	    SELECT COUNT(*) as total,
//	           COUNT(CASE WHEN enabled = true THEN 1 END) as enabled_count
//	    FROM users
//	    WHERE id_tenant = $1
//	`, tenantId)
//
//	var total, enabled int
//	err := row.Scan(&total, &enabled)
func (e *Executor) QueryRow(ctx context.Context, sql string, args ...interface{}) driver.Row {
	return e.db.QueryRow(ctx, sql, args...)
}

// Exec executes a raw SQL command (INSERT, UPDATE, DELETE)
//
// Example:
//
//	result, err := executor.Exec(ctx, `
//	    UPDATE users
//	    SET last_login_at = NOW()
//	    WHERE id_user = $1
//	`, userId)
func (e *Executor) Exec(ctx context.Context, sql string, args ...interface{}) (driver.Result, error) {
	return e.db.Exec(ctx, sql, args...)
}
