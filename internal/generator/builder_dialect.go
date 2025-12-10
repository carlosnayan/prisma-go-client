package generator

// generateBuilderDialect generates dialect.go with interface and implementations using templates
func generateBuilderDialect(builderDir string) error {
	// Define the order of templates to execute
	templateNames := []string{
		"imports.tmpl",
		"dialect_interface.tmpl",
		"get_dialect.tmpl",
		"postgresql_dialect.tmpl",
		"mysql_dialect.tmpl",
		"sqlite_dialect.tmpl",
	}

	data := FluentTemplateData{}

	return executeTemplatesFromDir(builderDir, "dialect.go", "dialect", templateNames, data)
}
