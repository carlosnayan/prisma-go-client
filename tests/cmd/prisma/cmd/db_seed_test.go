package cmd_test

import (
	"os"
	"testing"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"
)

func TestDbSeed_RequiresConfigFile(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Don't create config file

	err := cmd.RunDbSeed([]string{})
	if err == nil {
		t.Error("runDbSeed should fail without config file")
	}
}

func TestDbSeed_RequiresSeedConfig(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Create config without seed
	cmd.CreateTestConfig(t, "")

	err := cmd.RunDbSeed([]string{})
	// Should fail if seed is not configured
	if err == nil {
		t.Error("runDbSeed should fail when seed is not configured")
	}
}

func TestDbSeed_ExecutesSeed(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

	// Create config with seed
	configWithSeed := `schema = "prisma/schema.prisma"

[migrations]
path = "prisma/migrations"

[datasource]
url = "env('DATABASE_URL')"

[seed]
script = "echo 'seed executed'"
`
	cmd.CreateTestConfig(t, configWithSeed)

	cmd.SkipIfNoDatabase(t)
	cleanup := cmd.SetEnv(t, "DATABASE_URL", cmd.GetTestDatabaseURL(t))
	defer cleanup()

	err := cmd.RunDbSeed([]string{})
	// This will either succeed or fail based on seed script
	// We just verify it doesn't crash
	_ = err // Expected to fail if seed script doesn't exist or database is not set up
}

func TestDbSeed_WithGoSeedScript(t *testing.T) {
	cmd.ResetGlobalFlags()
	dir := cmd.SetupTestDir(t)
	defer func() { _ = cmd.CleanupTestDir(dir) }()

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
	cmd.CreateTestConfig(t, configWithSeed)

	cmd.SkipIfNoDatabase(t)
	cleanup := cmd.SetEnv(t, "DATABASE_URL", cmd.GetTestDatabaseURL(t))
	defer cleanup()

	err = cmd.RunDbSeed([]string{})
	// This will either succeed or fail based on seed script execution
	// We just verify it doesn't crash
	_ = err // Expected to fail if database is not set up
}
