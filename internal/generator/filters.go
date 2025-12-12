package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func GenerateFilters(schema *parser.Schema, outputDir string) error {
	filtersDir := filepath.Join(outputDir, "filters")
	if err := os.MkdirAll(filtersDir, 0755); err != nil {
		return fmt.Errorf("failed to create filters directory: %w", err)
	}

	if err := generateFiltersTypesFile(filtersDir, schema); err != nil {
		return fmt.Errorf("failed to generate filter types: %w", err)
	}

	if err := generateFiltersHelpersFile(filtersDir, schema); err != nil {
		return fmt.Errorf("failed to generate filter helpers: %w", err)
	}

	return nil
}

func generateFiltersTypesFile(filtersDir string, schema *parser.Schema) error {
	filtersFile := filepath.Join(filtersDir, "filters.go")

	neededFilters := make(map[string]bool)
	for _, model := range schema.Models {
		for _, field := range model.Fields {
			if !isRelation(field, schema) && field.Type != nil {
				filterType := getFilterType(field.Type)
				neededFilters[filterType] = true
			}
		}
	}

	stdlibImports := make([]string, 0, 2)
	if neededFilters["DateTimeFilter"] {
		stdlibImports = append(stdlibImports, "time")
	}
	if neededFilters["JsonFilter"] {
		stdlibImports = append(stdlibImports, "encoding/json")
	}

	data := FiltersTemplateData{
		StdlibImports: stdlibImports,
		NeededFilters: neededFilters,
	}

	templateNames := []string{"filters_imports.tmpl"}

	if neededFilters["StringFilter"] {
		templateNames = append(templateNames, "string_filter.tmpl")
	}
	if neededFilters["IntFilter"] {
		templateNames = append(templateNames, "int_filter.tmpl")
	}
	if neededFilters["Int64Filter"] {
		templateNames = append(templateNames, "int64_filter.tmpl")
	}
	if neededFilters["FloatFilter"] {
		templateNames = append(templateNames, "float_filter.tmpl")
	}
	if neededFilters["BooleanFilter"] {
		templateNames = append(templateNames, "boolean_filter.tmpl")
	}
	if neededFilters["DateTimeFilter"] {
		templateNames = append(templateNames, "datetime_filter.tmpl")
	}
	if neededFilters["JsonFilter"] {
		templateNames = append(templateNames, "json_filter.tmpl")
	}
	if neededFilters["BytesFilter"] {
		templateNames = append(templateNames, "bytes_filter.tmpl")
	}

	return executeFiltersTemplates(filtersFile, templateNames, data)
}

func generateFiltersHelpersFile(filtersDir string, schema *parser.Schema) error {
	helpersFile := filepath.Join(filtersDir, "helpers.go")

	neededFilters := determineNeededFilters(schema)

	imports := make([]string, 0, 3)
	if neededFilters["DateTimeFilter"] {
		imports = append(imports, "time")
	}
	if neededFilters["JsonFilter"] {
		imports = append(imports, "encoding/json")
	}

	data := HelpersTemplateData{
		Imports:       imports,
		InputsPath:    "",
		NeededFilters: neededFilters,
	}

	templateNames := []string{"imports.tmpl"}

	if neededFilters["StringFilter"] {
		templateNames = append(templateNames, "string.tmpl")
	}
	if neededFilters["IntFilter"] {
		templateNames = append(templateNames, "int.tmpl")
	}
	if neededFilters["Int64Filter"] {
		templateNames = append(templateNames, "int64.tmpl")
	}
	if neededFilters["FloatFilter"] {
		templateNames = append(templateNames, "float.tmpl")
	}
	if neededFilters["BooleanFilter"] {
		templateNames = append(templateNames, "boolean.tmpl")
	}
	if neededFilters["DateTimeFilter"] {
		templateNames = append(templateNames, "datetime.tmpl")
	}
	if neededFilters["JsonFilter"] {
		templateNames = append(templateNames, "json.tmpl")
	}
	if neededFilters["BytesFilter"] {
		templateNames = append(templateNames, "bytes.tmpl")
	}

	return executeFiltersHelpersTemplates(helpersFile, templateNames, data)
}

func determineNeededFilters(schema *parser.Schema) map[string]bool {
	neededFilters := make(map[string]bool)

	for _, model := range schema.Models {
		for _, field := range model.Fields {
			if field.Type == nil {
				continue
			}

			if isRelation(field, schema) {
				continue
			}

			filterType := getFilterType(field.Type)
			neededFilters[filterType] = true
		}
	}

	return neededFilters
}
