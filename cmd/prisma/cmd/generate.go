package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/generator"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

const version = "0.1.6"

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

	// Ensure path is absolute or relative to current working directory
	absoluteOutputDir := outputDir
	if !filepath.IsAbs(outputDir) {
		// Remove leading ./ if present
		cleanOutputDir := strings.TrimPrefix(outputDir, "./")

		// If output starts with .., resolve relative to schema directory
		// Otherwise, resolve relative to current working directory
		if strings.HasPrefix(cleanOutputDir, "..") {
			schemaDir := filepath.Dir(schemaPath)
			absoluteOutputDir = filepath.Join(schemaDir, cleanOutputDir)
			// Clean the path to resolve .. properly
			absoluteOutputDir, _ = filepath.Abs(absoluteOutputDir)
		} else {
			wd, _ := filepath.Abs(".")
			absoluteOutputDir = filepath.Join(wd, cleanOutputDir)
		}
	}

	// Generate code silently
	if err := generator.GenerateModels(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating models: %w", err)
	}

	if err := generator.GenerateRaw(absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating raw: %w", err)
	}

	if err := generator.GenerateBuilder(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating builder: %w", err)
	}

	if err := generator.GenerateInputs(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating inputs: %w", err)
	}

	if err := generator.GenerateQueries(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating queries: %w", err)
	}

	if err := generator.GenerateHelpers(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating helpers: %w", err)
	}

	if err := generator.GenerateClient(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating client: %w", err)
	}

	if err := generator.GenerateDriver(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating driver: %w", err)
	}

	// Calculate elapsed time
	elapsed := time.Since(startTime)
	elapsedMs := elapsed.Milliseconds()

	// Show success message
	fmt.Printf("\n✔ Generated Prisma Client (%s) to %s in %dms\n", version, outputDir, elapsedMs)

	// Atualizar cache do Go para que staticcheck e outras ferramentas reconheçam os novos pacotes
	fmt.Println("Updating Go module cache...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Executar no diretório do projeto
	wd, err := filepath.Abs(".")
	if err == nil {
		cmd.Dir = wd
	}

	if err := cmd.Run(); err != nil {
		// Não falhar se go mod tidy falhar, apenas avisar
		fmt.Printf("⚠ Warning: failed to run 'go mod tidy': %v\n", err)
		fmt.Println("  You may need to run 'go mod tidy' manually for staticcheck to recognize new packages.")
	} else {
		fmt.Println("✔ Go module cache updated")
	}

	// Após go mod tidy, forçar reconhecimento dos pacotes gerados
	fmt.Println("Refreshing Go package cache...")
	refreshCommands := []struct {
		name string
		args []string
		dir  string
	}{
		{"go list", []string{"go", "list", "./..."}, absoluteOutputDir},
		{"go build", []string{"go", "build", "./..."}, absoluteOutputDir},
	}

	for _, cmdInfo := range refreshCommands {
		cmd := exec.Command(cmdInfo.args[0], cmdInfo.args[1:]...)
		cmd.Stdout = os.Stderr // Redirecionar para stderr para não poluir stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = cmdInfo.dir

		if err := cmd.Run(); err != nil {
			// Não falhar, apenas avisar
			fmt.Printf("⚠ Warning: failed to run '%s': %v\n", cmdInfo.name, err)
		}
	}
	fmt.Println("✔ Go package cache refreshed")

	fmt.Println()

	return nil
}
