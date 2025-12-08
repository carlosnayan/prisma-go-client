package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func TestFormat_FormatsSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create unformatted schema
	unformattedSchema := `datasource db{provider="postgresql"}
generator client{provider="prisma-client-go" output="../db"}
model users{id String @id email String}
`
	err := os.MkdirAll("prisma", 0755)
	if err != nil {
		t.Fatalf("Failed to create prisma dir: %v", err)
	}
	err = os.WriteFile("prisma/schema.prisma", []byte(unformattedSchema), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	formatCheckFlag = false
	err = runFormat([]string{})
	if err != nil {
		t.Fatalf("runFormat failed: %v", err)
	}

	// Read formatted schema
	formatted := readFile(t, "prisma/schema.prisma")

	// Should have proper formatting
	if !contains(formatted, "datasource db {") {
		t.Error("Schema should be formatted with proper spacing")
	}
}

func TestFormat_CheckMode(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create unformatted schema
	unformattedSchema := `datasource db{provider="postgresql"}
`
	err := os.MkdirAll("prisma", 0755)
	if err != nil {
		t.Fatalf("Failed to create prisma dir: %v", err)
	}
	err = os.WriteFile("prisma/schema.prisma", []byte(unformattedSchema), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	formatCheckFlag = true
	err = runFormat([]string{})
	if err == nil {
		t.Error("runFormat should fail in check mode when schema needs formatting")
	}
}

func TestFormat_AlreadyFormatted(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	// Format it first to ensure it's properly formatted
	formatCheckFlag = false
	err := runFormat([]string{})
	if err != nil {
		t.Fatalf("Initial format failed: %v", err)
	}

	// Now check if it's already formatted
	formatCheckFlag = true
	err = runFormat([]string{})
	// It should not fail if already formatted
	if err != nil {
		t.Fatalf("runFormat should not fail when schema is already formatted: %v", err)
	}
}

func TestFormat_WriteMode(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create unformatted schema
	unformattedSchema := `datasource db{provider="postgresql"}
`
	err := os.MkdirAll("prisma", 0755)
	if err != nil {
		t.Fatalf("Failed to create prisma dir: %v", err)
	}
	err = os.WriteFile("prisma/schema.prisma", []byte(unformattedSchema), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	formatCheckFlag = false
	err = runFormat([]string{})
	if err != nil {
		t.Fatalf("runFormat failed: %v", err)
	}

	// Verify file was modified
	formatted := readFile(t, "prisma/schema.prisma")
	if !contains(formatted, "datasource db {") {
		t.Error("Schema should be formatted and written to file")
	}
}

func TestFormat_FailsWithInvalidSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createInvalidSchema(t)

	err := runFormat([]string{})
	if err == nil {
		t.Error("runFormat should fail with invalid schema")
		return
	}
	if !strings.Contains(err.Error(), "syntax errors") {
		t.Errorf("Error should mention syntax errors, got: %v", err)
	}
}

func TestFormat_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file
	createTestSchema(t, "")

	err := runFormat([]string{})
	if err == nil {
		t.Error("runFormat should fail without config file")
	}
}

// Test new features

func TestFormat_WithSchemaFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema in custom location
	customSchemaPath := "custom/schema.prisma"
	err := os.MkdirAll(filepath.Dir(customSchemaPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create custom dir: %v", err)
	}

	// Create unformatted schema
	unformattedSchema := `datasource db{provider="postgresql"}
generator client{provider="prisma-client-go" output="../db"}
model users{id String @id email String}
`
	err = os.WriteFile(customSchemaPath, []byte(unformattedSchema), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	schemaPath = customSchemaPath
	formatCheckFlag = false
	err = runFormat([]string{})
	if err != nil {
		t.Fatalf("runFormat failed with --schema flag: %v", err)
	}

	// Verify file was formatted
	formatted := readFile(t, customSchemaPath)
	if !contains(formatted, "datasource db {") {
		t.Error("Schema should be formatted with --schema flag")
	}
}

func TestFormat_ValidatesFormattedSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	formatCheckFlag = false
	err := runFormat([]string{})
	if err != nil {
		t.Fatalf("runFormat should succeed: %v", err)
	}

	// Verify formatted schema is still valid
	formatted := readFile(t, "prisma/schema.prisma")
	_, errors, err := parser.Parse(formatted)
	if err != nil || len(errors) > 0 {
		t.Errorf("Formatted schema should be valid, got errors: %v, err: %v", errors, err)
	}
}

func TestFormat_CheckModeWithFormattedSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	// Format first
	formatCheckFlag = false
	err := runFormat([]string{})
	if err != nil {
		t.Fatalf("Initial format failed: %v", err)
	}

	// Now check
	formatCheckFlag = true
	err = runFormat([]string{})
	if err != nil {
		t.Errorf("runFormat should not fail when schema is already formatted: %v", err)
	}
}

func TestFormat_OutputMessage(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create unformatted schema
	unformattedSchema := `datasource db{provider="postgresql"}
`
	err := os.MkdirAll("prisma", 0755)
	if err != nil {
		t.Fatalf("Failed to create prisma dir: %v", err)
	}
	err = os.WriteFile("prisma/schema.prisma", []byte(unformattedSchema), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	formatCheckFlag = false
	err = runFormat([]string{})
	if err != nil {
		t.Fatalf("runFormat failed: %v", err)
	}

	// Output should indicate success
	// We can't easily capture output, but we can verify it doesn't error
}
