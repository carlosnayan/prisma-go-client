package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/generator"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

const version = "0.1.4"

var generateCmd = &cli.Command{
	Name:  "generate",
	Short: "Generate Go types and query builders from schema.prisma",
	Long: `Generates type-safe Go code based on schema.prisma:
  - Structs for each model
  - Type-safe query builders
  - Auxiliary types (CreateInput, UpdateInput, WhereInput)
  - Main Prisma client`,
	Run: runGenerate,
}

func runGenerate(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	// Show config loaded
	fmt.Println()
	fmt.Println("Loaded Prisma config from prisma.conf.")

	schemaPath := getSchemaPath()
	fmt.Printf("Prisma schema loaded from %s\n", schemaPath)

	// Start timing
	startTime := time.Now()

	// Parse schema
	schema, errors, err := parser.ParseFile(schemaPath)
	if err != nil {
		if len(errors) > 0 {
			fmt.Println()
			fmt.Println("Errors found in schema:")
			for i, e := range errors {
				fmt.Printf("  %d. %s\n", i+1, e)
			}
			return fmt.Errorf("cannot generate code with invalid schema")
		}
		return err
	}

	// Determine output directory from generator in schema
	outputDir := "./db" // default
	for _, gen := range schema.Generators {
		for _, field := range gen.Fields {
			if field.Name == "output" {
				if val, ok := field.Value.(string); ok {
					outputDir = val
				}
			}
		}
	}

	// Try to load from prisma.conf as well (has priority)
	if cfg, err := loadConfig(); err == nil && cfg != nil {
		if cfg.Generator != nil && cfg.Generator.Output != "" {
			outputDir = cfg.Generator.Output
		}
	}

	// Ensure path is absolute or relative to current directory
	absoluteOutputDir := outputDir
	if !filepath.IsAbs(outputDir) {
		wd, _ := filepath.Abs(".")
		absoluteOutputDir = filepath.Join(wd, outputDir)
	}

	// Generate code silently
	if err := generator.GenerateModels(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating models: %w", err)
	}

	if err := generator.GenerateQueries(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating queries: %w", err)
	}

	if err := generator.GenerateClient(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating client: %w", err)
	}

	if err := generator.GenerateInputs(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating inputs: %w", err)
	}

	if err := generator.GenerateHelpers(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating helpers: %w", err)
	}

	if err := generator.GenerateDriver(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating driver: %w", err)
	}

	// Calculate elapsed time
	elapsed := time.Since(startTime)
	elapsedMs := elapsed.Milliseconds()

	// Show success message
	fmt.Printf("\n✔ Generated Prisma Client (%s) to %s in %dms\n", version, outputDir, elapsedMs)
	fmt.Println()

	return nil
}
