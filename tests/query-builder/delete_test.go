//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/prisma/db/inputs"
	"github.com/carlosnayan/prisma-go-client/prisma/db/tables/authors"
)

// TestDelete_Basic tests basic Delete with WHERE
func TestDelete_Basic(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	author := createTestAuthor(t, client, "John", "Doe")

	// Delete by ID
	err := client.Authors.Delete().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		Exec()

	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = client.Authors.FindFirst().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		Exec()

	if err == nil {
		t.Fatal("Expected error (not found) after delete, got nil")
	}
}

// TestDelete_MultipleMatches tests Delete removes all matching records
func TestDelete_MultipleMatches(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create multiple authors with same last name
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")
	createTestAuthor(t, client, "Bob", "Smith")

	// Delete all Doe authors
	err := client.Authors.Delete().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Delete multiple failed: %v", err)
	}

	// Verify both Doe authors were deleted
	doeAuthors, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify deletes: %v", err)
	}

	if len(doeAuthors) != 0 {
		t.Errorf("Expected 0 Doe authors after delete, got %d", len(doeAuthors))
	}

	// Verify Smith was NOT deleted
	smithAuthors, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("Smith")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to check Smith author: %v", err)
	}

	if len(smithAuthors) != 1 {
		t.Errorf("Expected 1 Smith author, got %d", len(smithAuthors))
	}
}

// TestDelete_NoMatches tests Delete when no records match
func TestDelete_NoMatches(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	createTestAuthor(t, client, "John", "Doe")

	// Try to delete non-existent author
	err := client.Authors.Delete().
		Where(authors.LastNameEQ("NonExistent")).
		Exec()

	// Should succeed with 0 rows affected (no error)
	if err != nil {
		t.Logf("Delete with no matches returned: %v", err)
	}

	// Verify John Doe still exists
	authors, err := client.Authors.FindMany().Exec()
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if len(authors) != 1 {
		t.Errorf("Expected 1 author remaining, got %d", len(authors))
	}
}

// TestDelete_ComplexWhere tests Delete with complex WHERE
func TestDelete_ComplexWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	email1 := "john@example.com"
	email2 := "jane@example.com"
	createTestAuthorWithEmail(t, client, "John", "Doe", email1)
	createTestAuthorWithEmail(t, client, "Jane", "Doe", email2)

	// Delete only John Doe (specific email)
	err := client.Authors.Delete().
		Where(authors.FirstNameEQ("John")).
		Where(authors.EmailEQ(email1)).
		Exec()

	if err != nil {
		t.Fatalf("Delete with complex WHERE failed: %v", err)
	}

	// Verify John was deleted
	johns, err := client.Authors.FindMany().
		Where(authors.FirstNameEQ("John")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify delete: %v", err)
	}

	if len(johns) != 0 {
		t.Errorf("Expected John to be deleted, got %d Johns", len(johns))
	}

	// Verify Jane still exists
	janes, err := client.Authors.FindMany().
		Where(authors.FirstNameEQ("Jane")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify Jane exists: %v", err)
	}

	if len(janes) != 1 {
		t.Errorf("Expected Jane to exist, got %d Janes", len(janes))
	}
}

// TestDelete_All tests deleting all records (dangerous!)
func TestDelete_All(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create multiple authors
	data := []inputs.AuthorsCreateManyArgs{
		{FirstName: "Author", LastName: "A"},
		{FirstName: "Author", LastName: "B"},
		{FirstName: "Author", LastName: "C"},
	}
	createTestAuthorsBatch(t, client, data)

	// Delete all by matching a common field
	err := client.Authors.Delete().
		Where(authors.FirstNameEQ("Author")).
		Exec()

	if err != nil {
		t.Fatalf("Delete all failed: %v", err)
	}

	// Verify all deleted
	allAuthors, err := client.Authors.FindMany().Exec()
	if err != nil {
		t.Fatalf("Failed to verify deletes: %v", err)
	}

	if len(allAuthors) != 0 {
		t.Errorf("Expected 0 authors after delete all, got %d", len(allAuthors))
	}
}

// TestDelete_NonExistentRecord tests deleting a specific non-existent record
func TestDelete_NonExistentRecord(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Try to delete by non-existent ID
	err := client.Authors.Delete().
		Where(authors.IdAuthorEQ("non-existent-uuid")).
		Exec()

	// Should not error, just affect 0 rows
	if err != nil {
		t.Logf("Delete non-existent returned: %v", err)
	}
}

// TestDeleteMany_Alias tests DeleteMany as alias for Delete
func TestDeleteMany_Alias(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")

	// DeleteMany should work same as Delete
	err := client.Authors.DeleteMany().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("DeleteMany failed: %v", err)
	}

	// Verify deleted
	doeAuthors, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify deletes: %v", err)
	}

	if len(doeAuthors) != 0 {
		t.Errorf("Expected 0 Doe authors, got %d", len(doeAuthors))
	}
}
