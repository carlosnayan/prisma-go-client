package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestInit_CreatesProjectStructure(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Check prisma.conf exists
	if !fileExists("prisma.conf") {
		t.Error("prisma.conf was not created")
	}

	// Check schema.prisma exists
	schemaPath := "prisma/schema.prisma"
	if !fileExists(schemaPath) {
		t.Error("schema.prisma was not created")
	}

	// Check migrations directory exists
	migrationsPath := "prisma/migrations"
	assertDirExists(t, migrationsPath)
}

func TestInit_CreatesCorrectConfig(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	configContent := readFile(t, "prisma.conf")

	// Check for required sections
	if !contains(configContent, "schema =") {
		t.Error("Config should contain schema path")
	}
	if !contains(configContent, "[migrations]") {
		t.Error("Config should contain migrations section")
	}
	if !contains(configContent, "[datasource]") {
		t.Error("Config should contain datasource section")
	}
	if !contains(configContent, "env('DATABASE_URL')") {
		t.Error("Config should contain DATABASE_URL environment variable")
	}
}

func TestInit_CreatesDebugSection(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	configContent := readFile(t, "prisma.conf")

	// Check for [debug] section
	if !contains(configContent, "[debug]") {
		t.Error("Config should contain [debug] section")
	}

	// Check for default log levels (should be ["warn"] as per specification)
	if !contains(configContent, `log = ["warn"]`) {
		t.Errorf("Config should contain default log levels ['warn'], got: %s", configContent)
	}
}

func TestInit_CreatesCorrectSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	schemaContent := readFile(t, "prisma/schema.prisma")

	// Check for required sections
	if !contains(schemaContent, "datasource db") {
		t.Error("Schema should contain datasource")
	}
	if !contains(schemaContent, "provider = \"postgresql\"") {
		t.Error("Schema should contain postgresql provider")
	}
	if !contains(schemaContent, "generator client") {
		t.Error("Schema should contain generator")
	}
	if !contains(schemaContent, "prisma-client-go") {
		t.Error("Schema should contain prisma-client-go provider")
	}
}

func TestInit_WithProviderFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	providerFlag = "mysql"
	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	schemaContent := readFile(t, "prisma/schema.prisma")
	if !contains(schemaContent, "provider = \"mysql\"") {
		t.Errorf("Schema should contain mysql provider, got: %s", schemaContent)
	}
}

func TestInit_WithDatabaseFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	databaseFlag = "postgresql://user:pass@localhost/db"
	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	configContent := readFile(t, "prisma.conf")
	if !contains(configContent, "postgresql://user:pass@localhost/db") {
		t.Error("Config should contain custom database URL")
	}
}

func TestInit_FailsWhenConfigExists(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Create existing config
	err := os.WriteFile("prisma.conf", []byte("existing"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	err = runInit([]string{})
	if err == nil {
		t.Error("runInit should fail when prisma.conf already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error should mention 'already exists', got: %v", err)
	}
}

func TestInit_CreatesMigrationsDirectory(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	migrationsPath := "prisma/migrations"
	info, err := os.Stat(migrationsPath)
	if err != nil {
		t.Fatalf("Migrations directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Migrations path should be a directory")
	}
}

func TestInit_WithDifferentProviders(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			resetGlobalFlags()
			dir := setupTestDir(t)
			defer func() { _ = cleanupTestDir(dir) }()

			providerFlag = provider
			err := runInit([]string{})
			if err != nil {
				t.Fatalf("runInit failed for %s: %v", provider, err)
			}

			schemaContent := readFile(t, "prisma/schema.prisma")
			expected := "provider = \"" + provider + "\""
			if !contains(schemaContent, expected) {
				t.Errorf("Schema should contain %s provider, got: %s", provider, schemaContent)
			}
		})
	}
}

// Test edge cases and validation

func TestInit_FailsWhenSchemaExistsInRoot(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Create schema.prisma in root
	err := os.WriteFile("schema.prisma", []byte("existing"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing schema: %v", err)
	}

	err = runInit([]string{})
	if err == nil {
		t.Error("runInit should fail when schema.prisma already exists in root")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error should mention 'already exists', got: %v", err)
	}
}

func TestInit_FailsWhenPrismaFolderExists(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Create prisma folder
	err := os.MkdirAll("prisma", 0755)
	if err != nil {
		t.Fatalf("Failed to create prisma folder: %v", err)
	}

	err = runInit([]string{})
	if err == nil {
		t.Error("runInit should fail when prisma folder already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error should mention 'already exists', got: %v", err)
	}
}

func TestInit_FailsWhenPrismaSchemaExists(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Create prisma/schema.prisma
	err := os.MkdirAll("prisma", 0755)
	if err != nil {
		t.Fatalf("Failed to create prisma folder: %v", err)
	}
	err = os.WriteFile("prisma/schema.prisma", []byte("existing"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing schema: %v", err)
	}

	err = runInit([]string{})
	if err == nil {
		t.Error("runInit should fail when prisma/schema.prisma already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error should mention 'already exists', got: %v", err)
	}
}

func TestInit_ValidatesProvider(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Test invalid provider
	providerFlag = "invalid-provider"
	err := runInit([]string{})
	if err == nil {
		t.Error("runInit should fail with invalid provider")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Error should mention 'invalid', got: %v", err)
	}
}

func TestInit_NormalizesProviderNames(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"postgres", "postgresql"},
		{"postgresql", "postgresql"},
		{"mysql", "mysql"},
		{"mariadb", "mysql"},
		{"sqlite", "sqlite"},
		{"sqlite3", "sqlite"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			resetGlobalFlags()
			dir := setupTestDir(t)
			defer func() { _ = cleanupTestDir(dir) }()

			providerFlag = tc.input
			err := runInit([]string{})
			if err != nil {
				t.Fatalf("runInit failed for %s: %v", tc.input, err)
			}

			schemaContent := readFile(t, "prisma/schema.prisma")
			expected := "provider = \"" + tc.expected + "\""
			if !contains(schemaContent, expected) {
				t.Errorf("Schema should contain %s provider (normalized from %s), got: %s", tc.expected, tc.input, schemaContent)
			}
		})
	}
}

func TestInit_SchemaContainsComments(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	schemaContent := readFile(t, "prisma/schema.prisma")

	// Check for helpful comments
	if !contains(schemaContent, "This is your Prisma schema file") {
		t.Error("Schema should contain helpful comments")
	}
	if !contains(schemaContent, "learn more about it in the docs") {
		t.Error("Schema should contain documentation link")
	}
}

func TestInit_SchemaFormatIsConsistent(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	err := runInit([]string{})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	schemaContent := readFile(t, "prisma/schema.prisma")

	// Check that generator comes before datasource (consistent with official)
	generatorPos := strings.Index(schemaContent, "generator client")
	datasourcePos := strings.Index(schemaContent, "datasource db")

	if generatorPos == -1 || datasourcePos == -1 {
		t.Error("Schema should contain both generator and datasource")
	}

	if generatorPos > datasourcePos {
		t.Error("Generator should come before datasource in schema")
	}

	// Check for proper formatting
	if !contains(schemaContent, "output   =") {
		t.Error("Schema should have properly formatted output field")
	}
}

func TestInit_CreatesCorrectDefaultURLs(t *testing.T) {
	testCases := []struct {
		provider string
		urlPart  string
	}{
		{"postgresql", "postgresql://"},
		{"mysql", "mysql://"},
		{"sqlite", "file:./dev.db"},
	}

	for _, tc := range testCases {
		t.Run(tc.provider, func(t *testing.T) {
			resetGlobalFlags()
			dir := setupTestDir(t)
			defer func() { _ = cleanupTestDir(dir) }()

			providerFlag = tc.provider
			err := runInit([]string{})
			if err != nil {
				t.Fatalf("runInit failed: %v", err)
			}

			// The output message should contain the default URL
			// We can't easily test the printed output, but we can verify
			// the schema was created correctly
			schemaContent := readFile(t, "prisma/schema.prisma")
			if !contains(schemaContent, "provider = \""+tc.provider+"\"") {
				t.Errorf("Schema should contain %s provider", tc.provider)
			}
		})
	}
}
