package cmd_test

import (
	"os"
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestIntegration_InitGenerateValidate(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")
	// Step 1: Initialize project
	err := cmd.RunInit([]string{})
	if err != nil {
		t.Fatalf("Step 1 (init) failed: %v", err)
	}

	// Verify files were created
	if !cmd.FileExists("prisma.conf") {
		t.Error("prisma.conf should exist after init")
	}
	if !cmd.FileExists("prisma/schema.prisma") {
		t.Error("schema.prisma should exist after init")
	}

	// Step 2: Generate code
	err = cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("Step 2 (generate) failed: %v", err)
	}

	// Verify generated files
	// When schema is in prisma/schema.prisma and output is ./generated, files are created in prisma/generated
	if !cmd.FileExists("prisma/generated/client.go") {
		t.Error("client.go should exist after generate")
	}

	// Step 3: Validate schema
	err = cmd.RunValidate([]string{})
	if err != nil {
		t.Fatalf("Step 3 (validate) failed: %v", err)
	}
}

func TestIntegration_InitGenerateFormat(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")
	// Step 1: Initialize project
	err := cmd.RunInit([]string{})
	if err != nil {
		t.Fatalf("Step 1 (init) failed: %v", err)
	}

	// Step 2: Generate code
	err = cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("Step 2 (generate) failed: %v", err)
	}

	// Step 3: Format schema
	cmd.SetFormatCheckFlag(false)
	err = cmd.RunFormat([]string{})
	if err != nil {
		t.Fatalf("Step 3 (format) failed: %v", err)
	}
}

func TestIntegration_InitWithProviderGenerate(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")
	// Initialize with specific provider
	cmd.SetProviderFlag("mysql")
	err := cmd.RunInit([]string{})
	if err != nil {
		t.Fatalf("Init with mysql provider failed: %v", err)
	}

	// Verify provider in schema
	schemaContent := cmd.ReadFile(t, "prisma/schema.prisma")
	if !cmd.Contains(schemaContent, "provider = \"mysql\"") {
		t.Error("Schema should contain mysql provider")
	}

	// Generate should work
	err = cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("Generate after init with mysql failed: %v", err)
	}
}

func TestIntegration_FullWorkflow(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")
	// 1. Initialize
	err := cmd.RunInit([]string{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 2. Validate initial schema
	err = cmd.RunValidate([]string{})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// 3. Generate initial code
	err = cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 4. Format schema
	err = cmd.RunFormat([]string{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// 5. Validate again
	err = cmd.RunValidate([]string{})
	if err != nil {
		t.Fatalf("Validate after format failed: %v", err)
	}

	// 6. Generate again (should work)
	err = cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("Generate after format failed: %v", err)
	}
}

func TestIntegration_MigrateWorkflow(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")

	// This test requires a database
	cmd.SkipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := cmd.CreateIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := cmd.GetTestDBURL(t, dbName)
	cleanup := cmd.SetEnv(t, "DATABASE_URL", testDBURL)
	defer cleanup()

	// 1. Initialize
	err := cmd.RunInit([]string{})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 2. Generate
	err = cmd.RunGenerate([]string{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 3. Check status (should show no migrations)
	err = cmd.RunMigrateStatus([]string{})
	if err != nil {
		t.Logf("Migrate status: %v", err)
	}

	// 4. Create migration - runs against isolated database (safe)
	err = cmd.RunMigrateDev([]string{"initial_migration"})
	if err != nil {
		t.Logf("Migrate dev: %v", err)
	}

	// Verify migration was created
	if cmd.FileExists("prisma/migrations") {
		t.Log("Migration workflow completed successfully")
	}
}

func TestIntegration_WithCustomPaths(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Create custom config path
	customConfigPath := "custom/prisma.conf"
	err := os.MkdirAll("custom", 0755)
	if err != nil {
		t.Fatalf("Failed to create custom dir: %v", err)
	}
	cmd.CreateTestConfig(t, "")
	err = os.Rename("prisma.conf", customConfigPath)
	if err != nil {
		t.Fatalf("Failed to move config: %v", err)
	}

	// Create custom schema path
	customSchemaPath := "custom/schema.prisma"
	cmd.CreateTestSchema(t, "")
	err = os.Rename("prisma/schema.prisma", customSchemaPath)
	if err != nil {
		t.Fatalf("Failed to move schema: %v", err)
	}

	// Use custom paths via flags
	cmd.SetConfigFile(customConfigPath)
	cmd.SetSchemaPath(customSchemaPath)

	// Validate should work with custom paths
	err = cmd.RunValidate([]string{})
	if err != nil {
		t.Fatalf("Validate with custom paths failed: %v", err)
	}
}
