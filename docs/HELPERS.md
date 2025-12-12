# Helper Functions

This document describes the helper functions available in the generated Prisma Go Client.

## Overview

The Prisma Go Client generates helper functions to improve the developer experience when working with optional fields and filters. These helpers eliminate boilerplate code and make your code more readable.

## Pointer Helpers (`inputs` package)

### Purpose

In Go, optional fields in structs are represented as pointers. Without helpers, you need to create temporary variables to get pointers to values:

```go
// ❌ Verbose approach
name := "John Doe"
age := 30
active := true

user, err := client.User.Create().
    Data(inputs.UserCreateInput{
        Name:   &name,
        Age:    &age,
        Active: &active,
    }).
    Exec()
```

### Usage

The `inputs` package provides helper functions that return pointers directly:

```go
// ✅ Clean approach with helpers
user, err := client.User.Create().
    Data(inputs.UserCreateInput{
        Name:   inputs.String("John Doe"),
        Age:    inputs.Int(30),
        Active: inputs.Bool(true),
    }).
    Exec()
```

### Available Helpers

| Helper                           | Type              | Returns            |
| -------------------------------- | ----------------- | ------------------ |
| `inputs.String(v string)`        | `string`          | `*string`          |
| `inputs.Int(v int)`              | `int`             | `*int`             |
| `inputs.Int64(v int64)`          | `int64`           | `*int64`           |
| `inputs.Float(v float64)`        | `float64`         | `*float64`         |
| `inputs.Bool(v bool)`            | `bool`            | `*bool`            |
| `inputs.DateTime(v time.Time)`   | `time.Time`       | `*time.Time`       |
| `inputs.Json(v json.RawMessage)` | `json.RawMessage` | `*json.RawMessage` |
| `inputs.Bytes(v []byte)`         | `[]byte`          | `*[]byte`          |

### Examples

#### Create Operation

```go
package main

import (
    "context"
    "my-app/db"
    "my-app/db/inputs"
    "time"
)

func createUser(ctx context.Context, client *db.Client) error {
    user, err := client.User.WithContext(ctx).Create().
        Data(inputs.UserCreateInput{
            Email:     "user@example.com",
            Name:      inputs.String("John Doe"),
            Age:       inputs.Int(30),
            Bio:       inputs.String("Software developer"),
            IsActive:  inputs.Bool(true),
            BirthDate: inputs.DateTime(time.Date(1993, 5, 15, 0, 0, 0, 0, time.UTC)),
        }).
        Exec()

    if err != nil {
        return err
    }

    // ... use user
    return nil
}
```

#### Update Operation

```go
func updateUser(ctx context.Context, client *db.Client, userID int) error {
    user, err := client.User.WithContext(ctx).Update().
        Where(inputs.UserWhereInput{
            ID: inputs.Int(userID),
        }).
        Data(inputs.UserUpdateInput{
            Name:     inputs.String("Jane Doe"),
            Age:      inputs.Int(31),
            IsActive: inputs.Bool(false),
        }).
        Exec()

    if err != nil {
        return err
    }

    // ... use user
    return nil
}
```

## Filter Helpers (`filters` package)

### Purpose

Filter helpers provide a type-safe way to construct complex query filters. They are organized in a separate `filters` package.

### Usage

```go
import (
    "my-app/db/inputs"
    "my-app/db/filters"
)

users, err := client.User.WithContext(ctx).FindMany().
    Where(inputs.UserWhereInput{
        Email: filters.Contains("@example.com"),
        Name:  filters.StartsWith("John"),
    }).
    Exec()
```

### Available Filter Helpers

#### String Filters

```go
filters.String("value")                    // Equals
filters.Contains("substring")              // Contains substring
filters.StartsWith("prefix")               // Starts with prefix
filters.EndsWith("suffix")                 // Ends with suffix
filters.ContainsInsensitive("substring")   // Case-insensitive contains
filters.StartsWithInsensitive("prefix")    // Case-insensitive starts with
filters.EndsWithInsensitive("suffix")      // Case-insensitive ends with
filters.StringIn("val1", "val2", "val3")   // In list
filters.StringNotIn("val1", "val2")        // Not in list
```

#### Numeric Filters

```go
filters.Int(42)                 // Equals
filters.IntGt(10)              // Greater than
filters.IntGte(10)             // Greater than or equal
filters.IntLt(100)             // Less than
filters.IntLte(100)            // Less than or equal
filters.IntIn(1, 2, 3)         // In list
filters.IntNotIn(4, 5)         // Not in list

// Similar helpers available for Int64 and Float
filters.Int64(1000)
filters.Float(3.14)
```

#### Boolean Filters

```go
filters.Bool(true)   // Equals true
filters.Bool(false)  // Equals false
```

#### DateTime Filters

```go
import "time"

now := time.Now()
yesterday := now.AddDate(0, 0, -1)

filters.DateTime(now)               // Equals
filters.DateTimeGt(yesterday)       // Greater than (after)
filters.DateTimeGte(yesterday)      // Greater than or equal
filters.DateTimeLt(now)             // Less than (before)
filters.DateTimeLte(now)            // Less than or equal
```

### Complex Filter Examples

#### Combining Filters

```go
users, err := client.User.WithContext(ctx).FindMany().
    Where(inputs.UserWhereInput{
        Email:    filters.Contains("@company.com"),
        IsActive: filters.Bool(true),
        Age:      filters.IntGte(18),
        Or: []inputs.UserWhereInput{
            {Name: filters.StartsWith("John")},
            {Name: filters.StartsWith("Jane")},
        },
    }).
    Exec()
```

#### Nested Conditions

```go
users, err := client.User.WithContext(ctx).FindMany().
    Where(inputs.UserWhereInput{
        And: []inputs.UserWhereInput{
            {Email: filters.Contains("@example.com")},
            {IsActive: filters.Bool(true)},
        },
        Not: &inputs.UserWhereInput{
            Name: filters.Contains("test"),
        },
    }).
    Exec()
```

## Package Organization

The generated client organizes helpers into two packages:

```
db/
├── inputs/
│   ├── helpers.go              # Pointer helpers (String, Int, Bool, etc.)
│   ├── user_input.go            # Input types for User model
│   └── ...
├── filters/
│   ├── filters.go              # Filter type definitions
│   └── helpers.go              # Filter helpers (Contains, StartsWith, etc.)
└── ...
```

### Import Pattern

```go
import (
    "my-app/db"
    "my-app/db/inputs"   // For CreateInput, UpdateInput, WhereInput
    "my-app/db/filters"  // For filter helpers (optional)
)
```

## Best Practices

### 1. Use Helpers for Optional Fields

✅ **Do:**

```go
user, err := client.User.Create().
    Data(inputs.UserCreateInput{
        Email: "user@example.com",
        Name:  inputs.String("John"),  // Optional field
    }).
    Exec()
```

❌ **Don't:**

```go
name := "John"
user, err := client.User.Create().
    Data(inputs.UserCreateInput{
        Email: "user@example.com",
        Name:  &name,  // Unnecessary temporary variable
    }).
    Exec()
```

### 2. Required Fields Don't Need Helpers

Required fields (non-pointer) should be passed directly:

```go
user, err := client.User.Create().
    Data(inputs.UserCreateInput{
        Email: "user@example.com",      // Required - no helper needed
        Name:  inputs.String("John"),   // Optional - use helper
    }).
    Exec()
```

### 3. Import Only What You Need

If you only need pointer helpers:

```go
import "my-app/db/inputs"
```

If you also need filter helpers:

```go
import (
    "my-app/db/inputs"
    "my-app/db/filters"
)
```

### 4. Consistent Naming

The helpers follow Go naming conventions:

- Package function names are concise (`String`, `Int`, `Bool`)
- Always prefixed with package name in usage (`inputs.String()`, `filters.Contains()`)

## Type Safety

All helpers maintain Go's type safety:

```go
// ✅ Correct
name := inputs.String("John")  // *string

// ❌ Compile error
name := inputs.String(123)  // Type mismatch
```

## Performance

Helper functions are lightweight wrappers with minimal overhead:

```go
func String(v string) *string {
    return &v
}
```

The compiler optimizes these inline, resulting in no performance penalty compared to manual pointer creation.

## Migration Guide

### From Manual Pointers

If you have existing code with manual pointer creation:

**Before:**

```go
description := "My description"
client.Item.Create().Data(inputs.ItemCreateInput{
    Description: &description,
})
```

**After:**

```go
client.Item.Create().Data(inputs.ItemCreateInput{
    Description: inputs.String("My description"),
})
```

### Backward Compatibility

Using helpers is optional. The old approach with manual pointers continues to work:

```go
// Both approaches work
name1 := inputs.String("John")       // New way
name2 := "Jane"; ptr := &name2       // Old way
```

## Troubleshooting

### Helper Not Found

If you get "undefined: inputs.String":

1. Ensure you've run `prisma generate`
2. Check that `inputs/helpers.go` was generated
3. Import the inputs package: `import "my-app/db/inputs"`

### Wrong Package

If you try to use filter helpers from inputs package:

```go
// ❌ Wrong - Contains is in filters package
Email: inputs.Contains("@example.com")

// ✅ Correct
import "my-app/db/filters"
Email: filters.Contains("@example.com")
```

## See Also

- [API Reference](API.md)
- [Query Guide](QUICKSTART.md#querying-data)
- [Best Practices](BEST_PRACTICES.md)
