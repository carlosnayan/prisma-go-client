# Migrations Guide

Learn how to manage database schema changes with Prisma migrations.

## Overview

Migrations allow you to version control your database schema and apply changes safely across different environments.

## Migration Workflow

### Development Workflow

1. **Modify Schema**: Edit `prisma/schema.prisma`
2. **Create Migration**: Run `prisma migrate dev --name migration_name`
3. **Review**: Check the generated SQL in `prisma/migrations/`
4. **Apply**: Migration is automatically applied
5. **Generate**: Code is automatically regenerated

### Production Workflow

1. **Create Migrations**: In development, create all migrations
2. **Commit**: Commit migration files to version control
3. **Deploy**: Run `prisma migrate deploy` in production
4. **Verify**: Check migration status with `prisma migrate status`

## Creating Migrations

### Basic Migration

```bash
# Edit schema.prisma, then:
prisma migrate dev --name add_user_table
```

This creates:

- `prisma/migrations/TIMESTAMP_add_user_table/migration.sql`
- Applies the migration
- Regenerates code

### Migration with Custom SQL

If you need custom SQL, edit the migration file after creation:

```bash
prisma migrate dev --name custom_migration
# Edit prisma/migrations/TIMESTAMP_custom_migration/migration.sql
# Add your custom SQL
```

## Migration Commands

### `prisma migrate dev`

Creates and applies a new migration in development:

```bash
# With --name flag
prisma migrate dev --name migration_name

# Or provide name as positional argument
prisma migrate dev migration_name
```

> [!NOTE]
> If name is not provided as argument or flag, you will be prompted interactively.

### `prisma migrate deploy`

Applies pending migrations in production:

```bash
prisma migrate deploy
```

This is non-interactive and safe for CI/CD.

### `prisma migrate reset`

Resets the database and reapplies all migrations:

```bash
prisma migrate reset
```

**Warning**: This drops all data!

Options:

- Automatically runs seed if configured

### `prisma migrate status`

Shows migration status:

```bash
prisma migrate status
```

Output:

- Local migrations
- Applied migrations
- Pending migrations
- Schema divergences

### `prisma migrate resolve`

Manually resolve migration state:

```bash
# Mark as applied
prisma migrate resolve --applied migration_name

# Mark as rolled back
prisma migrate resolve --rolled-back migration_name
```

Useful for fixing inconsistent states.

### `prisma migrate diff`

Compare schemas and generate migration SQL:

```bash
# Compare two schema files
prisma migrate diff --from schema1.prisma --to schema2.prisma --out migration.sql

# Compare schema with database
prisma migrate diff --from schema.prisma --to database

# Compare database with schema
prisma migrate diff --from database --to schema.prisma
```

### `prisma db push`

Push schema changes directly to the database without creating migration files:

```bash
prisma db push
```

**Options:**

- `--accept-data-loss` - Accept potential data loss from destructive operations (dropping columns/tables)
- `--skip-generate` - Do not run code generation after push

**Example:**

```bash
# Push changes that may cause data loss, skip generation
prisma db push --accept-data-loss --skip-generate
```

> [!WARNING]
> Use `db push` only in development. For production, always use migrations (`migrate deploy`).

### `prisma db execute`

Execute arbitrary SQL against your database:

**Options:**

- `--file` - Path to SQL file to execute
- `--stdin` - Read SQL from standard input

> [!WARNING]
> Only one of `--file` or `--stdin` can be specified at a time.

**Examples:**

```bash
# Execute SQL from file
prisma db execute --file schema_changes.sql

# Execute from stdin (pipe)
echo "TRUNCATE TABLE logs;" | prisma db execute --stdin

# Interactive mode (reads until Ctrl+D)
prisma db execute --stdin
DROP TABLE IF EXISTS old_table;
CREATE INDEX idx_user_email ON users(email);
^D
```

## Migration Files

### Structure

```
prisma/migrations/
  ├── 20240101120000_init/
  │   └── migration.sql
  ├── 20240102120000_add_email/
  │   └── migration.sql
  └── 20240103120000_add_indexes/
      └── migration.sql
```

### Migration SQL

Each migration contains SQL statements:

```sql
-- CreateTable
CREATE TABLE "users" (
    "id" SERIAL NOT NULL,
    "email" TEXT NOT NULL,
    "name" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "users_email_key" ON "users"("email");
```

## Schema Changes

### Adding a Model

```prisma
model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
```

Run: `prisma migrate dev --name add_post`

### Adding a Field

```prisma
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  age       Int?     // New field
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
```

Run: `prisma migrate dev --name add_age_to_user`

### Removing a Field

```prisma
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  // name removed
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
```

Run: `prisma migrate dev --name remove_name_from_user`

**Warning**: This will delete data!

### Changing Field Type

```prisma
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  age       String?  // Changed from Int to String
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
```

Run: `prisma migrate dev --name change_age_to_string`

**Warning**: May cause data loss if incompatible!

### Adding a Relation

```prisma
model User {
  id    Int    @id @default(autoincrement())
  email String @unique
  posts Post[]
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
}
```

Run: `prisma migrate dev --name add_user_posts_relation`

### Adding an Index

```prisma
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?  @index
  createdAt DateTime @default(now())
}
```

Run: `prisma migrate dev --name add_name_index`

## Migration Best Practices

### 1. Always Review Generated SQL

Check the migration SQL before applying:

```bash
prisma migrate dev --name migration_name
# Review prisma/migrations/TIMESTAMP_migration_name/migration.sql
```

### 2. Use Descriptive Names

```bash
# Good
prisma migrate dev --name add_email_to_user
prisma migrate dev --name create_post_table

# Bad
prisma migrate dev --name migration1
prisma migrate dev --name update
```

### 3. One Change Per Migration

Keep migrations focused:

```bash
# Good: Separate migrations
prisma migrate dev --name add_email
prisma migrate dev --name add_phone

# Bad: One migration with multiple changes
prisma migrate dev --name add_email_and_phone
```

### 4. Test Migrations

Always test migrations in development:

```bash
# Reset and test
prisma migrate reset
prisma migrate deploy
```

### 5. Backup Before Production

Always backup before applying migrations in production:

```bash
# PostgreSQL
pg_dump $DATABASE_URL > backup.sql

# MySQL
mysqldump $DATABASE_URL > backup.sql
```

### 6. Use Transactions

Migrations are automatically wrapped in transactions where supported.

### 7. Handle Data Migrations

For data migrations, add custom SQL:

```sql
-- Migration: add_default_values
ALTER TABLE users ADD COLUMN status TEXT;
UPDATE users SET status = 'active' WHERE status IS NULL;
ALTER TABLE users ALTER COLUMN status SET NOT NULL;
```

### 8. Rollback Strategy

Plan rollback strategies for critical migrations:

```sql
-- Create rollback migration
-- Migration: rollback_add_status
ALTER TABLE users DROP COLUMN status;
```

## Troubleshooting

### Migration Failed

If a migration fails:

1. Check the error message
2. Fix the issue in the migration file
3. Use `prisma migrate resolve` to mark as rolled back
4. Reapply the migration

### Divergence Detected

If schema diverges from database:

```bash
# Check status
prisma migrate status

# Option 1: Reset (development only)
prisma migrate reset

# Option 2: Create migration to sync
prisma migrate dev --name sync_schema
```

### Migration Already Applied

If migration is already applied but marked as pending:

```bash
prisma migrate resolve --applied migration_name
```

### Migration Not Applied

If migration should be applied but isn't:

```bash
# Check status
prisma migrate status

# Apply manually
prisma migrate deploy
```

## Database Push (Development Only)

For rapid development, use `db push` instead of migrations:

```bash
prisma db push
```

**Warning**:

- Not for production
- Doesn't create migration files
- May cause data loss

## Introspection

Generate schema from existing database:

```bash
prisma db pull
```

This updates `schema.prisma` based on the current database structure.

## Migration History

The `_prisma_migrations` table tracks applied migrations:

```sql
SELECT * FROM _prisma_migrations;
```

This table is automatically managed by Prisma.

## Examples

### Complete Migration Example

```bash
# 1. Edit schema
# Add model to schema.prisma

# 2. Create migration
prisma migrate dev --name add_product_model

# 3. Review generated SQL
cat prisma/migrations/*/migration.sql

# 4. Migration is automatically applied
# 5. Code is automatically regenerated

# 6. Use in code
product, err := client.Product.Create().
	Data(inputs.ProductCreateInput{
		// Add product data
	}).
	Exec(ctx)
```

### Custom Migration Example

```bash
# 1. Create migration
prisma migrate dev --name add_custom_index --create-only

# 2. Edit migration file
# Add custom SQL to migration.sql

# 3. Apply migration
prisma migrate deploy
```

## Next Steps

- Learn about [Relationships](RELATIONSHIPS.md)
- Read [Best Practices](BEST_PRACTICES.md)
- Check [API Reference](API.md)
