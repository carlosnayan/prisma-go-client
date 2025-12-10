package generator

import (
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/migrations"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateDriver generates the driver.go file based on the provider in the schema
func GenerateDriver(schema *parser.Schema, outputDir string) error {
	// Get provider from schema
	provider := migrations.GetProviderFromSchema(schema)
	provider = strings.ToLower(provider)

	// Detect user module for local imports
	userModule, err := detectUserModule(outputDir)
	if err != nil {
		// Fallback
		userModule = ""
	}

	// Calculate local import path for builder
	builderPath, _, err := calculateLocalImportPath(userModule, outputDir)
	if err != nil || builderPath == "" {
		// Fallback to old path if detection fails
		builderPath = "github.com/carlosnayan/prisma-go-client/db/builder"
	}

	// Prepare template data
	data := DriverTemplateData{
		Provider:    provider,
		BuilderPath: builderPath,
	}

	// Determine template names based on provider
	var templateNames []string
	templateNames = append(templateNames, "imports.tmpl")

	switch provider {
	case "postgresql":
		templateNames = append(templateNames,
			"postgresql_driver.tmpl",
			"config_helper.tmpl",
			"setup_client_postgresql.tmpl",
			"sqldb_adapter.tmpl",
		)
	case "mysql":
		templateNames = append(templateNames,
			"mysql_driver.tmpl",
			"config_helper.tmpl",
			"setup_client_sql.tmpl",
			"sqldb_adapter.tmpl",
		)
	case "sqlite":
		templateNames = append(templateNames,
			"sqlite_driver.tmpl",
			"config_helper.tmpl",
			"setup_client_sql.tmpl",
			"sqldb_adapter.tmpl",
		)
	default:
		// Default to PostgreSQL
		templateNames = append(templateNames,
			"postgresql_driver.tmpl",
			"config_helper.tmpl",
			"setup_client_postgresql.tmpl",
			"sqldb_adapter.tmpl",
		)
	}

	// Generate driver.go using templates
	// Generate driver.go using templates with package "db" for root directory
	return executeTemplatesFromDirWithPackage(outputDir, "driver.go", "driver", templateNames, data, "db")
}
