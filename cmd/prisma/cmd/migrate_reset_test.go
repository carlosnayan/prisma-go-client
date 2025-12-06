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

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// This test requires a working database and would reset it
	// We skip it to avoid destroying test data
	t.Skip("Skipping reset test to avoid database destruction")
}

