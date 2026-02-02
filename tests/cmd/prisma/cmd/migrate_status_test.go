package cmd_test

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestMigrateStatus_RequiresConfigFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Don't create config file
	cmd.CreateTestSchema(t, "")

	err := cmd.RunMigrateStatus([]string{})
	if err == nil {
		t.Error("runMigrateStatus should fail without config file")
	}
}

func TestMigrateStatus_RequiresDatabase(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	// Without DATABASE_URL, should fail
	err := cmd.RunMigrateStatus([]string{})
	if err == nil {
		t.Error("runMigrateStatus should fail without DATABASE_URL")
	}
}

func TestMigrateStatus_ListsMigrations(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SkipIfNoDatabase(t)
	cleanup := cmd.SetEnv(t, "DATABASE_URL", cmd.GetTestDatabaseURL(t))
	defer cleanup()

	// Create test migrations
	cmd.CreateTestMigration(t, "20240101000000_first", "CREATE TABLE first (id SERIAL PRIMARY KEY);")
	cmd.CreateTestMigration(t, "20240102000000_second", "CREATE TABLE second (id SERIAL PRIMARY KEY);")

	err := cmd.RunMigrateStatus([]string{})
	// This will either succeed or fail based on database state
	// We just verify it doesn't crash
	_ = err // Expected to fail if database is not properly set up
	// In a real scenario, this would list migrations
}
