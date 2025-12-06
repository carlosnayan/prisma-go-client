//go:build sqlite

package testing

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SetupSQLiteTestDB creates a SQLite test database (uses temp file)
func SetupSQLiteTestDB(t *testing.T) (driver.DB, func()) {
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

	adapter := driver.NewSQLDB(db)
	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return adapter, cleanup
}
