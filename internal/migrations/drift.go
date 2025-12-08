package migrations

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func DetectDrift(manager *Manager, db *sql.DB, schema *parser.Schema, provider string) (*SchemaDiff, bool, error) {
	local, err := manager.GetLocalMigrations()
	if err != nil {
		return nil, false, fmt.Errorf("error getting local migrations: %w", err)
	}

	applied, err := manager.GetAppliedMigrations()
	if err != nil {
		return nil, false, fmt.Errorf("error getting applied migrations: %w", err)
	}

	if len(local) == 0 {
		return nil, false, nil
	}

	appliedMap := make(map[string]bool)
	for _, name := range applied {
		appliedMap[name] = true
	}

	for _, migration := range local {
		if !appliedMap[migration.Name] {
			return nil, false, nil
		}
	}

	return nil, false, nil
}

func FormatDriftDiff(diff *SchemaDiff) string {
	var output strings.Builder

	for _, alter := range diff.TablesToAlter {
		output.WriteString(fmt.Sprintf("[*] Changed the `%s` table\n", alter.TableName))
		for _, col := range alter.AddColumns {
			output.WriteString(fmt.Sprintf("  [+] Added column `%s`\n", col.Name))
		}
		for _, colName := range alter.DropColumns {
			output.WriteString(fmt.Sprintf("  [-] Removed column `%s`\n", colName))
		}
		for _, colAlter := range alter.AlterColumns {
			output.WriteString(fmt.Sprintf("  [*] Changed column `%s`\n", colAlter.ColumnName))
		}
	}

	for _, table := range diff.TablesToCreate {
		output.WriteString(fmt.Sprintf("[+] Added table `%s`\n", table.Name))
		for _, col := range table.Columns {
			output.WriteString(fmt.Sprintf("  [+] Added column `%s`\n", col.Name))
		}
	}

	for _, tableName := range diff.TablesToDrop {
		output.WriteString(fmt.Sprintf("[-] Removed table `%s`\n", tableName))
	}

	for _, idx := range diff.IndexesToCreate {
		output.WriteString(fmt.Sprintf("[+] Added index `%s` on `%s`\n", idx.Name, idx.TableName))
	}

	for _, idxName := range diff.IndexesToDrop {
		output.WriteString(fmt.Sprintf("[-] Removed index `%s`\n", idxName))
	}

	return output.String()
}
