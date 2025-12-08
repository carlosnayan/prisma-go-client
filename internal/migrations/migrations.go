package migrations

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/config"
	"github.com/google/uuid"
)

// Migration represents a migration
type Migration struct {
	Name      string    // Migration name (e.g., "20241219120000_add_users")
	Path      string    // Full path to the migration directory
	SQL       string    // Content of the migration.sql file
	AppliedAt time.Time // When it was applied (zero if not applied)
}

// Manager manages migrations
type Manager struct {
	config         *config.Config
	db             *sql.DB
	migrationsPath string
}

// NewManager creates a new migration manager
func NewManager(cfg *config.Config, db *sql.DB) (*Manager, error) {
	migrationsPath := cfg.GetMigrationsPath()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(migrationsPath, 0755); err != nil {
		return nil, fmt.Errorf("error creating migrations directory: %w", err)
	}

	return &Manager{
		config:         cfg,
		db:             db,
		migrationsPath: migrationsPath,
	}, nil
}

// EnsureMigrationsTable ensures that the _prisma_migrations table exists
// This must match Prisma's exact table structure for 100% compatibility
func (m *Manager) EnsureMigrationsTable() error {
	// Get provider to use correct SQL syntax
	provider := m.getProvider()

	var query string
	switch provider {
	case "postgresql", "postgres":
		// PostgreSQL: Use TIMESTAMPTZ for better timezone handling (Prisma uses this)
		query = `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id VARCHAR(36) PRIMARY KEY,
				checksum VARCHAR(64) NOT NULL,
				finished_at TIMESTAMPTZ,
				migration_name VARCHAR(255) NOT NULL,
				logs TEXT,
				rolled_back_at TIMESTAMPTZ,
				started_at TIMESTAMPTZ NOT NULL,
				applied_steps_count INTEGER NOT NULL DEFAULT 0
			)
		`
	case "mysql":
		// MySQL: Use DATETIME(3) for millisecond precision (matching Prisma)
		query = `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id VARCHAR(36) PRIMARY KEY,
				checksum VARCHAR(64) NOT NULL,
				finished_at DATETIME(3),
				migration_name VARCHAR(255) NOT NULL,
				logs TEXT,
				rolled_back_at DATETIME(3),
				started_at DATETIME(3) NOT NULL,
				applied_steps_count INTEGER NOT NULL DEFAULT 0
			)
		`
	case "sqlite":
		// SQLite: Use TEXT for timestamps (ISO 8601 format)
		query = `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id TEXT PRIMARY KEY,
				checksum TEXT NOT NULL,
				finished_at TEXT,
				migration_name TEXT NOT NULL,
				logs TEXT,
				rolled_back_at TEXT,
				started_at TEXT NOT NULL,
				applied_steps_count INTEGER NOT NULL DEFAULT 0
			)
		`
	default:
		// Default to PostgreSQL format
		query = `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id VARCHAR(36) PRIMARY KEY,
				checksum VARCHAR(64) NOT NULL,
				finished_at TIMESTAMPTZ,
				migration_name VARCHAR(255) NOT NULL,
				logs TEXT,
				rolled_back_at TIMESTAMPTZ,
				started_at TIMESTAMPTZ NOT NULL,
				applied_steps_count INTEGER NOT NULL DEFAULT 0
			)
		`
	}

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating _prisma_migrations table: %w", err)
	}

	return nil
}

// getProvider determines the database provider from the connection string
func (m *Manager) getProvider() string {
	if m.config == nil || m.config.Datasource == nil {
		return "postgresql" // Default
	}

	dbURL := m.config.GetDatabaseURL()
	if dbURL == "" {
		return "postgresql" // Default
	}

	// Parse URL to detect provider
	if strings.HasPrefix(dbURL, "postgresql://") || strings.HasPrefix(dbURL, "postgres://") {
		return "postgresql"
	} else if strings.HasPrefix(dbURL, "mysql://") {
		return "mysql"
	} else if strings.HasPrefix(dbURL, "sqlite://") || strings.HasPrefix(dbURL, "file:") {
		return "sqlite"
	}

	return "postgresql" // Default
}

// GetAppliedMigrations returns list of applied migrations
func (m *Manager) GetAppliedMigrations() ([]string, error) {
	if err := m.EnsureMigrationsTable(); err != nil {
		return nil, err
	}

	query := `SELECT migration_name FROM _prisma_migrations WHERE finished_at IS NOT NULL ORDER BY started_at`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("error reading migration: %w", err)
		}
		migrations = append(migrations, name)
	}

	return migrations, nil
}

// GetLocalMigrations returns list of local migrations (files)
func (m *Manager) GetLocalMigrations() ([]*Migration, error) {
	entries, err := os.ReadDir(m.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("error reading migrations directory: %w", err)
	}

	var migrations []*Migration
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if it's a migration directory (format: YYYYMMDDHHMMSS_name)
		// Skip migration_lock.toml directory if it exists
		if name == "migration_lock.toml" || !isValidMigrationName(name) {
			continue
		}

		migrationPath := filepath.Join(m.migrationsPath, name)
		sqlPath := filepath.Join(migrationPath, "migration.sql")

		sqlContent, err := os.ReadFile(sqlPath)
		if err != nil {
			// If migration.sql doesn't exist, skip
			continue
		}

		migrations = append(migrations, &Migration{
			Name: name,
			Path: migrationPath,
			SQL:  string(sqlContent),
		})
	}

	// Sort by name (which contains timestamp)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}

// GetModifiedMigrations returns migrations that have been modified after being applied
// This checks if the checksum of the local migration file differs from the stored checksum
func (m *Manager) GetModifiedMigrations() ([]string, error) {
	if err := m.EnsureMigrationsTable(); err != nil {
		return nil, err
	}

	local, err := m.GetLocalMigrations()
	if err != nil {
		return nil, err
	}

	// Get checksums from database
	query := `SELECT migration_name, checksum FROM _prisma_migrations WHERE finished_at IS NOT NULL`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying migration checksums: %w", err)
	}
	defer rows.Close()

	// Build map of migration name -> stored checksum
	storedChecksums := make(map[string]string)
	for rows.Next() {
		var name, checksum string
		if err := rows.Scan(&name, &checksum); err != nil {
			return nil, fmt.Errorf("error reading checksum: %w", err)
		}
		storedChecksums[name] = checksum
	}

	// Check each local migration
	var modified []string
	for _, migration := range local {
		storedChecksum, exists := storedChecksums[migration.Name]
		if !exists {
			continue // Not applied yet, skip
		}

		// Calculate current checksum
		currentChecksum := calculateChecksum(migration.SQL)

		// If checksums differ, migration was modified
		if storedChecksum != currentChecksum && storedChecksum != "manual" {
			modified = append(modified, migration.Name)
		}
	}

	return modified, nil
}

// isValidMigrationName checks if the migration name is in the correct format
func isValidMigrationName(name string) bool {
	// Format: YYYYMMDDHHMMSS_name (at least 14 digits + underscore + name)
	if len(name) < 16 {
		return false
	}

	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return false
	}

	// Check if the first part is numeric and has 14 digits
	if len(parts[0]) != 14 {
		return false
	}

	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// GetPendingMigrations returns pending migrations (local but not applied)
func (m *Manager) GetPendingMigrations() ([]*Migration, error) {
	local, err := m.GetLocalMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[string]bool)
	for _, name := range applied {
		appliedMap[name] = true
	}

	var pending []*Migration
	for _, migration := range local {
		if !appliedMap[migration.Name] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// GetMissingMigrations returns migrations that are applied in the database but missing from local directory
func (m *Manager) GetMissingMigrations() ([]string, error) {
	local, err := m.GetLocalMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	// Create a map of local migration names for quick lookup
	localMap := make(map[string]bool)
	for _, migration := range local {
		localMap[migration.Name] = true
	}

	// Find applied migrations that are not in local directory
	var missing []string
	for _, appliedName := range applied {
		if !localMap[appliedName] {
			missing = append(missing, appliedName)
		}
	}

	return missing, nil
}

// ApplyMigration applies a migration to the database
func (m *Manager) ApplyMigration(migration *Migration) error {
	if err := m.EnsureMigrationsTable(); err != nil {
		return err
	}

	// Start transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Register migration start
	migrationID := generateMigrationID()
	startedAt := time.Now()

	insertQuery := `
		INSERT INTO _prisma_migrations (id, checksum, migration_name, started_at, applied_steps_count)
		VALUES ($1, $2, $3, $4, 1)
	`

	checksum := calculateChecksum(migration.SQL)
	_, err = tx.Exec(insertQuery, migrationID, checksum, migration.Name, startedAt)
	if err != nil {
		return fmt.Errorf("error registering migration: %w", err)
	}

	// Execute migration SQL
	// Split by ; to execute multiple statements
	statements := SplitSQLStatements(migration.SQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("error executing migration %s: %w\nSQL: %s", migration.Name, err, stmt)
		}
	}

	// Mark as finished
	finishedAt := time.Now()
	updateQuery := `
		UPDATE _prisma_migrations 
		SET finished_at = $1, applied_steps_count = $2
		WHERE id = $3
	`
	_, err = tx.Exec(updateQuery, finishedAt, len(statements), migrationID)
	if err != nil {
		return fmt.Errorf("error finishing migration: %w", err)
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing migration: %w", err)
	}

	return nil
}

// generateMigrationID generates a unique UUID v4 for the migration
// This must match Prisma's format for compatibility
func generateMigrationID() string {
	return uuid.New().String()
}

// calculateChecksum calculates SHA256 checksum of the SQL content
// This must match exactly how Prisma calculates checksums for compatibility
// Prisma's Schema Engine calculates checksum on the raw file content with specific normalization:
// 1. Remove trailing whitespace from each line (spaces, tabs, carriage returns)
// 2. Keep line endings as \n (normalize to Unix line endings)
// 3. Calculate SHA256 of the normalized content
func calculateChecksum(sql string) string {
	// Normalize line endings to \n (Unix style)
	// Replace \r\n (Windows) and \r (old Mac) with \n
	normalized := strings.ReplaceAll(sql, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	// Split into lines and remove trailing whitespace from each line
	lines := strings.Split(normalized, "\n")
	normalizedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		// Remove trailing spaces, tabs, and carriage returns
		normalizedLine := strings.TrimRight(line, " \t\r")
		normalizedLines = append(normalizedLines, normalizedLine)
	}

	// Rejoin with \n (ensuring consistent line endings)
	normalizedSQL := strings.Join(normalizedLines, "\n")

	// Calculate SHA256 hash (64 hex characters)
	hash := sha256.Sum256([]byte(normalizedSQL))
	return hex.EncodeToString(hash[:])
}

// SplitSQLStatements splits SQL into individual statements
func SplitSQLStatements(sql string) []string {
	// Split by ; but respect strings and comments
	var statements []string
	var current strings.Builder
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		if !inString {
			if ch == '\'' || ch == '"' {
				inString = true
				stringChar = ch
				current.WriteByte(ch)
			} else if ch == ';' {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" {
					statements = append(statements, stmt)
				}
				current.Reset()
			} else {
				current.WriteByte(ch)
			}
		} else {
			current.WriteByte(ch)
			if ch == stringChar && (i == 0 || sql[i-1] != '\\') {
				inString = false
			}
		}
	}

	// Add last statement if any
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// MarkMigrationAsApplied marks a migration as manually applied
func (m *Manager) MarkMigrationAsApplied(migrationName string) error {
	if err := m.EnsureMigrationsTable(); err != nil {
		return err
	}

	// Check if it already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM _prisma_migrations WHERE migration_name = $1)`
	if err := m.db.QueryRow(checkQuery, migrationName).Scan(&exists); err != nil {
		return fmt.Errorf("error checking migration: %w", err)
	}

	if exists {
		// Update to mark as applied
		updateQuery := `
			UPDATE _prisma_migrations 
			SET finished_at = COALESCE(finished_at, NOW()),
				rolled_back_at = NULL
			WHERE migration_name = $1
		`
		_, err := m.db.Exec(updateQuery, migrationName)
		if err != nil {
			return fmt.Errorf("error updating migration: %w", err)
		}
	} else {
		// Create record
		insertQuery := `
			INSERT INTO _prisma_migrations (id, checksum, migration_name, started_at, finished_at, applied_steps_count)
			VALUES ($1, $2, $3, NOW(), NOW(), 1)
		`
		migrationID := generateMigrationID()
		checksum := "manual"
		_, err := m.db.Exec(insertQuery, migrationID, checksum, migrationName)
		if err != nil {
			return fmt.Errorf("error creating migration record: %w", err)
		}
	}

	return nil
}

// MarkMigrationAsRolledBack marks a migration as rolled back (not applied)
func (m *Manager) MarkMigrationAsRolledBack(migrationName string) error {
	if err := m.EnsureMigrationsTable(); err != nil {
		return err
	}

	updateQuery := `
		UPDATE _prisma_migrations 
		SET rolled_back_at = NOW(),
			finished_at = NULL
		WHERE migration_name = $1
	`
	result, err := m.db.Exec(updateQuery, migrationName)
	if err != nil {
		return fmt.Errorf("error updating migration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("migration '%s' not found", migrationName)
	}

	return nil
}

// EnsureMigrationLockfile ensures that migration_lock.toml exists in the migrations directory
// This file locks the database provider to prevent mixing providers
func EnsureMigrationLockfile(migrationsPath, provider string) error {
	lockfilePath := filepath.Join(migrationsPath, "migration_lock.toml")

	// Check if lockfile already exists
	if _, err := os.Stat(lockfilePath); err == nil {
		// Read existing lockfile to verify provider matches
		content, err := os.ReadFile(lockfilePath)
		if err != nil {
			return fmt.Errorf("error reading migration_lock.toml: %w", err)
		}

		// Check if provider matches
		contentStr := string(content)
		expectedProvider := fmt.Sprintf("provider = \"%s\"", provider)
		if !strings.Contains(contentStr, expectedProvider) {
			return fmt.Errorf("migration_lock.toml exists with different provider. Expected: %s", provider)
		}

		return nil
	}

	// Create lockfile with provider
	lockfileContent := fmt.Sprintf("provider = \"%s\"\n", provider)
	if err := os.WriteFile(lockfilePath, []byte(lockfileContent), 0644); err != nil {
		return fmt.Errorf("error writing migration_lock.toml: %w", err)
	}

	return nil
}
