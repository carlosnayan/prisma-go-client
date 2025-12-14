package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func TestGenerateEnums_CreatesEnumFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prisma-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	schema := &parser.Schema{
		Enums: []*parser.Enum{
			{
				Name: "BookStatus",
				Values: []*parser.EnumValue{
					{Name: "DRAFT"},
					{Name: "PUBLISHED"},
				},
			},
			{
				Name: "Rating",
				Values: []*parser.EnumValue{
					{Name: "ONE"},
					{Name: "TWO"},
				},
			},
		},
	}

	err = GenerateEnums(schema, tmpDir)
	if err != nil {
		t.Fatalf("GenerateEnums failed: %v", err)
	}

	enumsDir := filepath.Join(tmpDir, "enums")
	if _, err := os.Stat(enumsDir); os.IsNotExist(err) {
		t.Fatal("enums directory was not created")
	}

	expectedFiles := []string{"book_status.go", "rating.go"}
	for _, file := range expectedFiles {
		filePath := filepath.Join(enumsDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("expected file %s was not created", file)
		}
	}
}

func TestGenerateEnums_CorrectContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prisma-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	schema := &parser.Schema{
		Enums: []*parser.Enum{
			{
				Name: "BookStatus",
				Values: []*parser.EnumValue{
					{Name: "DRAFT"},
					{Name: "PUBLISHED"},
				},
			},
		},
	}

	err = GenerateEnums(schema, tmpDir)
	if err != nil {
		t.Fatalf("GenerateEnums failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "enums", "book_status.go"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	contentStr := string(content)

	requiredStrings := []string{
		"package enums",
		"type BookStatus string",
		"BookStatusDraft BookStatus = \"DRAFT\"",
		"BookStatusPublished BookStatus = \"PUBLISHED\"",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(contentStr, required) {
			t.Errorf("generated file missing required content: %q", required)
		}
	}
}

func TestGenerateEnums_CorrectNaming(t *testing.T) {
	tests := []struct {
		name           string
		enumName       string
		expectedFile   string
		expectedType   string
		valueNames     []string
		expectedConsts []string
	}{
		{
			name:           "PascalCase",
			enumName:       "NodeType",
			expectedFile:   "node_type.go",
			expectedType:   "type NodeType string",
			valueNames:     []string{"start", "question"},
			expectedConsts: []string{"NodeTypeStart", "NodeTypeQuestion"},
		},
		{
			name:           "AlreadySnakeCase",
			enumName:       "node_type",
			expectedFile:   "node_type.go",
			expectedType:   "type node_type string",
			valueNames:     []string{"start"},
			expectedConsts: []string{"node_typeStart"},
		},
		{
			name:           "CamelCase",
			enumName:       "nodeType",
			expectedFile:   "node_type.go",
			expectedType:   "type nodeType string",
			valueNames:     []string{"start"},
			expectedConsts: []string{"nodeTypeStart"},
		},
		{
			name:           "MultiWord",
			enumName:       "EditionType",
			expectedFile:   "edition_type.go",
			expectedType:   "type EditionType string",
			valueNames:     []string{"HARDCOVER", "LIMITED_EDITION"},
			expectedConsts: []string{"EditionTypeHardcover", "EditionTypeLimitedEdition"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "prisma-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			values := make([]*parser.EnumValue, 0, len(tt.valueNames))
			for _, v := range tt.valueNames {
				values = append(values, &parser.EnumValue{Name: v})
			}

			schema := &parser.Schema{
				Enums: []*parser.Enum{
					{
						Name:   tt.enumName,
						Values: values,
					},
				},
			}

			err = GenerateEnums(schema, tmpDir)
			if err != nil {
				t.Fatalf("GenerateEnums failed: %v", err)
			}

			filePath := filepath.Join(tmpDir, "enums", tt.expectedFile)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Fatalf("expected file %s was not created", tt.expectedFile)
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read generated file: %v", err)
			}

			contentStr := string(content)

			if !strings.Contains(contentStr, tt.expectedType) {
				t.Errorf("generated file missing expected type: %q", tt.expectedType)
			}

			for _, constName := range tt.expectedConsts {
				if !strings.Contains(contentStr, constName) {
					t.Errorf("generated file missing expected const: %q", constName)
				}
			}
		})
	}
}

func TestGenerateEnums_BookstoreSchema(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prisma-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	schema := &parser.Schema{
		Enums: []*parser.Enum{
			{
				Name: "BookStatus",
				Values: []*parser.EnumValue{
					{Name: "DRAFT"},
					{Name: "PUBLISHED"},
					{Name: "ARCHIVED"},
					{Name: "OUT_OF_PRINT"},
				},
			},
			{
				Name: "RatingValue",
				Values: []*parser.EnumValue{
					{Name: "ONE"},
					{Name: "TWO"},
					{Name: "THREE"},
					{Name: "FOUR"},
					{Name: "FIVE"},
				},
			},
			{
				Name: "EditionType",
				Values: []*parser.EnumValue{
					{Name: "HARDCOVER"},
					{Name: "PAPERBACK"},
					{Name: "EBOOK"},
					{Name: "AUDIOBOOK"},
					{Name: "LIMITED_EDITION"},
				},
			},
		},
	}

	err = GenerateEnums(schema, tmpDir)
	if err != nil {
		t.Fatalf("GenerateEnums failed: %v", err)
	}

	expectedEnums := map[string][]string{
		"book_status.go": {
			"type BookStatus string",
			"BookStatusDraft",
			"BookStatusPublished",
			"BookStatusArchived",
			"BookStatusOutOfPrint",
		},
		"rating_value.go": {
			"type RatingValue string",
			"RatingValueOne",
			"RatingValueFive",
		},
		"edition_type.go": {
			"type EditionType string",
			"EditionTypeHardcover",
			"EditionTypeLimitedEdition",
		},
	}

	for file, expectedContent := range expectedEnums {
		filePath := filepath.Join(tmpDir, "enums", file)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read %s: %v", file, err)
			continue
		}

		contentStr := string(content)
		for _, expected := range expectedContent {
			if !strings.Contains(contentStr, expected) {
				t.Errorf("%s missing expected content: %q", file, expected)
			}
		}
	}
}

func TestGenerateEnums_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		enumName       string
		valueNames     []string
		expectedConsts []string
	}{
		{
			name:           "SingleLetterValue",
			enumName:       "Size",
			valueNames:     []string{"S", "M", "L"},
			expectedConsts: []string{"SizeS", "SizeM", "SizeL"},
		},
		{
			name:           "NumberInValue",
			enumName:       "HTTP",
			valueNames:     []string{"HTTP_200", "HTTP_404"},
			expectedConsts: []string{"HTTPHttp200", "HTTPHttp404"},
		},
		{
			name:           "LowercaseValue",
			enumName:       "Status",
			valueNames:     []string{"active", "inactive"},
			expectedConsts: []string{"StatusActive", "StatusInactive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "prisma-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			values := make([]*parser.EnumValue, 0, len(tt.valueNames))
			for _, v := range tt.valueNames {
				values = append(values, &parser.EnumValue{Name: v})
			}

			schema := &parser.Schema{
				Enums: []*parser.Enum{
					{
						Name:   tt.enumName,
						Values: values,
					},
				},
			}

			err = GenerateEnums(schema, tmpDir)
			if err != nil {
				t.Fatalf("GenerateEnums failed: %v", err)
			}

			files, err := os.ReadDir(filepath.Join(tmpDir, "enums"))
			if err != nil {
				t.Fatalf("failed to read enums dir: %v", err)
			}

			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}

			content, err := os.ReadFile(filepath.Join(tmpDir, "enums", files[0].Name()))
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			contentStr := string(content)
			for _, constName := range tt.expectedConsts {
				if !strings.Contains(contentStr, constName) {
					t.Errorf("missing expected const: %q", constName)
				}
			}
		})
	}
}
