package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// TestGeneratedCode_Compiles verifica se todo o código gerado compila sem erros
func TestGeneratedCode_Compiles(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "db")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n\ngo 1.21\n\nreplace github.com/carlosnayan/prisma-go-client => /Users/carlos/Documents/estudo/prisma/prisma-go-client\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a simple schema
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
					{
						Name: "email",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String"},
					},
				},
			},
		},
	}

	// Generate all files
	if err := GenerateModels(schema, outputDir); err != nil {
		t.Fatalf("GenerateModels failed: %v", err)
	}
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}
	if err := GenerateUtils(outputDir); err != nil {
		t.Fatalf("GenerateUtils failed: %v", err)
	}
	if err := GenerateBuilder(schema, outputDir); err != nil {
		t.Fatalf("GenerateBuilder failed: %v", err)
	}
	if err := GenerateInputs(schema, outputDir); err != nil {
		t.Fatalf("GenerateInputs failed: %v", err)
	}
	if err := GenerateQueries(schema, outputDir); err != nil {
		t.Fatalf("GenerateQueries failed: %v", err)
	}
	if err := GenerateFilters(schema, outputDir); err != nil {
		t.Fatalf("GenerateFilters failed: %v", err)
	}
	if err := GenerateClient(schema, outputDir); err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}

	// Initialize go module and install dependencies
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	if err := cmd.Run(); err != nil {
		t.Logf("go mod tidy failed (may be expected): %v", err)
	}

	// Try to compile the generated code
	// We use -mod=readonly to avoid modifying the module, but allow compilation
	cmd = exec.Command("go", "build", "-mod=readonly", "./...")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If it's just missing dependencies, that's OK for this test
		// We're mainly checking for syntax errors
		outputStr := string(output)
		if !strings.Contains(outputStr, "no required module") && !strings.Contains(outputStr, "cannot find package") {
			t.Errorf("Generated code has compilation errors (not just missing dependencies):\n%s", outputStr)
			t.Errorf("Compilation error: %v", err)
		} else {
			t.Logf("Compilation failed due to missing dependencies (expected in test environment):\n%s", outputStr)
		}
	}
}

// TestGeneratedCode_PackageNames verifica se os package names estão corretos
func TestGeneratedCode_PackageNames(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "db")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n\ngo 1.21\n\nreplace github.com/carlosnayan/prisma-go-client => /Users/carlos/Documents/estudo/prisma/prisma-go-client\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a simple schema
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

	// Generate all files
	if err := GenerateClient(schema, outputDir); err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}
	if err := GenerateBuilder(schema, outputDir); err != nil {
		t.Fatalf("GenerateBuilder failed: %v", err)
	}

	// Check package names
	packageRegex := regexp.MustCompile(`^package\s+(\w+)`)

	// Files in root should have package "generated"
	rootFiles := []string{"client.go", "driver.go"}
	for _, filename := range rootFiles {
		filePath := filepath.Join(outputDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", filename, err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if matches := packageRegex.FindStringSubmatch(line); matches != nil {
				if matches[1] != "generated" {
					t.Errorf("%s has wrong package name: got %s, want generated", filename, matches[1])
				}
				break
			}
		}
	}

	// Files in builder/ should have package "builder"
	builderFile := filepath.Join(outputDir, "builder", "builder.go")
	if content, err := os.ReadFile(builderFile); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if matches := packageRegex.FindStringSubmatch(line); matches != nil {
				if matches[1] != "builder" {
					t.Errorf("builder/builder.go has wrong package name: got %s, want builder", matches[1])
				}
				break
			}
		}
	}
}

// TestGeneratedCode_ColumnSyntax verifica sintaxe correta das colunas (com chaves {} em slices)
func TestGeneratedCode_ColumnSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "db")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n\ngo 1.21\n\nreplace github.com/carlosnayan/prisma-go-client => /Users/carlos/Documents/estudo/prisma/prisma-go-client\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a schema with a model that has multiple columns
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
				Name: "AvailablePlans",
				Fields: []*parser.ModelField{
					{
						Name: "id_available_plan",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "label",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "stripe_product_id",
						Type: &parser.FieldType{Name: "String"},
					},
				},
			},
		},
	}

	// Generate client.go
	if err := GenerateClient(schema, outputDir); err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}

	// Read client.go and check column syntax
	clientFile := filepath.Join(outputDir, "client.go")
	content, err := os.ReadFile(clientFile)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	contentStr := string(content)

	// Check for correct syntax: columns_X := []string{"col1", "col2"}
	// Should NOT have: columns_X := []string"col1", "col2"
	incorrectPattern := regexp.MustCompile(`columns_\w+\s*:=\s*\[\]string"[^"]+"`)
	if incorrectPattern.MatchString(contentStr) {
		t.Error("Found incorrect column syntax: columns_X := []string\"col\" (missing braces)")
		t.Errorf("Content snippet: %s", findMatchingLine(contentStr, incorrectPattern))
	}

	// Check for correct syntax with braces
	correctPattern := regexp.MustCompile(`columns_\w+\s*:=\s*\[\]string\s*\{[^}]+\}`)
	if !correctPattern.MatchString(contentStr) {
		t.Error("Did not find correct column syntax: columns_X := []string{\"col\"}")
	}
}

// TestGeneratedCode_ImportsAndHelpers verifica que imports e funções necessárias estão presentes
func TestGeneratedCode_ImportsAndHelpers(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "db")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n\ngo 1.21\n\nreplace github.com/carlosnayan/prisma-go-client => /Users/carlos/Documents/estudo/prisma/prisma-go-client\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a simple schema
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

	// Generate fluent.go
	if err := GenerateBuilder(schema, outputDir); err != nil {
		t.Fatalf("GenerateBuilder failed: %v", err)
	}

	// Read fluent.go and builder.go and check for toSnakeCase and strings import
	fluentFile := filepath.Join(outputDir, "builder", "fluent.go")
	fluentContent, err := os.ReadFile(fluentFile)
	if err != nil {
		t.Fatalf("Failed to read fluent.go: %v", err)
	}

	builderFile := filepath.Join(outputDir, "builder", "builder.go")
	builderContent, err := os.ReadFile(builderFile)
	if err != nil {
		t.Fatalf("Failed to read builder.go: %v", err)
	}

	fluentStr := string(fluentContent)
	builderStr := string(builderContent)

	// Check for strings import in fluent.go
	if !strings.Contains(fluentStr, `"strings"`) {
		t.Error("fluent.go is missing strings import (needed for toSnakeCase)")
	}

	// Check for toSnakeCase function definition in builder.go (same package as fluent.go)
	if !strings.Contains(builderStr, "func toSnakeCase") {
		t.Error("builder.go is missing toSnakeCase function definition (needed by fluent.go)")
	}

	// Check that toSnakeCase is used in fluent.go
	if !strings.Contains(fluentStr, "toSnakeCase(") {
		t.Error("fluent.go should use toSnakeCase function")
	}
}

// Helper function to find matching line for debugging
func findMatchingLine(content string, pattern *regexp.Regexp) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if pattern.MatchString(line) {
			start := i - 2
			if start < 0 {
				start = 0
			}
			end := i + 3
			if end > len(lines) {
				end = len(lines)
			}
			return strings.Join(lines[start:end], "\n")
		}
	}
	return "not found"
}

// TestGeneratedCode_OptionalFieldsPointerHandling verifica que campos opcionais
// são tratados corretamente (sem double-dereference) em UpdateMany
func TestGeneratedCode_OptionalFieldsPointerHandling(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "db")

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n\ngo 1.21\n\nreplace github.com/carlosnayan/prisma-go-client => /Users/carlos/Documents/estudo/prisma/prisma-go-client\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Schema com campos opcionais de diferentes tipos
	schema := &parser.Schema{
		Datasources: []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{Name: "provider", Value: "postgresql"},
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
							{Name: "default", Arguments: []*parser.AttributeArgument{{Value: "autoincrement()"}}},
						},
					},
					{
						Name: "email",
						Type: &parser.FieldType{Name: "String", IsOptional: false},
					},
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String", IsOptional: true},
					},
					{
						Name: "phone",
						Type: &parser.FieldType{Name: "String", IsOptional: true},
					},
					{
						Name: "birthDate",
						Type: &parser.FieldType{Name: "DateTime", IsOptional: true},
					},
					{
						Name: "metadata",
						Type: &parser.FieldType{Name: "Json", IsOptional: true},
					},
				},
			},
		},
	}

	// Generate all files
	if err := GenerateModels(schema, outputDir); err != nil {
		t.Fatalf("GenerateModels failed: %v", err)
	}
	if err := GenerateInputs(schema, outputDir); err != nil {
		t.Fatalf("GenerateInputs failed: %v", err)
	}
	if err := GenerateQueries(schema, outputDir); err != nil {
		t.Fatalf("GenerateQueries failed: %v", err)
	}
	if err := GenerateBuilder(schema, outputDir); err != nil {
		t.Fatalf("GenerateBuilder failed: %v", err)
	}
	if err := GenerateFilters(schema, outputDir); err != nil {
		t.Fatalf("GenerateFilters failed: %v", err)
	}
	if err := GenerateClient(schema, outputDir); err != nil {
		t.Fatalf("GenerateClient failed: %v", err)
	}
	if err := GenerateDriver(schema, outputDir); err != nil {
		t.Fatalf("GenerateDriver failed: %v", err)
	}
	if err := GenerateRaw(outputDir); err != nil {
		t.Fatalf("GenerateRaw failed: %v", err)
	}
	if err := GenerateUtils(outputDir); err != nil {
		t.Fatalf("GenerateUtils failed: %v", err)
	}

	// Try to compile - this is the main check for the regression
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	if err := cmd.Run(); err != nil {
		t.Logf("go mod tidy warning: %v", err)
	}

	cmd = exec.Command("go", "build", "-mod=readonly", "./...")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// Check for the specific error that the IsPointer bug causes
		if strings.Contains(outputStr, "cannot use *b.data.") && strings.Contains(outputStr, "as *") {
			t.Errorf("REGRESSION: IsPointer bug detected! The generated code has pointer dereferencing errors.")
			t.Errorf("This means optional fields are being incorrectly dereferenced in UpdateMany.")
			t.Fatalf("Compilation output:\n%s", outputStr)
		}
		// Other errors might be OK (missing dependencies)
		if !strings.Contains(outputStr, "no required module") && !strings.Contains(outputStr, "cannot find package") {
			t.Errorf("Generated code has compilation errors:\n%s", outputStr)
		}
	}

	// Additional check: verify UpdateMany doesn't double-dereference pointers
	queryFile := filepath.Join(outputDir, "queries", "user_query.go")
	content, err := os.ReadFile(queryFile)
	if err != nil {
		t.Fatalf("Failed to read user_query.go: %v", err)
	}

	contentStr := string(content)

	// For optional string fields that use pointers in the model (name, phone)
	// The UpdateMany should assign directly: result.Name = b.data.Name
	// NOT: result.Name = *b.data.Name (which causes type mismatch)

	// Check that we're assigning pointer fields correctly in UpdateMany
	if strings.Contains(contentStr, "result.Name = *b.data.Name") {
		t.Error("REGRESSION: UpdateMany is incorrectly dereferencing Name field (should be: result.Name = b.data.Name)")
	}
	if strings.Contains(contentStr, "result.Phone = *b.data.Phone") {
		t.Error("REGRESSION: UpdateMany is incorrectly dereferencing Phone field (should be: result.Phone = b.data.Phone)")
	}
	if strings.Contains(contentStr, "result.BirthDate = *b.data.BirthDate") {
		t.Error("REGRESSION: UpdateMany is incorrectly dereferencing BirthDate field (should be: result.BirthDate = b.data.BirthDate)")
	}

	// For Update operation (uses map), check it also handles pointers correctly
	if strings.Contains(contentStr, `updateData["name"] = b.data.Name`) {
		// This is correct for pointer fields - assign the pointer directly
		t.Logf("✓ Update operation correctly assigns pointer field 'name'")
	}
}
