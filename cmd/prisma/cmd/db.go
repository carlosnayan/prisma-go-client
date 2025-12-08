package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/formatter"
	"github.com/carlosnayan/prisma-go-client/internal/migrations"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

var (
	dbPushAcceptDataLossFlag bool
	dbPushSkipGenerateFlag   bool
	dbExecuteFileFlag        string
	dbExecuteStdinFlag       string
)

var dbCmd = &cli.Command{
	Name:  "db",
	Short: "Database management commands",
	Long: `Commands to interact directly with the database:
  - push: Apply schema changes directly to the database
  - pull: Generate schema.prisma from the database (introspection)
  - seed: Execute database seed
  - execute: Execute arbitrary SQL`,
	Subcommands: []*cli.Command{
		dbPushCmd,
		dbPullCmd,
		dbSeedCmd,
		dbExecuteCmd,
	},
}

var dbPushCmd = &cli.Command{
	Name:  "push",
	Short: "Apply schema changes directly to the database",
	Long: `Applies schema.prisma changes directly to the database
without creating migration files. Useful for rapid development.`,
	Flags: []*cli.Flag{
		{
			Name:  "accept-data-loss",
			Usage: "Accept data loss in destructive operations",
			Value: &dbPushAcceptDataLossFlag,
		},
		{
			Name:  "skip-generate",
			Usage: "Do not run prisma generate after push",
			Value: &dbPushSkipGenerateFlag,
		},
	},
	Run: runDbPush,
}

var dbPullCmd = &cli.Command{
	Name:  "pull",
	Short: "Generate schema.prisma from the database (introspection)",
	Long: `Connects to the database and generates schema.prisma based
on the current database structure. Preserves relationships and types.`,
	Run: runDbPull,
}

var dbSeedCmd = &cli.Command{
	Name:  "seed",
	Short: "Execute database seed",
	Long:  `Executes the seed script configured in prisma.conf to populate the database with initial data.`,
	Run:   runDbSeed,
}

var dbExecuteCmd = &cli.Command{
	Name:  "execute",
	Short: "Execute arbitrary SQL on the database",
	Long:  `Executes a SQL command on the database. Useful for administrative commands.`,
	Flags: []*cli.Flag{
		{
			Name:  "file",
			Usage: "SQL file to execute",
			Value: &dbExecuteFileFlag,
		},
		{
			Name:  "stdin",
			Usage: "Read SQL from stdin",
			Value: &dbExecuteStdinFlag,
		},
	},
	Run: runDbExecute,
}

func runDbPush(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	acceptDataLoss := dbPushAcceptDataLossFlag
	skipGenerate := dbPushSkipGenerateFlag

	// Parse schema
	schemaPath := getSchemaPath()
	schema, errors, err := parser.ParseFile(schemaPath)
	if err != nil || len(errors) > 0 {
		if len(errors) > 0 {
			fmt.Println("Errors in schema:")
			for _, e := range errors {
				fmt.Printf("  %s\n", e)
			}
		}
		return fmt.Errorf("error parsing schema: %w", err)
	}

	// Connect to database
	dbURL := cfg.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not configured")
	}

	db, err := migrations.ConnectDatabase(dbURL)
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer db.Close()

	// Get provider
	provider := migrations.GetProviderFromSchema(schema)

	// Introspect database
	fmt.Println("Introspecting database...")
	dbSchema, err := migrations.IntrospectDatabase(db, provider)
	if err != nil {
		return fmt.Errorf("error introspecting database: %w", err)
	}

	// Compare schema with database
	fmt.Println("Detecting changes...")
	diff, err := migrations.CompareSchema(schema, dbSchema, provider)
	if err != nil {
		return fmt.Errorf("error comparing schema: %w", err)
	}

	// Check if there are changes
	hasChanges := len(diff.TablesToCreate) > 0 ||
		len(diff.TablesToAlter) > 0 ||
		len(diff.TablesToDrop) > 0 ||
		len(diff.IndexesToCreate) > 0 ||
		len(diff.IndexesToDrop) > 0

	if !hasChanges {
		fmt.Println("No changes detected. Database is synchronized with schema.")
		return nil
	}

	// Check for destructive changes
	hasDestructiveChanges := len(diff.TablesToDrop) > 0
	for _, alter := range diff.TablesToAlter {
		if len(alter.DropColumns) > 0 {
			hasDestructiveChanges = true
			break
		}
	}

	if hasDestructiveChanges && !acceptDataLoss {
		fmt.Println("Warning: This operation may cause data loss:")
		if len(diff.TablesToDrop) > 0 {
			fmt.Printf("  - %d table(s) will be removed\n", len(diff.TablesToDrop))
		}
		for _, alter := range diff.TablesToAlter {
			if len(alter.DropColumns) > 0 {
				fmt.Printf("  - Columns will be removed from table %s\n", alter.TableName)
			}
		}
		fmt.Print("\nTo continue, run again with --accept-data-loss\n")
		return fmt.Errorf("destructive changes detected")
	}

	// Generate SQL
	sql, err := migrations.GenerateMigrationSQL(diff, provider)
	if err != nil {
		return fmt.Errorf("error generating SQL: %w", err)
	}

	if sql == "" || strings.TrimSpace(sql) == "" {
		fmt.Println("No changes to apply.")
		return nil
	}

	// Apply SQL directly to database
	fmt.Println("Applying changes to database...")
	statements := migrations.SplitSQLStatements(sql)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("error executing SQL: %w\nSQL: %s", err, stmt)
		}
	}

	fmt.Println("Changes applied successfully!")

	// Run generate if not skipped
	if !skipGenerate {
		fmt.Println("Generating code...")
		return runGenerate([]string{})
	}

	return nil
}

func runDbPull(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Connect to database
	dbURL := cfg.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not configured")
	}

	db, err := migrations.ConnectDatabase(dbURL)
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer db.Close()

	// Detect provider
	provider := migrations.DetectProvider(dbURL)

	// Introspect database
	fmt.Println("Introspecting database...")
	dbSchema, err := migrations.IntrospectDatabase(db, provider)
	if err != nil {
		return fmt.Errorf("error introspecting database: %w", err)
	}

	// Generate schema from database
	fmt.Println("Generating schema.prisma...")
	schema, err := migrations.GenerateSchemaFromDatabase(dbSchema, provider, db)
	if err != nil {
		return fmt.Errorf("error generating schema: %w", err)
	}

	// Read existing schema to preserve datasource, generator, and comments
	schemaPath := getSchemaPath()
	var headerSection string

	if existingData, err := os.ReadFile(schemaPath); err == nil {
		// Extract header (comments, datasource, generator) and models/enums separately
		headerSection, _ = extractHeaderAndBody(string(existingData))

		// Parse existing schema to preserve datasource and generator structure
		existingSchema, _, parseErr := parser.ParseFile(schemaPath)
		if parseErr == nil && existingSchema != nil {
			// Preserve datasource and generator from existing schema
			schema.Datasources = existingSchema.Datasources
			schema.Generators = existingSchema.Generators
		}
	}

	// Create default datasource and generator if they don't exist
	if len(schema.Datasources) == 0 {
		schema.Datasources = []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{Name: "provider", Value: provider},
				},
			},
		}
	}
	if len(schema.Generators) == 0 {
		schema.Generators = []*parser.Generator{
			{
				Name: "client",
				Fields: []*parser.Field{
					{Name: "provider", Value: "prisma-client-go"},
				},
			},
		}
	}

	// If headerSection is empty, format datasource and generator for header
	if headerSection == "" {
		var headerBuilder strings.Builder
		for _, ds := range schema.Datasources {
			headerBuilder.WriteString(formatter.FormatDatasource(ds))
			headerBuilder.WriteString("\n")
		}
		for _, gen := range schema.Generators {
			headerBuilder.WriteString(formatter.FormatGenerator(gen))
			headerBuilder.WriteString("\n")
		}
		headerSection = strings.TrimSpace(headerBuilder.String())
	}

	// Format only models and enums from the new schema
	modelsAndEnumsFormatted := formatModelsAndEnums(schema)

	// Combine header with new models/enums
	finalContent := headerSection
	if headerSection != "" {
		headerSection = strings.TrimRight(headerSection, "\n")
		if !strings.HasSuffix(headerSection, "\n") {
			finalContent += "\n"
		}
		finalContent += "\n"
	}
	finalContent += modelsAndEnumsFormatted

	// Write schema.prisma
	if err := os.WriteFile(schemaPath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("error writing schema.prisma: %w", err)
	}

	// Format the schema automatically (same as Ctrl+S in editor)
	// Note: This re-parses the file, so we need to ensure the parser correctly handles
	// index column names and function arguments
	// TEMPORARILY DISABLED TO DEBUG
	/*
		if err := formatSchemaFile(schemaPath); err != nil {
			// Don't fail if formatting has issues, just warn
			fmt.Printf("Warning: Could not format schema: %v\n", err)
		}
	*/

	fmt.Printf("Schema generated successfully: %s\n", schemaPath)
	fmt.Printf("  %d model(s) found\n", len(schema.Models))
	if len(schema.Enums) > 0 {
		fmt.Printf("  %d enum(s) found\n", len(schema.Enums))
	}

	return nil
}

// formatSchemaFile formats a schema file in place
func formatSchemaFile(schemaPath string) error {
	// Parse schema
	schema, errors, err := parser.ParseFile(schemaPath)
	if err != nil || len(errors) > 0 {
		return fmt.Errorf("error parsing schema for formatting: %w", err)
	}

	if schema == nil {
		return fmt.Errorf("schema is nil after parsing")
	}

	// Format
	formatted := formatter.FormatSchema(schema)
	if formatted == "" {
		return fmt.Errorf("formatting returned empty string")
	}

	// Write back - no need to re-parse and re-format
	// The formatted output is already correct
	if err := os.WriteFile(schemaPath, []byte(formatted), 0644); err != nil {
		return fmt.Errorf("error writing formatted file: %w", err)
	}

	return nil
}

func runDbSeed(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if cfg.Migrations == nil || cfg.Migrations.Seed == "" {
		return fmt.Errorf("seed not configured in prisma.conf")
	}

	fmt.Println("Running seed...")
	if err := migrations.ExecuteSeed(cfg.Migrations.Seed); err != nil {
		return fmt.Errorf("error running seed: %w", err)
	}

	fmt.Println("Seed executed successfully!")
	return nil
}

func runDbExecute(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fileFlag := dbExecuteFileFlag
	stdinFlag := dbExecuteStdinFlag

	var sql string

	if fileFlag != "" {
		// Read from file
		data, err := os.ReadFile(fileFlag)
		if err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}
		sql = string(data)
	} else if stdinFlag != "" {
		// Read from stdin
		sql = stdinFlag
	} else if len(args) > 0 {
		// SQL as argument
		sql = strings.Join(args, " ")
	} else {
		// Read from interactive stdin
		fmt.Println("Enter SQL (Ctrl+D to finish):")
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading stdin: %w", err)
		}
		sql = strings.Join(lines, "\n")
	}

	if sql == "" {
		return fmt.Errorf("no SQL provided")
	}

	// Connect to database
	dbURL := cfg.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not configured")
	}

	db, err := migrations.ConnectDatabase(dbURL)
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer db.Close()

	// Execute SQL
	fmt.Println("Executing SQL...")
	statements := migrations.SplitSQLStatements(sql)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		result, err := db.Exec(stmt)
		if err != nil {
			return fmt.Errorf("error executing SQL: %w\nSQL: %s", err, stmt)
		}
		rowsAffected, err := result.RowsAffected()
		if err == nil {
			fmt.Printf("Executed: %d row(s) affected\n", rowsAffected)
		} else {
			fmt.Println("Executed successfully")
		}
	}

	fmt.Println("SQL executed successfully!")
	return nil
}

// extractHeaderAndBody splits the schema file into header (comments, datasource, generator) and body (models, enums)
func extractHeaderAndBody(content string) (header, body string) {
	lines := strings.Split(content, "\n")
	var headerLines []string
	var bodyLines []string

	modelRegex := regexp.MustCompile(`^\s*model\s+`)
	enumRegex := regexp.MustCompile(`^\s*enum\s+`)
	datasourceRegex := regexp.MustCompile(`^\s*datasource\s+`)
	generatorRegex := regexp.MustCompile(`^\s*generator\s+`)
	commentRegex := regexp.MustCompile(`^\s*//`)

	inDatasource := false
	inGenerator := false
	braceCount := 0
	foundFirstModelOrEnum := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if we've reached models/enums
		if modelRegex.MatchString(trimmed) || enumRegex.MatchString(trimmed) {
			foundFirstModelOrEnum = true
			if !inDatasource && !inGenerator {
				bodyLines = append(bodyLines, line)
				continue
			}
		}

		// Track datasource block
		if datasourceRegex.MatchString(trimmed) {
			inDatasource = true
			braceCount = strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			headerLines = append(headerLines, line)
			continue
		} else if inDatasource {
			braceCount += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			headerLines = append(headerLines, line)
			if braceCount == 0 {
				inDatasource = false
			}
			continue
		}

		// Track generator block
		if generatorRegex.MatchString(trimmed) {
			inGenerator = true
			braceCount = strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			headerLines = append(headerLines, line)
			continue
		} else if inGenerator {
			braceCount += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			headerLines = append(headerLines, line)
			if braceCount == 0 {
				inGenerator = false
			}
			continue
		}

		// Include comments and empty lines in header if before first model/enum
		if !foundFirstModelOrEnum {
			if commentRegex.MatchString(trimmed) || trimmed == "" {
				headerLines = append(headerLines, line)
			}
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	header = strings.Join(headerLines, "\n")
	header = strings.TrimRight(header, "\n")
	body = strings.Join(bodyLines, "\n")
	return header, body
}

// formatModelsAndEnums formats only models and enums from the schema
func formatModelsAndEnums(schema *parser.Schema) string {
	var result strings.Builder

	models := schema.Models
	enums := schema.Enums

	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})
	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Name < enums[j].Name
	})

	for _, model := range models {
		// Use FormatModelWithSchema to pass the schema context
		// This is needed so enum values in @default can be detected correctly
		result.WriteString(formatter.FormatModelWithSchema(model, schema))
		result.WriteString("\n")
	}

	for _, enum := range enums {
		result.WriteString(formatter.FormatEnum(enum))
		result.WriteString("\n")
	}

	return strings.TrimSpace(result.String())
}
