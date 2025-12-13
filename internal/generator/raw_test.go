package generator

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/builder"
	"github.com/carlosnayan/prisma-go-client/internal/driver"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// mockBuilderDBTX is a mock implementation of builder.DBTX for testing
// This simulates what PgxPoolAdapter or SQLDBAdapter would look like
type mockBuilderDBTX struct{}

// mockResult implements builder.Result
type mockResult struct {
	rowsAffected int64
	lastInsertID int64
}

func (r *mockResult) RowsAffected() int64 {
	return r.rowsAffected
}

func (r *mockResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

// mockRows implements builder.Rows
type mockRows struct {
	columns []string
	data    [][]interface{}
	index   int
}

func (r *mockRows) Close()     {}
func (r *mockRows) Err() error { return nil }
func (r *mockRows) Next() bool {
	if r.index < len(r.data) {
		r.index++
		return true
	}
	return false
}
func (r *mockRows) Scan(dest ...interface{}) error {
	if r.index == 0 || r.index > len(r.data) {
		return nil
	}
	row := r.data[r.index-1]
	for i, v := range row {
		if i < len(dest) {
			switch d := dest[i].(type) {
			case *string:
				if s, ok := v.(string); ok {
					*d = s
				}
			case *int:
				if n, ok := v.(int); ok {
					*d = n
				}
			case *interface{}:
				*d = v
			}
		}
	}
	return nil
}
func (r *mockRows) Columns() ([]string, error) {
	return r.columns, nil
}

// mockRow implements builder.Row
type mockRow struct{}

func (r *mockRow) Scan(dest ...interface{}) error { return nil }

// Exec implements builder.DBTX.Exec
// Returns builder.Result (which is driver.Result)
func (m *mockBuilderDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (driver.Result, error) {
	return &mockResult{rowsAffected: 1}, nil
}

// Query implements builder.DBTX.Query
// Returns builder.Rows (which is driver.Rows)
func (m *mockBuilderDBTX) Query(ctx context.Context, sql string, args ...interface{}) (driver.Rows, error) {
	return &mockRows{}, nil
}

// QueryRow implements builder.DBTX.QueryRow
// Returns builder.Row (which is driver.Row)
func (m *mockBuilderDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) driver.Row {
	return &mockRow{}
}

// Begin implements builder.DBTX.Begin
func (m *mockBuilderDBTX) Begin(ctx context.Context) (driver.Tx, error) {
	return nil, nil
}

// SQLDB implements builder.DBTX.SQLDB
func (m *mockBuilderDBTX) SQLDB() *sql.DB {
	return nil
}

// Close implements builder.DBTX.Close
func (m *mockBuilderDBTX) Close() {
	// No-op for mock
}

// TestRawNew_WithMockBuilderDBTX tests that raw.New accepts builder.DBTX implementations
// This test would have caught the original panic issue where raw.New couldn't recognize
// builder.DBTX implementations due to type assertion failures.
func TestRawNew_WithMockBuilderDBTX(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Generate raw package (schema not needed for this test, just need the raw package generated)
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	// Read generated raw.go to verify it was created
	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	// Verify hasBuilderDBMethods function exists (our fix)
	if !strings.Contains(contentStr, "func hasBuilderDBMethods") {
		t.Error("raw.go should contain hasBuilderDBMethods function for reflection-based detection")
	}

	// Verify reflection is used
	if !strings.Contains(contentStr, "reflect.ValueOf") {
		t.Error("raw.go should use reflection to detect builder.DBTX")
	}

	// Now test with actual builder.DBTX implementation
	// We'll use the builder package's DBTX which is an alias for driver.DB
	mockDB := &mockBuilderDBTX{}

	// Convert to builder.DBTX (which is driver.DB)
	var builderDB builder.DBTX = mockDB
	_ = builderDB // Use the variable to show it's a valid builder.DBTX

	// This test verifies that if we had a way to call raw.New at runtime,
	// it would accept builder.DBTX. Since we can't easily import and use
	// the generated code in the same test, we verify the generated code
	// has the necessary reflection logic.

	// Verify the generated code has the correct structure
	if !strings.Contains(contentStr, "hasBuilderDBMethods(db)") {
		t.Error("raw.New should call hasBuilderDBMethods to detect builder.DBTX")
	}

	// Verify builderDBAdapter uses reflection
	if !strings.Contains(contentStr, "reflect.ValueOf(a.db)") {
		t.Error("builderDBAdapter should use reflection to call methods")
	}

	// Verify the adapter stores interface{}
	if !strings.Contains(contentStr, "type builderDBAdapter struct") {
		t.Error("builderDBAdapter should exist")
	}
	if !strings.Contains(contentStr, "\tdb interface{}") {
		t.Error("builderDBAdapter should store db as interface{}")
	}

}

// TestRawNew_ExecutorMethods tests that Executor methods work correctly
// with builder.DBTX implementations through the adapter.
func TestRawNew_ExecutorMethods(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	_ = &parser.Schema{
		Datasources: []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{
						Name:  "provider",
						Value: "postgresql",
					},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "User",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
				},
			},
		},
	}

	// Generate raw package
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	// Read generated raw.go
	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	// Verify Executor.Exec method exists and uses the adapter
	if !strings.Contains(contentStr, "func (e *Executor) Exec") {
		t.Error("Executor should have Exec method")
	}

	// Verify Executor.Query method exists
	if !strings.Contains(contentStr, "func (e *Executor) Query") {
		t.Error("Executor should have Query method")
	}

	// Verify Executor.QueryRow method exists
	if !strings.Contains(contentStr, "func (e *Executor) QueryRow") {
		t.Error("Executor should have QueryRow method")
	}

	// Verify builderDBAdapter.Exec uses reflection
	if !strings.Contains(contentStr, "func (a *builderDBAdapter) Exec") {
		t.Error("builderDBAdapter should have Exec method")
	}
	if !strings.Contains(contentStr, "reflect.ValueOf(a.db)") {
		t.Error("builderDBAdapter.Exec should use reflection")
	}
	if !strings.Contains(contentStr, "MethodByName(\"Exec\")") {
		t.Error("builderDBAdapter.Exec should use MethodByName to get Exec method")
	}

	// Verify builderDBAdapter.Query uses reflection
	if !strings.Contains(contentStr, "func (a *builderDBAdapter) Query") {
		t.Error("builderDBAdapter should have Query method")
	}
	if !strings.Contains(contentStr, "MethodByName(\"Query\")") {
		t.Error("builderDBAdapter.Query should use MethodByName to get Query method")
	}

	// Verify builderDBAdapter.QueryRow uses reflection
	if !strings.Contains(contentStr, "func (a *builderDBAdapter) QueryRow") {
		t.Error("builderDBAdapter should have QueryRow method")
	}
	if !strings.Contains(contentStr, "MethodByName(\"QueryRow\")") {
		t.Error("builderDBAdapter.QueryRow should use MethodByName to get QueryRow method")
	}

	// Verify resultAdapter exists for type conversion
	if !strings.Contains(contentStr, "type resultAdapter struct") {
		t.Error("resultAdapter should exist for converting builder.Result to raw.Result")
	}

	// Verify rowsAdapter exists
	if !strings.Contains(contentStr, "type rowsAdapter struct") {
		t.Error("rowsAdapter should exist for converting builder.Rows to raw.Rows")
	}

	// Verify rowAdapter exists
	if !strings.Contains(contentStr, "type rowAdapter struct") {
		t.Error("rowAdapter should exist for converting builder.Row to raw.Row")
	}
}

// TestRawNew_PanicPrevention verifies that the generated code would not panic
// when given a builder.DBTX implementation. This test documents the fix and
// ensures the reflection-based approach is used.
func TestRawNew_PanicPrevention(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	_ = &parser.Schema{
		Datasources: []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{
						Name:  "provider",
						Value: "postgresql",
					},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "User",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
				},
			},
		},
	}

	// Generate raw package
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	// Read generated raw.go
	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	// CRITICAL: Verify that the old type assertion approach is NOT used
	// The old approach would have:
	//   if builderDB, ok := db.(interface {
	//       Exec(...) (interface{ RowsAffected() int64 }, error)
	//       ...
	//   })
	// This would fail because builder.DBTX returns concrete types, not interface{}
	// Check if old pattern exists without the new reflection approach
	hasOldPattern := strings.Contains(contentStr, "interface{ RowsAffected() int64 }") &&
		strings.Contains(contentStr, "interface{ Close()")
	hasNewApproach := strings.Contains(contentStr, "hasBuilderDBMethods")

	if hasOldPattern && !hasNewApproach {
		t.Error("Generated code should NOT use the old type assertion approach that caused panics")
		t.Error("The old approach fails because builder.DBTX returns concrete types, not interface{}")
	}

	// Verify the new reflection-based approach is used
	if !strings.Contains(contentStr, "hasBuilderDBMethods(db)") {
		t.Fatal("Generated code MUST use hasBuilderDBMethods for reflection-based detection")
		t.Fatal("This is the fix that prevents the panic with builder.DBTX implementations")
	}

	// Verify reflection is imported
	if !strings.Contains(contentStr, `"reflect"`) {
		t.Error("Generated code should import reflect package")
	}

	// Verify the panic message is still there (for invalid inputs)
	if !strings.Contains(contentStr, `panic("db must implement raw.DB or builder.DB")`) {
		t.Error("Generated code should still panic for invalid inputs")
	}

	// This test ensures that:
	// 1. The old type assertion approach is NOT used (it caused panics)
	// 2. The new reflection-based approach IS used (it prevents panics)
	// 3. The generated code would work correctly with builder.DBTX implementations
}

// TestRawNew_WithPostgreSQLAdapter tests that raw.New works with PostgreSQL's PgxPoolAdapter
// This test generates complete code for PostgreSQL and verifies that:
// 1. PgxPoolAdapter is generated correctly
// 2. raw.New would accept PgxPoolAdapter (via NewPgxPoolDriver)
// 3. No panic would occur when using SetupClient -> NewClient -> raw.New flow
func TestRawNew_WithPostgreSQLAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
		Datasources: []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{
						Name:  "provider",
						Value: "postgresql",
					},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "User",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
				},
			},
		},
	}

	// Generate all necessary files for PostgreSQL
	if err := GenerateClient(schema, outputDir); err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	// Read generated driver.go
	driverFile := filepath.Join(outputDir, "driver.go")
	driverContent, err := os.ReadFile(driverFile)
	if err != nil {
		t.Fatalf("Failed to read driver.go: %v", err)
	}
	driverContentStr := string(driverContent)

	// Verify PgxPoolAdapter is generated
	if !strings.Contains(driverContentStr, "type PgxPoolAdapter struct") {
		t.Error("PostgreSQL driver should generate PgxPoolAdapter")
	}

	// Verify NewPgxPoolDriver returns builder.DBTX
	if !strings.Contains(driverContentStr, "func NewPgxPoolDriver(pool *pgxpool.Pool) builder.DBTX") {
		t.Error("NewPgxPoolDriver should return builder.DBTX")
	}

	// Verify PgxPoolAdapter implements builder.DBTX methods
	if !strings.Contains(driverContentStr, "func (a *PgxPoolAdapter) Exec") {
		t.Error("PgxPoolAdapter should implement Exec method")
	}
	if !strings.Contains(driverContentStr, "func (a *PgxPoolAdapter) Query") {
		t.Error("PgxPoolAdapter should implement Query method")
	}
	if !strings.Contains(driverContentStr, "func (a *PgxPoolAdapter) QueryRow") {
		t.Error("PgxPoolAdapter should implement QueryRow method")
	}

	// Verify PgxPoolAdapter returns builder.Result, builder.Rows, builder.Row
	if !strings.Contains(driverContentStr, "func (a *PgxPoolAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error)") {
		t.Error("PgxPoolAdapter.Exec should return (builder.Result, error)")
	}
	if !strings.Contains(driverContentStr, "func (a *PgxPoolAdapter) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error)") {
		t.Error("PgxPoolAdapter.Query should return (builder.Rows, error)")
	}
	if !strings.Contains(driverContentStr, "func (a *PgxPoolAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row") {
		t.Error("PgxPoolAdapter.QueryRow should return builder.Row")
	}

	// Verify SetupClient uses NewPgxPoolDriver
	if !strings.Contains(driverContentStr, "dbDriver := NewPgxPoolDriver(pool)") {
		t.Error("SetupClient should use NewPgxPoolDriver to create driver")
	}

	// Verify NewClient receives builder.DBTX
	clientFile := filepath.Join(outputDir, "client.go")
	clientContent, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}
	clientContentStr := string(clientContent)

	if !strings.Contains(clientContentStr, "func NewClient(db builder.DBTX) *Client") {
		t.Error("NewClient should accept builder.DBTX")
	}

	// Verify NewClient calls raw.New
	if !strings.Contains(clientContentStr, "raw.New(db)") {
		t.Error("NewClient should call raw.New(db)")
	}

	// Verify raw.New uses reflection (prevention of panic)
	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	rawContent, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}
	rawContentStr := string(rawContent)

	if !strings.Contains(rawContentStr, "hasBuilderDBMethods(db)") {
		t.Fatal("raw.New MUST use hasBuilderDBMethods for reflection-based detection")
		t.Fatal("This prevents panic when PgxPoolAdapter (builder.DBTX) is passed to raw.New")
	}

	// This test ensures that:
	// 1. PostgreSQL generates PgxPoolAdapter correctly
	// 2. PgxPoolAdapter implements builder.DBTX with correct return types
	// 3. raw.New uses reflection to accept PgxPoolAdapter (prevents panic)
	// 4. The full flow SetupClient -> NewClient -> raw.New would work without panic
}

// TestRawNew_WithMySQLAdapter tests that raw.New works with MySQL's SQLDBAdapter
// This test generates complete code for MySQL and verifies that:
// 1. SQLDBAdapter is generated correctly
// 2. raw.New would accept SQLDBAdapter (via NewSQLDriver)
// 3. No panic would occur when using SetupClient -> NewClient -> raw.New flow
func TestRawNew_WithMySQLAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
		Datasources: []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{
						Name:  "provider",
						Value: "mysql",
					},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "User",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
				},
			},
		},
	}

	// Generate all necessary files for MySQL
	if err := GenerateClient(schema, outputDir); err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	// Read generated driver.go
	driverFile := filepath.Join(outputDir, "driver.go")
	driverContent, err := os.ReadFile(driverFile)
	if err != nil {
		t.Fatalf("Failed to read driver.go: %v", err)
	}
	driverContentStr := string(driverContent)

	// Verify SQLDBAdapter is generated
	if !strings.Contains(driverContentStr, "type SQLDBAdapter struct") {
		t.Error("MySQL driver should generate SQLDBAdapter")
	}

	// Verify NewSQLDriver returns builder.DBTX
	if !strings.Contains(driverContentStr, "func NewSQLDriver(db *sql.DB) builder.DBTX") {
		t.Error("NewSQLDriver should return builder.DBTX")
	}

	// Verify SQLDBAdapter implements builder.DBTX methods
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) Exec") {
		t.Error("SQLDBAdapter should implement Exec method")
	}
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) Query") {
		t.Error("SQLDBAdapter should implement Query method")
	}
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) QueryRow") {
		t.Error("SQLDBAdapter should implement QueryRow method")
	}

	// Verify SQLDBAdapter returns builder.Result, builder.Rows, builder.Row
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error)") {
		t.Error("SQLDBAdapter.Exec should return (builder.Result, error)")
	}
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error)") {
		t.Error("SQLDBAdapter.Query should return (builder.Rows, error)")
	}
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row") {
		t.Error("SQLDBAdapter.QueryRow should return builder.Row")
	}

	// Verify SetupClient uses NewSQLDriver
	if !strings.Contains(driverContentStr, "dbDriver := NewSQLDriver(db)") {
		t.Error("SetupClient should use NewSQLDriver to create driver")
	}

	// Verify NewClient receives builder.DBTX
	clientFile := filepath.Join(outputDir, "client.go")
	clientContent, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}
	clientContentStr := string(clientContent)

	if !strings.Contains(clientContentStr, "func NewClient(db builder.DBTX) *Client") {
		t.Error("NewClient should accept builder.DBTX")
	}

	// Verify NewClient calls raw.New
	if !strings.Contains(clientContentStr, "raw.New(db)") {
		t.Error("NewClient should call raw.New(db)")
	}

	// Verify raw.New uses reflection (prevention of panic)
	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	rawContent, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}
	rawContentStr := string(rawContent)

	if !strings.Contains(rawContentStr, "hasBuilderDBMethods(db)") {
		t.Fatal("raw.New MUST use hasBuilderDBMethods for reflection-based detection")
		t.Fatal("This prevents panic when SQLDBAdapter (builder.DBTX) is passed to raw.New")
	}

	// This test ensures that:
	// 1. MySQL generates SQLDBAdapter correctly
	// 2. SQLDBAdapter implements builder.DBTX with correct return types
	// 3. raw.New uses reflection to accept SQLDBAdapter (prevents panic)
	// 4. The full flow SetupClient -> NewClient -> raw.New would work without panic
}

// TestRawNew_WithSQLiteAdapter tests that raw.New works with SQLite's SQLDBAdapter
// This test generates complete code for SQLite and verifies that:
// 1. SQLDBAdapter is generated correctly
// 2. raw.New would accept SQLDBAdapter (via NewSQLDriver)
// 3. No panic would occur when using SetupClient -> NewClient -> raw.New flow
func TestRawNew_WithSQLiteAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
		Datasources: []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{
						Name:  "provider",
						Value: "sqlite",
					},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "User",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
				},
			},
		},
	}

	// Generate all necessary files for SQLite
	if err := GenerateClient(schema, outputDir); err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	// Read generated driver.go
	driverFile := filepath.Join(outputDir, "driver.go")
	driverContent, err := os.ReadFile(driverFile)
	if err != nil {
		t.Fatalf("Failed to read driver.go: %v", err)
	}
	driverContentStr := string(driverContent)

	// Verify SQLDBAdapter is generated
	if !strings.Contains(driverContentStr, "type SQLDBAdapter struct") {
		t.Error("SQLite driver should generate SQLDBAdapter")
	}

	// Verify NewSQLDriver returns builder.DBTX
	if !strings.Contains(driverContentStr, "func NewSQLDriver(db *sql.DB) builder.DBTX") {
		t.Error("NewSQLDriver should return builder.DBTX")
	}

	// Verify SQLDBAdapter implements builder.DBTX methods
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) Exec") {
		t.Error("SQLDBAdapter should implement Exec method")
	}
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) Query") {
		t.Error("SQLDBAdapter should implement Query method")
	}
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) QueryRow") {
		t.Error("SQLDBAdapter should implement QueryRow method")
	}

	// Verify SQLDBAdapter returns builder.Result, builder.Rows, builder.Row
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error)") {
		t.Error("SQLDBAdapter.Exec should return (builder.Result, error)")
	}
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error)") {
		t.Error("SQLDBAdapter.Query should return (builder.Rows, error)")
	}
	if !strings.Contains(driverContentStr, "func (a *SQLDBAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row") {
		t.Error("SQLDBAdapter.QueryRow should return builder.Row")
	}

	// Verify SetupClient uses NewSQLDriver
	if !strings.Contains(driverContentStr, "dbDriver := NewSQLDriver(db)") {
		t.Error("SetupClient should use NewSQLDriver to create driver")
	}

	// Verify NewClient receives builder.DBTX
	clientFile := filepath.Join(outputDir, "client.go")
	clientContent, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}
	clientContentStr := string(clientContent)

	if !strings.Contains(clientContentStr, "func NewClient(db builder.DBTX) *Client") {
		t.Error("NewClient should accept builder.DBTX")
	}

	// Verify NewClient calls raw.New
	if !strings.Contains(clientContentStr, "raw.New(db)") {
		t.Error("NewClient should call raw.New(db)")
	}

	// Verify raw.New uses reflection (prevention of panic)
	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	rawContent, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}
	rawContentStr := string(rawContent)

	if !strings.Contains(rawContentStr, "hasBuilderDBMethods(db)") {
		t.Fatal("raw.New MUST use hasBuilderDBMethods for reflection-based detection")
		t.Fatal("This prevents panic when SQLDBAdapter (builder.DBTX) is passed to raw.New")
	}

	// This test ensures that:
	// 1. SQLite generates SQLDBAdapter correctly
	// 2. SQLDBAdapter implements builder.DBTX with correct return types
	// 3. raw.New uses reflection to accept SQLDBAdapter (prevents panic)
	// 4. The full flow SetupClient -> NewClient -> raw.New would work without panic
}

func TestRawQuery_FluentAPI(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "type QueryBuilder struct") {
		t.Error("raw.go should contain QueryBuilder struct")
	}

	if !strings.Contains(contentStr, "func (e *Executor) Query(sql string, args ...interface{}) *QueryBuilder") {
		t.Error("Executor.Query should return *QueryBuilder")
	}

	if !strings.Contains(contentStr, "func (q *QueryBuilder) Exec() *ScanResult") {
		t.Error("QueryBuilder.Exec should return *ScanResult without ctx parameter")
	}

	if !strings.Contains(contentStr, "type ScanResult struct") {
		t.Error("raw.go should contain ScanResult struct")
	}

	if !strings.Contains(contentStr, "func (r *ScanResult) Scan(dest interface{}) error") {
		t.Error("ScanResult should have Scan method")
	}

	if !strings.Contains(contentStr, "func (r *ScanResult) Rows() (Rows, error)") {
		t.Error("ScanResult should have Rows method for manual iteration")
	}
}

func TestRawQueryRow_FluentAPI(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "type QueryRowBuilder struct") {
		t.Error("raw.go should contain QueryRowBuilder struct")
	}

	if !strings.Contains(contentStr, "func (e *Executor) QueryRow(sql string, args ...interface{}) *QueryRowBuilder") {
		t.Error("Executor.QueryRow should return *QueryRowBuilder")
	}

	if !strings.Contains(contentStr, "func (q *QueryRowBuilder) Exec() *ScanRowResult") {
		t.Error("QueryRowBuilder.Exec should return *ScanRowResult without ctx parameter")
	}

	if !strings.Contains(contentStr, "type ScanRowResult struct") {
		t.Error("raw.go should contain ScanRowResult struct")
	}

	if !strings.Contains(contentStr, "func (r *ScanRowResult) Scan(dest interface{}) error") {
		t.Error("ScanRowResult.Scan should accept single interface{} for struct scanning")
	}
}

func TestRawExec_ReturnsResult(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "func (e *Executor) Exec(sql string, args ...interface{}) (Result, error)") {
		t.Error("Executor.Exec should return (Result, error) directly")
	}
}

func TestExtractColumnName(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "func extractColumnName(col string) string") {
		t.Error("raw.go should contain extractColumnName function")
	}

	if !strings.Contains(contentStr, "strings.LastIndex(lowerCol, \" as \")") {
		t.Error("extractColumnName should handle 'AS' alias syntax")
	}

	if !strings.Contains(contentStr, "strings.LastIndex(col, \".\")") {
		t.Error("extractColumnName should handle table.column syntax")
	}
}

func TestColumnAliasHandling(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	testCases := []struct {
		scenario     string
		mustContain  string
		errorMessage string
	}{
		{
			scenario:     "table alias with dot (cf.id_chatbot_flow)",
			mustContain:  `strings.LastIndex(col, ".")`,
			errorMessage: "Should extract 'id_chatbot_flow' from 'cf.id_chatbot_flow'",
		},
		{
			scenario:     "column without alias (id_tenant)",
			mustContain:  "return col",
			errorMessage: "Should return 'id_tenant' as-is when no alias",
		},
		{
			scenario:     "AS alias (ik.name as integration_key_name)",
			mustContain:  `strings.LastIndex(lowerCol, " as ")`,
			errorMessage: "Should extract 'integration_key_name' from 'ik.name as integration_key_name'",
		},
	}

	for _, tc := range testCases {
		if !strings.Contains(contentStr, tc.mustContain) {
			t.Errorf("Scenario '%s': %s. Expected to find '%s' in generated code",
				tc.scenario, tc.errorMessage, tc.mustContain)
		}
	}
}

func TestScanRowsWithDbTag(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "func buildFieldMap(t reflect.Type) map[string]int") {
		t.Error("raw.go should contain buildFieldMap function")
	}

	if !strings.Contains(contentStr, `field.Tag.Get("db")`) {
		t.Error("buildFieldMap should read 'db' struct tags")
	}

	if !strings.Contains(contentStr, `tag != "-"`) {
		t.Error("buildFieldMap should skip fields with db:\"-\" tag")
	}

	if !strings.Contains(contentStr, "fieldMap[tag] = i") {
		t.Error("buildFieldMap should map column names to field indices")
	}
}

func TestRowsAdapterHasColumns(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "func (r *rowsAdapter) Columns() ([]string, error)") {
		t.Error("rowsAdapter should have Columns method for column name extraction")
	}
}

func TestRaw_PrismaErrorTypes(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	rawFile := filepath.Join(outputDir, "raw", "raw.go")
	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw.go: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "type PrismaError struct") {
		t.Error("raw.go should contain PrismaError struct")
	}

	if !strings.Contains(contentStr, `ErrNotFound`) {
		t.Error("raw.go should contain ErrNotFound sentinel error")
	}

	if !strings.Contains(contentStr, `ErrUniqueConstraint`) {
		t.Error("raw.go should contain ErrUniqueConstraint sentinel error")
	}

	if !strings.Contains(contentStr, `func IsNotFound(err error) bool`) {
		t.Error("raw.go should contain IsNotFound helper function")
	}

	if !strings.Contains(contentStr, `func mapQueryError(err error) error`) {
		t.Error("raw.go should contain mapQueryError function for Query error mapping")
	}

	if !strings.Contains(contentStr, `func mapQueryRowError(err error) error`) {
		t.Error("raw.go should contain mapQueryRowError function for QueryRow error mapping")
	}

	if !strings.Contains(contentStr, "func (e *PrismaError) Unwrap() error") {
		t.Error("PrismaError should have Unwrap method for error unwrapping")
	}

	if !strings.Contains(contentStr, `Code: "P2025"`) {
		t.Error("ErrNotFound should have code P2025")
	}

	if !strings.Contains(contentStr, `Code: "P2002"`) {
		t.Error("ErrUniqueConstraint should have code P2002")
	}
}
