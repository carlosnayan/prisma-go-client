package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
)

// SeedTestData inserts test fixtures into the database
func SeedTestData(t *testing.T, db driver.DB, provider string) {
	// This is a placeholder - users should implement their own seed data
	// based on their schema
	_ = t
	_ = db
	_ = provider
}

// CleanTestData cleans all test data (TRUNCATE or DELETE)
func CleanTestData(t *testing.T, db driver.DB, provider string) {
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB() method")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all tables
	var tables []string
	switch provider {
	case "postgresql":
		rows, err := sqlDB.QueryContext(ctx, `
			SELECT tablename 
			FROM pg_tables 
			WHERE schemaname = 'public' 
			AND tablename NOT LIKE '_prisma%'
		`)
		if err != nil {
			return // No tables or error
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				continue
			}
			tables = append(tables, table)
		}
	case "mysql":
		rows, err := sqlDB.QueryContext(ctx, "SHOW TABLES")
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				continue
			}
			if table != "_prisma_migrations" {
				tables = append(tables, table)
			}
		}
	case "sqlite":
		rows, err := sqlDB.QueryContext(ctx, `
			SELECT name 
			FROM sqlite_master 
			WHERE type='table' 
			AND name NOT LIKE 'sqlite_%' 
			AND name != '_prisma_migrations'
		`)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				continue
			}
			tables = append(tables, table)
		}
	}

	// Clean each table
	for _, table := range tables {
		switch provider {
		case "postgresql":
			_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		case "mysql":
			_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s", table))
		case "sqlite":
			_, _ = sqlDB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table))
		}
	}
}

// WithTestTransaction executes a test function within a transaction and rolls back
func WithTestTransaction(t *testing.T, db driver.DB, fn func(tx driver.Tx)) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Always rollback
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Logf("warning: failed to rollback transaction: %v", err)
		}
	}()

	// Execute test function
	fn(tx)
}
