package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateModels generates Go structs for each model in the schema
func GenerateModels(schema *parser.Schema, outputDir string) error {
	modelsDir := filepath.Join(outputDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	for _, model := range schema.Models {
		modelFile := filepath.Join(modelsDir, toSnakeCase(model.Name)+".go")
		if err := generateModelFile(modelFile, model, schema); err != nil {
			return fmt.Errorf("failed to generate model %s: %w", model.Name, err)
		}
	}

	return nil
}

// generateModelFile generates the Go file for a model using templates
func generateModelFile(filePath string, model *parser.Model, schema *parser.Schema) error {
	// Determine necessary imports
	imports := determineImports(model, schema)

	// Prepare fields
	fields := make([]FieldInfo, 0)
	for _, field := range model.Fields {
		// Skip relations - only include actual database columns
		if isRelation(field, schema) {
			continue
		}

		fieldName := toPascalCase(field.Name)
		goType := fieldTypeToGo(field.Type, field.Attributes)
		jsonTag := toSnakeCase(field.Name)
		dbTag := field.Name

		// Check if it has @map
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					dbTag = val
				}
			}
		}

		fields = append(fields, FieldInfo{
			Name:    fieldName,
			GoType:  goType,
			JSONTag: jsonTag,
			DBTag:   dbTag,
		})
	}

	// Prepare template data
	data := ModelTemplateData{
		ModelName:  model.Name,
		PascalName: toPascalCase(model.Name),
		Imports:    imports,
		Fields:     fields,
	}

	// Generate model file using template
	return executeModelTemplate(filePath, "models", "models", "model.tmpl", data)
}

// fieldTypeToGo converts a Prisma FieldType to Go type
func fieldTypeToGo(fieldType *parser.FieldType, attributes []*parser.Attribute) string {
	if fieldType == nil {
		return "interface{}"
	}

	if fieldType.IsUnsupported {
		return "string" // Unsupported becomes string by default
	}

	// Check if it's an enum or model (relationship)
	// For now, we assume non-primitive types are strings or relationships
	typeMapping := parser.GetTypeGoMapping()
	nullableMapping := parser.GetTypeGoMappingNullable()

	var goType string
	if mapped, ok := typeMapping[fieldType.Name]; ok {
		// Primitive type
		if fieldType.IsOptional {
			if nullable, ok := nullableMapping[fieldType.Name]; ok {
				goType = nullable
			} else {
				goType = "*" + mapped
			}
		} else {
			goType = mapped
		}
	} else {
		// May be enum or model - for now string
		if fieldType.IsOptional {
			goType = "*string"
		} else {
			goType = "string"
		}
	}

	// If it's an array
	if fieldType.IsArray {
		return "[]" + strings.TrimPrefix(goType, "*")
	}

	return goType
}

// determineImports determines which imports are needed
func determineImports(model *parser.Model, schema *parser.Schema) []string {
	imports := make(map[string]bool)

	for _, field := range model.Fields {
		// Skip relations - only check actual database columns
		if isRelation(field, schema) {
			continue
		}

		if field.Type == nil {
			continue
		}

		typeMapping := parser.GetTypeGoMapping()
		if mapped, ok := typeMapping[field.Type.Name]; ok {
			switch mapped {
			case "time.Time":
				imports["time"] = true

			case "json.RawMessage":
				imports["encoding/json"] = true
			}
		}
	}

	// Convert map to ordered slice
	result := make([]string, 0, len(imports))
	if imports["time"] {
		result = append(result, "time")
	}
	if imports["encoding/json"] {
		result = append(result, "encoding/json")
	}

	return result
}

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
		}
	}
	return result.String()
}

// toSnakeCase converts PascalCase to snake_case
// Handles acronyms correctly (e.g., "ISBN" -> "isbn", "XMLHttp" -> "xml_http")
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if previous character was lowercase or if this is part of an acronym
			prev := s[i-1]
			// If previous was lowercase, add underscore (e.g., "HttpRequest" -> "http_request")
			// If previous was uppercase and next is lowercase, add underscore (e.g., "XMLHttp" -> "xml_http")
			if prev >= 'a' && prev <= 'z' {
				result.WriteByte('_')
			} else if i < len(s)-1 {
				next := s[i+1]
				// If next is lowercase, this is end of acronym (e.g., "XMLHttp" -> "xml_http")
				if next >= 'a' && next <= 'z' {
					result.WriteByte('_')
				}
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isStdlibImport checks if an import is from the standard library
func isStdlibImport(imp string) bool {
	stdlibPackages := map[string]bool{
		"context":       true,
		"reflect":       true,
		"time":          true,
		"encoding/json": true,
		"fmt":           true,
		"os":            true,
		"strings":       true,
		"sort":          true,
		"path/filepath": true,
	}

	return stdlibPackages[imp]
}
