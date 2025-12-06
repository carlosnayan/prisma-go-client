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

	db, err := sql.Open(driverName, url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database server: %w", err)
	}

	return db, nil
}

// DetectProvider detecta o provider pela URL
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

	// Default para PostgreSQL
	return "postgresql"
}

// GetProviderFromSchema obtém o provider do schema parseado
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
