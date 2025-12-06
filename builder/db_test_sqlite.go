//go:build sqlite

package builder

import (
	"context"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// TestBuilder_SQLite tests query builder with SQLite
func TestBuilder_SQLite(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t, "sqlite")
	defer cleanup()

	// Create test table
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB()")
	}

	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Test basic operations
	builder := NewTableQueryBuilder(db, "users", []string{"id", "email", "name", "created_at", "updated_at"})
	builder.SetDialect(dialect.GetDialect("sqlite"))
	builder.SetPrimaryKey("id")

	type User struct {
		ID        int       `json:"id"`
		Email     string    `json:"email"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	user := User{
		Email: "test@example.com",
		Name:  "Test User",
	}

	created, err := builder.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	createdUser, ok := created.(User)
	if !ok {
		t.Fatal("Create returned wrong type")
	}
	if createdUser.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", createdUser.Email)
	}
}
