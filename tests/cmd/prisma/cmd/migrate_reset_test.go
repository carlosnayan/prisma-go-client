package cmd_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestMigrateReset_RequiresConfigFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Don't create config file
	cmd.CreateTestSchema(t, "")

	err := cmd.RunMigrateReset([]string{})
	if err == nil {
		t.Error("runMigrateReset should fail without config file")
	}
}

func TestMigrateReset_RequiresDatabase(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	// Without DATABASE_URL, should fail
	err := cmd.RunMigrateReset([]string{})
	if err == nil {
		t.Error("runMigrateReset should fail without DATABASE_URL")
	}
}

func TestMigrateReset_ResetsDatabase(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")

	// Skip if database is not available (Docker not running in general tests)
	cmd.SkipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := cmd.CreateIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := cmd.GetTestDBURL(t, dbName)
	cleanupEnv := cmd.SetEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, `datasource db {
  provider = "postgresql"
}

generator client {
  provider = "prisma-client-go"
  output   = "./db"
}

model User {
  id    String @id @default(uuid())
  email String @unique
  name  String?
}`)

	// Create table directly to simulate an existing database
	cmd.ExecSQL(t, testDBURL, `
		CREATE TABLE "User" (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT
		)
	`)

	// Verify table exists before reset
	if !cmd.TableExists(t, testDBURL, "User") {
		t.Fatal("User table should exist before reset")
	}

	// Run reset with non-interactive mode (skip confirmation)
	cleanupSkipConfirm := cmd.SetEnv(t, "PRISMA_MIGRATE_SKIP_CONFIRM", "true")
	defer cleanupSkipConfirm()

	err := cmd.RunMigrateReset([]string{})
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Verify table was dropped after reset
	if cmd.TableExists(t, testDBURL, "User") {
		t.Error("User table should not exist after reset")
	}
}
