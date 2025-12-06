//go:build !pgx

package testing

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
)

// SetupPostgreSQLTestDB creates a PostgreSQL test database
// This stub version skips the test if PostgreSQL driver is not available
func SetupPostgreSQLTestDB(t *testing.T) (driver.DB, func()) {
	t.Skip("PostgreSQL driver not available. Install with: go get github.com/jackc/pgx/v5/stdlib and run tests with -tags=pgx")
	return nil, nil
}
