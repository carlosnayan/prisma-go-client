package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_ValidSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runValidate([]string{})
	if err != nil {
		t.Fatalf("runValidate failed with valid schema: %v", err)
	}
}

func TestValidate_InvalidSyntax(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createInvalidSchema(t)

	err := runValidate([]string{})
	if err == nil {
		t.Error("runValidate should fail with invalid syntax")
	}
}

func TestValidate_ValidationErrors(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema with a relation to non-existent model
	// This should trigger a validation error
	schemaWithError := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "../db"
}

model users {
  id String @id
  email String
  posts NonExistentModel[] @relation
}
`
	err := os.MkdirAll("prisma", 0755)
	if err != nil {
		t.Fatalf("Failed to create prisma dir: %v", err)
	}
	err = os.WriteFile("prisma/schema.prisma", []byte(schemaWithError), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	err = runValidate([]string{})
	if err == nil {
		t.Error("runValidate should fail with validation errors")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid schema") {
		t.Errorf("Error should mention 'invalid schema', got: %v", err)
	}
}

func TestValidate_ShowsSummary(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	// Capture output would be ideal, but for now just verify it doesn't error
	err := runValidate([]string{})
	if err != nil {
		t.Fatalf("runValidate failed: %v", err)
	}
}

func TestValidate_WithCustomSchemaPath(t *testing.T) {
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
	createTestSchema(t, "")
	err = os.Rename("prisma/schema.prisma", customSchemaPath)
	if err != nil {
		t.Fatalf("Failed to move schema: %v", err)
	}

	schemaPath = customSchemaPath
	err = runValidate([]string{})
	if err != nil {
		t.Fatalf("runValidate failed with custom schema path: %v", err)
	}
}

func TestValidate_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file
	createTestSchema(t, "")

	err := runValidate([]string{})
	if err == nil {
		t.Error("runValidate should fail without config file")
	}
	if !strings.Contains(err.Error(), "prisma.conf not found") {
		t.Errorf("Error should mention prisma.conf, got: %v", err)
	}
}
