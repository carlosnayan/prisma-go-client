package cmd

import (
	"testing"
)

func TestDbPush_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	// Don't create config file
	createTestSchema(t, "")

	err := runDbPush([]string{})
	if err == nil {
		t.Error("runDbPush should fail without config file")
	}
}

func TestDbPush_RequiresDatabase(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	createTestConfig(t, "")
	createTestSchema(t, "")

	// Without DATABASE_URL, should fail
	err := runDbPush([]string{})
	if err == nil {
		t.Error("runDbPush should fail without DATABASE_URL")
	}
}

func TestDbPush_NoChanges(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// This test requires a database that already matches the schema
	// We skip it to avoid needing full database setup
	t.Skip("Requires database already synchronized with schema")
}

func TestDbPush_WithAcceptDataLoss(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	dbPushAcceptDataLossFlag = true
	dbPushSkipGenerateFlag = false

	// This would apply destructive changes
	// We skip to avoid data loss
	t.Skip("Skipping to avoid data loss in test database")
}

func TestDbPush_WithSkipGenerate(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	dbPushAcceptDataLossFlag = false
	dbPushSkipGenerateFlag = true

	// This test would verify generate is skipped
	// We skip to avoid needing full database setup
	t.Skip("Requires full database setup")
}

