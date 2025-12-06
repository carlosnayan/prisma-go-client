package cmd

import (
	"fmt"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

var validateCmd = &cli.Command{
	Name:  "validate",
	Short: "Validate the schema.prisma",
	Long: `Validates the syntax and consistency of schema.prisma:
  - Checks correct syntax
  - Validates relationships
  - Detects errors and warnings`,
	Run: runValidate,
}

func runValidate(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	schemaPath := getSchemaPath()
	fmt.Printf("Validating %s...\n", schemaPath)
	fmt.Println()

	// Parse and validate schema
	schema, errors, err := parser.ParseFile(schemaPath)
	if err != nil {
		if len(errors) > 0 {
			fmt.Println("Errors found:")
			for i, e := range errors {
				fmt.Printf("  %d. %s\n", i+1, e)
			}
			return fmt.Errorf("invalid schema")
		}
		return err
	}

	// Show summary
	fmt.Printf("Schema is valid!\n\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  Datasources: %d\n", len(schema.Datasources))
	fmt.Printf("  Generators: %d\n", len(schema.Generators))
	fmt.Printf("  Models: %d\n", len(schema.Models))
	fmt.Printf("  Enums: %d\n", len(schema.Enums))

	return nil
}
