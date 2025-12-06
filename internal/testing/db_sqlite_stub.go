//go:build !sqlite

package testing

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
)

// SetupSQLiteTestDB creates a SQLite test database
// This stub version skips the test if SQLite driver is not available
func SetupSQLiteTestDB(t *testing.T) (driver.DB, func()) {
	t.Skip("SQLite driver not available. Install with: go get github.com/mattn/go-sqlite3 and run tests with -tags=sqlite")
	return nil, nil
}
