package testing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
)

// ApplyTestMigrations applies all migrations from the migrations path
func ApplyTestMigrations(t *testing.T, db driver.DB, migrationsPath string, provider string) {
	// Get SQLDB for migrations
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB() method")
	}

	// Read all migration files
	migrationDirs, err := os.ReadDir(migrationsPath)
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	// Sort migration directories by name (they should be timestamped)
	for _, dir := range migrationDirs {
		if !dir.IsDir() {
			continue
		}

		migrationFile := filepath.Join(migrationsPath, dir.Name(), "migration.sql")
		sqlContent, err := os.ReadFile(migrationFile)
		if err != nil {
			t.Fatalf("failed to read migration file %s: %v", migrationFile, err)
		}

		// Execute migration
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err = sqlDB.ExecContext(ctx, string(sqlContent))
		if err != nil {
			t.Fatalf("failed to apply migration %s: %v", dir.Name(), err)
		}
	}
}

// RollbackTestMigrations rolls back all migrations or drops schema
func RollbackTestMigrations(t *testing.T, db driver.DB, provider string) {
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB() method")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch provider {
	case "postgresql":
		// Drop all tables in public schema
		_, err := sqlDB.ExecContext(ctx, `
			DO $$ DECLARE
				r RECORD;
			BEGIN
				FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
					EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
				END LOOP;
			END $$;
		`)
		if err != nil {
			t.Fatalf("failed to rollback PostgreSQL migrations: %v", err)
		}
	case "mysql":
		// Drop all tables
		rows, err := sqlDB.QueryContext(ctx, "SHOW TABLES")
		if err != nil {
			t.Fatalf("failed to list MySQL tables: %v", err)
		}
		defer rows.Close()

		var tables []string
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				continue
			}
			tables = append(tables, table)
		}

		if len(tables) > 0 {
			query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tables[0])
			for i := 1; i < len(tables); i++ {
				query += ", " + tables[i]
			}
			_, err = sqlDB.ExecContext(ctx, query)
			if err != nil {
				t.Fatalf("failed to rollback MySQL migrations: %v", err)
			}
		}
	case "sqlite":
		// Drop all tables
		rows, err := sqlDB.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
		if err != nil {
			t.Fatalf("failed to list SQLite tables: %v", err)
		}
		defer rows.Close()

		var tables []string
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				continue
			}
			tables = append(tables, table)
		}

		for _, table := range tables {
			_, err = sqlDB.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
			if err != nil {
				t.Fatalf("failed to drop SQLite table %s: %v", table, err)
			}
		}
	}
}

// ResetTestDatabase drops all tables and reapplies migrations
func ResetTestDatabase(t *testing.T, db driver.DB, migrationsPath string, provider string) {
	RollbackTestMigrations(t, db, provider)
	ApplyTestMigrations(t, db, migrationsPath, provider)
}

// GetMigrationStatus gets the status of migrations
func GetMigrationStatus(t *testing.T, db driver.DB, provider string) ([]string, error) {
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		return nil, fmt.Errorf("database does not support SQLDB() method")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if _prisma_migrations table exists
	var migrations []string

	switch provider {
	case "postgresql":
		rows, err := sqlDB.QueryContext(ctx, `
			SELECT migration_name 
			FROM _prisma_migrations 
			ORDER BY finished_at
		`)
		if err != nil {
			// Table might not exist yet
			return migrations, nil
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				continue
			}
			migrations = append(migrations, name)
		}
	case "mysql", "sqlite":
		rows, err := sqlDB.QueryContext(ctx, `
			SELECT migration_name 
			FROM _prisma_migrations 
			ORDER BY finished_at
		`)
		if err != nil {
			// Table might not exist yet
			return migrations, nil
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				continue
			}
			migrations = append(migrations, name)
		}
	}

	return migrations, nil
}
