# Quick Start Guide

Get started with Prisma for Go in minutes. This complete beginner-friendly guide will walk you through everything from installation to your first database query.

## Prerequisites

- **Go 1.18 or later** - [Download Go](https://go.dev/dl/)
- **A database** - PostgreSQL, MySQL, or SQLite
- **Basic knowledge of Go** - Understanding of Go syntax and packages

## Step 1: Install Prisma for Go

### Install the CLI

```bash
go install github.com/carlosnayan/prisma-go-client/cmd/prisma@latest
```

Verify installation:

```bash
prisma --version
```

### Install Your Database Driver

Choose and install the driver for your database:

**PostgreSQL:**

```bash
go get github.com/jackc/pgx/v5/pgxpool
```

**MySQL:**

```bash
go get github.com/go-sql-driver/mysql
```

**SQLite:**

```bash
go get github.com/mattn/go-sqlite3
```

## Step 2: Create a New Project

Create a new directory for your project:

```bash
mkdir my-prisma-app
cd my-prisma-app
go mod init my-prisma-app
```

## Step 3: Initialize Prisma

Initialize a new Prisma project:

```bash
prisma init
```

This creates:

- `prisma.conf` - Project configuration file
- `prisma/schema.prisma` - Database schema definition
- `prisma/migrations/` - Directory for database migrations

## Step 4: Configure Your Database

### Configure Database URL

The database URL can be provided in two ways:

1. **In `prisma.conf`** (recommended for most cases)
2. **As a parameter to `SetupClient()`** (overrides prisma.conf)

#### Option 1: Configure in prisma.conf

Update your `prisma.conf` file:

```toml
schema = "prisma/schema.prisma"

[migrations]
path = "prisma/migrations"

[datasource]
# Direct URL
url = "postgresql://user:password@localhost:5432/mydb?sslmode=disable"

# Or use environment variable
# url = "env('DATABASE_URL')"
```

If using `env('DATABASE_URL')`, make sure the `DATABASE_URL` environment variable is set when running your application.

## Step 5: Define Your Schema

Edit `prisma/schema.prisma`:

```prisma
datasource db {
  provider = "postgresql"  // or "mysql" or "sqlite"
}

generator client {
  provider = "prisma-client-go"
  output   = "../db"
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  createdAt DateTime @map("created_at") @default(now())
  updatedAt DateTime @map("updated_at") @default(now())

  @@map("users")
}
```

**Note:** Change `provider = "postgresql"` to `"mysql"` or `"sqlite"` based on your database.

## Step 6: Generate Code

Generate type-safe Go code from your schema:

```bash
prisma generate
```

This creates:

- `db/models.go` - Go structs for your models
- `db/queries.go` - Type-safe query builders
- `db/client.go` - Prisma client
- `db/inputs.go` - Input types (CreateInput, UpdateInput, WhereInput)

## Step 7: Create and Apply Migrations

Create your first migration:

```bash
prisma migrate dev --name init
```

This will:

1. Generate SQL migration file
2. Apply it to your database
3. Track the migration in `_prisma_migrations` table

## Step 8: Write Your First Query

Create `main.go`:

### PostgreSQL Example

```go
package main

import (
	"context"
	"log"

	"my-prisma-app/db"
	"my-prisma-app/db/inputs"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// Setup client - reads DATABASE_URL from prisma.conf
	// Or pass URL directly: db.SetupClient(ctx, "postgresql://...")
	client, pool, err := db.SetupClient(ctx)
	if err != nil {
		log.Fatalf("Failed to setup client: %v", err)
	}
	defer pool.Close()

	// Alternative: Manual setup with more control
	// pool, err := db.NewPgxPoolFromURL(ctx, "postgresql://user:pass@localhost/db")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	// Wrap with generated driver adapter
	dbDriver := db.NewPgxPoolDriver(pool)

	// Create Prisma client
	client := db.NewClient(dbDriver)

	// Example 1: Create a user using fluent API
	user, err := client.User.Create().
		Data(inputs.UserCreateInput{
			Email: "alice@example.com",
			Name:  db.String("Alice"),
		}).
		Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	log.Printf("Created user: %+v\n", user)

	// Example 2: Find user by email with Select
	foundUser, err := client.User.FindFirst().
		Select(inputs.UserSelect{
			Email: true,
			Name:  true,
		}).
		Where(inputs.UserWhereInput{
			Email: db.String("alice@example.com"),
		}).
		Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to find user: %v", err)
	}
	log.Printf("Found user: %+v\n", foundUser)

	// Example 3: Find all users
	users, err := client.User.FindMany().
		Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to find users: %v", err)
	}
	log.Printf("Found %d users\n", len(users))

	// Example 4: Update user
	err = client.User.Update().
		Where(inputs.UserWhereInput{
			Id: db.Int(user.ID),
		}).
		Data(inputs.UserUpdateInput{
			Name: db.String("Alice Updated"),
		}).
		Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	log.Printf("Updated user successfully\n")

	// Example 5: Raw SQL query
	rows, err := client.Raw().Query(ctx,
		"SELECT id, email, name FROM users WHERE email LIKE $1",
		"%example%",
	)
	if err != nil {
		log.Fatalf("Failed to execute raw query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var email, name string
		if err := rows.Scan(&id, &email, &name); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		log.Printf("Raw query result: id=%d, email=%s, name=%s\n", id, email, name)
	}
}
```

### MySQL Example

```go
package main

import (
	"context"
	"log"

	"my-prisma-app/db"
	"my-prisma-app/db/inputs"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	ctx := context.Background()

	// Automatic setup from DATABASE_URL
	client, sqlDB, err := db.SetupClient(ctx)
	if err != nil {
		log.Fatalf("Failed to setup client: %v", err)
	}
	defer sqlDB.Close()

	// Use the same fluent API
	user, err := client.User.Create().
		Data(inputs.UserCreateInput{
			Email: "bob@example.com",
			Name:  db.String("Bob"),
		}).
		Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	log.Printf("Created user: %+v\n", user)
}
```

### SQLite Example

```go
package main

import (
	"context"
	"log"

	"my-prisma-app/db"
	"my-prisma-app/db/inputs"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	ctx := context.Background()

	// Automatic setup from DATABASE_URL
	client, sqlDB, err := db.SetupClient(ctx)
	if err != nil {
		log.Fatalf("Failed to setup client: %v", err)
	}
	defer sqlDB.Close()

	// Use the same fluent API
	user, err := client.User.Create().
		Data(inputs.UserCreateInput{
			Email: "charlie@example.com",
			Name:  db.String("Charlie"),
		}).
		Exec(ctx)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	log.Printf("Created user: %+v\n", user)
}
```

## Step 9: Run Your Application

```bash
go run main.go
```

You should see output like:

```
Created user: {ID:1 Email:alice@example.com Name:Alice ...}
Found user: {ID:1 Email:alice@example.com Name:Alice ...}
Found 1 users
Updated user: {ID:1 Email:alice@example.com Name:Alice Updated ...}
```

## Understanding the Code

### Query Builder

The Query Builder provides a type-safe, fluent API:

```go
// Create
user, err := client.User.Create().
    Data(inputs.UserCreateInput{
        Email: "test@example.com",
        Name:  db.String("Test"),
    }).
    Exec(ctx)

// Find First with Select
user, err := client.User.FindFirst().
    Select(inputs.UserSelect{
        Email: true,
        Name:  true,
    }).
    Where(inputs.UserWhereInput{
        Email: db.String("test@example.com"),
    }).
    Exec(ctx)

// Find Many with Select and Where
users, err := client.User.FindMany().
    Select(inputs.UserSelect{
        Email: true,
        Name:  true,
    }).
    Where(inputs.UserWhereInput{
        Name: db.Contains("Test"),
    }).
    Exec(ctx)

// Update
err := client.User.Update().
    Where(inputs.UserWhereInput{
        Id: db.Int(user.ID),
    }).
    Data(inputs.UserUpdateInput{
        Name: db.String("Updated Name"),
    }).
    Exec(ctx)

// Delete
err := client.User.Delete().
    Where(inputs.UserWhereInput{
        Id: db.Int(user.ID),
    }).
    Exec(ctx)
```

### Raw SQL

For complex queries, use Raw SQL:

```go
// Query with results
rows, err := client.Raw().Query(ctx,
    "SELECT * FROM users WHERE created_at > $1",
    time.Now().AddDate(0, -1, 0),
)
defer rows.Close()

for rows.Next() {
    // Scan results
}

// Query single row
row := client.Raw().QueryRow(ctx,
    "SELECT COUNT(*) FROM users",
)
var count int
row.Scan(&count)

// Execute (INSERT, UPDATE, DELETE)
result, err := client.Raw().Exec(ctx,
    "UPDATE users SET name = $1 WHERE id = $2",
    "New Name", userID,
)
rowsAffected := result.RowsAffected()
```

## Common Commands

```bash
# Generate code from schema
prisma generate

# Create and apply migration
prisma migrate dev --name migration_name

# Apply pending migrations (production)
prisma migrate deploy

# Push schema changes directly (development)
prisma db push

# Pull schema from database
prisma db pull

# Format schema file
prisma format

# Validate schema
prisma validate

# Check migration status
prisma migrate status

# Reset database (development only)
prisma migrate reset
```

## Troubleshooting

### Database Connection Issues

**PostgreSQL:**

```bash
# Test connection
psql $DATABASE_URL
```

**MySQL:**

```bash
# Test connection
mysql -u user -p database_name
```

**SQLite:**

```bash
# Check if file exists
ls -la dev.db
```

### Migration Issues

If migrations fail:

```bash
# Check migration status
prisma migrate status

# Reset database (development only)
prisma migrate reset
```

### Code Generation Issues

If generated code seems outdated:

```bash
# Regenerate code
prisma generate

# Validate schema first
prisma validate
```

### Driver Not Found

Make sure you've installed the correct driver:

```bash
# PostgreSQL
go get github.com/jackc/pgx/v5/pgxpool

# MySQL
go get github.com/go-sql-driver/mysql

# SQLite
go get github.com/mattn/go-sqlite3
```

## Next Steps

- Read the [API Reference](API.md) for detailed query builder documentation
- Learn about [Migrations](MIGRATIONS.md) for managing schema changes
- Explore [Relationships](RELATIONSHIPS.md) for working with related data
- Check [Best Practices](BEST_PRACTICES.md) for production-ready code
- See [Examples](EXAMPLES.md) for more code samples

## Getting Help

- Check the [API Reference](API.md)
- Review [Best Practices](BEST_PRACTICES.md)
- Open an issue on [GitHub](https://github.com/carlosnayan/prisma-go-client/issues)
