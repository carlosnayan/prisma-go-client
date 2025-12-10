package migrations

import (
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func isBuiltInType(typeName string) bool {
	builtInTypes := []string{
		"String", "Int", "Float", "Boolean", "DateTime", "Json", "Bytes",
		"BigInt", "Decimal",
	}
	for _, t := range builtInTypes {
		if t == typeName {
			return true
		}
	}
	return false
}

func CompareSchema(schema *parser.Schema, dbSchema *DatabaseSchema, provider string) (*SchemaDiff, error) {
	diff := &SchemaDiff{
		TablesToCreate:      []TableDefinition{},
		TablesToAlter:       []TableAlteration{},
		TablesToDrop:        []string{},
		IndexesToCreate:     []IndexDefinition{},
		IndexesToDrop:       []string{},
		ForeignKeysToCreate: []ForeignKeyDefinition{},
		ForeignKeysToAlter:  []ForeignKeyDefinition{},
		ForeignKeysToDrop:   []ForeignKeyDefinition{},
	}

	prismaTables := make(map[string]*TableDefinition)
	for _, model := range schema.Models {
		tableName := getTableNameFromModel(model)
		table := &TableDefinition{
			Name:        tableName,
			Columns:     []ColumnDefinition{},
			CompositePK: []string{},
		}

		for _, attr := range model.Attributes {
			if attr.Name == "id" {
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
					mappedPKFields := make([]string, len(pkFields))
					for i, pkField := range pkFields {
						for _, field := range model.Fields {
							if field.Name == pkField {
								mappedPKFields[i] = getColumnNameFromField(field)
								break
							}
						}
						if mappedPKFields[i] == "" {
							mappedPKFields[i] = pkField
						}
					}
					table.CompositePK = mappedPKFields
				}
			}
		}

		for _, field := range model.Fields {
			cleanTypeName := strings.TrimSuffix(strings.TrimSuffix(field.Type.Name, "[]"), "?")
			isRelationField := strings.HasSuffix(field.Type.Name, "[]") || (!isBuiltInType(cleanTypeName) && isModel(schema, cleanTypeName))

			if isRelationField {
				continue
			}

			columnName := getColumnNameFromField(field)
			col := ColumnDefinition{
				Name:       columnName,
				Type:       field.Type.Name,
				IsNullable: field.Type.IsOptional,
			}

			hasCompositePK := len(table.CompositePK) > 0
			for _, attr := range field.Attributes {
				switch attr.Name {
				case "id":
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

		prismaTables[tableName] = table
	}

	for tableName, prismaTable := range prismaTables {
		dbTable, exists := dbSchema.Tables[tableName]
		if !exists {
			diff.TablesToCreate = append(diff.TablesToCreate, *prismaTable)
			continue
		}

		alteration := TableAlteration{
			TableName:    tableName,
			AddColumns:   []ColumnDefinition{},
			DropColumns:  []string{},
			AlterColumns: []ColumnAlteration{},
		}

		for _, prismaCol := range prismaTable.Columns {
			dbCol, exists := dbTable.Columns[prismaCol.Name]
			if !exists {
				alteration.AddColumns = append(alteration.AddColumns, prismaCol)
				continue
			}

			prismaTypeSQL := mapTypeToSQL(prismaCol.Type, provider)
			if dbCol.Type != prismaTypeSQL || dbCol.IsNullable != prismaCol.IsNullable {
				alteration.AlterColumns = append(alteration.AlterColumns, ColumnAlteration{
					ColumnName:  prismaCol.Name,
					NewType:     prismaCol.Type,
					NewNullable: prismaCol.IsNullable,
				})
			}
		}

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

		if len(alteration.AddColumns) > 0 || len(alteration.DropColumns) > 0 || len(alteration.AlterColumns) > 0 {
			diff.TablesToAlter = append(diff.TablesToAlter, alteration)
		}
	}

	for dbTableName := range dbSchema.Tables {
		if _, exists := prismaTables[dbTableName]; !exists {
			diff.TablesToDrop = append(diff.TablesToDrop, dbTableName)
		}
	}

	// Calculate Indexes to Drop
	expectedIndexes := make(map[string]map[string]bool)
	for _, model := range schema.Models {
		tableName := getTableNameFromModel(model)
		expectedIndexes[tableName] = make(map[string]bool)

		// Field level @unique
		for _, field := range model.Fields {
			colName := getColumnNameFromField(field)
			for _, attr := range field.Attributes {
				if attr.Name == "unique" {
					indexName := fmt.Sprintf("%s_%s_key", tableName, colName)
					expectedIndexes[tableName][indexName] = true
				}
			}
		}

		// Model level @@unique and @@index
		for _, attr := range model.Attributes {
			if attr.Name == "unique" {
				if idx := extractUniqueIndex(tableName, attr); idx != nil {
					expectedIndexes[tableName][idx.Name] = true
				}
			} else if attr.Name == "index" {
				if idx := extractIndex(tableName, attr); idx != nil {
					expectedIndexes[tableName][idx.Name] = true
				}
			}
		}
	}

	for tableName, dbTable := range dbSchema.Tables {
		// Skip if table is being dropped
		if _, exists := prismaTables[tableName]; !exists {
			continue
		}

		for _, dbIdx := range dbTable.Indexes {
			// Check if index is expected (case-insensitive)
			expected := false
			if expectedMap, ok := expectedIndexes[tableName]; ok {
				for expectedName := range expectedMap {
					if strings.EqualFold(expectedName, dbIdx.Name) {
						expected = true
						break
					}
				}
			}

			if !expected {
				diff.IndexesToDrop = append(diff.IndexesToDrop, dbIdx.Name)
			}
		}
	}

	processRelationsAndUnique(schema, diff, dbSchema)

	// Detect FKs that exist in database but not in schema (need to be dropped)
	detectOrphanedForeignKeys(schema, diff, dbSchema)

	return diff, nil
}

// detectOrphanedForeignKeys finds foreign keys that exist in the database but not in the schema
func detectOrphanedForeignKeys(schema *parser.Schema, diff *SchemaDiff, dbSchema *DatabaseSchema) {
	modelMap := make(map[string]*parser.Model)
	for _, model := range schema.Models {
		modelMap[model.Name] = model
	}

	// Build a set of expected FKs from schema (by structure: table, columns, referenced table/columns)
	expectedFKs := make(map[string]bool)
	for _, model := range schema.Models {
		tableName := getTableNameFromModel(model)
		for _, field := range model.Fields {
			for _, attr := range field.Attributes {
				if attr.Name == "relation" {
					fkDef := extractForeignKey(tableName, field, attr, modelMap)
					if fkDef != nil {
						fkDef.Columns = mapColumnNames(model, fkDef.Columns)
						if refModel, exists := modelMap[fkDef.ReferencedTable]; exists {
							fkDef.ReferencedTable = getTableNameFromModel(refModel)
							fkDef.ReferencedColumns = mapColumnNames(refModel, fkDef.ReferencedColumns)
						}
						// Create a unique key based on structure (not name, as name might differ)
						fkKey := createFKStructureKey(fkDef.TableName, fkDef.Columns, fkDef.ReferencedTable, fkDef.ReferencedColumns)
						expectedFKs[fkKey] = true
					}
				}
			}
		}
	}

	// Check all FKs in database
	for tableName, dbTable := range dbSchema.Tables {
		// Skip if table is being dropped
		tableBeingDropped := false
		for _, dropTable := range diff.TablesToDrop {
			if dropTable == tableName {
				tableBeingDropped = true
				break
			}
		}
		if tableBeingDropped {
			continue
		}

		for _, dbFK := range dbTable.ForeignKeys {
			// Create structure key for this FK
			fkKey := createFKStructureKey(tableName, dbFK.Columns, dbFK.ReferencedTable, dbFK.ReferencedColumns)

			// Check if this FK structure is expected in schema
			if !expectedFKs[fkKey] {
				// Also check if it's in ForeignKeysToAlter (might be same structure but different attributes)
				isBeingAltered := false
				for _, alterFK := range diff.ForeignKeysToAlter {
					alterKey := createFKStructureKey(alterFK.TableName, alterFK.Columns, alterFK.ReferencedTable, alterFK.ReferencedColumns)
					if alterKey == fkKey {
						isBeingAltered = true
						break
					}
				}

				// If not expected and not being altered, mark for removal
				if !isBeingAltered {
					diff.ForeignKeysToDrop = append(diff.ForeignKeysToDrop, ForeignKeyDefinition{
						Name:              dbFK.Name,
						TableName:         tableName,
						Columns:           dbFK.Columns,
						ReferencedTable:   dbFK.ReferencedTable,
						ReferencedColumns: dbFK.ReferencedColumns,
						OnDelete:          dbFK.OnDelete,
						OnUpdate:          dbFK.OnUpdate,
					})
				}
			}
		}
	}
}

// createFKStructureKey creates a unique key for FK based on its structure
func createFKStructureKey(tableName string, columns []string, referencedTable string, referencedColumns []string) string {
	// Normalize column names for comparison
	colsKey := strings.Join(columns, ",")
	refColsKey := strings.Join(referencedColumns, ",")
	return fmt.Sprintf("%s|%s|%s|%s", strings.ToLower(tableName), strings.ToLower(colsKey), strings.ToLower(referencedTable), strings.ToLower(refColsKey))
}

func processRelationsAndUnique(schema *parser.Schema, diff *SchemaDiff, dbSchema *DatabaseSchema) {
	modelMap := make(map[string]*parser.Model)
	for _, model := range schema.Models {
		modelMap[model.Name] = model
	}

	for _, model := range schema.Models {
		tableName := getTableNameFromModel(model)

		for _, attr := range model.Attributes {
			var indexDef *IndexDefinition
			if attr.Name == "unique" {
				indexDef = extractUniqueIndex(tableName, attr)
			} else if attr.Name == "index" {
				indexDef = extractIndex(tableName, attr)
			}
			if indexDef != nil {
				mappedColumns := mapColumnNames(model, indexDef.Columns)
				indexDef.Columns = mappedColumns
				if !indexExists(dbSchema, tableName, indexDef.Name, indexDef.Columns) {
					diff.IndexesToCreate = append(diff.IndexesToCreate, *indexDef)
				}
			}
		}

		for _, field := range model.Fields {
			columnName := getColumnNameFromField(field)
			for _, attr := range field.Attributes {
				if attr.Name == "unique" {
					// Field-level unique attribute
					indexName := fmt.Sprintf("%s_%s_key", tableName, columnName)
					if !indexExists(dbSchema, tableName, indexName, []string{columnName}) {
						diff.IndexesToCreate = append(diff.IndexesToCreate, IndexDefinition{
							Name:      indexName,
							TableName: tableName,
							Columns:   []string{columnName},
							IsUnique:  true,
						})
					}
				}
				if attr.Name == "relation" {
					fkDef := extractForeignKey(tableName, field, attr, modelMap)
					// If extractForeignKey returned nil because field type is not a model,
					// try to find the related field that has the model type
					if fkDef == nil {
						// Look for a related field in the same model that has the model type
						// and shares the same relation (same fields/references)
						var relationFields []string
						var relationReferences []string
						for _, arg := range attr.Arguments {
							if arg.Name == "fields" {
								if fieldList, ok := arg.Value.([]interface{}); ok {
									for _, f := range fieldList {
										if fStr, ok := f.(string); ok {
											relationFields = append(relationFields, strings.Trim(fStr, `"`))
										}
									}
								}
							}
							if arg.Name == "references" {
								if refList, ok := arg.Value.([]interface{}); ok {
									for _, r := range refList {
										if rStr, ok := r.(string); ok {
											relationReferences = append(relationReferences, strings.Trim(rStr, `"`))
										}
									}
								}
							}
						}

						// Find the related field that has a model type and the same relation
						for _, relatedField := range model.Fields {
							if relatedField.Name == field.Name {
								continue // Skip the same field
							}
							// Check if this field has a model type
							if relatedField.Type != nil {
								cleanTypeName := strings.TrimSuffix(strings.TrimSuffix(relatedField.Type.Name, "[]"), "?")
								if _, isModel := modelMap[cleanTypeName]; isModel {
									// Check if this field has the same relation
									for _, relatedAttr := range relatedField.Attributes {
										if relatedAttr.Name == "relation" {
											var relatedFields []string
											var relatedReferences []string
											for _, arg := range relatedAttr.Arguments {
												if arg.Name == "fields" {
													if fieldList, ok := arg.Value.([]interface{}); ok {
														for _, f := range fieldList {
															if fStr, ok := f.(string); ok {
																relatedFields = append(relatedFields, strings.Trim(fStr, `"`))
															}
														}
													}
												}
												if arg.Name == "references" {
													if refList, ok := arg.Value.([]interface{}); ok {
														for _, r := range refList {
															if rStr, ok := r.(string); ok {
																relatedReferences = append(relatedReferences, strings.Trim(rStr, `"`))
															}
														}
													}
												}
											}
											// Check if fields and references match
											if len(relatedFields) == len(relationFields) && len(relatedReferences) == len(relationReferences) {
												match := true
												for i := range relationFields {
													if relationFields[i] != relatedFields[i] {
														match = false
														break
													}
												}
												if match {
													for i := range relationReferences {
														if relationReferences[i] != relatedReferences[i] {
															match = false
															break
														}
													}
												}
												if match {
													// Found the related field, extract FK using it
													fkDef = extractForeignKey(tableName, relatedField, relatedAttr, modelMap)
													if fkDef != nil {
														// Update fields/references from the original field's relation
														fkDef.Columns = relationFields
														fkDef.ReferencedColumns = relationReferences
													}
													break
												}
											}
										}
									}
									if fkDef != nil {
										break
									}
								}
							}
						}
					}

					if fkDef != nil {
						fkDef.Columns = mapColumnNames(model, fkDef.Columns)
						if refModel, exists := modelMap[fkDef.ReferencedTable]; exists {
							fkDef.ReferencedTable = getTableNameFromModel(refModel)
							fkDef.ReferencedColumns = mapColumnNames(refModel, fkDef.ReferencedColumns)
						}

						// Extract onDelete/onUpdate from the original field's relation attribute
						for _, arg := range attr.Arguments {
							if arg.Name == "onDelete" {
								if delStr, ok := arg.Value.(string); ok {
									fkDef.OnDelete = normalizeCascadeAction(strings.Trim(delStr, `"`))
								}
							}
							if arg.Name == "onUpdate" {
								if updStr, ok := arg.Value.(string); ok {
									fkDef.OnUpdate = normalizeCascadeAction(strings.Trim(updStr, `"`))
								}
							}
						}

						// Check if FK exists and if it needs alteration
						// Note: fkDef.Name might not match database FK name, so we rely on structure matching
						exists, needsAlter := foreignKeyMatches(dbSchema, fkDef.TableName, fkDef.Name, fkDef.Columns, fkDef.ReferencedTable, fkDef.ReferencedColumns, fkDef.OnDelete, fkDef.OnUpdate)

						if !exists {
							// FK doesn't exist, needs to be created
							diff.ForeignKeysToCreate = append(diff.ForeignKeysToCreate, *fkDef)
						} else if needsAlter {
							// FK exists but OnDelete/OnUpdate don't match, needs alteration
							diff.ForeignKeysToAlter = append(diff.ForeignKeysToAlter, *fkDef)
						}
						// If exists and doesn't need alter, do nothing
					}
				}
			}
		}
	}
}

func mapColumnNames(model *parser.Model, columns []string) []string {
	mapped := make([]string, len(columns))
	for i, col := range columns {
		for _, field := range model.Fields {
			if field.Name == col {
				mapped[i] = getColumnNameFromField(field)
				break
			}
		}
		if mapped[i] == "" {
			mapped[i] = col
		}
	}
	return mapped
}

func isModel(schema *parser.Schema, typeName string) bool {
	for _, m := range schema.Models {
		if m.Name == typeName {
			return true
		}
	}
	return false
}

// foreignKeyExists checks if a foreign key exists (backward compatibility)
// This function is kept for backward compatibility with existing code that may call it.
// It's a wrapper around foreignKeyMatches that only checks existence, not attributes.
// nolint:unused // Kept for backward compatibility
func foreignKeyExists(dbSchema *DatabaseSchema, tableName, fkName string, columns []string, referencedTable string, referencedColumns []string) bool {
	exists, _ := foreignKeyMatches(dbSchema, tableName, fkName, columns, referencedTable, referencedColumns, "", "")
	return exists
}

// foreignKeyMatches verifies if a foreign key exists and if its attributes (onDelete, onUpdate) match
// Returns: (exists, needsAlter)
// exists: true if FK exists in database
// needsAlter: true if FK exists but OnDelete/OnUpdate don't match
func foreignKeyMatches(dbSchema *DatabaseSchema, tableName, fkName string, columns []string, referencedTable string, referencedColumns []string, onDelete, onUpdate string) (bool, bool) {
	dbTable, exists := dbSchema.Tables[tableName]
	if !exists {
		return false, false
	}

	// If table has no FKs, return false
	if len(dbTable.ForeignKeys) == 0 {
		return false, false
	}

	// Normalize expected values
	normalizedOnDelete := normalizeCascadeActionForComparison(onDelete)
	normalizedOnUpdate := normalizeCascadeActionForComparison(onUpdate)

	// Default values if not specified
	if normalizedOnDelete == "" {
		normalizedOnDelete = "CASCADE"
	}
	if normalizedOnUpdate == "" {
		normalizedOnUpdate = "CASCADE"
	}

	for _, dbFK := range dbTable.ForeignKeys {
		// Try matching by name first (most reliable)
		nameMatch := strings.EqualFold(dbFK.Name, fkName)

		// Try matching by structure
		structureMatch := strings.EqualFold(dbFK.ReferencedTable, referencedTable) &&
			len(dbFK.Columns) == len(columns) &&
			len(dbFK.ReferencedColumns) == len(referencedColumns) &&
			columnsMatch(dbFK.Columns, columns) &&
			columnsMatch(dbFK.ReferencedColumns, referencedColumns)

		if nameMatch || structureMatch {
			// Normalize database values
			dbOnDelete := normalizeCascadeActionForComparison(dbFK.OnDelete)
			dbOnUpdate := normalizeCascadeActionForComparison(dbFK.OnUpdate)

			// Default values for database if empty
			if dbOnDelete == "" {
				dbOnDelete = "CASCADE"
			}
			if dbOnUpdate == "" {
				dbOnUpdate = "CASCADE"
			}

			// Check if attributes match
			if dbOnDelete != normalizedOnDelete || dbOnUpdate != normalizedOnUpdate {
				return true, true // Exists but needs alteration
			}
			return true, false // Exists and matches
		}
	}
	return false, false // Doesn't exist
}

// normalizeCascadeActionForComparison normalizes cascade action values for comparison
// Handles various formats from database and schema
// Returns normalized value that matches the format used by normalizeCascadeAction
func normalizeCascadeActionForComparison(action string) string {
	if action == "" {
		return ""
	}
	// Use the same normalization as normalizeCascadeAction
	return normalizeCascadeAction(action)
}

func indexExists(dbSchema *DatabaseSchema, tableName, indexName string, columns []string) bool {
	dbTable, exists := dbSchema.Tables[tableName]
	if !exists {
		return false
	}

	for _, dbIndex := range dbTable.Indexes {
		if strings.EqualFold(dbIndex.Name, indexName) {
			return true
		}
		if len(dbIndex.Columns) == len(columns) && columnsMatch(dbIndex.Columns, columns) {
			return true
		}
	}
	return false
}

func columnsMatch(cols1, cols2 []string) bool {
	if len(cols1) != len(cols2) {
		return false
	}
	for i, col := range cols1 {
		if !strings.EqualFold(col, cols2[i]) {
			return false
		}
	}
	return true
}
