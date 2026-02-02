package cmd_test

import (
	"os"
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestDbPull_RequiresConfigFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Don't create config file

	err := cmd.RunDbPull([]string{})
	if err == nil {
		t.Error("runDbPull should fail without config file")
	}
}

func TestDbPull_RequiresDatabase(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")

	// Without DATABASE_URL, should fail
	err := cmd.RunDbPull([]string{})
	if err == nil {
		t.Error("runDbPull should fail without DATABASE_URL")
	}
}

func TestDbPull_GeneratesSchema(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.SkipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := cmd.CreateIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := cmd.GetTestDBURL(t, dbName)
	cleanupEnv := cmd.SetEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	// Create test tables in the database using SQL
	cmd.ExecSQL(t, testDBURL,
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

	cmd.CreateTestConfig(t, "")

	// Ensure prisma directory exists for the generated schema
	if err := os.MkdirAll("prisma", 0755); err != nil {
		t.Fatalf("Failed to create prisma directory: %v", err)
	}

	// Run db pull to generate schema from existing tables
	err := cmd.RunDbPull([]string{})

	if err != nil {
		t.Logf("DB pull completed with: %v", err)
	}

	// Verify schema file was created
	if !cmd.FileExists("prisma/schema.prisma") {
		t.Error("Schema file should exist after db pull")
	} else {
		schemaContent := cmd.ReadFile(t, "prisma/schema.prisma")
		if !cmd.Contains(schemaContent, "model") {
			t.Error("Schema should contain model definitions")
		} else {
			t.Log("Schema generated successfully")
		}
	}
}
