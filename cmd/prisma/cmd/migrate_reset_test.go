package cmd

import (
	"testing"
)

func TestMigrateReset_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file
	createTestSchema(t, "")

	err := runMigrateReset([]string{})
	if err == nil {
		t.Error("runMigrateReset should fail without config file")
	}
}

func TestMigrateReset_RequiresDatabase(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	// Without DATABASE_URL, should fail
	err := runMigrateReset([]string{})
	if err == nil {
		t.Error("runMigrateReset should fail without DATABASE_URL")
	}
}

func TestMigrateReset_ResetsDatabase(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")

	// Skip if database is not available (Docker not running in general tests)
	skipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := createIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := getTestDBURL(t, dbName)
	cleanupEnv := setEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	createTestConfig(t, "")
	createTestSchema(t, `datasource db {
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
	execSQL(t, testDBURL, `
		CREATE TABLE "User" (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT
		)
	`)

	// Verify table exists before reset
	if !tableExists(t, testDBURL, "User") {
		t.Fatal("User table should exist before reset")
	}

	// Run reset with non-interactive mode (skip confirmation)
	cleanupSkipConfirm := setEnv(t, "PRISMA_MIGRATE_SKIP_CONFIRM", "true")
	defer cleanupSkipConfirm()

	err := runMigrateReset([]string{})
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Verify table was dropped after reset
	if tableExists(t, testDBURL, "User") {
		t.Error("User table should not exist after reset")
	}
}
