package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateRaw generates a standalone raw package in the output directory
// This package has no external dependencies on github.com/carlosnayan/prisma-go-client
func GenerateRaw(outputDir string) error {
	rawDir := filepath.Join(outputDir, "raw")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		return fmt.Errorf("failed to create raw directory: %w", err)
	}

	rawFile := filepath.Join(rawDir, "raw.go")

	// Create file and write package declaration
	file, err := createGeneratedFile(rawFile, "raw")
	if err != nil {
		return err
	}
	file.Close()

	// Define template order - imports must come first, then interfaces
	templateNames := []string{
		"imports.tmpl",
		"dbtx_alias.tmpl",
	}

	// Generate imports first
	if err := executeRawTemplatesAppend(rawFile, templateNames); err != nil {
		return fmt.Errorf("failed to generate imports: %w", err)
	}

	// Generate shared interfaces (after imports)
	file, err = os.OpenFile(rawFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for appending: %w", err)
	}
	if err := generateDBInterfaces(file); err != nil {
		file.Close()
		return fmt.Errorf("failed to generate DB interfaces: %w", err)
	}
	file.Close()

	// Generate rest of the templates
	restTemplateNames := []string{
		"executor_struct.tmpl",
		"has_builder_db_methods.tmpl",
		"new_function.tmpl",
		"builder_db_adapter.tmpl",
		"adapters.tmpl",
		"executor_methods.tmpl",
		"scan_result.tmpl",
	}

	return executeRawTemplatesAppend(rawFile, restTemplateNames)
}

// executeRawTemplatesAppend executes templates and appends to existing file
func executeRawTemplatesAppend(filePath string, templateNames []string) error {
	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for appending: %w", err)
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir("raw")
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Execute each template in order
	for _, tmplName := range templateNames {
		tmplPath := filepath.Join(templatesDir, tmplName)
		if err := executeTemplate(file, tmplPath, nil); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
		}
	}

	return nil
}
