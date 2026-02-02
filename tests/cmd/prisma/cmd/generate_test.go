package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestGenerate_CreatesOutputFiles(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	outputDir := "prisma/db"
	if !cmd.FileExists(outputDir) {
		t.Error("Output directory was not created")
	}

	expectedFiles := []string{
		"prisma/db/client.go",
		"prisma/db/models",
		"prisma/db/queries",
		"prisma/db/inputs",
	}

	for _, file := range expectedFiles {
		if !cmd.FileExists(file) {
			t.Errorf("Expected file/dir %s was not created", file)
		}
	}
}

func TestGenerate_CreatesModelFiles(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	modelFile := "prisma/db/models/users.go"
	if !cmd.FileExists(modelFile) {
		t.Error("Model file was not created")
	}

	content := cmd.ReadFile(t, modelFile)
	if !cmd.Contains(content, "type Users struct") {
		t.Error("Model file should contain Users struct")
	}
}

func TestGenerate_CreatesQueryFiles(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	queryFile := "prisma/db/queries/users_query.go"
	if !cmd.FileExists(queryFile) {
		t.Error("Query file was not created")
	}

	content := cmd.ReadFile(t, queryFile)
	if !cmd.Contains(content, "type UsersQuery struct") {
		t.Error("Query file should contain UsersQuery struct")
	}
}

func TestGenerate_CreatesClientFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	clientFile := "prisma/db/client.go"
	if !cmd.FileExists(clientFile) {
		t.Error("Client file was not created")
	}

	content := cmd.ReadFile(t, clientFile)
	if !cmd.Contains(content, "type Client struct") {
		t.Error("Client file should contain Client struct")
	}
	if !cmd.Contains(content, "func NewClient") {
		t.Error("Client file should contain NewClient function")
	}
}

func TestGenerate_CreatesInputFiles(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	inputFile := "prisma/db/inputs/users_input.go"
	if !cmd.FileExists(inputFile) {
		t.Error("Input file was not created")
	}

	content := cmd.ReadFile(t, inputFile)
	if !cmd.Contains(content, "UsersCreateManyArgs") {
		t.Error("Input file should contain CreateManyArgs type")
	}
}

func TestGenerate_FailsWithInvalidSchema(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestConfig(t, "")
	cmd.CreateInvalidSchema(t)

	err := cmd.RunGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail with invalid schema")
	}
}

func TestGenerate_WithCustomSchemaPath(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")

	customSchemaPath := "custom/schema.prisma"
	err := os.MkdirAll(filepath.Dir(customSchemaPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create custom dir: %v", err)
	}
	cmd.CreateTestSchema(t, "")
	err = os.Rename("prisma/schema.prisma", customSchemaPath)
	if err != nil {
		t.Fatalf("Failed to move schema: %v", err)
	}

	cmd.SetSchemaPath(customSchemaPath)
	err = cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed with custom schema path: %v", err)
	}
}

func TestGenerate_WithCustomOutputDir(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")

	customOutput := "./custom_output"
	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "` + customOutput + `"
}

model users {
  id String @id
  email String
}
`
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, schemaContent)

	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	expectedOutput := "prisma/custom_output"
	if !cmd.FileExists(expectedOutput) {
		t.Error("Custom output directory was not created")
	}
}

func TestGenerate_NoUnusedImports(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	clientFile := "prisma/db/client.go"
	content := cmd.ReadFile(t, clientFile)

	if cmd.Contains(content, "import (") {
		lines := strings.Split(content, "\n")
		inImports := false
		hasContext := false
		for _, line := range lines {
			if strings.Contains(line, "import (") {
				inImports = true
			}
			if inImports && strings.Contains(line, "context") {
				hasContext = true
			}
			if inImports && strings.Contains(line, ")") {
				break
			}
		}
		if hasContext && !cmd.Contains(content, "context.Context") && !cmd.Contains(content, "context.") {
			t.Error("Client file should not import context if not used")
		}
	}
}

func TestGenerate_RequiresConfigFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})
	cmd.CreateTestSchema(t, "")

	err := cmd.RunGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail without config file")
	}
}

func TestGenerate_WithNoHintsFlag(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SetNoHintsFlag(true)
	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	if !cmd.FileExists("prisma/db/client.go") {
		t.Error("Client file should still be created with --no-hints")
	}
}

func TestGenerate_WithRequireModelsFlag_NoModels(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")

	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}
`
	cmd.CreateTestSchema(t, schemaContent)

	cmd.SetRequireModelsFlag(true)
	err := cmd.RunGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail with --require-models when no models exist")
	}
	if !strings.Contains(err.Error(), "no models") {
		t.Errorf("Error should mention 'no models', got: %v", err)
	}
}

func TestGenerate_WithRequireModelsFlag_WithModels(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SetRequireModelsFlag(true)
	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate should succeed with --require-models when models exist: %v", err)
	}
}

func TestGenerate_ProgressIndicators(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	err := cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	if !cmd.FileExists("prisma/db/client.go") {
		t.Error("Client file should be created")
	}
}

func TestGenerate_ErrorHandling_InvalidSchema(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")
	cmd.CreateInvalidSchema(t)

	err := cmd.RunGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail with invalid schema")
	}

	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "error") {
		t.Errorf("Error message should be helpful, got: %v", err)
	}
}

func TestGenerate_ErrorHandling_MissingSchema(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SetGeneratorFlags([]string{})

	cmd.CreateTestGoMod(t, "test-module")
	cmd.CreateTestConfig(t, "")

	cmd.SetSchemaPath("nonexistent.prisma")
	err := cmd.RunGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail with missing schema")
	}
}

func TestGenerate_ParseGeneratorFlags(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "single generator flag",
			args:     []string{"--generator", "client"},
			expected: []string{"client"},
		},
		{
			name:     "multiple generator flags",
			args:     []string{"--generator", "client", "--generator", "client2"},
			expected: []string{"client", "client2"},
		},
		{
			name:     "generator with equals",
			args:     []string{"--generator=client"},
			expected: []string{"client"},
		},
		{
			name:     "no generator flags",
			args:     []string{},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cmd.ParseGeneratorFlags(tc.args)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d generators, got %d", len(tc.expected), len(result))
			}
			for i, expected := range tc.expected {
				if i < len(result) && result[i] != expected {
					t.Errorf("Expected generator[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}
