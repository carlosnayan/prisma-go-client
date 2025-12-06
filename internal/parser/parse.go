package parser

import (
	"fmt"
	"os"
)

// ParseFile parseia um arquivo schema.prisma e retorna a AST
func ParseFile(filePath string) (*Schema, []string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	return Parse(string(data))
}

// Parse parseia uma string contendo o schema.prisma e retorna a AST
func Parse(input string) (*Schema, []string, error) {
	lexer := NewLexer(input)
	parser := NewParser(lexer)

	schema := parser.ParseSchema()

	// Combinar erros do parser e do validator
	errors := parser.Errors()
	validationErrors := Validate(schema)
	errors = append(errors, validationErrors...)

	if len(errors) > 0 {
		return schema, errors, fmt.Errorf("erros encontrados durante o parsing")
	}

	return schema, nil, nil
}

// ParseAndValidate parseia e valida um schema, retornando apenas erros se houver
func ParseAndValidate(input string) (*Schema, error) {
	schema, errors, err := Parse(input)
	if err != nil {
		return schema, err
	}

	if len(errors) > 0 {
		return schema, fmt.Errorf("erros de validação:\n%s", formatErrors(errors))
	}

	return schema, nil
}

// formatErrors formata os erros para exibição
func formatErrors(errors []string) string {
	result := ""
	for i, err := range errors {
		result += fmt.Sprintf("  %d. %s\n", i+1, err)
	}
	return result
}
