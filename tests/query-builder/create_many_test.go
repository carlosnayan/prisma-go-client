//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/prisma/db/inputs"
)

// TestCreateMany_Basic tests basic CreateMany functionality
func TestCreateMany_Basic(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	data := []inputs.AuthorsCreateManyArgs{
		{
			FirstName: "Author",
			LastName:  "One",
		},
		{
			FirstName: "Author",
			LastName:  "Two",
		},
		{
			FirstName: "Author",
			LastName:  "Three",
		},
	}

	result, err := client.Authors.CreateMany().
		Data(data).
		Exec()

	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected BatchPayload to be returned, got nil")
	}

	if result.Count != len(data) {
		t.Errorf("Expected count %d, got %d", len(data), result.Count)
	}
}

// TestCreateMany_WithNullableFields tests CreateMany with nullable fields
func TestCreateMany_WithNullableFields(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	email1 := "author1@example.com"
	email2 := "author2@example.com"

	data := []inputs.AuthorsCreateManyArgs{
		{
			FirstName: "Author",
			LastName:  "One",
			Email:     &email1,
		},
		{
			FirstName: "Author",
			LastName:  "Two",
			Email:     &email2,
		},
	}

	result, err := client.Authors.CreateMany().
		Data(data).
		Exec()

	if err != nil {
		t.Fatalf("CreateMany with nullable fields failed: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Expected count 2, got %d", result.Count)
	}

	// Verify the records were created
	authors, err := client.Authors.FindMany().Exec()
	if err != nil {
		t.Fatalf("Failed to verify created records: %v", err)
	}

	if len(authors) != 2 {
		t.Errorf("Expected 2 authors in database, got %d", len(authors))
	}
}

// TestCreateMany_EmptyData tests CreateMany with empty data
func TestCreateMany_EmptyData(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	result, err := client.Authors.CreateMany().
		Data([]inputs.AuthorsCreateManyArgs{}).
		Exec()

	if err != nil {
		t.Fatalf("CreateMany with empty data failed: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Expected count 0 for empty data, got %d", result.Count)
	}
}

// TestCreateMany_NilData tests CreateMany with nil data
func TestCreateMany_NilData(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	result, err := client.Authors.CreateMany().
		Data(nil).
		Exec()

	if err != nil {
		t.Fatalf("CreateMany with nil data failed: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Expected count 0 for nil data, got %d", result.Count)
	}
}

// TestCreateMany_MissingRequiredField tests validation for missing required fields
func TestCreateMany_MissingRequiredField(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Missing last_name in first item
	data := []inputs.AuthorsCreateManyArgs{
		{
			FirstName: "Author",
			// LastName is missing
		},
		{
			FirstName: "Author",
			LastName:  "Two",
		},
	}

	_, err := client.Authors.CreateMany().
		Data(data).
		Exec()

	if err == nil {
		t.Fatal("Expected error for missing required field, got nil")
	}
}

// TestCreateMany_MixedValidInvalid tests CreateMany with mixed valid/invalid data
func TestCreateMany_MixedValidInvalid(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	data := []inputs.AuthorsCreateManyArgs{
		{
			FirstName: "Valid",
			LastName:  "Author",
		},
		{
			FirstName: "Invalid",
			// Missing last_name
		},
	}

	_, err := client.Authors.CreateMany().
		Data(data).
		Exec()

	if err == nil {
		t.Fatal("Expected error for invalid data in batch, got nil")
	}

	// Verify no records were created (transaction rollback)
	authors, _ := client.Authors.FindMany().Exec()
	if len(authors) > 0 {
		t.Errorf("Expected 0 authors after failed CreateMany, got %d", len(authors))
	}
}

// TestCreateMany_SkipDuplicates tests SkipDuplicates flag
func TestCreateMany_SkipDuplicates(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create first author
	customID := "550e8400-e29b-41d4-a716-446655440000"
	_, err := client.Authors.Create().
		SetIdAuthor(customID).
		SetFirstName("Existing").
		SetLastName("Author").
		Exec()

	if err != nil {
		t.Fatalf("Failed to create initial author: %v", err)
	}

	// Try to create batch with duplicate ID
	data := []inputs.AuthorsCreateManyArgs{
		{
			IdAuthor:  customID, // Duplicate
			FirstName: "Duplicate",
			LastName:  "Author",
		},
		{
			FirstName: "New",
			LastName:  "Author",
		},
	}

	result, err := client.Authors.CreateMany().
		Data(data).
		SkipDuplicates(true).
		Exec()

	if err != nil {
		t.Fatalf("CreateMany with SkipDuplicates failed: %v", err)
	}

	// Should skip duplicate and create only the new one
	if result.Count != 1 {
		t.Errorf("Expected count 1 (skipped duplicate), got %d", result.Count)
	}
}

// TestCreateMany_WithoutSkipDuplicates tests CreateMany fails on duplicate without flag
func TestCreateMany_WithoutSkipDuplicates(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create first author
	customID := "550e8400-e29b-41d4-a716-446655440001"
	_, err := client.Authors.Create().
		SetIdAuthor(customID).
		SetFirstName("Existing").
		SetLastName("Author").
		Exec()

	if err != nil {
		t.Fatalf("Failed to create initial author: %v", err)
	}

	// Try to create batch with duplicate ID (without SkipDuplicates)
	data := []inputs.AuthorsCreateManyArgs{
		{
			IdAuthor:  customID, // Duplicate
			FirstName: "Duplicate",
			LastName:  "Author",
		},
	}

	_, err = client.Authors.CreateMany().
		Data(data).
		SkipDuplicates(false).
		Exec()

	if err == nil {
		t.Fatal("Expected error for duplicate without SkipDuplicates, got nil")
	}
}

// TestCreateMany_LargeBatch tests creating a large batch of records
func TestCreateMany_LargeBatch(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	batchSize := 100
	data := make([]inputs.AuthorsCreateManyArgs, batchSize)

	for i := 0; i < batchSize; i++ {
		data[i] = inputs.AuthorsCreateManyArgs{
			FirstName: "Author",
			LastName:  string(rune('A' + (i % 26))),
		}
	}

	result, err := client.Authors.CreateMany().
		Data(data).
		Exec()

	if err != nil {
		t.Fatalf("CreateMany large batch failed: %v", err)
	}

	if result.Count != batchSize {
		t.Errorf("Expected count %d, got %d", batchSize, result.Count)
	}

	// Verify all records were created
	authors, err := client.Authors.FindMany().Exec()
	if err != nil {
		t.Fatalf("Failed to verify created records: %v", err)
	}

	if len(authors) != batchSize {
		t.Errorf("Expected %d authors in database, got %d", batchSize, len(authors))
	}
}

// TestCreateMany_WithDefaults tests CreateMany respects default values
func TestCreateMany_WithDefaults(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	data := []inputs.AuthorsCreateManyArgs{
		{
			FirstName: "Author",
			LastName:  "One",
			// created_at and updated_at should get defaults
		},
	}

	result, err := client.Authors.CreateMany().
		Data(data).
		Exec()

	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}

	if result.Count != 1 {
		t.Errorf("Expected count 1, got %d", result.Count)
	}

	// Verify defaults were applied
	authors, err := client.Authors.FindMany().Exec()
	if err != nil {
		t.Fatalf("Failed to verify created records: %v", err)
	}

	if len(authors) == 0 {
		t.Fatal("No authors found")
	}

	author := authors[0]
	if author.CreatedAt.IsZero() {
		t.Error("Expected created_at default to be set")
	}

	if author.UpdatedAt.IsZero() {
		t.Error("Expected updated_at default to be set")
	}
}
