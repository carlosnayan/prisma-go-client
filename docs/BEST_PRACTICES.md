# Best Practices

Production-ready practices for using Prisma for Go.

## Database Connection

### Connection Pooling

Configure connection pool in `prisma.conf`:

```toml
[datasource]
url = "env('DATABASE_URL')"

[datasource.pool]
max_open_connections = 25
max_idle_connections = 5
connection_max_lifetime = "5m"
connection_max_idle_time = "10m"
```

### Health Checks

```go
// Check database health
healthy, err := db.CheckHealth(conn)
if !healthy {
	log.Fatal("Database unhealthy")
}
```

## Error Handling

### Always Handle Errors

```go
// Good
user, err := client.User.FindFirst().
	Where(inputs.UserWhereInput{
		Id: db.Int(1),
	}).
	Exec(ctx)
if err != nil {
	if errors.Is(err, db.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	return nil, fmt.Errorf("failed to find user: %w", err)
}

// Bad
user, _ := client.User.FindFirst().
	Where(inputs.UserWhereInput{Id: db.Int(1)}).
	Exec(ctx)
```

### Custom Error Types

```go
var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidInput = errors.New("invalid input")
)

func GetUser(id int) (*db.User, error) {
	user, err := client.User.FindFirst().
		Where(inputs.UserWhereInput{
			Id: db.Int(id),
		}).
		Exec(ctx)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
```

## Query Optimization

### Select Only Needed Fields

```go
// Good: Select specific fields using type-safe Select
users, err := client.User.FindMany().
	Select(inputs.UserSelect{
		Id:    true,
		Email: true,
		Name:  true,
	}).
	Exec(ctx)

// Bad: Select all fields
users, err := client.User.FindMany().Exec(ctx)
```

### Use Pagination

```go
// Good: Use Where with conditions to limit results
const pageSize = 20
page := 1

users, err := client.User.FindMany().
	Where(inputs.UserWhereInput{
		// Add conditions as needed
	}).
	Exec(ctx)

// Bad: Load all records without filtering
users, err := client.User.FindMany().Exec(ctx)
```

### Avoid N+1 Queries

```go
// Good: Use includes (when implemented)
posts, err := client.Post.FindMany().
	Where(inputs.PostWhereInput{
		// Add conditions
	}).
	Exec(ctx)

// Bad: N+1 queries
posts, err := client.Post.FindMany().Exec(ctx)
for _, post := range posts {
	author, _ := client.User.FindFirst().
		Where(inputs.UserWhereInput{
			Id: db.Int(post.AuthorId),
		}).
		Exec(ctx)
}
```

### Use Indexes

```prisma
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?  @index
  createdAt DateTime @default(now()) @index
}
```

## Transactions

### Use for Related Operations

Transactions ensure atomicity - either all operations succeed or all are rolled back. Use transactions when operations must be consistent together.

```go
// Good: Use transaction for related operations
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	user, err := tx.User.Create().
		Data(inputs.UserCreateInput{
			Email: "user@example.com",
			Name:  db.String("John Doe"),
		}).
		Exec(ctx)
	if err != nil {
		return err // Transaction automatically rolled back
	}

	_, err = tx.Post.Create().
		Data(inputs.PostCreateInput{
			AuthorId: db.String(user.ID),
			Title:    db.String("Post Title"),
		}).
		Exec(ctx)
	return err // If this fails, user creation is rolled back
})

// Bad: Separate operations without transaction
user, err := client.User.Create().
	Data(inputs.UserCreateInput{Email: "user@example.com"}).
	Exec(ctx)
if err != nil {
	return err
}

// If this fails, user exists but post doesn't - inconsistent state!
post, err := client.Post.Create().
	Data(inputs.PostCreateInput{AuthorId: user.ID}).
	Exec(ctx)
```

### Keep Transactions Short

Keep transactions as short as possible. Long transactions hold database locks and can cause performance issues.

```go
// Good: Short transaction with only database operations
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	// Only database operations
	_, err := tx.User.Create().
		Data(inputs.UserCreateInput{
			Email: "user@example.com",
		}).
		Exec(ctx)
	return err
})

// Bad: Long transaction with external calls
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	_, err := tx.User.Create().
		Data(inputs.UserCreateInput{Email: "user@example.com"}).
		Exec(ctx)
	if err != nil {
		return err
	}

	// External API call - keeps transaction open unnecessarily
	resp, err := http.Post("https://api.example.com", ...)
	if err != nil {
		return err
	}
	// Transaction is still open while processing response
	_ = resp
	return nil
})
```

### Handle Errors Properly

Always check for errors and let the transaction rollback automatically:

```go
// Good: Return errors to trigger rollback
err := client.Transaction(ctx, func(tx *db.TransactionClient) error {
	user, err := tx.User.Create().
		Data(inputs.UserCreateInput{
			Email: "user@example.com",
		}).
		Exec(ctx)
	if err != nil {
		return err // Transaction rolled back automatically
	}

	// Validate before continuing
	if user.Email == "" {
		return fmt.Errorf("invalid user") // Transaction rolled back
	}

	return nil // Transaction committed
})

if err != nil {
	// Handle error - transaction was already rolled back
	log.Printf("Transaction failed: %v", err)
}
```

## Input Validation

### Validate Before Database

```go
// Good: Validate input
func CreateUser(input inputs.UserCreateInput) error {
	if input.Email == "" {
		return ErrInvalidInput
	}
	if !isValidEmail(input.Email) {
		return ErrInvalidEmail
	}

	_, err := client.User.Create().
		Data(input).
		Exec(ctx)
	return err
}

// Bad: Let database handle validation
func CreateUser(input inputs.UserCreateInput) error {
	_, err := client.User.Create().
		Data(input).
		Exec(ctx)
	return err
}
```

### Use Struct Validation

```go
type UserInput struct {
	Email string `validate:"required,email"`
	Name  string `validate:"min=2,max=100"`
}

func CreateUser(input UserInput) error {
	if err := db.ValidateStruct(input); err != nil {
		return err
	}

	_, err := client.User.Create().
		Data(inputs.UserCreateInput{
			Email: input.Email,
			Name:  db.String(input.Name),
		}).
		Exec(ctx)
	return err
}
```

## Security

### Use Parameterized Queries

Prisma automatically uses parameterized queries, but for raw SQL:

```go
// Good: Parameterized
rows, err := conn.Query(ctx, "SELECT * FROM users WHERE id = $1", userID)

// Bad: String concatenation
query := fmt.Sprintf("SELECT * FROM users WHERE id = %d", userID)
rows, err := conn.Query(ctx, query)
```

### Sanitize Input

```go
// Always validate and sanitize user input
func SearchUsers(query string) ([]*db.User, error) {
	// Sanitize
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return nil, ErrInvalidInput
	}

	return client.User.FindMany().
		Where(inputs.UserWhereInput{
			Name: db.Contains(query),
		}).
		Exec(ctx)
}
```

### Use Environment Variables

```go
// Good: Use environment variables
dbURL := os.Getenv("DATABASE_URL")

// Bad: Hardcode credentials
dbURL := "postgresql://user:password@localhost/db"
```

## Logging

### Log Queries in Development

Enable query logging in `prisma.conf`:

```toml
[log]
levels = ["query", "info", "warn", "error"]
```

### Structured Logging

```go
logger.Info("User created",
	"user_id", user.ID,
	"email", user.Email,
)
```

## Performance

### Use Connection Pooling

```toml
[datasource.pool]
max_open_connections = 25
max_idle_connections = 5
```

### Monitor Query Performance

```go
start := time.Now()
users, err := client.User.FindMany().Exec(ctx)
duration := time.Since(start)

if duration > 1*time.Second {
	logger.Warn("Slow query", "duration", duration)
}
```

### Use Indexes Strategically

```prisma
// Index frequently queried fields
model User {
  email     String   @unique  // Automatically indexed
  name      String?  @index   // Add index for searches
  createdAt DateTime @index   // Index for sorting
}
```

## Testing

### Use Test Database

```go
func TestUserCreation(t *testing.T) {
	// Use separate test database
	testDB := setupTestDB(t)
	defer teardownTestDB(t, testDB)

	client := db.NewClient(testDB)

	user, err := client.User.Create().
		Data(inputs.UserCreateInput{
			Email: "test@example.com",
		}).
		Exec(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, user)
}
```

### Clean Up After Tests

```go
func setupTestDB(t *testing.T) *sql.DB {
	db := connectTestDB()

	// Run migrations
	runMigrations(db)

	// Cleanup
	t.Cleanup(func() {
		cleanupTestDB(db)
	})

	return db
}
```

## Code Organization

### Repository Pattern

```go
type UserRepository struct {
	client *db.Client
}

func (r *UserRepository) FindByID(id int) (*db.User, error) {
	return r.client.User.FindFirst().
		Where(inputs.UserWhereInput{
			Id: db.Int(id),
		}).
		Exec(ctx)
}

func (r *UserRepository) Create(input inputs.UserCreateInput) (*db.User, error) {
	return r.client.User.Create().
		Data(input).
		Exec(ctx)
}
```

### Service Layer

```go
type UserService struct {
	repo *UserRepository
}

func (s *UserService) CreateUser(email, name string) (*db.User, error) {
	// Validation
	if !isValidEmail(email) {
		return nil, ErrInvalidEmail
	}

	// Business logic
	// Database operation
	return s.repo.Create(db.UserCreateInput{
		Email: email,
		Name:  db.String(name),
	})
}
```

## Migration Best Practices

### Review Migrations

Always review generated SQL:

```bash
prisma migrate dev --name migration_name
# Review migration.sql before applying
```

### Test Migrations

```bash
# Test in development
prisma migrate reset
prisma migrate deploy
```

### Backup Before Production

```bash
# Always backup before production migrations
pg_dump $DATABASE_URL > backup.sql
prisma migrate deploy
```

## Error Messages

### User-Friendly Messages

```go
// Good: User-friendly error
if errors.Is(err, db.ErrNotFound) {
	return nil, fmt.Errorf("user not found")
}

// Bad: Technical error
if errors.Is(err, db.ErrNotFound) {
	return nil, err
}
```

### Log Technical Details

```go
// Log technical details, return user-friendly message
if err != nil {
	logger.Error("Failed to create user",
		"error", err,
		"input", input,
	)
	return nil, fmt.Errorf("failed to create user")
}
```

## Monitoring

### Health Checks

```go
// Regular health checks
func healthCheck() error {
	healthy, err := db.CheckHealth(conn)
	if !healthy {
		return fmt.Errorf("database unhealthy: %w", err)
	}
	return nil
}
```

### Metrics

```go
// Track query metrics
func trackQuery(operation string, duration time.Duration) {
	metrics.RecordQuery(operation, duration)
	if duration > 1*time.Second {
		metrics.RecordSlowQuery(operation, duration)
	}
}
```

## Documentation

### Document Complex Queries

```go
// FindUsersWithActivePosts finds users who have at least one
// published post created in the last 30 days
func FindUsersWithActivePosts() ([]*db.User, error) {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	return client.User.FindMany().
		Where(inputs.UserWhereInput{
			// Relation filters will be added in future version
			// For now, use raw SQL or separate queries
		}).
		Exec(ctx)
}
```

## Next Steps

- Read [API Reference](API.md)
- Learn about [Migrations](MIGRATIONS.md)
- Explore [Relationships](RELATIONSHIPS.md)
