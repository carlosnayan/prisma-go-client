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

// Migration representa uma migration
type Migration struct {
	Name      string    // Nome da migration (ex: "20241219120000_add_users")
	Path      string    // Caminho completo para o diretório da migration
	SQL       string    // Conteúdo do arquivo migration.sql
	AppliedAt time.Time // Quando foi aplicada (zero se não aplicada)
}

// Manager gerencia migrations
type Manager struct {
	config         *config.Config
	db             *sql.DB
	migrationsPath string
}

// NewManager cria um novo gerenciador de migrations
func NewManager(cfg *config.Config, db *sql.DB) (*Manager, error) {
	migrationsPath := cfg.GetMigrationsPath()

	// Criar diretório se não existir
	if err := os.MkdirAll(migrationsPath, 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório de migrations: %w", err)
	}

	return &Manager{
		config:         cfg,
		db:             db,
		migrationsPath: migrationsPath,
	}, nil
}

// EnsureMigrationsTable garante que a tabela _prisma_migrations existe
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
		return fmt.Errorf("erro ao criar tabela _prisma_migrations: %w", err)
	}

	return nil
}

// GetAppliedMigrations retorna lista de migrations aplicadas
func (m *Manager) GetAppliedMigrations() ([]string, error) {
	if err := m.EnsureMigrationsTable(); err != nil {
		return nil, err
	}

	query := `SELECT migration_name FROM _prisma_migrations WHERE finished_at IS NOT NULL ORDER BY started_at`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar migrations aplicadas: %w", err)
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("erro ao ler migration: %w", err)
		}
		migrations = append(migrations, name)
	}

	return migrations, nil
}

// GetLocalMigrations retorna lista de migrations locais (arquivos)
func (m *Manager) GetLocalMigrations() ([]*Migration, error) {
	entries, err := os.ReadDir(m.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler diretório de migrations: %w", err)
	}

	var migrations []*Migration
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Verificar se é um diretório de migration (formato: YYYYMMDDHHMMSS_nome)
		if !isValidMigrationName(name) {
			continue
		}

		migrationPath := filepath.Join(m.migrationsPath, name)
		sqlPath := filepath.Join(migrationPath, "migration.sql")

		sqlContent, err := os.ReadFile(sqlPath)
		if err != nil {
			// Se não tem migration.sql, pular
			continue
		}

		migrations = append(migrations, &Migration{
			Name: name,
			Path: migrationPath,
			SQL:  string(sqlContent),
		})
	}

	// Ordenar por nome (que contém timestamp)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}

// isValidMigrationName verifica se o nome da migration está no formato correto
func isValidMigrationName(name string) bool {
	// Formato: YYYYMMDDHHMMSS_nome (pelo menos 14 dígitos + underscore + nome)
	if len(name) < 16 {
		return false
	}

	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return false
	}

	// Verificar se a primeira parte é numérica e tem 14 dígitos
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

// GetPendingMigrations retorna migrations pendentes (locais mas não aplicadas)
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

// ApplyMigration aplica uma migration no banco
func (m *Manager) ApplyMigration(migration *Migration) error {
	if err := m.EnsureMigrationsTable(); err != nil {
		return err
	}

	// Iniciar transação
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Registrar início da migration
	migrationID := generateMigrationID()
	startedAt := time.Now()

	insertQuery := `
		INSERT INTO _prisma_migrations (id, checksum, migration_name, started_at, applied_steps_count)
		VALUES ($1, $2, $3, $4, 1)
	`

	checksum := calculateChecksum(migration.SQL)
	_, err = tx.Exec(insertQuery, migrationID, checksum, migration.Name, startedAt)
	if err != nil {
		return fmt.Errorf("erro ao registrar migration: %w", err)
	}

	// Executar SQL da migration
	// Dividir por ; para executar múltiplos statements
	statements := SplitSQLStatements(migration.SQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("erro ao executar migration %s: %w\nSQL: %s", migration.Name, err, stmt)
		}
	}

	// Marcar como finalizada
	finishedAt := time.Now()
	updateQuery := `
		UPDATE _prisma_migrations 
		SET finished_at = $1, applied_steps_count = $2
		WHERE id = $3
	`
	_, err = tx.Exec(updateQuery, finishedAt, len(statements), migrationID)
	if err != nil {
		return fmt.Errorf("erro ao finalizar migration: %w", err)
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao commitar migration: %w", err)
	}

	return nil
}

// generateMigrationID gera um ID único para a migration
func generateMigrationID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// calculateChecksum calcula checksum simples do SQL
func calculateChecksum(sql string) string {
	// Checksum simples (em produção, usar SHA256)
	return fmt.Sprintf("%d", len(sql))
}

// SplitSQLStatements divide SQL em statements individuais
func SplitSQLStatements(sql string) []string {
	// Dividir por ; mas respeitar strings e comentários
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

	// Adicionar último statement se houver
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// MarkMigrationAsApplied marca uma migration como aplicada manualmente
func (m *Manager) MarkMigrationAsApplied(migrationName string) error {
	if err := m.EnsureMigrationsTable(); err != nil {
		return err
	}

	// Verificar se já existe
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM _prisma_migrations WHERE migration_name = $1)`
	if err := m.db.QueryRow(checkQuery, migrationName).Scan(&exists); err != nil {
		return fmt.Errorf("erro ao verificar migration: %w", err)
	}

	if exists {
		// Atualizar para marcar como aplicada
		updateQuery := `
			UPDATE _prisma_migrations 
			SET finished_at = COALESCE(finished_at, NOW()),
				rolled_back_at = NULL
			WHERE migration_name = $1
		`
		_, err := m.db.Exec(updateQuery, migrationName)
		if err != nil {
			return fmt.Errorf("erro ao atualizar migration: %w", err)
		}
	} else {
		// Criar registro
		insertQuery := `
			INSERT INTO _prisma_migrations (id, checksum, migration_name, started_at, finished_at, applied_steps_count)
			VALUES ($1, $2, $3, NOW(), NOW(), 1)
		`
		migrationID := generateMigrationID()
		checksum := "manual"
		_, err := m.db.Exec(insertQuery, migrationID, checksum, migrationName)
		if err != nil {
			return fmt.Errorf("erro ao criar registro de migration: %w", err)
		}
	}

	return nil
}

// MarkMigrationAsRolledBack marca uma migration como rolled back (não aplicada)
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
		return fmt.Errorf("erro ao atualizar migration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("migration '%s' não encontrada", migrationName)
	}

	return nil
}
