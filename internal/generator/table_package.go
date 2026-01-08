package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateTablePackages generates a package for each table/model with column-specific filter methods
func GenerateTablePackages(schema *parser.Schema, outputDir string) error {
	for _, model := range schema.Models {
		// Create table package directory inside tables/
		tablePkgDir := filepath.Join(outputDir, "tables", toSnakeCase(model.Name))
		if err := os.MkdirAll(tablePkgDir, 0755); err != nil {
			return fmt.Errorf("failed to create table package directory for %s: %w", model.Name, err)
		}

		// Prepare field info for templates
		fields := prepareFieldsForFilters(model, schema)

		data := TablePackageTemplateData{
			PackageName: toSnakeCase(model.Name),
			ModelName:   model.Name,
			PascalName:  toPascalCase(model.Name),
			TableName:   getTableName(model),
			Fields:      fields,
			NeedsJSON:   checkNeedsJSON(model),
		}

		// Generate the filter methods file
		templateNames := []string{
			"imports.tmpl",
			"conditions.tmpl",
			"filter_methods.tmpl",
			"constants.tmpl",
		}

		outputFile := filepath.Join(tablePkgDir, toSnakeCase(model.Name)+".go")
		if err := executeTablePackageTemplates(outputFile, templateNames, data); err != nil {
			return fmt.Errorf("failed to generate table package for %s: %w", model.Name, err)
		}
	}

	return nil
}

// FilterFieldInfo holds information about a field for filter generation
type FilterFieldInfo struct {
	Name       string   // Field name in Go (e.g., "Email")
	DBName     string   // Column name in DB (e.g., "email")
	GoType     string   // Go type (e.g., "string", "int")
	IsNullable bool     // Whether the field is nullable
	FilterType string   // Filter type category: "string", "int", "float", "bool", "datetime"
	FilterOps  []string // Available filter operations for this type
}

// prepareFieldsForFilters prepares field information for filter generation
func prepareFieldsForFilters(model *parser.Model, schema *parser.Schema) []FilterFieldInfo {
	var fields []FilterFieldInfo

	for _, field := range model.Fields {
		// Skip relations
		if isRelation(field, schema) {
			continue
		}

		// Determine filter type and operations
		filterType, filterOps := getFilterTypeAndOps(field.Type)

		fields = append(fields, FilterFieldInfo{
			Name:       toPascalCase(field.Name),
			DBName:     field.Name,
			GoType:     mapPrismaTypeToGo(field.Type),
			IsNullable: field.Type.IsOptional,
			FilterType: filterType,
			FilterOps:  filterOps,
		})
	}

	return fields
}

// getFilterTypeAndOps returns the filter type category and available operations
func getFilterTypeAndOps(fieldType *parser.FieldType) (string, []string) {
	typeName := strings.ToLower(fieldType.Name)

	switch typeName {
	case "string":
		return "string", []string{"EQ", "NEQ", "Contains", "StartsWith", "EndsWith", "In", "NotIn", "GT", "GTE", "LT", "LTE"}
	case "int":
		return "int", []string{"EQ", "NEQ", "GT", "GTE", "LT", "LTE", "In", "NotIn"}
	case "bigint":
		return "int64", []string{"EQ", "NEQ", "GT", "GTE", "LT", "LTE", "In", "NotIn"}
	case "float", "decimal":
		return "float", []string{"EQ", "NEQ", "GT", "GTE", "LT", "LTE"}
	case "boolean":
		return "bool", []string{"EQ", "NEQ"}
	case "datetime":
		return "datetime", []string{"EQ", "NEQ", "GT", "GTE", "LT", "LTE"}
	case "json":
		return "json", []string{"EQ", "NEQ"}
	case "bytes":
		return "bytes", []string{"EQ", "NEQ"}
	default:
		return "string", []string{"EQ", "NEQ"}
	}
}

// checkNeedsJSON checks if any field in the model is JSON type
func checkNeedsJSON(model *parser.Model) bool {
	for _, field := range model.Fields {
		if field.Type != nil && strings.ToLower(field.Type.Name) == "json" {
			return true
		}
	}
	return false
}

// executeTablePackageTemplates executes templates from table_package directory
func executeTablePackageTemplates(outputFile string, templateNames []string, data TablePackageTemplateData) error {
	return executeTemplatesFromDirWithPackage(
		filepath.Dir(outputFile),
		filepath.Base(outputFile),
		"table_package",
		templateNames,
		data,
		data.PackageName,
	)
}

// mapPrismaTypeToGo converts a Prisma FieldType to Go type string (without pointer prefix)
func mapPrismaTypeToGo(fieldType *parser.FieldType) string {
	if fieldType == nil {
		return "interface{}"
	}

	typeMapping := parser.GetTypeGoMapping()
	if mapped, ok := typeMapping[fieldType.Name]; ok {
		// Remove pointer prefix if exists
		return strings.TrimPrefix(mapped, "*")
	}

	// Default to string for unknown types
	return "string"
}
