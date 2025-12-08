package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func TestSetupClient_GeneratedForPostgreSQL(t *testing.T) {
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

	// Generate all necessary files
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
	content, err := os.ReadFile(driverFile)
	if err != nil {
		t.Fatalf("Failed to read driver.go: %v", err)
	}

	contentStr := string(content)

	// Verify SetupClient function exists
	if !strings.Contains(contentStr, "func SetupClient(ctx context.Context, databaseURL ...string)") {
		t.Error("SetupClient function should be generated")
	}

	// Verify it returns (*Client, *pgxpool.Pool, error) for PostgreSQL
	if !strings.Contains(contentStr, "func SetupClient(ctx context.Context, databaseURL ...string) (*Client, *pgxpool.Pool, error)") {
		t.Error("SetupClient should return (*Client, *pgxpool.Pool, error) for PostgreSQL")
	}

	// Verify priority 1: explicit parameter
	if !strings.Contains(contentStr, "// Priority 1: Explicit parameter") {
		t.Error("SetupClient should check explicit parameter first")
	}
	if !strings.Contains(contentStr, "if len(databaseURL) > 0 && databaseURL[0] != \"\"") {
		t.Error("SetupClient should check if databaseURL parameter is provided")
	}

	// Verify priority 2: prisma.conf
	if !strings.Contains(contentStr, "// Priority 2: Read from prisma.conf") {
		t.Error("SetupClient should read from prisma.conf as fallback")
	}
	if !strings.Contains(contentStr, "url, err = getDatabaseURLFromConfig()") {
		t.Error("SetupClient should call getDatabaseURLFromConfig()")
	}

	// Verify error when URL is empty
	if !strings.Contains(contentStr, "DATABASE_URL is not set") {
		t.Error("SetupClient should return error when DATABASE_URL is not set")
	}

	// Verify NewPgxPoolFromURL is called
	if !strings.Contains(contentStr, "NewPgxPoolFromURL(ctx, url)") {
		t.Error("SetupClient should call NewPgxPoolFromURL for PostgreSQL")
	}

	// Verify NewPgxPoolDriver is called
	if !strings.Contains(contentStr, "NewPgxPoolDriver(pool)") {
		t.Error("SetupClient should call NewPgxPoolDriver to create adapter")
	}

	// Verify NewClient is called
	if !strings.Contains(contentStr, "NewClient(dbDriver)") {
		t.Error("SetupClient should call NewClient with the driver adapter")
	}
}

func TestSetupClient_GeneratedForMySQL(t *testing.T) {
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

	// Generate all necessary files
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
	content, err := os.ReadFile(driverFile)
	if err != nil {
		t.Fatalf("Failed to read driver.go: %v", err)
	}

	contentStr := string(content)

	// Verify SetupClient function exists
	if !strings.Contains(contentStr, "func SetupClient(ctx context.Context, databaseURL ...string)") {
		t.Error("SetupClient function should be generated")
	}

	// Verify it returns (*Client, *sql.DB, error) for MySQL
	if !strings.Contains(contentStr, "func SetupClient(ctx context.Context, databaseURL ...string) (*Client, *sql.DB, error)") {
		t.Error("SetupClient should return (*Client, *sql.DB, error) for MySQL")
	}

	// Verify sql.Open is called
	if !strings.Contains(contentStr, "sql.Open(\"mysql\", url)") {
		t.Error("SetupClient should call sql.Open for MySQL")
	}

	// Verify NewSQLDriver is called
	if !strings.Contains(contentStr, "NewSQLDriver(db)") {
		t.Error("SetupClient should call NewSQLDriver to create adapter")
	}
}

func TestSetupClient_GeneratedForSQLite(t *testing.T) {
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

	// Generate all necessary files
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
	content, err := os.ReadFile(driverFile)
	if err != nil {
		t.Fatalf("Failed to read driver.go: %v", err)
	}

	contentStr := string(content)

	// Verify SetupClient function exists
	if !strings.Contains(contentStr, "func SetupClient(ctx context.Context, databaseURL ...string)") {
		t.Error("SetupClient function should be generated")
	}

	// Verify it returns (*Client, *sql.DB, error) for SQLite
	if !strings.Contains(contentStr, "func SetupClient(ctx context.Context, databaseURL ...string) (*Client, *sql.DB, error)") {
		t.Error("SetupClient should return (*Client, *sql.DB, error) for SQLite")
	}

	// Verify sql.Open is called
	if !strings.Contains(contentStr, "sql.Open(\"sqlite3\", url)") {
		t.Error("SetupClient should call sql.Open for SQLite")
	}

	// Verify NewSQLDriver is called
	if !strings.Contains(contentStr, "NewSQLDriver(db)") {
		t.Error("SetupClient should call NewSQLDriver to create adapter")
	}
}

func TestSetupClient_GetDatabaseURLFromConfig(t *testing.T) {
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

	// Generate driver
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}

	// Read generated driver.go
	driverFile := filepath.Join(outputDir, "driver.go")
	content, err := os.ReadFile(driverFile)
	if err != nil {
		t.Fatalf("Failed to read driver.go: %v", err)
	}

	contentStr := string(content)

	// Verify getDatabaseURLFromConfig function exists
	if !strings.Contains(contentStr, "func getDatabaseURLFromConfig() (string, error)") {
		t.Error("getDatabaseURLFromConfig function should be generated")
	}

	// Verify it searches for prisma.conf
	if !strings.Contains(contentStr, "prisma.conf") {
		t.Error("getDatabaseURLFromConfig should search for prisma.conf")
	}

	// Verify it handles env() format
	if !strings.Contains(contentStr, "env(\"") || !strings.Contains(contentStr, "env('") {
		t.Error("getDatabaseURLFromConfig should handle env() format")
	}

	// Verify it handles ${VAR} format
	if !strings.Contains(contentStr, "${") {
		t.Error("getDatabaseURLFromConfig should handle ${VAR} format")
	}
}

func TestSetupClient_NewClientUsesRawNew(t *testing.T) {
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

	// Generate all necessary files
	if err := GenerateClient(schema, outputDir); err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}

	// Read generated client.go
	clientFile := filepath.Join(outputDir, "client.go")
	content, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	contentStr := string(content)

	// Verify NewClient calls raw.New
	if !strings.Contains(contentStr, "raw.New(db)") {
		t.Error("NewClient should call raw.New(db) to create raw executor")
	}

	// Verify raw.New receives builder.DBTX
	// NOTE: This test only does static code analysis - it checks if the string "raw.New(db)"
	// exists in the generated code. It does NOT actually execute the generated code or test
	// with a real builder.DBTX implementation. This is why it didn't catch the panic issue
	// where raw.New couldn't recognize builder.DBTX implementations.
	//
	// For actual integration testing that would catch runtime panics, see:
	// - TestRawNew_WithMockBuilderDBTX in raw_test.go
	// - TestRawNew_ExecutorMethods in raw_test.go
}

func TestSetupClient_ErrorHandling(t *testing.T) {
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

	// Generate driver
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}

	// Read generated driver.go
	driverFile := filepath.Join(outputDir, "driver.go")
	content, err := os.ReadFile(driverFile)
	if err != nil {
		t.Fatalf("Failed to read driver.go: %v", err)
	}

	contentStr := string(content)

	// Verify error handling for getDatabaseURLFromConfig failure
	if !strings.Contains(contentStr, "error reading prisma.conf") {
		t.Error("SetupClient should handle error from getDatabaseURLFromConfig")
	}

	// Verify error handling for empty URL
	if !strings.Contains(contentStr, "DATABASE_URL is not set") {
		t.Error("SetupClient should return error when DATABASE_URL is empty")
	}

	// Verify error handling for NewPgxPoolFromURL failure
	if !strings.Contains(contentStr, "NewPgxPoolFromURL(ctx, url)") {
		t.Error("SetupClient should handle error from NewPgxPoolFromURL")
	}

	// Verify error handling for ping failure
	if !strings.Contains(contentStr, "pool.Ping(ctx)") {
		t.Error("SetupClient should ping the database and handle errors")
	}
	if !strings.Contains(contentStr, "error pinging database") {
		t.Error("SetupClient should return error when ping fails")
	}
}
