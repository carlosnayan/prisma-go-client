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

## Raw SQL Examples

### Query with Results

```go
// Query multiple rows
rows, err := client.Raw().Query(ctx,
    "SELECT id, email, name FROM users WHERE created_at > $1",
    time.Now().AddDate(0, -1, 0),
)
if err != nil {
    log.Fatal(err)
}
defer rows.Close()

for rows.Next() {
    var id int
    var email, name string
    if err := rows.Scan(&id, &email, &name); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %d - %s (%s)\n", id, name, email)
}
```

### Query Single Row

```go
// Query single row
row := client.Raw().QueryRow(ctx,
    "SELECT COUNT(*) FROM users WHERE email LIKE $1",
    "%@example.com",
)
var count int
if err := row.Scan(&count); err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d users\n", count)
```

### Execute Commands

```go
// Execute INSERT, UPDATE, DELETE
result, err := client.Raw().Exec(ctx,
    "UPDATE users SET name = $1 WHERE id = $2",
    "Updated Name", userID,
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Updated %d rows\n", result.RowsAffected())
```

### Complex Raw Queries

```go
// Join query
rows, err := client.Raw().Query(ctx, `
    SELECT u.id, u.email, COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON u.id = p.author_id
    GROUP BY u.id, u.email
    HAVING COUNT(p.id) > $1
`, 5)
defer rows.Close()

for rows.Next() {
    var userID int
    var email string
    var postCount int
    rows.Scan(&userID, &email, &postCount)
    fmt.Printf("User %s has %d posts\n", email, postCount)
}
```

## Transaction Examples

### Basic Transaction

Use transactions to ensure multiple operations succeed or fail together:

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
        return err // Transaction rolled back automatically
    }

    // Create related post
    _, err = tx.Post.Create().
        Data(inputs.PostCreateInput{
            Title:    db.String("My First Post"),
            AuthorId: db.String(user.ID),
        }).
        Exec(ctx)
    return err // If this fails, user creation is rolled back
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
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

    // Create multiple posts
    for _, title := range []string{"Post 1", "Post 2", "Post 3"} {
        _, err = tx.Post.Create().
            Data(inputs.PostCreateInput{
                Title:    db.String(title),
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

You can mix fluent API and raw SQL within the same transaction:

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    // Use fluent API
    user, err := tx.User.Create().
        Data(inputs.UserCreateInput{
            Email: "user@example.com",
            Name:  db.String("John Doe"),
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

If any operation fails, the entire transaction is rolled back:

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
    user, err := tx.User.Create().
        Data(inputs.UserCreateInput{
            Email: "user@example.com",
        }).
        Exec(ctx)
    if err != nil {
        return err // Transaction rolled back
    }

    // This will fail if email already exists
    _, err = tx.User.Create().
        Data(inputs.UserCreateInput{
            Email: "user@example.com", // Duplicate!
        }).
        Exec(ctx)
    if err != nil {
        return err // Transaction rolled back, first user creation undone
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

	// Raw query
	rows, err := client.Raw().Query(ctx,
		"SELECT id, email FROM users WHERE email LIKE $1",
		"%example%",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	log.Println("Raw query results:")
	for rows.Next() {
		var id int
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			log.Fatal(err)
		}
		log.Printf("  ID: %d, Email: %s\n", id, email)
	}

	// Raw count
	row := client.Raw().QueryRow(ctx, "SELECT COUNT(*) FROM users")
	var count int
	if err := row.Scan(&count); err != nil {
		log.Fatal(err)
	}
	log.Printf("Total users: %d\n", count)
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
