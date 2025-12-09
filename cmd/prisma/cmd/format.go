package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/formatter"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

var (
	formatCheckFlag bool
)

var formatCmd = &cli.Command{
	Name:  "format",
	Short: "Format the schema.prisma file",
	Long: `Formats the schema.prisma file applying style conventions:
  - Sorts models, fields and attributes
  - Applies consistent indentation
  - Removes unnecessary whitespace`,
	Flags: []*cli.Flag{
		{
			Name:  "schema",
			Short: "s",
			Usage: "Custom path to your Prisma schema",
			Value: &schemaPath,
		},
		{
			Name:  "check",
			Usage: "Only check if file is formatted (does not write)",
			Value: &formatCheckFlag,
		},
	},
	Run: runFormat,
}

func runFormat(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	startTime := time.Now()
	check := formatCheckFlag

	schemaPath := getSchemaPath()

	// Print schema loaded message (consistent with official)
	relPath, err := filepath.Rel(".", schemaPath)
	if err != nil {
		relPath = schemaPath
	}
	fmt.Printf("Prisma schema loaded from %s\n", relPath)
	fmt.Println()

	// Parse schema
	schema, errors, err := parser.ParseFile(schemaPath)
	if err != nil || len(errors) > 0 {
		fmt.Println("Syntax errors found in schema:")
		if len(errors) > 0 {
			for i, e := range errors {
				fmt.Printf("  %d. %s\n", i+1, e)
			}
		}
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		}
		return fmt.Errorf("cannot format schema with syntax errors - fix errors first")
	}

	// Safety check: schema should not be nil at this point
	if schema == nil {
		return fmt.Errorf("error: schema is nil after parsing")
	}

	// Format
	formatted := formatter.FormatSchema(schema)
	if formatted == "" {
		return fmt.Errorf("error: formatting returned empty string")
	}

	// Validate the formatted schema
	formattedSchema, validationErrors, validateErr := parser.Parse(formatted)
	if validateErr != nil || len(validationErrors) > 0 {
		return fmt.Errorf("formatted schema is invalid - this should not happen")
	}

	// Ensure formatted schema is valid
	if formattedSchema == nil {
		return fmt.Errorf("formatted schema is nil after parsing")
	}

	// Read original file to compare
	original, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Normalize strings for comparison
	// Remove trailing whitespace from each line and normalize line endings
	originalLines := strings.Split(string(original), "\n")
	formattedLines := strings.Split(formatted, "\n")

	// Trim trailing whitespace from each line
	for i := range originalLines {
		originalLines[i] = strings.TrimRight(originalLines[i], " \t")
	}
	for i := range formattedLines {
		formattedLines[i] = strings.TrimRight(formattedLines[i], " \t")
	}

	originalStr := strings.Join(originalLines, "\n")
	formattedStr := strings.Join(formattedLines, "\n")

	// Normalize line endings and trim
	originalStr = strings.TrimSpace(originalStr)
	formattedStr = strings.TrimSpace(formattedStr)

	// Check if formatting is needed
	if originalStr == formattedStr {
		if check {
			fmt.Println("All files are formatted correctly!")
			return nil
		}
		fmt.Printf("The schema at %s is already formatted!\n", relPath)
		return nil
	}

	if check {
		fmt.Printf("There are unformatted files. Run %s to format them.\n", "prisma format")
		return fmt.Errorf("schema is not formatted")
	}

	// Write back
	if err := os.WriteFile(schemaPath, []byte(formatted), 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	// Calculate elapsed time
	elapsed := time.Since(startTime)
	elapsedMs := elapsed.Milliseconds()

	// Show success message (consistent with official format)
	fmt.Printf("Formatted %s in %dms 🚀\n", relPath, elapsedMs)

	return nil
}
