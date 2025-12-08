package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/config"
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
func (m *Manager) EnsureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS _prisma_migrations (
			id VARCHAR(36) PRIMARY KEY,
			checksum VARCHAR(64) NOT NULL,
			finished_at TIMESTAMP,
			migration_name VARCHAR(255) NOT NULL,
			logs TEXT,
			rolled_back_at TIMESTAMP,
			started_at TIMESTAMP NOT NULL,
			applied_steps_count INTEGER NOT NULL DEFAULT 0
		)
	`

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating _prisma_migrations table: %w", err)
	}

	return nil
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
		if !isValidMigrationName(name) {
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

// generateMigrationID generates a unique ID for the migration
func generateMigrationID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// calculateChecksum calculates a simple checksum of the SQL
func calculateChecksum(sql string) string {
	// Simple checksum (in production, use SHA256)
	return fmt.Sprintf("%d", len(sql))
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
