# Prisma for Go

[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A type-safe ORM library for Go inspired by Prisma, offering an intuitive API for working with databases.

**Important:** This library is **not official** and is **not supported** by the official Prisma team. It is an independent, community-driven project inspired by Prisma's API design.

**Note:** This library requires Go 1.18 or later for generics support (used in `ExecTyped[T]()` method).

## ✨ Features

- 🔍 **Prisma-like Query Builder** - Familiar and intuitive API
- 🛡️ **Type-Safe** - Leverage Go's type system
- 🔄 **Migrations** - Database schema management
- ⚡ **Performance** - Driver-agnostic architecture
- 🎨 **Code Generation** - Automatically generate types and query builders
- 🔌 **Raw SQL** - Flexibility for complex queries
- 🗄️ **Multi-Database** - Support for PostgreSQL, MySQL, and SQLite
- 📝 **Schema Management** - Define your schema with `schema.prisma`
- 🚀 **CLI Tools** - Complete CLI for migrations, code generation, and more

## 📦 Installation

### CLI

```bash
go install github.com/carlosnayan/prisma-go-client/cmd/prisma@latest
```

### Database Drivers

Install the driver for your database:

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

## 🚀 Quick Start

### 1. Initialize a Project

```bash
prisma init
```

This creates:

- `prisma.conf`: Project configuration
- `prisma/schema.prisma`: Database schema definition
- `prisma/migrations/`: Directory for migrations

### 2. Define Your Schema

Edit `prisma/schema.prisma`:

```prisma
datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "../db"
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
```

### 3. Generate Code

```bash
prisma generate
```

This generates type-safe Go code in the `db` directory.

### 4. Run Migrations

```bash
prisma migrate dev --name initial_migration
```

### 5. Install Database Driver

Choose and install your database driver:

**For PostgreSQL:**

```bash
go get github.com/jackc/pgx/v5/pgxpool
```

**For MySQL:**

```bash
go get github.com/go-sql-driver/mysql
```

**For SQLite:**

```bash
go get github.com/mattn/go-sqlite3
```

### 6. Setup Database Connection

**PostgreSQL Setup:**

```go
package database

import (
    "context"
    "log"
    "os"

    db "my-app/db" // Your generated client
)

var (
    Client *db.Client
)

func SetupPrismaClient() {
    ctx := context.Background()

    // Automatic setup - handles pool creation, configuration, and connection
    var err error
    Client, _, err = db.SetupClient(ctx)
    if err != nil {
        log.Fatalf("Error setting up client: %v", err)
    }
}

// Alternative: Manual setup if you need more control
// Note: NewPgxPoolFromURL automatically configures PgBouncer compatibility
func SetupPrismaClientManual() {
    ctx := context.Background()
    databaseURL := os.Getenv("DATABASE_URL")

    // NewPgxPoolFromURL creates pool with PgBouncer-compatible settings by default
    pool, err := db.NewPgxPoolFromURL(ctx, databaseURL)
    if err != nil {
        log.Fatalf("Error creating pool: %v", err)
    }

    if err := pool.Ping(ctx); err != nil {
        log.Fatalf("Error pinging database: %v", err)
    }

    // Use the generated driver adapter
    dbDriver := db.NewPgxPoolDriver(pool)
    Client = db.NewClient(dbDriver)
}
```

### 7. Use in Your Code

**PostgreSQL Example:**

```go
package main

import (
    "context"
    "log"

    "my-app/db" // Your generated client
    "my-app/db/inputs" // Generated input types (WhereInput, etc.)
    "my-app/database" // Your database setup package
)

func main() {
    ctx := context.Background()

    // Setup client (call once at application startup)
    database.SetupPrismaClient()

    // Set context once and reuse it
    query := database.Client.User.WithContext(ctx)

    // Create a user using fluent API (using stored context)
    user, err := query.Create().
        Data(inputs.UserCreateInput{
            Email: "test@example.com",
            Name:  db.String("Test User"),
        }).
        Exec() // Uses stored context
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created user: %+v\n", user)

    // Find users using fluent API with type-safe WhereInput
    users, err := query.FindMany().
        Where(inputs.UserWhereInput{
            Email: db.Contains("example.com"),
        }).
        Exec() // Uses stored context
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found %d users\n", len(users))

    // Find first user with Select
    foundUser, err := query.FindFirst().
        Select(inputs.UserSelect{
            Email: true,
            Name:  true,
        }).
        Where(inputs.UserWhereInput{
            Email: db.String("test@example.com"),
        }).
        Exec() // Uses stored context
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found user: %+v\n", foundUser)

    // Find with custom DTO using ExecTyped (requires Go 1.18+)
    type UserDTO struct {
        Email string `json:"email" db:"email"`
        Name  string `json:"name" db:"name"`
    }

    var userDTO *UserDTO
    err = query.FindFirst().
        Select(inputs.UserSelect{
            Email: true,
            Name:  true,
        }).
        Where(inputs.UserWhereInput{
            Email: db.String("test@example.com"),
        }).
        ExecTyped(&userDTO) // Uses stored context
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found user DTO: %+v\n", userDTO)

    // Find many with custom DTO
    var usersDTO []UserDTO
    err = query.FindMany().
        Select(inputs.UserSelect{
            Email: true,
            Name:  true,
        }).
        ExecTyped(&usersDTO) // Uses stored context
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found %d users\n", len(usersDTO))

    // Raw SQL example
    rows, err := database.Client.Raw().Query(ctx, "SELECT * FROM users WHERE email LIKE $1", "%example%")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    // Process rows...
}
```

**MySQL Example:**

```go
package database

import (
    "context"
    "log"

    db "my-app/db"
)

var (
    Client *db.Client
)

func SetupPrismaClient() {
    ctx := context.Background()

    // Automatic setup - handles connection and configuration
    var err error
    Client, _, err = db.SetupClient(ctx)
    if err != nil {
        log.Fatalf("Error setting up client: %v", err)
    }
}
```

**SQLite Example:**

```go
package database

import (
    "context"
    "log"

    db "my-app/db"
)

var (
    Client *db.Client
)

func SetupPrismaClient() {
    ctx := context.Background()

    // Automatic setup - handles connection and configuration
    var err error
    Client, _, err = db.SetupClient(ctx)
    if err != nil {
        log.Fatalf("Error setting up client: %v", err)
    }
}

    dbDriver := db.NewSQLDriver(sqlDB)
    Client = db.NewClient(dbDriver)
}
```

## 📚 Documentation

- [Quick Start Guide](docs/QUICKSTART.md)
- [API Reference](docs/API.md)
- [Migrations Guide](docs/MIGRATIONS.md)
- [Relationships Guide](docs/RELATIONSHIPS.md)
- [Best Practices](docs/BEST_PRACTICES.md)
- [Examples](docs/EXAMPLES.md)

## 🛠️ CLI Commands

### Project Management

- `prisma init` - Initialize a new project
- `prisma generate` - Generate Go code from schema.prisma
- `prisma validate` - Validate schema.prisma
- `prisma format` - Format schema.prisma

### Migrations

- `prisma migrate dev` - Create and apply migrations in development
- `prisma migrate deploy` - Apply pending migrations in production
- `prisma migrate reset` - Reset database and reapply all migrations
- `prisma migrate status` - Check migration status
- `prisma migrate resolve` - Manually resolve migration conflicts
- `prisma migrate diff` - Compare schemas and generate migration SQL

### Database Operations

- `prisma db push` - Apply schema changes directly to database
- `prisma db pull` - Introspect database and generate schema.prisma
- `prisma db seed` - Execute database seed scripts
- `prisma db execute` - Execute arbitrary SQL

## 🗄️ Supported Databases

- **PostgreSQL** - Full support with JSONB, full-text search
- **MySQL** - Full support with JSON, full-text search
- **SQLite** - Full support with JSON functions

## 📖 Examples

See [Examples Guide](docs/EXAMPLES.md) for practical examples including:

- Query Builder usage
- Raw SQL queries
- Working with different databases

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Prisma](https://www.prisma.io/)
- Supports multiple database drivers (pgx, mysql, sqlite3)

## 📝 Roadmap

See [ROADMAP.md](ROADMAP.md) for the complete development roadmap.

## 🐛 Reporting Issues

If you find a bug or have a feature request, please [open an issue](https://github.com/carlosnayan/prisma-go-client/issues).
