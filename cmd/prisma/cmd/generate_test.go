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
	outputDir := "./db"
	if !fileExists(outputDir) {
		t.Error("Output directory was not created")
	}

	// Check for generated files
	expectedFiles := []string{
		"./db/client.go",
		"./db/models",
		"./db/queries",
		"./db/inputs",
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
	modelFile := "./db/models/users.go"
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
	queryFile := "./db/queries/users_query.go"
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
	clientFile := "./db/client.go"
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
	inputFile := "./db/inputs/users_input.go"
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
	if !fileExists(customOutput) {
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
	clientFile := "./db/client.go"
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
