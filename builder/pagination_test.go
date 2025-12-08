package builder

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// TestQuery_Take tests the Take() method for limiting results
func TestQuery_Take(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
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
				insertSQL = "INSERT INTO pagination_test (name) VALUES ($1)"
			default:
				insertSQL = "INSERT INTO pagination_test (name) VALUES (?)"
			}
			for i := 1; i <= 10; i++ {
				_, err := sqlDB.ExecContext(ctx, insertSQL, fmt.Sprintf("User %d", i))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
				}
			}

			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			// Test Take() with fluent API
			query := NewQuery(db, "pagination_test", []string{"id", "name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.SetModelType(reflect.TypeOf(TestRecord{}))

			var results []TestRecord
			err = query.Take(5).Find(ctx, &results)
			if err != nil {
				t.Fatalf("Find with Take failed: %v", err)
			}

			if len(results) != 5 {
				t.Errorf("Expected 5 results, got %d", len(results))
			}
		})
	}
}

// TestQuery_Skip tests the Skip() method for skipping results
func TestQuery_Skip(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
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
				insertSQL = "INSERT INTO pagination_test (name) VALUES ($1)"
			default:
				insertSQL = "INSERT INTO pagination_test (name) VALUES (?)"
			}
			for i := 1; i <= 10; i++ {
				_, err := sqlDB.ExecContext(ctx, insertSQL, fmt.Sprintf("User %d", i))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
				}
			}

			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			// Test Skip() with fluent API
			query := NewQuery(db, "pagination_test", []string{"id", "name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.SetModelType(reflect.TypeOf(TestRecord{}))

			// Get all records without skip
			var allResults []TestRecord
			err = query.Find(ctx, &allResults)
			if err != nil {
				t.Fatalf("Find failed: %v", err)
			}

			// Get results with Skip(5) - should skip first 5
			var skippedResults []TestRecord
			err = query.Skip(5).Find(ctx, &skippedResults)
			if err != nil {
				t.Fatalf("Find with Skip failed: %v", err)
			}

			if len(skippedResults) != 5 {
				t.Errorf("Expected 5 results after skipping 5, got %d", len(skippedResults))
			}

			// Verify that skipped results start from the 6th record
			if len(allResults) >= 6 && skippedResults[0].ID != allResults[5].ID {
				t.Errorf("Expected first skipped result ID to be %d, got %d", allResults[5].ID, skippedResults[0].ID)
			}
		})
	}
}

// TestQuery_SkipAndTake tests pagination with both Skip() and Take()
func TestQuery_SkipAndTake(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
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
				insertSQL = "INSERT INTO pagination_test (name) VALUES ($1)"
			default:
				insertSQL = "INSERT INTO pagination_test (name) VALUES (?)"
			}
			for i := 1; i <= 10; i++ {
				_, err := sqlDB.ExecContext(ctx, insertSQL, fmt.Sprintf("User %d", i))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
				}
			}

			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			// Test pagination: page 2, pageSize 3
			// Should skip 3 records (page 1) and take 3 records (page 2)
			query := NewQuery(db, "pagination_test", []string{"id", "name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.SetModelType(reflect.TypeOf(TestRecord{}))

			page := 2
			pageSize := 3
			skip := (page - 1) * pageSize

			var results []TestRecord
			err = query.Skip(skip).Take(pageSize).Find(ctx, &results)
			if err != nil {
				t.Fatalf("Find with Skip and Take failed: %v", err)
			}

			if len(results) != pageSize {
				t.Errorf("Expected %d results, got %d", pageSize, len(results))
			}
		})
	}
}

// TestQueryOptions_TakeAndSkip tests Take and Skip in QueryOptions (TableQueryBuilder)
func TestQueryOptions_TakeAndSkip(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS pagination_test (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
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
				insertSQL = "INSERT INTO pagination_test (name) VALUES ($1)"
			default:
				insertSQL = "INSERT INTO pagination_test (name) VALUES (?)"
			}
			for i := 1; i <= 10; i++ {
				_, err := sqlDB.ExecContext(ctx, insertSQL, fmt.Sprintf("User %d", i))
				if err != nil {
					t.Fatalf("failed to insert test data: %v", err)
				}
			}

			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			builder := NewTableQueryBuilder(db, "pagination_test", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			// Test Take only
			take := 5
			results, err := builder.FindMany(ctx, QueryOptions{
				Take: Ptr(take),
			})
			if err != nil {
				t.Fatalf("FindMany with Take failed: %v", err)
			}

			resultsSlice, ok := results.([]TestRecord)
			if !ok {
				t.Fatal("FindMany returned wrong type")
			}
			if len(resultsSlice) != take {
				t.Errorf("Expected %d results with Take(%d), got %d", take, take, len(resultsSlice))
			}

			// Test Skip only
			skip := 3
			results, err = builder.FindMany(ctx, QueryOptions{
				Skip: Ptr(skip),
			})
			if err != nil {
				t.Fatalf("FindMany with Skip failed: %v", err)
			}

			resultsSlice, ok = results.([]TestRecord)
			if !ok {
				t.Fatal("FindMany returned wrong type")
			}
			if len(resultsSlice) != 7 { // 10 total - 3 skipped = 7
				t.Errorf("Expected 7 results with Skip(%d), got %d", skip, len(resultsSlice))
			}

			// Test Skip and Take together
			results, err = builder.FindMany(ctx, QueryOptions{
				Skip: Ptr(2),
				Take: Ptr(3),
			})
			if err != nil {
				t.Fatalf("FindMany with Skip and Take failed: %v", err)
			}

			resultsSlice, ok = results.([]TestRecord)
			if !ok {
				t.Fatal("FindMany returned wrong type")
			}
			if len(resultsSlice) != 3 {
				t.Errorf("Expected 3 results with Skip(2) and Take(3), got %d", len(resultsSlice))
			}
		})
	}
}
