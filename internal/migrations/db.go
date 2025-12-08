package migrations

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
	// Note: Users must import their database driver, e.g.:
	// _ "github.com/jackc/pgx/v5/stdlib" for PostgreSQL
	// _ "github.com/go-sql-driver/mysql" for MySQL
	// _ "github.com/mattn/go-sqlite3" for SQLite
)

// ConnectDatabase connects to the database using the provided URL
func ConnectDatabase(url string) (*sql.DB, error) {
	// Detect provider from URL
	provider := DetectProvider(url)

	// Get dialect to use the correct driver
	d := dialect.GetDialect(provider)
	driverName := d.GetDriverName()

	// For PostgreSQL, ensure we're using the correct driver
	// The pgx driver via stdlib requires the driver to be imported
	if provider == "postgresql" {
		// Check if pgx driver is available by trying to open
		// If driver is not registered, sql.Open will fail with "sql: unknown driver"
		db, err := sql.Open(driverName, url)
		if err != nil {
			// Check if it's a driver registration error
			if strings.Contains(err.Error(), "unknown driver") || strings.Contains(err.Error(), "sql: unknown driver") {
				return nil, fmt.Errorf(`driver "pgx" is not registered. Import the driver in your code:
  import _ "github.com/jackc/pgx/v5/stdlib"
  
Or use the default PostgreSQL driver by changing the dialect to use "postgres" instead of "pgx"`)
			}
			return nil, fmt.Errorf("failed to open database connection: %w", err)
		}

		// Test connection with timeout context
		if err := db.Ping(); err != nil {
			db.Close()
			// Provide more detailed error information
			return nil, fmt.Errorf("failed to connect to database server: %w\n\nCheck:\n  - If the PostgreSQL server is running\n  - If the URL is correct: %s\n  - If the credentials are correct\n  - If the port is accessible", err, url)
		}

		return db, nil
	}

	// For other databases
	db, err := sql.Open(driverName, url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database server: %w", err)
	}

	return db, nil
}

// DetectProvider detects the provider from the URL
func DetectProvider(url string) string {
	url = strings.ToLower(url)

	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return "postgresql"
	}
	if strings.HasPrefix(url, "mysql://") {
		return "mysql"
	}
	if strings.HasPrefix(url, "sqlite://") || strings.HasPrefix(url, "file:") {
		return "sqlite"
	}

	// Default to PostgreSQL
	return "postgresql"
}

// GetProviderFromSchema gets the provider from the parsed schema
func GetProviderFromSchema(schema *parser.Schema) string {
	if len(schema.Datasources) > 0 {
		for _, field := range schema.Datasources[0].Fields {
			if field.Name == "provider" {
				if str, ok := field.Value.(string); ok {
					return strings.ToLower(str)
				}
			}
		}
	}
	return "postgresql" // Default
}
