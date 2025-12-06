package migrations

import (
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// CompareSchema compara o schema Prisma com o banco de dados e retorna diferenças
func CompareSchema(schema *parser.Schema, dbSchema *DatabaseSchema, provider string) (*SchemaDiff, error) {
	diff := &SchemaDiff{
		TablesToCreate:  []TableDefinition{},
		TablesToAlter:   []TableAlteration{},
		TablesToDrop:    []string{},
		IndexesToCreate: []IndexDefinition{},
		IndexesToDrop:   []string{},
	}

	// Converter schema Prisma para estrutura comparável
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

			// Verificar atributos
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
			// Tabela não existe no banco, precisa criar
			diff.TablesToCreate = append(diff.TablesToCreate, *prismaTable)
		} else {
			// Tabela existe, comparar colunas
			alteration := TableAlteration{
				TableName:    modelName,
				AddColumns:   []ColumnDefinition{},
				DropColumns:  []string{},
				AlterColumns: []ColumnAlteration{},
			}

			// Verificar colunas a adicionar ou alterar
			for _, prismaCol := range prismaTable.Columns {
				dbCol, exists := dbTable.Columns[prismaCol.Name]

				if !exists {
					// Coluna não existe, precisa adicionar
					alteration.AddColumns = append(alteration.AddColumns, prismaCol)
				} else {
					// Coluna existe, verificar se precisa alterar
					needsAlter := false
					colAlter := ColumnAlteration{
						ColumnName:  prismaCol.Name,
						NewType:     prismaCol.Type,
						NewNullable: prismaCol.IsNullable,
					}

					// Comparar tipo
					prismaTypeSQL := mapTypeToSQL(prismaCol.Type, provider)
					if dbCol.Type != prismaTypeSQL {
						needsAlter = true
					}

					// Comparar nullable
					if dbCol.IsNullable != prismaCol.IsNullable {
						needsAlter = true
					}

					if needsAlter {
						alteration.AlterColumns = append(alteration.AlterColumns, colAlter)
					}
				}
			}

			// Verificar colunas a remover (existem no banco mas não no schema)
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

			// Só adicionar alteração se houver mudanças
			if len(alteration.AddColumns) > 0 || len(alteration.DropColumns) > 0 || len(alteration.AlterColumns) > 0 {
				diff.TablesToAlter = append(diff.TablesToAlter, alteration)
			}
		}
	}

	// Verificar tabelas a remover (existem no banco mas não no schema)
	for dbTableName := range dbSchema.Tables {
		_, exists := prismaTables[dbTableName]
		if !exists {
			diff.TablesToDrop = append(diff.TablesToDrop, dbTableName)
		}
	}

	return diff, nil
}
