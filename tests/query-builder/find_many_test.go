//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/prisma/db/inputs"
	"github.com/carlosnayan/prisma-go-client/prisma/db/tables/authors"
)

// TestFindMany_Basic tests basic FindMany functionality
func TestFindMany_Basic(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Smith")
	createTestAuthor(t, client, "Bob", "Johnson")

	// Find all authors
	authors, err := client.Authors.FindMany().Exec()

	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(authors) != 3 {
		t.Errorf("Expected 3 authors, got %d", len(authors))
	}
}

// TestFindMany_EmptyTable tests FindMany on empty table
func TestFindMany_EmptyTable(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Table is empty
	authors, err := client.Authors.FindMany().Exec()

	if err != nil {
		t.Fatalf("FindMany on empty table failed: %v", err)
	}

	if len(authors) != 0 {
		t.Errorf("Expected 0 authors, got %d", len(authors))
	}
}

// TestFindMany_WithWhere tests FindMany with WHERE clause
func TestFindMany_WithWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")
	createTestAuthor(t, client, "Bob", "Smith")

	// Find by last name
	authorsFound, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("FindMany with WHERE failed: %v", err)
	}

	if len(authorsFound) != 2 {
		t.Errorf("Expected 2 authors with last_name 'Doe', got %d", len(authorsFound))
	}

	for _, author := range authorsFound {
		if author.LastName != "Doe" {
			t.Errorf("Expected all authors to have last_name 'Doe', got '%s'", author.LastName)
		}
	}
}

// TestFindMany_WithTake tests pagination with Take (LIMIT)
func TestFindMany_WithTake(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create 10 test authors
	data := make([]inputs.AuthorsCreateManyArgs, 10)
	for i := 0; i < 10; i++ {
		data[i] = inputs.AuthorsCreateManyArgs{
			FirstName: "Author",
			LastName:  string(rune('A' + i)),
		}
	}
	createTestAuthorsBatch(t, client, data)

	// Take only 5
	authorsFound, err := client.Authors.FindMany().
		Take(5).
		Exec()

	if err != nil {
		t.Fatalf("FindMany with Take failed: %v", err)
	}

	if len(authorsFound) != 5 {
		t.Errorf("Expected 5 authors (Take 5), got %d", len(authorsFound))
	}
}

// TestFindMany_WithSkip tests pagination with Skip (OFFSET)
func TestFindMany_WithSkip(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create 10 test authors
	data := make([]inputs.AuthorsCreateManyArgs, 10)
	for i := 0; i < 10; i++ {
		data[i] = inputs.AuthorsCreateManyArgs{
			FirstName: "Author",
			LastName:  string(rune('A' + i)),
		}
	}
	createTestAuthorsBatch(t, client, data)

	// Skip first 7, take remaining 3
	authorsFound, err := client.Authors.FindMany().
		Skip(7).
		Exec()

	if err != nil {
		t.Fatalf("FindMany with Skip failed: %v", err)
	}

	if len(authorsFound) != 3 {
		t.Errorf("Expected 3 authors (Skip 7 from 10), got %d", len(authorsFound))
	}
}

// TestFindMany_Pagination tests Take + Skip together
func TestFindMany_Pagination(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create 20 test authors
	data := make([]inputs.AuthorsCreateManyArgs, 20)
	for i := 0; i < 20; i++ {
		data[i] = inputs.AuthorsCreateManyArgs{
			FirstName: "Author",
			LastName:  string(rune('A' + (i % 26))),
		}
	}
	createTestAuthorsBatch(t, client, data)

	// Page 2: Skip 10, Take 10
	page2, err := client.Authors.FindMany().
		Skip(10).
		Take(10).
		Exec()

	if err != nil {
		t.Fatalf("FindMany pagination failed: %v", err)
	}

	if len(page2) != 10 {
		t.Errorf("Expected 10 authors on page 2, got %d", len(page2))
	}

	// Page 3: Skip 20, Take 10 (should be empty)
	page3, err := client.Authors.FindMany().
		Skip(20).
		Take(10).
		Exec()

	if err != nil {
		t.Fatalf("FindMany pagination (page 3) failed: %v", err)
	}

	if len(page3) != 0 {
		t.Errorf("Expected 0 authors on page 3, got %d", len(page3))
	}
}

// TestFindMany_WithOrderBy tests ORDER BY
func TestFindMany_WithOrderBy(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "Charlie", "Brown")
	createTestAuthor(t, client, "Alice", "Anderson")
	createTestAuthor(t, client, "Bob", "Baker")

	// Order by last name ASC
	authorsAsc, err := client.Authors.FindMany().
		OrderBy("last_name", authors.OrderAsc).
		Exec()

	if err != nil {
		t.Fatalf("FindMany with ORDER BY ASC failed: %v", err)
	}

	if len(authorsAsc) != 3 {
		t.Fatalf("Expected 3 authors, got %d", len(authorsAsc))
	}

	if authorsAsc[0].LastName != "Anderson" {
		t.Errorf("Expected first author to be 'Anderson', got '%s'", authorsAsc[0].LastName)
	}

	if authorsAsc[1].LastName != "Baker" {
		t.Errorf("Expected second author to be 'Baker', got '%s'", authorsAsc[1].LastName)
	}

	if authorsAsc[2].LastName != "Brown" {
		t.Errorf("Expected third author to be 'Brown', got '%s'", authorsAsc[2].LastName)
	}

	// Order by last name DESC
	authorsDesc, err := client.Authors.FindMany().
		OrderBy("last_name", authors.OrderDesc).
		Exec()

	if err != nil {
		t.Fatalf("FindMany with ORDER BY DESC failed: %v", err)
	}

	if len(authorsDesc) != 3 {
		t.Fatalf("Expected 3 authors, got %d", len(authorsDesc))
	}

	if authorsDesc[0].LastName != "Brown" {
		t.Errorf("Expected first author to be 'Brown', got '%s'", authorsDesc[0].LastName)
	}
}

// TestFindMany_WithSelect tests field selection
func TestFindMany_WithSelect(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	createTestAuthor(t, client, "John", "Doe")

	// Select only specific fields
	authorsFound, err := client.Authors.FindMany().
		Select("id_author", "first_name").
		Exec()

	if err != nil {
		t.Fatalf("FindMany with SELECT failed: %v", err)
	}

	if len(authorsFound) != 1 {
		t.Fatalf("Expected 1 author, got %d", len(authorsFound))
	}

	author := authorsFound[0]
	if author.IdAuthor == "" {
		t.Error("Expected id_author to be populated")
	}

	if author.FirstName == "" {
		t.Error("Expected first_name to be populated")
	}
}

// TestFindMany_ComplexWhere tests complex WHERE conditions
func TestFindMany_ComplexWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	email1 := "john@example.com"
	email2 := "jane@example.com"
	createTestAuthorWithEmail(t, client, "John", "Doe", email1)
	createTestAuthorWithEmail(t, client, "Jane", "Doe", email2)
	createTestAuthor(t, client, "Bob", "Smith")

	// Find all Doe authors
	authorsFound, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("FindMany with complex WHERE failed: %v", err)
	}

	if len(authorsFound) != 2 {
		t.Errorf("Expected 2 authors with last_name 'Doe', got %d", len(authorsFound))
	}
}

// TestFindMany_WithWhereNoMatches tests WHERE with no matches
func TestFindMany_WithWhereNoMatches(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	createTestAuthor(t, client, "John", "Doe")

	// Find non-existent
	authorsFound, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("NonExistent")).
		Exec()

	if err != nil {
		t.Fatalf("FindMany with no matches failed: %v", err)
	}

	if len(authorsFound) != 0 {
		t.Errorf("Expected 0 authors, got %d", len(authorsFound))
	}
}

// TestFindMany_OrderByPagination tests ORDER BY with pagination
func TestFindMany_OrderByPagination(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "Charlie", "Brown")
	createTestAuthor(t, client, "Alice", "Anderson")
	createTestAuthor(t, client, "David", "Davis")
	createTestAuthor(t, client, "Bob", "Baker")

	// Get page 2 (skip 2, take 2) ordered by last name
	page2, err := client.Authors.FindMany().
		OrderBy("last_name", authors.OrderAsc).
		Skip(2).
		Take(2).
		Exec()

	if err != nil {
		t.Fatalf("FindMany with ORDER BY + pagination failed: %v", err)
	}

	if len(page2) != 2 {
		t.Errorf("Expected 2 authors on page 2, got %d", len(page2))
	}

	// Ordered: Anderson, Baker, Brown, Davis
	// Skip 2 (Anderson, Baker) → Take 2 (Brown, Davis)
	if page2[0].LastName != "Brown" {
		t.Errorf("Expected first result to be 'Brown', got '%s'", page2[0].LastName)
	}

	if page2[1].LastName != "Davis" {
		t.Errorf("Expected second result to be 'Davis', got '%s'", page2[1].LastName)
	}
}
