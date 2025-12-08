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

	"my-app/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Option 1: Setup from prisma.conf
// Reads DATABASE_URL from prisma.conf [datasource] url
client, pool, err := db.SetupClient(context.Background())
if err != nil {
	log.Fatal(err)
}
defer pool.Close()

// Option 2: Explicit URL parameter (overrides prisma.conf)
client, pool, err := db.SetupClient(context.Background(), "postgresql://user:pass@localhost/db")
if err != nil {
	log.Fatal(err)
}
defer pool.Close()

// Option 3: Manual setup with more control
databaseURL := "postgresql://user:pass@localhost/db"
pool, err := db.NewPgxPoolFromURL(context.Background(), databaseURL)
if err != nil {
	log.Fatal(err)
}
defer pool.Close()

dbDriver := db.NewPgxPoolDriver(pool)
client := db.NewClient(dbDriver)
```

## Fluent API

Each model has fluent builders accessible through the client.

### Available Builders

- `client.User.Create()` - Returns a Create builder
- `client.User.FindMany()` - Returns a FindMany builder
- `client.User.FindFirst()` - Returns a FindFirst builder
- `client.User.Update()` - Returns an Update builder
- `client.User.Delete()` - Returns a Delete builder

## CRUD Operations

### Create

```go
// Create a single record using fluent API
user, err := client.User.Create().
	Data(inputs.UserCreateInput{
		Email: "user@example.com",
		Name:  db.String("John Doe"),
	}).
	Exec(ctx)
```

### Read

```go
// Find first matching record with Select
user, err := client.User.FindFirst().
	Select(inputs.UserSelect{
		Email: true,
		Name:  true,
	}).
	Where(inputs.UserWhereInput{
		Email: db.String("user@example.com"),
	}).
	Exec(ctx)

// Find many records with Select and Where
users, err := client.User.FindMany().
	Select(inputs.UserSelect{
		Email: true,
		Name:  true,
	}).
	Where(inputs.UserWhereInput{
		Email: db.Contains("example.com"),
	}).
	Exec(ctx)
```

### Update

```go
// Update single record
err := client.User.Update().
	Where(inputs.UserWhereInput{
		Id: db.Int(1),
	}).
	Data(inputs.UserUpdateInput{
		Name: db.String("Updated Name"),
	}).
	Exec(ctx)
```

### Delete

```go
// Delete single record
err := client.User.Delete().
	Where(inputs.UserWhereInput{
		Id: db.Int(1),
	}).
	Exec(ctx)
```

## Query Options

### Where Clauses

```go
// Simple where
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		Email: db.String("user@example.com"),
	}).
	Exec(ctx)

// Multiple conditions (AND)
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		Email: db.String("user@example.com"),
		Name:  db.String("John"),
	}).
	Exec(ctx)

// OR conditions
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		Or: []inputs.UserWhereInput{
			{Email: db.String("user1@example.com")},
			{Email: db.String("user2@example.com")},
		},
	}).
	Exec(ctx)

// NOT conditions
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		Not: &inputs.UserWhereInput{
			Email: db.String("admin@example.com"),
		},
	}).
	Exec(ctx)
```

### Text Operators

```go
// Contains
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		Email: db.Contains("example"),
	}).
	Exec(ctx)

// Starts with
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		Email: db.StartsWith("user"),
	}).
	Exec(ctx)

// Ends with
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		Email: db.EndsWith(".com"),
	}).
	Exec(ctx)
```

### Comparison Operators

```go
// Greater than
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		Views: db.IntGt(100),
	}).
	Exec(ctx)

// Less than
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		Views: db.IntLt(10),
	}).
	Exec(ctx)

// Greater than or equal
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		Views: db.IntGte(100),
	}).
	Exec(ctx)

// Less than or equal
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		Views: db.IntLte(10),
	}).
	Exec(ctx)

// In array (using StringIn helper)
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		Id: db.Int(1), // For exact match, use Int()
	}).
	Exec(ctx)
```

### Ordering

```go
import (
	"my-app/db/builder"
	"my-app/db/models"
)

// Order by single field using the query builder directly
// Note: Order() is not available on Prisma-style builders (FindFirst/FindMany)
var users []models.User
err := client.User.
	Where(builder.Where{"email": builder.Contains("example.com")}).
	Order("created_at DESC").
	Find(ctx, &users)

// Order by multiple fields
err = client.User.
	Where(builder.Where{"active": true}).
	Order("created_at DESC").
	Order("name ASC").
	Find(ctx, &users)
```

### Pagination

```go
import (
	"my-app/db/builder"
	"my-app/db/models"
)

// Take and Skip are available on the query builder directly
// Note: Take() and Skip() are not available on Prisma-style builders (FindFirst/FindMany)
var users []models.User
err := client.User.
	Where(builder.Where{"email": builder.Contains("example.com")}).
	Take(10).
	Find(ctx, &users)

// Skip results
err = client.User.
	Where(builder.Where{"active": true}).
	Skip(20).
	Find(ctx, &users)

// Take and skip together (pagination)
page := 1
pageSize := 10
err = client.User.
	Where(builder.Where{"active": true}).
	Skip((page - 1) * pageSize).
	Take(pageSize).
	Find(ctx, &users)
```

### Selecting Fields

```go
// Select specific fields using type-safe Select
users, err := client.User.FindMany().
	Select(inputs.UserSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Exec(ctx)
```

### Custom Types with ExecTyped

The `ExecTyped()` method allows you to scan query results into custom DTOs (Data Transfer Objects) instead of the default generated models. This is useful when you need to return different structures to your API clients.

**Requirements:**

- Custom structs must have `json` or `db` tags for field mapping

**Example:**

```go
// Define a custom DTO
type UserDTO struct {
	ID    int    `json:"id" db:"id"`
	Email string `json:"email" db:"email"`
	Name  string `json:"name" db:"name"`
}

// Find first with custom DTO
var userDTO *UserDTO
err := client.User.FindFirst().
	Select(inputs.UserSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Where(inputs.UserWhereInput{
		Email: db.String("user@example.com"),
	}).
	ExecTyped(ctx, &userDTO)
if err != nil {
	log.Fatal(err)
}
// userDTO is now populated with the query results

// Find many with custom DTO
var usersDTO []UserDTO
err = client.User.FindMany().
	Select(inputs.UserSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Where(inputs.UserWhereInput{
		Email: db.Contains("example.com"),
	}).
	ExecTyped(ctx, &usersDTO)
if err != nil {
	log.Fatal(err)
}
// usersDTO is now populated with the query results
```

**Field Mapping:**

- Fields are mapped using `json` or `db` tags
- If a tag matches the database column name, the field will be populated
- Fields without matching tags are ignored
- Snake_case field names are automatically converted

**Note:** For single results, pass a pointer to a struct (e.g., `&userDTO`). For multiple results, pass a pointer to a slice (e.g., `&usersDTO`).

### Including Relations

```go
// Include related data
// Note: Include functionality will be added in a future version
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		// Add conditions
	}).
	Exec(ctx)

// Nested includes
// Note: Include functionality will be added in a future version
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		// Add conditions
	}).
	Exec(ctx)
```

## Aggregations

### Count

```go
// Count all
count, err := client.User.Count().
	Exec(ctx)

// Count with where
count, err := client.User.Count().
	Where(inputs.UserWhereInput{
		Email: db.Contains("@example.com"),
	}).
	Exec(ctx)
```

### Sum

```go
// Sum numeric field
// Note: Aggregation methods will be added in a future version
// For now, use raw SQL for aggregations
```

### Average

```go
// Average numeric field
// Note: Aggregation methods will be added in a future version
// For now, use raw SQL for aggregations
```

### Min/Max

```go
// Minimum/Maximum value
// Note: Aggregation methods will be added in a future version
// For now, use raw SQL for aggregations
```

### Group By

```go
// Group by field
// Note: GroupBy functionality will be added in a future version
// For now, use raw SQL for grouping
```

## Transactions

Transactions allow you to execute multiple operations atomically. If any operation fails, all changes are rolled back automatically.

### Basic Transaction

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	// Create user
	user, err := tx.User.Create().
		Data(inputs.UserCreateInput{
			Email: "user@example.com",
			Name:  db.String("John Doe"),
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Create post
	_, err = tx.Post.Create().
		Data(inputs.PostCreateInput{
			Title:    db.String("My Post"),
			AuthorId: db.String(user.ID),
		}).
		Exec(ctx)
	return err
})
```

### Transaction with Multiple Operations

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	// Create user
	user, err := tx.User.Create().
		Data(inputs.UserCreateInput{
			Email: "user@example.com",
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Update user
	err = tx.User.Update().
		Where(inputs.UserWhereInput{
			Id: db.String(user.ID),
		}).
		Data(inputs.UserUpdateInput{
			Name: db.String("Updated Name"),
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Create related records
	for _, postData := range posts {
		_, err = tx.Post.Create().
			Data(inputs.PostCreateInput{
				Title:    db.String(postData.Title),
				AuthorId: db.String(user.ID),
			}).
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
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	// Use fluent API
	user, err := tx.User.Create().
		Data(inputs.UserCreateInput{
			Email: "user@example.com",
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Use raw SQL within transaction
	_, err = tx.Raw().Exec(ctx, `
		UPDATE users
		SET last_login = NOW()
		WHERE id = $1
	`, user.ID)
	return err
})
```

### Error Handling in Transactions

If any operation returns an error, the transaction is automatically rolled back:

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	user, err := tx.User.Create().
		Data(inputs.UserCreateInput{
			Email: "user@example.com",
		}).
		Exec(ctx)
	if err != nil {
		return err // Transaction will be rolled back
	}

	// If this fails, the user creation above will be rolled back
	_, err = tx.Post.Create().
		Data(inputs.PostCreateInput{
			Title: db.String("Post"),
		}).
		Exec(ctx)
	return err
})

if err != nil {
	// Handle error - transaction was already rolled back
	log.Printf("Transaction failed: %v", err)
}
```

## Raw SQL

For complex queries, you can use raw SQL:

```go
// Query with results
rows, err := client.Raw().Query(ctx, `
	SELECT u.*, COUNT(p.id) as post_count
	FROM users u
	LEFT JOIN posts p ON p.author_id = u.id
	GROUP BY u.id
`)
if err != nil {
	log.Fatal(err)
}
defer rows.Close()

// Process rows
for rows.Next() {
	// Scan results
}

// Query single row
row := client.Raw().QueryRow(ctx, "SELECT COUNT(*) FROM users")
var count int
if err := row.Scan(&count); err != nil {
	log.Fatal(err)
}

// Execute (INSERT, UPDATE, DELETE)
result, err := client.Raw().Exec(ctx, "UPDATE users SET name = $1 WHERE id = $2", "New Name", userID)
if err != nil {
	log.Fatal(err)
}
rowsAffected := result.RowsAffected()
```

## Soft Deletes

If your model has `deletedAt` field:

```go
// Soft delete
err := client.User.Delete().
	Where(inputs.UserWhereInput{
		Id: db.Int(1),
	}).
	Exec(ctx)

// Restore
// Note: Restore functionality will be added in a future version

// Force delete (permanent)
// Note: ForceDelete functionality will be added in a future version

// Include deleted records
// Note: IncludeDeleted functionality will be added in a future version

// Only deleted records
// Note: OnlyDeleted functionality will be added in a future version
```

## JSON Fields

```go
// Set JSON field
err := client.User.Update().
	Where(inputs.UserWhereInput{
		Id: db.Int(1),
	}).
	Data(inputs.UserUpdateInput{
		// Add JSON field update when supported
	}).
	Exec(ctx)

// Get JSON field
// Note: JSON field access methods will be added in a future version

// Check if JSON contains key
// Note: JSON field access methods will be added in a future version
```

## Full-Text Search (PostgreSQL)

```go
// Search
// Note: Full-text search functionality will be added in a future version
// For now, use raw SQL for full-text search

// Search with ranking
// Note: Full-text search functionality will be added in a future version

// Search in where clause
// Note: Full-text search functionality will be added in a future version
```

## Validation

```go
// Validate struct
err := db.ValidateStruct(user)
if err != nil {
	// Handle validation errors
}
```

## Hooks/Middleware

```go
// Before create hook
// Note: Hooks functionality will be added in a future version

// After create hook
// Note: Hooks functionality will be added in a future version
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

```go
user, err := client.User.FindFirst().
	Where(inputs.UserWhereInput{
		Id: db.Int(1),
	}).
	Exec(ctx)
if err != nil {
	if errors.Is(err, db.ErrNotFound) {
		// Record not found
	} else if errors.Is(err, db.ErrUniqueConstraint) {
		// Unique constraint violation
	} else {
		// Other error
	}
}
```

## Best Practices

1. Always handle errors
2. Use transactions for multiple related operations
3. Use pagination for large datasets
4. Select only needed fields
5. Use indexes for frequently queried fields
6. Validate input data
7. Use soft deletes when appropriate
8. Monitor query performance
