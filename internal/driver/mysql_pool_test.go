//go:build mysql

package driver

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestApplyPoolSettingsMySQL(t *testing.T) {
	// We use a fake DSN as sql.Open doesn't usually connect immediately
	// This is enough to get a *sql.DB with the mysql driver registered
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
	if err != nil {
		t.Fatalf("failed to open mysql: %v", err)
	}
	defer db.Close()

	// Test with various values to ensures no panics and logic works for mysql driver
	tests := []struct {
		name           string
		maxConns       int
		minConns       int
		maxLifetimeMin int
	}{
		{"all zero", 0, 0, 0},
		{"only max", 10, 0, 0},
		{"only min", 0, 5, 0},
		{"only lifetime", 0, 0, 30},
		{"all values", 50, 20, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure it doesn't panic when applying to a mysql *sql.DB
			ApplyPoolSettings(db, tt.maxConns, tt.minConns, tt.maxLifetimeMin)
		})
	}
}
