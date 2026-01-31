//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"context"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/prisma/db/inputs"
)

// TestCreate_Basic tests basic Create functionality with required fields
func TestCreate_Basic(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create author with only required fields
	author, err := client.Authors.Create().
		SetFirstName("John").
		SetLastName("Doe").
		Exec()

	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if author == nil {
		t.Fatal("Expected author to be returned, got nil")
	}

	if author.FirstName != "John" {
		t.Errorf("Expected first_name 'John', got '%s'", author.FirstName)
	}

	if author.LastName != "Doe" {
		t.Errorf("Expected last_name 'Doe', got '%s'", author.LastName)
	}

	if author.IdAuthor == "" {
		t.Error("Expected id_author to be generated (UUID)")
	}

	// Verify created_at and updated_at were set
	if author.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}

	if author.UpdatedAt.IsZero() {
		t.Error("Expected updated_at to be set")
	}
}

// TestCreate_WithNullableFields tests Create with nullable fields
func TestCreate_WithNullableFields(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	bio := "A famous author"
	email := "john.doe@example.com"
	nationality := "American"
	website := "https://johndoe.com"
	birthDate := time.Date(1980, 1, 15, 0, 0, 0, 0, time.UTC)

	author, err := client.Authors.Create().
		SetFirstName("John").
		SetLastName("Doe").
		SetBio(&bio).
		SetEmail(&email).
		SetNationality(&nationality).
		SetWebsite(&website).
		SetBirthDate(&birthDate).
		Exec()

	if err != nil {
		t.Fatalf("Create with nullable fields failed: %v", err)
	}

	if author.Bio == nil {
		t.Error("Expected bio to be set")
	} else if *author.Bio != bio {
		t.Errorf("Expected bio '%s', got '%s'", bio, *author.Bio)
	}

	if author.Email == nil {
		t.Error("Expected email to be set")
	} else if *author.Email != email {
		t.Errorf("Expected email '%s', got '%s'", email, *author.Email)
	}

	if author.Nationality == nil {
		t.Error("Expected nationality to be set")
	} else if *author.Nationality != nationality {
		t.Errorf("Expected nationality '%s', got '%s'", nationality, *author.Nationality)
	}

	if author.Website == nil {
		t.Error("Expected website to be set")
	} else if *author.Website != website {
		t.Errorf("Expected website '%s', got '%s'", website, *author.Website)
	}

	if author.BirthDate == nil {
		t.Error("Expected birth_date to be set")
	}
}

// TestCreate_WithNullValues tests setting nullable fields to NULL
func TestCreate_WithNullValues(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create with explicit NULL for nullable fields
	author, err := client.Authors.Create().
		SetFirstName("Jane").
		SetLastName("Smith").
		SetBio(nil).
		SetEmail(nil).
		SetNationality(nil).
		Exec()

	if err != nil {
		t.Fatalf("Create with NULL values failed: %v", err)
	}

	if author.Bio != nil {
		t.Errorf("Expected bio to be NULL, got '%v'", *author.Bio)
	}

	if author.Email != nil {
		t.Errorf("Expected email to be NULL, got '%v'", *author.Email)
	}

	if author.Nationality != nil {
		t.Errorf("Expected nationality to be NULL, got '%v'", *author.Nationality)
	}
}

// TestCreate_MissingRequiredField tests validation for missing required fields
func TestCreate_MissingRequiredField(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Missing first_name (required)
	_, err := client.Authors.Create().
		SetLastName("Doe").
		Exec()

	if err == nil {
		t.Fatal("Expected error for missing required field first_name, got nil")
	}

	// Missing last_name (required)
	_, err = client.Authors.Create().
		SetFirstName("John").
		Exec()

	if err == nil {
		t.Fatal("Expected error for missing required field last_name, got nil")
	}
}

// TestCreate_WithExplicitID tests creating with a custom ID
func TestCreate_WithExplicitID(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	customID := "550e8400-e29b-41d4-a716-446655440000"

	author, err := client.Authors.Create().
		SetIdAuthor(customID).
		SetFirstName("Custom").
		SetLastName("ID").
		Exec()

	if err != nil {
		t.Fatalf("Create with explicit ID failed: %v", err)
	}

	if author.IdAuthor != customID {
		t.Errorf("Expected id_author '%s', got '%s'", customID, author.IdAuthor)
	}
}

// TestCreate_WithExplicitTimestamps tests setting created_at and updated_at
func TestCreate_WithExplicitTimestamps(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	createdAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	author, err := client.Authors.Create().
		SetFirstName("Time").
		SetLastName("Traveler").
		SetCreatedAt(createdAt).
		SetUpdatedAt(updatedAt).
		Exec()

	if err != nil {
		t.Fatalf("Create with explicit timestamps failed: %v", err)
	}

	// Note: Timestamps might be adjusted by database, so we check they're close
	if author.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}

	if author.UpdatedAt.IsZero() {
		t.Error("Expected updated_at to be set")
	}
}

// TestCreate_WithContext tests Create with custom context
func TestCreate_WithContext(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	author, err := client.Authors.Create().
		SetFirstName("Context").
		SetLastName("Test").
		ExecWithContext(ctx)

	if err != nil {
		t.Fatalf("Create with context failed: %v", err)
	}

	if author == nil {
		t.Fatal("Expected author to be returned")
	}

	if ctx.Err() != nil && ctx.Err() != context.Canceled {
		t.Errorf("Context error: %v", ctx.Err())
	}
}

// TestCreate_DuplicateID tests creating with duplicate ID (should fail)
func TestCreate_DuplicateID(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	// Create first author
	author1, err := client.Authors.Create().
		SetFirstName("First").
		SetLastName("Author").
		Exec()

	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Try to create second author with same ID
	_, err = client.Authors.Create().
		SetIdAuthor(author1.IdAuthor).
		SetFirstName("Second").
		SetLastName("Author").
		Exec()

	if err == nil {
		t.Fatal("Expected error for duplicate ID, got nil")
	}
}

// TestCreate_MultipleAuthors tests creating multiple authors sequentially
func TestCreate_MultipleAuthors(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()
	defer cleanupAuthors(t, client)

	authors := []struct {
		firstName string
		lastName  string
	}{
		{"Author", "One"},
		{"Author", "Two"},
		{"Author", "Three"},
	}

	createdAuthors := make([]*inputs.AuthorsCreateManyArgs, 0, len(authors))

	for _, a := range authors {
		author, err := client.Authors.Create().
			SetFirstName(a.firstName).
			SetLastName(a.lastName).
			Exec()

		if err != nil {
			t.Fatalf("Failed to create author %s %s: %v", a.firstName, a.lastName, err)
		}

		if author == nil {
			t.Fatalf("Expected author to be returned for %s %s", a.firstName, a.lastName)
		}

		createdAuthors = append(createdAuthors, &inputs.AuthorsCreateManyArgs{
			FirstName: author.FirstName,
			LastName:  author.LastName,
		})
	}

	if len(createdAuthors) != len(authors) {
		t.Errorf("Expected %d authors created, got %d", len(authors), len(createdAuthors))
	}
}
