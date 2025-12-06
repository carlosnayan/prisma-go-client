package testing

import (
	"os"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
)

// SetupTestDB creates a test database and returns connection + cleanup function
// Note: Driver-specific implementations are in separate files with build tags
func SetupTestDB(t *testing.T, provider string) (driver.DB, func()) {
	switch provider {
	case "postgresql":
		return SetupPostgreSQLTestDB(t)
	case "mysql":
		return SetupMySQLTestDB(t)
	case "sqlite":
		return SetupSQLiteTestDB(t)
	default:
		t.Fatalf("unsupported provider: %s", provider)
		return nil, nil
	}
}

// GetTestDatabaseURL gets test database URL from environment variables
func GetTestDatabaseURL(provider string) string {
	var envVar string
	switch provider {
	case "postgresql":
		envVar = os.Getenv("TEST_DATABASE_URL_POSTGRESQL")
		if envVar == "" {
			envVar = os.Getenv("TEST_DATABASE_URL")
		}
	case "mysql":
		envVar = os.Getenv("TEST_DATABASE_URL_MYSQL")
		if envVar == "" {
			envVar = os.Getenv("TEST_DATABASE_URL")
		}
	case "sqlite":
		envVar = os.Getenv("TEST_DATABASE_URL_SQLITE")
		if envVar == "" {
			envVar = os.Getenv("TEST_DATABASE_URL")
		}
	}
	return envVar
}

// CleanupTestDB cleans up test database
func CleanupTestDB(t *testing.T, db driver.DB, provider string) {
	// Cleanup is handled by the cleanup function returned from SetupTestDB
	// This function is kept for compatibility but does nothing
	_ = t
	_ = db
	_ = provider
}

// Helper functions

// replaceDatabaseName replaces database name in URL
//
//nolint:unused // Used by test files with build tags
func replaceDatabaseName(url, dbName string) string {
	// Simple implementation - works for most cases
	// For PostgreSQL: postgresql://user:pass@host:port/dbname -> replace dbname
	// For MySQL: mysql://user:pass@host:port/dbname -> replace dbname
	if url == "" {
		return url
	}

	// Find last / and replace everything after it
	lastSlash := -1
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash == -1 {
		return url + "/" + dbName
	}

	// Check if there's a query string
	queryStart := -1
	for i := lastSlash; i < len(url); i++ {
		if url[i] == '?' {
			queryStart = i
			break
		}
	}

	if queryStart != -1 {
		return url[:lastSlash+1] + dbName + url[queryStart:]
	}

	return url[:lastSlash+1] + dbName
}

// removeDatabaseFromURL removes database name from URL
//
//nolint:unused // Used by test files with build tags
func removeDatabaseFromURL(url string) string {
	// Find last / and remove everything after it (except query string)
	lastSlash := -1
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash == -1 {
		return url
	}

	// Check if there's a query string
	queryStart := -1
	for i := lastSlash; i < len(url); i++ {
		if url[i] == '?' {
			queryStart = i
			break
		}
	}

	if queryStart != -1 {
		return url[:lastSlash+1] + url[queryStart:]
	}

	return url[:lastSlash+1]
}
