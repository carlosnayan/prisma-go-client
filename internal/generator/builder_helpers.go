package generator

// generateBuilderLimits generates limits.go using templates
func generateBuilderLimits(builderDir string) error {
	return executeSingleTemplate(builderDir, "limits.go", "builder_helpers", "limits.tmpl")
}

// generateBuilderContext generates context.go using templates
func generateBuilderContext(builderDir string) error {
	return executeSingleTemplate(builderDir, "context.go", "builder_helpers", "context.tmpl")
}

// generateBuilderErrors generates errors.go using templates
func generateBuilderErrors(builderDir string) error {
	return executeSingleTemplate(builderDir, "errors.go", "builder_helpers", "errors.tmpl")
}

// generateBuilderOptions generates options.go using templates
func generateBuilderOptions(builderDir string) error {
	return executeSingleTemplate(builderDir, "options.go", "builder_helpers", "options.tmpl")
}

// generateBuilderWhere generates where.go using templates
func generateBuilderWhere(builderDir string) error {
	return executeSingleTemplate(builderDir, "where.go", "builder_helpers", "where.tmpl")
}
