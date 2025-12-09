package formatter

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// FormatSchema formata um schema Prisma parseado
func FormatSchema(schema *parser.Schema) string {
	var result strings.Builder

	// Ordenar elementos
	datasources := schema.Datasources
	generators := schema.Generators
	models := schema.Models
	enums := schema.Enums

	sort.Slice(datasources, func(i, j int) bool {
		return datasources[i].Name < datasources[j].Name
	})
	sort.Slice(generators, func(i, j int) bool {
		return generators[i].Name < generators[j].Name
	})
	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})
	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Name < enums[j].Name
	})

	// Formatar datasources
	for _, ds := range datasources {
		result.WriteString(formatDatasource(ds))
		result.WriteString("\n")
	}

	// Formatar generators
	for _, gen := range generators {
		result.WriteString(formatGenerator(gen))
		result.WriteString("\n")
	}

	// Formatar models
	for _, model := range models {
		result.WriteString(formatModelWithSchema(model, schema))
		result.WriteString("\n")
	}

	// Formatar enums
	for _, enum := range enums {
		result.WriteString(formatEnum(enum))
		result.WriteString("\n")
	}

	return strings.TrimSpace(result.String())
}

func formatDatasource(ds *parser.Datasource) string {
	var result strings.Builder
	result.WriteString("datasource ")
	result.WriteString(ds.Name)
	result.WriteString(" {\n")

	// Formatar campos
	for _, field := range ds.Fields {
		result.WriteString(fmt.Sprintf("  %s = %s\n", field.Name, formatFieldValue(field.Value)))
	}

	result.WriteString("}\n")
	return result.String()
}

func formatGenerator(gen *parser.Generator) string {
	var result strings.Builder
	result.WriteString("generator ")
	result.WriteString(gen.Name)
	result.WriteString(" {\n")

	// Formatar campos
	for _, field := range gen.Fields {
		result.WriteString(fmt.Sprintf("  %s = %s\n", field.Name, formatFieldValue(field.Value)))
	}

	result.WriteString("}\n")
	return result.String()
}

func formatModel(model *parser.Model) string {
	return formatModelWithSchema(model, nil)
}

func formatModelWithSchema(model *parser.Model, schema *parser.Schema) string {
	var result strings.Builder
	result.WriteString("model ")
	result.WriteString(model.Name)
	result.WriteString(" {\n")

	// Manter ordem original dos campos (não ordenar)
	fields := model.Fields

	// Calcular larguras máximas para alinhamento
	maxNameLen := 0
	maxTypeLen := 0
	for _, field := range fields {
		// Skip relation fields (they don't need alignment)
		isRelationField := false
		for _, attr := range field.Attributes {
			if attr.Name == "relation" {
				hasFields := false
				for _, arg := range attr.Arguments {
					if arg.Name == "fields" {
						hasFields = true
						break
					}
				}
				if !hasFields {
					isRelationField = true
					break
				}
			}
		}
		if isRelationField {
			continue
		}

		nameLen := len(field.Name)
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}

		typeStr := ""
		if field.Type != nil {
			if field.Type.IsArray {
				typeStr = field.Type.Name + "[]"
			} else {
				typeStr = field.Type.Name
			}
			if field.Type.IsOptional {
				typeStr += "?"
			}
		}
		typeLen := len(typeStr)
		if typeLen > maxTypeLen {
			maxTypeLen = typeLen
		}
	}

	// Formatar campos com alinhamento
	for _, field := range fields {
		result.WriteString(formatModelFieldWithSchemaAligned(field, schema, maxNameLen, maxTypeLen))
	}

	// Atributos de model (@@id, @@unique, etc.)
	if len(model.Attributes) > 0 {
		result.WriteString("\n")
		for _, attr := range model.Attributes {
			result.WriteString("  ")
			result.WriteString(formatAttribute(attr, 0))
		}
	}

	result.WriteString("}\n")
	return result.String()
}

func formatModelFieldWithSchemaAligned(field *parser.ModelField, schema *parser.Schema, maxNameLen, maxTypeLen int) string {
	var result strings.Builder
	result.WriteString("  ")
	result.WriteString(field.Name)

	// Check if this is a relation field (array field without fields/references)
	isRelationField := false
	for _, attr := range field.Attributes {
		if attr.Name == "relation" {
			hasFields := false
			for _, arg := range attr.Arguments {
				if arg.Name == "fields" {
					hasFields = true
					break
				}
			}
			if !hasFields {
				isRelationField = true
				break
			}
		}
	}

	// Align field name if maxNameLen > 0 (formatting with alignment)
	if maxNameLen > 0 && !isRelationField {
		padding := maxNameLen - len(field.Name)
		result.WriteString(strings.Repeat(" ", padding))
		result.WriteString(" ")
	} else {
		result.WriteString("  ")
	}

	// Tipo
	typeStr := ""
	if field.Type != nil {
		if field.Type.IsArray {
			typeStr = field.Type.Name + "[]"
		} else {
			typeStr = field.Type.Name
		}
		if field.Type.IsOptional {
			typeStr += "?"
		}
	}
	result.WriteString(typeStr)

	// Align type if maxTypeLen > 0 (formatting with alignment)
	if maxTypeLen > 0 && !isRelationField {
		padding := maxTypeLen - len(typeStr)
		if len(field.Attributes) > 0 {
			// Only add padding if there are attributes
			result.WriteString(strings.Repeat(" ", padding))
		}
	} else if len(field.Attributes) > 0 {
		result.WriteString(" ")
	}

	// Atributos
	if len(field.Attributes) > 0 {
		for _, attr := range field.Attributes {
			result.WriteString(" ")
			result.WriteString(formatFieldAttributeWithTypeAndSchema(attr, field.Type, schema))
		}
	}

	result.WriteString("\n")
	return result.String()
}

func formatFieldAttributeWithTypeAndSchema(attr *parser.Attribute, fieldType *parser.FieldType, schema *parser.Schema) string {
	var result strings.Builder
	result.WriteString("@")
	result.WriteString(attr.Name)

	if len(attr.Arguments) > 0 {
		result.WriteString("(")
		args := make([]string, 0, len(attr.Arguments))
		for _, arg := range attr.Arguments {
			if arg.Name != "" {
				args = append(args, fmt.Sprintf("%s: %s", arg.Name, formatAttributeArgumentWithTypeAndSchema(arg, attr.Name, fieldType, schema)))
			} else {
				args = append(args, formatAttributeArgumentWithTypeAndSchema(arg, attr.Name, fieldType, schema))
			}
		}
		result.WriteString(strings.Join(args, ", "))
		result.WriteString(")")
	}

	return result.String()
}

func formatAttribute(attr *parser.Attribute, indent int) string {
	var result strings.Builder
	if indent > 0 {
		indentStr := strings.Repeat("  ", indent)
		result.WriteString(indentStr)
	}
	result.WriteString("@@")
	result.WriteString(attr.Name)

	if len(attr.Arguments) > 0 {
		result.WriteString("(")
		args := make([]string, 0, len(attr.Arguments))
		// Check if this is an index/unique attribute
		isIndexAttr := attr.Name == "index" || attr.Name == "unique"
		for _, arg := range attr.Arguments {
			if arg.Name != "" {
				formattedArg := formatAttributeArgument(arg, attr.Name)
				// For index/unique attributes, remove quotes from field names in lists
				if isIndexAttr && arg.Name == "" {
					// This is the fields argument (list of column names)
					if listVal, ok := arg.Value.([]interface{}); ok {
						formattedArg = formatIndexList(listVal)
					}
				}
				args = append(args, fmt.Sprintf("%s: %s", arg.Name, formattedArg))
			} else {
				formattedArg := formatAttributeArgument(arg, attr.Name)
				// For index/unique attributes, remove quotes from field names in lists
				if isIndexAttr {
					if listVal, ok := arg.Value.([]interface{}); ok {
						formattedArg = formatIndexList(listVal)
					}
				}
				args = append(args, formattedArg)
			}
		}
		result.WriteString(strings.Join(args, ", "))
		result.WriteString(")")
	}

	result.WriteString("\n")
	return result.String()
}

// formatIndexList formats a list of index columns, removing quotes from field names
func formatIndexList(listVal []interface{}) string {
	parts := make([]string, 0, len(listVal))
	for _, item := range listVal {
		if str, ok := item.(string); ok {
			// Remove quotes if present (from parser re-parsing)
			cleanStr := str
			if strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"") && len(str) > 2 {
				cleanStr = str[1 : len(str)-1]
			}
			// Only add without quotes if it's a valid identifier
			if isValidIdentifier(cleanStr) {
				parts = append(parts, cleanStr)
			} else {
				parts = append(parts, str)
			}
		} else if m, ok := item.(map[string]interface{}); ok {
			// Handle function calls like created_at(sort: Desc)
			if function, isFunction := m["function"].(string); isFunction {
				args, _ := m["args"].([]interface{})
				if len(args) == 0 {
					parts = append(parts, fmt.Sprintf("%s()", function))
				} else {
					argStrs := make([]string, 0, len(args))
					for _, arg := range args {
						if argMap, ok := arg.(map[string]interface{}); ok {
							if argName, hasArgName := argMap["name"].(string); hasArgName {
								argValueRaw := argMap["value"]
								var argValue string
								if strVal, ok := argValueRaw.(string); ok {
									if strings.HasPrefix(strVal, "\"") && strings.HasSuffix(strVal, "\"") && len(strVal) > 2 {
										strVal = strVal[1 : len(strVal)-1]
									}
									if isValidIdentifier(strVal) {
										argValue = strVal
									} else {
										argValue = fmt.Sprintf("%q", strVal)
									}
								} else {
									argValue = fmt.Sprintf("%v", argValueRaw)
								}
								argStrs = append(argStrs, fmt.Sprintf("%s: %s", argName, argValue))
							}
						}
					}
					parts = append(parts, fmt.Sprintf("%s(%s)", function, strings.Join(argStrs, ", ")))
				}
			} else if name, hasName := m["name"].(string); hasName {
				// Handle {name: "col", sort: "Desc"}
				if sort, hasSort := m["sort"].(string); hasSort {
					parts = append(parts, fmt.Sprintf("%s(sort: %s)", name, sort))
				} else {
					parts = append(parts, name)
				}
			} else {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
		} else {
			parts = append(parts, fmt.Sprintf("%v", item))
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func formatAttributeArgument(arg *parser.AttributeArgument, attrName string) string {
	return formatAttributeArgumentWithType(arg, attrName, nil)
}

func formatAttributeArgumentWithType(arg *parser.AttributeArgument, attrName string, fieldType *parser.FieldType) string {
	return formatAttributeArgumentWithTypeAndSchema(arg, attrName, fieldType, nil)
}

func formatAttributeArgumentWithTypeAndSchema(arg *parser.AttributeArgument, attrName string, fieldType *parser.FieldType, schema *parser.Schema) string {
	if arg == nil {
		return ""
	}

	if m, ok := arg.Value.(map[string]interface{}); ok {
		if function, ok := m["function"].(string); ok {
			args, _ := m["args"].([]interface{})
			if len(args) == 0 {
				return fmt.Sprintf("%s()", function)
			}
			argStrs := make([]string, 0, len(args))
			for _, arg := range args {
				argStrs = append(argStrs, formatValueWithDepth(arg, 0, make(map[interface{}]bool), attrName, fieldType, schema))
			}
			return fmt.Sprintf("%s(%s)", function, strings.Join(argStrs, ", "))
		}
	}

	// For @db.* attributes, pass the attribute name to formatValueWithDepth
	// so numeric values are not quoted
	val := formatValueWithDepth(arg.Value, 0, make(map[interface{}]bool), attrName, fieldType, schema)

	if arg.Name == "onDelete" || arg.Name == "onUpdate" {
		if str, ok := arg.Value.(string); ok {
			switch str {
			case "Cascade", "SetNull", "Restrict", "NoAction":
				return str
			}
		}
	}

	return val
}

// formatValueWithDepth formats a value with protection against infinite recursion
func formatValueWithDepth(v interface{}, depth int, visited map[interface{}]bool, attrName string, fieldType *parser.FieldType, schema *parser.Schema) string {
	if depth > 10 {
		return "..."
	}

	if v == nil {
		return "null"
	}

	// Only check visited for hashable types
	if isHashable(v) {
		if visited[v] {
			return "..."
		}
		// Mark as visited before processing
		visited[v] = true
		defer func() {
			delete(visited, v)
		}()
	}

	switch val := v.(type) {
	case string:
		// For @default attribute, check field type to determine if value should be unquoted
		if attrName == "default" && fieldType != nil {
			// Check if field type is Json - format JSON correctly
			if fieldType.Name == "Json" {
				// Escape internal quotes for Prisma format
				escaped := strings.ReplaceAll(val, "\"", "\\\"")
				return fmt.Sprintf("\"%s\"", escaped)
			}
			// Check if field type is Boolean
			if fieldType.Name == "Boolean" {
				if val == "true" || val == "\"true\"" {
					return "true"
				}
				if val == "false" || val == "\"false\"" {
					return "false"
				}
			}
			// Check if field type is Int or BigInt
			if fieldType.Name == "Int" || fieldType.Name == "BigInt" {
				// Remove quotes if present
				cleanVal := val
				if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") && len(val) > 2 {
					cleanVal = val[1 : len(val)-1]
				}
				// Try to parse as integer
				if intVal, err := strconv.Atoi(cleanVal); err == nil {
					return fmt.Sprintf("%d", intVal)
				}
			}
			// Check if field type is Float
			if fieldType.Name == "Float" {
				// Remove quotes if present
				cleanVal := val
				if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") && len(val) > 2 {
					cleanVal = val[1 : len(val)-1]
				}
				// Try to parse as float
				if floatVal, err := strconv.ParseFloat(cleanVal, 64); err == nil {
					return fmt.Sprintf("%g", floatVal)
				}
			}
		}
		// For @db.* attributes, check if string is a numeric value and return without quotes
		if attrName != "" && strings.HasPrefix(attrName, "db.") {
			if intVal, err := strconv.Atoi(val); err == nil {
				return fmt.Sprintf("%d", intVal)
			}
			if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
				return fmt.Sprintf("%g", floatVal)
			}
		}
		if val == "{}" {
			// For @default attribute, return "{}" with quotes
			if attrName == "default" {
				return "\"{}\""
			}
			return "\"{}\""
		}
		// Remove quotes if present (might come from convertDefaultValue in some edge cases)
		cleanVal := val
		if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") && len(val) > 2 {
			cleanVal = val[1 : len(val)-1]
		}
		// For @default attribute, check if it's an enum value
		if attrName == "default" && fieldType != nil && schema != nil {
			// First, try to match by field type name (most accurate)
			for _, enum := range schema.Enums {
				// Compare enum name with field type name (case-insensitive to handle any case mismatches)
				if strings.EqualFold(enum.Name, fieldType.Name) {
					// Check if the value matches an enum value (case-insensitive comparison)
					for _, enumVal := range enum.Values {
						if strings.EqualFold(enumVal.Name, cleanVal) {
							// It's an enum value, return without quotes (use original enum value name for consistency)
							return enumVal.Name
						}
					}
				}
			}
			// Fallback: check if the value matches any enum value in the schema
			// This handles cases where the enum name might not match exactly
			// This is safe because enum values are typically unique across all enums
			for _, enum := range schema.Enums {
				for _, enumVal := range enum.Values {
					if strings.EqualFold(enumVal.Name, cleanVal) {
						// It's an enum value, return without quotes (use original enum value name for consistency)
						return enumVal.Name
					}
				}
			}
		}
		// Handle empty string specially - return "" (with quotes)
		if val == "" {
			return "\"\""
		}
		// If we got here and the value had quotes, return it as-is
		if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") && len(val) > 2 {
			return val
		}
		if strings.Contains(val, "()") && !strings.HasPrefix(val, "\"") {
			return fmt.Sprintf("%q", val)
		}
		if strings.Contains(val, "()") {
			return val
		}
		return fmt.Sprintf("%q", val)
	case int, int32, int64:
		// For @db.* attributes, don't quote numeric values
		if attrName != "" && strings.HasPrefix(attrName, "db.") {
			return fmt.Sprintf("%d", val)
		}
		return fmt.Sprintf("%d", val)
	case float32, float64:
		// For @db.* attributes, don't quote numeric values
		if attrName != "" && strings.HasPrefix(attrName, "db.") {
			return fmt.Sprintf("%g", val)
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case []string:
		parts := append([]string{}, val...)
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	case []interface{}:
		// Special handling for JSON default values that were parsed as slices
		// If all items are strings and it looks like a JSON string that was split, reconstruct it
		if attrName == "default" && fieldType != nil && fieldType.Name == "Json" {
			allStrings := true
			for _, item := range val {
				if _, ok := item.(string); !ok {
					allStrings = false
					break
				}
			}
			if allStrings {
				// Reconstruct JSON string from slice
				jsonParts := make([]string, len(val))
				for i, item := range val {
					jsonParts[i] = item.(string)
				}
				jsonStr := strings.Join(jsonParts, "")
				// Escape quotes for Prisma format
				escaped := strings.ReplaceAll(jsonStr, "\"", "\\\"")
				return fmt.Sprintf("\"%s\"", escaped)
			}
		}
		parts := make([]string, 0, len(val))
		// Check if this is an index/unique list (all items are strings or maps with name/sort)
		isIndexList := true
		for _, item := range val {
			if _, ok := item.(string); ok {
				// String - will be added without quotes for index lists
				continue
			} else if m, ok := item.(map[string]interface{}); ok {
				// Map with name/sort - valid for index lists
				if _, hasName := m["name"].(string); !hasName {
					isIndexList = false
					break
				}
			} else {
				isIndexList = false
				break
			}
		}

		for _, item := range val {
			// Special handling for index column with sort order: {name: "col", sort: "Desc"} -> col(sort: Desc)
			if m, ok := item.(map[string]interface{}); ok {
				if name, hasName := m["name"].(string); hasName {
					if sort, hasSort := m["sort"].(string); hasSort {
						parts = append(parts, fmt.Sprintf("%s(sort: %s)", name, sort))
						continue
					}
				}
				// Check if this is a function call (created_at(sort: Desc))
				if function, isFunction := m["function"].(string); isFunction {
					args, _ := m["args"].([]interface{})
					if len(args) == 0 {
						parts = append(parts, fmt.Sprintf("%s()", function))
						continue
					}
					// Format function with named arguments
					argStrs := make([]string, 0, len(args))
					for _, arg := range args {
						// Check if arg is a map with name (named argument)
						if argMap, ok := arg.(map[string]interface{}); ok {
							if argName, hasArgName := argMap["name"].(string); hasArgName {
								argValueRaw := argMap["value"]
								// For sort order values like "Desc", don't quote them
								var argValue string
								if strVal, ok := argValueRaw.(string); ok {
									// Remove quotes if present
									if strings.HasPrefix(strVal, "\"") && strings.HasSuffix(strVal, "\"") && len(strVal) > 2 {
										strVal = strVal[1 : len(strVal)-1]
									}
									// Check if it's a valid identifier (like "Desc", "Asc")
									if isValidIdentifier(strVal) {
										argValue = strVal
									} else {
										argValue = formatValueWithDepth(argValueRaw, depth+1, visited, attrName, fieldType, schema)
									}
								} else {
									argValue = formatValueWithDepth(argValueRaw, depth+1, visited, attrName, fieldType, schema)
								}
								argStrs = append(argStrs, fmt.Sprintf("%s: %s", argName, argValue))
								continue
							}
						}
						argStrs = append(argStrs, formatValueWithDepth(arg, depth+1, visited, attrName, fieldType, schema))
					}
					parts = append(parts, fmt.Sprintf("%s(%s)", function, strings.Join(argStrs, ", ")))
					continue
				}
			}
			// For index lists, don't quote string field names
			if isIndexList {
				if str, ok := item.(string); ok {
					// Remove quotes if present (from parser re-parsing)
					cleanStr := str
					if strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"") && len(str) > 2 {
						cleanStr = str[1 : len(str)-1]
					}
					parts = append(parts, cleanStr)
					continue
				}
			}
			parts = append(parts, formatValueWithDepth(item, depth+1, visited, attrName, fieldType, schema))
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	case map[string]interface{}:
		// Map - formatar como objeto
		// Note: visited is already handled at the beginning of the function

		// Formatação especial para funções do Prisma (estrutura criada pelo parser)
		if function, ok := val["function"].(string); ok {
			args, _ := val["args"].([]interface{})
			if len(args) == 0 {
				return fmt.Sprintf("%s()", function)
			}
			argStrs := make([]string, 0, len(args))
			for _, arg := range args {
				// Check if arg is a map representing a named argument (from parser)
				// When parser reads "sort: Desc", it may create a map or AttributeArgument
				if argMap, ok := arg.(map[string]interface{}); ok {
					// Check if this is a named argument structure
					if argName, hasArgName := argMap["name"].(string); hasArgName {
						argValueRaw := argMap["value"]
						// For sort order values like "Desc", don't quote them
						var argValue string
						if strVal, ok := argValueRaw.(string); ok {
							// Remove quotes if present
							if strings.HasPrefix(strVal, "\"") && strings.HasSuffix(strVal, "\"") && len(strVal) > 2 {
								strVal = strVal[1 : len(strVal)-1]
							}
							// Check if it's a valid identifier (like "Desc", "Asc")
							if isValidIdentifier(strVal) {
								argValue = strVal
							} else {
								argValue = formatValueWithDepth(argValueRaw, depth+1, visited, attrName, fieldType, schema)
							}
						} else {
							argValue = formatValueWithDepth(argValueRaw, depth+1, visited, attrName, fieldType, schema)
						}
						argStrs = append(argStrs, fmt.Sprintf("%s: %s", argName, argValue))
						continue
					}
				}
				argStrs = append(argStrs, formatValueWithDepth(arg, depth+1, visited, attrName, fieldType, schema))
			}
			return fmt.Sprintf("%s(%s)", function, strings.Join(argStrs, ", "))
		}

		// Formatação genérica para outros maps
		parts := make([]string, 0, len(val))
		for k, v := range val {
			formattedValue := formatValueWithDepth(v, depth+1, visited, attrName, fieldType, schema)
			parts = append(parts, fmt.Sprintf("%s: %s", k, formattedValue))
		}
		return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
	default:
		return fmt.Sprintf("%v", val)
	}
}

func formatFieldValue(value interface{}) string {
	if str, ok := value.(string); ok {
		return fmt.Sprintf("%q", str)
	}
	return fmt.Sprintf("%v", value)
}

// FormatModel formats a single model
func FormatModel(model *parser.Model) string {
	return formatModel(model)
}

// FormatModelWithSchema formats a single model with schema context (for enum detection)
func FormatModelWithSchema(model *parser.Model, schema *parser.Schema) string {
	return formatModelWithSchema(model, schema)
}

// FormatEnum formats a single enum
func FormatEnum(enum *parser.Enum) string {
	return formatEnum(enum)
}

// FormatDatasource formats a single datasource
func FormatDatasource(ds *parser.Datasource) string {
	return formatDatasource(ds)
}

// FormatGenerator formats a single generator
func FormatGenerator(gen *parser.Generator) string {
	return formatGenerator(gen)
}

func formatEnum(enum *parser.Enum) string {
	var result strings.Builder
	result.WriteString("enum ")
	result.WriteString(enum.Name)
	result.WriteString(" {\n")

	// Ordenar valores
	values := make([]*parser.EnumValue, len(enum.Values))
	copy(values, enum.Values)
	sort.Slice(values, func(i, j int) bool {
		return values[i].Name < values[j].Name
	})

	for _, val := range values {
		result.WriteString(fmt.Sprintf("  %s\n", val.Name))
	}

	result.WriteString("}\n")
	return result.String()
}

func isHashable(v interface{}) bool {
	switch v.(type) {
	case map[string]interface{}, map[interface{}]interface{}:
		// Maps are not hashable in Go, cannot be used as map keys
		return false
	case []string, []interface{}:
		// Slices are not hashable
		return false
	default:
		return true
	}
}

// isValidIdentifier checks if a string is a valid Prisma identifier (enum value, etc.)
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	// Check if it contains only alphanumeric characters, underscores, and hyphens
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	// Must start with a letter or underscore
	first := rune(s[0])
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_'
}
