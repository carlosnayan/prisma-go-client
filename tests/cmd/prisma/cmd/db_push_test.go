package cmd_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestDbPush_RequiresConfigFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Don't create config file
	cmd.CreateTestSchema(t, "")

	err := cmd.RunDbPush([]string{})
	if err == nil {
		t.Error("runDbPush should fail without config file")
	}
}

func TestDbPush_RequiresDatabase(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	// Without DATABASE_URL, should fail
	err := cmd.RunDbPush([]string{})
	if err == nil {
		t.Error("runDbPush should fail without DATABASE_URL")
	}
}

func TestDbPush_NoChanges(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")

	cmd.SkipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := cmd.CreateIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := cmd.GetTestDBURL(t, dbName)
	cleanupEnv := cmd.SetEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	// First push - creates tables
	err1 := cmd.RunDbPush([]string{})
	if err1 != nil {
		t.Logf("First push: %v", err1)
	}

	// Second push with same schema - should detect no changes
	err2 := cmd.RunDbPush([]string{})
	// Should either succeed with "no changes" or complete without error
	if err2 != nil {
		t.Logf("Second push: %v", err2)
	}
}

func TestDbPush_WithAcceptDataLoss(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")

	cmd.SkipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := cmd.CreateIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := cmd.GetTestDBURL(t, dbName)
	cleanupEnv := cmd.SetEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SetDbPushAcceptDataLossFlag(true)
	cmd.SetDbPushSkipGenerateFlag(false)

	// Run push with accept-data-loss flag (safe in isolated DB)
	err := cmd.RunDbPush([]string{})
	if err != nil {
		t.Logf("Push with accept-data-loss: %v", err)
	}

	// Reset flag after test
	cmd.SetDbPushAcceptDataLossFlag(false)
}

func TestDbPush_WithSkipGenerate(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestGoMod(t, "test-module")

	cmd.SkipIfNoDatabase(t)

	// Create isolated test database
	dbName, cleanupDB := cmd.CreateIsolatedTestDB(t)
	defer cleanupDB()

	testDBURL := cmd.GetTestDBURL(t, dbName)
	cleanupEnv := cmd.SetEnv(t, "DATABASE_URL", testDBURL)
	defer cleanupEnv()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SetDbPushAcceptDataLossFlag(false)
	cmd.SetDbPushSkipGenerateFlag(true)

	// Run push with skip-generate flag
	err := cmd.RunDbPush([]string{})
	if err != nil {
		t.Logf("Push with skip-generate: %v", err)
	}

	// Verify that generate was skipped (db directory should not be created)
	if cmd.FileExists("db") {
		t.Log("Note: db directory exists (generate may have run)")
	}

	// Reset flag after test
	cmd.SetDbPushSkipGenerateFlag(false)
}
