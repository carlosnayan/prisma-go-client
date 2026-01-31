//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/prisma/db/tables/authors"
)

// TestFindFirst_Basic tests basic FindFirst functionality
func TestFindFirst_Basic(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Smith")

	// Find first author
	author, err := client.Authors.FindFirst().Exec()

	if err != nil {
		t.Fatalf("FindFirst failed: %v", err)
	}

	if author == nil {
		t.Fatal("Expected author to be returned, got nil")
	}

	if author.FirstName == "" || author.LastName == "" {
		t.Error("Expected author to have name fields populated")
	}
}

// TestFindFirst_WithWhere tests FindFirst with WHERE clause
func TestFindFirst_WithWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Smith")

	// Find by last name
	author, err := client.Authors.FindFirst().
		Where(authors.LastNameEQ("Smith")).
		Exec()

	if err != nil {
		t.Fatalf("FindFirst with WHERE failed: %v", err)
	}

	if author == nil {
		t.Fatal("Expected author to be found")
	}

	if author.LastName != "Smith" {
		t.Errorf("Expected last_name 'Smith', got '%s'", author.LastName)
	}
}

// TestFindFirst_WithOrderBy tests FindFirst with ORDER BY
func TestFindFirst_WithOrderBy(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "Charlie", "Brown")
	createTestAuthor(t, client, "Alice", "Anderson")
	createTestAuthor(t, client, "Bob", "Baker")

	// Find first ordered by last name ascending
	author, err := client.Authors.FindFirst().
		OrderBy("last_name", authors.OrderAsc).
		Exec()

	if err != nil {
		t.Fatalf("FindFirst with ORDER BY failed: %v", err)
	}

	if author == nil {
		t.Fatal("Expected author to be found")
	}

	if author.LastName != "Anderson" {
		t.Errorf("Expected first author (ordered by last_name ASC) to be 'Anderson', got '%s'", author.LastName)
	}

	// Find first ordered by last name descending
	author2, err := client.Authors.FindFirst().
		OrderBy("last_name", authors.OrderDesc).
		Exec()

	if err != nil {
		t.Fatalf("FindFirst with ORDER BY DESC failed: %v", err)
	}

	if author2 == nil {
		t.Fatal("Expected author to be found")
	}

	if author2.LastName != "Brown" {
		t.Errorf("Expected first author (ordered by last_name DESC) to be 'Brown', got '%s'", author2.LastName)
	}
}

// TestFindFirst_WithSelect tests FindFirst with specific field selection
func TestFindFirst_WithSelect(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	createTestAuthor(t, client, "John", "Doe")

	// Find with only specific fields
	author, err := client.Authors.FindFirst().
		Select("id_author", "first_name").
		Exec()

	if err != nil {
		t.Fatalf("FindFirst with SELECT failed: %v", err)
	}

	if author == nil {
		t.Fatal("Expected author to be found")
	}

	if author.IdAuthor == "" {
		t.Error("Expected id_author to be populated")
	}

	if author.FirstName == "" {
		t.Error("Expected first_name to be populated")
	}

	// Note: last_name might be empty string (not selected)
	// This depends on how the scan handles non-selected fields
}

// TestFindFirst_NotFound tests FindFirst when no record matches
func TestFindFirst_NotFound(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Try to find non-existent author
	_, err := client.Authors.FindFirst().
		Where(authors.LastNameEQ("NonExistent")).
		Exec()

	if err == nil {
		t.Fatal("Expected error for not found, got nil")
	}

	// Should return ErrRecordNotFound or similar
}

// TestFindFirst_ComplexWhere tests FindFirst with complex WHERE conditions
func TestFindFirst_ComplexWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	email1 := "john@example.com"
	email2 := "jane@example.com"
	createTestAuthorWithEmail(t, client, "John", "Doe", email1)
	createTestAuthorWithEmail(t, client, "Jane", "Smith", email2)

	// Find with multiple conditions
	author, err := client.Authors.FindFirst().
		Where(authors.FirstNameEQ("John")).
		Where(authors.EmailEQ(email1)).
		Exec()

	if err != nil {
		t.Fatalf("FindFirst with complex WHERE failed: %v", err)
	}

	if author == nil {
		t.Fatal("Expected author to be found")
	}

	if author.FirstName != "John" {
		t.Errorf("Expected first_name 'John', got '%s'", author.FirstName)
	}

	if author.Email == nil || *author.Email != email1 {
		t.Errorf("Expected email '%s', got '%v'", email1, author.Email)
	}
}

// TestFindFirst_EmptyTable tests FindFirst on empty table
func TestFindFirst_EmptyTable(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Table is empty
	_, err := client.Authors.FindFirst().Exec()

	if err == nil {
		t.Fatal("Expected error for empty table, got nil")
	}
}

// TestFindFirst_MultipleMatches tests FindFirst returns only first match
func TestFindFirst_MultipleMatches(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create multiple authors with same last name
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")
	createTestAuthor(t, client, "Jack", "Doe")

	// Find should return only ONE author
	author, err := client.Authors.FindFirst().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("FindFirst failed: %v", err)
	}

	if author == nil {
		t.Fatal("Expected one author to be found")
	}

	// Verify it's ONE of the Doe authors (any is fine)
	if author.LastName != "Doe" {
		t.Errorf("Expected last_name 'Doe', got '%s'", author.LastName)
	}
}
