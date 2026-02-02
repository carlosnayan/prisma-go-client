package cmd_test

import (
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestMigrateResolve_RequiresConfigFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Don't create config file
	cmd.CreateTestSchema(t, "")

	err := cmd.RunMigrateResolve([]string{})
	if err == nil {
		t.Error("runMigrateResolve should fail without config file")
	}
}

func TestMigrateResolve_RequiresFlag(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SkipIfNoDatabase(t)
	cleanup := cmd.SetEnv(t, "DATABASE_URL", cmd.GetTestDatabaseURL(t))
	defer cleanup()

	// No flags set
	cmd.SetMigrateResolveAppliedFlag("")
	cmd.SetMigrateResolveRolledBackFlag("")

	err := cmd.RunMigrateResolve([]string{})
	if err == nil {
		t.Error("runMigrateResolve should fail without --applied or --rolled-back")
	}
	if !strings.Contains(err.Error(), "--applied or --rolled-back") {
		t.Errorf("Error should mention required flags, got: %v", err)
	}
}

func TestMigrateResolve_WithAppliedFlag(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SkipIfNoDatabase(t)
	cleanup := cmd.SetEnv(t, "DATABASE_URL", cmd.GetTestDatabaseURL(t))
	defer cleanup()

	cmd.SetMigrateResolveAppliedFlag("20240101000000_test")
	cmd.SetMigrateResolveRolledBackFlag("")

	err := cmd.RunMigrateResolve([]string{})
	// This will either succeed or fail based on database state
	// We just verify it doesn't crash immediately
	if err != nil && strings.Contains(err.Error(), "--applied or --rolled-back") {
		t.Error("Should not fail with flag requirement when --applied is set")
	}
}

func TestMigrateResolve_WithRolledBackFlag(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	cmd.CreateTestConfig(t, "")
	cmd.CreateTestSchema(t, "")

	cmd.SkipIfNoDatabase(t)
	cleanup := cmd.SetEnv(t, "DATABASE_URL", cmd.GetTestDatabaseURL(t))
	defer cleanup()

	cmd.SetMigrateResolveAppliedFlag("")
	cmd.SetMigrateResolveRolledBackFlag("20240101000000_test")

	err := cmd.RunMigrateResolve([]string{})
	// This will either succeed or fail based on database state
	// We just verify it doesn't crash immediately
	if err != nil && strings.Contains(err.Error(), "--applied or --rolled-back") {
		t.Error("Should not fail with flag requirement when --rolled-back is set")
	}
}
