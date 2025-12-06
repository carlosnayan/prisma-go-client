//go:build !mysql

package testing

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
)

// SetupMySQLTestDB creates a MySQL test database
// This stub version skips the test if MySQL driver is not available
func SetupMySQLTestDB(t *testing.T) (driver.DB, func()) {
	t.Skip("MySQL driver not available. Install with: go get github.com/go-sql-driver/mysql and run tests with -tags=mysql")
	return nil, nil
}
