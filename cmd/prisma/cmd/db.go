package cmd

import (
	"bufio"
	"fmt"
	"os"
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
	schema, err := migrations.GenerateSchemaFromDatabase(dbSchema, provider)
	if err != nil {
		return fmt.Errorf("error generating schema: %w", err)
	}

	// Format schema
	formatted := formatter.FormatSchema(schema)

	// Write schema.prisma
	schemaPath := getSchemaPath()
	if err := os.WriteFile(schemaPath, []byte(formatted), 0644); err != nil {
		return fmt.Errorf("error writing schema.prisma: %w", err)
	}

	fmt.Printf("Schema generated successfully: %s\n", schemaPath)
	fmt.Printf("  %d model(s) found\n", len(schema.Models))

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
