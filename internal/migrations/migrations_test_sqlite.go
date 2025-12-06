//go:build sqlite

package migrations

import (
	"os"
	"path/filepath"
	"testing"

	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// TestMigrations_SQLite tests migrations on SQLite
func TestMigrations_SQLite(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t, "sqlite")
	defer cleanup()

	// Create test migrations directory
	migrationsDir := t.TempDir()
	migration1Dir := filepath.Join(migrationsDir, "20240101000000_initial")
	os.MkdirAll(migration1Dir, 0755)

	migrationSQL := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			name TEXT
		);
	`
	os.WriteFile(filepath.Join(migration1Dir, "migration.sql"), []byte(migrationSQL), 0644)

	// Apply migrations
	testutil.ApplyTestMigrations(t, db, migrationsDir, "sqlite")

	// Verify table exists
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB()")
	}

	rows, err := sqlDB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='users'")
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("users table was not created")
	}
}
