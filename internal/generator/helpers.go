package generator

import (
	"fmt"
	"path/filepath"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateHelpers generates type-safe helper functions in the generated package
func GenerateHelpers(schema *parser.Schema, outputDir string) error {
	// Detect user module
	userModule, err := detectUserModule(outputDir)
	if err != nil {
		return fmt.Errorf("failed to detect user module: %w", err)
	}

	helpersFile := filepath.Join(outputDir, "helpers.go")
	return generateHelpersFile(helpersFile, userModule, outputDir, schema)
}

// generateHelpersFile generates the file with type-safe helper functions using templates
func generateHelpersFile(filePath string, userModule, outputDir string, schema *parser.Schema) error {
	// Calculate import path for inputs
	_, _, inputsPath, err := calculateImportPath(userModule, outputDir)
	if err != nil {
		// Fallback
		inputsPath = "github.com/carlosnayan/prisma-go-client/generated/inputs"
	}

	// Determine which filter types are needed based on schema
	neededFilters := determineNeededFilters(schema)

	// Build imports list based on needed filters
	imports := make([]string, 0, 3)
	if neededFilters["DateTimeFilter"] {
		imports = append(imports, "time")
	}
	if neededFilters["JsonFilter"] {
		imports = append(imports, "encoding/json")
	}

	// Prepare template data
	data := HelpersTemplateData{
		Imports:       imports,
		InputsPath:    inputsPath,
		NeededFilters: neededFilters,
	}

	// Build list of templates to execute based on needed filters
	templateNames := []string{"imports.tmpl"}

	if neededFilters["StringFilter"] {
		templateNames = append(templateNames, "string.tmpl")
	}
	if neededFilters["IntFilter"] {
		templateNames = append(templateNames, "int.tmpl")
	}
	if neededFilters["Int64Filter"] {
		templateNames = append(templateNames, "int64.tmpl")
	}
	if neededFilters["FloatFilter"] {
		templateNames = append(templateNames, "float.tmpl")
	}
	if neededFilters["BooleanFilter"] {
		templateNames = append(templateNames, "boolean.tmpl")
	}
	if neededFilters["DateTimeFilter"] {
		templateNames = append(templateNames, "datetime.tmpl")
	}
	if neededFilters["JsonFilter"] {
		templateNames = append(templateNames, "json.tmpl")
	}
	if neededFilters["BytesFilter"] {
		templateNames = append(templateNames, "bytes.tmpl")
	}

	// Generate helpers.go using templates
	return executeHelpersTemplates(filePath, templateNames, data)
}

// determineNeededFilters determines which filter types are needed based on the schema
func determineNeededFilters(schema *parser.Schema) map[string]bool {
	neededFilters := make(map[string]bool)

	// Check all models for field types to determine which filters are actually needed
	for _, model := range schema.Models {
		for _, field := range model.Fields {
			if field.Type == nil {
				continue
			}

			// Check if it's a relation using the same logic as inputs.go
			if isRelationForHelpers(field, schema) {
				continue
			}

			filterType := getFilterTypeForHelpers(field.Type)
			neededFilters[filterType] = true
		}
	}

	return neededFilters
}

// getFilterTypeForHelpers returns the appropriate Filter type for a field type
// Uses the shared getFilterType function from inputs.go
func getFilterTypeForHelpers(fieldType *parser.FieldType) string {
	return getFilterType(fieldType)
}

// isRelationForHelpers checks if a field is a relationship (non-primitive type)
// Uses the shared isRelation function from inputs.go
// Note: This function requires schema to be passed, but for backward compatibility
// it accepts nil schema (will treat enums as relations in that case)
func isRelationForHelpers(field *parser.ModelField, schema *parser.Schema) bool {
	return isRelation(field, schema)
}
