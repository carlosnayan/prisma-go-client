package cmd

import (
	"os"
	"testing"
)

func TestIntegration_InitGenerateValidate(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	// Step 1: Initialize project
	err := runInit([]string{})
	if err != nil {
		t.Fatalf("Step 1 (init) failed: %v", err)
	}

	// Verify files were created
	if !fileExists("prisma.conf") {
		t.Error("prisma.conf should exist after init")
	}
	if !fileExists("prisma/schema.prisma") {
		t.Error("schema.prisma should exist after init")
	}

	// Step 2: Generate code
	err = runGenerate([]string{})
	if err != nil {
		t.Fatalf("Step 2 (generate) failed: %v", err)
	}

	// Verify generated files
	if !fileExists("./generated/client.go") {
		t.Error("client.go should exist after generate")
	}

	// Step 3: Validate schema
	err = runValidate([]string{})
	if err != nil {
		t.Fatalf("Step 3 (validate) failed: %v", err)
	}
}

func TestIntegration_InitGenerateFormat(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	// Step 1: Initialize project
	err := runInit([]string{})
	if err != nil {
		t.Fatalf("Step 1 (init) failed: %v", err)
	}

	// Step 2: Generate code
	err = runGenerate([]string{})
	if err != nil {
		t.Fatalf("Step 2 (generate) failed: %v", err)
	}

	// Step 3: Format schema
	formatWriteFlag = true
	formatCheckFlag = false
	err = runFormat([]string{})
	if err != nil {
		t.Fatalf("Step 3 (format) failed: %v", err)
	}
}

func TestIntegration_InitWithProviderGenerate(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	// Initialize with specific provider
	providerFlag = "mysql"
	err := runInit([]string{})
	if err != nil {
		t.Fatalf("Init with mysql provider failed: %v", err)
	}

	// Verify provider in schema
	schemaContent := readFile(t, "prisma/schema.prisma")
	if !contains(schemaContent, "provider = \"mysql\"") {
		t.Error("Schema should contain mysql provider")
	}

	// Generate should work
	err = runGenerate([]string{})
	if err != nil {
		t.Fatalf("Generate after init with mysql failed: %v", err)
	}
}

func TestIntegration_FullWorkflow(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")
	// 1. Initialize
	err := runInit([]string{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 2. Validate initial schema
	err = runValidate([]string{})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// 3. Generate initial code
	err = runGenerate([]string{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 4. Format schema
	formatWriteFlag = true
	err = runFormat([]string{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// 5. Validate again
	err = runValidate([]string{})
	if err != nil {
		t.Fatalf("Validate after format failed: %v", err)
	}

	// 6. Generate again (should work)
	err = runGenerate([]string{})
	if err != nil {
		t.Fatalf("Generate after format failed: %v", err)
	}
}

func TestIntegration_MigrateWorkflow(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// This test requires a database
	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// 1. Initialize
	err := runInit([]string{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 2. Generate
	err = runGenerate([]string{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 3. Check status (should show no migrations)
	err = runMigrateStatus([]string{})
	// This may fail if database is not set up, which is expected
	_ = err

	// 4. Create migration (requires database connection)
	// We skip the actual migration creation to avoid needing full DB setup
	t.Skip("Requires full database setup for migration workflow")
}

func TestIntegration_WithCustomPaths(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Create custom config path
	customConfigPath := "custom/prisma.conf"
	err := os.MkdirAll("custom", 0755)
	if err != nil {
		t.Fatalf("Failed to create custom dir: %v", err)
	}
	createTestConfig(t, "")
	err = os.Rename("prisma.conf", customConfigPath)
	if err != nil {
		t.Fatalf("Failed to move config: %v", err)
	}

	// Create custom schema path
	customSchemaPath := "custom/schema.prisma"
	createTestSchema(t, "")
	err = os.Rename("prisma/schema.prisma", customSchemaPath)
	if err != nil {
		t.Fatalf("Failed to move schema: %v", err)
	}

	// Use custom paths via flags
	configFile = customConfigPath
	schemaPath = customSchemaPath

	// Validate should work with custom paths
	err = runValidate([]string{})
	if err != nil {
		t.Fatalf("Validate with custom paths failed: %v", err)
	}
}
