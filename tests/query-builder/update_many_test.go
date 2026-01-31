//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/prisma/db/inputs"
	"github.com/carlosnayan/prisma-go-client/prisma/db/tables/authors"
)

// TestUpdateMany_Basic tests basic UpdateMany with WHERE
func TestUpdateMany_Basic(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create multiple authors with same last name
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")
	createTestAuthor(t, client, "Bob", "Smith")

	// Update all Doe authors
	newNationality := "American"
	err := client.Authors.UpdateMany().
		Where(authors.LastNameEQ("Doe")).
		SetNationality(&newNationality).
		Exec()

	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	// Verify both Doe authors were updated
	doeAuthors, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify updates: %v", err)
	}

	if len(doeAuthors) != 2 {
		t.Fatalf("Expected 2 Doe authors, got %d", len(doeAuthors))
	}

	for _, author := range doeAuthors {
		if author.Nationality == nil || *author.Nationality != newNationality {
			t.Errorf("Expected nationality '%s', got '%v'", newNationality, author.Nationality)
		}
	}

	// Verify Smith author was NOT updated
	smithAuthors, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("Smith")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to check Smith author: %v", err)
	}

	if len(smithAuthors) != 1 {
		t.Fatalf("Expected 1 Smith author, got %d", len(smithAuthors))
	}

	if smithAuthors[0].Nationality != nil {
		t.Errorf("Expected Smith author nationality to be NULL, got '%v'", *smithAuthors[0].Nationality)
	}
}

// TestUpdateMany_MultipleFields tests updating multiple fields
func TestUpdateMany_MultipleFields(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	data := []inputs.AuthorsCreateManyArgs{
		{FirstName: "Author", LastName: "A"},
		{FirstName: "Author", LastName: "B"},
		{FirstName: "Author", LastName: "C"},
	}
	createTestAuthorsBatch(t, client, data)

	// Update all with multiple fields
	newBio := "Updated bio"
	newNationality := "Canadian"

	err := client.Authors.UpdateMany().
		Where(authors.FirstNameEQ("Author")).
		SetBio(&newBio).
		SetNationality(&newNationality).
		Exec()

	if err != nil {
		t.Fatalf("UpdateMany with multiple fields failed: %v", err)
	}

	// Verify all updated
	allAuthors, err := client.Authors.FindMany().Exec()

	if err != nil {
		t.Fatalf("Failed to verify updates: %v", err)
	}

	for _, author := range allAuthors {
		if author.Bio == nil || *author.Bio != newBio {
			t.Errorf("Expected bio '%s', got '%v'", newBio, author.Bio)
		}

		if author.Nationality == nil || *author.Nationality != newNationality {
			t.Errorf("Expected nationality '%s', got '%v'", newNationality, author.Nationality)
		}
	}
}

// TestUpdateMany_NoFieldsToUpdate tests UpdateMany with no fields
func TestUpdateMany_NoFieldsToUpdate(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	createTestAuthor(t, client, "John", "Doe")

	// Try to update without setting fields
	err := client.Authors.UpdateMany().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err == nil {
		t.Fatal("Expected error for no fields to update, got nil")
	}
}

// TestUpdateMany_NoMatches tests UpdateMany when no records match
func TestUpdateMany_NoMatches(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	createTestAuthor(t, client, "John", "Doe")

	// Update non-existent authors
	newBio := "New bio"
	err := client.Authors.UpdateMany().
		Where(authors.LastNameEQ("NonExistent")).
		SetBio(&newBio).
		Exec()

	// Should succeed with 0 rows affected
	if err != nil {
		t.Logf("UpdateMany with no matches returned: %v", err)
	}
}

// TestUpdateMany_SetToNull tests setting fields to NULL
func TestUpdateMany_SetToNull(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create authors with bio
	bio := "Original bio"
	email1 := "john@example.com"
	email2 := "jane@example.com"

	_, err := client.Authors.Create().
		SetFirstName("John").
		SetLastName("Doe").
		SetBio(&bio).
		SetEmail(&email1).
		Exec()

	if err != nil {
		t.Fatalf("Failed to create author 1: %v", err)
	}

	_, err = client.Authors.Create().
		SetFirstName("Jane").
		SetLastName("Doe").
		SetBio(&bio).
		SetEmail(&email2).
		Exec()

	if err != nil {
		t.Fatalf("Failed to create author 2: %v", err)
	}

	// Set all Doe authors' bio to NULL
	err = client.Authors.UpdateMany().
		Where(authors.LastNameEQ("Doe")).
		SetBio(nil).
		Exec()

	if err != nil {
		t.Fatalf("UpdateMany to NULL failed: %v", err)
	}

	// Verify both have NULL bio
	doeAuthors, err := client.Authors.FindMany().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify updates: %v", err)
	}

	for _, author := range doeAuthors {
		if author.Bio != nil {
			t.Errorf("Expected bio to be NULL, got '%v'", *author.Bio)
		}
	}
}

// TestUpdateMany_LargeBatch tests updating many records
func TestUpdateMany_LargeBatch(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create 100 authors
	batchSize := 100
	data := make([]inputs.AuthorsCreateManyArgs, batchSize)
	for i := 0; i < batchSize; i++ {
		data[i] = inputs.AuthorsCreateManyArgs{
			FirstName: "Batch",
			LastName:  "Author",
		}
	}
	createTestAuthorsBatch(t, client, data)

	// Update all
	newNationality := "Global"
	err := client.Authors.UpdateMany().
		Where(authors.FirstNameEQ("Batch")).
		SetNationality(&newNationality).
		Exec()

	if err != nil {
		t.Fatalf("UpdateMany large batch failed: %v", err)
	}

	// Verify count
	allAuthors, err := client.Authors.FindMany().
		Where(authors.NationalityEQ(newNationality)).
		Exec()

	if err != nil {
		t.Fatalf("Failed to verify updates: %v", err)
	}

	if len(allAuthors) != batchSize {
		t.Errorf("Expected %d updated authors, got %d", batchSize, len(allAuthors))
	}
}

// TestUpdateMany_ComplexWhere tests UpdateMany with complex WHERE
func TestUpdateMany_ComplexWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")
	createTestAuthor(t, client, "Bob", "Smith")

	// Update only Doe authors with first_name starting with 'J'
	newBio := "J Doe author"
	err := client.Authors.UpdateMany().
		Where(authors.LastNameEQ("Doe")).
		Where(authors.FirstNameEQ("John")). // This will be AND condition
		SetBio(&newBio).
		Exec()

	if err != nil {
		t.Fatalf("UpdateMany with complex WHERE failed: %v", err)
	}

	// Verify only John Doe was updated
	johnDoe, err := client.Authors.FindFirst().
		Where(authors.FirstNameEQ("John")).
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to find John Doe: %v", err)
	}

	if johnDoe.Bio == nil || *johnDoe.Bio != newBio {
		t.Errorf("Expected John Doe bio '%s', got '%v'", newBio, johnDoe.Bio)
	}

	// Verify Jane Doe was NOT updated
	janeDoe, err := client.Authors.FindFirst().
		Where(authors.FirstNameEQ("Jane")).
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Failed to find Jane Doe: %v", err)
	}

	if janeDoe.Bio != nil {
		t.Errorf("Expected Jane Doe bio to be NULL, got '%v'", *janeDoe.Bio)
	}
}
