//go:build sqlite

package driver

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestApplyPoolSettings(t *testing.T) {
	// Use SQLite in-memory for testing the helper
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Test with various values
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
			// We can't easily inspect the values back from sql.DB directly
			// but we can ensure it doesn't panic and we can call it.
			ApplyPoolSettings(db, tt.maxConns, tt.minConns, tt.maxLifetimeMin)
		})
	}
}
