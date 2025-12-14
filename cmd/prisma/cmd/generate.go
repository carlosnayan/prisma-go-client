package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/generator"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

const version = "0.2.2"

var (
	watchFlag         bool
	generatorFlags    []string
	noHintsFlag       bool
	requireModelsFlag bool
)

var generateCmd = &cli.Command{
	Name:  "generate",
	Short: "Generate Go types and query builders from schema.prisma",
	Long: `Generates type-safe Go code based on schema.prisma:
  - Structs for each model
  - Type-safe query builders
  - Auxiliary types (CreateInput, UpdateInput, WhereInput)
  - Main Prisma client`,
	Flags: []*cli.Flag{
		{
			Name:  "watch",
			Short: "w",
			Usage: "Watch the Prisma schema and rerun after a change",
			Value: &watchFlag,
		},
		{
			Name:  "generator",
			Usage: "Generator to use (may be provided multiple times)",
			Value: &generatorFlags,
		},
		{
			Name:  "no-hints",
			Usage: "Hides the hint messages but still outputs errors and warnings",
			Value: &noHintsFlag,
		},
		{
			Name:  "require-models",
			Usage: "Do not allow generating a client without models",
			Value: &requireModelsFlag,
		},
	},
	Run: runGenerate,
}

func runGenerate(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	schemaPath := getSchemaPath()

	// Parse --generator flags from args (can appear multiple times)
	// This is done manually because the CLI framework doesn't support multiple values natively
	parsedGeneratorFlags := parseGeneratorFlags(args)
	if len(parsedGeneratorFlags) > 0 {
		generatorFlags = parsedGeneratorFlags
	}

	// If watch mode, run in watch loop
	if watchFlag {
		return runGenerateWatch(schemaPath)
	}

	// Single generation
	return runGenerateOnce(schemaPath)
}

// parseGeneratorFlags extracts --generator flags from args
func parseGeneratorFlags(args []string) []string {
	var generators []string
	for i, arg := range args {
		if arg == "--generator" || arg == "-g" {
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				generators = append(generators, args[i+1])
			}
		} else if strings.HasPrefix(arg, "--generator=") {
			value := strings.TrimPrefix(arg, "--generator=")
			generators = append(generators, value)
		}
	}
	return generators
}

func runGenerateOnce(schemaPath string) error {
	// Show config loaded
	if !noHintsFlag {
		fmt.Println()
		fmt.Println("Loaded Prisma config from prisma.conf.")
		fmt.Printf("Prisma schema loaded from %s\n", schemaPath)
	}

	// Start timing
	startTime := time.Now()

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
		return fmt.Errorf("error parsing schema: %w", err)
	}

	// Check if models are required
	if requireModelsFlag && len(schema.Models) == 0 {
		return fmt.Errorf("no models found in schema. Use --require-models=false to allow generating without models")
	}

	// Filter generators if --generator flags are provided
	generatorsToUse := filterGenerators(schema, generatorFlags)

	// Determine output directory from generator in schema
	outputDir := "./db" // default
	for _, gen := range generatorsToUse {
		for _, field := range gen.Fields {
			if field.Name == "output" {
				if val, ok := field.Value.(string); ok {
					outputDir = val
					break
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

	// Ensure path is absolute or relative to schema directory
	// If output starts with ./, it means relative to the schema directory
	// This ensures that ./db in prisma/schema.prisma generates to prisma/db
	absoluteOutputDir := outputDir
	if !filepath.IsAbs(outputDir) {
		schemaDir := filepath.Dir(schemaPath)

		// If output starts with ./, resolve relative to schema directory
		if strings.HasPrefix(outputDir, "./") {
			cleanOutputDir := strings.TrimPrefix(outputDir, "./")
			absoluteOutputDir = filepath.Join(schemaDir, cleanOutputDir)
			absoluteOutputDir, _ = filepath.Abs(absoluteOutputDir)
		} else if strings.HasPrefix(outputDir, "..") {
			// If output starts with .., resolve relative to schema directory
			absoluteOutputDir = filepath.Join(schemaDir, outputDir)
			absoluteOutputDir, _ = filepath.Abs(absoluteOutputDir)
		} else {
			// For other relative paths, resolve relative to schema directory
			// This ensures consistency: all relative paths are relative to schema location
			absoluteOutputDir = filepath.Join(schemaDir, outputDir)
			absoluteOutputDir, _ = filepath.Abs(absoluteOutputDir)
		}
	}

	// Cleanup existing directories to ensure fresh generation
	dirsToClean := []string{"inputs", "models", "queries", "filters"}
	for _, dirName := range dirsToClean {
		dirPath := filepath.Join(absoluteOutputDir, dirName)
		if _, err := os.Stat(dirPath); err == nil {
			if err := os.RemoveAll(dirPath); err != nil {
				return fmt.Errorf("error cleaning %s directory: %w", dirName, err)
			}
		}
	}

	if err := generator.GenerateModels(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating models: %w", err)
	}

	if err := generator.GenerateRaw(absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating raw: %w", err)
	}

	if err := generator.GenerateUtils(absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating utils: %w", err)
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

	if err := generator.GenerateFilters(schema, absoluteOutputDir); err != nil {
		return fmt.Errorf("error generating filters: %w", err)
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
	if !noHintsFlag {
		fmt.Printf("\n✔ Generated Prisma Client (%s) to %s in %dms\n", version, outputDir, elapsedMs)
	} else {
		fmt.Printf("✔ Generated Prisma Client to %s in %dms\n", outputDir, elapsedMs)
	}

	// Update Go module cache (only if not in watch mode and hints are enabled)
	if !noHintsFlag {
		fmt.Print("\nUpdating Go module cache... ")
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Stdout = os.Stderr // Redirect to stderr to avoid polluting output
		cmd.Stderr = os.Stderr

		// Execute in project directory
		wd, err := filepath.Abs(".")
		if err == nil {
			cmd.Dir = wd
		}

		if err := cmd.Run(); err != nil {
			// Don't fail if go mod tidy fails, just warn
			fmt.Printf("⚠ Warning: failed to run 'go mod tidy': %v\n", err)
			fmt.Println("  You may need to run 'go mod tidy' manually for staticcheck to recognize new packages.")
		}
	}

	if !noHintsFlag {
		fmt.Println()
		fmt.Println("\nNext steps:")
		fmt.Println("  Import your Prisma Client in your code:")
		fmt.Printf("    import \"%s\"\n", absoluteOutputDir)
		fmt.Println()
	}

	return nil
}

// filterGenerators filters generators based on --generator flags
func filterGenerators(schema *parser.Schema, generatorNames []string) []*parser.Generator {
	if len(generatorNames) == 0 {
		return schema.Generators
	}

	var filtered []*parser.Generator
	for _, gen := range schema.Generators {
		// Get generator provider name
		var providerName string
		for _, field := range gen.Fields {
			if field.Name == "provider" {
				if val, ok := field.Value.(string); ok {
					providerName = val
					break
				}
			}
		}

		// Check if this generator matches any of the requested names
		for _, requestedName := range generatorNames {
			if providerName == requestedName || strings.Contains(providerName, requestedName) {
				filtered = append(filtered, gen)
				break
			}
		}
	}

	return filtered
}

// runGenerateWatch runs generate in watch mode, monitoring schema changes
func runGenerateWatch(schemaPath string) error {
	schemaDir := filepath.Dir(schemaPath)
	if schemaDir == "." {
		schemaDir, _ = filepath.Abs(".")
	}

	fmt.Println()
	fmt.Printf("Watching... %s\n", schemaDir)
	fmt.Println()

	if err := runGenerateOnce(schemaPath); err != nil {
		fmt.Printf("Error in initial generation: %v\n", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	lastModTime := getFileModTime(schemaPath)
	pollInterval := 500 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentModTime := getFileModTime(schemaPath)
			if currentModTime != lastModTime && !currentModTime.IsZero() {
				lastModTime = currentModTime

				relPath, _ := filepath.Rel(".", schemaPath)
				fmt.Printf("Change detected in %s\n", relPath)
				fmt.Println("Building...")
				fmt.Println()

				if err := runGenerateOnce(schemaPath); err != nil {
					fmt.Printf("Error during generation: %v\n", err)
					fmt.Println()
				} else {
					fmt.Printf("Watching... %s\n", schemaDir)
					fmt.Println()
				}
			}

		case <-sigChan:
			fmt.Println("\nStopping watch mode...")
			return nil
		}
	}
}

func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
