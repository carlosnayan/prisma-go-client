package cmd

import (
	"strings"
	"testing"
)

func TestMigrateResolve_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	// Don't create config file
	createTestSchema(t, "")

	err := runMigrateResolve([]string{})
	if err == nil {
		t.Error("runMigrateResolve should fail without config file")
	}
}

func TestMigrateResolve_RequiresFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	// No flags set
	migrateResolveAppliedFlag = ""
	migrateResolveRolledBackFlag = ""

	err := runMigrateResolve([]string{})
	if err == nil {
		t.Error("runMigrateResolve should fail without --applied or --rolled-back")
	}
	if !strings.Contains(err.Error(), "--applied or --rolled-back") {
		t.Errorf("Error should mention required flags, got: %v", err)
	}
}

func TestMigrateResolve_WithAppliedFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	migrateResolveAppliedFlag = "20240101000000_test"
	migrateResolveRolledBackFlag = ""

	err := runMigrateResolve([]string{})
	// This will either succeed or fail based on database state
	// We just verify it doesn't crash immediately
	if err != nil && strings.Contains(err.Error(), "--applied or --rolled-back") {
		t.Error("Should not fail with flag requirement when --applied is set")
	}
}

func TestMigrateResolve_WithRolledBackFlag(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	createTestSchema(t, "")

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	migrateResolveAppliedFlag = ""
	migrateResolveRolledBackFlag = "20240101000000_test"

	err := runMigrateResolve([]string{})
	// This will either succeed or fail based on database state
	// We just verify it doesn't crash immediately
	if err != nil && strings.Contains(err.Error(), "--applied or --rolled-back") {
		t.Error("Should not fail with flag requirement when --rolled-back is set")
	}
}
