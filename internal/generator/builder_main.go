package generator

import (
	"strings"
)

// generateBuilderMain generates builder.go with TableQueryBuilder using templates
func generateBuilderMain(builderDir string, provider string, utilsPath string) error {
	// Define the order of templates to execute
	templateNames := []string{
		"imports.tmpl",
		"interfaces.tmpl",
		"tx_interface.tmpl",
		"table_query_builder.tmpl",
		"methods.tmpl",
		"build_query.tmpl",
		"scan.tmpl",
		"helpers.tmpl",
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

	return executeTemplatesFromDir(builderDir, "builder.go", "builder_main", templateNames, data)
}
