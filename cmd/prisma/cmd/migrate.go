package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

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

// normalizeMigrationName normalizes a migration name by:
// - Converting to lowercase
// - Trimming whitespace
// - Replacing spaces with underscores
// - Removing invalid special characters (keeps only letters, numbers, and underscores)
// - Removing consecutive duplicate underscores
// - Removing leading and trailing underscores
// - Ensuring the result is not empty
func normalizeMigrationName(name string) string {
	// Trim whitespace
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces and hyphens with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Remove invalid characters (keep only letters, numbers, and underscores)
	var builder strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			builder.WriteRune(r)
		}
	}
	name = builder.String()

	// Remove consecutive duplicate underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// Remove leading and trailing underscores
	name = strings.Trim(name, "_")

	// If empty after normalization, return a default value
	if name == "" {
		return "migration"
	}

	return name
}

func runMigrateDev(args []string) error {
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
	fmt.Printf("%s\n", Info(fmt.Sprintf("Loaded Prisma config from %s.", configPath)))

	schemaPath := getSchemaPath()
	fmt.Printf("%s\n", Info(fmt.Sprintf("Prisma schema loaded from %s", schemaPath)))

	// Get database URL and parse info
	dbURL := cfg.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not configured")
	}

	dbInfo := parseDatabaseURL(dbURL)
	fmt.Printf("%s\n", Info(fmt.Sprintf("Datasource \"db\": %s database \"%s\", schema \"%s\" at \"%s\"",
		dbInfo.Provider, dbInfo.Database, dbInfo.Schema, dbInfo.Host)))

	// Parse schema
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

	// Step 1: Run devDiagnostic (equivalent to Prisma's devDiagnostic)
	// This checks for drift, modified migrations, and missing migrations
	devDiagnostic, err := migrations.DevDiagnostic(manager, db, schema, provider)
	if err != nil {
		return fmt.Errorf("error running diagnostic: %w", err)
	}

	// Step 2: If diagnostic says we need to reset, show message and exit
	if devDiagnostic.Action.Tag == "reset" {
		fmt.Println()
		fmt.Println(devDiagnostic.Action.Reason)
		fmt.Println()

		// Build reset message based on provider
		var resetMessage string
		if dbInfo.Provider == "PostgreSQL" || dbInfo.Provider == "SQL Server" {
			if dbInfo.Schema != "" {
				resetMessage = fmt.Sprintf("We need to reset the \"%s\" schema", dbInfo.Schema)
			} else {
				resetMessage = "We need to reset the database schema"
			}
		} else {
			resetMessage = fmt.Sprintf("We need to reset the %s database \"%s\"", dbInfo.Provider, dbInfo.Database)
		}

		if dbInfo.Host != "" {
			resetMessage += fmt.Sprintf(" at \"%s\"", dbInfo.Host)
		}

		fmt.Println(resetMessage)
		fmt.Println()
		fmt.Println("You may use prisma migrate reset to drop the development database.")
		fmt.Println("All data will be lost.")

		// Exit with code 130 (SIGINT) to match Prisma's behavior
		os.Exit(130)
	}

	// Step 3: Apply pending migrations (equivalent to migrate.applyMigrations())
	// This applies any migrations that exist locally but haven't been applied yet
	appliedMigrations, err := manager.GetPendingMigrations()
	if err != nil {
		return fmt.Errorf("error getting pending migrations: %w", err)
	}

	if len(appliedMigrations) > 0 {
		// Apply each pending migration
		for _, migration := range appliedMigrations {
			if err := manager.ApplyMigration(migration); err != nil {
				return fmt.Errorf("error applying migration %s: %w", migration.Name, err)
			}
		}

		// Show applied migrations (matching Prisma's format)
		fmt.Println()
		fmt.Println("The following migration(s) have been applied:")
		fmt.Println()
		fmt.Println("migrations/")
		for _, migration := range appliedMigrations {
			fmt.Printf("  └─ %s/\n", MigrationName(migration.Name+"/"))
			fmt.Printf("    └─ migration.sql\n")
		}
		fmt.Println()
	}

	// Step 4: Check for pending changes (schema.prisma vs current database)
	// This is equivalent to evaluateDataLoss + createMigration
	// Introspect database to detect incremental changes
	dbSchema, err := migrations.IntrospectDatabase(db, provider)
	if err != nil {
		// If introspection fails, we can't proceed safely
		return fmt.Errorf("error introspecting database: %w", err)
	}

	// Compare schema with database to detect incremental changes
	diff, err := migrations.CompareSchema(schema, dbSchema, provider)
	if err != nil {
		return fmt.Errorf("error comparing schema: %w", err)
	}

	// Check if there are changes
	// Note: ForeignKeysToCreate is not included because foreign keys are usually
	// created as part of table creation/alteration, not as separate operations
	hasChanges := len(diff.TablesToCreate) > 0 ||
		len(diff.TablesToAlter) > 0 ||
		len(diff.TablesToDrop) > 0 ||
		len(diff.IndexesToCreate) > 0 ||
		len(diff.IndexesToDrop) > 0

	// Step 5: If no changes, show sync message and return
	if !hasChanges {
		if len(appliedMigrations) > 0 {
			fmt.Println(Success("Your database is now in sync with your schema."))
		} else {
			fmt.Println()
			fmt.Println("Already in sync, no schema change or pending migration was found.")
		}
		fmt.Println()
		return nil
	}

	// Step 6: There are changes, so we need to create a migration
	// Generate migration SQL first to check if there are real changes
	sql, err := migrations.GenerateMigrationSQL(diff, provider)
	if err != nil {
		return fmt.Errorf("error generating migration SQL: %w", err)
	}

	// If SQL is empty, there are no real changes (only relation differences)
	if sql == "" || strings.TrimSpace(sql) == "" {
		fmt.Println()
		fmt.Println("Already in sync, no schema change or pending migration was found.")
		fmt.Println()
		return nil
	}

	// Get migration name
	migrationName := ""
	if len(args) > 0 {
		migrationName = args[0]
	} else {
		fmt.Println()
		fmt.Print(Prompt("? ") + PromptText("Enter a name for the new migration: › "))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		migrationName = strings.TrimSpace(input)
		if migrationName == "" {
			return fmt.Errorf("migration name is required")
		}
	}

	// Normalize migration name (convert spaces to underscores, lowercase, etc.)
	migrationName = normalizeMigrationName(migrationName)
	if migrationName == "" {
		return fmt.Errorf("migration name is required")
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

	// Normal text (no color) for migration created message
	fmt.Printf("Migration created: %s\n", migrationDirName)

	// Ensure migration_lock.toml exists (compatible with Prisma official)
	// Note: lockfile goes in migrations directory root, not in individual migration directories
	// This is created AFTER the first migration is created (not before)
	if err := migrations.EnsureMigrationLockfile(cfg.GetMigrationsPath(), provider); err != nil {
		return fmt.Errorf("error ensuring migration_lock.toml: %w", err)
	}

	// Apply migration
	migration := &migrations.Migration{
		Name: migrationDirName,
		Path: migrationPath,
		SQL:  sql,
	}

	// Apply the newly created migration
	fmt.Println("Applying migration...")
	if err := manager.ApplyMigration(migration); err != nil {
		return fmt.Errorf("error applying migration: %w", err)
	}

	// Show success message (matching Prisma's format)
	fmt.Println()
	fmt.Println("The following migration(s) have been created and applied from new schema changes:")
	fmt.Println()
	fmt.Println("migrations/")
	fmt.Printf("  └─ %s/\n", MigrationName(migrationDirName+"/"))
	fmt.Printf("    └─ migration.sql\n")
	fmt.Println()
	fmt.Println(Success("Your database is now in sync with your schema."))

	// Run generate automatically
	// Normal text (no color) for progress message
	fmt.Println()
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
	fmt.Printf("%s\n", Info(fmt.Sprintf("Loaded Prisma config from %s.", configPath)))

	schemaPath := getSchemaPath()
	fmt.Printf("%s\n", Info(fmt.Sprintf("Prisma schema loaded from %s", schemaPath)))

	// Get database URL and parse info
	dbURL := cfg.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not configured")
	}

	dbInfo := parseDatabaseURL(dbURL)
	fmt.Printf("%s\n", Info(fmt.Sprintf("Datasource \"db\": %s database \"%s\", schema \"%s\" at \"%s\"",
		dbInfo.Provider, dbInfo.Database, dbInfo.Schema, dbInfo.Host)))

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
		fmt.Printf("Applying migration `%s`\n", MigrationName(migration.Name))
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
		fmt.Printf("  └─ %s/\n", MigrationName(migration.Name+"/"))
		fmt.Printf("    └─ migration.sql\n")
	}
	fmt.Println()
	fmt.Println(Success("All migrations have been successfully applied."))
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
	fmt.Printf("%s\n", Info(fmt.Sprintf("Loaded Prisma config from %s.", configPath)))

	schemaPath := getSchemaPath()
	fmt.Printf("%s\n", Info(fmt.Sprintf("Prisma schema loaded from %s", schemaPath)))

	// Get database URL and parse info
	dbURL := cfg.GetDatabaseURL()
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not configured")
	}

	dbInfo := parseDatabaseURL(dbURL)
	fmt.Printf("%s\n", Info(fmt.Sprintf("Datasource \"db\": %s database \"%s\", schema \"%s\" at \"%s\"",
		dbInfo.Provider, dbInfo.Database, dbInfo.Schema, dbInfo.Host)))

	// Confirm destructive action
	fmt.Printf("\n%s Are you sure you want to reset your database? %s %s",
		Prompt("?"), Warning("All data will be lost."), PromptText("› (y/N)"))
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
		fmt.Printf("Applying migration `%s`\n", MigrationName(migration.Name))
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
	fmt.Println(Success("Database reset successful"))
	fmt.Println()

	if len(local) > 0 {
		fmt.Println("The following migration(s) have been applied:")
		fmt.Println()
		fmt.Println("migrations/")
		for _, migration := range local {
			fmt.Printf("  └─ %s/\n", MigrationName(migration.Name))
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

	fmt.Println(Info("Migration Status"))
	fmt.Println()
	fmt.Printf("%s\n", Info(fmt.Sprintf("Local migrations: %d", len(local))))
	fmt.Printf("%s\n\n", Info(fmt.Sprintf("Applied migrations: %d", len(applied))))

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
					fmt.Println(Warning("Warning: Divergences detected between schema.prisma and database:"))
					if len(diff.TablesToCreate) > 0 {
						fmt.Printf("%s\n", Info(fmt.Sprintf("  - %d table(s) to create", len(diff.TablesToCreate))))
					}
					if len(diff.TablesToAlter) > 0 {
						fmt.Printf("%s\n", Info(fmt.Sprintf("  - %d table(s) to alter", len(diff.TablesToAlter))))
					}
					if len(diff.TablesToDrop) > 0 {
						fmt.Printf("%s\n", Info(fmt.Sprintf("  - %d table(s) to remove", len(diff.TablesToDrop))))
					}
					fmt.Println()
				}
			}
		}
	}

	if len(local) == 0 {
		fmt.Println(Warning("Warning: No local migrations found"))
		return nil
	}

	fmt.Println(Info("Migrations:"))
	for _, migration := range local {
		status := "Pending"
		if appliedMap[migration.Name] {
			status = "Applied"
		}
		fmt.Printf("%s\n", Info(fmt.Sprintf("  %s %s", status, migration.Name)))
	}

	pending := len(local) - len(applied)
	if pending > 0 {
		fmt.Printf("\n%s\n", Warning(fmt.Sprintf("Warning: %d pending migration(s)", pending)))
	} else {
		fmt.Printf("\n%s\n", Info("All migrations are applied"))
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
