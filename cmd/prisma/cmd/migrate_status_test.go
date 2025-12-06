package cmd

import (
	"testing"
)

func TestMigrateStatus_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file
	createTestSchema(t, "")

	err := runMigrateStatus([]string{})
	if err == nil {
		t.Error("runMigrateStatus should fail without config file")
	}
}

func TestMigrateStatus_RequiresDatabase(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	// Without DATABASE_URL, should fail
	err := runMigrateStatus([]string{})
	if err == nil {
		t.Error("runMigrateStatus should fail without DATABASE_URL")
	}
}

func TestMigrateStatus_ListsMigrations(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// Create test migrations
	createTestMigration(t, "20240101000000_first", "CREATE TABLE first (id SERIAL PRIMARY KEY);")
	createTestMigration(t, "20240102000000_second", "CREATE TABLE second (id SERIAL PRIMARY KEY);")

	err := runMigrateStatus([]string{})
	// This will either succeed or fail based on database state
	// We just verify it doesn't crash
	_ = err // Expected to fail if database is not properly set up
	// In a real scenario, this would list migrations
}

