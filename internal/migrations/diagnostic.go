package migrations

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

type DevAction struct {
	Tag    string
	Reason string
}

type DevDiagnosticOutput struct {
	Action DevAction
}

func DevDiagnostic(manager *Manager, db *sql.DB, schema *parser.Schema, provider string) (*DevDiagnosticOutput, error) {
	modifiedMigrations, err := manager.GetModifiedMigrations()
	if err != nil {
		return nil, fmt.Errorf("error checking for modified migrations: %w", err)
	}

	if len(modifiedMigrations) > 0 {
		reason := fmt.Sprintf(
			"The following migration(s) have been modified since they were applied:\n  %s\n\nMigrations that have been applied to the database should not be modified.\nIf you need to change a migration, you must reset the database first.",
			strings.Join(modifiedMigrations, ", "),
		)
		return &DevDiagnosticOutput{Action: DevAction{Tag: "reset", Reason: reason}}, nil
	}

	missingMigrations, err := manager.GetMissingMigrations()
	if err != nil {
		return nil, fmt.Errorf("error checking for missing migrations: %w", err)
	}

	if len(missingMigrations) > 0 {
		reason := fmt.Sprintf(
			"The following migration(s) are applied to the database but missing from the local migrations directory:\n  %s",
			strings.Join(missingMigrations, ", "),
		)
		return &DevDiagnosticOutput{Action: DevAction{Tag: "reset", Reason: reason}}, nil
	}

	driftDiff, hasDrift, err := DetectDrift(manager, db, schema, provider)
	if err != nil {
		return nil, fmt.Errorf("error detecting drift: %w", err)
	}

	if hasDrift {
		return &DevDiagnosticOutput{Action: DevAction{Tag: "reset", Reason: buildDriftReason(driftDiff)}}, nil
	}

	return &DevDiagnosticOutput{Action: DevAction{Tag: "createMigration"}}, nil
}

func buildDriftReason(diff *SchemaDiff) string {
	parts := []string{
		"Drift detected: Your database schema is not in sync with your migration history.",
		"",
		"The following is a summary of the differences between the expected database schema given your migrations files, and the actual schema of the database.",
		"",
		"It should be understood as the set of changes to get from the expected schema to the actual schema.",
		"",
		"If you are running this the first time on an existing database, please make sure to read this documentation page:",
		"https://www.prisma.io/docs/guides/database/developing-with-prisma-migrate/troubleshooting-development",
		"",
	}

	hasChanges := false

	if len(diff.TablesToCreate) > 0 {
		parts = append(parts, "[+] Added tables")
		for _, table := range diff.TablesToCreate {
			parts = append(parts, fmt.Sprintf("  - %s", table.Name))
		}
		hasChanges = true
	}

	if len(diff.TablesToDrop) > 0 {
		if hasChanges {
			parts = append(parts, "")
		}
		parts = append(parts, "[-] Removed tables")
		for _, tableName := range diff.TablesToDrop {
			parts = append(parts, fmt.Sprintf("  - %s", tableName))
		}
		hasChanges = true
	}

	for _, alter := range diff.TablesToAlter {
		if hasChanges {
			parts = append(parts, "")
		}
		parts = append(parts, fmt.Sprintf("[*] Changed the `%s` table", alter.TableName))
		hasChanges = true
		for _, col := range alter.AddColumns {
			parts = append(parts, fmt.Sprintf("  [+] Added column `%s`", col.Name))
		}
		for _, colName := range alter.DropColumns {
			parts = append(parts, fmt.Sprintf("  [-] Removed column `%s`", colName))
		}
		for _, colAlter := range alter.AlterColumns {
			parts = append(parts, fmt.Sprintf("  [*] Changed column `%s`", colAlter.ColumnName))
		}
	}

	if len(diff.IndexesToCreate) > 0 {
		if hasChanges {
			parts = append(parts, "")
		}
		parts = append(parts, "[+] Added indexes")
		for _, idx := range diff.IndexesToCreate {
			parts = append(parts, fmt.Sprintf("  - %s on `%s`", idx.Name, idx.TableName))
		}
		hasChanges = true
	}

	if len(diff.IndexesToDrop) > 0 {
		if hasChanges {
			parts = append(parts, "")
		}
		parts = append(parts, "[-] Removed indexes")
		for _, idxName := range diff.IndexesToDrop {
			parts = append(parts, fmt.Sprintf("  - %s", idxName))
		}
	}

	return strings.Join(parts, "\n")
}
