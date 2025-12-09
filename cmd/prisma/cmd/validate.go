package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

var validateCmd = &cli.Command{
	Name:  "validate",
	Short: "Validate the schema.prisma",
	Long: `Validates the syntax and consistency of schema.prisma:
  - Checks correct syntax
  - Validates relationships
  - Validates custom types (enums, models)
  - Detects errors and warnings`,
	Flags: []*cli.Flag{
		{
			Name:  "schema",
			Short: "s",
			Usage: "Custom path to your Prisma schema",
			Value: &schemaPath,
		},
	},
	Run: runValidate,
}

func runValidate(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	schemaPath := getSchemaPath()

	// Print schema loaded message (consistent with official)
	relPath, err := filepath.Rel(".", schemaPath)
	if err != nil {
		relPath = schemaPath
	}
	fmt.Printf("Prisma schema loaded from %s\n", relPath)
	fmt.Println()

	// Parse and validate schema
	schema, errors, err := parser.ParseFile(schemaPath)
	if err != nil {
		if len(errors) > 0 {
			fmt.Println("Prisma schema validation errors:")
			fmt.Println()
			for i, e := range errors {
				fmt.Printf("  %d. %s\n", i+1, e)
			}
			fmt.Println()
			fmt.Printf("Validation Error Count: %d\n", len(errors))
			return fmt.Errorf("invalid schema")
		}
		return fmt.Errorf("error parsing schema: %w", err)
	}

	// Additional validations
	validationErrors := performAdditionalValidations(schema)
	if len(validationErrors) > 0 {
		fmt.Println("Prisma schema validation errors:")
		fmt.Println()
		for i, e := range validationErrors {
			fmt.Printf("  %d. %s\n", i+1, e)
		}
		fmt.Println()
		fmt.Printf("Validation Error Count: %d\n", len(validationErrors))
		return fmt.Errorf("invalid schema")
	}

	// Validate relations more thoroughly
	relationErrors := validateRelationsThoroughly(schema)
	if len(relationErrors) > 0 {
		fmt.Println("Prisma schema validation errors:")
		fmt.Println()
		for i, e := range relationErrors {
			fmt.Printf("  %d. %s\n", i+1, e)
		}
		fmt.Println()
		fmt.Printf("Validation Error Count: %d\n", len(relationErrors))
		return fmt.Errorf("invalid schema")
	}

	// Show success message (consistent with official format)
	fmt.Printf("The schema at %s is valid 🚀\n", relPath)

	return nil
}

// performAdditionalValidations performs additional validations beyond parser
func performAdditionalValidations(schema *parser.Schema) []string {
	var errors []string

	// Create maps for quick lookup
	modelNames := make(map[string]bool)
	enumNames := make(map[string]bool)

	for _, model := range schema.Models {
		modelNames[model.Name] = true
	}

	for _, enum := range schema.Enums {
		enumNames[enum.Name] = true
	}

	// Validate that all field types reference valid models or enums
	for _, model := range schema.Models {
		for _, field := range model.Fields {
			if field.Type != nil {
				typeName := field.Type.Name

				// Check if it's a custom type (model or enum)
				if !isBuiltinType(typeName) {
					// Check if it's a valid model or enum
					if !modelNames[typeName] && !enumNames[typeName] {
						errors = append(errors, fmt.Sprintf(
							"Field '%s' in model '%s' has invalid type '%s'. Type must be a built-in type, model, or enum",
							field.Name, model.Name, typeName))
					}
				}
			}
		}
	}

	// Validate that datasource exists
	if len(schema.Datasources) == 0 {
		errors = append(errors, "Schema must have at least one datasource")
	}

	// Note: Generators are optional but recommended
	// Schemas without generators are allowed

	return errors
}

// isBuiltinType checks if a type is a built-in Prisma type
func isBuiltinType(typeName string) bool {
	// Remove array and optional markers
	cleanType := strings.TrimSuffix(strings.TrimSuffix(typeName, "[]"), "?")

	builtinTypes := map[string]bool{
		"String":      true,
		"Int":         true,
		"BigInt":      true,
		"Float":       true,
		"Decimal":     true,
		"Boolean":     true,
		"DateTime":    true,
		"Json":        true,
		"Bytes":       true,
		"Unsupported": true,
	}

	return builtinTypes[cleanType]
}

// validateRelationsThoroughly performs thorough relation validation
func validateRelationsThoroughly(schema *parser.Schema) []string {
	var errors []string

	// Create model map
	models := make(map[string]*parser.Model)
	for _, model := range schema.Models {
		models[model.Name] = model
	}

	// Validate all relations
	for _, model := range schema.Models {
		for _, field := range model.Fields {
			// Check if field type references another model
			if field.Type != nil {
				typeName := field.Type.Name

				// If it's not a builtin type and not an enum, it should be a model
				if !isBuiltinType(typeName) {
					// Check if it's an enum
					isEnum := false
					for _, enum := range schema.Enums {
						if enum.Name == typeName {
							isEnum = true
							break
						}
					}

					// If it's not an enum and not a model, it's an error
					if !isEnum && models[typeName] == nil {
						errors = append(errors, fmt.Sprintf(
							"Field '%s' in model '%s' references unknown type '%s'",
							field.Name, model.Name, typeName))
					}
				}
			}

			// Validate @relation attributes
			for _, attr := range field.Attributes {
				if attr.Name == "relation" {
					errors = append(errors, validateRelationAttribute(attr, model, field, models)...)
				}
			}
		}
	}

	return errors
}

// validateRelationAttribute validates a @relation attribute
func validateRelationAttribute(attr *parser.Attribute, model *parser.Model, field *parser.ModelField, models map[string]*parser.Model) []string {
	var errors []string

	// Find the related model
	var relatedModelName string
	if field.Type != nil {
		relatedModelName = field.Type.Name
	}

	// Check if related model exists
	if relatedModelName != "" && !isBuiltinType(relatedModelName) {
		if relatedModel := models[relatedModelName]; relatedModel == nil {
			errors = append(errors, fmt.Sprintf(
				"Relation in field '%s' of model '%s' references non-existent model '%s'",
				field.Name, model.Name, relatedModelName))
			return errors
		}

		// Validate fields and references if present
		var fields []string
		var references []string

		for _, arg := range attr.Arguments {
			if arg.Name == "fields" {
				if vals, ok := arg.Value.([]interface{}); ok {
					for _, val := range vals {
						if str, ok := val.(string); ok {
							fields = append(fields, str)
						}
					}
				}
			}
			if arg.Name == "references" {
				if vals, ok := arg.Value.([]interface{}); ok {
					for _, val := range vals {
						if str, ok := val.(string); ok {
							references = append(references, str)
						}
					}
				}
			}
		}

		// If fields/references are specified, validate them
		if len(fields) > 0 || len(references) > 0 {
			if len(fields) != len(references) {
				errors = append(errors, fmt.Sprintf(
					"Relation in field '%s' of model '%s' must have equal number of 'fields' and 'references'",
					field.Name, model.Name))
			}

			// Validate that fields exist in current model
			for _, fieldName := range fields {
				found := false
				for _, f := range model.Fields {
					if f.Name == fieldName {
						found = true
						break
					}
				}
				if !found {
					errors = append(errors, fmt.Sprintf(
						"Relation in field '%s' of model '%s' references non-existent field '%s'",
						field.Name, model.Name, fieldName))
				}
			}

			// Validate that references exist in related model
			if relatedModel := models[relatedModelName]; relatedModel != nil {
				for _, refName := range references {
					found := false
					for _, f := range relatedModel.Fields {
						if f.Name == refName {
							found = true
							break
						}
					}
					if !found {
						errors = append(errors, fmt.Sprintf(
							"Relation in field '%s' of model '%s' references non-existent field '%s' in model '%s'",
							field.Name, model.Name, refName, relatedModelName))
					}
				}
			}
		}
	}

	return errors
}
