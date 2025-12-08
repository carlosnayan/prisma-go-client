package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// TestTableMap_WithAtAtMap tests that @@map is correctly used for table names
func TestTableMap_WithAtAtMap(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "User",
				Attributes: []*parser.Attribute{
					{
						Name: "map",
						Arguments: []*parser.AttributeArgument{
							{Value: "users"},
						},
					},
				},
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "email",
						Type: &parser.FieldType{Name: "String"},
					},
				},
			},
		},
	}

	// Generate client
	err := GenerateClient(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}

	// Read generated client.go
	clientFile := filepath.Join(outputDir, "client.go")
	content, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	contentStr := string(content)

	// Verify that table name "users" is used (from @@map), not "user" (snake_case of "User")
	if !strings.Contains(contentStr, `builder.NewQuery(client.db, "users"`) {
		t.Error("Table name should be 'users' from @@map, but found different table name")
	}

	// Verify that "User" (exact name from schema) is NOT used when @@map is present
	if strings.Contains(contentStr, `builder.NewQuery(client.db, "User"`) {
		t.Error("Table name should NOT be 'User' (exact name from schema), should be 'users' from @@map")
	}
}

// TestTableMap_WithoutAtAtMap tests that without @@map, snake_case is used
func TestTableMap_WithoutAtAtMap(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
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
					{
						Name: "email",
						Type: &parser.FieldType{Name: "String"},
					},
				},
			},
		},
	}

	// Generate client
	err := GenerateClient(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}

	// Read generated client.go
	clientFile := filepath.Join(outputDir, "client.go")
	content, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	contentStr := string(content)

	// Verify that table name "User" (exact name from schema) is used when no @@map
	if !strings.Contains(contentStr, `builder.NewQuery(client.db, "User"`) {
		t.Error("Table name should be 'User' (exact name from schema) when @@map is not present")
	}
}

// TestColumnMap_WithAtMap tests that @map is correctly used for column names
func TestColumnMap_WithAtMap(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
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
					{
						Name: "emailAddress",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: "email_address"},
								},
							},
						},
					},
					{
						Name: "fullName",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: "full_name"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Generate client
	err := GenerateClient(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}

	// Read generated client.go
	clientFile := filepath.Join(outputDir, "client.go")
	content, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	contentStr := string(content)

	// Verify that mapped column names are used
	if !strings.Contains(contentStr, `"email_address"`) {
		t.Error("Column name should be 'email_address' from @map, but not found")
	}

	if !strings.Contains(contentStr, `"full_name"`) {
		t.Error("Column name should be 'full_name' from @map, but not found")
	}

	// Verify that original field names are NOT used as column names
	if strings.Contains(contentStr, `"emailAddress"`) {
		t.Error("Column name should NOT be 'emailAddress' (original field name), should be 'email_address' from @map")
	}

	if strings.Contains(contentStr, `"fullName"`) {
		t.Error("Column name should NOT be 'fullName' (original field name), should be 'full_name' from @map")
	}
}

// TestColumnMap_InWhereInputConverter tests that @map is used in WhereInput converter
func TestColumnMap_InWhereInputConverter(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
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
					{
						Name: "emailAddress",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: "email_address"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Generate queries
	err := GenerateQueries(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateQueries failed: %v", err)
	}

	// Read generated query file
	queryFile := filepath.Join(outputDir, "queries", "user_query.go")
	content, err := os.ReadFile(queryFile)
	if err != nil {
		t.Fatalf("Failed to read query file: %v", err)
	}

	contentStr := string(content)

	// Verify that ConvertWhereInputToWhere uses mapped column name
	// The converter should use "email_address" not "emailAddress"
	if !strings.Contains(contentStr, `result["email_address"]`) {
		t.Error("WhereInput converter should use 'email_address' (from @map), but not found")
	}

	// Verify that original field name is NOT used
	if strings.Contains(contentStr, `result["emailAddress"]`) {
		t.Error("WhereInput converter should NOT use 'emailAddress' (original field name), should use 'email_address' from @map")
	}
}

// TestColumnMap_InUpdateBuilder tests that @map is used in Update builder
func TestColumnMap_InUpdateBuilder(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
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
					{
						Name: "emailAddress",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: "email_address"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Generate queries
	err := GenerateQueries(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateQueries failed: %v", err)
	}

	// Read generated query file
	queryFile := filepath.Join(outputDir, "queries", "user_query.go")
	content, err := os.ReadFile(queryFile)
	if err != nil {
		t.Fatalf("Failed to read query file: %v", err)
	}

	contentStr := string(content)

	// Verify that Update builder uses mapped column name
	// The updateData map should use "email_address" not "emailAddress"
	if !strings.Contains(contentStr, `updateData["email_address"]`) {
		t.Error("Update builder should use 'email_address' (from @map) in updateData map, but not found")
	}

	// Verify that original field name is NOT used
	if strings.Contains(contentStr, `updateData["emailAddress"]`) {
		t.Error("Update builder should NOT use 'emailAddress' (original field name), should use 'email_address' from @map")
	}
}

// TestColumnMap_InSelectFields tests that @map is used in Select fields
func TestColumnMap_InSelectFields(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
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
					{
						Name: "emailAddress",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: "email_address"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Generate queries
	err := GenerateQueries(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateQueries failed: %v", err)
	}

	// Read generated query file
	queryFile := filepath.Join(outputDir, "queries", "user_query.go")
	content, err := os.ReadFile(queryFile)
	if err != nil {
		t.Fatalf("Failed to read query file: %v", err)
	}

	contentStr := string(content)

	// Verify that Select uses mapped column name
	// The selectedFields should use "email_address" not "emailAddress"
	if !strings.Contains(contentStr, `selectedFields = append(selectedFields, "email_address")`) {
		t.Error("Select should use 'email_address' (from @map) in selectedFields, but not found")
	}

	// Verify that original field name is NOT used
	if strings.Contains(contentStr, `selectedFields = append(selectedFields, "emailAddress")`) {
		t.Error("Select should NOT use 'emailAddress' (original field name), should use 'email_address' from @map")
	}
}

// TestTableAndColumnMap_Combined tests both @@map and @map together
func TestTableAndColumnMap_Combined(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "User",
				Attributes: []*parser.Attribute{
					{
						Name: "map",
						Arguments: []*parser.AttributeArgument{
							{Value: "users"},
						},
					},
				},
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "emailAddress",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: "email_address"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Generate client and queries
	err := GenerateClient(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}

	err = GenerateQueries(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateQueries failed: %v", err)
	}

	// Read generated client.go
	clientFile := filepath.Join(outputDir, "client.go")
	content, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	contentStr := string(content)

	// Verify table name from @@map
	if !strings.Contains(contentStr, `builder.NewQuery(client.db, "users"`) {
		t.Error("Table name should be 'users' from @@map")
	}

	// Verify column name from @map
	if !strings.Contains(contentStr, `"email_address"`) {
		t.Error("Column name should be 'email_address' from @map")
	}

	// Read generated query file
	queryFile := filepath.Join(outputDir, "queries", "user_query.go")
	queryContent, err := os.ReadFile(queryFile)
	if err != nil {
		t.Fatalf("Failed to read query file: %v", err)
	}

	queryContentStr := string(queryContent)

	// Verify WhereInput converter uses mapped column
	if !strings.Contains(queryContentStr, `result["email_address"]`) {
		t.Error("WhereInput converter should use 'email_address' from @map")
	}
}
