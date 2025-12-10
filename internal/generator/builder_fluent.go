package generator

import (
	"strings"
)

// generateBuilderFluent generates fluent.go with Query struct using templates
func generateBuilderFluent(builderDir string, provider string, utilsPath string) error {
	// Define the order of templates to execute
	// Note: helpers.tmpl is NOT included here because contains and toSnakeCase
	// are already defined in builder.go (from builder_main/helpers.tmpl)
	templateNames := []string{
		"imports.tmpl",
		"logger.tmpl",
		"query_struct.tmpl",
		"query_constructors.tmpl",
		"query_where.tmpl",
		"query_builder.tmpl",
		"query_execution.tmpl",
		"query_build_helpers.tmpl",
		"query_scan.tmpl",
		"fulltext.tmpl",
		"logging.tmpl",
		"transaction.tmpl",
	}

	// Extract package name from utilsPath (last segment)
	utilsPackageName := "utils"
	if utilsPath != "" {
		parts := strings.Split(utilsPath, "/")
		if len(parts) > 0 {
			utilsPackageName = parts[len(parts)-1]
		}
	}

	data := FluentTemplateData{
		Provider:         provider,
		UtilsPath:        utilsPath,
		UtilsPackageName: utilsPackageName,
	}

	return executeTemplates(builderDir, "fluent.go", templateNames, data)
}
