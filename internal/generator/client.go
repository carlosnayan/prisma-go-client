package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/carlosnayan/prisma-go-client/internal/migrations"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateClient generates the main client.go file
func GenerateClient(schema *parser.Schema, outputDir string) error {
	clientFile := filepath.Join(outputDir, "client.go")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Detect user module
	userModule, err := detectUserModule(outputDir)
	if err != nil {
		return fmt.Errorf("failed to detect user module: %w", err)
	}

	file, err := createGeneratedFile(clientFile, "db")
	if err != nil {
		return err
	}
	defer file.Close()

	// Determine required imports
	imports, driverImports := determineClientImports(schema, userModule, outputDir)
	if len(imports) > 0 || len(driverImports) > 0 {
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

		writeImportsWithGroups(file, stdlib, thirdParty)

		// Write driver imports (blank imports) with comment
		if len(driverImports) > 0 {
			if len(thirdParty) > 0 || len(stdlib) > 0 {
				fmt.Fprintf(file, "\n")
			}
			fmt.Fprintf(file, "\t// Database driver (blank import to register driver)\n")
			for _, imp := range driverImports {
				fmt.Fprintf(file, "\t%s\n", imp)
			}
		}

		closeImports(file)
	}

	// Prepare model names (sorted) for use in struct and NewClient
	var modelNamesForStruct []string
	for _, model := range schema.Models {
		modelNamesForStruct = append(modelNamesForStruct, model.Name)
	}
	sort.Strings(modelNamesForStruct)

	// Client struct
	fmt.Fprintf(file, "// Client is the main Prisma client\n")
	fmt.Fprintf(file, "type Client struct {\n")
	fmt.Fprintf(file, "\tdb builder.DBTX\n")
	fmt.Fprintf(file, "\traw *raw.Executor\n")

	// Add fields for each model
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
		pascalModelName := toPascalCase(modelName)
		fmt.Fprintf(file, "\t%s *queries.%sQuery\n", pascalModelName, pascalModelName)
	}

	fmt.Fprintf(file, "}\n\n")

	// Generate logger configuration helper
	generateLoggerConfigHelper(file)

	// NewClient
	fmt.Fprintf(file, "// NewClient creates a new Prisma client\n")
	fmt.Fprintf(file, "// db must be a builder.DBTX implementation (e.g., driver.NewPgxPool, driver.NewSQLDB)\n")
	fmt.Fprintf(file, "// The logger is automatically configured from prisma.conf if present\n")
	fmt.Fprintf(file, "func NewClient(db builder.DBTX) *Client {\n")
	fmt.Fprintf(file, "\t// Configure logger from prisma.conf if available (only once)\n")
	fmt.Fprintf(file, "\tconfigureLoggerFromConfig()\n")
	fmt.Fprintf(file, "\tclient := &Client{\n")
	fmt.Fprintf(file, "\t\tdb:  db,\n")
	fmt.Fprintf(file, "\t\traw: raw.New(db),\n")
	fmt.Fprintf(file, "\t}\n")

	// Initialize model queries
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
		columns := getModelColumns(model)
		primaryKey := getPrimaryKey(model)
		hasDeleted := hasDeletedAt(model)
		pascalModelName := toPascalCase(modelName)

		fmt.Fprintf(file, "\t// Initialize %s query\n", pascalModelName)
		fmt.Fprintf(file, "\tcolumns_%s := []string{%s}\n", pascalModelName, formatColumns(columns))
		fmt.Fprintf(file, "\tquery_%s := builder.NewQuery(client.db, %q, columns_%s)\n", pascalModelName, toSnakeCase(modelName), pascalModelName)
		if primaryKey != "" {
			fmt.Fprintf(file, "\tquery_%s.SetPrimaryKey(%q)\n", pascalModelName, primaryKey)
		}
		if hasDeleted {
			fmt.Fprintf(file, "\tquery_%s.SetHasDeleted(true)\n", pascalModelName)
		}
		fmt.Fprintf(file, "\tmodelType_%s := reflect.TypeOf(models.%s{})\n", pascalModelName, pascalModelName)
		fmt.Fprintf(file, "\tquery_%s.SetModelType(modelType_%s)\n", pascalModelName, pascalModelName)
		fmt.Fprintf(file, "\tclient.%s = &queries.%sQuery{Query: query_%s}\n", pascalModelName, pascalModelName, pascalModelName)
	}

	fmt.Fprintf(file, "\treturn client\n")
	fmt.Fprintf(file, "}\n\n")

	// Raw
	fmt.Fprintf(file, "// Raw returns the raw SQL executor\n")
	fmt.Fprintf(file, "func (c *Client) Raw() *raw.Executor {\n")
	fmt.Fprintf(file, "\treturn c.raw\n")
	fmt.Fprintf(file, "}\n\n")

	// Note: Model access is now via fields (e.g., client.Users) instead of methods (e.g., client.Users())
	// This allows for a cleaner API: client.Users.Update() instead of client.Users().Update()

	// Generate TransactionClient and Transaction method
	generateTransactionClient(file, schema)
	generateTransactionMethod(file, schema)

	// Helper functions are generated in helpers.go, not here

	return nil
}

// Note: Model access is now via fields (e.g., client.Users) instead of methods (e.g., client.Users())
// This allows for a cleaner API: client.Users.Update() instead of client.Users().Update()

// getModelColumns returns the columns of a model
func getModelColumns(model *parser.Model) []string {
	columns := []string{}
	for _, field := range model.Fields {
		// Skip relations - only include actual database columns
		if isRelation(field) {
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

// hasDeletedAt checks if the model has a deleted_at field
func hasDeletedAt(model *parser.Model) bool {
	for _, field := range model.Fields {
		if field.Name == "deleted_at" || field.Name == "deletedAt" {
			return true
		}
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					if val == "deleted_at" {
						return true
					}
				}
			}
		}
	}
	return false
}

// formatColumns formats columns for Go code
func formatColumns(columns []string) string {
	if len(columns) == 0 {
		return ""
	}
	result := ""
	for i, col := range columns {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%q", col)
	}
	return result
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
		modelsPath = "github.com/carlosnayan/prisma-go-client/db/models"
		queriesPath = "github.com/carlosnayan/prisma-go-client/db/queries"
	}

	// Calculate local import paths for builder and raw (standalone packages)
	builderPath, rawPath, err = calculateLocalImportPath(userModule, outputDir)
	if err != nil {
		// Fallback to old paths if detection fails
		builderPath = "github.com/carlosnayan/prisma-go-client/db/builder"
		rawPath = "github.com/carlosnayan/prisma-go-client/db/raw"
	}

	// These are always needed
	imports[builderPath] = true
	imports[modelsPath] = true
	imports[queriesPath] = true
	imports[rawPath] = true
	// Add imports for logger configuration
	imports["os"] = true
	imports["path/filepath"] = true
	imports["sync"] = true
	imports["github.com/BurntSushi/toml"] = true
	imports["github.com/joho/godotenv"] = true

	// Add driver import based on provider (blank import)
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
	if imports[builderPath] {
		result = append(result, builderPath)
	}
	if imports[modelsPath] {
		result = append(result, modelsPath)
	}
	// Add logger config imports
	if imports["os"] {
		result = append(result, "os")
	}
	if imports["path/filepath"] {
		result = append(result, "path/filepath")
	}
	if imports["sync"] {
		result = append(result, "sync")
	}
	if imports["github.com/BurntSushi/toml"] {
		result = append(result, "github.com/BurntSushi/toml")
	}
	if imports["github.com/joho/godotenv"] {
		result = append(result, "github.com/joho/godotenv")
	}
	if imports[queriesPath] {
		result = append(result, queriesPath)
	}
	if imports[rawPath] {
		result = append(result, rawPath)
	}

	return result, driverImports
}

// generateLoggerConfigHelper generates the logger configuration helper function
func generateLoggerConfigHelper(file *os.File) {
	fmt.Fprintf(file, "var (\n")
	fmt.Fprintf(file, "\tloggerConfigOnce sync.Once\n")
	fmt.Fprintf(file, ")\n\n")

	fmt.Fprintf(file, "// configureLoggerFromConfig configures the logger from prisma.conf\n")
	fmt.Fprintf(file, "// This function is called automatically when NewClient is created\n")
	fmt.Fprintf(file, "func configureLoggerFromConfig() {\n")
	fmt.Fprintf(file, "\tloggerConfigOnce.Do(func() {\n")
	fmt.Fprintf(file, "\t\t// Look for prisma.conf in project root\n")
	fmt.Fprintf(file, "\t\twd, err := os.Getwd()\n")
	fmt.Fprintf(file, "\t\tif err != nil {\n")
	fmt.Fprintf(file, "\t\t\treturn\n")
	fmt.Fprintf(file, "\t\t}\n\n")

	fmt.Fprintf(file, "\t\t// Search up directories for prisma.conf\n")
	fmt.Fprintf(file, "\t\tdir := wd\n")
	fmt.Fprintf(file, "\t\tfor {\n")
	fmt.Fprintf(file, "\t\t\tconfigPath := filepath.Join(dir, \"prisma.conf\")\n")
	fmt.Fprintf(file, "\t\t\tif _, err := os.Stat(configPath); err == nil {\n")
	fmt.Fprintf(file, "\t\t\t\t// Load .env if exists\n")
	fmt.Fprintf(file, "\t\t\t\tconfigDir := filepath.Dir(configPath)\n")
	fmt.Fprintf(file, "\t\t\t\tenvPath := filepath.Join(configDir, \".env\")\n")
	fmt.Fprintf(file, "\t\t\t\tif _, err := os.Stat(envPath); err == nil {\n")
	fmt.Fprintf(file, "\t\t\t\t\t_ = godotenv.Load(envPath)\n")
	fmt.Fprintf(file, "\t\t\t\t}\n\n")

	fmt.Fprintf(file, "\t\t\t\t// Read and parse prisma.conf\n")
	fmt.Fprintf(file, "\t\t\t\tdata, err := os.ReadFile(configPath)\n")
	fmt.Fprintf(file, "\t\t\t\tif err != nil {\n")
	fmt.Fprintf(file, "\t\t\t\t\treturn\n")
	fmt.Fprintf(file, "\t\t\t\t}\n\n")

	fmt.Fprintf(file, "\t\t\t\t// Parse TOML config\n")
	fmt.Fprintf(file, "\t\t\t\ttype Config struct {\n")
	fmt.Fprintf(file, "\t\t\t\t\tDebug struct {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tLog []string `toml:\"log,omitempty\"`\n")
	fmt.Fprintf(file, "\t\t\t\t\t} `toml:\"debug,omitempty\"`\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\tvar cfg Config\n")
	fmt.Fprintf(file, "\t\t\t\tif _, err := toml.Decode(string(data), &cfg); err != nil {\n")
	fmt.Fprintf(file, "\t\t\t\t\treturn\n")
	fmt.Fprintf(file, "\t\t\t\t}\n\n")

	fmt.Fprintf(file, "\t\t\t\t// Configure logger if log levels are specified in [debug] section\n")
	fmt.Fprintf(file, "\t\t\t\tif len(cfg.Debug.Log) > 0 {\n")
	fmt.Fprintf(file, "\t\t\t\t\tbuilder.SetLogLevels(cfg.Debug.Log)\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\treturn\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")

	fmt.Fprintf(file, "\t\t\tparent := filepath.Dir(dir)\n")
	fmt.Fprintf(file, "\t\t\tif parent == dir {\n")
	fmt.Fprintf(file, "\t\t\t\t// Reached root, not found\n")
	fmt.Fprintf(file, "\t\t\t\treturn\n")
	fmt.Fprintf(file, "\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\tdir = parent\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t})\n")
	fmt.Fprintf(file, "}\n\n")
}

// generateTransactionClient generates the TransactionClient struct and its methods
func generateTransactionClient(file *os.File, schema *parser.Schema) {
	// Prepare model names (sorted) for use in struct
	var modelNamesForTx []string
	for _, model := range schema.Models {
		modelNamesForTx = append(modelNamesForTx, model.Name)
	}
	sort.Strings(modelNamesForTx)

	// TransactionClient struct
	fmt.Fprintf(file, "// TransactionClient is a client that executes operations within a transaction\n")
	fmt.Fprintf(file, "// All operations executed through TransactionClient are part of the same transaction\n")
	fmt.Fprintf(file, "type TransactionClient struct {\n")
	fmt.Fprintf(file, "\ttx  *builder.Transaction\n")
	fmt.Fprintf(file, "\traw *raw.Executor\n")

	// Add fields for each model
	for _, modelName := range modelNamesForTx {
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
		pascalModelName := toPascalCase(modelName)
		fmt.Fprintf(file, "\t%s *queries.%sQuery\n", pascalModelName, pascalModelName)
	}

	fmt.Fprintf(file, "}\n\n")

	// Raw method for TransactionClient
	fmt.Fprintf(file, "// Raw returns the raw SQL executor for the transaction\n")
	fmt.Fprintf(file, "func (tc *TransactionClient) Raw() *raw.Executor {\n")
	fmt.Fprintf(file, "\treturn tc.raw\n")
	fmt.Fprintf(file, "}\n\n")
}

// Note: Model access in TransactionClient is now via fields (e.g., tx.Users) instead of methods (e.g., tx.Users())
// This allows for a cleaner API: tx.Users.Update() instead of tx.Users().Update()

// generateTransactionMethod generates the Transaction method on Client
func generateTransactionMethod(file *os.File, schema *parser.Schema) {
	fmt.Fprintf(file, "// Transaction executes a function within a database transaction\n")
	fmt.Fprintf(file, "// If the function returns an error, the transaction is automatically rolled back\n")
	fmt.Fprintf(file, "// Example:\n")
	fmt.Fprintf(file, "//   err := client.Transaction(ctx, func(tx *TransactionClient) error {\n")
	fmt.Fprintf(file, "//       user, err := tx.User.Create().Data(...).Exec(ctx)\n")
	fmt.Fprintf(file, "//       if err != nil { return err }\n")
	fmt.Fprintf(file, "//       _, err = tx.Post.Create().Data(...).Exec(ctx)\n")
	fmt.Fprintf(file, "//       return err\n")
	fmt.Fprintf(file, "//   })\n")
	fmt.Fprintf(file, "func (c *Client) Transaction(ctx context.Context, fn func(*TransactionClient) error) error {\n")
	fmt.Fprintf(file, "\treturn builder.ExecuteTransaction(ctx, c.db, func(tx *builder.Transaction) error {\n")
	fmt.Fprintf(file, "\t\t// Create adapter for raw executor\n")
	fmt.Fprintf(file, "\t\ttxAdapter := tx.DB()\n")
	fmt.Fprintf(file, "\t\ttxClient := &TransactionClient{\n")
	fmt.Fprintf(file, "\t\t\ttx:  tx,\n")
	fmt.Fprintf(file, "\t\t\traw: raw.New(txAdapter),\n")
	fmt.Fprintf(file, "\t\t}\n")

	// Initialize model queries for TransactionClient
	var modelNamesForTxInit []string
	for _, model := range schema.Models {
		modelNamesForTxInit = append(modelNamesForTxInit, model.Name)
	}
	sort.Strings(modelNamesForTxInit)

	for _, modelName := range modelNamesForTxInit {
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
		columns := getModelColumns(model)
		primaryKey := getPrimaryKey(model)
		hasDeleted := hasDeletedAt(model)
		pascalModelName := toPascalCase(modelName)

		fmt.Fprintf(file, "\t\t// Initialize %s query\n", pascalModelName)
		fmt.Fprintf(file, "\t\tcolumns_%s := []string{%s}\n", pascalModelName, formatColumns(columns))
		fmt.Fprintf(file, "\t\tquery_%s := txClient.tx.Query(%q, columns_%s)\n", pascalModelName, toSnakeCase(modelName), pascalModelName)
		if primaryKey != "" {
			fmt.Fprintf(file, "\t\tquery_%s.SetPrimaryKey(%q)\n", pascalModelName, primaryKey)
		}
		if hasDeleted {
			fmt.Fprintf(file, "\t\tquery_%s.SetHasDeleted(true)\n", pascalModelName)
		}
		fmt.Fprintf(file, "\t\tmodelType_%s := reflect.TypeOf(models.%s{})\n", pascalModelName, pascalModelName)
		fmt.Fprintf(file, "\t\tquery_%s.SetModelType(modelType_%s)\n", pascalModelName, pascalModelName)
		fmt.Fprintf(file, "\t\ttxClient.%s = &queries.%sQuery{Query: query_%s}\n", pascalModelName, pascalModelName, pascalModelName)
	}

	fmt.Fprintf(file, "\t\treturn fn(txClient)\n")
	fmt.Fprintf(file, "\t})\n")
	fmt.Fprintf(file, "}\n\n")
}
