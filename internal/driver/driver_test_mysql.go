//go:build mysql

package driver

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// setupMySQLTestDB is a helper function to setup MySQL test database
func setupMySQLTestDB(t *testing.T) (DB, func()) {
	baseURL := getTestDatabaseURL("mysql")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL_MYSQL not set, skipping MySQL test")
		return nil, nil
	}

	mysqlURL := removeDatabaseFromURL(baseURL)
	db, err := sql.Open("mysql", mysqlURL)
	if err != nil {
		t.Fatalf("failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	testDBName := fmt.Sprintf("prisma_test_%d", time.Now().UnixNano())
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	testURL := replaceDatabaseName(baseURL, testDBName)
	testDB, err := sql.Open("mysql", testURL)
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
		cleanupDB, err := sql.Open("mysql", mysqlURL)
		if err == nil {
			cleanupDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
			cleanupDB.Close()
		}
	}

	return adapter, cleanup
}

// TestSQLDBAdapter_MySQL tests SQL adapter with MySQL
func TestSQLDBAdapter_MySQL(t *testing.T) {
	db, cleanup := setupMySQLTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()

	// Test Exec
	result, err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS test_table (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255))")
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
