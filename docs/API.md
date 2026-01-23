# API Reference

Complete reference for Prisma for Go API.

## Client

The Prisma client is the main entry point for all database operations.

### Creating a Client

```go
import (
	"context"
	"log"
	"os"
	"time"

	prisma "my-app/prisma/generated"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Option 1: Setup from prisma.conf
// Reads DATABASE_URL from prisma.conf [datasource] url
client, pool, err := prisma.SetupClient(context.Background())
if err != nil {
	log.Fatal(err)
}
defer pool.Close()

// Option 2: Explicit URL parameter (overrides prisma.conf)
client, pool, err := prisma.SetupClient(context.Background(), "postgresql://user:pass@localhost/db")
if err != nil {
	log.Fatal(err)
}
defer pool.Close()

// Option 3: Manual setup with more control
databaseURL := "postgresql://user:pass@localhost/db"
pool, err := prisma.NewPgxPoolFromURL(context.Background(), databaseURL)
if err != nil {
	log.Fatal(err)
}
defer pool.Close()

dbDriver := prisma.NewPgxPoolDriver(pool)
client := prisma.NewClient(dbDriver)

// Option 4: SetupClientWithOptions - Programmatic pool configuration (Recommended for Production)
// Configure pool settings in code instead of relying on prisma.conf
opts := &prisma.ClientOptions{
	DatabaseURL: os.Getenv("DATABASE_URL"),
	Pool: &prisma.PoolOptions{
		MaxConns:          25,
		MinConns:          5,
		MaxConnLifetime:   30 * time.Minute,
		MaxIdleTime:       5 * time.Minute,  // PostgreSQL only
		HealthCheckPeriod: 1 * time.Minute,  // PostgreSQL only
		ConnectTimeout:    5 * time.Second,  // PostgreSQL only
	},
}

client, pool, err := prisma.SetupClientWithOptions(context.Background(), opts)
if err != nil {
	log.Fatal(err)
}
defer pool.Close()

// If Pool is nil, default settings are used
optsDefault := &prisma.ClientOptions{
	DatabaseURL: os.Getenv("DATABASE_URL"),
	// Pool is nil - uses defaults: MaxConns=25, MinConns=5, MaxConnLifetime=30min
}

client, pool, err = prisma.SetupClientWithOptions(context.Background(), optsDefault)
if err != nil {
	log.Fatal(err)
}
defer pool.Close()
```

### Pool Configuration Options

#### PostgreSQL

Supports all pool configuration options:

- `MaxConns` - Maximum number of connections (default: 25)
- `MinConns` - Minimum number of idle connections (default: 5)
- `MaxConnLifetime` - Maximum connection lifetime (default: 30 minutes)
- `MaxIdleTime` - Maximum idle time for a connection (default: 5 minutes)
- `HealthCheckPeriod` - Interval between health checks (default: 1 minute)
- `ConnectTimeout` - Timeout for establishing connections (default: 5 seconds)

#### MySQL / SQLite

Supports basic pool configuration options:

- `MaxConns` - Maximum number of connections
- `MinConns` - Minimum number of idle connections
- `MaxConnLifetime` - Maximum connection lifetime

**Note:** MaxIdleTime, HealthCheckPeriod, and ConnectTimeout are ignored for MySQL and SQLite.

## Fluent API

Each model has fluent builders accessible through the client using Prisma-like method names with Ent-like fluency.

### Available Methods

**Query Methods:**

- `client.Authors.FindFirst()` - Find first matching record
- `client.Authors.FindMany()` - Find multiple records
- `client.Authors.Count()` - Count matching records

**Mutation Methods:**

- `client.Authors.Create()` - Create a single record
- `client.Authors.CreateMany()` - Create multiple records
- `client.Authors.UpdateMany()` - Update multiple records
- `client.Authors.UpdateOneID(id)` - Update single record by ID
- `client.Authors.Delete()` - Delete matching records

### Context Management (Optional)

Context can be added optionally using `WithContext()`. If not provided, `context.Background()` is used:

```go
import (
    "context"
    prisma "my-app/prisma/generated"
    "my-app/prisma/generated/authors" // Table package for type-safe filters
)

// WithContext is optional
user, err := client.Authors.FindFirst().
    Where(authors.EmailEQ("test@example.com")).
    WithContext(ctx). // optional
    Exec()

// Without WithContext - uses context.Background()
user, err := client.Authors.FindFirst().
    Where(authors.EmailEQ("test@example.com")).
    Exec() // Uses context.Background()
```

## CRUD Operations

### Create

```go
// Create a single record using fluent API with Set methods
user, err := client.Authors.Create().
    SetEmail("author@example.com").
    SetFirstName("John").
    SetLastName("Doe").
    WithContext(ctx). // optional
    Exec()
```

#### Required Fields Validation

When creating records, all required fields must be provided. A field is considered required if:

- It is not optional (no `?` suffix in the Prisma schema)
- It does not have a `@default` value

If required fields are missing, a validation error is returned:

```go
// Missing required field 'email'
user, err := client.Authors.Create().
	SetFirstName("John").
	SetLastName("Doe").
	Exec(ctx)
// Error: validation error: required fields missing: Email
```

**Error Format:**

- Single field: `"validation error: required fields missing: FieldName"`
- Multiple fields: `"validation error: required fields missing: Field1, Field2"`

**Fields with Default Values:**
Fields with `@default` are not required, even if they are not optional:

```prisma
model User {
  id        Int    @id @default(autoincrement())
  email     String
  name      String
  status    String @default("active")  // Not required
  createdAt DateTime @default(now())  // Not required
}
```

**Optional Fields:**
Optional fields (with `?` suffix) can be omitted or set to `nil`:

```go
user, err := client.Authors.Create().
	SetEmail("author@example.com").
	SetName("John Doe").
	// Age is optional, can be omitted or set with SetAge(nil)
	Exec(ctx)
```

### CreateMany

Create multiple records in a single operation:

```go
// Create multiple records
result, err := client.Authors.CreateMany().
	Data([]inputs.AuthorsCreateInput{
		{Email: "user1@example.com", Name: "User 1", Bio: "Bio 1"},
		{Email: "user2@example.com", Name: "User 2", Bio: "Bio 2"},
	}).
	Exec(ctx)

// result.Count contains the number of records created
fmt.Printf("Created %d users\n", result.Count)
```

#### Required Fields Validation in CreateMany

The same validation rules apply to `CreateMany`. Each item in the data slice is validated before insertion:

```go
// Item with missing required field
result, err := client.Authors.CreateMany().
	Data([]inputs.AuthorsCreateInput{
		{Email: "user1@example.com", Name: "User 1", Bio: "Bio 1"}, // Valid
		{Email: "user2@example.com"}, // Missing 'name' and 'bio'
	}).
	Exec(ctx)
// Error: validation error: required fields missing in item 1: Name, Bio
```

**Error Format for CreateMany:**

- `"validation error: required fields missing in item {index}: Field1, Field2"`

The index is 0-based, so `item 0` is the first item, `item 1` is the second, etc.

**Skip Duplicates:**

You can skip duplicate records using `SkipDuplicates`:

```go
result, err := client.Authors.CreateMany().
	Data([]inputs.AuthorsCreateInput{
		{Email: prisma.String("author@example.com"), Name: "User", Bio: "Bio"},
		{Email: prisma.String("author@example.com"), Name: "User", Bio: "Bio"}, // Duplicate
	}).
	SkipDuplicates(true).
	Exec(ctx)
// Duplicate records are skipped (PostgreSQL: ON CONFLICT DO NOTHING, MySQL: ON DUPLICATE KEY UPDATE)
```

### Read

```go
import (
    prisma "my-app/prisma/generated"
    "my-app/prisma/generated/authors" // Table package for type-safe filters
)

// Find first matching record
user, err := client.Authors.FindFirst().
    Where(authors.EmailEQ("author@example.com")).
    WithContext(ctx). // optional
    Exec()

// Find many records with filters
users, err := client.Authors.FindMany().
    Where(authors.EmailContains("author")).
    Limit(10).
    WithContext(ctx). // optional
    Exec()

// Find with multiple conditions (AND)
users, err := client.Authors.FindMany().
    Where(authors.And(
        authors.EmailContains("author"),
        authors.ActiveEQ(true),
    )).
    Exec()

// Find with OR conditions
users, err := client.Authors.FindMany().
    Where(authors.Or(
        authors.EmailEQ("user1@example.com"),
        authors.EmailEQ("user2@example.com"),
    )).
    Exec()
```

### Update

```go
// UpdateMany - update multiple records
err := client.Authors.UpdateMany().
    Where(authors.StatusEQ("inactive")).
    SetActive(false).
    SetUpdatedAt(time.Now()).
    WithContext(ctx). // optional
    Exec()

// UpdateOneID - update single record by ID
err := client.Authors.UpdateOneID(123).
    SetBio("Updated biography").
    SetActive(true).
    WithContext(ctx). // optional
    Exec()
```

### Delete

```go
// Delete matching records
err := client.Authors.Delete().
    Where(authors.EmailEQ("old@example.com")).
    WithContext(ctx). // optional
    Exec()

// Delete with multiple conditions
err := client.Authors.Delete().
    Where(authors.And(
        authors.StatusEQ("inactive"),
        authors.EmailContains("temp"),
    )).
    Exec()
```

### Count

Count matching records:

```go
// Count all records
count, err := client.Authors.Count().
    WithContext(ctx). // optional
    Exec()

// Count with filter
count, err := client.Authors.Count().
    Where(authors.ActiveEQ(true)).
    Exec()

// Count with multiple conditions
count, err := client.Authors.Count().
    Where(authors.And(
        authors.EmailContains("@example.com"),
        authors.ActiveEQ(true),
    )).
    Exec()
```

## Query Options

### Where Clauses with Table Package Filters

The new API uses type-safe filters from table packages:

```go
import "my-app/db/authors"

// Simple equality
users, err := client.Authors.FindMany().
    Where(authors.EmailEQ("author@example.com")).
    Exec()

// Multiple conditions (AND)
users, err := client.Authors.FindMany().
    Where(authors.And(
        authors.EmailEQ("author@example.com"),
        authors.ActiveEQ(true),
    )).
    Exec()

// OR conditions
users, err := client.Authors.FindMany().
    Where(authors.Or(
        authors.EmailEQ("author1@example.com"),
        authors.EmailEQ("author2@example.com"),
    )).
    Exec()
```

### Text Operators

Table packages provide type-safe text filter methods:

```go
// Contains
users, err := client.Authors.FindMany().
    Where(authors.EmailContains("author")).
    Exec()

// Starts with
users, err := client.Authors.FindMany().
    Where(authors.FirstNameHasPrefix("John")).
    Exec()

// Ends with
users, err := client.Authors.FindMany().
    Where(authors.LastNameHasSuffix("son")).
    Exec()
```

### Comparison Operators

Table packages provide comparison operators for all field types:

```go
import "my-app/db/books"

// Greater than
books, err := client.Books.FindMany().
    Where(books.PageCountGT(100)).
    Exec()

// Less than
books, err := client.Books.FindMany().
    Where(books.PageCountLT(50)).
    Exec()

// Greater than or equal
books, err := client.Books.FindMany().
    Where(books.PageCountGTE(100)).
    Exec()

// Less than or equal
books, err := client.Books.FindMany().
    Where(books.PageCountLTE(500)).
    Exec()

// Not equal
books, err := client.Books.FindMany().
    Where(books.StatusNEQ("draft")).
    Exec()
```

### Ordering

```go
import "my-app/db/authors"

// Order by single field (descending)
users, err := client.Authors.FindMany().
    OrderBy(authors.FieldCreatedAt, authors.OrderDesc).
    Exec()

// Order by single field (ascending)
users, err := client.Authors.FindMany().
    OrderBy(authors.FieldEmail, authors.OrderAsc).
    Exec()
```

### Pagination

```go
// Take results
users, err := client.Authors.FindMany().
    Take(10).
    Exec()

// Skip results
users, err := client.Authors.FindMany().
    Skip(20).
    Exec()

// Take and skip (pagination)
page := 1
pageSize := 10
users, err := client.Authors.FindMany().
    Skip((page - 1) * pageSize).
    Take(pageSize).
    Exec()
```

#### Practical Pagination Example

Implementing page-based navigation with books:

```go
import (
    "context"
    prisma "my-app/prisma/generated"
    "my-app/prisma/generated/books"
)

// GetBooks returns a paginated list of published books
func GetBooks(page int, pageSize int) ([]models.Books, error) {
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 20 // default
    }

    books, err := client.Books.FindMany().
        Where(books.StatusEQ(books.EnumBookStatusPUBLISHED)).
        OrderBy(books.FieldPublicationDate, books.OrderDesc).
        Skip((page - 1) * pageSize).
        Take(pageSize).
        Exec()

    if err != nil {
        return nil, err
    }

    return books, nil
}

// Usage examples
books, _ := GetBooks(1, 20)  // First page, 20 items
books, _ := GetBooks(2, 20)  // Second page, 20 items
books, _ := GetBooks(5, 10)  // Fifth page, 10 items
```

### Selecting Fields

```go
import "my-app/db/authors"

// Select specific fields using table package constants
users, err := client.Authors.FindMany().
    Select(authors.FieldEmail, authors.FieldFirstName, authors.FieldLastName).
    Where(authors.ActiveEQ(true)).
    Exec()
```

### Custom Types with ExecTyped (Go 1.18+)

The `ExecTyped()` method allows you to scan query results into custom DTOs (Data Transfer Objects) instead of the default generated models. This is useful when you need to return different structures to your API clients.

**Requirements:**

- Go 1.18 or later (for generics support)
- Custom structs must have `json` or `db` tags for field mapping

**Example with explicit context:**

````go
// Define a custom DTO
type UserDTO struct {
	ID    int    `json:"id" db:"id"`
	Email string `json:"email" db:"email"`
	Name  string `json:"name" db:"name"`
}

// Find first with custom DTO
var userDTO *UserDTO
err := client.Authors.FindFirst().
	Select(inputs.AuthorsSelect{
- Fields are mapped using `json` or `db` tags
- If a tag matches the database column name, the field will be populated
- Fields without matching tags are ignored
- Snake_case field names are automatically converted

**Note:** Use `ExecTyped[*YourType]()` for single results and `ExecTyped[[]YourType]()` for multiple results.

### Advanced Query Operators

#### Join Operations

The query builder supports various types of SQL joins for complex queries:

```go
// Inner join
authors, err := client.Authors.
	InnerJoin("books", "books.author_id = authors.id").
	Where("books.status = ?", "PUBLISHED").
	Find(ctx, &authors)

// Left join
authors, err := client.Authors.
	LeftJoin("books", "books.author_id = authors.id").
	Find(ctx, &authors)

// Right join
authors, err := client.Authors.
	RightJoin("books", "books.author_id = authors.id").
	Find(ctx, &authors)
````

#### Group By and Having

Group results and apply conditions to grouped data:

```go
// Group by with having clause
results, err := client.Books.
	Select("author_id", "COUNT(*) as count").
	Group("author_id").
	Having("COUNT(*) > ?", 5).
	Find(ctx, &results)
```

#### Full-Text Search (PostgreSQL/MySQL)

> [!NOTE]
> Full-text search is available for PostgreSQL and MySQL databases with appropriate text search indexes.

```go
import (
	prisma "my-app/prisma/generated"
	"my-app/prisma/generated/filters"
	"my-app/prisma/generated/inputs"

// ✅ Setting values
author, err := client.Authors.Create().
    SetFirstName("Isaac").
    SetLastName("Asimov").
    SetBio("Science fiction writer").
    SetNationality("American").
    // Email is not set - will be NULL
    Exec(ctx)
```

#### Practical Example: User Profile Update

```go
// User wants to clear their phone number but keep email
func UpdateUserProfile(ctx context.Context, userID string, updates ProfileUpdates) error {
    // Build update with direct values
    updateBuilder := client.Authors.Update().
        Where(authors.IdAuthorEQ(userID)).
        SetFirstName(updates.FirstName)

    // Clear phone if user requested (passing nil for NULL)
    if updates.ClearPhone {
        updateBuilder = updateBuilder.SetPhone(nil)
    } else if updates.Phone != "" {
        updateBuilder = updateBuilder.SetPhone(&updates.Phone)
    }
    // If neither condition is true, phone Setter is omitted and field won't be modified

    return updateBuilder.Exec(ctx)
}
```

#### Best Practices

1.  **Use direct values for required fields**: Just pass the value directly
2.  **Use pointers for nullable fields**: Pass nil to clear/nullify a field, or a pointer to a value
3.  **Omit Setter calls to leave unchanged**: Don't call SetField() for fields you don't want to modify

```go
// ✅ Correct
SetFirstName("Isaac")
SetBio(nil) // Explicitly clear bio
```

### Including Relations

```go
// Include related data
posts, err := client.Books.FindMany().
	Include(prisma.BooksIncludeInput{
		Author: true,
	}).Exec()

// Nested includes
posts, err := client.Books.FindMany().
	Include(prisma.BooksIncludeInput{
		Author: prisma.AuthorsIncludeInput{
			Posts: true,
		},
	}).Exec()
```

## Aggregations

### Count

Count is the only aggregation currently implemented:

```go
// Count all records
count, err := client.Authors.Count(ctx)

// Count with where condition
count, err := client.Authors.Count(ctx, Where{"email": Contains("@example.com")})
```

> [!NOTE]
> Additional aggregation functions (Sum, Avg, Min, Max, GroupBy with aggregations) are planned but not yet implemented. For complex aggregations, use Raw SQL queries:
>
> ```go
> type AggResult struct {
>     Total int     `db:"total"`
>     AvgPrice float64 `db:"avg_price"`
>     MaxViews int     `db:"max_views"`
> }
> var result AggResult
> err := client.Raw().QueryRow(`
>     SELECT
>         COUNT(*) as total,
>         AVG(price) as avg_price,
>         MAX(views) as max_views
>     FROM books
>     WHERE deleted_at IS NULL
> `).Exec().Scan(&result)
> if err != nil {
>     log.Fatal(err)
> }
> fmt.Printf("Total: %d, Avg Price: %.2f, Max Views: %d\n",
>     result.Total, result.AvgPrice, result.MaxViews)
> ```

## Transactions

Transactions allow you to execute multiple operations atomically. If any operation fails, all changes are rolled back automatically.

### Basic Transaction

```go
err := client.Transaction(ctx, func(tx *prisma.TransactionClient) error {
	// Create author
	author, err := tx.Authors.Create().
		SetFirstName("John").
		SetLastName("Doe").
		Exec(ctx)
	if err != nil {
		return err
	}

	// Create book (would need book_authors junction in real scenario)
	_, err = tx.Books.Create().
		SetTitle("My Book").
		Exec(ctx)
	return err
})
```

### Transaction with Multiple Operations

```go
err := client.Transaction(ctx, func(tx *prisma.TransactionClient) error {
	// Create author
	author, err := tx.Authors.Create().
		SetFirstName("John").
		SetLastName("Doe").
		Exec(ctx)
	if err != nil {
		return err
	}

	// Update author
	err = tx.Authors.Update().
		Where(authors.IdAuthorEQ(author.IdAuthor)).
		SetFirstName("Updated Name").
		SetBio("Updated biography").
		Exec(ctx)
	if err != nil {
		return err
	}

	// Create related books
	for _, title := range []string{"Book 1", "Book 2", "Book 3"} {
		_, err = tx.Books.Create().
			SetTitle(title).
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
})
```

### Transaction with Raw SQL

```go
err := client.Transaction(ctx, func(tx *prisma.TransactionClient) error {
	// Use fluent API
	author, err := tx.Authors.Create().
		SetFirstName("John").
		SetLastName("Doe").
		Exec(ctx)
	if err != nil {
		return err
	}

	// Use raw SQL within transaction
	_, err = tx.Raw().Exec(`
		UPDATE authors
		SET updated_at = NOW()
		WHERE id_author = $1
	`, author.IdAuthor)
	return err
})
```

### Error Handling in Transactions

If any operation returns an error, the transaction is automatically rolled back:

```go
err := client.Transaction(ctx, func(tx *prisma.TransactionClient) error {
	author, err := tx.Authors.Create().
		SetFirstName("John").
		SetLastName("Doe").
		Exec(ctx)
	if err != nil {
		return err // Transaction will be rolled back
	}

	// If this fails, the author creation above will be rolled back
	_, err = tx.Books.Create().
		SetTitle("My Book").
		Exec(ctx)
	return err
})

if err != nil {
	// Handle error - transaction was already rolled back
	log.Printf("Transaction failed: %v", err)
}
```

## Raw SQL

For complex queries, you can use raw SQL with the fluent API.

> **Important:** Structs used with `Scan()` **must** have `db:"column_name"` tags to map columns correctly. The column name should match the final column name after any alias.

### Query Multiple Rows

`Query()` requires a **slice** destination for `Scan()`:

```go
type BookWithAuthor struct {
	IdBook    string `db:"id_book"`
	Title     string `db:"title"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
}

var books []BookWithAuthor
err := client.Raw().Query(`
	SELECT b.id_book, b.title, a.first_name, a.last_name
	FROM books b
	INNER JOIN book_authors ba ON b.id_book = ba.id_book
	INNER JOIN authors a ON ba.id_author = a.id_author
	WHERE b.status = $1 AND b.deleted_at IS NULL
	ORDER BY b.created_at DESC
`, "PUBLISHED").Scan(&books).Exec()
if err != nil {
	log.Fatal(err)
}
```

### Query Single Row

For single row queries, use `Query()` with a single struct or primitive:

```go
// With struct
type BookStats struct {
	TotalBooks     int `db:"total_books"`
	PublishedBooks int `db:"published_books"`
}

var stats BookStats
err := client.Raw().Query(`
	SELECT
		COUNT(*) as total_books,
		COUNT(*) FILTER (WHERE status = 'PUBLISHED') as published_books
	FROM books
	WHERE deleted_at IS NULL
`).Scan(&stats).Exec()

// With primitive
var count int
err = client.Raw().Query("SELECT COUNT(*) FROM authors WHERE deleted_at IS NULL").
	Scan(&count).
	Exec()
```

### Execute (DDL, DML Without Results)

For queries that don't return data (DDL, INSERT, UPDATE, DELETE without RETURNING), simply call `Exec()` without `Scan()`:

```go
// DDL
err := client.Raw().Query("CREATE INDEX idx_books_status ON books(status)").Exec()

// DELETE
err := client.Raw().Query("DELETE FROM books WHERE status = $1", "DRAFT").Exec()

// UPDATE
err := client.Raw().Query(`
	UPDATE books
	SET status = $1, updated_at = NOW()
	WHERE publication_date < NOW() - INTERVAL '1 year'
`, "ARCHIVED").Exec()

// INSERT
err := client.Raw().Query(`
	INSERT INTO authors (first_name, last_name, email)
	VALUES ($1, $2, $3)
`, "John", "Doe", "john@example.com").Exec()
```

### Execute with Result Metadata

Use `Exec()` directly on the executor for operations where you need the result metadata:

```go
result, err := client.Raw().Exec("UPDATE books SET status = $1 WHERE id_book = $2", "ARCHIVED", bookId)
if err != nil {
	log.Fatal(err)
}
rowsAffected := result.RowsAffected()
```

### Column Alias Handling

The `Scan()` function automatically handles various column naming patterns:

| SQL Column Expression             | `db` Tag to Use             |
| --------------------------------- | --------------------------- |
| `id_tenant`                       | `db:"id_tenant"`            |
| `cf.id_chatbot_flow`              | `db:"id_chatbot_flow"`      |
| `ik.name as integration_key_name` | `db:"integration_key_name"` |
| `COUNT(*) as total`               | `db:"total"`                |

## JSON Fields

```go
// Set JSON field
user, err := client.Authors.Update().
	Where(authors.IdAuthorEQ(authorID)).
	SetMetadata(prisma.JSON(map[string]interface{}{
		"key": "value",
	})).
	Exec(ctx)

// Get JSON field
metadata := user.Metadata.Get("key")

// Check if JSON contains key
hasKey := user.Metadata.Contains("key")
```

## Full-Text Search (PostgreSQL)

```go
// Search
posts, err := client.Books.Search("search term").Exec()

// Search with ranking
results, err := client.Books.SearchRanked("search term").Exec()

// Search in where clause
posts, err := client.Books.FindMany(
	inputs.BooksWhereInput{
		Content: prisma.StringSearch("term"),
	},
).Exec()
```

## Validation

```go
// Validate struct
err := prisma.ValidateStruct(user)
if err != nil {
	// Handle validation errors
}
```

## Hooks/Middleware

```go
// Before create hook
client.Authors.BeforeCreate(func(user *prisma.User) error {
	// Validate or modify before creation
	return nil
})

// After create hook
client.Authors.AfterCreate(func(user *prisma.User) error {
	// Send notification, log, etc.
	return nil
})
```

Available hooks:

- `BeforeCreate`
- `AfterCreate`
- `BeforeUpdate`
- `AfterUpdate`
- `BeforeDelete`
- `AfterDelete`
- `BeforeFind`
- `AfterFind`

## Error Handling

Prisma Go Client provides standardized error types following the official Prisma error codes.

### PrismaError Type

All database errors are wrapped in a `PrismaError` with a code and message:

```go
type PrismaError struct {
	Code    string // P1xxx (connection) or P2xxx (query)
	Message string
}

func (e *PrismaError) Error() string   // Returns message with cause
func (e *PrismaError) Unwrap() error   // Returns original driver error
```

### Error Codes

| Code  | Error                     | Description                   |
| ----- | ------------------------- | ----------------------------- |
| P2025 | `ErrNotFound`             | Record not found              |
| P2002 | `ErrUniqueConstraint`     | Unique constraint violation   |
| P2003 | `ErrForeignKeyConstraint` | Foreign key violation         |
| P2011 | `ErrNullConstraint`       | Not null constraint violation |
| P2010 | `ErrRawQueryFailed`       | Raw query execution failed    |
| P1001 | `ErrConnectionFailed`     | Database not reachable        |
| P1008 | `ErrTimeout`              | Operation timeout             |

### Checking Errors

```go
import "errors"

// Using helper functions (Recommended)
if prisma.IsNotFound(err) {
	// Record not found
}
if prisma.IsUniqueConstraint(err) {
	// Duplicate key
}

// Using errors.Is
if errors.Is(err, prisma.ErrNotFound) {
	// Record not found
}

// Using error code
var prismaErr *prisma.PrismaError
if errors.As(err, &prismaErr) {
	switch prismaErr.Code {
	case "P2002":
		// Unique constraint
	case "P2025":
		// Not found
	}
}
```

> [!IMPORTANT]
> Always use `errors.Is()` or helper functions like `IsNotFound()`. Do **not** use direct comparison `==` because errors are wrapped with the original driver cause and will not match exactly.

### Getting Original Error

Use `errors.Unwrap()` to access the original driver error:

```go
if raw.IsUniqueConstraint(err) {
	originalErr := errors.Unwrap(err)
	log.Printf("Driver error: %v", originalErr)
}
```

### Query vs QueryRow Behavior

| Operation              | No Rows Found | Returns       |
| ---------------------- | ------------- | ------------- |
| `Query().Scan(&slice)` | Empty slice   | `nil` error   |
| `QueryRow().Scan()`    | ErrNotFound   | `P2025` error |
| `FindMany()`           | Empty slice   | `nil` error   |
| `FindFirst()`          | ErrNotFound   | `P2025` error |

```go
// Query returns empty slice, no error
var books []Book
err := client.Raw().Query("SELECT * FROM books WHERE id = $1", 999).Scan(&books).Exec()
// err == nil, books == []Book{}

// Query with single struct returns error when no rows found
var book Book
err = client.Raw().Query("SELECT * FROM books WHERE id = $1", 999).Scan(&book).Exec()
// err != nil ("no rows found")
```

## Best Practices

1. Always handle errors
2. Use transactions for multiple related operations
3. Use pagination for large datasets
4. Select only needed fields
5. Use indexes for frequently queried fields
6. Validate input data
7. Monitor query performance
