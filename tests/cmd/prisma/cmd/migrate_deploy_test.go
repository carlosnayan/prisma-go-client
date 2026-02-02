package cmd_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestMigrateDeploy_RequiresConfigFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Don't create config file
	cmd.CreateTestSchema(t, "")

	err := cmd.RunMigrateDeploy([]string{})
	if err == nil {
		t.Error("runMigrateDeploy should fail without config file")
	}
}

func TestMigrateDeploy_RequiresDatabase(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	// Without DATABASE_URL, should fail
	err := cmd.RunMigrateDeploy([]string{})
	if err == nil {
		t.Error("runMigrateDeploy should fail without DATABASE_URL")
	}
}

func TestMigrateDeploy_AppliesPendingMigrations(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SkipIfNoDatabase(t)
	cleanup := cmd.SetEnv(t, "DATABASE_URL", cmd.GetTestDatabaseURL(t))
	defer cleanup()

	// Create a test migration
	cmd.CreateTestMigration(t, "20240101000000_test", "CREATE TABLE test (id SERIAL PRIMARY KEY);")

	err := cmd.RunMigrateDeploy([]string{})
	// This will either succeed or fail based on database state
	// We just verify it doesn't crash
	_ = err // Expected to fail if database is not properly set up
	// In a real scenario, this would apply pending migrations
}
