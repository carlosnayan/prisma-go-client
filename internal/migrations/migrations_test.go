package migrations

import (
	"os"
	"path/filepath"
	"testing"

	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// TestMigrations_PostgreSQL tests migrations on PostgreSQL
func TestMigrations_PostgreSQL(t *testing.T) {
	testutil.SkipIfNoDatabase(t, "postgresql")
	db, cleanup := testutil.SetupTestDB(t, "postgresql")
	defer cleanup()

	// Create test migrations directory
	migrationsDir := t.TempDir()
	migration1Dir := filepath.Join(migrationsDir, "20240101000000_initial")
	if err := os.MkdirAll(migration1Dir, 0755); err != nil {
		t.Fatalf("failed to create migration directory: %v", err)
	}

	migrationSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255)
		);
	`
	if err := os.WriteFile(filepath.Join(migration1Dir, "migration.sql"), []byte(migrationSQL), 0644); err != nil {
		t.Fatalf("failed to write migration file: %v", err)
	}

	// Apply migrations
	testutil.ApplyTestMigrations(t, db, migrationsDir, "postgresql")

	// Verify table exists
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB()")
	}

	rows, err := sqlDB.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users'")
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("users table was not created")
	}
}

// TestMigrations_MySQL tests migrations on MySQL
func TestMigrations_MySQL(t *testing.T) {
	testutil.SkipIfNoDatabase(t, "mysql")
	db, cleanup := testutil.SetupTestDB(t, "mysql")
	defer cleanup()

	// Create test migrations directory
	migrationsDir := t.TempDir()
	migration1Dir := filepath.Join(migrationsDir, "20240101000000_initial")
	if err := os.MkdirAll(migration1Dir, 0755); err != nil {
		t.Fatalf("failed to create migration directory: %v", err)
	}

	migrationSQL := `
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255)
		);
	`
	if err := os.WriteFile(filepath.Join(migration1Dir, "migration.sql"), []byte(migrationSQL), 0644); err != nil {
		t.Fatalf("failed to write migration file: %v", err)
	}

	// Apply migrations
	testutil.ApplyTestMigrations(t, db, migrationsDir, "mysql")

	// Verify table exists
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB()")
	}

	rows, err := sqlDB.Query("SHOW TABLES LIKE 'users'")
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("users table was not created")
	}
}

// TestMigrations_SQLite is tested in migrations_test_sqlite.go (requires build tag)

