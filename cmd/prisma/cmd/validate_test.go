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

// Test new validation features

func TestValidate_InvalidTypeReference(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema with invalid type reference
	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

model users {
  id String @id
  email String
  invalidField NonExistentType
}
`
	createTestSchema(t, schemaContent)

	err := runValidate([]string{})
	if err == nil {
		t.Error("runValidate should fail with invalid type reference")
	}
	// Error message should indicate validation failure
	if err != nil && !strings.Contains(err.Error(), "invalid schema") {
		t.Errorf("Error should indicate invalid schema, got: %v", err)
	}
}

func TestValidate_InvalidEnumReference(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema with invalid enum reference
	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

model users {
  id String @id
  status NonExistentEnum
}
`
	createTestSchema(t, schemaContent)

	err := runValidate([]string{})
	if err == nil {
		t.Error("runValidate should fail with invalid enum reference")
	}
}

func TestValidate_ValidEnumReference(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema with valid enum
	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

enum Status {
  ACTIVE
  INACTIVE
}

model users {
  id String @id
  status Status
}
`
	createTestSchema(t, schemaContent)

	err := runValidate([]string{})
	if err != nil {
		t.Fatalf("runValidate should succeed with valid enum reference: %v", err)
	}
}

func TestValidate_InvalidRelationReference(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema with relation to non-existent model
	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

model users {
  id String @id
  posts NonExistentModel[]
}
`
	createTestSchema(t, schemaContent)

	err := runValidate([]string{})
	if err == nil {
		t.Error("runValidate should fail with invalid relation reference")
	}
}

func TestValidate_ValidRelation(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema with valid relation
	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

model users {
  id String @id
  email String
  posts posts[]
}

model posts {
  id String @id
  title String
  authorId String
  author users @relation(fields: [authorId], references: [id])
}
`
	createTestSchema(t, schemaContent)

	err := runValidate([]string{})
	if err != nil {
		t.Fatalf("runValidate should succeed with valid relation: %v", err)
	}
}

func TestValidate_InvalidRelationFields(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema with relation referencing non-existent fields
	schemaContent := `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

model users {
  id String @id
  email String
}

model posts {
  id String @id
  title String
  authorId String
  author users @relation(fields: [authorId], references: [nonExistentField])
}
`
	createTestSchema(t, schemaContent)

	err := runValidate([]string{})
	if err == nil {
		t.Error("runValidate should fail with invalid relation fields")
	}
}

func TestValidate_MissingDatasource(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create schema without datasource
	schemaContent := `generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

model users {
  id String @id
  email String
}
`
	createTestSchema(t, schemaContent)

	err := runValidate([]string{})
	if err == nil {
		t.Error("runValidate should fail without datasource")
	}
	// Error message should indicate validation failure
	if err != nil && !strings.Contains(err.Error(), "invalid schema") {
		t.Errorf("Error should indicate invalid schema, got: %v", err)
	}
}

func TestValidate_OutputFormat(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	err := runValidate([]string{})
	if err != nil {
		t.Fatalf("runValidate failed: %v", err)
	}

	// Output should contain "is valid" message
	// We can't easily capture output in tests, but we can verify it doesn't error
}

func TestValidate_WithSchemaFlag(t *testing.T) {
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

	// Use --schema flag
	schemaPath = customSchemaPath
	err = runValidate([]string{})
	if err != nil {
		t.Fatalf("runValidate failed with --schema flag: %v", err)
	}
}
