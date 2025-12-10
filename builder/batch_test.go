package builder

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// TestOrderBy_SingleField tests ordering by a single field
func TestOrderBy_SingleField(t *testing.T) {
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

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL,
						created_at DATETIME DEFAULT CURRENT_TIMESTAMP
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO books (title, author) VALUES ($1, $2), ($3, $4), ($5, $6)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO books (title, author) VALUES (?, ?), (?, ?), (?, ?)"
			}
			_, err = sqlDB.Exec(insertSQL,
				"Book C", "Author C",
				"Book A", "Author A",
				"Book B", "Author B")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test OrderBy ASC
			columns := []string{"id", "title", "author", "created_at"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			opts := QueryOptions{
				OrderBy: []OrderBy{{Field: "title", Order: "ASC"}},
			}

			results, err := builder.FindMany(ctx, opts)
			if err != nil {
				t.Fatalf("FindMany failed: %v", err)
			}

			books, ok := results.([]Book)
			if !ok {
				t.Fatalf("Expected []Book, got %T", results)
			}

			if len(books) != 3 {
				t.Fatalf("Expected 3 books, got %d", len(books))
			}

			if books[0].Title != "Book A" {
				t.Errorf("Expected first book to be 'Book A', got '%s'", books[0].Title)
			}
			if books[1].Title != "Book B" {
				t.Errorf("Expected second book to be 'Book B', got '%s'", books[1].Title)
			}
			if books[2].Title != "Book C" {
				t.Errorf("Expected third book to be 'Book C', got '%s'", books[2].Title)
			}
		})
	}
}

// TestOrderBy_MultipleFields tests ordering by multiple fields
func TestOrderBy_MultipleFields(t *testing.T) {
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

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						year INT
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						year INT
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL,
						year INTEGER
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data with same author, different years
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO books (title, author, year) VALUES ($1, $2, $3), ($4, $5, $6), ($7, $8, $9)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO books (title, author, year) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)"
			}
			_, err = sqlDB.Exec(insertSQL,
				"Book 1", "Author A", 2020,
				"Book 2", "Author A", 2021,
				"Book 3", "Author B", 2020)
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			columns := []string{"id", "title", "author", "year"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Order by author ASC, then year DESC
			opts := QueryOptions{
				OrderBy: []OrderBy{
					{Field: "author", Order: "ASC"},
					{Field: "year", Order: "DESC"},
				},
			}

			results, err := builder.FindMany(ctx, opts)
			if err != nil {
				t.Fatalf("FindMany failed: %v", err)
			}

			books, ok := results.([]Book)
			if !ok {
				t.Fatalf("Expected []Book, got %T", results)
			}

			if len(books) != 3 {
				t.Fatalf("Expected 3 books, got %d", len(books))
			}

			// First two should be Author A (year DESC: 2021, then 2020)
			if books[0].Author != "Author A" || books[0].Title != "Book 2" {
				t.Errorf("Expected first book to be Author A, Book 2, got %s, %s", books[0].Author, books[0].Title)
			}
			if books[1].Author != "Author A" || books[1].Title != "Book 1" {
				t.Errorf("Expected second book to be Author A, Book 1, got %s, %s", books[1].Author, books[1].Title)
			}
			if books[2].Author != "Author B" {
				t.Errorf("Expected third book to be Author B, got %s", books[2].Author)
			}
		})
	}
}

// TestOrderBy_WithWhere tests OrderBy combined with WHERE clause
func TestOrderBy_WithWhere(t *testing.T) {
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

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO books (title, author) VALUES ($1, $2), ($3, $4), ($5, $6)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO books (title, author) VALUES (?, ?), (?, ?), (?, ?)"
			}
			_, err = sqlDB.Exec(insertSQL,
				"Book C", "Author A",
				"Book A", "Author A",
				"Book B", "Author B")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			columns := []string{"id", "title", "author"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Filter by author and order by title
			opts := QueryOptions{
				Where:   Where{"author": "Author A"},
				OrderBy: []OrderBy{{Field: "title", Order: "ASC"}},
			}

			results, err := builder.FindMany(ctx, opts)
			if err != nil {
				t.Fatalf("FindMany failed: %v", err)
			}

			books, ok := results.([]Book)
			if !ok {
				t.Fatalf("Expected []Book, got %T", results)
			}

			if len(books) != 2 {
				t.Fatalf("Expected 2 books, got %d", len(books))
			}

			if books[0].Title != "Book A" {
				t.Errorf("Expected first book to be 'Book A', got '%s'", books[0].Title)
			}
			if books[1].Title != "Book C" {
				t.Errorf("Expected second book to be 'Book C', got '%s'", books[1].Title)
			}
		})
	}
}

// TestCreateMany_Basic tests basic CreateMany functionality
func TestCreateMany_Basic(t *testing.T) {
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

			// Create table with auto-increment ID
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			columns := []string{"id", "title", "author"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Create multiple books
			books := []interface{}{
				Book{Title: "Book 1", Author: "Author A"},
				Book{Title: "Book 2", Author: "Author B"},
				Book{Title: "Book 3", Author: "Author C"},
			}

			result, err := builder.CreateMany(ctx, books, false)
			if err != nil {
				t.Fatalf("CreateMany failed: %v", err)
			}

			if result.Count != 3 {
				t.Errorf("Expected 3 books created, got %d", result.Count)
			}

			// Verify books were created
			var count int
			err = sqlDB.QueryRow("SELECT COUNT(*) FROM books").Scan(&count)
			if err != nil {
				t.Fatalf("Failed to count books: %v", err)
			}

			if count != 3 {
				t.Errorf("Expected 3 books in database, got %d", count)
			}
		})
	}
}

// TestCreateMany_EmptySlice tests CreateMany with empty slice
func TestCreateMany_EmptySlice(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t, "postgresql")
	defer cleanup()

	ctx := context.Background()

	columns := []string{"id", "title", "author"}
	builder := NewTableQueryBuilder(db, "books", columns)
	builder.SetDialect(dialect.GetDialect("postgresql"))
	builder.SetPrimaryKey("id")
	builder.SetModelType(reflect.TypeOf(Book{}))

	result, err := builder.CreateMany(ctx, []interface{}{}, false)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Expected 0 books created, got %d", result.Count)
	}
}

// TestUpdateMany_Basic tests basic UpdateMany functionality
func TestUpdateMany_Basic(t *testing.T) {
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

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO books (title, author) VALUES ($1, $2), ($3, $4), ($5, $6)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO books (title, author) VALUES (?, ?), (?, ?), (?, ?)"
			}
			_, err = sqlDB.Exec(insertSQL,
				"Book 1", "Author A",
				"Book 2", "Author A",
				"Book 3", "Author B")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			columns := []string{"id", "title", "author"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Update all books by Author A - using a struct with only the fields we want to update
			// Since Book doesn't have Status, we'll update title instead
			updateData := Book{Title: "Updated Title"}
			where := Where{"author": "Author A"}

			result, err := builder.UpdateMany(ctx, where, updateData)
			if err != nil {
				t.Fatalf("UpdateMany failed: %v", err)
			}

			if result.Count != 2 {
				t.Errorf("Expected 2 books updated, got %d", result.Count)
			}

			// Verify updates
			var count int
			var verifySQL string
			switch provider {
			case "postgresql":
				verifySQL = "SELECT COUNT(*) FROM books WHERE author = $1 AND title = $2"
			case "mysql", "sqlite":
				verifySQL = "SELECT COUNT(*) FROM books WHERE author = ? AND title = ?"
			}
			err = sqlDB.QueryRow(verifySQL, "Author A", "Updated Title").Scan(&count)
			if err != nil {
				t.Fatalf("Failed to verify updates: %v", err)
			}

			if count != 2 {
				t.Errorf("Expected 2 books with updated title, got %d", count)
			}
		})
	}
}

// TestUpdateMany_NoMatches tests UpdateMany when no records match
func TestUpdateMany_NoMatches(t *testing.T) {
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

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			columns := []string{"id", "title", "author"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Try to update non-existent author
			updateData := Book{Title: "Updated"}
			where := Where{"author": "NonExistent"}

			result, err := builder.UpdateMany(ctx, where, updateData)
			if err != nil {
				t.Fatalf("UpdateMany failed: %v", err)
			}

			if result.Count != 0 {
				t.Errorf("Expected 0 books updated, got %d", result.Count)
			}
		})
	}
}

// TestUpdateMany_EmptyWhere tests UpdateMany with empty where (should fail)
func TestUpdateMany_EmptyWhere(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t, "postgresql")
	defer cleanup()

	ctx := context.Background()

	columns := []string{"id", "title", "author"}
	builder := NewTableQueryBuilder(db, "books", columns)
	builder.SetDialect(dialect.GetDialect("postgresql"))
	builder.SetPrimaryKey("id")
	builder.SetModelType(reflect.TypeOf(Book{}))

	updateData := Book{Title: "Updated"}
	where := Where{}

	_, err := builder.UpdateMany(ctx, where, updateData)
	if err == nil {
		t.Error("Expected error for empty where condition, got nil")
	}

	if err != nil && err.Error() == "" {
		t.Error("Expected error message about empty where condition")
	}
}

// TestOrderBy_ASC_DESC tests both ASC and DESC ordering
func TestOrderBy_ASC_DESC(t *testing.T) {
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

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO books (title) VALUES ($1), ($2), ($3)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO books (title) VALUES (?), (?), (?)"
			}
			_, err = sqlDB.Exec(insertSQL,
				"Book C", "Book A", "Book B")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			columns := []string{"id", "title"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Test DESC
			opts := QueryOptions{
				OrderBy: []OrderBy{{Field: "title", Order: "DESC"}},
			}

			results, err := builder.FindMany(ctx, opts)
			if err != nil {
				t.Fatalf("FindMany failed: %v", err)
			}

			books, ok := results.([]Book)
			if !ok {
				t.Fatalf("Expected []Book, got %T", results)
			}

			if len(books) != 3 {
				t.Fatalf("Expected 3 books, got %d", len(books))
			}

			if books[0].Title != "Book C" {
				t.Errorf("Expected first book to be 'Book C', got '%s'", books[0].Title)
			}
			if books[2].Title != "Book A" {
				t.Errorf("Expected last book to be 'Book A', got '%s'", books[2].Title)
			}
		})
	}
}

// TestOrderBy_WithLimitOffset tests OrderBy with LIMIT and OFFSET
func TestOrderBy_WithLimitOffset(t *testing.T) {
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

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert test data
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = "INSERT INTO books (title) VALUES ($1), ($2), ($3), ($4), ($5)"
			case "mysql", "sqlite":
				insertSQL = "INSERT INTO books (title) VALUES (?), (?), (?), (?), (?)"
			}
			_, err = sqlDB.Exec(insertSQL,
				"Book A", "Book B", "Book C", "Book D", "Book E")
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			columns := []string{"id", "title"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Test with LIMIT and OFFSET
			take := 2
			skip := 1
			opts := QueryOptions{
				OrderBy: []OrderBy{{Field: "title", Order: "ASC"}},
				Take:    &take,
				Skip:    &skip,
			}

			results, err := builder.FindMany(ctx, opts)
			if err != nil {
				t.Fatalf("FindMany failed: %v", err)
			}

			books, ok := results.([]Book)
			if !ok {
				t.Fatalf("Expected []Book, got %T", results)
			}

			if len(books) != 2 {
				t.Fatalf("Expected 2 books, got %d", len(books))
			}

			// Should get Book B and Book C (skip Book A, take 2)
			if books[0].Title != "Book B" {
				t.Errorf("Expected first book to be 'Book B', got '%s'", books[0].Title)
			}
			if books[1].Title != "Book C" {
				t.Errorf("Expected second book to be 'Book C', got '%s'", books[1].Title)
			}
		})
	}
}

// TestCreateMany_LargeBatch tests CreateMany with a large batch
func TestCreateMany_LargeBatch(t *testing.T) {
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

			// Create table
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.Exec(createTableSQL)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			columns := []string{"id", "title", "author"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Create 1500 books (larger than batch size of 1000)
			books := make([]interface{}, 1500)
			for i := 0; i < 1500; i++ {
				books[i] = Book{
					Title:  fmt.Sprintf("Book %d", i+1),
					Author: fmt.Sprintf("Author %d", (i%10)+1),
				}
			}

			result, err := builder.CreateMany(ctx, books, false)
			if err != nil {
				t.Fatalf("CreateMany failed: %v", err)
			}

			if result.Count != 1500 {
				t.Errorf("Expected 1500 books created, got %d", result.Count)
			}

			// Verify books were created
			var count int
			err = sqlDB.QueryRow("SELECT COUNT(*) FROM books").Scan(&count)
			if err != nil {
				t.Fatalf("Failed to count books: %v", err)
			}

			if count != 1500 {
				t.Errorf("Expected 1500 books in database, got %d", count)
			}
		})
	}
}
