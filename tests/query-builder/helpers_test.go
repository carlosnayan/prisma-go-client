//go:build postgresql
// +build postgresql

package querybuilder_test

import (
	"context"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
	prisma "github.com/carlosnayan/prisma-go-client/prisma/db"
	"github.com/carlosnayan/prisma-go-client/prisma/db/builder"
	"github.com/carlosnayan/prisma-go-client/prisma/db/inputs"
	"github.com/carlosnayan/prisma-go-client/prisma/db/models"
)

// dbAdapter adapts driver.DB to builder.DB interface
type dbAdapter struct {
	db driver.DB
}

func (a *dbAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error) {
	return a.db.Exec(ctx, sql, args...)
}

func (a *dbAdapter) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error) {
	return a.db.Query(ctx, sql, args...)
}

func (a *dbAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row {
	return a.db.QueryRow(ctx, sql, args...)
}

func (a *dbAdapter) Begin(ctx context.Context) (builder.Tx, error) {
	driverTx, err := a.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &txAdapter{tx: driverTx}, nil
}

func (a *dbAdapter) Close() {
	a.db.Close()
}

// txAdapter adapts driver.Tx to builder.Tx interface
type txAdapter struct {
	tx driver.Tx
}

func (t *txAdapter) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *txAdapter) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (t *txAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error) {
	return t.tx.Exec(ctx, sql, args...)
}

func (t *txAdapter) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t *txAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

// setupTestClient sets up a PostgreSQL test database and returns a client
func setupTestClient(t *testing.T) (*prisma.Client, func()) {
	t.Helper()

	testutil.SkipIfNoDatabase(t, "postgresql")
	dbConn, cleanup := testutil.SetupTestDB(t, "postgresql")

	// Wrap driver.DB with adapter to satisfy builder.DBTX interface
	adapter := &dbAdapter{db: dbConn}

	// Create Prisma client using the adapted database connection
	client := prisma.NewClient(adapter)

	return client, func() {
		client.Close()
		cleanup()
	}
}

// cleanupAuthors removes all test authors from the database
func cleanupAuthors(t *testing.T, client *prisma.Client) {
	t.Helper()

	err := client.Authors.DeleteMany().Exec()
	if err != nil {
		t.Logf("Warning: failed to cleanup authors: %v", err)
	}
}

// cleanupBooks removes all test books from the database
func cleanupBooks(t *testing.T, client *prisma.Client) {
	t.Helper()

	err := client.Books.DeleteMany().Exec()
	if err != nil {
		t.Logf("Warning: failed to cleanup books: %v", err)
	}
}

// createTestAuthor creates a test author with the given email
func createTestAuthor(t *testing.T, client *prisma.Client, firstName, lastName string) *models.Authors {
	t.Helper()

	author, err := client.Authors.Create().
		SetFirstName(firstName).
		SetLastName(lastName).
		Exec()

	if err != nil {
		t.Fatalf("Failed to create test author: %v", err)
	}

	return author
}

// createTestAuthorWithEmail creates a test author with email
func createTestAuthorWithEmail(t *testing.T, client *prisma.Client, firstName, lastName, email string) *models.Authors {
	t.Helper()

	emailPtr := &email
	author, err := client.Authors.Create().
		SetFirstName(firstName).
		SetLastName(lastName).
		SetEmail(emailPtr).
		Exec()

	if err != nil {
		t.Fatalf("Failed to create test author with email: %v", err)
	}

	return author
}

// createTestAuthors creates multiple test authors
func createTestAuthors(t *testing.T, client *prisma.Client, count int) []*models.Authors {
	t.Helper()

	authors := make([]*models.Authors, 0, count)
	for i := 1; i <= count; i++ {
		author := createTestAuthor(t, client, "Author", string(rune('A'+i-1)))
		authors = append(authors, author)
	}

	return authors
}

// createTestAuthorsBatch creates multiple authors using CreateMany
func createTestAuthorsBatch(t *testing.T, client *prisma.Client, data []inputs.AuthorsCreateManyArgs) *builder.BatchPayload {
	t.Helper()

	result, err := client.Authors.CreateMany().
		Data(data).
		Exec()

	if err != nil {
		t.Fatalf("Failed to create authors batch: %v", err)
	}

	return result
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

// intPtr returns a pointer to the given int
func intPtr(i int) *int {
	return &i
}
