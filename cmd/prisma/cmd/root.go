package cmd

import (
	"fmt"
	"os"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/config"
	"github.com/carlosnayan/prisma-go-client/internal/logger"
)

var (
	configFile string
	schemaPath string
	verbose    bool
)

var app *cli.App

// Execute runs the CLI application
func Execute() error {
	app = cli.NewApp(
		"prisma",
		"0.1.7",
		"Prisma CLI for Go - Type-safe and intuitive ORM",
	)

	// Global flags
	app.AddGlobalFlag(&cli.Flag{
		Name:  "config",
		Short: "c",
		Usage: "Path to configuration file (default: prisma.conf)",
		Value: &configFile,
	})
	app.AddGlobalFlag(&cli.Flag{
		Name:  "schema",
		Short: "s",
		Usage: "Path to schema.prisma (default: prisma/schema.prisma)",
		Value: &schemaPath,
	})
	app.AddGlobalFlag(&cli.Flag{
		Name:  "verbose",
		Short: "v",
		Usage: "Verbose mode (show detailed logs)",
		Value: &verbose,
	})

	// Commands
	app.AddCommand(initCmd)
	app.AddCommand(generateCmd)
	app.AddCommand(validateCmd)
	app.AddCommand(formatCmd)
	app.AddCommand(migrateCmd)
	app.AddCommand(dbCmd)

	return app.Execute()
}

// getConfigPath returns the path to the configuration file
func getConfigPath() string {
	if configFile != "" {
		return configFile
	}
	// Look for prisma.conf in project root
	if _, err := os.Stat("prisma.conf"); err == nil {
		return "prisma.conf"
	}
	return ""
}

// getSchemaPath returns the path to schema.prisma
func getSchemaPath() string {
	if schemaPath != "" {
		return schemaPath
	}
	// Try to load from configuration
	if cfg, err := loadConfig(); err == nil {
		return cfg.GetSchemaPath()
	}
	// Default: prisma/schema.prisma
	if _, err := os.Stat("prisma/schema.prisma"); err == nil {
		return "prisma/schema.prisma"
	}
	// Fallback: schema.prisma in root
	if _, err := os.Stat("schema.prisma"); err == nil {
		return "schema.prisma"
	}
	return "prisma/schema.prisma"
}

// checkProjectRoot checks if we are in the root of a Prisma project
func checkProjectRoot() error {
	configPath := getConfigPath()
	if configPath == "" {
		return fmt.Errorf("prisma.conf not found. Run 'prisma init' to initialize the project")
	}
	return nil
}

// loadConfig loads the configuration from prisma.conf
func loadConfig() (*config.Config, error) {
	configPath := getConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("prisma.conf not found")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	// Configure logger if specified in [debug] section
	if cfg.Debug != nil && len(cfg.Debug.Log) > 0 {
		logger.SetLogLevels(cfg.Debug.Log)
	}

	return cfg, nil
}
