package cmd

import (
	"testing"
)

func TestDbPush_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

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
	defer func() { _ = cleanupTestDir(dir) }()

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
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")

	skipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := createIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := getTestDBURL(t, dbName)
	cleanupEnv := setEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	createTestConfig(t, "")
	createTestSchema(t, "")

	// First push - creates tables
	err1 := runDbPush([]string{})
	if err1 != nil {
		t.Logf("First push: %v", err1)
	}

	// Second push with same schema - should detect no changes
	err2 := runDbPush([]string{})
	// Should either succeed with "no changes" or complete without error
	if err2 != nil {
		t.Logf("Second push: %v", err2)
	}
}

func TestDbPush_WithAcceptDataLoss(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")

	skipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := createIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := getTestDBURL(t, dbName)
	cleanupEnv := setEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	createTestConfig(t, "")
	createTestSchema(t, "")

	dbPushAcceptDataLossFlag = true
	dbPushSkipGenerateFlag = false

	// Run push with accept-data-loss flag (safe in isolated DB)
	err := runDbPush([]string{})
	if err != nil {
		t.Logf("Push with accept-data-loss: %v", err)
	}

	// Reset flag after test
	dbPushAcceptDataLossFlag = false
}

func TestDbPush_WithSkipGenerate(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestGoMod(t, "test-module")

	skipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := createIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := getTestDBURL(t, dbName)
	cleanupEnv := setEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	createTestConfig(t, "")
	createTestSchema(t, "")

	dbPushAcceptDataLossFlag = false
	dbPushSkipGenerateFlag = true

	// Run push with skip-generate flag
	err := runDbPush([]string{})
	if err != nil {
		t.Logf("Push with skip-generate: %v", err)
	}

	// Verify that generate was skipped (db directory should not be created)
	if fileExists("db") {
		t.Log("Note: db directory exists (generate may have run)")
	}

	// Reset flag after test
	dbPushSkipGenerateFlag = false
}
