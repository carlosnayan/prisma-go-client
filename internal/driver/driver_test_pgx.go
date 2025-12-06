//go:build pgx

package driver

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

// setupPostgreSQLTestDB is a helper function to setup PostgreSQL test database
func setupPostgreSQLTestDB(t *testing.T) (DB, func()) {
	baseURL := getTestDatabaseURL("postgresql")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL_POSTGRESQL not set, skipping PostgreSQL test")
		return nil, nil
	}

	postgresURL := replaceDatabaseName(baseURL, "postgres")
	db, err := sql.Open("pgx", postgresURL)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	testDBName := fmt.Sprintf("prisma_test_%d", time.Now().UnixNano())
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	testURL := replaceDatabaseName(baseURL, testDBName)
	testDB, err := sql.Open("pgx", testURL)
	if err != nil {
		db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
		t.Fatalf("failed to connect to test database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := testDB.PingContext(ctx); err != nil {
		testDB.Close()
		db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
		t.Fatalf("failed to ping test database: %v", err)
	}

	adapter := NewSQLDB(testDB)
	cleanup := func() {
		testDB.Close()
		cleanupDB, err := sql.Open("pgx", postgresURL)
		if err == nil {
			cleanupDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
			cleanupDB.Close()
		}
	}

	return adapter, cleanup
}

// TestSQLDBAdapter_PostgreSQL tests SQL adapter with PostgreSQL
func TestSQLDBAdapter_PostgreSQL(t *testing.T) {
	db, cleanup := setupPostgreSQLTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()

	// Test Exec
	result, err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS test_table (id SERIAL PRIMARY KEY, name VARCHAR(255))")
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

	_, err = tx.Exec(ctx, "INSERT INTO test_table (name) VALUES ($1)", "test")
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("Exec in transaction failed: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
}

// TestPgxPoolAdapter tests pgx adapter (requires build tag)
func TestPgxPoolAdapter(t *testing.T) {
	baseURL := getTestDatabaseURL("postgresql")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL_POSTGRESQL not set, skipping pgx pool test")
		return
	}

	// This test requires pgx pool setup
	// For now, we'll test the SQL adapter which works with pgx stdlib driver
	// Full pgx pool adapter test would require additional setup
	t.Skip("PgxPoolAdapter test requires additional pgx pool setup")
}
