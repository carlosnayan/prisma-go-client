package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// setupTestDir creates a temporary directory for testing and changes to it
func setupTestDir(t testingT) string {
	dir, err := os.MkdirTemp("", "prisma-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}

	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Store old dir for cleanup
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
		_ = os.RemoveAll(dir)
	})

	return dir
}

// cleanupTestDir removes the test directory
func cleanupTestDir(dir string) error {
	return os.RemoveAll(dir)
}

// createTestSchema creates a test schema.prisma file
func createTestSchema(t testingT, content string) string {
	if content == "" {
		content = `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

model users {
  id         String   @id @default(uuid())
  email      String   @unique
  name       String?
  created_at DateTime @default(now())
  updated_at DateTime @default(now())
}
`
	}

	schemaPath := "prisma/schema.prisma"
	err := os.MkdirAll(filepath.Dir(schemaPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create prisma dir: %v", err)
	}

	err = os.WriteFile(schemaPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	return schemaPath
}

// createTestConfig creates a test prisma.conf file
func createTestConfig(t testingT, content string) string {
	if content == "" {
		content = `schema = "prisma/schema.prisma"

[migrations]
path = "prisma/migrations"

[datasource]
url = "env('DATABASE_URL')"
`
	}

	configPath := "prisma.conf"
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	return configPath
}

// createTestMigrationsDir creates the migrations directory
func createTestMigrationsDir(t testingT) string {
	migrationsPath := "prisma/migrations"
	err := os.MkdirAll(migrationsPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create migrations dir: %v", err)
	}
	return migrationsPath
}

// createTestMigration creates a test migration file
func createTestMigration(t testingT, name, sql string) string {
	migrationsPath := createTestMigrationsDir(t)
	migrationPath := filepath.Join(migrationsPath, name)

	err := os.MkdirAll(migrationPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create migration dir: %v", err)
	}

	sqlPath := filepath.Join(migrationPath, "migration.sql")
	err = os.WriteFile(sqlPath, []byte(sql), 0644)
	if err != nil {
		t.Fatalf("Failed to write migration file: %v", err)
	}

	return migrationPath
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readFile reads a file and returns its content
func readFile(t testingT, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// testingT is an interface that matches both *testing.T and *testing.B
type testingT interface {
	Fatalf(format string, args ...interface{})
	Cleanup(func())
}

// resetGlobalFlags resets global flag variables to their default values
func resetGlobalFlags() {
	configFile = ""
	schemaPath = ""
	verbose = false
	providerFlag = ""
	databaseFlag = ""
	formatCheckFlag = false
	migrateResolveAppliedFlag = ""
	migrateResolveRolledBackFlag = ""
	dbPushAcceptDataLossFlag = false
	dbPushSkipGenerateFlag = false
	dbExecuteFileFlag = ""
	dbExecuteStdinFlag = ""
	diffFrom = ""
	diffTo = ""
	diffOut = ""
	watchFlag = false
	generatorFlags = nil
	noHintsFlag = false
	requireModelsFlag = false
}

// setEnv sets an environment variable and returns a cleanup function
func setEnv(t testingT, key, value string) func() {
	oldValue := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		if oldValue == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, oldValue)
		}
	}
}

// createInvalidSchema creates a schema with syntax errors
func createInvalidSchema(t testingT) string {
	content := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "../db"
}

model users {
  id String @id
  email String
  invalidField = invalid syntax here
  // Missing closing brace - this will cause a syntax error
`
	return createTestSchema(t, content)
}

// contains checks if a string contains a substring (case-sensitive)
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// assertDirExists checks if a directory exists
func assertDirExists(t testingT, path string) {
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Directory %s does not exist: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("Path %s exists but is not a directory", path)
	}
}

// getTestDatabaseURL returns a test database URL or skips the test if not available
func getTestDatabaseURL(t testingT) string {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		// Use a default test database URL (can be overridden)
		url = "postgresql://postgres:postgres@localhost:5432/prisma_test?sslmode=disable"
	}
	return url
}

// skipIfNoDatabase skips the test if no database is available
// Note: This requires testing.T which has Skip method, so it should be called directly from test functions
// Deprecated: Use testing.SkipIfNoDatabase from internal/testing package instead
func skipIfNoDatabase(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database test")
	}
}

// createTestGoMod creates a go.mod file in the test directory
// This is needed for generate tests that require module detection
func createTestGoMod(t testingT, moduleName string) string {
	if moduleName == "" {
		moduleName = "test-module"
	}

	content := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)

	goModPath := "go.mod"
	err := os.WriteFile(goModPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write go.mod file: %v", err)
	}

	return goModPath
}
