# Examples

Simple, practical examples using the new Prisma+Ent Query API with table package filters.

## Query Builder Examples

### Basic CRUD Operations

```go
import (
    "context"
    "my-app/db"
    "my-app/db/users" // Table package for type-safe filters
    "github.com/jackc/pgx/v5/pgxpool"
)

ctx := context.Background()
pool, _ := db.NewPgxPoolFromURL(ctx, databaseURL)
dbDriver := db.NewPgxPoolDriver(pool)
client := db.NewClient(dbDriver)

// Create a user
user, err := client.Users().Create().
    SetEmail("user@example.com").
    SetName("John Doe").
    WithContext(ctx). // optional
    Exec()

// Find first matching record
foundUser, err := client.Users().FindFirst().
    Where(users.EmailEQ("user@example.com")).
    Select(users.FieldEmail, users.FieldName).
    WithContext(ctx). // optional
    Exec()

// Find many with filters
activeUsers, err := client.Users().FindMany().
    Where(users.And(
        users.ActiveEQ(true),
        users.EmailContains("@example.com"),
    )).
    OrderBy(users.FieldCreatedAt, users.OrderDesc).
    Limit(20).
    Exec()

// Count matching records
count, err := client.Users().Count().
    Where(users.ActiveEQ(true)).
    Exec()

// Update by ID
err = client.Users().UpdateOneID(userId).
    SetName("Jane Doe").
    SetActive(true).
    Exec()

// Update many
err = client.Users().UpdateMany().
    Where(users.StatusEQ("inactive")).
    SetActive(false).
    Exec()

// Delete with conditions
err = client.Users().Delete().
    Where(users.EmailEQ("old@example.com")).
    Exec()
```

### Context Management

Context is optional and defaults to `context.Background()`:

```go
// Without context (uses context.Background())
users, err := client.Users().FindMany().
    Where(users.ActiveEQ(true)).
    Exec()

// With explicit context
users, err := client.Users().FindMany().
    Where(users.ActiveEQ(true)).
    WithContext(ctx).
    Exec()
```

### Field Selection and Ordering

```go
// Select specific fields
users, err := client.Users().FindMany().
    Select(users.FieldEmail, users.FieldName, users.FieldCreatedAt).
    Where(users.ActiveEQ(true)).
    Exec()

// Order by field
users, err := client.Users().FindMany().
    OrderBy(users.FieldCreatedAt, users.OrderDesc).
    Limit(10).
    Exec()

// Pagination
page := 2
pageSize := 20
users, err := client.Users().FindMany().
    Skip((page - 1) * pageSize).
    Limit(pageSize).
    OrderBy(users.FieldCreatedAt, users.OrderAsc).
    Exec()
```

### Advanced Filtering

```go
import "my-app/db/books"

// String filters
books, err := client.Books().FindMany().
    Where(books.TitleContains("Foundation")).
    Exec()

// Comparison filters
books, err := client.Books().FindMany().
    Where(books.And(
        books.PageCountGTE(200),
        books.PageCountLTE(500),
    )).
    Exec()

// Multiple conditions with OR
books, err := client.Books().FindMany().
    Where(books.Or(
        books.StatusEQ("published"),
        books.StatusEQ("archived"),
    )).
    Exec()
```

## Transactions

### Basic Transaction

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    // Create author
    author, err := tx.Authors().Create().
        SetFirstName("John").
        SetLastName("Doe").
        Exec()
    if err != nil {
        return err // Transaction rolled back automatically
    }

    // Create related book
    _, err = tx.Books().Create().
        SetTitle("My First Book").
        SetAuthorID(author.IdAuthor).
        Exec()
    return err // If this fails, author creation is rolled back
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

### Transaction with Multiple Operations

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    // Update multiple users
    err := tx.Users().UpdateMany().
        Where(users.StatusEQ("pending")).
        SetStatus("active").
        SetActivatedAt(time.Now()).
        Exec()
    if err != nil {
        return err
    }

    // Create audit log
    _, err = tx.AuditLogs().Create().
        SetAction("bulk_activation").
        SetTimestamp(time.Now()).
        Exec()

    return err
})
```

### Transaction with Raw SQL

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    // Use fluent API
    author, err := tx.Authors().Create().
        SetFirstName("John").
        SetLastName("Doe").
        Exec()
    if err != nil {
        return err
    }

    // Mix with raw SQL
    _, err = tx.Raw().Exec(`
        UPDATE authors
        SET updated_at = NOW()
        WHERE id_author = $1
    `, author.IdAuthor)
    return err
})
```

## Raw SQL Examples

Raw SQL provides full flexibility when you need complex queries or database-specific features.

> **Important:** Structs used with `Scan()` must have `db:"column_name"` tags.

### Query Multiple Rows

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
    WHERE b.status = $1
    ORDER BY b.created_at DESC
`, "PUBLISHED").Exec().Scan(&books)

if err != nil {
    log.Fatal(err)
}
```

### Query Single Row

```go
// With struct
type BookStats struct {
    TotalBooks     int `db:"total_books"`
    PublishedBooks int `db:"published_books"`
}

var stats BookStats
err := client.Raw().QueryRow(`
    SELECT
        COUNT(*) as total_books,
        COUNT(*) FILTER (WHERE status = 'PUBLISHED') as published_books
    FROM books
`).Exec().Scan(&stats)

// With primitive
var count int
err = client.Raw().QueryRow("SELECT COUNT(*) FROM authors").
    Exec().
    Scan(&count)
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

### Manual Row Iteration

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

## Complete Working Example

```go
package main

import (
    "context"
    "log"
    "os"

    "my-app/db"
    "my-app/db/users"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()

    // Connect to database
    databaseURL := os.Getenv("DATABASE_URL")
    pool, err := db.NewPgxPoolFromURL(ctx, databaseURL)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create Prisma client
    dbDriver := db.NewPgxPoolDriver(pool)
    client := db.NewClient(dbDriver)

    // Create user
    user, err := client.Users().Create().
        SetEmail("alice@example.com").
        SetName("Alice").
        Exec()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created: %+v\n", user)

    // Find user
    found, err := client.Users().FindFirst().
        Where(users.EmailEQ("alice@example.com")).
        Exec()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found: %+v\n", found)

    // Update user
    err = client.Users().UpdateOneID(user.ID).
        SetName("Alice Updated").
        Exec()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Updated successfully\n")

    // Count users
    count, err := client.Users().Count().
        Where(users.ActiveEQ(true)).
        Exec()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Active users: %d\n", count)
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
