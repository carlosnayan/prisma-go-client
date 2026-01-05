package generator

import "github.com/carlosnayan/prisma-go-client/internal/parser"

// fieldTypeToNullableWrapper converts a Prisma field type to the corresponding nullable wrapper type
// This is used for optional fields in CreateInput and all fields in UpdateInput
func fieldTypeToNullableWrapper(fieldType *parser.FieldType) string {
	if fieldType == nil {
		return "interface{}"
	}

	if fieldType.IsUnsupported {
		return "NullableString"
	}

	// Array types are not wrapped in nullable wrappers
	// They use slices directly which can be nil
	if fieldType.IsArray {
		baseType := fieldTypeToGoBase(fieldType)
		return baseType
	}

	// Map field type to nullable wrapper
	switch fieldType.Name {
	case "String":
		return "NullableString"
	case "Int":
		return "NullableInt"
	case "BigInt":
		return "NullableInt64"
	case "Float", "Decimal":
		return "NullableFloat"
	case "Boolean":
		return "NullableBool"
	case "DateTime":
		return "NullableDateTime"
	case "Json":
		return "NullableJson"
	case "Bytes":
		return "NullableBytes"
	default:
		// For unknown types (enums, etc.), use string
		return "NullableString"
	}
}
