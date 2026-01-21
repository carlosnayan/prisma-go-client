package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateBuilder generates a standalone builder package in the output directory
// This package has no external dependencies on github.com/carlosnayan/prisma-go-client
func GenerateBuilder(schema *parser.Schema, outputDir string) error {
	builderDir := filepath.Join(outputDir, "builder")
	if err := os.MkdirAll(builderDir, 0755); err != nil {
		return fmt.Errorf("failed to create builder directory: %w", err)
	}

	userModule, err := detectUserModule(outputDir)
	if err != nil {
		return fmt.Errorf("failed to detect user module: %w", err)
	}

	utilsPath, err := calculateUtilsImportPath(userModule, outputDir)
	if err != nil {
		return fmt.Errorf("failed to calculate utils import path: %w", err)
	}

	if err := generateBuilderFoundation(builderDir, utilsPath); err != nil {
		return fmt.Errorf("failed to generate foundation.go: %w", err)
	}

	if err := generateBuilderDialect(builderDir); err != nil {
		return fmt.Errorf("failed to generate dialect.go: %w", err)
	}

	provider := getProviderFromSchema(schema)
	if err := generateBuilderMain(builderDir, provider, utilsPath); err != nil {
		return fmt.Errorf("failed to generate builder.go: %w", err)
	}

	if err := generateBuilderFluent(builderDir, provider, utilsPath); err != nil {
		return fmt.Errorf("failed to generate fluent.go: %w", err)
	}

	if err := generateBuilderFields(builderDir, utilsPath); err != nil {
		return fmt.Errorf("failed to generate builder_fields.go: %w", err)
	}

	return nil
}

// getProviderFromSchema extracts the provider from the schema
func getProviderFromSchema(schema *parser.Schema) string {
	if len(schema.Datasources) > 0 {
		for _, field := range schema.Datasources[0].Fields {
			if field.Name == "provider" {
				if str, ok := field.Value.(string); ok {
					return strings.ToLower(str)
				}
			}
		}
	}
	return "postgresql" // default
}

// generateBuilderFoundation generates foundation.go using templates
func generateBuilderFoundation(builderDir, utilsPath string) error {
	data := struct {
		UtilsPath        string
		UtilsPackageName string
	}{
		UtilsPath:        utilsPath,
		UtilsPackageName: "utils",
	}
	filePath := filepath.Join(builderDir, "foundation.go")
	return executeModelTemplate(filePath, "builder", "builder_main", "foundation.tmpl", data)
}

// generateBuilderFields generates builder_fields.go using templates
func generateBuilderFields(builderDir, utilsPath string) error {
	data := struct {
		UtilsPath        string
		UtilsPackageName string
	}{
		UtilsPath:        utilsPath,
		UtilsPackageName: "utils",
	}

	filePath := filepath.Join(builderDir, "builder_fields.go")
	return executeModelTemplate(filePath, "builder", "builder_main", "builder_fields.tmpl", data)
}
