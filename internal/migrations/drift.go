package migrations

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// DetectDrift detects if there's a drift between the expected schema (based on migrations)
// and the actual database schema. Drift occurs when:
// 1. All local migrations are applied
// 2. But the database schema differs from schema.prisma
func DetectDrift(manager *Manager, db *sql.DB, schema *parser.Schema, provider string) (*SchemaDiff, bool, error) {
	// Get local and applied migrations
	local, err := manager.GetLocalMigrations()
	if err != nil {
		return nil, false, fmt.Errorf("error getting local migrations: %w", err)
	}

	applied, err := manager.GetAppliedMigrations()
	if err != nil {
		return nil, false, fmt.Errorf("error getting applied migrations: %w", err)
	}

	// Create map of applied migrations
	appliedMap := make(map[string]bool)
	for _, name := range applied {
		appliedMap[name] = true
	}

	// Check if all local migrations are applied
	// If there are no local migrations, we can't detect drift (database might be empty or manually created)
	if len(local) == 0 {
		return nil, false, nil
	}

	allLocalApplied := true
	for _, migration := range local {
		if !appliedMap[migration.Name] {
			allLocalApplied = false
			break
		}
	}

	// If not all local migrations are applied, there's no drift (just pending changes)
	if !allLocalApplied {
		return nil, false, nil
	}

	// All local migrations are applied, so compare schema.prisma with database
	// If they differ, it's drift
	dbSchema, err := IntrospectDatabase(db, provider)
	if err != nil {
		return nil, false, fmt.Errorf("error introspecting database: %w", err)
	}

	diff, err := CompareSchema(schema, dbSchema, provider)
	if err != nil {
		return nil, false, fmt.Errorf("error comparing schema: %w", err)
	}

	// Check if there are differences
	hasDrift := len(diff.TablesToCreate) > 0 ||
		len(diff.TablesToAlter) > 0 ||
		len(diff.TablesToDrop) > 0 ||
		len(diff.IndexesToCreate) > 0 ||
		len(diff.IndexesToDrop) > 0

	return diff, hasDrift, nil
}

// FormatDriftDiff formats the drift differences in Prisma style
// Returns the formatted string with drift information
func FormatDriftDiff(diff *SchemaDiff) string {
	var output strings.Builder

	// Format table changes (most common in drift)
	for _, alter := range diff.TablesToAlter {
		output.WriteString(fmt.Sprintf("[*] Changed the `%s` table\n", alter.TableName))

		// Added columns
		for _, col := range alter.AddColumns {
			output.WriteString(fmt.Sprintf("  [+] Added column `%s`\n", col.Name))
		}

		// Removed columns
		for _, colName := range alter.DropColumns {
			output.WriteString(fmt.Sprintf("  [-] Removed column `%s`\n", colName))
		}

		// Altered columns (type or nullable changes)
		for _, colAlter := range alter.AlterColumns {
			output.WriteString(fmt.Sprintf("  [*] Changed column `%s`\n", colAlter.ColumnName))
		}
	}

	// Format tables to create (shouldn't happen in drift usually, but handle it)
	for _, table := range diff.TablesToCreate {
		output.WriteString(fmt.Sprintf("[+] Added table `%s`\n", table.Name))
		for _, col := range table.Columns {
			output.WriteString(fmt.Sprintf("  [+] Added column `%s`\n", col.Name))
		}
	}

	// Format tables to drop
	for _, tableName := range diff.TablesToDrop {
		output.WriteString(fmt.Sprintf("[-] Removed table `%s`\n", tableName))
	}

	// Format indexes (if any)
	for _, idx := range diff.IndexesToCreate {
		output.WriteString(fmt.Sprintf("[+] Added index `%s` on `%s`\n", idx.Name, idx.TableName))
	}

	for _, idxName := range diff.IndexesToDrop {
		output.WriteString(fmt.Sprintf("[-] Removed index `%s`\n", idxName))
	}

	return output.String()
}
