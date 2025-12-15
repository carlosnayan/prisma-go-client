# Examples

Simple, practical examples to help you get started with Prisma for Go.

## Query Builder Examples

### Context Management

You can set a context once and reuse it for multiple operations:

```go
import (
    "context"
    "my-app/db"
    "my-app/db/inputs"
    "github.com/jackc/pgx/v5/pgxpool"
)

ctx := context.Background()
pool, _ := db.NewPgxPoolFromURL(ctx, databaseURL)
dbDriver := db.NewPgxPoolDriver(pool)
client := db.NewClient(dbDriver)

// Set context once
query := client.User.WithContext(ctx)

// Use Exec() without passing context explicitly
user, err := query.Create().
    Data(inputs.UserCreateInput{
        Email: "user@example.com",
        Name:  db.String("John Doe"),
    }).
    Exec() // Uses stored context

// Explicit context still works and takes priority
user, err = query.Create().
    Data(inputs.UserCreateInput{
        Email: "user2@example.com",
    }).
    ExecWithContext(otherCtx) // Uses otherCtx instead
```

### Create a Record

```go
import (
    "context"
    "my-app/db"
    "my-app/db/inputs"
    "github.com/jackc/pgx/v5/pgxpool"
)

ctx := context.Background()
pool, _ := db.NewPgxPoolFromURL(ctx, databaseURL)
dbDriver := db.NewPgxPoolDriver(pool)
client := db.NewClient(dbDriver)

// Create a user (with explicit context)
user, err := client.User.Create().
    Data(inputs.UserCreateInput{
        Email: "user@example.com",
        Name:  db.String("John Doe"),
    }).
    ExecWithContext(ctx)

// Or using WithContext()
query := client.User.WithContext(ctx)
user, err = query.Create().
    Data(inputs.UserCreateInput{
        Email: "user@example.com",
        Name:  db.String("John Doe"),
    }).
    Exec() // Uses stored context
```

### Handling Validation Errors

When creating records, required fields must be provided. Handle validation errors appropriately:

```go
import (
    "context"
    "errors"
    "fmt"
    "my-app/db"
    "my-app/db/inputs"
    "strings"
)

ctx := context.Background()
pool, _ := db.NewPgxPoolFromURL(ctx, databaseURL)
dbDriver := db.NewPgxPoolDriver(pool)
client := db.NewClient(dbDriver)

// Attempt to create user with missing required field
user, err := client.User.Create().
    Data(inputs.UserCreateInput{
        Name: "John Doe",
        // Missing required 'email' field
    }).
    ExecWithContext(ctx)

if err != nil {
    // Check if it's a validation error
    errMsg := err.Error()
    if strings.Contains(errMsg, "validation error: required fields missing") {
        fmt.Printf("Validation failed: %s\n", errMsg)
        // Handle validation error (e.g., show to user, log, etc.)
    } else {
        // Handle other errors (database errors, etc.)
        fmt.Printf("Unexpected error: %v\n", err)
    }
}
```

**Example: CreateMany with Validation**

```go
// Create multiple users with validation
result, err := client.User.CreateMany().
    Data([]inputs.UserCreateInput{
        {Email: "user1@example.com", Name: "User 1", Bio: "Bio 1"}, // Valid
        {Email: "user2@example.com"}, // Missing required fields
        {Email: "user3@example.com", Name: "User 3", Bio: "Bio 3"}, // Valid
    }).
    ExecWithContext(ctx)

if err != nil {
    errMsg := err.Error()
    if strings.Contains(errMsg, "validation error: required fields missing in item") {
        // Extract item index from error message
        fmt.Printf("Validation failed: %s\n", errMsg)
        // Error format: "validation error: required fields missing in item 1: Name, Bio"
    } else {
        fmt.Printf("Unexpected error: %v\n", err)
    }
    return
}

fmt.Printf("Successfully created %d users\n", result.Count)
```

### Find Records

```go
// Find first matching record
user, err := client.User.FindFirst().
    Select(inputs.UserSelect{
        Email: true,
        Name:  true,
    }).
    Where(inputs.UserWhereInput{
        Email: db.String("user@example.com"),
    }).
    Exec(ctx)

// Find all matching records
users, err := client.User.FindMany().
    Select(inputs.UserSelect{
        Email: true,
        Name:  true,
    }).
    Where(inputs.UserWhereInput{
        Name: db.Contains("John"),
    }).
    Exec(ctx)
```

### Update a Record

```go
err := client.User.Update().
    Where(inputs.UserWhereInput{
        Id: db.Int(user.ID),
    }).
    Data(inputs.UserUpdateInput{
        Name: db.String("Jane Doe"),
    }).
    Exec(ctx)
```

### Delete a Record

```go
err := client.User.Delete().
    Where(inputs.UserWhereInput{
        Id: db.Int(user.ID),
    }).
    Exec(ctx)
```

### Advanced Queries

```go
// With conditions and select (explicit context)
users, err := client.User.FindMany().
    Select(inputs.UserSelect{
        Email: true,
        Name:  true,
    }).
    Where(inputs.UserWhereInput{
        Name:  db.Contains("John"),
        Email: db.EndsWith("@example.com"),
    }).
    ExecWithContext(ctx)

// Or using WithContext()
query := client.User.WithContext(ctx)
users, err = query.FindMany().
    Select(inputs.UserSelect{
        Email: true,
        Name:  true,
    }).
    Where(inputs.UserWhereInput{
        Name:  db.Contains("John"),
        Email: db.EndsWith("@example.com"),
    }).
    Exec() // Uses stored context
```

### Custom Types with ExecTyped

You can scan query results into custom DTOs (Data Transfer Objects) instead of the default generated models:

```go
// Define a custom DTO
type UserDTO struct {
	ID    int    `json:"id" db:"id"`
	Email string `json:"email" db:"email"`
	Name  string `json:"name" db:"name"`
}

// Find first with custom DTO (explicit context)
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
	ExecTypedWithContext(ctx, &userDTO)

// Find many with custom DTO using WithContext()
query := client.User.WithContext(ctx)
var usersDTO []UserDTO
err = query.FindMany().
	Select(inputs.UserSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Where(inputs.UserWhereInput{
		Email: db.Contains("example.com"),
	}).
	ExecTyped(&usersDTO) // Uses stored context
```

## Advanced Query Examples

### Using Advanced WHERE Operators

#### Full-Text Search

PostgreSQL and MySQL support full-text search:

```go
import (
	"context"
	"my-app/db"
	"my-app/db/filters"
	"my-app/db/inputs"
)

// Basic full-text search
articles, err := client.Articles.FindMany().
	Where(inputs.ArticlesWhereInput{
		Content: filters.FullTextSearch("postgresql optimization"),
	}).
	Exec(ctx)

// Language-specific search (PostgreSQL)
articles, err = client.Articles.FindMany().
	Where(inputs.ArticlesWhereInput{
		Content: filters.FullTextSearchConfig(map[string]interface{}{
			"query": "otimização & performance",
			"config": "portuguese",
		}),
	}).
	Exec(ctx)
```

#### JSON Array Operations

Working with JSON array fields:

```go
// Find records where array contains specific value
admins, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Roles: filters.Has("admin"),
	}).
	Exec(ctx)

// Array contains all specified values
powerUsers, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Roles: filters.HasEvery([]interface{}{"admin", "editor"}),
	}).
	Exec(ctx)

// Array contains any of the values
moderators, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Roles: filters.HasSome([]interface{}{"admin", "moderator", "editor"}),
	}).
	Exec(ctx)

// Empty array check
unassigned, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Roles: filters.IsEmpty(),
	}).
	Exec(ctx)
```

### Join and Group By

Complex queries using joins and grouping:

```go
// Count books per author using Raw SQL
type AuthorBookCount struct {
	AuthorID   string `db:"author_id"`
	AuthorName string `db:"author_name"`
	BookCount  int    `db:"book_count"`
}

var results []AuthorBookCount
err := client.Raw().Query(`
	SELECT
		a.id_author as author_id,
		a.first_name || ' ' || a.last_name as author_name,
		COUNT(ba.id_book) as book_count
	FROM authors a
	LEFT JOIN book_authors ba ON a.id_author = ba.id_author
	GROUP BY a.id_author, a.first_name, a.last_name
	HAVING COUNT(ba.id_book) > $1
	ORDER BY book_count DESC
`, 5).Exec().Scan(&results)

if err != nil {
	log.Fatal(err)
}

// Print results
for _, result := range results {
	fmt.Printf("%s: %d books\n", result.AuthorName, result.BookCount)
}
```

## Raw SQL Examples

Raw SQL provides full flexibility when you need complex queries, JOINs, or database-specific features.

> **Important:** Structs used with `Scan()` must have `db:"column_name"` tags. `Query()` requires a slice, `QueryRow()` requires a struct or primitive.

### Query Multiple Rows (slice)

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
`, "PUBLISHED").Exec().Scan(&books)
if err != nil {
    log.Fatal(err)
}

for _, b := range books {
    fmt.Printf("Book: %s by %s %s\n", b.Title, b.FirstName, b.LastName)
}
```

### Query Single Row (struct or primitive)

```go
// With struct
type BookStats struct {
    TotalBooks     int `db:"total_books"`
    PublishedBooks int `db:"published_books"`
    AvgPageCount   int `db:"avg_page_count"`
}

var stats BookStats
err := client.Raw().QueryRow(`
    SELECT
        COUNT(*) as total_books,
        COUNT(*) FILTER (WHERE status = 'PUBLISHED') as published_books,
        COALESCE(AVG(page_count), 0)::int as avg_page_count
    FROM books
    WHERE deleted_at IS NULL
`).Exec().Scan(&stats)

// With primitive
var count int
err = client.Raw().QueryRow("SELECT COUNT(*) FROM authors WHERE deleted_at IS NULL").
    Exec().
    Scan(&count)
```

### Manual Row Iteration

When you need more control, use `.Rows()` for manual iteration:

```go
rows, err := client.Raw().Query(
    "SELECT id_author, first_name, last_name FROM authors WHERE nationality = $1",
    "Brazilian",
).Exec().Rows()
if err != nil {
    log.Fatal(err)
}
defer rows.Close()

for rows.Next() {
    var id, firstName, lastName string
    if err := rows.Scan(&id, &firstName, &lastName); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Author: %s %s\n", firstName, lastName)
}
```

### Execute Commands

```go
result, err := client.Raw().Exec(
    "UPDATE books SET status = $1 WHERE id_book = $2",
    "ARCHIVED", bookId,
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Updated %d rows\n", result.RowsAffected())
```

### Column Alias Handling

The `Scan()` function automatically extracts column names from various SQL patterns:

| SQL Column               | Use `db` Tag   |
| ------------------------ | -------------- |
| `id_book`                | `db:"id_book"` |
| `b.title`                | `db:"title"`   |
| `a.first_name as author` | `db:"author"`  |
| `COUNT(*) as total`      | `db:"total"`   |

**Example with table aliases and column aliases:**

```go
type BookWithGenre struct {
    IdBook    string  `db:"id_book"`
    Title     string  `db:"title"`
    GenreName *string `db:"genre_name"`
}

var books []BookWithGenre
err := client.Raw().Query(`
    SELECT
        b.id_book,
        b.title,
        g.name as genre_name
    FROM books b
    LEFT JOIN book_genres bg ON b.id_book = bg.id_book
    LEFT JOIN genres g ON bg.id_genre = g.id_genre
    WHERE b.status = $1 AND b.deleted_at IS NULL
`, "PUBLISHED").Exec().Scan(&books)
```

## Transaction Examples

### Basic Transaction

Use transactions to ensure multiple operations succeed or fail together:

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    // Create author
    author, err := tx.Authors.Create().
        Data(inputs.AuthorsCreateInput{
            FirstName: "John",
            LastName:  "Doe",
        }).
        Exec(ctx)
    if err != nil {
        return err // Transaction rolled back automatically
    }

    // Create related book
    _, err = tx.Books.Create().
        Data(inputs.BooksCreateInput{
            Title: "My First Book",
        }).
        Exec(ctx)
    return err // If this fails, author creation is rolled back
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

### Transaction with Multiple Operations

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    // Create author
    author, err := tx.Authors.Create().
        Data(inputs.AuthorsCreateInput{
            FirstName: "John",
            LastName:  "Doe",
        }).
        Exec(ctx)
    if err != nil {
        return err
    }

    // Update author
    err = tx.Authors.Update().
        Where(inputs.AuthorsWhereInput{
            IdAuthor: db.String(author.IdAuthor),
        }).
        Data(inputs.AuthorsUpdateInput{
            Bio: db.String("Updated biography"),
        }).
        Exec(ctx)
    if err != nil {
        return err
    }

    // Create multiple books
    for _, title := range []string{"Book 1", "Book 2", "Book 3"} {
        _, err = tx.Books.Create().
            Data(inputs.BooksCreateInput{
                Title: title,
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

You can mix fluent API and raw SQL within the same transaction:

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    // Use fluent API
    author, err := tx.Authors.Create().
        Data(inputs.AuthorsCreateInput{
            FirstName: "John",
            LastName:  "Doe",
        }).
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

If any operation fails, the entire transaction is rolled back:

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    author, err := tx.Authors.Create().
        Data(inputs.AuthorsCreateInput{
            FirstName: "John",
            LastName:  "Doe",
            Email:     db.String("john@example.com"),
        }).
        Exec(ctx)
    if err != nil {
        return err // Transaction rolled back
    }

    // This will fail if email already exists (unique constraint)
    _, err = tx.Authors.Create().
        Data(inputs.AuthorsCreateInput{
            FirstName: "Jane",
            LastName:  "Doe",
            Email:     db.String("john@example.com"), // Duplicate!
        }).
        Exec(ctx)
    if err != nil {
        return err // Transaction rolled back, first author creation undone
    }

    return nil
})

if err != nil {
    // Both operations were rolled back
    log.Printf("Transaction failed, all changes rolled back: %v", err)
}
```

## Complete Working Example

Here's a complete example that demonstrates both Query Builder and Raw SQL:

```go
package main

import (
	"context"
	"log"
	"os"

	"my-app/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// Connect to database using generated helper
	databaseURL := os.Getenv("DATABASE_URL")
	pool, err := db.NewPgxPoolFromURL(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Create Prisma client using generated driver adapter
	dbDriver := db.NewPgxPoolDriver(pool)
	client := db.NewClient(dbDriver)

	// === Query Builder Examples ===

	// Create
	user, err := client.User.Create().
		Data(inputs.UserCreateInput{
			Email: "alice@example.com",
			Name:  db.String("Alice"),
		}).
		Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created: %+v\n", user)

	// Find First
	found, err := client.User.FindFirst().
		Select(inputs.UserSelect{
			Email: true,
			Name:  true,
		}).
		Where(inputs.UserWhereInput{
			Email: db.String("alice@example.com"),
		}).
		Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Found: %+v\n", found)

	// Update
	err = client.User.Update().
		Where(inputs.UserWhereInput{
			Id: db.Int(user.ID),
		}).
		Data(inputs.UserUpdateInput{
			Name: db.String("Alice Updated"),
		}).
		Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Updated successfully\n")

	// === Raw SQL Examples ===

	// Define struct with db tags for automatic scanning
	type BookResult struct {
		IdBook string `db:"id_book"`
		Title  string `db:"title"`
	}

	// Raw query with automatic scan (requires slice)
	var books []BookResult
	err = client.Raw().Query(
		"SELECT id_book, title FROM books WHERE status = $1",
		"PUBLISHED",
	).Exec().Scan(&books)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Raw query results:")
	for _, b := range books {
		log.Printf("  Book: %s - %s\n", b.IdBook, b.Title)
	}

	// Raw count with single row (requires primitive or struct)
	var count int
	err = client.Raw().QueryRow("SELECT COUNT(*) FROM authors").
		Exec().
		Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Total authors: %d\n", count)
}
```

## Database-Specific Examples

### PostgreSQL

```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
    "my-app/db"
)

pool, _ := db.NewPgxPoolFromURL(ctx, databaseURL)
dbDriver := db.NewPgxPoolDriver(pool)
client := db.NewClient(dbDriver)
```

### MySQL

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "my-app/db"
)

sqlDB, _ := sql.Open("mysql", databaseURL)
dbDriver := db.NewSQLDriver(sqlDB)
client := db.NewClient(dbDriver)
```

### SQLite

```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "my-app/db"
)

sqlDB, _ := sql.Open("sqlite3", "./database.db")
dbDriver := db.NewSQLDriver(sqlDB)
client := db.NewClient(dbDriver)
```

## Next Steps

- Read the [Quick Start Guide](QUICKSTART.md) for a complete walkthrough
- Check the [API Reference](API.md) for detailed documentation
- Learn about [Migrations](MIGRATIONS.md) for schema management
- Explore [Best Practices](BEST_PRACTICES.md) for production code
