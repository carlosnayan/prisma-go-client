package parser

import (
	"testing"
)

func TestParseIndexWithType(t *testing.T) {
	input := `
model book_categories {
  id String @id
  type String
  @@index([type], map: "idx_book_categories_type")
}
`
	lexer := NewLexer(input)
	p := NewParser(lexer)
	schema := p.ParseSchema()

	if len(schema.Models) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(schema.Models))
	}

	model := schema.Models[0]
	if len(model.Attributes) != 1 {
		t.Fatalf("Expected 1 attribute, got %d", len(model.Attributes))
	}

	attr := model.Attributes[0]
	if attr.Name != "index" {
		t.Errorf("Expected attribute name 'index', got '%s'", attr.Name)
	}

	// Verify arguments
	// Arguemnts[0] should be fields: [type] (unnamed or named fields)
	// Arguments[1] should be map: "idx_book_categories_type"

	fieldsFound := false
	for _, arg := range attr.Arguments {
		if arg.Name == "" || arg.Name == "fields" {
			if fields, ok := arg.Value.([]interface{}); ok {
				if len(fields) == 1 {
					if fieldName, ok := fields[0].(string); ok {
						if fieldName == "type" {
							fieldsFound = true
						}
					}
				}
			}
		}
	}

	if !fieldsFound {
		t.Error("Failed to parse field 'type' inside @@index")
	}
}
