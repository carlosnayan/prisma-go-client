package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
)

// FluentTemplateData holds data for fluent.go template generation
type FluentTemplateData struct {
	Provider         string
	UtilsPath        string
	UtilsPackageName string // Package name extracted from UtilsPath (last segment)
}

// DriverTemplateData holds data for driver.go template generation
type DriverTemplateData struct {
	Provider    string
	BuilderPath string
}

// ModelInfo holds information about a model for template generation
type ModelInfo struct {
	Name       string
	PascalName string
	Columns    []string
	PrimaryKey string
	TableName  string
}

// ClientTemplateData holds data for client.go template generation
type ClientTemplateData struct {
	StdlibImports     []string
	ThirdPartyImports []string
	DriverImports     []string
	BuilderPath       string
	ModelsPath        string
	QueriesPath       string
	RawPath           string
	Models            []ModelInfo
}

// FieldInfo holds information about a model field for template generation
type FieldInfo struct {
	Name    string
	GoType  string
	JSONTag string
	DBTag   string
}

// ModelTemplateData holds data for model file template generation
type ModelTemplateData struct {
	ModelName  string
	PascalName string
	Imports    []string
	Fields     []FieldInfo
}

// HelpersTemplateData holds data for helpers.go template generation
type HelpersTemplateData struct {
	Imports       []string
	InputsPath    string
	NeededFilters map[string]bool
}

// QueryResultTemplateData holds data for query_result.go template generation
type QueryResultTemplateData struct {
	BuilderPath string
}

// FieldFilterInfo holds information about a field's filter type for query generation
type FieldFilterInfo struct {
	FieldName   string // PascalCase field name
	DBFieldName string // Actual database column name
	FilterType  string // Filter type (StringFilter, IntFilter, etc.)
}

// QueryTemplateData holds data for query file template generation
type QueryTemplateData struct {
	ModelName         string
	PascalName        string
	StdlibImports     []string
	ThirdPartyImports []string
	BuilderPath       string
	ModelsPath        string
	InputsPath        string
	Fields            []FieldFilterInfo
	SelectFields      []SelectFieldInfo // Fields for Select operations
	UpdateFields      []UpdateFieldInfo // Fields for Update operations
	CreateFields      []CreateFieldInfo // Fields for Create operations
	Columns           []string
	PrimaryKey        string
	TableName         string
}

// SelectFieldInfo holds information about a field for Select operations
type SelectFieldInfo struct {
	FieldName  string // PascalCase field name
	ColumnName string // Actual database column name
}

// UpdateFieldInfo holds information about a field for Update operations
type UpdateFieldInfo struct {
	FieldName   string // PascalCase field name
	DBFieldName string // Actual database column name
	IsPointer   bool   // Whether the field in the model is a pointer type
}

// CreateFieldInfo holds information about a field for Create operations
type CreateFieldInfo struct {
	FieldName            string // PascalCase field name
	IsOptional           bool   // Whether field is optional (pointer)
	IsRequired           bool   // Whether field is required (not optional and no default)
	IsNonPointerOptional bool   // Whether field doesn't use pointer in model even when optional (Json, Bytes)
}

// FiltersTemplateData holds data for filters.go template generation
type FiltersTemplateData struct {
	StdlibImports []string
	NeededFilters map[string]bool
}

// InputFieldInfo holds information about a field for input types
type InputFieldInfo struct {
	FieldName string // PascalCase field name
	GoType    string // Go type (with pointer if optional)
	JSONTag   string // JSON tag name
}

// WhereInputFieldInfo holds information about a field for WhereInput
type WhereInputFieldInfo struct {
	FieldName  string // PascalCase field name
	FilterType string // Filter type name (StringFilter, IntFilter, etc.)
	JSONTag    string // JSON tag name
}

// InputSelectFieldInfo holds information about a field for Select in input types
type InputSelectFieldInfo struct {
	FieldName string // PascalCase field name
	JSONTag   string // JSON tag name
}

// InputTemplateData holds data for model input file template generation
type InputTemplateData struct {
	ModelName        string
	PascalName       string
	StdlibImports    []string
	FiltersPath      string
	CreateFields     []InputFieldInfo
	UpdateFields     []InputFieldInfo
	WhereInputFields []WhereInputFieldInfo
	SelectFields     []InputSelectFieldInfo
}

// InputHelpersTemplateData holds data for inputs/helpers.go template generation
type InputHelpersTemplateData struct {
	StdlibImports []string
	NeedsDateTime bool
	NeedsJson     bool
}

// executeTemplates executes multiple templates and writes them to a file
func executeTemplates(outputDir, filename string, templateNames []string, data FluentTemplateData) error {
	return executeTemplatesFromDir(outputDir, filename, "fluent", templateNames, data)
}

// executeTemplatesFromDir executes multiple templates from a specific template directory
func executeTemplatesFromDir(outputDir, filename, templateDir string, templateNames []string, data interface{}) error {
	return executeTemplatesFromDirWithPackage(outputDir, filename, templateDir, templateNames, data, "builder")
}

// executeTemplatesFromDirWithPackage executes multiple templates from a specific template directory with a custom package name
func executeTemplatesFromDirWithPackage(outputDir, filename, templateDir string, templateNames []string, data interface{}, packageName string) error {
	file, err := createGeneratedFile(filepath.Join(outputDir, filename), packageName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir(templateDir)
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Execute each template in order
	for _, tmplName := range templateNames {
		tmplPath := filepath.Join(templatesDir, tmplName)
		if err := executeTemplate(file, tmplPath, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
		}
	}

	return nil
}

// executeTemplate loads and executes a single template
func executeTemplate(file *os.File, templatePath string, data interface{}) error {
	// Read template file
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	// Write to file
	if _, err := file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write template output: %w", err)
	}

	return nil
}

var (
	// generatorSourceDir is the directory where template_loader.go is located
	// This is initialized once using runtime.Caller
	generatorSourceDir = initGeneratorSourceDir()
)

// initGeneratorSourceDir finds the directory where this source file is located
func initGeneratorSourceDir() string {
	// Get the directory of template_loader.go using runtime.Caller
	// We use Caller(1) to skip this function itself
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		// Fallback: try to find it relative to current working directory
		wd, _ := os.Getwd()
		possibleDirs := []string{
			filepath.Join(wd, "internal", "generator"),
			filepath.Join(wd, "prisma-go-client", "internal", "generator"),
		}
		for _, dir := range possibleDirs {
			if _, err := os.Stat(filepath.Join(dir, "template_loader.go")); err == nil {
				return dir
			}
		}
		return wd
	}
	return filepath.Dir(filename)
}

// getTemplatesDir returns the templates directory for a specific template subdirectory
func getTemplatesDir(templateSubdir string) (string, string, error) {
	// Use the pre-initialized generator source directory
	sourceDir := generatorSourceDir

	// Templates are in internal/generator/templates/ relative to template_loader.go
	templatesPath := filepath.Join(sourceDir, "templates", templateSubdir)

	// Verify the path exists
	if _, err := os.Stat(templatesPath); err != nil {
		// Try alternative paths for different project structures
		possiblePaths := []string{
			templatesPath,
			filepath.Join(sourceDir, "..", "templates", templateSubdir),
			filepath.Join(sourceDir, "..", "..", "templates", templateSubdir),
		}

		found := false
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				templatesPath = path
				found = true
				break
			}
		}

		if !found {
			return "", "", fmt.Errorf("templates directory not found: %s (tried: %v)", templateSubdir, possiblePaths)
		}
	}

	// Return the generator directory and the full templates path
	generatorDir := sourceDir
	return generatorDir, templatesPath, nil
}

// executeSingleTemplate executes a single template without data
func executeSingleTemplate(outputDir, filename, templateDir, templateName string) error {
	file, err := createGeneratedFile(filepath.Join(outputDir, filename), "builder")
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir(templateDir)
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Execute template
	tmplPath := filepath.Join(templatesDir, templateName)
	return executeTemplate(file, tmplPath, nil)
}

// executeModelTemplate executes a single template for a model file with data
func executeModelTemplate(filePath, packageName, templateDir, templateName string, data interface{}) error {
	file, err := createGeneratedFile(filePath, packageName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir(templateDir)
	if err != nil {
		return fmt.Errorf("failed to get templates directory for %s: %w", templateDir, err)
	}

	// Execute template
	tmplPath := filepath.Join(templatesDir, templateName)
	return executeTemplate(file, tmplPath, data)
}

// executeFiltersHelpersTemplates executes multiple templates for filters/helpers.go
func executeFiltersHelpersTemplates(filePath string, templateNames []string, data HelpersTemplateData) error {
	file, err := createGeneratedFile(filePath, "filters")
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir("filters")
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Execute each template in order
	for _, tmplName := range templateNames {
		tmplPath := filepath.Join(templatesDir, tmplName)
		if err := executeTemplate(file, tmplPath, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
		}
	}

	return nil
}

// executeQueryTemplates executes multiple templates for query files
func executeQueryTemplates(filePath string, templateNames []string, data QueryTemplateData) error {
	file, err := createGeneratedFile(filePath, "queries")
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir("queries")
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Execute each template in order
	for _, tmplName := range templateNames {
		tmplPath := filepath.Join(templatesDir, tmplName)
		if err := executeTemplate(file, tmplPath, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
		}
	}

	return nil
}

// executeFiltersTemplates executes multiple templates for filters.go
func executeFiltersTemplates(filePath string, templateNames []string, data FiltersTemplateData) error {
	file, err := createGeneratedFile(filePath, "filters")
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir("filters")
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Execute each template in order
	for _, tmplName := range templateNames {
		tmplPath := filepath.Join(templatesDir, tmplName)
		if err := executeTemplate(file, tmplPath, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
		}
	}

	return nil
}

// executeInputHelpersTemplates executes multiple templates for inputs/helpers.go
func executeInputHelpersTemplates(filePath string, templateNames []string, data InputHelpersTemplateData) error {
	file, err := createGeneratedFile(filePath, "inputs")
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir("inputs")
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Execute each template in order
	for _, tmplName := range templateNames {
		tmplPath := filepath.Join(templatesDir, tmplName)
		if err := executeTemplate(file, tmplPath, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
		}
	}

	return nil
}

// executeInputTemplates executes multiple templates for model input files
func executeInputTemplates(filePath string, templateNames []string, data InputTemplateData) error {
	file, err := createGeneratedFile(filePath, "inputs")
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the templates directory
	_, templatesDir, err := getTemplatesDir("inputs")
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	// Execute each template in order
	for _, tmplName := range templateNames {
		tmplPath := filepath.Join(templatesDir, tmplName)
		if err := executeTemplate(file, tmplPath, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
		}
	}

	return nil
}
