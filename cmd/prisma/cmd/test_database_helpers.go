package cmd

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// generateTestUUID generates a simple UUID for test database names
func generateTestUUID() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	uuid := make([]byte, 36)
	template := "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"

	for i, c := range template {
		switch c {
		case 'x':
			uuid[i] = "0123456789abcdef"[rng.Intn(16)]
		case 'y':
			uuid[i] = "89ab"[rng.Intn(4)]
		case '-':
			uuid[i] = '-'
		default:
			uuid[i] = byte(c)
		}
	}

	return string(uuid)
}

// createIsolatedTestDB creates a unique isolated test database
// Returns the database name and a cleanup function to drop the database
func createIsolatedTestDB(t *testing.T) (dbName string, cleanup func()) {
	t.Helper()

	// Check if TEST_DATABASE_URL is set (don't use default value)
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database test")
	}

	// Generate unique database name
	dbName = fmt.Sprintf("prisma_test_%s", strings.ReplaceAll(generateTestUUID()[:8], "-", ""))

	// Connect to the default/postgres database
	db, err := sql.Open("pgx", baseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create the test database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("Failed to create test database %s: %v", dbName, err)
	}

	t.Logf("Created isolated test database: %s", dbName)

	// Return cleanup function
	cleanup = func() {
		db, err := sql.Open("pgx", baseURL)
		if err != nil {
			t.Logf("Warning: failed to connect for cleanup: %v", err)
			return
		}
		defer db.Close()

		// Terminate active connections to the database
		_, _ = db.Exec(fmt.Sprintf(`
			SELECT pg_terminate_backend(pg_stat_activity.pid)
			FROM pg_stat_activity
			WHERE pg_stat_activity.datname = '%s'
			AND pid <> pg_backend_pid()
		`, dbName))

		// Drop the database
		_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		if err != nil {
			t.Logf("Warning: failed to drop test database %s: %v", dbName, err)
		} else {
			t.Logf("Dropped isolated test database: %s", dbName)
		}
	}

	return dbName, cleanup
}

// getTestDBURL returns the connection URL for the isolated test database
// Replaces the database name in the base URL with the provided dbName
func getTestDBURL(t *testing.T, dbName string) string {
	t.Helper()

	baseURL := getTestDatabaseURL(t)
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	return replaceDBNameInURL(baseURL, dbName)
}

// replaceDBNameInURL replaces the database name in a connection URL
// Example: postgresql://user:pass@host:port/olddb -> postgresql://user:pass@host:port/newdb
func replaceDBNameInURL(connectionURL, newDBName string) string {
	// Parse the URL
	u, err := url.Parse(connectionURL)
	if err != nil {
		// If parsing fails, try simple string replacement
		parts := strings.Split(connectionURL, "/")
		if len(parts) > 3 {
			// Remove query params from last part if exists
			lastPart := parts[len(parts)-1]
			if idx := strings.Index(lastPart, "?"); idx != -1 {
				parts[len(parts)-1] = newDBName + lastPart[idx:]
			} else {
				parts[len(parts)-1] = newDBName
			}
			return strings.Join(parts, "/")
		}
		return connectionURL
	}

	// Replace the path (database name) preserving query parameters
	u.Path = "/" + newDBName

	return u.String()
}

// execSQL executes SQL statements on the test database
// Useful for setting up test data or schema before running commands
func execSQL(t *testing.T, dbURL string, sqlStatements ...string) {
	t.Helper()

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	for _, stmt := range sqlStatements {
		_, err := db.Exec(stmt)
		if err != nil {
			t.Fatalf("Failed to execute SQL: %v\nSQL: %s", err, stmt)
		}
	}
}

// tableExists checks if a table exists in the database
func tableExists(t *testing.T, dbURL, tableName string) bool {
	t.Helper()

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`
	err = db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}

	return exists
}
