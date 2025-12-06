# Relationships Guide

Learn how to work with relationships in Prisma for Go.

## Overview

Relationships connect models and allow you to query related data efficiently.

## Types of Relationships

### One-to-Many

A user has many posts:

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

### Many-to-Many

Users can have many roles, roles can have many users:

```prisma
model User {
  id    Int     @id @default(autoincrement())
  email String  @unique
  roles Role[]
}

model Role {
  id    Int    @id @default(autoincrement())
  name  String @unique
  users User[]
}
```

### One-to-One

A user has one profile:

```prisma
model User {
  id      Int      @id @default(autoincrement())
  email   String   @unique
  profile Profile?
}

model Profile {
  id     Int    @id @default(autoincrement())
  bio    String?
  userId Int    @unique
  user   User   @relation(fields: [userId], references: [id])
}
```

## Querying Relationships

### Include Related Data

```go
// Include author when fetching posts (when implemented)
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		// Add conditions
	}).
	Exec(ctx)

// Note: Include functionality will be added in a future version
```

### Filter by Relation

```go
// Find posts by specific author
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		// Relations will be supported in future versions
		// Author: &inputs.UserWhereInput{
		// 	Email: db.String("user@example.com"),
		// },
	}).
	Exec(ctx)
```

### Filter Related Records

```go
// Find users who have posts (when relations are implemented)
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		// Relations will be supported in future versions
		// Posts: &inputs.PostListRelationFilter{
		// 	Some: inputs.PostWhereInput{
		// 		Published: db.Bool(true),
		// 	},
		// },
	}).
	Exec(ctx)
```

## Creating Related Records

### Connect Existing Record

```go
// Create post and connect to existing user
// Note: Relations will be fully supported in a future version
// For now, use the foreign key directly
post, err := client.Post.Create().
	Data(inputs.PostCreateInput{
		Title:    db.String("My Post"),
		AuthorId: db.Int(userID),
	}).
	Exec(ctx)
```

### Create Related Record

```go
// Create post and create new user
// First create user, then create post
user, err := client.User.Create().
	Data(inputs.UserCreateInput{
		Email: "newuser@example.com",
		Name:  db.String("New User"),
	}).
	Exec(ctx)

post, err := client.Post.Create().
	Data(inputs.PostCreateInput{
		Title:    db.String("My Post"),
		AuthorId: db.Int(user.ID),
	}).
	Exec(ctx)
```

### Connect or Create

```go
// Connect if exists, create if not
post, err := client.Posts().Create(db.PostCreateInput{
	Title: "My Post",
	Author: db.UserCreateNestedOneInput{
		ConnectOrCreate: &db.UserConnectOrCreateInput{
			Where: db.UserWhereUniqueInput{
				Email: db.String("user@example.com"),
			},
			Create: db.UserCreateInput{
				Email: "user@example.com",
				Name:  db.String("User"),
			},
		},
	},
}).Exec()
```

### Create Many Related Records

```go
// Create user with multiple posts
// First create user, then create posts
user, err := client.User.Create().
	Data(inputs.UserCreateInput{
		Email: "user@example.com",
	}).
	Exec(ctx)

// Then create posts
for _, title := range []string{"Post 1", "Post 2"} {
	_, err := client.Post.Create().
		Data(inputs.PostCreateInput{
			Title:    db.String(title),
			AuthorId: db.Int(user.ID),
		}).
		Exec(ctx)
}
```

### Connect Many Records

```go
// Create user and connect to existing posts
// First create user, then update posts
user, err := client.User.Create().
	Data(inputs.UserCreateInput{
		Email: "user@example.com",
	}).
	Exec(ctx)

// Then update posts to connect them
for _, postID := range []int{1, 2} {
	err := client.Post.Update().
		Where(inputs.PostWhereInput{
			Id: db.Int(postID),
		}).
		Data(inputs.PostUpdateInput{
			AuthorId: db.Int(user.ID),
		}).
		Exec(ctx)
}
```

## Updating Relationships

### Update Related Record

```go
// Update post's author
post, err := client.Posts().Update(
	db.PostWhereUniqueInput{ID: db.Int(postID)},
	db.PostUpdateInput{
		Author: db.UserUpdateNestedOneInput{
			Connect: &db.UserWhereUniqueInput{
				ID: db.Int(newAuthorID),
			},
		},
	},
).Exec()
```

### Disconnect Relation

```go
// Remove relation (set to null)
post, err := client.Posts().Update(
	db.PostWhereUniqueInput{ID: db.Int(postID)},
	db.PostUpdateInput{
		Author: db.UserUpdateNestedOneInput{
			Disconnect: db.Bool(true),
		},
	},
).Exec()
```

### Update Many Relations

```go
// Add posts to user
user, err := client.Users().Update(
	db.UserWhereUniqueInput{ID: db.Int(userID)},
	db.UserUpdateInput{
		Posts: db.PostUpdateNestedManyInput{
			Connect: []db.PostWhereUniqueInput{
				{ID: db.Int(3)},
				{ID: db.Int(4)},
			},
		},
	},
).Exec()

// Remove posts from user
user, err := client.Users().Update(
	db.UserWhereUniqueInput{ID: db.Int(userID)},
	db.UserUpdateInput{
		Posts: db.PostUpdateNestedManyInput{
			Disconnect: []db.PostWhereUniqueInput{
				{ID: db.Int(1)},
				{ID: db.Int(2)},
			},
		},
	},
).Exec()
```

## Many-to-Many Relationships

### Create with Relations

```go
// Create user with roles
user, err := client.Users().Create(db.UserCreateInput{
	Email: "user@example.com",
	Roles: db.RoleCreateNestedManyInput{
		Connect: []db.RoleWhereUniqueInput{
			{ID: db.Int(1)}, // Admin
			{ID: db.Int(2)}, // User
		},
	},
}).Exec()
```

### Add Relations

```go
// Add role to user
// Update user roles (when UserRole model exists)
// This will be simplified in future version
// For now, create/delete role assignments manually
```

### Remove Relations

```go
// Remove role from user
// Remove user roles (when UserRole model exists)
// This will be simplified in future version
// For now, delete role assignments manually
```

## One-to-One Relationships

### Create with Profile

```go
// Create user with profile
// First create user, then create profile
user, err := client.User.Create().
	Data(inputs.UserCreateInput{
		Email: "user@example.com",
	}).
	Exec(ctx)

profile, err := client.Profile.Create().
	Data(inputs.ProfileCreateInput{
		UserId: db.Int(user.ID),
		Bio:    db.String("My bio"),
	}).
	Exec(ctx)
```

### Update Profile

```go
// Update user's profile
err := client.Profile.Update().
	Where(inputs.ProfileWhereInput{
		UserId: db.Int(userID),
	}).
	Data(inputs.ProfileUpdateInput{
		Bio: db.String("Updated bio"),
	}).
	Exec(ctx)
```

## Cascading Deletes

### On Delete Cascade

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
  author    User     @relation(fields: [authorId], references: [id], onDelete: Cascade)
}
```

When user is deleted, posts are automatically deleted.

### On Delete Set Null

```prisma
model Post {
  id        Int      @id @default(autoincrement())
  title     String
  authorId  Int?
  author    User?    @relation(fields: [authorId], references: [id], onDelete: SetNull)
}
```

When user is deleted, `authorId` is set to null.

## Querying with Relations

### Count Related Records

```go
// Find users with post count
users, err := client.Users().FindMany().
	Include(db.UserIncludeInput{
		Posts: true,
	}).Exec()

for _, user := range users {
	fmt.Printf("User %s has %d posts\n", user.Email, len(user.Posts))
}
```

### Filter by Related Count

```go
// Find users with at least 5 posts
// Find users who have posts
// Note: Relation filters will be supported in future version
// For now, use a subquery or join with Raw SQL
users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		// Relation filters will be added in future version
	}).
	Exec(ctx)
```

### Order by Related Field

```go
// This requires a custom query or aggregation
// Use raw SQL for complex ordering
```

## Best Practices

### 1. Use Includes Wisely

Only include what you need:

```go
// Good: Only include needed relations
// Good: Select only needed fields
posts, err := client.Post.FindMany().
	Select(inputs.PostSelect{
		Id:    true,
		Title: true,
	}).
	Exec(ctx)

// Then fetch authors separately if needed
for _, post := range posts {
	author, _ := client.User.FindFirst().
		Select(inputs.UserSelect{
			Id:    true,
			Email: true,
		}).
		Where(inputs.UserWhereInput{
			Id: db.Int(post.AuthorId),
		}).
		Exec(ctx)
	// Use author data
}
```

### 2. Avoid N+1 Queries

```go
// Good: Single query with include
// Good: Fetch posts with authors in batch
posts, err := client.Post.FindMany().
	Select(inputs.PostSelect{
		Id:       true,
		Title:    true,
		AuthorId: true,
	}).
	Exec(ctx)

// Fetch all authors in one query
authorIDs := make([]int, len(posts))
for i, post := range posts {
	authorIDs[i] = post.AuthorId
}

// Note: Batch fetching will be optimized in future version

// Bad: N+1 queries
posts, err := client.Post.FindMany().Exec(ctx)
for _, post := range posts {
	author, _ := client.User.FindFirst().
		Where(inputs.UserWhereInput{
			Id: db.Int(post.AuthorId),
		}).
		Exec(ctx)
	// This creates N+1 queries
}
```

### 3. Use Transactions for Related Operations

```go
err := client.Transaction(func(tx *db.TransactionClient) error {
	// Create user
	user, err := tx.Users().Create(...).Exec()
	if err != nil {
		return err
	}

	// Create related posts
	for _, postData := range posts {
		_, err = tx.Posts().Create(db.PostCreateInput{
			Title:   postData.Title,
			AuthorId: user.ID,
		}).Exec()
		if err != nil {
			return err
		}
	}
	return nil
})
```

### 4. Handle Optional Relations

```go
// Check if relation exists
if user.Profile != nil {
	fmt.Printf("Bio: %s\n", user.Profile.Bio)
}
```

### 5. Use Unique Constraints

Always use unique constraints for one-to-one relations:

```prisma
model Profile {
  userId Int  @unique
  user   User @relation(...)
}
```

## Examples

### Complete Example

```go
// Create user with posts
user, err := client.Users().Create(db.UserCreateInput{
	Email: "author@example.com",
	Name:  db.String("Author"),
	Posts: db.PostCreateNestedManyInput{
		Create: []db.PostCreateInput{
			{
				Title:     "First Post",
				Content:   db.String("Content here"),
				Published: db.Bool(true),
			},
			{
				Title:     "Second Post",
				Content:   db.String("More content"),
				Published: db.Bool(false),
			},
		},
	},
}).Exec()

// Query with relations
// Find posts by author
author, err := client.User.FindFirst().
	Where(inputs.UserWhereInput{
		Email: db.String("author@example.com"),
	}).
	Exec(ctx)

posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		AuthorId:  db.Int(author.ID),
		Published: db.Bool(true),
	}).
	Exec(ctx)
}).Exec()

for _, post := range posts {
	fmt.Printf("%s by %s\n", post.Title, post.Author.Email)
}
```

## Next Steps

- Read [API Reference](API.md) for detailed query options
- Learn about [Migrations](MIGRATIONS.md) for schema changes
- Check [Best Practices](BEST_PRACTICES.md) for production code
