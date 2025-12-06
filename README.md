# Prisma for Go

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A type-safe ORM library for Go inspired by Prisma, offering an intuitive API for working with databases.

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

### Library

```bash
go get github.com/carlosnayan/prisma-go-client
```

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

### 6. Use in Your Code

**PostgreSQL Example:**

```go
package main

import (
    "context"
    "log"
    "os"

    "my-app/db" // Your generated client
    "my-app/db/inputs" // Generated input types (WhereInput, etc.)
    "my-app/db/models" // Generated models
    "github.com/carlosnayan/prisma-go-client/builder"
    "github.com/carlosnayan/prisma-go-client/internal/driver"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()

    databaseURL := os.Getenv("DATABASE_URL")
    pool, err := pgxpool.New(ctx, databaseURL)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Wrap pgx pool with driver adapter
    dbDriver := driver.NewPgxPool(pool)
    client := db.NewClient(dbDriver)

    // Create a user using fluent API
    user, err := client.User.Create().
        Data(inputs.UserCreateInput{
            Email: "test@example.com",
            Name:  db.String("Test User"),
        }).
        Exec(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created user: %+v\n", user)

    // Find users using fluent API with type-safe WhereInput
    users, err := client.User.FindMany().
        Where(inputs.UserWhereInput{
            Email: db.Contains("example.com"),
        }).
        Exec(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found %d users\n", len(users))

    // Find first user with Select
    foundUser, err := client.User.FindFirst().
        Select(inputs.UserSelect{
            Email: true,
            Name:  true,
        }).
        Where(inputs.UserWhereInput{
            Email: db.String("test@example.com"),
        }).
        Exec(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found user: %+v\n", foundUser)

    // Raw SQL example
    rows, err := client.Raw().Query(ctx, "SELECT * FROM users WHERE email LIKE $1", "%example%")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    // Process rows...
}
```

**MySQL Example:**

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/carlosnayan/prisma-go-client/internal/driver"
)

db, err := sql.Open("mysql", databaseURL)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

dbDriver := driver.NewSQLDB(db)
client := db.NewClient(dbDriver)
```

**SQLite Example:**

```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "github.com/carlosnayan/prisma-go-client/internal/driver"
)

db, err := sql.Open("sqlite3", "./database.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

dbDriver := driver.NewSQLDB(db)
client := db.NewClient(dbDriver)
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
