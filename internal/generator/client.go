package generator

import (
	"fmt"
	"sort"

	"github.com/carlosnayan/prisma-go-client/internal/migrations"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateClient generates the main client.go file
func GenerateClient(schema *parser.Schema, outputDir string) error {
	// Detect user module
	userModule, err := detectUserModule(outputDir)
	if err != nil {
		return fmt.Errorf("failed to detect user module: %w", err)
	}

	// Determine required imports
	imports, driverImports := determineClientImports(schema, userModule, outputDir)

	// Separate stdlib and third-party imports
	stdlib := make([]string, 0, len(imports))
	thirdParty := make([]string, 0, len(imports))

	for _, imp := range imports {
		if isStdlibImport(imp) {
			stdlib = append(stdlib, imp)
		} else {
			thirdParty = append(thirdParty, imp)
		}
	}

	// Calculate import paths
	modelsPath, queriesPath, _, err := calculateImportPath(userModule, outputDir)
	if err != nil {
		modelsPath = "github.com/carlosnayan/prisma-go-client/generated/models"
		queriesPath = "github.com/carlosnayan/prisma-go-client/generated/queries"
	}

	builderPath, rawPath, err := calculateLocalImportPath(userModule, outputDir)
	if err != nil {
		builderPath = "github.com/carlosnayan/prisma-go-client/generated/builder"
		rawPath = "github.com/carlosnayan/prisma-go-client/generated/raw"
	}

	// Prepare model names (sorted) for use in struct and NewClient
	var modelNamesForStruct []string
	for _, model := range schema.Models {
		modelNamesForStruct = append(modelNamesForStruct, model.Name)
	}
	sort.Strings(modelNamesForStruct)

	// Prepare model info for templates
	models := make([]ModelInfo, 0, len(modelNamesForStruct))
	for _, modelName := range modelNamesForStruct {
		var model *parser.Model
		for _, m := range schema.Models {
			if m.Name == modelName {
				model = m
				break
			}
		}
		if model == nil {
			continue
		}
		columns := getModelColumns(model, schema)
		primaryKey := getPrimaryKey(model)
		tableName := getTableName(model)
		pascalModelName := toPascalCase(modelName)

		models = append(models, ModelInfo{
			Name:       modelName,
			PascalName: pascalModelName,
			Columns:    columns,
			PrimaryKey: primaryKey,
			TableName:  tableName,
		})
	}

	// Prepare template data
	data := ClientTemplateData{
		StdlibImports:     stdlib,
		ThirdPartyImports: thirdParty,
		DriverImports:     driverImports,
		BuilderPath:       builderPath,
		ModelsPath:        modelsPath,
		QueriesPath:       queriesPath,
		RawPath:           rawPath,
		Models:            models,
	}

	// Define template order
	templateNames := []string{
		"imports.tmpl",
		"client_struct.tmpl",
		"logger_config.tmpl",
		"new_client.tmpl",
		"close_method.tmpl",
		"raw_method.tmpl",
		"transaction_client.tmpl",
		"transaction_method.tmpl",
	}

	// Generate client.go using templates with package "generated" for root directory
	return executeTemplatesFromDirWithPackage(outputDir, "client.go", "client", templateNames, data, "generated")
}

// Note: Model access is now via fields (e.g., client.Users) instead of methods (e.g., client.Users())
// This allows for a cleaner API: client.Users.Update() instead of client.Users().Update()

// getModelColumns returns the columns of a model
func getModelColumns(model *parser.Model, schema *parser.Schema) []string {
	columns := []string{}
	for _, field := range model.Fields {
		// Skip relations - only include actual database columns
		if isRelation(field, schema) {
			continue
		}
		columnName := field.Name
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					columnName = val
				}
			}
		}
		columns = append(columns, columnName)
	}
	return columns
}

// getPrimaryKey returns the primary key of a model
func getPrimaryKey(model *parser.Model) string {
	for _, attr := range model.Attributes {
		if attr.Name == "id" {
			if len(attr.Arguments) > 0 {
				if fields, ok := attr.Arguments[0].Value.([]interface{}); ok && len(fields) > 0 {
					if fieldName, ok := fields[0].(string); ok {
						return fieldName
					}
				}
			}
		}
	}

	for _, field := range model.Fields {
		for _, attr := range field.Attributes {
			if attr.Name == "id" {
				columnName := field.Name
				for _, mapAttr := range field.Attributes {
					if mapAttr.Name == "map" && len(mapAttr.Arguments) > 0 {
						if val, ok := mapAttr.Arguments[0].Value.(string); ok {
							columnName = val
						}
					}
				}
				return columnName
			}
		}
	}

	return "id"
}

// getTableName returns the table name for a model
// Checks for @@map attribute first, otherwise uses the exact model name as declared in schema
func getTableName(model *parser.Model) string {
	// Check for @@map attribute
	for _, attr := range model.Attributes {
		if attr.Name == "map" && len(attr.Arguments) > 0 {
			if val, ok := attr.Arguments[0].Value.(string); ok {
				return val
			}
		}
	}
	// Default to exact model name as declared in schema (no conversion)
	return model.Name
}

// determineClientImports determines which imports are needed for client.go
// Returns regular imports and driver imports (blank imports) separately
func determineClientImports(schema *parser.Schema, userModule, outputDir string) ([]string, []string) {
	imports := make(map[string]bool)
	var driverImports []string
	var builderPath, rawPath string

	// context is needed for Transaction method
	imports["context"] = true
	// reflect is always needed for SetModelType
	imports["reflect"] = true

	// Calculate import paths for generated packages
	modelsPath, queriesPath, _, err := calculateImportPath(userModule, outputDir)
	if err != nil {
		// Fallback to old paths if detection fails
		modelsPath = "github.com/carlosnayan/prisma-go-client/generated/models"
		queriesPath = "github.com/carlosnayan/prisma-go-client/generated/queries"
	}

	// Calculate local import paths for builder and raw (standalone packages)
	builderPath, rawPath, err = calculateLocalImportPath(userModule, outputDir)
	if err != nil {
		// Fallback to old paths if detection fails
		builderPath = "github.com/carlosnayan/prisma-go-client/generated/builder"
		rawPath = "github.com/carlosnayan/prisma-go-client/generated/raw"
	}

	// These are always needed
	imports[builderPath] = true
	imports[modelsPath] = true
	imports[queriesPath] = true
	imports[rawPath] = true
	imports["os"] = true
	imports["path/filepath"] = true
	imports["sync"] = true
	imports["bufio"] = true
	imports["strings"] = true

	provider := migrations.GetProviderFromSchema(schema)
	switch provider {
	case "postgresql":
		driverImports = append(driverImports, `_ "github.com/jackc/pgx/v5/stdlib"`)
	case "mysql":
		driverImports = append(driverImports, `_ "github.com/go-sql-driver/mysql"`)
	case "sqlite":
		driverImports = append(driverImports, `_ "github.com/mattn/go-sqlite3"`)
	}

	result := []string{}
	if imports["context"] {
		result = append(result, "context")
	}
	if imports["reflect"] {
		result = append(result, "reflect")
	}
	if imports["bufio"] {
		result = append(result, "bufio")
	}
	if imports["strings"] {
		result = append(result, "strings")
	}
	if imports[builderPath] {
		result = append(result, builderPath)
	}
	if imports[modelsPath] {
		result = append(result, modelsPath)
	}
	if imports["os"] {
		result = append(result, "os")
	}
	if imports["path/filepath"] {
		result = append(result, "path/filepath")
	}
	if imports["sync"] {
		result = append(result, "sync")
	}
	if imports[queriesPath] {
		result = append(result, queriesPath)
	}
	if imports[rawPath] {
		result = append(result, rawPath)
	}

	return result, driverImports
}
