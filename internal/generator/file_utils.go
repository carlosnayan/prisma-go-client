package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// FileHeaderData holds data for file header template
type FileHeaderData struct {
	PackageName string
}

// createGeneratedFile creates a new file and writes the standard header using templates
// Returns the file handle and any error encountered
func createGeneratedFile(filePath string, packageName string) (*os.File, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	// Write standard header using template
	if err := writeFileHeader(file, packageName); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write file header: %w", err)
	}

	return file, nil
}

// writeFileHeader writes the file header using a template
func writeFileHeader(file *os.File, packageName string) error {
	// Get the templates directory
	_, templatesDir, err := getTemplatesDir("shared")
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Load and execute the header template
	tmplPath := filepath.Join(templatesDir, "file_header.tmpl")
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplPath, err)
	}

	// Parse template
	tmpl, err := template.New("file_header").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplPath, err)
	}

	// Prepare data
	data := FileHeaderData{
		PackageName: packageName,
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", tmplPath, err)
	}

	// Write to file (template already includes newline)
	if _, err := file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write template output: %w", err)
	}

	return nil
}
