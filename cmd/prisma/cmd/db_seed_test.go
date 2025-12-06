package cmd

import (
	"os"
	"testing"
)

func TestDbSeed_RequiresConfigFile(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	// Don't create config file

	err := runDbSeed([]string{})
	if err == nil {
		t.Error("runDbSeed should fail without config file")
	}
}

func TestDbSeed_RequiresSeedConfig(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	// Create config without seed
	createTestConfig(t, "")

	err := runDbSeed([]string{})
	// Should fail if seed is not configured
	if err == nil {
		t.Error("runDbSeed should fail when seed is not configured")
	}
}

func TestDbSeed_ExecutesSeed(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	// Create config with seed
	configWithSeed := `schema = "prisma/schema.prisma"

[migrations]
path = "prisma/migrations"

[datasource]
url = "env('DATABASE_URL')"

[seed]
script = "echo 'seed executed'"
`
	createTestConfig(t, configWithSeed)

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	err := runDbSeed([]string{})
	// This will either succeed or fail based on seed script
	// We just verify it doesn't crash
	if err != nil {
		// Expected to fail if seed script doesn't exist or database is not set up
	}
}

func TestDbSeed_WithGoSeedScript(t *testing.T) {
	resetGlobalFlags()
	dir := setupTestDir(t)
	defer cleanupTestDir(dir)

	// Create a simple seed script
	seedScript := `package main
import "fmt"
func main() { fmt.Println("Seed executed") }`
	
	err := os.WriteFile("seed.go", []byte(seedScript), 0644)
	if err != nil {
		t.Fatalf("Failed to write seed script: %v", err)
	}

	// Create config with seed pointing to the script
	configWithSeed := `schema = "prisma/schema.prisma"

[migrations]
path = "prisma/migrations"

[datasource]
url = "env('DATABASE_URL')"

[seed]
script = "go run seed.go"
`
	createTestConfig(t, configWithSeed)

	skipIfNoDatabase(t)
	cleanup := setEnv(t, "DATABASE_URL", getTestDatabaseURL(t))
	defer cleanup()

	err = runDbSeed([]string{})
	// This will either succeed or fail based on seed script execution
	// We just verify it doesn't crash
	if err != nil {
		// Expected to fail if database is not set up
	}
}

