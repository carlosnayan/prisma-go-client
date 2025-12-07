package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/migrations"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// DatabaseInfo contains parsed database connection information
type DatabaseInfo struct {
	Provider string // postgresql, mysql, sqlite
	Database string // database name
	Schema   string // schema name (usually "public" for postgres)
	Host     string // host:port
}

// parseDatabaseURL parses a database URL and extracts connection information
func parseDatabaseURL(dbURL string) *DatabaseInfo {
	info := &DatabaseInfo{
		Schema: "public", // default
	}

	// Parse URL
	u, err := url.Parse(dbURL)
	if err != nil {
		return info
	}

	// Get provider from scheme
	switch u.Scheme {
	case "postgresql", "postgres":
		info.Provider = "PostgreSQL"
	case "mysql":
		info.Provider = "MySQL"
	case "sqlite", "file":
		info.Provider = "SQLite"
	default:
		info.Provider = u.Scheme
	}

	// Get host
	info.Host = u.Host

	// Get database name from path
	if u.Path != "" {
		info.Database = strings.TrimPrefix(u.Path, "/")
	}

	// Get schema from query params if present
	if schema := u.Query().Get("schema"); schema != "" {
		info.Schema = schema
	}

	return info
}

var (
	migrateResolveAppliedFlag    string
	migrateResolveRolledBackFlag string
)

var migrateCmd = &cli.Command{
	Name:  "migrate",
	Short: "Manage database migrations",
	Long: `Commands to manage migrations:
  - dev: Create and apply migrations in development
  - deploy: Apply pending migrations in production
  - reset: Reset database and reapply all migrations
  - status: Check migration status`,
	Subcommands: []*cli.Command{
		migrateDevCmd,
		migrateDeployCmd,
		migrateResetCmd,
		migrateStatusCmd,
		migrateResolveCmd,
		migrateDiffCmd,
	},
}

var migrateDevCmd = &cli.Command{
	Name:  "dev",
	Short: "Create and apply migration in development",
	Long: `Creates a new migration based on schema.prisma changes
and applies it to the database. If a name is not provided,
it will be requested interactively.`,
	Usage: "migrate dev [name]",
	Run:   runMigrateDev,
}

var migrateDeployCmd = &cli.Command{
	Name:  "deploy",
	Short: "Apply pending migrations in production",
	Long:  `Applies all pending migrations to the database. Non-interactive mode for CI/CD.`,
	Run:   runMigrateDeploy,
}

var migrateResetCmd = &cli.Command{
	Name:  "reset",
	Short: "Reset database and reapply migrations",
	Long: `Resets the database completely:
  - Drops the database
  - Recreates the database
  - Applies all migrations
  - Runs seed (if configured)`,
	Run: runMigrateReset,
}

var migrateStatusCmd = &cli.Command{
	Name:  "status",
	Short: "Check migration status",
	Long:  `Lists applied and pending migrations, detecting divergences between schema and database.`,
	Run:   runMigrateStatus,
}

var migrateResolveCmd = &cli.Command{
	Name:  "resolve",
	Short: "Resolve migration status",
	Long: `Manually marks a migration as applied or not applied.
Useful for resolving conflicts or fixing inconsistent state.`,
	Flags: []*cli.Flag{
		{
			Name:  "applied",
			Usage: "Mark migration as applied",
			Value: &migrateResolveAppliedFlag,
		},
		{
			Name:  "rolled-back",
			Usage: "Mark migration as not applied (rolled back)",
			Value: &migrateResolveRolledBackFlag,
		},
	},
	Run: runMigrateResolve,
}

func runMigrateDev(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Get migration name
	migrationName := ""
	if len(args) > 0 {
		migrationName = args[0]
	} else {
		fmt.Print("Migration name: ")
		_, _ = fmt.Scanln(&migrationName)
		if migrationName == "" {
			return fmt.Errorf("migration name is required")
		}
	}

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

	// Create migration manager
	manager, err := migrations.NewManager(cfg, db)
	if err != nil {
		return fmt.Errorf("error creating migration manager: %w", err)
	}

	// Get provider
	provider := migrations.GetProviderFromSchema(schema)

	var sql string

	// Introspect database to detect incremental changes
	dbSchema, err := migrations.IntrospectDatabase(db, provider)
	if err != nil {
		// If introspection fails, create everything from scratch
		fmt.Println("Warning: Could not introspect database, creating full migration")
		diff, err := migrations.SchemaToSQL(schema, provider)
		if err != nil {
			return fmt.Errorf("error generating SQL: %w", err)
		}
		sql, err = migrations.GenerateMigrationSQL(diff, provider)
		if err != nil {
			return fmt.Errorf("error generating migration SQL: %w", err)
		}
	} else {
		// Compare schema with database to detect incremental changes
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
			fmt.Println("No changes detected in schema")
			return nil
		}

		// Generate migration SQL
		sql, err = migrations.GenerateMigrationSQL(diff, provider)
		if err != nil {
			return fmt.Errorf("error generating migration SQL: %w", err)
		}

		if sql == "" || strings.TrimSpace(sql) == "" {
			fmt.Println("No changes detected in schema")
			return nil
		}
	}

	// Create migration directory
	timestamp := time.Now().Format("20060102150405")
	migrationDirName := fmt.Sprintf("%s_%s", timestamp, migrationName)
	migrationsPath := cfg.GetMigrationsPath()
	migrationPath := filepath.Join(migrationsPath, migrationDirName)

	if err := os.MkdirAll(migrationPath, 0755); err != nil {
		return fmt.Errorf("error creating migration directory: %w", err)
	}

	// Write migration.sql file
	sqlPath := filepath.Join(migrationPath, "migration.sql")
	if err := os.WriteFile(sqlPath, []byte(sql), 0644); err != nil {
		return fmt.Errorf("error writing migration.sql: %w", err)
	}

	fmt.Printf("Migration created: %s\n", migrationDirName)

	// Apply migration
	migration := &migrations.Migration{
		Name: migrationDirName,
		Path: migrationPath,
		SQL:  sql,
	}

	fmt.Println("Applying migration...")
	if err := manager.ApplyMigration(migration); err != nil {
		return fmt.Errorf("error applying migration: %w", err)
	}

	fmt.Println("Migration applied successfully!")

	// Run generate automatically
	fmt.Println("Generating code...")
	return runGenerate([]string{})
}

func runMigrateDeploy(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Show config and schema info (similar to Prisma)
	configPath := getConfigPath()
	fmt.Println()
	fmt.Printf("Loaded Prisma config from %s.\n", configPath)

	schemaPath := getSchemaPath()
	fmt.Printf("Prisma schema loaded from %s\n", schemaPath)

	// Get database URL and parse info
	dbURL := cfg.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not configured")
	}

	dbInfo := parseDatabaseURL(dbURL)
	fmt.Printf("Datasource \"db\": %s database \"%s\", schema \"%s\" at \"%s\"\n",
		dbInfo.Provider, dbInfo.Database, dbInfo.Schema, dbInfo.Host)

	// Connect to database with better error message
	db, err := migrations.ConnectDatabase(dbURL)
	if err != nil {
		fmt.Println()
		fmt.Printf("Error: P1001: Can't reach database server at `%s`\n\n", dbInfo.Host)
		fmt.Printf("Please make sure your database server is running at `%s`.\n", dbInfo.Host)
		return err
	}
	defer db.Close()

	// Create migration manager
	manager, err := migrations.NewManager(cfg, db)
	if err != nil {
		return fmt.Errorf("error creating migration manager: %w", err)
	}

	// Get pending migrations
	pending, err := manager.GetPendingMigrations()
	if err != nil {
		return fmt.Errorf("error getting pending migrations: %w", err)
	}

	if len(pending) == 0 {
		fmt.Println()
		fmt.Println("No pending migrations to apply.")
		return nil
	}

	fmt.Printf("\n%d migration(s) found in prisma/migrations\n\n", len(pending))

	// Apply each migration
	for _, migration := range pending {
		fmt.Printf("Applying migration `%s`\n", migration.Name)
		if err := manager.ApplyMigration(migration); err != nil {
			return fmt.Errorf("error applying migration %s: %w", migration.Name, err)
		}
	}

	// Show success message and migrations tree
	fmt.Println()
	fmt.Println("The following migration(s) have been applied:")
	fmt.Println()
	fmt.Println("migrations/")
	for _, migration := range pending {
		fmt.Printf("  └─ %s/\n", migration.Name)
		fmt.Printf("    └─ migration.sql\n")
	}
	fmt.Println()
	fmt.Println("All migrations have been successfully applied.")
	return nil
}

func runMigrateReset(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Show config and schema info (similar to Prisma)
	configPath := getConfigPath()
	fmt.Println()
	fmt.Printf("Loaded Prisma config from %s.\n", configPath)

	schemaPath := getSchemaPath()
	fmt.Printf("Prisma schema loaded from %s\n", schemaPath)

	// Get database URL and parse info
	dbURL := cfg.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not configured")
	}

	dbInfo := parseDatabaseURL(dbURL)
	fmt.Printf("Datasource \"db\": %s database \"%s\", schema \"%s\" at \"%s\"\n",
		dbInfo.Provider, dbInfo.Database, dbInfo.Schema, dbInfo.Host)

	// Confirm destructive action
	fmt.Print("\n✔ Are you sure you want to reset your database? All data will be lost. › (y/N)")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "yes" && confirm != "y" {
		return nil
	}

	// Connect to database with better error message
	db, err := migrations.ConnectDatabase(dbURL)
	if err != nil {
		fmt.Println()
		fmt.Printf("Error: P1001: Can't reach database server at `%s`\n\n", dbInfo.Host)
		
		// Show parsed URL info for debugging
		fmt.Printf("Connection details:\n")
		fmt.Printf("  Provider: %s\n", dbInfo.Provider)
		fmt.Printf("  Host: %s\n", dbInfo.Host)
		fmt.Printf("  Database: %s\n", dbInfo.Database)
		fmt.Printf("  Schema: %s\n", dbInfo.Schema)
		
		// Show masked URL (hide password)
		maskedURL := dbURL
		if strings.Contains(maskedURL, "@") {
			parts := strings.Split(maskedURL, "@")
			if len(parts) > 0 && strings.Contains(parts[0], ":") {
				authParts := strings.Split(parts[0], ":")
				if len(authParts) >= 3 {
					authParts[2] = "***"
					maskedURL = strings.Join(authParts, ":") + "@" + strings.Join(parts[1:], "@")
				} else if len(authParts) >= 2 {
					authParts[1] = "***"
					maskedURL = strings.Join(authParts, ":") + "@" + strings.Join(parts[1:], "@")
				}
			}
		}
		fmt.Printf("  URL: %s\n\n", maskedURL)
		
		fmt.Printf("Please make sure your database server is running at `%s`.\n", dbInfo.Host)
		fmt.Printf("\nTroubleshooting:\n")
		fmt.Printf("  1. Verify PostgreSQL is running: pg_isready -h localhost -p 5432\n")
		fmt.Printf("  2. Check if the host and port are correct: %s\n", dbInfo.Host)
		fmt.Printf("  3. Verify credentials (user: %s)\n", func() string {
			if strings.Contains(dbURL, "@") {
				parts := strings.Split(strings.Split(dbURL, "@")[0], ":")
				if len(parts) > 0 {
					return strings.TrimPrefix(parts[0], "postgresql://")
				}
			}
			return "unknown"
		}())
		fmt.Printf("  4. Check firewall/network settings\n")
		fmt.Printf("  5. Verify the database '%s' exists\n", dbInfo.Database)
		return err
	}
	defer db.Close()

	// Drop all tables (reset the schema)
	_, err = db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	if err != nil {
		return fmt.Errorf("error clearing database: %w", err)
	}

	// Create migration manager
	manager, err := migrations.NewManager(cfg, db)
	if err != nil {
		return fmt.Errorf("error creating migration manager: %w", err)
	}

	// Get all local migrations
	local, err := manager.GetLocalMigrations()
	if err != nil {
		return fmt.Errorf("error getting local migrations: %w", err)
	}

	// Apply all migrations
	for _, migration := range local {
		fmt.Printf("Applying migration `%s`\n", migration.Name)
		if err := manager.ApplyMigration(migration); err != nil {
			return fmt.Errorf("error applying migration %s: %w", migration.Name, err)
		}
	}

	// Execute seed if configured
	if cfg.Migrations != nil && cfg.Migrations.Seed != "" {
		fmt.Println()
		fmt.Println("Running seed...")
		if err := migrations.ExecuteSeed(cfg.Migrations.Seed); err != nil {
			return fmt.Errorf("error running seed: %w", err)
		}
	}

	// Show success message and migrations tree
	fmt.Println()
	fmt.Println("Database reset successful")
	fmt.Println()

	if len(local) > 0 {
		fmt.Println("The following migration(s) have been applied:")
		fmt.Println()
		fmt.Println("migrations/")
		for _, migration := range local {
			fmt.Printf("  └─ %s/\n", migration.Name)
			fmt.Printf("    └─ migration.sql\n")
		}
		fmt.Println()
	}

	return nil
}

func runMigrateStatus(args []string) error {
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

	// Create migration manager
	manager, err := migrations.NewManager(cfg, db)
	if err != nil {
		return fmt.Errorf("error creating migration manager: %w", err)
	}

	// Get local and applied migrations
	local, err := manager.GetLocalMigrations()
	if err != nil {
		return fmt.Errorf("error getting local migrations: %w", err)
	}

	applied, err := manager.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("error getting applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, name := range applied {
		appliedMap[name] = true
	}

	fmt.Println("Migration Status")
	fmt.Println()
	fmt.Printf("Local migrations: %d\n", len(local))
	fmt.Printf("Applied migrations: %d\n\n", len(applied))

	// Check divergences between schema and database
	schemaPath := getSchemaPath()
	schema, _, err := parser.ParseFile(schemaPath)
	if err == nil {
		provider := migrations.GetProviderFromSchema(schema)
		dbSchema, err := migrations.IntrospectDatabase(db, provider)
		if err == nil {
			diff, err := migrations.CompareSchema(schema, dbSchema, provider)
			if err == nil {
				hasDivergences := len(diff.TablesToCreate) > 0 ||
					len(diff.TablesToAlter) > 0 ||
					len(diff.TablesToDrop) > 0 ||
					len(diff.IndexesToCreate) > 0 ||
					len(diff.IndexesToDrop) > 0

				if hasDivergences {
					fmt.Println("Warning: Divergences detected between schema.prisma and database:")
					if len(diff.TablesToCreate) > 0 {
						fmt.Printf("  - %d table(s) to create\n", len(diff.TablesToCreate))
					}
					if len(diff.TablesToAlter) > 0 {
						fmt.Printf("  - %d table(s) to alter\n", len(diff.TablesToAlter))
					}
					if len(diff.TablesToDrop) > 0 {
						fmt.Printf("  - %d table(s) to remove\n", len(diff.TablesToDrop))
					}
					fmt.Println()
				}
			}
		}
	}

	if len(local) == 0 {
		fmt.Println("Warning: No local migrations found")
		return nil
	}

	fmt.Println("Migrations:")
	for _, migration := range local {
		status := "Pending"
		if appliedMap[migration.Name] {
			status = "Applied"
		}
		fmt.Printf("  %s %s\n", status, migration.Name)
	}

	pending := len(local) - len(applied)
	if pending > 0 {
		fmt.Printf("\nWarning: %d pending migration(s)\n", pending)
	} else {
		fmt.Println("\nAll migrations are applied")
	}

	return nil
}

func runMigrateResolve(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	applied := migrateResolveAppliedFlag
	rolledBack := migrateResolveRolledBackFlag

	if applied == "" && rolledBack == "" {
		return fmt.Errorf("specify --applied or --rolled-back with migration name")
	}

	if applied != "" && rolledBack != "" {
		return fmt.Errorf("specify only --applied or --rolled-back, not both")
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

	// Create migration manager
	manager, err := migrations.NewManager(cfg, db)
	if err != nil {
		return fmt.Errorf("error creating migration manager: %w", err)
	}

	if applied != "" {
		// Mark as applied
		if err := manager.MarkMigrationAsApplied(applied); err != nil {
			return fmt.Errorf("error marking migration as applied: %w", err)
		}
		fmt.Printf("Migration '%s' marked as applied\n", applied)
	} else {
		// Mark as rolled back
		if err := manager.MarkMigrationAsRolledBack(rolledBack); err != nil {
			return fmt.Errorf("error marking migration as rolled back: %w", err)
		}
		fmt.Printf("Migration '%s' marked as rolled back\n", rolledBack)
	}

	return nil
}
