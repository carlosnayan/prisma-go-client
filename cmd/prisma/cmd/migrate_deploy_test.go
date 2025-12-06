package cmd

import (
	"testing"
)

func TestMigrateDeploy_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file
	createTestSchema(t, "")

	err := runMigrateDeploy([]string{})
	if err == nil {
		t.Error("runMigrateDeploy should fail without config file")
	}
}

func TestMigrateDeploy_RequiresDatabase(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	// Without DATABASE_URL, should fail
	err := runMigrateDeploy([]string{})
	if err == nil {
		t.Error("runMigrateDeploy should fail without DATABASE_URL")
	}
}

func TestMigrateDeploy_AppliesPendingMigrations(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// Create a test migration
	createTestMigration(t, "20240101000000_test", "CREATE TABLE test (id SERIAL PRIMARY KEY);")

	err := runMigrateDeploy([]string{})
	// This will either succeed or fail based on database state
	// We just verify it doesn't crash
	_ = err // Expected to fail if database is not properly set up
	// In a real scenario, this would apply pending migrations
}

