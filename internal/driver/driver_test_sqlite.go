//go:build sqlite

package driver

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// setupSQLiteTestDB is a helper function to setup SQLite test database
func setupSQLiteTestDB(t *testing.T) (DB, func()) {
	tmpFile, err := os.CreateTemp("", "prisma_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	dbPath := tmpFile.Name()
	dbURL := fmt.Sprintf("file:%s", dbPath)

	db, err := sql.Open("sqlite3", dbURL)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("failed to connect to SQLite: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		os.Remove(dbPath)
		t.Fatalf("failed to ping SQLite database: %v", err)
	}

	adapter := NewSQLDB(db)
	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return adapter, cleanup
}

// TestSQLDBAdapter_SQLite tests SQL adapter with SQLite
func TestSQLDBAdapter_SQLite(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test Exec
	result, err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	_ = result

	// Test Query
	rows, err := db.Query(ctx, "SELECT 1 as value")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("Query returned no rows")
	}

	var value int
	if err := rows.Scan(&value); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if value != 1 {
		t.Errorf("Expected value 1, got %d", value)
	}

	// Test QueryRow
	row := db.QueryRow(ctx, "SELECT 2 as value")
	var value2 int
	if err := row.Scan(&value2); err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if value2 != 2 {
		t.Errorf("Expected value 2, got %d", value2)
	}

	// Test Transaction
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO test_table (name) VALUES (?)", "test")
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("Exec in transaction failed: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
}
