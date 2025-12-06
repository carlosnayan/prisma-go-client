package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestInit_CreatesProjectStructure(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

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
	defer cleanupTestDir(dir)

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

func TestInit_CreatesCorrectSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

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
	defer cleanupTestDir(dir)

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
	defer cleanupTestDir(dir)

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
	defer cleanupTestDir(dir)

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
	defer cleanupTestDir(dir)

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
			defer cleanupTestDir(dir)

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

