package formatter

import (
	"fmt"
	"sort"
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
		result.WriteString(formatModel(model))
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
	var result strings.Builder
	result.WriteString("model ")
	result.WriteString(model.Name)
	result.WriteString(" {\n")

	// Ordenar campos
	fields := make([]*parser.ModelField, len(model.Fields))
	copy(fields, model.Fields)
	sort.Slice(fields, func(i, j int) bool {
		// Campos com @id primeiro
		hasID1 := hasAttribute(fields[i].Attributes, "id")
		hasID2 := hasAttribute(fields[j].Attributes, "id")
		if hasID1 != hasID2 {
			return hasID1
		}
		return fields[i].Name < fields[j].Name
	})

	// Formatar campos
	for _, field := range fields {
		result.WriteString(formatModelField(field))
	}

	// Atributos de model (@@id, @@unique, etc.)
	if len(model.Attributes) > 0 {
		result.WriteString("\n")
		for _, attr := range model.Attributes {
			result.WriteString(formatAttribute(attr, 2))
		}
	}

	result.WriteString("}\n")
	return result.String()
}

func formatModelField(field *parser.ModelField) string {
	var result strings.Builder
	result.WriteString("  ")
	result.WriteString(field.Name)
	result.WriteString("  ")

	// Tipo
	if field.Type != nil {
		if field.Type.IsArray {
			result.WriteString(field.Type.Name)
			result.WriteString("[]")
		} else {
			result.WriteString(field.Type.Name)
		}
		if field.Type.IsOptional {
			result.WriteString("?")
		}
	}

	// Atributos
	if len(field.Attributes) > 0 {
		for _, attr := range field.Attributes {
			result.WriteString(" ")
			result.WriteString(formatFieldAttribute(attr))
		}
	}

	result.WriteString("\n")
	return result.String()
}

func formatFieldAttribute(attr *parser.Attribute) string {
	var result strings.Builder
	result.WriteString("@")
	result.WriteString(attr.Name)

	if len(attr.Arguments) > 0 {
		result.WriteString("(")
		args := make([]string, 0, len(attr.Arguments))
		for _, arg := range attr.Arguments {
			if arg.Name != "" {
				args = append(args, fmt.Sprintf("%s: %s", arg.Name, formatAttributeArgument(arg)))
			} else {
				args = append(args, formatAttributeArgument(arg))
			}
		}
		result.WriteString(strings.Join(args, ", "))
		result.WriteString(")")
	}

	return result.String()
}

func formatAttribute(attr *parser.Attribute, indent int) string {
	var result strings.Builder
	indentStr := strings.Repeat("  ", indent)
	result.WriteString(indentStr)
	result.WriteString("@@")
	result.WriteString(attr.Name)

	if len(attr.Arguments) > 0 {
		result.WriteString("(")
		args := make([]string, 0, len(attr.Arguments))
		for _, arg := range attr.Arguments {
			if arg.Name != "" {
				args = append(args, fmt.Sprintf("%s: %s", arg.Name, formatAttributeArgument(arg)))
			} else {
				args = append(args, formatAttributeArgument(arg))
			}
		}
		result.WriteString(strings.Join(args, ", "))
		result.WriteString(")")
	}

	result.WriteString("\n")
	return result.String()
}

func formatAttributeArgument(arg *parser.AttributeArgument) string {
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
				argStrs = append(argStrs, formatValueWithDepth(arg, 0, make(map[interface{}]bool)))
			}
			return fmt.Sprintf("%s(%s)", function, strings.Join(argStrs, ", "))
		}
	}

	return formatValueWithDepth(arg.Value, 0, make(map[interface{}]bool))
}

// formatValueWithDepth formats a value with protection against infinite recursion
func formatValueWithDepth(v interface{}, depth int, visited map[interface{}]bool) string {
	if depth > 10 {
		return "..."
	}

	if v == nil {
		return "null"
	}

	if visited[v] {
		return "..."
	}

	switch val := v.(type) {
	case string:
		// Se já contém parênteses, provavelmente é uma função
		if strings.Contains(val, "()") {
			return val
		}
		return fmt.Sprintf("%q", val)
	case int, int32, int64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case []interface{}:
		// Array
		parts := make([]string, 0, len(val))
		for _, item := range val {
			parts = append(parts, formatValueWithDepth(item, depth+1, visited))
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	case map[string]interface{}:
		// Map - formatar como objeto
		// Marcar como visitado antes de processar
		visited[val] = true
		defer func() {
			delete(visited, val)
		}()

		// Formatação especial para funções do Prisma (estrutura criada pelo parser)
		if function, ok := val["function"].(string); ok {
			args, _ := val["args"].([]interface{})
			if len(args) == 0 {
				return fmt.Sprintf("%s()", function)
			}
			argStrs := make([]string, 0, len(args))
			for _, arg := range args {
				argStrs = append(argStrs, formatValueWithDepth(arg, depth+1, visited))
			}
			return fmt.Sprintf("%s(%s)", function, strings.Join(argStrs, ", "))
		}

		// Formatação genérica para outros maps
		parts := make([]string, 0, len(val))
		for k, v := range val {
			formattedValue := formatValueWithDepth(v, depth+1, visited)
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

func hasAttribute(attrs []*parser.Attribute, name string) bool {
	for _, attr := range attrs {
		if attr.Name == name {
			return true
		}
	}
	return false
}
