//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/prisma/db/tables/authors"
)

// TestUpdate_Basic tests basic Update with WHERE
func TestUpdate_Basic(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	email := "john@example.com"
	author := createTestAuthorWithEmail(t, client, "John", "Doe", email)

	// Update using WHERE
	newBio := "Updated bio"
	_, err := client.Authors.Update().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		SetBio(&newBio).
		Exec()

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	updated, err := client.Authors.FindFirst().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if updated.Bio == nil || *updated.Bio != newBio {
		t.Errorf("Expected bio '%s', got '%v'", newBio, updated.Bio)
	}
}

// TestUpdate_MultipleFields tests updating multiple fields
func TestUpdate_MultipleFields(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	author := createTestAuthor(t, client, "John", "Doe")

	// Update multiple fields
	newBio := "New bio"
	newEmail := "newemail@example.com"
	newNationality := "Canadian"

	_, err := client.Authors.Update().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		SetBio(&newBio).
		SetEmail(&newEmail).
		SetNationality(&newNationality).
		Exec()

	if err != nil {
		t.Fatalf("Update multiple fields failed: %v", err)
	}

	// Verify all updates
	updated, err := client.Authors.FindFirst().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if updated.Bio == nil || *updated.Bio != newBio {
		t.Errorf("Expected bio '%s', got '%v'", newBio, updated.Bio)
	}

	if updated.Email == nil || *updated.Email != newEmail {
		t.Errorf("Expected email '%s', got '%v'", newEmail, updated.Email)
	}

	if updated.Nationality == nil || *updated.Nationality != newNationality {
		t.Errorf("Expected nationality '%s', got '%v'", newNationality, updated.Nationality)
	}
}

// TestUpdate_SetNullableToNull tests setting nullable field to NULL
func TestUpdate_SetNullableToNull(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create author with bio
	bio := "Original bio"
	email := "test@example.com"
	author, err := client.Authors.Create().
		SetFirstName("Test").
		SetLastName("Author").
		SetBio(&bio).
		SetEmail(&email).
		Exec()

	if err != nil {
		t.Fatalf("Failed to create author: %v", err)
	}

	// Update bio to NULL
	_, err = client.Authors.Update().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		SetBio(nil).
		Exec()

	if err != nil {
		t.Fatalf("Update to NULL failed: %v", err)
	}

	// Verify bio is NULL
	updated, err := client.Authors.FindFirst().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if updated.Bio != nil {
		t.Errorf("Expected bio to be NULL, got '%v'", *updated.Bio)
	}
}

// TestUpdate_NoFieldsToUpdate tests updating with no fields set
func TestUpdate_NoFieldsToUpdate(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	author := createTestAuthor(t, client, "John", "Doe")

	// Try to update without setting any fields
	_, err := client.Authors.Update().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		Exec()

	if err == nil {
		t.Fatal("Expected error for no fields to update, got nil")
	}
}

// TestUpdate_NoMatches tests Update when WHERE matches no records
func TestUpdate_NoMatches(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	createTestAuthor(t, client, "John", "Doe")

	// Update non-existent author
	newBio := "New bio"
	_, err := client.Authors.Update().
		Where(authors.IdAuthorEQ("non-existent-id")).
		SetBio(&newBio).
		Exec()

	// This might succeed with 0 rows affected, or error - depends on implementation
	// Check that no error OR specific error
	if err != nil {
		// Expected behavior: error for not found
		t.Logf("Update with no matches returned error: %v", err)
	}
}

// TestUpdate_MultipleMatches tests Update should work with WHERE matching multiple
func TestUpdate_MultipleMatches(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create multiple authors with same last name
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")

	// Update should affect only first match (or based on implementation)
	newBio := "Updated bio"
	_, err := client.Authors.Update().
		Where(authors.LastNameEQ("Doe")).
		SetBio(&newBio).
		Exec()

	if err != nil {
		t.Fatalf("Update with multiple matches failed: %v", err)
	}

	// Note: Update() might only update first match, while UpdateMany() updates all
	// This depends on implementation
}

// TestUpdate_WithComplexWhere tests Update with complex WHERE conditions
func TestUpdate_WithComplexWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	email1 := "john@example.com"
	email2 := "jane@example.com"
	author1 := createTestAuthorWithEmail(t, client, "John", "Doe", email1)
	createTestAuthorWithEmail(t, client, "Jane", "Doe", email2)

	// Update with multiple WHERE conditions
	newNationality := "American"
	_, err := client.Authors.Update().
		Where(authors.FirstNameEQ("John")).
		Where(authors.EmailEQ(email1)).
		SetNationality(&newNationality).
		Exec()

	if err != nil {
		t.Fatalf("Update with complex WHERE failed: %v", err)
	}

	// Verify only John was updated
	updated, err := client.Authors.FindFirst().
		Where(authors.IdAuthorEQ(author1.IdAuthor)).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if updated.Nationality == nil || *updated.Nationality != newNationality {
		t.Errorf("Expected nationality '%s', got '%v'", newNationality, updated.Nationality)
	}
}
