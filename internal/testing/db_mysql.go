//go:build mysql

package testing

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// SetupMySQLTestDB creates a MySQL test database
func SetupMySQLTestDB(t *testing.T) (driver.DB, func()) {
	baseURL := GetTestDatabaseURL("mysql")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL_MYSQL not set, skipping MySQL test")
		return nil, nil
	}

	mysqlURL := removeDatabaseFromURL(baseURL)
	db, err := sql.Open("mysql", mysqlURL)
	if err != nil {
		t.Skipf("failed to open MySQL connection: %v", err)
		return nil, nil
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("MySQL not available: %v", err)
		return nil, nil
	}

	testDBName := fmt.Sprintf("prisma_test_%d", time.Now().UnixNano())
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	testURL := replaceDatabaseName(baseURL, testDBName)
	testDB, err := sql.Open("mysql", testURL)
	if err != nil {
		db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
		t.Skipf("failed to connect to test database: %v", err)
		return nil, nil
	}

	testCtx, testCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer testCancel()
	if err := testDB.PingContext(testCtx); err != nil {
		testDB.Close()
		db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
		t.Skipf("failed to ping test database: %v", err)
		return nil, nil
	}

	adapter := driver.NewSQLDB(testDB)
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
