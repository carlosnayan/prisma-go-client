package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func TestWhereInput_IncludesEnumFields(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir := t.TempDir()
	inputsDir := filepath.Join(tmpDir, "inputs")
	if err := os.MkdirAll(inputsDir, 0755); err != nil {
		t.Fatalf("Failed to create inputs directory: %v", err)
	}

	// Create a schema with an enum and a model that uses it
	schema := &parser.Schema{
		Enums: []*parser.Enum{
			{
				Name: "Role",
				Values: []*parser.EnumValue{
					{Name: "INTERNAL"},
					{Name: "OWNER"},
					{Name: "ADMIN"},
					{Name: "AGENT"},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "users",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "email",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "role",
						Type: &parser.FieldType{Name: "Role"}, // Enum field
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"}, // Relation field
						Attributes: []*parser.Attribute{
							{
								Name: "relation",
								Arguments: []*parser.AttributeArgument{
									{Name: "fields", Value: []interface{}{"id_tenant"}},
									{Name: "references", Value: []interface{}{"id_tenant"}},
								},
							},
						},
					},
				},
			},
		},
	}

	// Generate inputs
	err := GenerateInputs(schema, tmpDir)
	if err != nil {
		t.Fatalf("GenerateInputs failed: %v", err)
	}

	// Read the generated input file
	inputFile := filepath.Join(inputsDir, "users_input.go")
	content, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("Failed to read generated input file: %v", err)
	}

	contentStr := string(content)

	// Verify that Role field is included in WhereInput
	if !strings.Contains(contentStr, "Role *StringFilter") {
		t.Error("WhereInput should include Role field with StringFilter type")
	}

	// Verify that the relation field is NOT included in WhereInput
	if strings.Contains(contentStr, "Tenant *") && strings.Contains(contentStr, "type UsersWhereInput") {
		// Check that Tenant is not in WhereInput (it should only appear in the model, not in WhereInput)
		// We check by ensuring Tenant doesn't appear between "type UsersWhereInput struct {" and the closing brace
		whereInputStart := strings.Index(contentStr, "type UsersWhereInput struct {")
		whereInputEnd := strings.Index(contentStr[whereInputStart:], "}\n\n")
		if whereInputEnd == -1 {
			whereInputEnd = len(contentStr) - whereInputStart
		}
		whereInputSection := contentStr[whereInputStart : whereInputStart+whereInputEnd]
		if strings.Contains(whereInputSection, "\tTenant") {
			t.Error("WhereInput should NOT include relation field Tenant")
		}
	}

	// Verify that email (String field) is included
	if !strings.Contains(contentStr, "Email *StringFilter") {
		t.Error("WhereInput should include Email field with StringFilter type")
	}
}

func TestWhereInput_ExcludesRelations(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir := t.TempDir()
	inputsDir := filepath.Join(tmpDir, "inputs")
	if err := os.MkdirAll(inputsDir, 0755); err != nil {
		t.Fatalf("Failed to create inputs directory: %v", err)
	}

	// Create a schema with a model that has relations
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "posts",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "title",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "author",
						Type: &parser.FieldType{Name: "users"}, // Relation field
						Attributes: []*parser.Attribute{
							{
								Name: "relation",
								Arguments: []*parser.AttributeArgument{
									{Name: "fields", Value: []interface{}{"author_id"}},
									{Name: "references", Value: []interface{}{"id"}},
								},
							},
						},
					},
				},
			},
		},
	}

	// Generate inputs
	err := GenerateInputs(schema, tmpDir)
	if err != nil {
		t.Fatalf("GenerateInputs failed: %v", err)
	}

	// Read the generated input file
	inputFile := filepath.Join(inputsDir, "posts_input.go")
	content, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("Failed to read generated input file: %v", err)
	}

	contentStr := string(content)

	// Verify that Author (relation) is NOT included in WhereInput
	whereInputStart := strings.Index(contentStr, "type PostsWhereInput struct {")
	if whereInputStart == -1 {
		t.Fatal("PostsWhereInput not found in generated file")
	}

	// Find the end of WhereInput struct
	whereInputEnd := strings.Index(contentStr[whereInputStart:], "}\n\n")
	if whereInputEnd == -1 {
		whereInputEnd = len(contentStr) - whereInputStart
	}
	whereInputSection := contentStr[whereInputStart : whereInputStart+whereInputEnd]

	if strings.Contains(whereInputSection, "\tAuthor") {
		t.Error("WhereInput should NOT include relation field Author")
	}

	// Verify that Title (non-relation field) is included
	if !strings.Contains(whereInputSection, "Title *StringFilter") {
		t.Error("WhereInput should include Title field with StringFilter type")
	}
}

func TestWhereInput_MultipleEnums(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir := t.TempDir()
	inputsDir := filepath.Join(tmpDir, "inputs")
	if err := os.MkdirAll(inputsDir, 0755); err != nil {
		t.Fatalf("Failed to create inputs directory: %v", err)
	}

	// Create a schema with multiple enums
	schema := &parser.Schema{
		Enums: []*parser.Enum{
			{
				Name: "Role",
				Values: []*parser.EnumValue{
					{Name: "ADMIN"},
					{Name: "USER"},
				},
			},
			{
				Name: "Status",
				Values: []*parser.EnumValue{
					{Name: "ACTIVE"},
					{Name: "INACTIVE"},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "users",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "role",
						Type: &parser.FieldType{Name: "Role"}, // First enum
					},
					{
						Name: "status",
						Type: &parser.FieldType{Name: "Status"}, // Second enum
					},
				},
			},
		},
	}

	// Generate inputs
	err := GenerateInputs(schema, tmpDir)
	if err != nil {
		t.Fatalf("GenerateInputs failed: %v", err)
	}

	// Read the generated input file
	inputFile := filepath.Join(inputsDir, "users_input.go")
	content, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("Failed to read generated input file: %v", err)
	}

	contentStr := string(content)

	// Verify that both enum fields are included in WhereInput
	if !strings.Contains(contentStr, "Role *StringFilter") {
		t.Error("WhereInput should include Role field with StringFilter type")
	}

	if !strings.Contains(contentStr, "Status *StringFilter") {
		t.Error("WhereInput should include Status field with StringFilter type")
	}
}
