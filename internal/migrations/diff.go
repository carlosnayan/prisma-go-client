package migrations

import (
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// SchemaDiff represents differences between schema and database
type SchemaDiff struct {
	TablesToCreate      []TableDefinition
	TablesToAlter       []TableAlteration
	TablesToDrop        []string
	IndexesToCreate     []IndexDefinition
	IndexesToDrop       []string
	ForeignKeysToCreate []ForeignKeyDefinition
	ForeignKeysToAlter  []ForeignKeyDefinition // FKs that need to be altered (drop + recreate)
	ForeignKeysToDrop   []ForeignKeyDefinition // FKs that need to be removed
}

// ForeignKeyDefinition represents a foreign key constraint
type ForeignKeyDefinition struct {
	Name              string   // Constraint name (e.g., "table_column_fkey")
	TableName         string   // Table containing the FK
	Columns           []string // Local columns (fields)
	ReferencedTable   string   // Referenced table
	ReferencedColumns []string // Referenced columns (references)
	OnDelete          string   // "CASCADE", "SET NULL", "RESTRICT", "NO ACTION"
	OnUpdate          string   // "CASCADE", "SET NULL", "RESTRICT", "NO ACTION"
}

// TableDefinition represents a table to be created
type TableDefinition struct {
	Name        string
	Columns     []ColumnDefinition
	CompositePK []string // For composite primary keys from @@id([field1, field2])
}

// ColumnDefinition represents a column
type ColumnDefinition struct {
	Name         string
	Type         string
	IsNullable   bool
	IsPrimaryKey bool
	IsUnique     bool
	DefaultValue string
}

// TableAlteration represents alterations to a table
type TableAlteration struct {
	TableName    string
	AddColumns   []ColumnDefinition
	DropColumns  []string
	AlterColumns []ColumnAlteration
}

// ColumnAlteration represents an alteration to a column
type ColumnAlteration struct {
	ColumnName  string
	NewType     string
	NewNullable bool
}

// IndexDefinition represents an index
type IndexDefinition struct {
	Name      string
	TableName string
	Columns   []string
	IsUnique  bool
}

// needsUUIDExtension checks if the migration needs the pgcrypto extension for gen_random_uuid()
func needsUUIDExtension(diff *SchemaDiff) bool {
	// Check in tables to be created
	for _, table := range diff.TablesToCreate {
		for _, col := range table.Columns {
			if strings.Contains(strings.ToLower(col.DefaultValue), "gen_random_uuid") {
				return true
			}
		}
	}

	// Check in columns to be added
	for _, alter := range diff.TablesToAlter {
		for _, col := range alter.AddColumns {
			if strings.Contains(strings.ToLower(col.DefaultValue), "gen_random_uuid") {
				return true
			}
		}
	}

	return false
}

// GenerateMigrationSQL generates migration SQL based on differences
func GenerateMigrationSQL(diff *SchemaDiff, provider string) (string, error) {
	var steps []string
	d := dialect.GetDialect(provider)

	// If PostgreSQL and needs gen_random_uuid(), create extension
	if provider == "postgresql" && needsUUIDExtension(diff) {
		var sql strings.Builder
		sql.WriteString("-- Enable pgcrypto extension for gen_random_uuid()\n")
		sql.WriteString("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";\n")
		steps = append(steps, sql.String())
	}

	// Create tables
	if len(diff.TablesToCreate) > 0 {
		var sql strings.Builder
		sql.WriteString("-- CreateTable\n")
		for _, table := range diff.TablesToCreate {
			sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", d.QuoteIdentifier(table.Name)))

			var columns []string
			var primaryKeys []string

			for _, col := range table.Columns {
				colDef := fmt.Sprintf("  %s %s", d.QuoteIdentifier(col.Name), d.MapType(col.Type, col.IsNullable))

				if !col.IsNullable {
					colDef += " NOT NULL"
				}

				if col.DefaultValue != "" {
					colDef += " DEFAULT " + col.DefaultValue
				}

				if col.IsPrimaryKey {
					primaryKeys = append(primaryKeys, col.Name)
				}

				columns = append(columns, colDef)
			}

			sql.WriteString(strings.Join(columns, ",\n"))

			// Handle composite primary key from @@id
			if len(table.CompositePK) > 0 {
				quotedPKs := make([]string, len(table.CompositePK))
				for i, pk := range table.CompositePK {
					quotedPKs[i] = d.QuoteIdentifier(pk)
				}
				if provider == "mysql" {
					sql.WriteString(fmt.Sprintf(",\n  PRIMARY KEY (%s)", strings.Join(quotedPKs, ", ")))
				} else {
					sql.WriteString(fmt.Sprintf(",\n  CONSTRAINT %s PRIMARY KEY (%s)",
						d.QuoteIdentifier(table.Name+"_pkey"),
						strings.Join(quotedPKs, ", ")))
				}
			} else if len(primaryKeys) > 0 {
				quotedPKs := make([]string, len(primaryKeys))
				for i, pk := range primaryKeys {
					quotedPKs[i] = d.QuoteIdentifier(pk)
				}
				if provider == "mysql" {
					sql.WriteString(fmt.Sprintf(",\n  PRIMARY KEY (%s)", strings.Join(quotedPKs, ", ")))
				} else {
					sql.WriteString(fmt.Sprintf(",\n  CONSTRAINT %s PRIMARY KEY (%s)",
						d.QuoteIdentifier(table.Name+"_pkey"),
						strings.Join(quotedPKs, ", ")))
				}
			}

			sql.WriteString("\n);\n")
		}
		steps = append(steps, sql.String())
	}

	// Alter tables (Drop columns)
	for _, alter := range diff.TablesToAlter {
		if len(alter.DropColumns) > 0 {
			var sql strings.Builder
			sql.WriteString("-- AlterTable\n")
			for _, colName := range alter.DropColumns {
				sql.WriteString(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;\n",
					d.QuoteIdentifier(alter.TableName),
					d.QuoteIdentifier(colName)))
			}
			steps = append(steps, sql.String())
		}
	}

	// Alter tables (Add columns)
	for _, alter := range diff.TablesToAlter {
		if len(alter.AddColumns) > 0 {
			var sql strings.Builder
			// Only add header if we haven't already processed drop columns for this table in the previous block
			// But since we split drops and adds into different steps blocks for clarity
			// We can just add -- AlterTable here too if needed, or rely on the previous block.
			// However, typically all alters for a table are grouped.
			// For simplicity and matching standard output, we group drops then adds.
			// If we just had drops, we already added -- AlterTable.
			// If we have adds, we might want another header or just continue?
			// Standard Prisma CLI groups them.
			// Let's stick to the user request: spacing between operations.

			sql.WriteString("-- AlterTable\n")
			for _, col := range alter.AddColumns {
				colDef := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
					d.QuoteIdentifier(alter.TableName),
					d.QuoteIdentifier(col.Name),
					d.MapType(col.Type, col.IsNullable))

				if !col.IsNullable {
					colDef += " NOT NULL"
				}

				if col.DefaultValue != "" {
					colDef += " DEFAULT " + col.DefaultValue
				}

				sql.WriteString(colDef + ";\n")
			}
			steps = append(steps, sql.String())
		}
	}

	// Drop indexes
	if len(diff.IndexesToDrop) > 0 {
		var sql strings.Builder
		sql.WriteString("-- DropIndex\n")
		for _, idxName := range diff.IndexesToDrop {
			sql.WriteString(fmt.Sprintf("DROP INDEX %s;\n", d.QuoteIdentifier(idxName)))
		}
		steps = append(steps, sql.String())
	}

	// Drop tables
	if len(diff.TablesToDrop) > 0 {
		var sql strings.Builder
		sql.WriteString("-- DropTable\n")
		for _, tableName := range diff.TablesToDrop {
			sql.WriteString(fmt.Sprintf("DROP TABLE %s;\n", d.QuoteIdentifier(tableName)))
		}
		steps = append(steps, sql.String())
	}

	// Create indexes
	if len(diff.IndexesToCreate) > 0 {
		var sql strings.Builder
		sql.WriteString("-- CreateIndex\n")
		for _, idx := range diff.IndexesToCreate {
			unique := ""
			if idx.IsUnique {
				unique = "UNIQUE "
			}
			quotedCols := make([]string, len(idx.Columns))
			for i, col := range idx.Columns {
				quotedCols[i] = d.QuoteIdentifier(col)
			}
			sql.WriteString(fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);\n",
				unique,
				d.QuoteIdentifier(idx.Name),
				d.QuoteIdentifier(idx.TableName),
				strings.Join(quotedCols, ", ")))
		}
		steps = append(steps, sql.String())
	}

	// Drop foreign keys that need to be removed
	if len(diff.ForeignKeysToDrop) > 0 {
		var sql strings.Builder
		sql.WriteString("-- DropForeignKey\n")
		for _, fk := range diff.ForeignKeysToDrop {
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;\n",
				d.QuoteIdentifier(fk.TableName),
				d.QuoteIdentifier(fk.Name)))
		}
		steps = append(steps, sql.String())
	}

	// Drop foreign keys that need to be altered (before recreating)
	if len(diff.ForeignKeysToAlter) > 0 {
		var sql strings.Builder
		sql.WriteString("-- AlterForeignKey (drop old)\n")
		for _, fk := range diff.ForeignKeysToAlter {
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;\n",
				d.QuoteIdentifier(fk.TableName),
				d.QuoteIdentifier(fk.Name)))
		}
		steps = append(steps, sql.String())
	}

	// Add foreign keys (new ones)
	if len(diff.ForeignKeysToCreate) > 0 {
		var sql strings.Builder
		sql.WriteString("-- AddForeignKey\n")
		for _, fk := range diff.ForeignKeysToCreate {
			quotedCols := make([]string, len(fk.Columns))
			for i, col := range fk.Columns {
				quotedCols[i] = d.QuoteIdentifier(col)
			}
			quotedRefCols := make([]string, len(fk.ReferencedColumns))
			for i, col := range fk.ReferencedColumns {
				quotedRefCols[i] = d.QuoteIdentifier(col)
			}

			onDelete := fk.OnDelete
			if onDelete == "" {
				onDelete = "CASCADE"
			}
			onUpdate := fk.OnUpdate
			if onUpdate == "" {
				onUpdate = "CASCADE"
			}

			sql.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s) ON DELETE %s ON UPDATE %s;\n",
				d.QuoteIdentifier(fk.TableName),
				d.QuoteIdentifier(fk.Name),
				strings.Join(quotedCols, ", "),
				d.QuoteIdentifier(fk.ReferencedTable),
				strings.Join(quotedRefCols, ", "),
				onDelete,
				onUpdate))
		}
		steps = append(steps, sql.String())
	}

	// Recreate foreign keys that were altered
	if len(diff.ForeignKeysToAlter) > 0 {
		var sql strings.Builder
		sql.WriteString("-- AlterForeignKey (recreate with new attributes)\n")
		for _, fk := range diff.ForeignKeysToAlter {
			quotedCols := make([]string, len(fk.Columns))
			for i, col := range fk.Columns {
				quotedCols[i] = d.QuoteIdentifier(col)
			}
			quotedRefCols := make([]string, len(fk.ReferencedColumns))
			for i, col := range fk.ReferencedColumns {
				quotedRefCols[i] = d.QuoteIdentifier(col)
			}

			onDelete := fk.OnDelete
			if onDelete == "" {
				onDelete = "CASCADE"
			}
			onUpdate := fk.OnUpdate
			if onUpdate == "" {
				onUpdate = "CASCADE"
			}

			sql.WriteString(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s) ON DELETE %s ON UPDATE %s;\n",
				d.QuoteIdentifier(fk.TableName),
				d.QuoteIdentifier(fk.Name),
				strings.Join(quotedCols, ", "),
				d.QuoteIdentifier(fk.ReferencedTable),
				strings.Join(quotedRefCols, ", "),
				onDelete,
				onUpdate))
		}
		steps = append(steps, sql.String())
	}

	return strings.Join(steps, "\n"), nil
}

// SchemaToSQL converts a Prisma schema to SQL (creates everything from scratch)
// Use CompareSchema to detect incremental changes
func SchemaToSQL(schema *parser.Schema, provider string) (*SchemaDiff, error) {
	diff := &SchemaDiff{
		ForeignKeysToCreate: []ForeignKeyDefinition{},
		IndexesToCreate:     []IndexDefinition{},
	}

	// Create a map of model names for quick lookup
	modelMap := make(map[string]*parser.Model)
	for _, model := range schema.Models {
		modelMap[model.Name] = model
	}

	for _, model := range schema.Models {
		// Use @@map if present, otherwise use model name
		tableName := getTableNameFromModel(model)
		table := TableDefinition{
			Name:    tableName,
			Columns: []ColumnDefinition{},
		}

		for _, field := range model.Fields {
			// Check if field type is a known model (relation field)
			if _, isModel := modelMap[field.Type.Name]; isModel {
				continue // Skip relation fields (they don't create columns)
			}

			// Also skip array fields (already handled implicitly above if model, but good for safety)
			if field.Type.IsArray {
				continue
			}

			// Use @map if present, otherwise use field name
			columnName := getColumnNameFromField(field)
			col := ColumnDefinition{
				Name:       columnName,
				Type:       field.Type.Name,
				IsNullable: field.Type.IsOptional,
			}

			// Check attributes
			for _, attr := range field.Attributes {
				switch attr.Name {
				case "id":
					col.IsPrimaryKey = true
					col.IsNullable = false
				case "unique":
					col.IsUnique = true
					// Add explicit unique index
					indexName := fmt.Sprintf("%s_%s_key", tableName, columnName)
					diff.IndexesToCreate = append(diff.IndexesToCreate, IndexDefinition{
						Name:      indexName,
						TableName: tableName,
						Columns:   []string{columnName},
						IsUnique:  true,
					})
				case "default":
					if len(attr.Arguments) > 0 {
						// Extract default value
						col.DefaultValue = extractDefaultValue(attr.Arguments[0])
					}
				case "updatedAt":
					// @updatedAt doesn't need special SQL, but mark it for reference
					// Usually has @default(now()) which is already handled
				case "db.Uuid", "db.UUID":
					col.Type = "UUID"
				case "db.VarChar":
					if len(attr.Arguments) > 0 {
						size := getNumericValue(attr.Arguments[0].Value)
						if size != "" {
							col.Type = "VARCHAR(" + size + ")"
						} else {
							col.Type = "VARCHAR(255)"
						}
					} else {
						col.Type = "VARCHAR(255)"
					}
				case "db.Text":
					col.Type = "TEXT"
				case "db.Char":
					if len(attr.Arguments) > 0 {
						size := getNumericValue(attr.Arguments[0].Value)
						if size != "" {
							col.Type = "CHAR(" + size + ")"
						} else {
							col.Type = "CHAR(1)"
						}
					} else {
						col.Type = "CHAR(1)"
					}
				case "db.Date":
					col.Type = "DATE"
				case "db.Time":
					col.Type = "TIME"
				case "db.Timestamp":
					col.Type = "TIMESTAMP"
				case "db.Timestamptz":
					col.Type = "TIMESTAMPTZ"
				case "db.Decimal":
					if len(attr.Arguments) >= 2 {
						precision := getNumericValue(attr.Arguments[0].Value)
						scale := getNumericValue(attr.Arguments[1].Value)
						if precision != "" && scale != "" {
							col.Type = "DECIMAL(" + precision + "," + scale + ")"
						} else if precision != "" {
							col.Type = "DECIMAL(" + precision + ",0)"
						} else {
							col.Type = "DECIMAL(65,30)"
						}
					} else if len(attr.Arguments) == 1 {
						precision := getNumericValue(attr.Arguments[0].Value)
						if precision != "" {
							col.Type = "DECIMAL(" + precision + ",0)"
						} else {
							col.Type = "DECIMAL(65,30)"
						}
					} else {
						col.Type = "DECIMAL(65,30)"
					}
				case "db.SmallInt":
					col.Type = "SMALLINT"
				case "db.Integer":
					col.Type = "INTEGER"
				case "db.BigInt":
					col.Type = "BIGINT"
				case "db.Real":
					col.Type = "REAL"
				case "db.DoublePrecision":
					col.Type = "DOUBLE PRECISION"
				case "db.Boolean":
					col.Type = "BOOLEAN"
				case "db.Json":
					col.Type = "JSON"
				case "db.JsonB":
					col.Type = "JSONB"
				case "db.Bytes":
					col.Type = "BYTEA" // Will be mapped per provider
				case "db.ByteA":
					col.Type = "BYTEA"
				case "db.Inet":
					col.Type = "INET"
				case "db.Cidr":
					col.Type = "CIDR"
				case "db.Money":
					col.Type = "MONEY"
				case "db.Bit":
					if len(attr.Arguments) > 0 {
						size := getNumericValue(attr.Arguments[0].Value)
						if size != "" {
							col.Type = "BIT(" + size + ")"
						} else {
							col.Type = "BIT(1)"
						}
					} else {
						col.Type = "BIT(1)"
					}
				case "db.VarBit":
					if len(attr.Arguments) > 0 {
						size := getNumericValue(attr.Arguments[0].Value)
						if size != "" {
							col.Type = "VARBIT(" + size + ")"
						} else {
							col.Type = "VARBIT"
						}
					} else {
						col.Type = "VARBIT"
					}
				}
			}

			table.Columns = append(table.Columns, col)
		}

		diff.TablesToCreate = append(diff.TablesToCreate, table)
	}

	// Process relations and @@unique attributes
	processRelationsAndUniqueForSchema(schema, diff, modelMap)

	return diff, nil
}

// processRelationsAndUniqueForSchema processes @relation, @@unique, and @@index for SchemaToSQL
func processRelationsAndUniqueForSchema(schema *parser.Schema, diff *SchemaDiff, modelMap map[string]*parser.Model) {
	// Process each model
	for _, model := range schema.Models {
		// Get mapped table name
		tableName := getTableNameFromModel(model)

		// Find the table definition to add composite PK if needed
		var tableDef *TableDefinition
		for i := range diff.TablesToCreate {
			if diff.TablesToCreate[i].Name == tableName {
				tableDef = &diff.TablesToCreate[i]
				break
			}
		}

		// Process @@id attribute (composite primary key)
		for _, attr := range model.Attributes {
			if attr.Name == "id" {
				// Extract fields from @@id([field1, field2])
				var pkFields []string
				for _, arg := range attr.Arguments {
					if arg.Name == "" {
						if fields, ok := arg.Value.([]interface{}); ok {
							for _, field := range fields {
								if fieldStr, ok := field.(string); ok {
									pkFields = append(pkFields, strings.Trim(fieldStr, `"`))
								}
							}
						}
					}
				}
				if len(pkFields) > 0 && tableDef != nil {
					// Map column names
					mappedPKFields := make([]string, len(pkFields))
					for i, pkField := range pkFields {
						for _, field := range model.Fields {
							if field.Name == pkField {
								mappedPKFields[i] = getColumnNameFromField(field)
								break
							}
						}
						if mappedPKFields[i] == "" {
							mappedPKFields[i] = pkField // Fallback
						}
					}
					tableDef.CompositePK = mappedPKFields
					// Remove individual PK flags from columns
					for i := range tableDef.Columns {
						for _, pkField := range mappedPKFields {
							if tableDef.Columns[i].Name == pkField {
								tableDef.Columns[i].IsPrimaryKey = false
							}
						}
					}
				}
			}
		}

		// Process @@unique attributes
		for _, attr := range model.Attributes {
			if attr.Name == "unique" {
				indexDef := extractUniqueIndex(tableName, attr)
				if indexDef != nil {
					// Map column names in index
					mappedColumns := make([]string, len(indexDef.Columns))
					for i, col := range indexDef.Columns {
						// Find field and get mapped name
						for _, field := range model.Fields {
							if field.Name == col {
								mappedColumns[i] = getColumnNameFromField(field)
								break
							}
						}
						if mappedColumns[i] == "" {
							mappedColumns[i] = col // Fallback to original name
						}
					}
					indexDef.Columns = mappedColumns
					diff.IndexesToCreate = append(diff.IndexesToCreate, *indexDef)
				}
			}
			// Process @@index attributes (non-unique indexes)
			if attr.Name == "index" {
				indexDef := extractIndex(tableName, attr)
				if indexDef != nil {
					// Map column names in index
					mappedColumns := make([]string, len(indexDef.Columns))
					for i, col := range indexDef.Columns {
						// Find field and get mapped name
						for _, field := range model.Fields {
							if field.Name == col {
								mappedColumns[i] = getColumnNameFromField(field)
								break
							}
						}
						if mappedColumns[i] == "" {
							mappedColumns[i] = col // Fallback to original name
						}
					}
					indexDef.Columns = mappedColumns
					diff.IndexesToCreate = append(diff.IndexesToCreate, *indexDef)
				}
			}
		}

		// Process @relation attributes to extract foreign keys
		for _, field := range model.Fields {
			for _, attr := range field.Attributes {
				if attr.Name == "relation" {
					fkDef := extractForeignKey(tableName, field, attr, modelMap)
					if fkDef != nil {
						// Map column names in foreign key
						mappedColumns := make([]string, len(fkDef.Columns))
						for i, col := range fkDef.Columns {
							// Find field and get mapped name
							for _, f := range model.Fields {
								if f.Name == col {
									mappedColumns[i] = getColumnNameFromField(f)
									break
								}
							}
							if mappedColumns[i] == "" {
								mappedColumns[i] = col // Fallback to original name
							}
						}
						fkDef.Columns = mappedColumns
						// Map referenced columns if referenced table has @@map
						if refModel, exists := modelMap[fkDef.ReferencedTable]; exists {
							fkDef.ReferencedTable = getTableNameFromModel(refModel)
							// Map referenced column names
							mappedRefColumns := make([]string, len(fkDef.ReferencedColumns))
							for i, refCol := range fkDef.ReferencedColumns {
								// Find field in referenced model and get mapped name
								for _, refField := range refModel.Fields {
									if refField.Name == refCol {
										mappedRefColumns[i] = getColumnNameFromField(refField)
										break
									}
								}
								if mappedRefColumns[i] == "" {
									mappedRefColumns[i] = refCol // Fallback to original name
								}
							}
							fkDef.ReferencedColumns = mappedRefColumns
						}
						diff.ForeignKeysToCreate = append(diff.ForeignKeysToCreate, *fkDef)
					}
				}
			}
		}
	}
}

// extractUniqueIndex extracts a unique index from @@unique attribute
// tableName should already be the mapped table name
func extractUniqueIndex(tableName string, attr *parser.Attribute) *IndexDefinition {
	var columns []string
	var indexName string

	// Extract fields from the unique attribute
	// @@unique([field1, field2], map: "index_name")
	for _, arg := range attr.Arguments {
		if arg.Name == "map" {
			if name, ok := arg.Value.(string); ok {
				indexName = strings.Trim(name, `"`)
			}
		} else if arg.Name == "" || arg.Name == "fields" {
			// First unnamed argument should be the array of fields
			if fields, ok := arg.Value.([]interface{}); ok {
				for _, field := range fields {
					if fieldStr, ok := field.(string); ok {
						columns = append(columns, strings.Trim(fieldStr, `"`))
					}
				}
			}
		}
	}

	// If no fields found, try to get from first argument (might be array without name)
	if len(columns) == 0 && len(attr.Arguments) > 0 {
		firstArg := attr.Arguments[0]
		if firstArg.Name == "" {
			if fields, ok := firstArg.Value.([]interface{}); ok {
				for _, field := range fields {
					if fieldStr, ok := field.(string); ok {
						columns = append(columns, strings.Trim(fieldStr, `"`))
					}
				}
			}
		}
	}

	if len(columns) == 0 {
		return nil
	}

	// Generate index name if not provided
	if indexName == "" {
		if len(columns) == 1 {
			indexName = fmt.Sprintf("%s_%s_key", tableName, columns[0])
		} else {
			indexName = fmt.Sprintf("%s_%s_key", tableName, columns[0])
		}
	}

	return &IndexDefinition{
		Name:      indexName,
		TableName: tableName,
		Columns:   columns,
		IsUnique:  true,
	}
}

// extractIndex extracts a non-unique index from @@index attribute
// tableName should already be the mapped table name
func extractIndex(tableName string, attr *parser.Attribute) *IndexDefinition {
	var columns []string
	var indexName string

	// Extract fields from the index attribute
	// @@index([field1, field2], map: "index_name")
	for _, arg := range attr.Arguments {
		if arg.Name == "map" {
			if name, ok := arg.Value.(string); ok {
				indexName = strings.Trim(name, `"`)
			}
		} else if arg.Name == "" || arg.Name == "fields" {
			// First unnamed argument should be the array of fields
			if fields, ok := arg.Value.([]interface{}); ok {
				for _, field := range fields {
					if fieldStr, ok := field.(string); ok {
						columns = append(columns, strings.Trim(fieldStr, `"`))
					}
				}
			}
		}
	}

	// If no fields found, try to get from first argument (might be array without name)
	if len(columns) == 0 && len(attr.Arguments) > 0 {
		firstArg := attr.Arguments[0]
		if firstArg.Name == "" {
			if fields, ok := firstArg.Value.([]interface{}); ok {
				for _, field := range fields {
					if fieldStr, ok := field.(string); ok {
						columns = append(columns, strings.Trim(fieldStr, `"`))
					}
				}
			}
		}
	}

	if len(columns) == 0 {
		return nil
	}

	// Generate index name if not provided
	if indexName == "" {
		if len(columns) == 1 {
			indexName = fmt.Sprintf("%s_%s_idx", tableName, columns[0])
		} else {
			indexName = fmt.Sprintf("%s_%s_idx", tableName, columns[0])
		}
	}

	return &IndexDefinition{
		Name:      indexName,
		TableName: tableName,
		Columns:   columns,
		IsUnique:  false, // Non-unique index
	}
}

// extractForeignKey extracts foreign key information from @relation attribute
// Only processes relations that have explicit fields and references (actual foreign keys)
func extractForeignKey(tableName string, field *parser.ModelField, attr *parser.Attribute, modelMap map[string]*parser.Model) *ForeignKeyDefinition {
	var fields []string
	var references []string
	var referencedTable string
	onDelete := "CASCADE" // Default
	onUpdate := "CASCADE" // Default

	// Skip if field is an array (it's the "other side" of the relation)
	if field.Type != nil && field.Type.IsArray {
		return nil
	}

	// Extract fields, references, onDelete, onUpdate from relation attribute
	hasFields := false
	hasReferences := false
	for _, arg := range attr.Arguments {
		switch arg.Name {
		case "fields":
			hasFields = true
			if fieldList, ok := arg.Value.([]interface{}); ok {
				for _, f := range fieldList {
					if fStr, ok := f.(string); ok {
						fields = append(fields, strings.Trim(fStr, `"`))
					}
				}
			}
		case "references":
			hasReferences = true
			if refList, ok := arg.Value.([]interface{}); ok {
				for _, r := range refList {
					if rStr, ok := r.(string); ok {
						references = append(references, strings.Trim(rStr, `"`))
					}
				}
			}
		case "onDelete":
			if delStr, ok := arg.Value.(string); ok {
				onDelete = normalizeCascadeAction(strings.Trim(delStr, `"`))
			}
		case "onUpdate":
			if updStr, ok := arg.Value.(string); ok {
				onUpdate = normalizeCascadeAction(strings.Trim(updStr, `"`))
			}
		}
	}

	// Only process if both fields and references are present
	// This ensures we only create foreign keys for explicit relations
	if !hasFields || !hasReferences {
		return nil
	}

	if len(fields) == 0 || len(references) == 0 {
		return nil
	}

	// Determine referenced table from field type
	// If field type is not a model (e.g., String), we need to find the related field
	// that has the model type (e.g., tenant field with Type: "tenants")
	if field.Type != nil {
		referencedTable = field.Type.Name
		// If field type is not a model, try to find the related field
		if _, isModel := modelMap[referencedTable]; !isModel {
			// Find the related field in the same model that has the model type
			// This happens when we have: id_tenant String @relation(fields: [id_tenant], references: [id_tenant])
			// and tenant tenants @relation(fields: [id_tenant], references: [id_tenant])
			// We need to find the "tenant" field to get the referenced table
			// We'll need to get the model from the context, but we don't have it here
			// So we'll return nil and let the caller handle it
			// Actually, we can look for a field in the same model that has a relation attribute
			// with the same fields/references but with a model type
			return nil // Will be handled by finding the related field in processRelationsAndUnique
		}
	} else {
		return nil
	}

	// Validate that referenced table exists
	if _, exists := modelMap[referencedTable]; !exists {
		return nil
	}

	// Generate foreign key constraint name
	fkName := generateForeignKeyName(tableName, fields)

	return &ForeignKeyDefinition{
		Name:              fkName,
		TableName:         tableName,
		Columns:           fields,
		ReferencedTable:   referencedTable,
		ReferencedColumns: references,
		OnDelete:          onDelete,
		OnUpdate:          onUpdate,
	}
}

// generateForeignKeyName generates a foreign key constraint name
func generateForeignKeyName(tableName string, columns []string) string {
	if len(columns) == 1 {
		return fmt.Sprintf("%s_%s_fkey", tableName, columns[0])
	}
	return fmt.Sprintf("%s_%s_fkey", tableName, columns[0])
}

// normalizeCascadeAction normalizes cascade action values to SQL format
func normalizeCascadeAction(action string) string {
	action = strings.ToUpper(action)
	switch action {
	case "CASCADE", "Cascade":
		return "CASCADE"
	case "SETNULL", "SET_NULL", "SetNull":
		return "SET NULL"
	case "RESTRICT", "Restrict":
		return "RESTRICT"
	case "NOACTION", "NO_ACTION", "NoAction":
		return "NO ACTION"
	default:
		return action
	}
}

// extractDefaultValue extracts default value from an argument
func extractDefaultValue(arg *parser.AttributeArgument) string {
	if str, ok := arg.Value.(string); ok {
		return fmt.Sprintf("'%s'", strings.ReplaceAll(str, "'", "''"))
	}

	// If it's a function (autoincrement, now, dbgenerated, etc.)
	if m, ok := arg.Value.(map[string]interface{}); ok {
		if fn, ok := m["function"].(string); ok {
			switch fn {
			case "autoincrement":
				return "" // Will be treated as SERIAL/BIGSERIAL
			case "now":
				return "CURRENT_TIMESTAMP"
			case "dbgenerated":
				// dbgenerated("gen_random_uuid()") -> extract the argument
				if args, ok := m["args"].([]interface{}); ok && len(args) > 0 {
					if sqlStr, ok := args[0].(string); ok {
						// Return SQL directly, without quotes
						return sqlStr
					}
				}
				return ""
			case "uuid":
				return "" // Client-side generation preferred (no Default in DB)
			}
		}
	}

	return ""
}

// mapTypeToSQL maps Prisma type to SQL
func mapTypeToSQL(prismaType string, provider string) string {
	switch provider {
	case "postgresql":
		return mapTypeToPostgreSQL(prismaType)
	case "mysql":
		return mapTypeToMySQL(prismaType)
	case "sqlite":
		return mapTypeToSQLite(prismaType)
	default:
		return mapTypeToPostgreSQL(prismaType) // Default
	}
}

func mapTypeToPostgreSQL(prismaType string) string {
	switch prismaType {
	case "String":
		return "VARCHAR(255)"
	case "Int":
		return "INTEGER"
	case "BigInt":
		return "BIGINT"
	case "Boolean":
		return "BOOLEAN"
	case "DateTime":
		return "TIMESTAMP"
	case "Float":
		return "DOUBLE PRECISION"
	case "Decimal":
		return "DECIMAL"
	case "Json":
		return "JSONB"
	case "Bytes":
		return "BYTEA"
	case "UUID", "Uuid":
		return "UUID"
	default:
		// If it starts with VARCHAR, return as is (already comes from @db.VarChar)
		if strings.HasPrefix(prismaType, "VARCHAR") {
			return prismaType
		}
		return "TEXT"
	}
}

func mapTypeToMySQL(prismaType string) string {
	switch prismaType {
	case "String":
		return "VARCHAR(255)"
	case "Int":
		return "INT"
	case "BigInt":
		return "BIGINT"
	case "Boolean":
		return "BOOLEAN"
	case "DateTime":
		return "DATETIME"
	case "Float":
		return "DOUBLE"
	case "Decimal":
		return "DECIMAL(65,30)"
	case "Json":
		return "JSON"
	case "Bytes":
		return "BLOB"
	default:
		return "TEXT"
	}
}

func mapTypeToSQLite(prismaType string) string {
	switch prismaType {
	case "String":
		return "TEXT"
	case "Int":
		return "INTEGER"
	case "BigInt":
		return "INTEGER"
	case "Boolean":
		return "INTEGER"
	case "DateTime":
		return "TEXT"
	case "Float":
		return "REAL"
	case "Decimal":
		return "TEXT"
	case "Json":
		return "TEXT"
	case "Bytes":
		return "BLOB"
	default:
		return "TEXT"
	}
}
