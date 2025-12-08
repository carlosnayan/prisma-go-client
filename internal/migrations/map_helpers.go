package migrations

import (
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// getTableNameFromModel returns the actual table name considering @@map attribute
func getTableNameFromModel(model *parser.Model) string {
	for _, attr := range model.Attributes {
		if attr.Name == "map" && len(attr.Arguments) > 0 {
			if name, ok := attr.Arguments[0].Value.(string); ok {
				return strings.Trim(name, `"`)
			}
		}
	}
	return model.Name
}

// getColumnNameFromField returns the actual column name considering @map attribute
func getColumnNameFromField(field *parser.ModelField) string {
	for _, attr := range field.Attributes {
		if attr.Name == "map" && len(attr.Arguments) > 0 {
			if name, ok := attr.Arguments[0].Value.(string); ok {
				return strings.Trim(name, `"`)
			}
		}
	}
	return field.Name
}

// getNumericValue extracts a numeric value as string from an attribute argument
// Handles both string and numeric types
func getNumericValue(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	// Try to convert numeric types to string
	switch v := value.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	}
	return ""
}
