package cmd

import (
	"os"
	"testing"
)

func TestDbPull_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file

	err := runDbPull([]string{})
	if err == nil {
		t.Error("runDbPull should fail without config file")
	}
}

func TestDbPull_RequiresDatabase(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Without DATABASE_URL, should fail
	err := runDbPull([]string{})
	if err == nil {
		t.Error("runDbPull should fail without DATABASE_URL")
	}
}

func TestDbPull_GeneratesSchema(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	skipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := createIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := getTestDBURL(t, dbName)
	cleanupEnv := setEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	// Create test tables in the database using SQL
	execSQL(t, testDBURL,
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT
		)`,
		`CREATE TABLE posts (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			user_id TEXT REFERENCES users(id)
		)`,
	)

	createTestConfig(t, "")

	// Ensure prisma directory exists for the generated schema
	if err := os.MkdirAll("prisma", 0755); err != nil {
		t.Fatalf("Failed to create prisma directory: %v", err)
	}

	// Run db pull to generate schema from existing tables
	err := runDbPull([]string{})

	if err != nil {
		t.Logf("DB pull completed with: %v", err)
	}

	// Verify schema file was created
	if !fileExists("prisma/schema.prisma") {
		t.Error("Schema file should exist after db pull")
	} else {
		schemaContent := readFile(t, "prisma/schema.prisma")
		if !contains(schemaContent, "model") {
			t.Error("Schema should contain model definitions")
		} else {
			t.Log("Schema generated successfully")
		}
	}
}
