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
		return fmt.Errorf("erro ao criar diretório builder: %w", err)
	}

	// Generate all builder files
	if err := generateBuilderWhere(builderDir); err != nil {
		return fmt.Errorf("erro ao gerar where.go: %w", err)
	}

	if err := generateBuilderOptions(builderDir); err != nil {
		return fmt.Errorf("erro ao gerar options.go: %w", err)
	}

	if err := generateBuilderDialect(builderDir); err != nil {
		return fmt.Errorf("erro ao gerar dialect.go: %w", err)
	}

	if err := generateBuilderErrors(builderDir); err != nil {
		return fmt.Errorf("erro ao gerar errors.go: %w", err)
	}

	if err := generateBuilderLimits(builderDir); err != nil {
		return fmt.Errorf("erro ao gerar limits.go: %w", err)
	}

	if err := generateBuilderContext(builderDir); err != nil {
		return fmt.Errorf("erro ao gerar context.go: %w", err)
	}

	// Get provider from schema to generate appropriate builder
	provider := getProviderFromSchema(schema)
	if err := generateBuilderMain(builderDir, provider); err != nil {
		return fmt.Errorf("erro ao gerar builder.go: %w", err)
	}

	if err := generateBuilderFluent(builderDir, provider); err != nil {
		return fmt.Errorf("erro ao gerar fluent.go: %w", err)
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
