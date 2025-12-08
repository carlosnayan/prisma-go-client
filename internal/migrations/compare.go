package migrations

import (
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// CompareSchema compares the Prisma schema with the database and returns differences
func CompareSchema(schema *parser.Schema, dbSchema *DatabaseSchema, provider string) (*SchemaDiff, error) {
	diff := &SchemaDiff{
		TablesToCreate:     []TableDefinition{},
		TablesToAlter:      []TableAlteration{},
		TablesToDrop:       []string{},
		IndexesToCreate:    []IndexDefinition{},
		IndexesToDrop:      []string{},
		ForeignKeysToCreate: []ForeignKeyDefinition{},
	}

	// Convert Prisma schema to comparable structure
	prismaTables := make(map[string]*TableDefinition)
	for _, model := range schema.Models {
		// Use @@map if present, otherwise use model name
		tableName := getTableNameFromModel(model)
		table := &TableDefinition{
			Name:        tableName,
			Columns:     []ColumnDefinition{},
			CompositePK: []string{},
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
				if len(pkFields) > 0 {
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
					table.CompositePK = mappedPKFields
				}
			}
		}

		for _, field := range model.Fields {
			// Use @map if present, otherwise use field name
			columnName := getColumnNameFromField(field)
			col := ColumnDefinition{
				Name:       columnName,
				Type:       field.Type.Name,
				IsNullable: field.Type.IsOptional,
			}

			// Check attributes
			// Skip @id if there's a composite PK (handled by @@id)
			hasCompositePK := len(table.CompositePK) > 0
			for _, attr := range field.Attributes {
				switch attr.Name {
				case "id":
					// Only set as PK if there's no composite PK
					if !hasCompositePK {
						col.IsPrimaryKey = true
						col.IsNullable = false
					}
				case "unique":
					col.IsUnique = true
				case "default":
					if len(attr.Arguments) > 0 {
						col.DefaultValue = extractDefaultValue(attr.Arguments[0])
					}
				case "updatedAt":
					// @updatedAt doesn't need special SQL
				case "db.Uuid", "db.UUID":
					col.Type = "UUID"
				case "db.VarChar":
					if len(attr.Arguments) > 0 {
						if size, ok := attr.Arguments[0].Value.(string); ok {
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
					col.Type = "BYTEA"
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

		prismaTables[model.Name] = table
	}

	// Comparar tabelas
	for modelName, prismaTable := range prismaTables {
		dbTable, exists := dbSchema.Tables[modelName]

		if !exists {
			// Table doesn't exist in database, needs to be created
			diff.TablesToCreate = append(diff.TablesToCreate, *prismaTable)
		} else {
			// Table exists, compare columns
			alteration := TableAlteration{
				TableName:    modelName,
				AddColumns:   []ColumnDefinition{},
				DropColumns:  []string{},
				AlterColumns: []ColumnAlteration{},
			}

			// Check columns to add or alter
			for _, prismaCol := range prismaTable.Columns {
				dbCol, exists := dbTable.Columns[prismaCol.Name]

				if !exists {
					// Column doesn't exist, needs to be added
					alteration.AddColumns = append(alteration.AddColumns, prismaCol)
				} else {
					// Column exists, check if it needs to be altered
					needsAlter := false
					colAlter := ColumnAlteration{
						ColumnName:  prismaCol.Name,
						NewType:     prismaCol.Type,
						NewNullable: prismaCol.IsNullable,
					}

					// Compare type
					prismaTypeSQL := mapTypeToSQL(prismaCol.Type, provider)
					if dbCol.Type != prismaTypeSQL {
						needsAlter = true
					}

					// Compare nullable
					if dbCol.IsNullable != prismaCol.IsNullable {
						needsAlter = true
					}

					if needsAlter {
						alteration.AlterColumns = append(alteration.AlterColumns, colAlter)
					}
				}
			}

			// Check columns to remove (exist in database but not in schema)
			for dbColName := range dbTable.Columns {
				found := false
				for _, prismaCol := range prismaTable.Columns {
					if prismaCol.Name == dbColName {
						found = true
						break
					}
				}
				if !found {
					alteration.DropColumns = append(alteration.DropColumns, dbColName)
				}
			}

			// Only add alteration if there are changes
			if len(alteration.AddColumns) > 0 || len(alteration.DropColumns) > 0 || len(alteration.AlterColumns) > 0 {
				diff.TablesToAlter = append(diff.TablesToAlter, alteration)
			}
		}
	}

	// Check tables to remove (exist in database but not in schema)
	for dbTableName := range dbSchema.Tables {
		_, exists := prismaTables[dbTableName]
		if !exists {
			diff.TablesToDrop = append(diff.TablesToDrop, dbTableName)
		}
	}

	// Process relations, @@unique, and @@index attributes
	processRelationsAndUnique(schema, diff)

	return diff, nil
}

// processRelationsAndUnique processes @relation attributes, @@unique, and @@index attributes
func processRelationsAndUnique(schema *parser.Schema, diff *SchemaDiff) {
	// Create a map of model names for quick lookup
	modelMap := make(map[string]*parser.Model)
	for _, model := range schema.Models {
		modelMap[model.Name] = model
	}

	// Process each model
	for _, model := range schema.Models {
		// Get mapped table name
		tableName := getTableNameFromModel(model)
		
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
						// Map referenced table and columns if referenced table has @@map
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
