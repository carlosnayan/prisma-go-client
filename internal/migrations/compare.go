package migrations

import (
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// CompareSchema compares the Prisma schema with the database and returns differences
func CompareSchema(schema *parser.Schema, dbSchema *DatabaseSchema, provider string) (*SchemaDiff, error) {
	diff := &SchemaDiff{
		TablesToCreate:  []TableDefinition{},
		TablesToAlter:   []TableAlteration{},
		TablesToDrop:    []string{},
		IndexesToCreate: []IndexDefinition{},
		IndexesToDrop:   []string{},
	}

	// Convert Prisma schema to comparable structure
	prismaTables := make(map[string]*TableDefinition)
	for _, model := range schema.Models {
		table := &TableDefinition{
			Name:    model.Name,
			Columns: []ColumnDefinition{},
		}

		for _, field := range model.Fields {
			col := ColumnDefinition{
				Name:       field.Name,
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
				case "default":
					if len(attr.Arguments) > 0 {
						col.DefaultValue = extractDefaultValue(attr.Arguments[0])
					}
				case "db.Uuid", "db.UUID":
					// @db.Uuid sobrescreve o tipo para UUID
					col.Type = "UUID"
				case "db.VarChar":
					// @db.VarChar(255) - extrair tamanho se houver
					if len(attr.Arguments) > 0 {
						if size, ok := attr.Arguments[0].Value.(string); ok {
							col.Type = "VARCHAR(" + size + ")"
						} else {
							col.Type = "VARCHAR(255)"
						}
					} else {
						col.Type = "VARCHAR(255)"
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

	return diff, nil
}
