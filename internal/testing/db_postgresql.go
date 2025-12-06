//go:build pgx

package testing

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

// SetupPostgreSQLTestDB creates a PostgreSQL test database
func SetupPostgreSQLTestDB(t *testing.T) (driver.DB, func()) {
	baseURL := GetTestDatabaseURL("postgresql")
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

	adapter := driver.NewSQLDB(testDB)
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
