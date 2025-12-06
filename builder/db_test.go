package builder

import (
	"context"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// TestBuilder_PostgreSQL tests query builder with PostgreSQL
func TestBuilder_PostgreSQL(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t, "postgresql")
	defer cleanup()

	// Create test table
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB()")
	}

	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Test Create
	builder := NewTableQueryBuilder(db, "users", []string{"id", "email", "name", "created_at", "updated_at"})
	builder.SetDialect(dialect.GetDialect("postgresql"))
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

	// Test FindFirst
	found, err := builder.FindFirst(ctx, Where{"email": "test@example.com"})
	if err != nil {
		t.Fatalf("FindFirst failed: %v", err)
	}

	foundUser, ok := found.(User)
	if !ok {
		t.Fatal("FindFirst returned wrong type")
	}
	if foundUser.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", foundUser.Email)
	}

	// Test Update
	updateData := User{
		Name: "Updated Name",
	}
	updated, err := builder.Update(ctx, createdUser.ID, updateData)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updatedUser, ok := updated.(User)
	if !ok {
		t.Fatal("Update returned wrong type")
	}
	if updatedUser.Name != "Updated Name" {
		t.Errorf("Expected name Updated Name, got %s", updatedUser.Name)
	}

	// Test Count
	count, err := builder.Count(ctx, Where{"email": "test@example.com"})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test Delete
	err = builder.Delete(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	count, err = builder.Count(ctx, Where{"email": "test@example.com"})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after delete, got %d", count)
	}
}

// TestBuilder_MySQL tests query builder with MySQL
func TestBuilder_MySQL(t *testing.T) {
	testutil.SkipIfNoDatabase(t, "mysql")
	db, cleanup := testutil.SetupTestDB(t, "mysql")
	defer cleanup()

	// Create test table
	sqlDB := db.SQLDB()
	if sqlDB == nil {
		t.Fatal("database does not support SQLDB()")
	}

	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Test basic operations (similar to PostgreSQL test)
	builder := NewTableQueryBuilder(db, "users", []string{"id", "email", "name", "created_at", "updated_at"})
	builder.SetDialect(dialect.GetDialect("mysql"))
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

// TestBuilder_AllProviders runs the same test across all providers
func TestBuilder_AllProviders(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table based on provider
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_table (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_table (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_table (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Test basic CRUD
			builder := NewTableQueryBuilder(db, "test_table", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")

			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			record := TestRecord{Name: "Test"}

			// Create
			created, err := builder.Create(ctx, record)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			createdRecord, ok := created.(TestRecord)
			if !ok {
				t.Fatal("Create returned wrong type")
			}

			// Read
			found, err := builder.FindFirst(ctx, Where{"id": createdRecord.ID})
			if err != nil {
				t.Fatalf("FindFirst failed: %v", err)
			}

			foundRecord, ok := found.(TestRecord)
			if !ok {
				t.Fatal("FindFirst returned wrong type")
			}
			if foundRecord.Name != "Test" {
				t.Errorf("Expected name Test, got %s", foundRecord.Name)
			}

			// Count
			count, err := builder.Count(ctx, Where{})
			if err != nil {
				t.Fatalf("Count failed: %v", err)
			}
			if count < 1 {
				t.Errorf("Expected count >= 1, got %d", count)
			}
		})
	}
}
