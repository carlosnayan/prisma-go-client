package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// TestQuickstartSchema validates the exact schema from QUICKSTART.md
func TestQuickstartSchema(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Exact schema from QUICKSTART.md
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
							{
								Name: "default",
								Arguments: []*parser.AttributeArgument{
									{Value: map[string]interface{}{"function": "autoincrement", "args": []interface{}{}}},
								},
							},
						},
					},
					{
						Name: "email",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "unique"},
						},
					},
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String", IsOptional: true},
					},
					{
						Name: "createdAt",
						Type: &parser.FieldType{Name: "DateTime"},
						Attributes: []*parser.Attribute{
							{
								Name: "default",
								Arguments: []*parser.AttributeArgument{
									{Value: map[string]interface{}{"function": "now", "args": []interface{}{}}},
								},
							},
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: "created_at"},
								},
							},
						},
					},
					{
						Name: "updatedAt",
						Type: &parser.FieldType{Name: "DateTime"},
						Attributes: []*parser.Attribute{
							{
								Name: "default",
								Arguments: []*parser.AttributeArgument{
									{Value: map[string]interface{}{"function": "now", "args": []interface{}{}}},
								},
							},
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: "updated_at"},
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

	// Verify table name from @@map
	if !strings.Contains(contentStr, `builder.NewQuery(client.db, "users"`) {
		t.Error("Table name should be 'users' from @@map(\"users\")")
	}

	// Verify column names
	// id - no @map, should use exact name "id"
	if !strings.Contains(contentStr, `"id"`) {
		t.Error("Column 'id' should be present")
	}

	// email - no @map, should use exact name "email"
	if !strings.Contains(contentStr, `"email"`) {
		t.Error("Column 'email' should be present")
	}

	// name - no @map, should use exact name "name"
	if !strings.Contains(contentStr, `"name"`) {
		t.Error("Column 'name' should be present")
	}

	// createdAt - has @map("created_at"), should use "created_at"
	if !strings.Contains(contentStr, `"created_at"`) {
		t.Error("Column 'createdAt' should be mapped to 'created_at' via @map")
	}

	// updatedAt - has @map("updated_at"), should use "updated_at"
	if !strings.Contains(contentStr, `"updated_at"`) {
		t.Error("Column 'updatedAt' should be mapped to 'updated_at' via @map")
	}

	// Verify that original field names are NOT used as column names when @map is present
	if strings.Contains(contentStr, `"createdAt"`) {
		t.Error("Column should NOT be 'createdAt' (original field name), should be 'created_at' from @map")
	}

	if strings.Contains(contentStr, `"updatedAt"`) {
		t.Error("Column should NOT be 'updatedAt' (original field name), should be 'updated_at' from @map")
	}

	// Generate queries to verify WhereInput converter
	err = GenerateQueries(schema, outputDir)
	if err != nil {
		t.Fatalf("GenerateQueries failed: %v", err)
	}

	// Read generated query file
	queryFile := filepath.Join(outputDir, "queries", "user_query.go")
	queryContent, err := os.ReadFile(queryFile)
	if err != nil {
		t.Fatalf("Failed to read query file: %v", err)
	}

	queryContentStr := string(queryContent)

	// Verify WhereInput converter uses mapped column names
	if !strings.Contains(queryContentStr, `result["created_at"]`) {
		t.Error("WhereInput converter should use 'created_at' (from @map) for createdAt field")
	}

	if !strings.Contains(queryContentStr, `result["updated_at"]`) {
		t.Error("WhereInput converter should use 'updated_at' (from @map) for updatedAt field")
	}

	// Verify that exact field names are used for fields without @map
	if !strings.Contains(queryContentStr, `result["email"]`) {
		t.Error("WhereInput converter should use 'email' (exact field name) for email field")
	}

	if !strings.Contains(queryContentStr, `result["name"]`) {
		t.Error("WhereInput converter should use 'name' (exact field name) for name field")
	}
}
