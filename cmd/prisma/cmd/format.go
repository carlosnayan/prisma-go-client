package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/formatter"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

var (
	formatWriteFlag bool
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
			Name:  "write",
			Usage: "Write changes to file (default: true)",
			Value: &formatWriteFlag,
		},
		{
			Name:  "check",
			Usage: "Only check if file is formatted",
			Value: &formatCheckFlag,
		},
	},
	Run: runFormat,
}

func runFormat(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	write := formatWriteFlag
	if !formatCheckFlag && !formatWriteFlag {
		write = true // default
	}
	check := formatCheckFlag

	schemaPath := getSchemaPath()
	fmt.Printf("Formatting %s...\n", schemaPath)
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

	// Read original file to compare
	original, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Normalize strings for comparison (remove extra spaces)
	originalStr := strings.TrimSpace(string(original))
	formattedStr := strings.TrimSpace(formatted)

	// Check if formatting is needed
	if originalStr == formattedStr {
		fmt.Println("Schema is already formatted!")
		return nil
	}

	if check {
		fmt.Println("Warning: Schema needs formatting")
		fmt.Println("   Run 'prisma format' without --check to format")
		return fmt.Errorf("schema is not formatted")
	}

	// Write back
	if write {
		if err := os.WriteFile(schemaPath, []byte(formatted), 0644); err != nil {
			return fmt.Errorf("error writing file: %w", err)
		}
		fmt.Println("Schema formatted and saved!")
	} else {
		fmt.Println("Formatted schema (use --write to save):")
		fmt.Println()
		fmt.Println(formatted)
	}

	return nil
}
