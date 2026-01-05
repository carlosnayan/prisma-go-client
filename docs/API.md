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

- `client.Authors.Create()` - Returns a Create builder
- `client.Authors.FindMany()` - Returns a FindMany builder
- `client.Authors.FindFirst()` - Returns a FindFirst builder
- `client.Authors.Update()` - Returns an Update builder
- `client.Authors.Delete()` - Returns a Delete builder
- `client.Authors.Upsert()` - Returns an Upsert builder (create or update)
- `client.Authors.WithContext(ctx)` - Sets the context for subsequent operations

### Context Management

You can set a context once and reuse it for multiple operations:

```go
// Set context once
query := client.Authors.WithContext(ctx)

// Use Exec() without passing context explicitly
user, err := query.Create().
    Data(inputs.AuthorsCreateInput{
        Email: db.String("author@example.com"),
        FirstName: "John", LastName: "Doe",
    }).
    Exec() // Uses stored context

// Explicit context still works and takes priority
user, err := query.Create().
    Data(inputs.AuthorsCreateInput{
        Email: db.String("author@example.com"),
    }).
    ExecWithContext(otherCtx) // Uses otherCtx instead
```

If no context is stored and `Exec()` is called without parameters, `context.Background()` is used as fallback.

## CRUD Operations

### Create

```go
// Create a single record using fluent API
user, err := client.Authors.Create().
	Data(inputs.AuthorsCreateInput{
		Email: db.String("author@example.com"),
		FirstName: "John", LastName: "Doe",
	}).
	Exec(ctx)
```

#### Required Fields Validation

When creating records, all required fields must be provided. A field is considered required if:

- It is not optional (no `?` suffix in the Prisma schema)
- It does not have a `@default` value

If required fields are missing, a validation error is returned:

```go
// Missing required field 'email'
user, err := client.Authors.Create().
	Data(inputs.AuthorsCreateInput{
		FirstName: "John", LastName: "Doe",
	}).
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
	Data(inputs.AuthorsCreateInput{
		Email: db.String("author@example.com"),
		Name:  "John Doe",
		Age:   nil, // Optional field
	}).
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
		{Email: db.String("author@example.com"), Name: "User", Bio: "Bio"},
		{Email: db.String("author@example.com"), Name: "User", Bio: "Bio"}, // Duplicate
	}).
	SkipDuplicates(true).
	Exec(ctx)
// Duplicate records are skipped (PostgreSQL: ON CONFLICT DO NOTHING, MySQL: ON DUPLICATE KEY UPDATE)
```

### Read

```go
// Find first matching record with Select
user, err := client.Authors.FindFirst().
	Select(inputs.AuthorsSelect{
		Email: true,
		Name:  true,
	}).
	Where(inputs.AuthorsWhereInput{
		Email: db.String("author@example.com"),
	}).
	Exec(ctx)

// Find many records with Select and Where
users, err := client.Authors.FindMany().
	Select(inputs.AuthorsSelect{
		Email: true,
		Name:  true,
	}).
	Where(inputs.AuthorsWhereInput{
		Email: db.Contains("author"),
	}).
	Exec(ctx)
```

### Update

```go
// Update single record
err := client.Authors.Update().
	Where(inputs.AuthorsWhereInput{
		Id: db.Int(1),
	}).
	Data(inputs.AuthorsUpdateInput{
		Bio: db.String("Updated biography"),
	}).
	Exec(ctx)
```

### Delete

```go
// Delete single record
err := client.Authors.Delete().
	Where(inputs.AuthorsWhereInput{
		Id: db.Int(1),
	}).
	Exec()
```

### DeleteMany

Delete multiple records in a single operation. Unlike `Delete`, the `Where` clause is optional.

```go
// Delete records matching a condition
query := client.Genres.WithContext(ctx)
result, err := query.DeleteMany().
	Where(inputs.GenresWhereInput{
		Name: filters.Strings.Contains("Fiction"),
	}).
	Exec()
fmt.Printf("Deleted %d genres\n", result.Count)
```

**Without Where - deletes ALL records from the table:**

```go
// Delete ALL records from the genres table
result, err := client.Genres.DeleteMany().Exec()
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Deleted %d records\n", result.Count)
```

**With ExecWithContext:**

```go
// Using explicit context
result, err := client.Authors.DeleteMany().
	Where(inputs.AuthorsWhereInput{
		Nationality: filters.Strings.Equals("Unknown"),
	}).
	ExecWithContext(ctx)
```

| Method         | Where    | Returns                 |
| -------------- | -------- | ----------------------- |
| `Delete()`     | Required | `error`                 |
| `DeleteMany()` | Optional | `*BatchPayload` (count) |

### Upsert

Upsert combines create and update into a single operation: if a matching record exists, it updates it; otherwise, it creates a new record.

```go
// Create or update a genre based on unique name
genre, err := client.Genres.Upsert().
	Where(inputs.GenresWhereInput{
		Name: filters.Strings.Equals("Science Fiction"),
	}).
	Create(inputs.GenresCreateInput{
		Name:        "Science Fiction",
		Description: inputs.String.Set("Stories about futuristic science and technology"),
	}).
	Update(inputs.GenresUpdateInput{
		Description: inputs.String.Set("Updated description for Science Fiction"),
	}).
	Exec(ctx)
```

#### How Upsert Works

1. **Where** - Condition to find existing record (accepts **any column**, not just unique fields)
2. **Create** - Data for new record if not found
3. **Update** - Data to apply if record exists

| Scenario                  | Result                     |
| ------------------------- | -------------------------- |
| Record **does not exist** | Creates with `Create` data |
| Record **exists**         | Updates with `Update` data |

> [!NOTE]
> The `Where` clause accepts **any column**, not just `@unique` fields. When you use a unique index in the Where clause, the query is automatically optimized for better performance.

#### Upsert with Unique Field (Recommended)

For optimal performance with models that have `@unique` fields, use the unique field in Where:

```go
// books.isbn is @unique - automatically optimized
book, err := client.Books.Upsert().
	Where(inputs.BooksWhereInput{
		Isbn: filters.Strings.Equals("978-0-13-468599-1"),
	}).
	Create(inputs.BooksCreateInput{
		Title: "The Pragmatic Programmer",
		Isbn:  inputs.String.Set("978-0-13-468599-1"),
	}).
	Update(inputs.BooksUpdateInput{
		Title: inputs.String.Set("The Pragmatic Programmer (Updated)"),
	}).
	ExecWithContext(ctx)
```

#### Upsert with Composite Unique Constraint

For models with `@@unique([field1, field2])`, include ALL constraint fields in Where:

```go
// book_authors has @@unique([id_book, id_author])
bookAuthor, err := client.BookAuthors.Upsert().
	Where(inputs.BookAuthorsWhereInput{
		IdBook:   filters.String(bookId),
		IdAuthor: filters.String(authorId),
	}).
	Create(inputs.BookAuthorsCreateInput{
		IdBook:   bookId,
		IdAuthor: authorId,
		Role:     inputs.String.Set("author"),
		Order:    0,
	}).
	Update(inputs.BookAuthorsUpdateInput{
		Role:  inputs.String.Set("co-author"),
		Order: inputs.Int.Set(1),
	}).
	Exec(ctx)
```

#### Unique Constraints and Upsert Behavior

When using Upsert with models that have unique constraints, the behavior depends on which fields are used in the Where clause.

**Example Schema (from schema.prisma):**

```prisma
model chapters {
  id_chapter     String @id @default(dbgenerated("gen_random_uuid()"))
  id_book        String
  chapter_number Int
  title          String
  content        String?

  @@unique([id_book, chapter_number], map: "chapters_unique_book_number")
}
```

**Decision Table for `@@unique([id_book, chapter_number])`:**

| Where Clause                     | Match Type      | Behavior                            |
| -------------------------------- | --------------- | ----------------------------------- |
| `{IdBook, ChapterNumber}`        | ✅ Exact match  | Works correctly                     |
| `{IdBook}`                       | ❌ Incomplete   | May find/update wrong record        |
| `{ChapterNumber}`                | ❌ Incomplete   | May find/update wrong record        |
| `{IdBook, ChapterNumber, Title}` | ⚠️ Extra fields | Works, but Title used for filtering |

**Best Practice:** When using Upsert with composite unique constraints, always include **all fields** that form the unique constraint in your Where clause.

```go
// ✅ Correct: All unique constraint fields
client.Chapters.Upsert().
	Where(inputs.ChaptersWhereInput{
		IdBook:        filters.Strings.Equals(bookId),
		ChapterNumber: filters.Int.Equals(1),
	})

// ⚠️ Incomplete: Missing ChapterNumber
client.Chapters.Upsert().
	Where(inputs.ChaptersWhereInput{
		IdBook: filters.String(bookId),
	})
```

#### Upsert Validation Errors

All three methods (Where, Create, Update) are required:

```go
// ❌ Error: Missing Where
_, err := client.Genres.Upsert().Create(...).Update(...).Exec()
// "where is required for upsert"

// ❌ Error: Missing Create
_, err := client.Genres.Upsert().Where(...).Update(...).Exec()
// "create is required for upsert"

// ❌ Error: Missing Update
_, err := client.Genres.Upsert().Where(...).Create(...).Exec()
// "update is required for upsert"
```

#### Upsert in Data Sync Scenarios

```go
// Sync book-store availability from external source
for _, storeData := range externalStoreData {
	// book_stores has @@unique([id_book, id_store])
	_, err := client.BookStores.Upsert().
		Where(inputs.BookStoresWhereInput{
			IdBook:  filters.String(storeData.BookID),
			IdStore: filters.String(storeData.StoreID),
		}).
		Create(inputs.BookStoresCreateInput{
			IdBook:        storeData.BookID,
			IdStore:       storeData.StoreID,
			Price:         storeData.Price,
			StockQuantity: storeData.Stock,
		}).
		Update(inputs.BookStoresUpdateInput{
			Price:         inputs.Decimal.Set(storeData.Price),
			StockQuantity: inputs.Int.Set(storeData.Stock),
		}).
		Exec(ctx)

	if err != nil {
		log.Printf("Failed to sync store %s: %v", storeData.StoreID, err)
	}
}
```

## Query Options

### Where Clauses

```go
// Simple where
users, err := client.Authors.FindMany().
	Where(inputs.AuthorsWhereInput{
		Email: db.String("author@example.com"),
	}).
	Exec(ctx)

// Multiple conditions (AND)
users, err := client.Authors.FindMany().
	Where(inputs.AuthorsWhereInput{
		Email: db.String("author@example.com"),
		Name:  db.String("John"),
	}).
	Exec(ctx)

// OR conditions
users, err := client.Authors.FindMany().
	Where(inputs.AuthorsWhereInput{
		Or: []inputs.AuthorsWhereInput{
			{Email: db.String("author1@example.com")},
			{Email: db.String("author2@example.com")},
		},
	}).
	Exec(ctx)

// NOT conditions
users, err := client.Authors.FindMany().
	Where(inputs.AuthorsWhereInput{
		Not: &inputs.AuthorsWhereInput{
			Email: db.String("admin@example.com"),
		},
	}).
	Exec(ctx)
```

### Text Operators

```go
// Contains
users, err := client.Authors.FindMany().
	Where(inputs.AuthorsWhereInput{
		Email: db.Contains("author"),
	}).
	Exec(ctx)

// Starts with
users, err := client.Authors.FindMany().
	Where(inputs.AuthorsWhereInput{
		FirstName: db.StartsWith("John"),
	}).
	Exec(ctx)

// Ends with
users, err := client.Authors.FindMany().
	Where(inputs.AuthorsWhereInput{
		LastName: db.EndsWith("son"),
	}).
	Exec(ctx)
```

### Comparison Operators

```go
// Greater than
posts, err := client.Books.FindMany().
	Where(inputs.BooksWhereInput{
		PageCount: db.IntGt(100),
	}).
	Exec(ctx)

// Less than
posts, err := client.Books.FindMany().
	Where(inputs.BooksWhereInput{
		PageCount: db.IntLt(10),
	}).
	Exec(ctx)

// Greater than or equal
posts, err := client.Books.FindMany().
	Where(inputs.BooksWhereInput{
		PageCount: db.IntGte(100),
	}).
	Exec(ctx)

// Less than or equal
posts, err := client.Books.FindMany().
	Where(inputs.BooksWhereInput{
		PageCount: db.IntLte(10),
	}).
	Exec(ctx)

// In array (using StringIn helper)
users, err := client.Authors.FindMany().
	Where(inputs.AuthorsWhereInput{
		Id: db.Int(1), // For exact match, use Int()
	}).
	Exec(ctx)
```

### Ordering

```go
// Order by single field
users, err := client.Authors.FindMany().
	OrderBy(db.AuthorsOrderByInput{
		CreatedAt: db.SortOrderDesc,
	}).Exec()

// Order by multiple fields
users, err := client.Authors.FindMany().
	OrderBy(db.AuthorsOrderByInput{
		CreatedAt: db.SortOrderDesc,
		Name:      db.SortOrderAsc,
	}).Exec()
```

### Pagination

```go
// Take results
users, err := client.Authors.FindMany().
	Take(10).Exec()

// Skip results
users, err := client.Authors.FindMany().
	Skip(20).Exec()

// Take and skip (pagination)
page := 1
pageSize := 10
users, err := client.Authors.FindMany().
	Skip((page - 1) * pageSize).
	Take(pageSize).
	Exec()
```

### Selecting Fields

```go
// Select specific fields using type-safe Select
users, err := client.Authors.FindMany().
	Select(inputs.AuthorsSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Exec(ctx)
```

### Custom Types with ExecTyped (Go 1.18+)

The `ExecTyped()` method allows you to scan query results into custom DTOs (Data Transfer Objects) instead of the default generated models. This is useful when you need to return different structures to your API clients.

**Requirements:**

- Go 1.18 or later (for generics support)
- Custom structs must have `json` or `db` tags for field mapping

**Example with explicit context:**

```go
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
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Where(inputs.AuthorsWhereInput{
		Email: db.String("author@example.com"),
	}).
	ExecTypedWithContext(ctx, &userDTO)
if err != nil {
	log.Fatal(err)
}
// userDTO is automatically of type *UserDTO, no casting needed!

// Find many with custom DTO
var usersDTO []UserDTO
err = client.Authors.FindMany().
	Select(inputs.AuthorsSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Where(inputs.AuthorsWhereInput{
		Email: db.Contains("author"),
	}).
	ExecTypedWithContext(ctx, &usersDTO)
if err != nil {
	log.Fatal(err)
}
// usersDTO is automatically of type []UserDTO, no casting needed!
```

**Example with WithContext():**

```go
// Set context once
query := client.Authors.WithContext(ctx)

// Find first with custom DTO using stored context
var userDTO *UserDTO
err := query.FindFirst().
	Select(inputs.AuthorsSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Where(inputs.AuthorsWhereInput{
		Email: db.String("author@example.com"),
	}).
	ExecTyped(&userDTO) // Uses stored context

// Find many with custom DTO using stored context
var usersDTO []UserDTO
err = query.FindMany().
	Select(inputs.AuthorsSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Where(inputs.AuthorsWhereInput{
		Email: db.Contains("author"),
	}).
	ExecTyped(&usersDTO) // Uses stored context
```

**Field Mapping:**

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
```

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
	"my-app/db"
	"my-app/db/filters"
	"my-app/db/inputs"
)

// Basic full-text search
posts, err := client.Posts.FindMany().
	Where(inputs.PostsWhereInput{
		Content: filters.FullTextSearch("golang prisma"),
	}).
	Exec(ctx)

// With custom configuration (PostgreSQL)
posts, err := client.Posts.FindMany().
	Where(inputs.PostsWhereInput{
		Content: filters.FullTextSearchConfig(map[string]interface{}{
			"query": "golang & prisma",
			"config": "english", // or "portuguese", "spanish", etc.
		}),
	}).
	Exec(ctx)
```

#### JSON Array Operators

For fields with JSON array type, use these specialized operators:

### Nullable Field Handling

The Prisma Go Client provides a fluent API for handling nullable fields with explicit NULL support. This allows you to distinguish between "not providing a value" and "explicitly setting to NULL".

#### Basic Usage

Use the nullable wrapper API from the `inputs` package:

```go
// ✅ Setting non-null values
author, err := client.Authors.Create().
    Data(inputs.AuthorsCreateInput{
        FirstName:   inputs.String.Set("Isaac"),
        LastName:    inputs.String.Set("Asimov"),
        Bio:         inputs.String.Set("Science fiction writer"),
        Nationality: inputs.String.Set("American"),
    }).
    Exec(ctx)

// ✅ Setting fields to explicit NULL
author, err := client.Authors.Update().
    Where(inputs.AuthorsWhereInput{
        IdAuthor: filters.String.Equals(authorId),
    }).
    Data(inputs.AuthorsUpdateInput{
        Website: inputs.String.SetNull(),  // Explicitly set to NULL
        Email:   inputs.String.SetNull(),  // Explicitly set to NULL
    }).
    Exec(ctx)
```

#### Available Nullable Wrappers

All primitive types have nullable wrappers:

| Wrapper           | Set Method                | SetNull Method | Use Case             |
| ----------------- | ------------------------- | -------------- | -------------------- |
| `inputs.String`   | `.Set(v string)`          | `.SetNull()`   | String fields        |
| `inputs.Int`      | `.Set(v int)`             | `.SetNull()`   | Integer fields       |
| `inputs.Int64`    | `.Set(v int64)`           | `.SetNull()`   | Int64 fields         |
| `inputs.Float`    | `.Set(v float64)`         | `.SetNull()`   | Float/Decimal fields |
| `inputs.Bool`     | `.Set(v bool)`            | `.SetNull()`   | Boolean fields       |
| `inputs.DateTime` | `.Set(v time.Time)`       | `.SetNull()`   | DateTime fields      |
| `inputs.Json`     | `.Set(v json.RawMessage)` | `.SetNull()`   | JSON fields          |
| `inputs.Bytes`    | `.Set(v []byte)`          | `.SetNull()`   | Bytes fields         |

#### Mix of Values and NULL

You can combine regular values with explicit NULLs:

```go
book, err := client.Books.Create().
    Data(inputs.BooksCreateInput{
        Title:       inputs.String.Set("Foundation"),
        Subtitle:    inputs.String.SetNull(),  // No subtitle
        Description: inputs.String.Set("First book in the Foundation series"),
        PageCount:   inputs.Int.Set(255),
        Price:       inputs.Float.SetNull(),  // Price not set yet
        IsActive:    true,  // Required fields don't need wrappers
    }).
    Exec(ctx)
```

#### Update Operations with Partial Data

In updates, you can set some fields to NULL while updating others:

```go
// Update a book: clear subtitle, update description, leave price unchanged
err := client.Books.Update().
    Where(inputs.BooksWhereInput{
        IdBook: filters.String.Equals(bookId),
    }).
    Data(inputs.BooksUpdateInput{
        Subtitle:    inputs.String.SetNull(),                  // Clear subtitle
        Description: inputs.String.Set("Updated description"), // Update description
        // Price field omitted - will not be changed
    }).
    Exec(ctx)
```

#### Three States of Nullable Fields

Nullable fields support three distinct states:

| State       | JSON Output   | Use Case                | Example                            |
| ----------- | ------------- | ----------------------- | ---------------------------------- |
| **Not Set** | Field omitted | Don't modify this field | Field not included in Input struct |
| **NULL**    | `null`        | Clear the field value   | `inputs.String.SetNull()`          |
| **Value**   | Actual value  | Set a specific value    | `inputs.String.Set("value")`       |

```go
// Example showing all three states
client.Authors.Update().
    Data(inputs.AuthorsUpdateInput{
        Bio: inputs.String.Set("New bio"),     // 1. Set to value
        Website: inputs.String.SetNull(),       // 2. Set to NULL
        // Email is not set                      // 3. Not modified
    }).
    Exec(ctx)
```

#### Practical Example: User Profile Update

```go
// User wants to clear their phone number but keep email
func UpdateUserProfile(ctx context.Context, userID string, updates ProfileUpdates) error {
    updateData := inputs.AuthorsUpdateInput{
        FirstName: inputs.String.Set(updates.FirstName),
    }

    // Clear phone if user requested
    if updates.ClearPhone {
        updateData.Phone = inputs.String.SetNull()
    } else if updates.Phone != "" {
        updateData.Phone = inputs.String.Set(updates.Phone)
    }
    // If neither condition is true, phone field is omitted and won't be modified

    return client.Authors.Update().
        Where(inputs.AuthorsWhereInput{
            IdAuthor: filters.String.Equals(userID),
        }).
        Data(updateData).
        Exec(ctx)
}
```

#### Best Practices

1. **Use `.Set()` for non-null values**: Always use `.Set()` when you have a value
2. **Use `.SetNull()` for explicit NULL**: Use when you want to clear/nullify a field
3. **Omit fields to leave unchanged**: Don't include fields in UpdateInput if you don't want to modify them
4. **Type safety**: Each wrapper is type-safe - you can't pass wrong types

```go
// ✅ Correct
inputs.String.Set("text")
inputs.Int.Set(42)

// ❌ Compile error - type mismatch
inputs.String.Set(42)  // Cannot use int as string
inputs.Int.Set("text") // Cannot use string as int
```

####

```go
// Has - check if array contains a specific value
users, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Tags: filters.Has("admin"),
	}).
	Exec(ctx)

// HasEvery - array contains all specified values
users, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Tags: filters.HasEvery([]interface{}{"admin", "editor"}),
	}).
	Exec(ctx)

// HasSome - array contains any of the specified values
users, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Tags: filters.HasSome([]interface{}{"admin", "editor", "viewer"}),
	}).
	Exec(ctx)

// IsEmpty - array is empty
users, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Tags: filters.IsEmpty(),
	}).
	Exec(ctx)
```

#### Null Check Operators

Check for NULL values in database fields:

```go
import (
	"my-app/db/filters"
	"my-app/db/inputs"
)

// IS NULL - check if string field is null
users, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Bio: filters.Strings.IsNull(),
	}).
	Exec(ctx)

// IS NOT NULL - check if string field is not null
users, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Bio: filters.Strings.IsNotNull(),
	}).
	Exec(ctx)

// For DateTime fields
users, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		DeletedAt: filters.DateTime.IsNull(),
	}).
	Exec(ctx)
```

#### Case-Insensitive Operations

Perform case-insensitive string operations (uses PostgreSQL's ILIKE):

```go
// Case-insensitive contains (matches "John", "JOHN", "john", etc.)
users, err := client.Users.FindMany().
	Where(inputs.UsersWhereInput{
		Name: filters.Strings.ContainsInsensitive("john"),
	}).
	Exec(ctx)

// Other case-insensitive operators
filters.Strings.StartsWithInsensitive("prefix")
filters.Strings.EndsWithInsensitive("suffix")
```

### Including Relations

```go
// Include related data
posts, err := client.Books.FindMany().
	Include(db.BooksIncludeInput{
		Author: true,
	}).Exec()

// Nested includes
posts, err := client.Books.FindMany().
	Include(db.BooksIncludeInput{
		Author: db.AuthorsIncludeInput{
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

	// Create book (would need book_authors junction in real scenario)
	_, err = tx.Books.Create().
		Data(inputs.BooksCreateInput{
			Title: "My Book",
		}).
		Exec(ctx)
	return err
})
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

	// Create related books
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

If any operation returns an error, the transaction is automatically rolled back:

```go
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	author, err := tx.Authors.Create().
		Data(inputs.AuthorsCreateInput{
			FirstName: "John",
			LastName:  "Doe",
		}).
		Exec(ctx)
	if err != nil {
		return err // Transaction will be rolled back
	}

	// If this fails, the author creation above will be rolled back
	_, err = tx.Books.Create().
		Data(inputs.BooksCreateInput{
			Title: "My Book",
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
`, "PUBLISHED").Exec().Scan(&books)
if err != nil {
	log.Fatal(err)
}
```

### Query Single Row

`QueryRow()` requires a **struct** or primitive destination for `Scan()` (not a slice):

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
	WHERE deleted_at IS NULL
`).Exec().Scan(&stats)

// With primitive
var count int
err = client.Raw().QueryRow("SELECT COUNT(*) FROM authors WHERE deleted_at IS NULL").
	Exec().
	Scan(&count)
```

### Manual Row Iteration

If you need more control, you can iterate rows manually:

```go
rows, err := client.Raw().Query("SELECT id_author, first_name, last_name FROM authors").Exec().Rows()
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

if err := rows.Err(); err != nil {
	log.Fatal(err)
}
```

### Execute (INSERT, UPDATE, DELETE)

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

## Soft Deletes

If your model has `deletedAt` field:

```go
// Soft delete
err := client.Authors.Delete(...).Exec()

// Restore
err := client.Authors.Restore(...).Exec()

// Force delete (permanent)
err := client.Authors.ForceDelete(...).Exec()

// Include deleted records
users, err := client.Authors.FindMany().
	IncludeDeleted().Exec()

// Only deleted records
users, err := client.Authors.FindMany().
	OnlyDeleted().Exec()
```

## JSON Fields

```go
// Set JSON field
user, err := client.Authors.Update(
	...,
	db.UserUpdateInput{
		Metadata: db.JSON(map[string]interface{}{
			"key": "value",
		}),
	},
).Exec()

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
		Content: db.StringSearch("term"),
	},
).Exec()
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
client.Authors.BeforeCreate(func(user *db.User) error {
	// Validate or modify before creation
	return nil
})

// After create hook
client.Authors.AfterCreate(func(user *db.User) error {
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
// Using helper functions
if raw.IsNotFound(err) {
	// Record not found
}
if raw.IsUniqueConstraint(err) {
	// Duplicate key
}

// Using errors.Is
if errors.Is(err, raw.ErrNotFound) {
	// Record not found
}

// Using error code
var prismaErr *raw.PrismaError
if errors.As(err, &prismaErr) {
	switch prismaErr.Code {
	case "P2002":
		// Unique constraint
	case "P2025":
		// Not found
	}
}
```

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
err := client.Raw().Query("SELECT * FROM books WHERE id = $1", 999).Exec().Scan(&books)
// err == nil, books == []Book{}

// QueryRow returns ErrNotFound
var book Book
err = client.Raw().QueryRow("SELECT * FROM books WHERE id = $1", 999).Exec().Scan(&book)
// raw.IsNotFound(err) == true
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
