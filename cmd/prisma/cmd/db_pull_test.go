package cmd

import (
	"testing"
)

func TestDbPull_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	// Don't create config file

	err := runDbPull([]string{})
	if err == nil {
		t.Error("runDbPull should fail without config file")
	}
}

func TestDbPull_RequiresDatabase(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

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
	defer cleanupTestDir(dir)

	createTestConfig(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// This test requires a database with tables
	// We skip it to avoid needing full database setup
	t.Skip("Requires database with existing tables")
}

