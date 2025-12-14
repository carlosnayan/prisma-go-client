package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func GenerateEnums(schema *parser.Schema, outputDir string) error {
	enumsDir := filepath.Join(outputDir, "enums")
	if err := os.MkdirAll(enumsDir, 0755); err != nil {
		return fmt.Errorf("failed to create enums directory: %w", err)
	}

	for _, enum := range schema.Enums {
		enumFile := filepath.Join(enumsDir, toSnakeCase(enum.Name)+".go")
		if err := generateEnumFile(enumFile, enum); err != nil {
			return fmt.Errorf("failed to generate enum %s: %w", enum.Name, err)
		}
	}

	return nil
}

func generateEnumFile(filePath string, enum *parser.Enum) error {
	pascalName := enum.Name

	values := make([]EnumValueInfo, 0, len(enum.Values))
	for _, value := range enum.Values {
		constName := toConstName(pascalName, value.Name)
		values = append(values, EnumValueInfo{
			ConstName: constName,
			Value:     value.Name,
		})
	}

	data := EnumTemplateData{
		EnumName:   enum.Name,
		PascalName: pascalName,
		Values:     values,
	}

	return executeEnumTemplate(filePath, "enums", "enum.tmpl", data)
}

func toConstName(enumName, value string) string {
	return enumName + toPascalCase(value)
}
