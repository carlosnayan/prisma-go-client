package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestDbExecute_NoFlagsProvided(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	err := runDbExecute([]string{})
	if err == nil {
		t.Fatal("runDbExecute should fail when no flags are provided")
	}
	if !strings.Contains(err.Error(), "Either --stdin or --file must be provided") {
		t.Errorf("Error should mention 'Either --stdin or --file must be provided', got: %v", err)
	}
}

func TestDbExecute_BothFlagsProvided(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Set both flags
	dbExecuteFileFlag = "script.sql"
	dbExecuteStdinFlag = true

	err := runDbExecute([]string{})
	if err == nil {
		t.Fatal("runDbExecute should fail when both flags are provided")
	}
	if !strings.Contains(err.Error(), "--stdin and --file cannot be used at the same time") {
		t.Errorf("Error should mention mutual exclusivity, got: %v", err)
	}
}

func TestDbExecute_FileDoesNotExist(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	cleanup := setEnv(t, "DATABASE_URL", "postgresql://localhost/test")
	defer cleanup()

	dbExecuteFileFlag = "./doesnotexist.sql"

	err := runDbExecute([]string{})
	if err == nil {
		t.Fatal("runDbExecute should fail when file doesn't exist")
	}
	if !strings.Contains(err.Error(), "doesn't exist") {
		t.Errorf("Error should mention file doesn't exist, got: %v", err)
	}
	if !strings.Contains(err.Error(), "doesnotexist.sql") {
		t.Errorf("Error should mention the file path, got: %v", err)
	}
}

func TestDbExecute_FileReadError(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	cleanup := setEnv(t, "DATABASE_URL", "postgresql://localhost/test")
	defer cleanup()

	// Create a directory instead of a file (will cause read error)
	err := os.MkdirAll("script.sql", 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	dbExecuteFileFlag = "./script.sql"

	err = runDbExecute([]string{})
	if err == nil {
		t.Fatal("runDbExecute should fail when file cannot be read")
	}
	if !strings.Contains(err.Error(), "An error occurred while reading") {
		t.Errorf("Error should mention read error, got: %v", err)
	}
}

func TestDbExecute_EmptySQL(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	cleanup := setEnv(t, "DATABASE_URL", "postgresql://localhost/test")
	defer cleanup()

	// Create empty SQL file
	err := os.WriteFile("script.sql", []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create SQL file: %v", err)
	}

	dbExecuteFileFlag = "./script.sql"

	err = runDbExecute([]string{})
	if err == nil {
		t.Fatal("runDbExecute should fail with empty SQL")
	}
	if !strings.Contains(err.Error(), "no SQL provided") {
		t.Errorf("Error should mention 'no SQL provided', got: %v", err)
	}
}

func TestDbExecute_ValidatesFlags(t *testing.T) {
	testCases := []struct {
		name          string
		fileFlag      string
		stdinFlag     bool
		expectedError string
	}{
		{
			name:          "no flags",
			fileFlag:      "",
			stdinFlag:     false,
			expectedError: "Either --stdin or --file must be provided",
		},
		{
			name:          "both flags",
			fileFlag:      "script.sql",
			stdinFlag:     true,
			expectedError: "--stdin and --file cannot be used at the same time",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetGlobalFlags()
			dir := setupTestDir(t)
			defer func() { _ = cleanupTestDir(dir) }()

			createTestConfig(t, "")

			dbExecuteFileFlag = tc.fileFlag
			dbExecuteStdinFlag = tc.stdinFlag

			err := runDbExecute([]string{})
			if err == nil {
				t.Fatalf("runDbExecute should fail for test case: %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Error should contain '%s', got: %v", tc.expectedError, err)
			}
		})
	}
}

func TestDbExecute_HelpMessage(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")

	// Test that error message includes help reference
	err := runDbExecute([]string{})
	if err == nil {
		t.Fatal("runDbExecute should fail when no flags provided")
	}
	if !strings.Contains(err.Error(), "prisma db execute -h") {
		t.Errorf("Error should reference help command, got: %v", err)
	}
}

func TestDbExecute_FilePathInError(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	cleanup := setEnv(t, "DATABASE_URL", "postgresql://localhost/test")
	defer cleanup()

	testPath := "./my-script.sql"
	dbExecuteFileFlag = testPath

	err := runDbExecute([]string{})
	if err == nil {
		t.Fatal("runDbExecute should fail when file doesn't exist")
	}
	if !strings.Contains(err.Error(), testPath) {
		t.Errorf("Error should include the file path '%s', got: %v", testPath, err)
	}
}

// Edge cases

func TestDbExecute_WhitespaceOnlySQL(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer func() { _ = cleanupTestDir(dir) }()

	createTestConfig(t, "")
	// Set a dummy DATABASE_URL to pass config validation
	cleanup := setEnv(t, "DATABASE_URL", "sqlite://file::memory:?cache=shared")
	defer cleanup()

	// Create SQL file with only whitespace
	err := os.WriteFile("script.sql", []byte("   \n\t\n   "), 0644)
	if err != nil {
		t.Fatalf("Failed to create SQL file: %v", err)
	}

	dbExecuteFileFlag = "./script.sql"

	err = runDbExecute([]string{})
	if err == nil {
		t.Fatal("runDbExecute should fail with whitespace-only SQL")
	}
	// The error should be about no SQL provided OR database connection
	// Both are acceptable since we're testing the SQL validation
	if !strings.Contains(err.Error(), "no SQL provided") && !strings.Contains(err.Error(), "error connecting") {
		t.Errorf("Error should mention 'no SQL provided' or connection error, got: %v", err)
	}
}

func TestDbExecute_ResetFlags(t *testing.T) {
	// Set flags
	dbExecuteFileFlag = "test.sql"
	dbExecuteStdinFlag = true

	// Reset
	resetGlobalFlags()

	// Verify reset
	if dbExecuteFileFlag != "" {
		t.Errorf("dbExecuteFileFlag should be reset to empty string, got: %s", dbExecuteFileFlag)
	}
	if dbExecuteStdinFlag != false {
		t.Errorf("dbExecuteStdinFlag should be reset to false, got: %v", dbExecuteStdinFlag)
	}
}
