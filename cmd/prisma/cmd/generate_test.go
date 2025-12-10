package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate_CreatesOutputFiles(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Check that output directory exists
	// When schema is in prisma/schema.prisma and output is ./db, files are created in prisma/db
	outputDir := "prisma/db"
	if !fileExists(outputDir) {
		t.Error("Output directory was not created")
	}

	// Check for generated files
	expectedFiles := []string{
		"prisma/db/client.go",
		"prisma/db/models",
		"prisma/db/queries",
		"prisma/db/inputs",
	}

	for _, file := range expectedFiles {
		if !fileExists(file) {
			t.Errorf("Expected file/dir %s was not created", file)
		}
	}
}

func TestGenerate_CreatesModelFiles(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Check for model file
	// When schema is in prisma/schema.prisma and output is ./db, files are created in prisma/db
	modelFile := "prisma/db/models/users.go"
	if !fileExists(modelFile) {
		t.Error("Model file was not created")
	}

	content := readFile(t, modelFile)
	if !contains(content, "type Users struct") {
		t.Error("Model file should contain Users struct")
	}
}

func TestGenerate_CreatesQueryFiles(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Check for query file
	// When schema is in prisma/schema.prisma and output is ./db, files are created in prisma/db
	queryFile := "prisma/db/queries/users_query.go"
	if !fileExists(queryFile) {
		t.Error("Query file was not created")
	}

	content := readFile(t, queryFile)
	if !contains(content, "type UsersQuery struct") {
		t.Error("Query file should contain UsersQuery struct")
	}
}

func TestGenerate_CreatesClientFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Check for client file
	// When schema is in prisma/schema.prisma and output is ./db, files are created in prisma/db
	clientFile := "prisma/db/client.go"
	if !fileExists(clientFile) {
		t.Error("Client file was not created")
	}

	content := readFile(t, clientFile)
	if !contains(content, "type Client struct") {
		t.Error("Client file should contain Client struct")
	}
	if !contains(content, "func NewClient") {
		t.Error("Client file should contain NewClient function")
	}
}

func TestGenerate_CreatesInputFiles(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Check for input file
	// When schema is in prisma/schema.prisma and output is ./db, files are created in prisma/db
	inputFile := "prisma/db/inputs/users_input.go"
	if !fileExists(inputFile) {
		t.Error("Input file was not created")
	}

	content := readFile(t, inputFile)
	if !contains(content, "UsersCreateInput") {
		t.Error("Input file should contain CreateInput type")
	}
}

func TestGenerate_FailsWithInvalidSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createInvalidSchema(t)

	err := runGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail with invalid schema")
	}
}

func TestGenerate_WithCustomSchemaPath(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")

	// Create schema in custom location
	customSchemaPath := "custom/schema.prisma"
	err := os.MkdirAll(filepath.Dir(customSchemaPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create custom dir: %v", err)
	}
	createTestSchema(t, "")
	err = os.Rename("prisma/schema.prisma", customSchemaPath)
	if err != nil {
		t.Fatalf("Failed to move schema: %v", err)
	}

	schemaPath = customSchemaPath
	err = runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed with custom schema path: %v", err)
	}
}

func TestGenerate_WithCustomOutputDir(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	// Create schema with custom output
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
	createTestConfig(t, "")
	createTestSchema(t, schemaContent)

	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Check that custom output directory was used
	// When schema is in prisma/schema.prisma and output is ./custom_output, files are created in prisma/custom_output
	expectedOutput := "prisma/custom_output"
	if !fileExists(expectedOutput) {
		t.Error("Custom output directory was not created")
	}
}

func TestGenerate_NoUnusedImports(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Check client.go doesn't have unused context import
	// When schema is in prisma/schema.prisma and output is ./db, files are created in prisma/db
	clientFile := "prisma/db/client.go"
	content := readFile(t, clientFile)

	// Should not import context if not used
	if contains(content, "import (") {
		// Check that context is not imported if not used
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
		// Context should not be imported if not used in the code
		// Check for context.Context (type) or context. (package usage)
		if hasContext && !contains(content, "context.Context") && !contains(content, "context.") {
			t.Error("Client file should not import context if not used")
		}
	}
}

func TestGenerate_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file
	createTestSchema(t, "")

	err := runGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail without config file")
	}
}

// Test new features

func TestGenerate_WithNoHintsFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "")

	noHintsFlag = true
	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// With no-hints, we should still generate files but with less output
	// When schema is in prisma/schema.prisma and output is ./db, files are created in prisma/db
	if !fileExists("prisma/db/client.go") {
		t.Error("Client file should still be created with --no-hints")
	}
}

func TestGenerate_WithRequireModelsFlag_NoModels(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")

	// Create schema without models
	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}
`
	createTestSchema(t, schemaContent)

	requireModelsFlag = true
	err := runGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail with --require-models when no models exist")
	}
	if !strings.Contains(err.Error(), "no models") {
		t.Errorf("Error should mention 'no models', got: %v", err)
	}
}

func TestGenerate_WithRequireModelsFlag_WithModels(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "") // This has a users model

	requireModelsFlag = true
	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate should succeed with --require-models when models exist: %v", err)
	}
}

func TestGenerate_ProgressIndicators(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runGenerate([]string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Progress indicators should show during generation
	// We can't easily test the output, but we can verify files were created
	// When schema is in prisma/schema.prisma and output is ./db, files are created in prisma/db
	if !fileExists("prisma/db/client.go") {
		t.Error("Client file should be created")
	}
}

func TestGenerate_ErrorHandling_InvalidSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	createInvalidSchema(t)

	err := runGenerate([]string{})
	if err == nil {
		t.Error("runGenerate should fail with invalid schema")
	}

	// Error message should be helpful
	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "error") {
		t.Errorf("Error message should be helpful, got: %v", err)
	}
}

func TestGenerate_ErrorHandling_MissingSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	createTestConfig(t, "")
	// Don't create schema

	schemaPath = "nonexistent.prisma"
	err := runGenerate([]string{})
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
			result := parseGeneratorFlags(tc.args)
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
