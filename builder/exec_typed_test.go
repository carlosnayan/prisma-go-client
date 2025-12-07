package builder

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// TestQuery_ScanFirst tests ScanFirst with custom DTOs
func TestQuery_ScanFirst(t *testing.T) {
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

			// Create test table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS tenants (
						id_tenant SERIAL PRIMARY KEY,
						tenant_name VARCHAR(255) NOT NULL,
						metadata JSONB
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS tenants (
						id_tenant INT AUTO_INCREMENT PRIMARY KEY,
						tenant_name VARCHAR(255) NOT NULL,
						metadata JSON
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS tenants (
						id_tenant INTEGER PRIMARY KEY AUTOINCREMENT,
						tenant_name TEXT NOT NULL,
						metadata TEXT
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = `INSERT INTO tenants (tenant_name, metadata) VALUES ($1, $2) RETURNING id_tenant`
			case "mysql":
				insertSQL = `INSERT INTO tenants (tenant_name, metadata) VALUES (?, ?)`
			case "sqlite":
				insertSQL = `INSERT INTO tenants (tenant_name, metadata) VALUES (?, ?)`
			}

			metadataJSON := `{"key": "value", "number": 42}`
			var tenantID int
			if provider == "postgresql" {
				err = sqlDB.QueryRowContext(ctx, insertSQL, "Test Tenant", metadataJSON).Scan(&tenantID)
			} else {
				result, err := sqlDB.ExecContext(ctx, insertSQL, "Test Tenant", metadataJSON)
				if err != nil {
					t.Fatalf("failed to insert: %v", err)
				}
				id, err := result.LastInsertId()
				if err != nil {
					t.Fatalf("failed to get last insert id: %v", err)
				}
				tenantID = int(id)
			}
			if err != nil {
				t.Fatalf("failed to insert: %v", err)
			}

			// Define custom DTO with JSON tags
			type TenantsDTO struct {
				IdTenant   int             `json:"id_tenant" db:"id_tenant"`
				TenantName string          `json:"tenant_name" db:"tenant_name"`
				Metadata   json.RawMessage `json:"metadata" db:"metadata"`
			}

			// Create Query builder
			query := NewQuery(db, "tenants", []string{"id_tenant", "tenant_name", "metadata"})
			query.SetDialect(dialect.GetDialect(provider))
			query.Select("id_tenant", "tenant_name", "metadata")
			query.Where("id_tenant = ?", tenantID)

			// Test ScanFirst with custom DTO
			result := reflect.New(reflect.TypeOf(TenantsDTO{})).Interface()
			err = query.ScanFirst(ctx, result, reflect.TypeOf(TenantsDTO{}))
			if err != nil {
				t.Fatalf("ScanFirst failed: %v", err)
			}

			dto, ok := result.(*TenantsDTO)
			if !ok {
				t.Fatalf("ScanFirst returned wrong type: %T", result)
			}

			if dto.IdTenant != tenantID {
				t.Errorf("Expected IdTenant %d, got %d", tenantID, dto.IdTenant)
			}
			if dto.TenantName != "Test Tenant" {
				t.Errorf("Expected TenantName 'Test Tenant', got '%s'", dto.TenantName)
			}
			if len(dto.Metadata) == 0 {
				t.Error("Expected Metadata to be populated")
			}

			// Verify metadata content
			var metadata map[string]interface{}
			if err := json.Unmarshal(dto.Metadata, &metadata); err != nil {
				t.Fatalf("Failed to unmarshal metadata: %v", err)
			}
			if metadata["key"] != "value" {
				t.Errorf("Expected metadata key 'value', got '%v'", metadata["key"])
			}
		})
	}
}

// TestQuery_ScanFirst_WithDBTags tests ScanFirst with DB tags
func TestQuery_ScanFirst_WithDBTags(t *testing.T) {
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

			// Create test table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						user_id SERIAL PRIMARY KEY,
						user_email VARCHAR(255) NOT NULL,
						user_name VARCHAR(255)
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						user_id INT AUTO_INCREMENT PRIMARY KEY,
						user_email VARCHAR(255) NOT NULL,
						user_name VARCHAR(255)
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						user_id INTEGER PRIMARY KEY AUTOINCREMENT,
						user_email TEXT NOT NULL,
						user_name TEXT
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = `INSERT INTO users (user_email, user_name) VALUES ($1, $2) RETURNING user_id`
			case "mysql", "sqlite":
				insertSQL = `INSERT INTO users (user_email, user_name) VALUES (?, ?)`
			}

			var userID int
			if provider == "postgresql" {
				err = sqlDB.QueryRowContext(ctx, insertSQL, "test@example.com", "Test User").Scan(&userID)
			} else {
				result, err := sqlDB.ExecContext(ctx, insertSQL, "test@example.com", "Test User")
				if err != nil {
					t.Fatalf("failed to insert: %v", err)
				}
				id, err := result.LastInsertId()
				if err != nil {
					t.Fatalf("failed to get last insert id: %v", err)
				}
				userID = int(id)
			}
			if err != nil {
				t.Fatalf("failed to insert: %v", err)
			}

			// Define custom DTO with DB tags only
			type UserDTO struct {
				ID    int    `db:"user_id"`
				Email string `db:"user_email"`
				Name  string `db:"user_name"`
			}

			// Create Query builder
			query := NewQuery(db, "users", []string{"user_id", "user_email", "user_name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.Select("user_id", "user_email", "user_name")
			query.Where("user_id = ?", userID)

			// Test ScanFirst with custom DTO
			result := reflect.New(reflect.TypeOf(UserDTO{})).Interface()
			err = query.ScanFirst(ctx, result, reflect.TypeOf(UserDTO{}))
			if err != nil {
				t.Fatalf("ScanFirst failed: %v", err)
			}

			dto, ok := result.(*UserDTO)
			if !ok {
				t.Fatalf("ScanFirst returned wrong type: %T", result)
			}

			if dto.ID != userID {
				t.Errorf("Expected ID %d, got %d", userID, dto.ID)
			}
			if dto.Email != "test@example.com" {
				t.Errorf("Expected Email 'test@example.com', got '%s'", dto.Email)
			}
			if dto.Name != "Test User" {
				t.Errorf("Expected Name 'Test User', got '%s'", dto.Name)
			}
		})
	}
}

// TestQuery_ScanFind tests ScanFind with custom DTOs
func TestQuery_ScanFind(t *testing.T) {
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

			// Create test table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS products (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						price DECIMAL(10,2)
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS products (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						price DECIMAL(10,2)
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS products (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL,
						price REAL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Insert multiple test records
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = `INSERT INTO products (name, price) VALUES ($1, $2) RETURNING id`
			case "mysql", "sqlite":
				insertSQL = `INSERT INTO products (name, price) VALUES (?, ?)`
			}

			products := []struct {
				name  string
				price float64
			}{
				{"Product 1", 10.50},
				{"Product 2", 20.75},
				{"Product 3", 30.00},
			}

			for _, p := range products {
				if provider == "postgresql" {
					var id int
					err = sqlDB.QueryRowContext(ctx, insertSQL, p.name, p.price).Scan(&id)
				} else {
					_, err = sqlDB.ExecContext(ctx, insertSQL, p.name, p.price)
				}
				if err != nil {
					t.Fatalf("failed to insert: %v", err)
				}
			}

			// Define custom DTO
			type ProductDTO struct {
				ID    int     `json:"id" db:"id"`
				Name  string  `json:"name" db:"name"`
				Price float64 `json:"price" db:"price"`
			}

			// Create Query builder
			query := NewQuery(db, "products", []string{"id", "name", "price"})
			query.SetDialect(dialect.GetDialect(provider))
			query.Select("id", "name", "price")
			query.Order("id ASC")

			// Test ScanFind with custom DTO
			results := make([]ProductDTO, 0)
			err = query.ScanFind(ctx, &results, reflect.TypeOf(ProductDTO{}))
			if err != nil {
				t.Fatalf("ScanFind failed: %v", err)
			}

			if len(results) != 3 {
				t.Fatalf("Expected 3 results, got %d", len(results))
			}

			// Verify results
			for i, result := range results {
				if result.Name != products[i].name {
					t.Errorf("Result %d: Expected Name '%s', got '%s'", i, products[i].name, result.Name)
				}
				if result.Price != products[i].price {
					t.Errorf("Result %d: Expected Price %.2f, got %.2f", i, products[i].price, result.Price)
				}
			}
		})
	}
}

// TestQuery_ScanFirst_WithMissingFields tests ScanFirst with DTO that has missing fields
func TestQuery_ScanFirst_WithMissingFields(t *testing.T) {
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

			// Create test table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255),
						age INT
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255),
						age INT
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL,
						name TEXT,
						age INTEGER
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = `INSERT INTO users (email, name, age) VALUES ($1, $2, $3) RETURNING id`
			case "mysql", "sqlite":
				insertSQL = `INSERT INTO users (email, name, age) VALUES (?, ?, ?)`
			}

			var userID int
			if provider == "postgresql" {
				err = sqlDB.QueryRowContext(ctx, insertSQL, "test@example.com", "Test User", 25).Scan(&userID)
			} else {
				result, err := sqlDB.ExecContext(ctx, insertSQL, "test@example.com", "Test User", 25)
				if err != nil {
					t.Fatalf("failed to insert: %v", err)
				}
				id, err := result.LastInsertId()
				if err != nil {
					t.Fatalf("failed to get last insert id: %v", err)
				}
				userID = int(id)
			}
			if err != nil {
				t.Fatalf("failed to insert: %v", err)
			}

			// Define custom DTO with only some fields (missing 'age')
			type UserDTO struct {
				ID    int    `json:"id" db:"id"`
				Email string `json:"email" db:"email"`
				Name  string `json:"name" db:"name"`
				// Age field is missing - should be ignored
			}

			// Create Query builder - select all fields including age
			query := NewQuery(db, "users", []string{"id", "email", "name", "age"})
			query.SetDialect(dialect.GetDialect(provider))
			query.Select("id", "email", "name", "age")
			query.Where("id = ?", userID)

			// Test ScanFirst - should work even though DTO doesn't have 'age' field
			result := reflect.New(reflect.TypeOf(UserDTO{})).Interface()
			err = query.ScanFirst(ctx, result, reflect.TypeOf(UserDTO{}))
			if err != nil {
				t.Fatalf("ScanFirst failed: %v", err)
			}

			dto, ok := result.(*UserDTO)
			if !ok {
				t.Fatalf("ScanFirst returned wrong type: %T", result)
			}

			if dto.ID != userID {
				t.Errorf("Expected ID %d, got %d", userID, dto.ID)
			}
			if dto.Email != "test@example.com" {
				t.Errorf("Expected Email 'test@example.com', got '%s'", dto.Email)
			}
			if dto.Name != "Test User" {
				t.Errorf("Expected Name 'Test User', got '%s'", dto.Name)
			}
		})
	}
}

// TestQuery_ScanFirst_WithSelectFields tests ScanFirst with Select() limiting fields
func TestQuery_ScanFirst_WithSelectFields(t *testing.T) {
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

			// Create test table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255)
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255)
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL,
						name TEXT
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = `INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id`
			case "mysql", "sqlite":
				insertSQL = `INSERT INTO users (email, name) VALUES (?, ?)`
			}

			var userID int
			if provider == "postgresql" {
				err = sqlDB.QueryRowContext(ctx, insertSQL, "test@example.com", "Test User").Scan(&userID)
			} else {
				result, err := sqlDB.ExecContext(ctx, insertSQL, "test@example.com", "Test User")
				if err != nil {
					t.Fatalf("failed to insert: %v", err)
				}
				id, err := result.LastInsertId()
				if err != nil {
					t.Fatalf("failed to get last insert id: %v", err)
				}
				userID = int(id)
			}
			if err != nil {
				t.Fatalf("failed to insert: %v", err)
			}

			// Define custom DTO
			type UserDTO struct {
				ID    int    `json:"id" db:"id"`
				Email string `json:"email" db:"email"`
				Name  string `json:"name" db:"name"`
			}

			// Create Query builder with Select limiting to only email and name
			query := NewQuery(db, "users", []string{"id", "email", "name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.Select("email", "name") // Only select email and name, not id
			query.Where("id = ?", userID)

			// Test ScanFirst - should work even though we're not selecting 'id'
			result := reflect.New(reflect.TypeOf(UserDTO{})).Interface()
			err = query.ScanFirst(ctx, result, reflect.TypeOf(UserDTO{}))
			if err != nil {
				t.Fatalf("ScanFirst failed: %v", err)
			}

			dto, ok := result.(*UserDTO)
			if !ok {
				t.Fatalf("ScanFirst returned wrong type: %T", result)
			}

			// ID should be zero value since we didn't select it
			if dto.ID != 0 {
				t.Errorf("Expected ID 0 (not selected), got %d", dto.ID)
			}
			if dto.Email != "test@example.com" {
				t.Errorf("Expected Email 'test@example.com', got '%s'", dto.Email)
			}
			if dto.Name != "Test User" {
				t.Errorf("Expected Name 'Test User', got '%s'", dto.Name)
			}
		})
	}
}

// TestQuery_ScanFind_EmptyResult tests ScanFind with no results
func TestQuery_ScanFind_EmptyResult(t *testing.T) {
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

			// Create test table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS products (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS products (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS products (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Don't insert any data - table is empty

			// Define custom DTO
			type ProductDTO struct {
				ID   int    `json:"id" db:"id"`
				Name string `json:"name" db:"name"`
			}

			// Create Query builder
			query := NewQuery(db, "products", []string{"id", "name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.Select("id", "name")

			// Test ScanFind with empty result
			results := make([]ProductDTO, 0)
			err = query.ScanFind(ctx, &results, reflect.TypeOf(ProductDTO{}))
			if err != nil {
				t.Fatalf("ScanFind failed: %v", err)
			}

			if len(results) != 0 {
				t.Fatalf("Expected 0 results, got %d", len(results))
			}
		})
	}
}
