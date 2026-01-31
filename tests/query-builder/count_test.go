//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/prisma/db/inputs"
	"github.com/carlosnayan/prisma-go-client/prisma/db/tables/authors"
)

// TestCount_Basic tests basic Count functionality
func TestCount_Basic(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Smith")
	createTestAuthor(t, client, "Bob", "Johnson")

	// Count all
	count, err := client.Authors.Count().Exec()

	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

// TestCount_EmptyTable tests Count on empty table
func TestCount_EmptyTable(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Table is empty
	count, err := client.Authors.Count().Exec()

	if err != nil {
		t.Fatalf("Count on empty table failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

// TestCount_WithWhere tests Count with WHERE clause
func TestCount_WithWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")
	createTestAuthor(t, client, "Bob", "Smith")

	// Count Doe authors
	count, err := client.Authors.Count().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Count with WHERE failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2 (Doe authors), got %d", count)
	}
}

// TestCount_NoMatches tests Count when WHERE matches nothing
func TestCount_NoMatches(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test author
	createTestAuthor(t, client, "John", "Doe")

	// Count non-existent
	count, err := client.Authors.Count().
		Where(authors.LastNameEQ("NonExistent")).
		Exec()

	if err != nil {
		t.Fatalf("Count with no matches failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

// TestCount_ComplexWhere tests Count with complex WHERE conditions
func TestCount_ComplexWhere(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	email1 := "john@example.com"
	email2 := "jane@example.com"
	createTestAuthorWithEmail(t, client, "John", "Doe", email1)
	createTestAuthorWithEmail(t, client, "Jane", "Doe", email2)
	createTestAuthor(t, client, "Bob", "Smith")

	// Count Doe authors with email
	count, err := client.Authors.Count().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Count with complex WHERE failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2 (Doe authors), got %d", count)
	}
}

// TestCount_LargeDataset tests Count with large dataset
func TestCount_LargeDataset(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create 500 authors
	batchSize := 500
	data := make([]inputs.AuthorsCreateManyArgs, batchSize)
	for i := 0; i < batchSize; i++ {
		data[i] = inputs.AuthorsCreateManyArgs{
			FirstName: "Batch",
			LastName:  "Author",
		}
	}
	createTestAuthorsBatch(t, client, data)

	// Count all
	count, err := client.Authors.Count().Exec()

	if err != nil {
		t.Fatalf("Count large dataset failed: %v", err)
	}

	if count != int64(batchSize) {
		t.Errorf("Expected count %d, got %d", batchSize, count)
	}
}

// TestCount_AfterDelete tests Count reflects deletions
func TestCount_AfterDelete(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Smith")
	createTestAuthor(t, client, "Bob", "Johnson")

	// Initial count
	count1, err := client.Authors.Count().Exec()
	if err != nil {
		t.Fatalf("Initial count failed: %v", err)
	}

	if count1 != 3 {
		t.Errorf("Expected initial count 3, got %d", count1)
	}

	// Delete one author
	err = client.Authors.Delete().
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Count after delete
	count2, err := client.Authors.Count().Exec()
	if err != nil {
		t.Fatalf("Count after delete failed: %v", err)
	}

	if count2 != 2 {
		t.Errorf("Expected count 2 after delete, got %d", count2)
	}
}

// TestCount_AfterUpdate tests Count unchanged after updates
func TestCount_AfterUpdate(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Smith")

	// Initial count
	count1, err := client.Authors.Count().Exec()
	if err != nil {
		t.Fatalf("Initial count failed: %v", err)
	}

	if count1 != 2 {
		t.Errorf("Expected initial count 2, got %d", count1)
	}

	// Update authors
	newBio := "Updated"
	err = client.Authors.UpdateMany().
		Where(authors.LastNameEQ("Doe")).
		SetBio(&newBio).
		Exec()

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Count after update (should be same)
	count2, err := client.Authors.Count().Exec()
	if err != nil {
		t.Fatalf("Count after update failed: %v", err)
	}

	if count2 != count1 {
		t.Errorf("Expected count unchanged after update (%d), got %d", count1, count2)
	}
}

// TestCount_MultipleConditions tests Count with multiple WHERE conditions
func TestCount_MultipleConditions(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create test authors
	createTestAuthor(t, client, "John", "Doe")
	createTestAuthor(t, client, "Jane", "Doe")
	createTestAuthor(t, client, "John", "Smith")

	// Count authors with first_name = "John"
	count1, err := client.Authors.Count().
		Where(authors.FirstNameEQ("John")).
		Exec()

	if err != nil {
		t.Fatalf("Count with first_name failed: %v", err)
	}

	if count1 != 2 {
		t.Errorf("Expected count 2 (Johns), got %d", count1)
	}

	// Count authors with first_name = "John" AND last_name = "Doe"
	count2, err := client.Authors.Count().
		Where(authors.FirstNameEQ("John")).
		Where(authors.LastNameEQ("Doe")).
		Exec()

	if err != nil {
		t.Fatalf("Count with multiple conditions failed: %v", err)
	}

	if count2 != 1 {
		t.Errorf("Expected count 1 (John Doe), got %d", count2)
	}
}
