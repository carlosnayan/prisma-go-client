package builder

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// TestUpdate_WithWhere tests Update() with Where conditions (NEW)
func TestUpdate_WithWhere(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id TEXT PRIMARY KEY,
						email TEXT UNIQUE NOT NULL,
						name TEXT NOT NULL,
						bio TEXT
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id VARCHAR(191) PRIMARY KEY,
						email VARCHAR(191) UNIQUE NOT NULL,
						name VARCHAR(255) NOT NULL,
						bio TEXT
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id TEXT PRIMARY KEY,
						email TEXT UNIQUE NOT NULL,
						name TEXT NOT NULL,
						bio TEXT
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO authors (id, email, name) VALUES ($1, $2, $3), ($4, $5, $6)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO authors (id, email, name) VALUES (?, ?, ?), (?, ?, ?)"
			}
			_, err = db.SQLDB().Exec(insertSQL,
				"1", "john@example.com", "John Doe",
				"2", "jane@example.com", "Jane Doe")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test Update with Where
			q := NewQuery(db, "authors", []string{"id", "email", "name", "bio"})
			q.SetDialect(dialect.GetDialect(provider))
			q.SetPrimaryKey("id")

			// Build Update query
			q.Where("email = ?", "john@example.com")
			updateData := map[string]interface{}{
				"name": "Jonathan Doe",
				"bio":  "Updated bio",
			}

			err = q.Updates(ctx, updateData)
			if err != nil {
				t.Fatalf("Update failed: %v", err)
			}

			// Verify update
			var name, bio string
			var verifySQL string
			switch provider {
			case "postgresql":
				verifySQL = "SELECT name, bio FROM authors WHERE email = $1"
			case "mysql", "sqlite":
				verifySQL = "SELECT name, bio FROM authors WHERE email = ?"
			}
			err = db.SQLDB().QueryRow(verifySQL, "john@example.com").Scan(&name, &bio)
			if err != nil {
				t.Fatalf("Failed to verify update: %v", err)
			}

			if name != "Jonathan Doe" {
				t.Errorf("Expected name 'Jonathan Doe', got '%s'", name)
			}
			if bio != "Updated bio" {
				t.Errorf("Expected bio 'Updated bio', got '%s'", bio)
			}

			// Verify SQL generation includes WHERE clause
			query, _ := q.buildUpdatesQuery(updateData)
			if !strings.Contains(query, "WHERE") {
				t.Errorf("Expected UPDATE query to contain WHERE clause, got: %s", query)
			}
		})
	}
}

// TestFindMany_Basic tests basic FindMany functionality
func TestFindMany_Basic(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id SERIAL PRIMARY KEY,
						email TEXT NOT NULL,
						name TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL,
						name TEXT NOT NULL
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO authors (email, name) VALUES ($1, $2), ($3, $4), ($5, $6)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO authors (email, name) VALUES (?, ?), (?, ?), (?, ?)"
			}
			_, err = db.SQLDB().Exec(insertSQL,
				"john@example.com", "John Doe",
				"jane@example.com", "Jane Doe",
				"bob@example.com", "Bob Smith")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test FindMany
			type Author struct {
				ID    int
				Email string
				Name  string
			}
			q := NewQuery(db, "authors", []string{"id", "email", "name"})
			q.SetDialect(dialect.GetDialect(provider))
			q.SetModelType(reflect.TypeOf(Author{}))

			var authors []Author
			err = q.Find(ctx, &authors)
			if err != nil {
				t.Fatalf("FindMany failed: %v", err)
			}

			if len(authors) != 3 {
				t.Errorf("Expected 3 authors, got %d", len(authors))
			}

			// Verify SQL generation
			query, _ := q.buildSelectQuery(false)
			if !strings.Contains(query, "SELECT") {
				t.Errorf("Expected SELECT query, got: %s", query)
			}
			if !strings.Contains(query, "FROM") {
				t.Errorf("Expected FROM clause, got: %s", query)
			}
		})
	}
}

// TestCount_Basic tests Count functionality
func TestCount_Basic(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id SERIAL PRIMARY KEY,
						email TEXT NOT NULL,
						active BOOLEAN DEFAULT true
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						active BOOLEAN DEFAULT 1
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL,
						active INTEGER DEFAULT 1
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO authors (email, active) VALUES ($1, $2), ($3, $4), ($5, $6)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO authors (email, active) VALUES (?, ?), (?, ?), (?, ?)"
			}
			_, err = db.SQLDB().Exec(insertSQL,
				"john@example.com", true,
				"jane@example.com", true,
				"bob@example.com", false)
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test Count all
			q := NewQuery(db, "authors", []string{"id", "email", "active"})
			q.SetDialect(dialect.GetDialect(provider))

			count, err := q.Count(ctx)
			if err != nil {
				t.Fatalf("Count failed: %v", err)
			}

			if count != 3 {
				t.Errorf("Expected count 3, got %d", count)
			}

			// Test Count with WHERE
			q2 := NewQuery(db, "authors", []string{"id", "email", "active"})
			q2.SetDialect(dialect.GetDialect(provider))
			q2.Where("active = ?", true)

			count, err = q2.Count(ctx)
			if err != nil {
				t.Fatalf("Count with WHERE failed: %v", err)
			}

			if count != 2 {
				t.Errorf("Expected count 2, got %d", count)
			}

			// Verify SQL generation
			query, _ := q2.buildCountQuery()
			if !strings.Contains(query, "COUNT(*)") {
				t.Errorf("Expected COUNT(*) in query, got: %s", query)
			}
			if !strings.Contains(query, "WHERE") {
				t.Errorf("Expected WHERE clause in query, got: %s", query)
			}
		})
	}
}

// TestDelete_WithWhere tests Delete with WHERE conditions
func TestDelete_WithWhere(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id SERIAL PRIMARY KEY,
						email TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INT AUTO_INCREMENT PRIMARY KEY,
						email VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						email TEXT NOT NULL
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO authors (email) VALUES ($1), ($2), ($3)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO authors (email) VALUES (?), (?), (?)"
			}
			_, err = db.SQLDB().Exec(insertSQL,
				"john@example.com",
				"jane@example.com",
				"bob@example.com")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test Delete with WHERE
			q := NewQuery(db, "authors", []string{"id", "email"})
			q.SetDialect(dialect.GetDialect(provider))
			q.Where("email = ?", "john@example.com")

			err = q.Delete(ctx, nil)
			if err != nil {
				t.Fatalf("Delete failed: %v", err)
			}

			// Verify deletion
			var count int
			err = db.SQLDB().QueryRow("SELECT COUNT(*) FROM authors").Scan(&count)
			if err != nil {
				t.Fatalf("Failed to verify deletion: %v", err)
			}

			if count != 2 {
				t.Errorf("Expected 2 authors remaining, got %d", count)
			}

			// Verify SQL generation
			query, _ := q.buildDeleteQuery()
			if !strings.Contains(query, "DELETE FROM") {
				t.Errorf("Expected DELETE FROM query, got: %s", query)
			}
			if !strings.Contains(query, "WHERE") {
				t.Errorf("Expected WHERE clause, got: %s", query)
			}
		})
	}
}

// TestUpdateOneID_Basic tests UpdateOneID functionality
func TestUpdateOneID_Basic(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id TEXT PRIMARY KEY,
						email TEXT NOT NULL,
						name TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id VARCHAR(191) PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id TEXT PRIMARY KEY,
						email TEXT NOT NULL,
						name TEXT NOT NULL
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO authors (id, email, name) VALUES ($1, $2, $3)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO authors (id, email, name) VALUES (?, ?, ?)"
			}
			_, err = db.SQLDB().Exec(insertSQL, "author-1", "john@example.com", "John Doe")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test UpdateOneID
			q := NewQuery(db, "authors", []string{"id", "email", "name"})
			q.SetDialect(dialect.GetDialect(provider))
			q.SetPrimaryKey("id")
			q.Where("id = ?", "author-1")

			updateData := map[string]interface{}{
				"name": "Jonathan Doe",
			}
			err = q.Updates(ctx, updateData)
			if err != nil {
				t.Fatalf("UpdateOneID failed: %v", err)
			}

			// Verify update
			var name string
			var verifySQL string
			switch provider {
			case "postgresql":
				verifySQL = "SELECT name FROM authors WHERE id = $1"
			case "mysql", "sqlite":
				verifySQL = "SELECT name FROM authors WHERE id = ?"
			}
			err = db.SQLDB().QueryRow(verifySQL, "author-1").Scan(&name)
			if err != nil {
				t.Fatalf("Failed to verify update: %v", err)
			}

			if name != "Jonathan Doe" {
				t.Errorf("Expected name 'Jonathan Doe', got '%s'", name)
			}
		})
	}
}

// TestLimitSkip_Pagination tests Limit and Skip for pagination
func TestLimitSkip_Pagination(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id SERIAL PRIMARY KEY,
						name TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert 10 records
			for i := 1; i <= 10; i++ {
				var insertSQL string
				switch provider {
				case "postgresql":
					insertSQL = "INSERT INTO authors (name) VALUES ($1)"
				case "mysql", "sqlite":
					insertSQL = "INSERT INTO authors (name) VALUES (?)"
				}
				_, err = db.SQLDB().Exec(insertSQL, "Author "+string(rune(i)))
				if err != nil {
					t.Fatalf("Failed to insert test data: %v", err)
				}
			}

			// Test Limit
			type Author struct {
				ID   int
				Name string
			}
			q := NewQuery(db, "authors", []string{"id", "name"})
			q.SetDialect(dialect.GetDialect(provider))
			q.SetModelType(reflect.TypeOf(Author{}))
			q.Take(5)

			var authors []Author
			err = q.Find(ctx, &authors)
			if err != nil {
				t.Fatalf("FindMany with Limit failed: %v", err)
			}

			if len(authors) != 5 {
				t.Errorf("Expected 5 authors, got %d", len(authors))
			}

			// Test Skip
			q2 := NewQuery(db, "authors", []string{"id", "name"})
			q2.SetDialect(dialect.GetDialect(provider))
			q2.SetModelType(reflect.TypeOf(Author{}))
			q2.Skip(5)
			q2.Take(3)

			var authors2 []Author
			err = q2.Find(ctx, &authors2)
			if err != nil {
				t.Fatalf("FindMany with Skip failed: %v", err)
			}

			if len(authors2) != 3 {
				t.Errorf("Expected 3 authors, got %d", len(authors2))
			}

			// Verify SQL generation
			query, _ := q2.buildSelectQuery(false)
			if provider == "postgresql" {
				if !strings.Contains(query, "LIMIT") || !strings.Contains(query, "OFFSET") {
					t.Errorf("Expected LIMIT and OFFSET in PostgreSQL query, got: %s", query)
				}
			}
		})
	}
}

// TestWithContext tests WithContext functionality
func TestWithContext(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id SERIAL PRIMARY KEY,
						name TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Test query with context
			type Author struct {
				ID   int
				Name string
			}
			q := NewQuery(db, "authors", []string{"id", "name"})
			q.SetDialect(dialect.GetDialect(provider))
			q.SetModelType(reflect.TypeOf(Author{}))
			q.WithContext(ctx)

			var authors []Author
			err = q.Find(ctx, &authors)
			if err != nil {
				t.Fatalf("FindMany with context failed: %v", err)
			}

			// Verify context was used (no error means context worked)
			if ctx.Err() != nil && ctx.Err() != context.Canceled {
				t.Errorf("Context error: %v", ctx.Err())
			}
		})
	}
}

// TestRawSQL_Query tests Raw().Query() for multiple rows
func TestRawSQL_Query(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id SERIAL PRIMARY KEY,
						name TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO authors (name) VALUES ($1), ($2)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO authors (name) VALUES (?), (?)"
			}
			_, err = db.SQLDB().Exec(insertSQL, "John Doe", "Jane Smith")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test Raw Query
			var selectSQL string
			switch provider {
			case "postgresql":
				selectSQL = "SELECT id, name FROM authors WHERE name LIKE $1"
			case "mysql", "sqlite":
				selectSQL = "SELECT id, name FROM authors WHERE name LIKE ?"
			}

			rows, err := db.Query(ctx, selectSQL, "%Doe%")
			if err != nil {
				t.Fatalf("Raw Query failed: %v", err)
			}
			defer rows.Close()

			count := 0
			for rows.Next() {
				var id int
				var name string
				err = rows.Scan(&id, &name)
				if err != nil {
					t.Fatalf("Failed to scan row: %v", err)
				}
				count++
			}

			if count != 1 {
				t.Errorf("Expected 1 result, got %d", count)
			}
		})
	}
}

// TestRawSQL_QueryRow tests Raw().QueryRow() for single row
func TestRawSQL_QueryRow(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id SERIAL PRIMARY KEY,
						name TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO authors (name) VALUES ($1), ($2)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO authors (name) VALUES (?), (?)"
			}
			_, err = db.SQLDB().Exec(insertSQL, "John Doe", "Jane Smith")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test Raw QueryRow
			var count int
			err = db.QueryRow(ctx, "SELECT COUNT(*) FROM authors").Scan(&count)
			if err != nil {
				t.Fatalf("Raw QueryRow failed: %v", err)
			}

			if count != 2 {
				t.Errorf("Expected count 2, got %d", count)
			}
		})
	}
}

// TestRawSQL_Exec tests Raw().Exec() for INSERT/UPDATE/DELETE
func TestRawSQL_Exec(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id SERIAL PRIMARY KEY,
						name TEXT NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS authors (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := db.SQLDB().Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Test Raw Exec - INSERT
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO authors (name) VALUES ($1)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO authors (name) VALUES (?)"
			}

			_, err = db.Exec(ctx, insertSQL, "John Doe")
			if err != nil {
				t.Fatalf("Raw Exec INSERT failed: %v", err)
			}

			// Verify insert
			var count int
			err = db.QueryRow(ctx, "SELECT COUNT(*) FROM authors").Scan(&count)
			if err != nil {
				t.Fatalf("Failed to verify insert: %v", err)
			}

			if count != 1 {
				t.Errorf("Expected 1 author, got %d", count)
			}

			// Test Raw Exec - UPDATE
			var updateSQL string
			switch provider {
			case "postgresql":
				updateSQL = "UPDATE authors SET name = $1 WHERE name = $2"
			case "mysql", "sqlite":
				updateSQL = "UPDATE authors SET name = ? WHERE name = ?"
			}

			_, err = db.Exec(ctx, updateSQL, "Jonathan Doe", "John Doe")
			if err != nil {
				t.Fatalf("Raw Exec UPDATE failed: %v", err)
			}

			// Verify update
			var name string
			var selectSQL string
			switch provider {
			case "postgresql":
				selectSQL = "SELECT name FROM authors LIMIT 1"
			case "mysql", "sqlite":
				selectSQL = "SELECT name FROM authors LIMIT 1"
			}
			err = db.QueryRow(ctx, selectSQL).Scan(&name)
			if err != nil {
				t.Fatalf("Failed to verify update: %v", err)
			}

			if name != "Jonathan Doe" {
				t.Errorf("Expected name 'Jonathan Doe', got '%s'", name)
			}

			// Test Raw Exec - DELETE
			var deleteSQL string
			switch provider {
			case "postgresql":
				deleteSQL = "DELETE FROM authors WHERE name = $1"
			case "mysql", "sqlite":
				deleteSQL = "DELETE FROM authors WHERE name = ?"
			}

			_, err = db.Exec(ctx, deleteSQL, "Jonathan Doe")
			if err != nil {
				t.Fatalf("Raw Exec DELETE failed: %v", err)
			}

			// Verify delete
			err = db.QueryRow(ctx, "SELECT COUNT(*) FROM authors").Scan(&count)
			if err != nil {
				t.Fatalf("Failed to verify delete: %v", err)
			}

			if count != 0 {
				t.Errorf("Expected 0 authors, got %d", count)
			}
		})
	}
}
