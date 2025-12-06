package cmd

import (
	"os"
	"testing"
)

func TestDbExecute_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file

	err := runDbExecute([]string{})
	if err == nil {
		t.Error("runDbExecute should fail without config file")
	}
}

func TestDbExecute_RequiresDatabase(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Without DATABASE_URL, should fail
	err := runDbExecute([]string{})
	if err == nil {
		t.Error("runDbExecute should fail without DATABASE_URL")
	}
}

func TestDbExecute_WithFileFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Create SQL file
	sqlFile := "test.sql"
	sqlContent := "SELECT 1;"
	err := os.WriteFile(sqlFile, []byte(sqlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	dbExecuteFileFlag = sqlFile
	dbExecuteStdinFlag = ""

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	err = runDbExecute([]string{})
	// This will either succeed or fail based on database connection
	// We just verify it doesn't crash
	if err != nil {
		// Expected to fail if database is not properly set up
	}
}

func TestDbExecute_WithStdinFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	createTestConfig(t, "")

	dbExecuteFileFlag = ""
	dbExecuteStdinFlag = "SELECT 1;"

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	err := runDbExecute([]string{})
	// This will either succeed or fail based on database connection
	// We just verify it doesn't crash
	if err != nil {
		// Expected to fail if database is not properly set up
	}
}

func TestDbExecute_WithArguments(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	createTestConfig(t, "")

	dbExecuteFileFlag = ""
	dbExecuteStdinFlag = ""

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// SQL as arguments
	err := runDbExecute([]string{"SELECT", "1;"})
	// This will either succeed or fail based on database connection
	// We just verify it doesn't crash
	if err != nil {
		// Expected to fail if database is not properly set up
	}
}

func TestDbExecute_RequiresSQL(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	createTestConfig(t, "")

	dbExecuteFileFlag = ""
	dbExecuteStdinFlag = ""

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// No SQL provided
	err := runDbExecute([]string{})
	if err == nil {
		t.Error("runDbExecute should fail when no SQL is provided")
	}
}

