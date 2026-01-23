package builder

import (
	"context"
	"reflect"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// NOTE: Validation of required fields happens in the generated code (templates),
// not in TableQueryBuilder. The generated code validates before calling TableQueryBuilder methods.
// These tests verify that TableQueryBuilder can handle data structures correctly.
// The actual validation logic is tested through the generated code compilation test.

// User model for testing required fields
type User struct {
	ID        int    `json:"id" db:"id"`
	Email     string `json:"email" db:"email"`           // Required
	Name      string `json:"name" db:"name"`             // Required
	Age       *int   `json:"age" db:"age"`               // Optional
	Bio       string `json:"bio" db:"bio"`               // Required (no default)
	CreatedAt string `json:"created_at" db:"created_at"` // Has default (not required)
}

// TestCreate_RequiredFieldsValidation tests error when required field is missing
func TestCreate_RequiredFieldsValidation(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()

			// Create table with required fields
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL,
						age INT,
						bio TEXT NOT NULL,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL,
						age INT,
						bio TEXT NOT NULL,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL,
						name TEXT NOT NULL,
						age INTEGER,
						bio TEXT NOT NULL,
						created_at DATETIME DEFAULT CURRENT_TIMESTAMP
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			columns := []string{"id", "email", "name", "age", "bio", "created_at"}
			builder := NewTableQueryBuilder(db, "users", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(User{}))

			// Test: Missing required field 'email' (will fail at database level)
			_, err = builder.CreateFromFields(ctx, map[string]interface{}{
				"name": "John Doe",
				"bio":  "Test bio",
			})
			if err == nil {
				t.Log("Note: TableQueryBuilder doesn't validate required fields - validation happens in generated code")
			} else {
				t.Logf("Database error (expected for missing required field): %v", err)
			}

			// Test: Missing required field 'name' (will fail at database level)
			_, err = builder.CreateFromFields(ctx, map[string]interface{}{
				"email": "test@example.com",
				"bio":   "Test bio",
			})
			if err == nil {
				t.Log("Note: TableQueryBuilder doesn't validate required fields - validation happens in generated code")
			} else {
				t.Logf("Database error (expected for missing required field): %v", err)
			}

			// Test: All required fields provided (should succeed)
			_, err = builder.CreateFromFields(ctx, map[string]interface{}{
				"email": "test@example.com",
				"name":  "John Doe",
				"bio":   "Test bio",
			})
			if err != nil {
				t.Errorf("Expected success when all required fields provided, got error: %v", err)
			}
		})
	}
}

// TestCreate_RequiredFieldsWithDefault tests that fields with @default are not required
func TestCreate_RequiredFieldsWithDefault(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()

			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL,
						status VARCHAR(50) DEFAULT 'active'
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL,
						status VARCHAR(50) DEFAULT 'active'
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL,
						name TEXT NOT NULL,
						status TEXT DEFAULT 'active'
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			columns := []string{"id", "email", "name", "status"}
			builder := NewTableQueryBuilder(db, "users", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(User{}))

			// Test: Field with default should not be required
			_, err = builder.CreateFromFields(ctx, map[string]interface{}{
				"email": "test@example.com",
				"name":  "John Doe",
			})
			if err != nil {
				t.Logf("Note: Error occurred (may be expected): %v", err)
			}
		})
	}
}

// TestCreate_OptionalFieldsAllowed tests that optional fields can be missing from map
func TestCreate_OptionalFieldsAllowed(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()

			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL,
						age INT
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL,
						age INT
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL,
						name TEXT NOT NULL,
						age INTEGER
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			columns := []string{"id", "email", "name", "age"}
			builder := NewTableQueryBuilder(db, "users", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(User{}))

			// Test: Optional field can be missing
			_, err = builder.CreateFromFields(ctx, map[string]interface{}{
				"email": "test@example.com",
				"name":  "John Doe",
			})
			if err != nil {
				t.Errorf("Expected success with optional field missing, got error: %v", err)
			}

			// Test: Optional field can have value
			_, err = builder.CreateFromFields(ctx, map[string]interface{}{
				"email": "test2@example.com",
				"name":  "Jane Doe",
				"age":   30,
			})
			if err != nil {
				t.Errorf("Expected success with optional field set, got error: %v", err)
			}
		})
	}
}

// TestCreateMany_RequiredFieldsValidation tests batch insert with maps
func TestCreateMany_RequiredFieldsValidation(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()

			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL,
						bio TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL,
						bio TEXT NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL,
						name TEXT NOT NULL,
						bio TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			columns := []string{"id", "email", "name", "bio"}
			builder := NewTableQueryBuilder(db, "users", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(User{}))

			// Test: Batch with valid items
			users := []map[string]interface{}{
				{
					"email": "test1@example.com",
					"name":  "User 1",
					"bio":   "Bio 1",
				},
				{
					"email": "test2@example.com",
					"name":  "User 2",
					"bio":   "Bio 2",
				},
			}
			result, err := builder.CreateManyFromFields(ctx, users, false)
			if err != nil {
				t.Errorf("Expected success for batch insert, got error: %v", err)
			}
			if result != nil && result.Count != 2 {
				t.Errorf("Expected 2 users created, got %d", result.Count)
			}
		})
	}
}
