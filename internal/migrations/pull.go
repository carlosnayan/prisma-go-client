package migrations

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

type EnumInfo struct {
	Name   string
	Values []string
}

func GenerateSchemaFromDatabase(dbSchema *DatabaseSchema, provider string, db *sql.DB) (*parser.Schema, error) {
	schema := &parser.Schema{
		Datasources: []*parser.Datasource{},
		Generators:  []*parser.Generator{},
		Models:      []*parser.Model{},
		Enums:       []*parser.Enum{},
	}

	enums := []*EnumInfo{}
	if provider == "postgresql" {
		enums = detectPostgreSQLEnumsWithValues(dbSchema, db)
	}

	enumMap := make(map[string]*EnumInfo)
	for _, e := range enums {
		enumMap[e.Name] = e
		enum := &parser.Enum{
			Name:   e.Name,
			Values: []*parser.EnumValue{},
		}
		for _, val := range e.Values {
			enum.Values = append(enum.Values, &parser.EnumValue{
				Name:       val,
				Attributes: []*parser.Attribute{},
			})
		}
		schema.Enums = append(schema.Enums, enum)
	}

	sort.Slice(schema.Enums, func(i, j int) bool {
		return schema.Enums[i].Name < schema.Enums[j].Name
	})

	tableNames := make([]string, 0, len(dbSchema.Tables))
	for tableName := range dbSchema.Tables {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)

	for _, tableName := range tableNames {
		tableInfo := dbSchema.Tables[tableName]
		model := convertTableToModel(tableName, tableInfo, dbSchema, provider, enumMap)
		schema.Models = append(schema.Models, model)
	}

	return schema, nil
}

func detectPostgreSQLEnumsWithValues(dbSchema *DatabaseSchema, db *sql.DB) []*EnumInfo {
	enumMap := make(map[string]map[string]bool)

	for _, table := range dbSchema.Tables {
		for _, col := range table.Columns {
			if col.Type == "USER-DEFINED" && col.UdtName != "" {
				enumName := col.UdtName
				if enumMap[enumName] == nil {
					enumMap[enumName] = make(map[string]bool)
				}
			}
		}
	}

	enums := make([]*EnumInfo, 0, len(enumMap))
	for enumName := range enumMap {
		values := getPostgreSQLEnumValues(db, enumName)
		enums = append(enums, &EnumInfo{
			Name:   enumName,
			Values: values,
		})
	}

	return enums
}

func getPostgreSQLEnumValues(db *sql.DB, enumName string) []string {
	query := `
		SELECT enumlabel
		FROM pg_enum e
		JOIN pg_type t ON e.enumtypid = t.oid
		WHERE t.typname = $1
		ORDER BY e.enumsortorder
	`

	rows, err := db.Query(query, enumName)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err == nil {
			values = append(values, value)
		}
	}

	return values
}

func convertTableToModel(tableName string, tableInfo *TableInfo, dbSchema *DatabaseSchema, provider string, enumMap map[string]*EnumInfo) *parser.Model {
	model := &parser.Model{
		Name:       tableName,
		Fields:     []*parser.ModelField{},
		Attributes: []*parser.Attribute{},
	}

	// Use original column order from database (ordinal_position)
	columnNames := tableInfo.ColumnOrder
	if len(columnNames) == 0 {
		// Fallback: if ColumnOrder is empty, use all column names
		columnNames = make([]string, 0, len(tableInfo.Columns))
		for colName := range tableInfo.Columns {
			columnNames = append(columnNames, colName)
		}
	}

	for _, colName := range columnNames {
		colInfo := tableInfo.Columns[colName]
		field := convertColumnToField(colName, colInfo, provider, enumMap)
		model.Fields = append(model.Fields, field)
	}

	relationFields := generateRelations(tableName, tableInfo, dbSchema)
	model.Fields = append(model.Fields, relationFields...)

	indexes := generateIndexes(tableName, tableInfo)
	model.Attributes = append(model.Attributes, indexes...)

	return model
}

func convertColumnToField(colName string, colInfo *ColumnInfo, provider string, enumMap map[string]*EnumInfo) *parser.ModelField {
	field := &parser.ModelField{
		Name:       colName,
		Type:       convertTypeToPrisma(colInfo, provider, enumMap),
		Attributes: []*parser.Attribute{},
	}

	if colInfo.IsPrimaryKey {
		field.Attributes = append(field.Attributes, &parser.Attribute{
			Name:      "id",
			Arguments: []*parser.AttributeArgument{},
		})
	}

	if colInfo.IsUnique && !colInfo.IsPrimaryKey {
		field.Attributes = append(field.Attributes, &parser.Attribute{
			Name:      "unique",
			Arguments: []*parser.AttributeArgument{},
		})
	}

	if colInfo.DefaultValue != nil && *colInfo.DefaultValue != "" {
		defaultAttr := convertDefaultValue(*colInfo.DefaultValue, colInfo.Type, provider, colInfo.UdtName, enumMap)
		if defaultAttr != nil {
			field.Attributes = append(field.Attributes, defaultAttr)
		}
	}

	dbAttr := convertToDbAttribute(colInfo, provider)
	if dbAttr != nil {
		field.Attributes = append(field.Attributes, dbAttr)
	}

	return field
}

func convertTypeToPrisma(colInfo *ColumnInfo, provider string, enumMap map[string]*EnumInfo) *parser.FieldType {
	dbType := strings.ToLower(colInfo.Type)
	prismaType := mapDatabaseTypeToPrisma(dbType, colInfo.UdtName, provider, enumMap)

	return &parser.FieldType{
		Name:       prismaType,
		IsOptional: colInfo.IsNullable,
		IsArray:    false,
	}
}

func mapDatabaseTypeToPrisma(dbType, udtName string, provider string, enumMap map[string]*EnumInfo) string {
	dbType = strings.ToLower(strings.TrimSpace(dbType))
	udtNameOriginal := strings.TrimSpace(udtName)
	udtNameLower := strings.ToLower(udtNameOriginal)

	if provider == "postgresql" {
		// Check enum map with both original case and lowercase
		if udtNameOriginal != "" {
			if enumInfo, exists := enumMap[udtNameOriginal]; exists {
				return enumInfo.Name
			}
			if enumInfo, exists := enumMap[udtNameLower]; exists {
				return enumInfo.Name
			}
		}
		if udtNameLower == "uuid" || strings.Contains(dbType, "uuid") {
			return "String"
		}
		if strings.Contains(dbType, "character varying") || strings.Contains(dbType, "varchar") {
			return "String"
		}
		if dbType == "text" || dbType == "char" || dbType == "character" {
			return "String"
		}
		if dbType == "integer" || dbType == "int" || dbType == "int4" {
			return "Int"
		}
		if dbType == "bigint" || dbType == "int8" {
			return "BigInt"
		}
		if dbType == "boolean" || dbType == "bool" {
			return "Boolean"
		}
		if strings.Contains(dbType, "timestamp") || udtNameLower == "timestamptz" || udtNameLower == "timestamp" {
			return "DateTime"
		}
		if dbType == "date" || dbType == "time" {
			return "DateTime"
		}
		if dbType == "real" || dbType == "double precision" || dbType == "float8" || dbType == "float4" {
			return "Float"
		}
		if strings.HasPrefix(dbType, "numeric") || strings.HasPrefix(dbType, "decimal") {
			return "Decimal"
		}
		if dbType == "jsonb" || dbType == "json" || udtNameLower == "jsonb" || udtNameLower == "json" {
			return "Json"
		}
		if dbType == "bytea" {
			return "Bytes"
		}
	}

	return "String"
}

func convertDefaultValue(defaultVal, dbType string, provider string, udtName string, enumMap map[string]*EnumInfo) *parser.Attribute {
	defaultVal = strings.TrimSpace(defaultVal)
	dbType = strings.ToLower(dbType)

	if strings.Contains(defaultVal, "nextval") || strings.Contains(defaultVal, "gen_random_uuid") {
		if strings.Contains(defaultVal, "gen_random_uuid") {
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: map[string]interface{}{
						"function": "dbgenerated",
						"args":     []interface{}{"gen_random_uuid()"},
					}},
				},
			}
		}
		return &parser.Attribute{
			Name: "default",
			Arguments: []*parser.AttributeArgument{
				{Value: map[string]interface{}{
					"function": "autoincrement",
					"args":     []interface{}{},
				}},
			},
		}
	}

	if defaultVal == "now()" || defaultVal == "CURRENT_TIMESTAMP" || strings.Contains(defaultVal, "now()") {
		return &parser.Attribute{
			Name: "default",
			Arguments: []*parser.AttributeArgument{
				{Value: map[string]interface{}{
					"function": "now",
					"args":     []interface{}{},
				}},
			},
		}
	}

	if strings.HasSuffix(defaultVal, "::jsonb") {
		val := strings.TrimSuffix(defaultVal, "::jsonb")
		val = strings.TrimSpace(val)
		if val == "'{}'" || val == "''::jsonb" {
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: "{}"},
				},
			}
		}
		// Handle JSON values with escaped quotes
		if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
			jsonVal := strings.Trim(val, "'")
			// Unescape PostgreSQL string escapes ('' -> ')
			jsonVal = strings.ReplaceAll(jsonVal, "''", "'")
			// Return as string without additional escaping (will be quoted by formatter)
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: jsonVal},
				},
			}
		}
	}

	if strings.HasSuffix(defaultVal, "::json") {
		val := strings.TrimSuffix(defaultVal, "::json")
		val = strings.TrimSpace(val)
		if val == "'{}'" || val == "''::json" {
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: "{}"},
				},
			}
		}
		// Handle JSON values with escaped quotes
		if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
			jsonVal := strings.Trim(val, "'")
			// Unescape PostgreSQL string escapes ('' -> ')
			jsonVal = strings.ReplaceAll(jsonVal, "''", "'")
			// Return as string without additional escaping (will be quoted by formatter)
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: jsonVal},
				},
			}
		}
	}

	// Check for '{}'::jsonb or '{}'::json at the beginning
	if strings.Contains(defaultVal, "'{}'::") {
		return &parser.Attribute{
			Name: "default",
			Arguments: []*parser.AttributeArgument{
				{Value: "{}"},
			},
		}
	}

	if strings.Contains(defaultVal, "::") {
		parts := strings.Split(defaultVal, "::")
		if len(parts) > 0 {
			val := strings.TrimSpace(parts[0])
			if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
				val = strings.Trim(val, "'")
				if val == "true" {
					return &parser.Attribute{
						Name: "default",
						Arguments: []*parser.AttributeArgument{
							{Value: true},
						},
					}
				}
				if val == "false" {
					return &parser.Attribute{
						Name: "default",
						Arguments: []*parser.AttributeArgument{
							{Value: false},
						},
					}
				}
				if udtName != "" && enumMap[udtName] != nil {
					for _, enumVal := range enumMap[udtName].Values {
						if val == enumVal {
							return &parser.Attribute{
								Name: "default",
								Arguments: []*parser.AttributeArgument{
									{Value: val},
								},
							}
						}
					}
				}
				if val == "{}" {
					return &parser.Attribute{
						Name: "default",
						Arguments: []*parser.AttributeArgument{
							{Value: "{}"},
						},
					}
				}
				// Handle empty string specially
				if val == "" {
					return &parser.Attribute{
						Name: "default",
						Arguments: []*parser.AttributeArgument{
							{Value: ""},
						},
					}
				}
				return &parser.Attribute{
					Name: "default",
					Arguments: []*parser.AttributeArgument{
						{Value: fmt.Sprintf("%q", val)},
					},
				}
			}
		}
	}

	if strings.HasPrefix(defaultVal, "'") && strings.HasSuffix(defaultVal, "'") {
		val := strings.Trim(defaultVal, "'")
		if val == "true" {
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: true},
				},
			}
		}
		if val == "false" {
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: false},
				},
			}
		}
		if udtName != "" && enumMap[udtName] != nil {
			for _, enumVal := range enumMap[udtName].Values {
				if val == enumVal {
					return &parser.Attribute{
						Name: "default",
						Arguments: []*parser.AttributeArgument{
							{Value: val},
						},
					}
				}
			}
		}
		if val == "{}" {
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: "{}"},
				},
			}
		}
		// Handle empty string specially
		if val == "" {
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: ""},
				},
			}
		}
		return &parser.Attribute{
			Name: "default",
			Arguments: []*parser.AttributeArgument{
				{Value: fmt.Sprintf("%q", val)},
			},
		}
	}

	if defaultVal == "true" {
		return &parser.Attribute{
			Name: "default",
			Arguments: []*parser.AttributeArgument{
				{Value: true},
			},
		}
	}
	if defaultVal == "false" {
		return &parser.Attribute{
			Name: "default",
			Arguments: []*parser.AttributeArgument{
				{Value: false},
			},
		}
	}

	if defaultVal != "" {
		if intVal, err := strconv.Atoi(defaultVal); err == nil {
			return &parser.Attribute{
				Name: "default",
				Arguments: []*parser.AttributeArgument{
					{Value: intVal},
				},
			}
		}
	}

	return nil
}

func convertToDbAttribute(colInfo *ColumnInfo, provider string) *parser.Attribute {
	dbType := strings.ToLower(strings.TrimSpace(colInfo.Type))
	udtName := strings.ToLower(strings.TrimSpace(colInfo.UdtName))

	if provider == "postgresql" {
		if udtName == "uuid" {
			return &parser.Attribute{
				Name:      "db.Uuid",
				Arguments: []*parser.AttributeArgument{},
			}
		}
		// Check for VARCHAR with length - both data_type and udt_name can indicate this
		if (strings.Contains(dbType, "character varying") || strings.Contains(dbType, "varchar")) && colInfo.CharacterMaximumLength != nil {
			return &parser.Attribute{
				Name: "db.VarChar",
				Arguments: []*parser.AttributeArgument{
					{Value: *colInfo.CharacterMaximumLength},
				},
			}
		}
		if strings.Contains(dbType, "char(") {
			size := extractSizeFromDataType(dbType)
			if size != "" {
				if sizeInt, err := strconv.Atoi(size); err == nil {
					return &parser.Attribute{
						Name: "db.Char",
						Arguments: []*parser.AttributeArgument{
							{Value: sizeInt},
						},
					}
				}
			}
		}
		if udtName == "timestamptz" {
			if colInfo.DateTimePrecision != nil {
				return &parser.Attribute{
					Name: "db.Timestamptz",
					Arguments: []*parser.AttributeArgument{
						{Value: *colInfo.DateTimePrecision},
					},
				}
			}
			return &parser.Attribute{
				Name:      "db.Timestamptz",
				Arguments: []*parser.AttributeArgument{},
			}
		}
		if udtName == "timestamp" {
			if colInfo.DateTimePrecision != nil {
				return &parser.Attribute{
					Name: "db.Timestamp",
					Arguments: []*parser.AttributeArgument{
						{Value: *colInfo.DateTimePrecision},
					},
				}
			}
			return &parser.Attribute{
				Name:      "db.Timestamp",
				Arguments: []*parser.AttributeArgument{},
			}
		}
	}

	return nil
}

func extractSizeFromDataType(dbType string) string {
	start := strings.Index(dbType, "(")
	end := strings.Index(dbType, ")")
	if start != -1 && end != -1 && end > start {
		return dbType[start+1 : end]
	}
	return ""
}

func extractSize(dbType string) string {
	start := strings.Index(dbType, "(")
	end := strings.Index(dbType, ")")
	if start != -1 && end != -1 && end > start {
		return dbType[start+1 : end]
	}
	return ""
}

func extractPrecision(dbType string) string {
	start := strings.Index(dbType, "(")
	end := strings.Index(dbType, ")")
	if start != -1 && end != -1 && end > start {
		return dbType[start+1 : end]
	}
	return ""
}

func generateRelations(tableName string, tableInfo *TableInfo, dbSchema *DatabaseSchema) []*parser.ModelField {
	relations := []*parser.ModelField{}

	fkMap := make(map[string][]*ForeignKeyInfo)
	for _, fk := range tableInfo.ForeignKeys {
		key := fk.ReferencedTable
		fkMap[key] = append(fkMap[key], fk)
	}

	for refTableName, fks := range fkMap {
		if len(fks) == 1 {
			fk := fks[0]
			fieldName := inferRelationFieldName(fk, refTableName, tableInfo)
			isOptional := false
			for _, colName := range fk.Columns {
				if col, exists := tableInfo.Columns[colName]; exists && col.IsNullable {
					isOptional = true
					break
				}
			}
			relationField := &parser.ModelField{
				Name: fieldName,
				Type: &parser.FieldType{
					Name:       refTableName,
					IsOptional: isOptional,
					IsArray:    false,
				},
				Attributes: []*parser.Attribute{
					{
						Name: "relation",
						Arguments: []*parser.AttributeArgument{
							{Name: "fields", Value: fk.Columns},
							{Name: "references", Value: fk.ReferencedColumns},
							{Name: "onDelete", Value: mapOnDeleteAction(fk.OnDelete)},
						},
					},
				},
			}
			relations = append(relations, relationField)
		} else {
			for _, fk := range fks {
				fieldName := inferRelationFieldNameForMultiple(fk, refTableName, tableName)
				relationName := generateRelationName(tableName, fk, refTableName)
				isOptional := false
				for _, colName := range fk.Columns {
					if col, exists := tableInfo.Columns[colName]; exists && col.IsNullable {
						isOptional = true
						break
					}
				}
				relationField := &parser.ModelField{
					Name: fieldName,
					Type: &parser.FieldType{
						Name:       refTableName,
						IsOptional: isOptional,
						IsArray:    false,
					},
					Attributes: []*parser.Attribute{
						{
							Name: "relation",
							Arguments: []*parser.AttributeArgument{
								{Value: relationName},
								{Name: "fields", Value: fk.Columns},
								{Name: "references", Value: fk.ReferencedColumns},
								{Name: "onDelete", Value: mapOnDeleteAction(fk.OnDelete)},
							},
						},
					},
				}
				relations = append(relations, relationField)
			}
		}
	}

	reverseRelations := generateReverseRelations(tableName, dbSchema)
	relations = append(relations, reverseRelations...)

	return relations
}

func inferRelationFieldName(fk *ForeignKeyInfo, refTableName string, tableInfo *TableInfo) string {
	colName := fk.Columns[0]
	if strings.HasPrefix(colName, "id_") {
		baseName := strings.TrimPrefix(colName, "id_")
		if baseName == refTableName || strings.HasSuffix(baseName, "_"+refTableName) {
			return refTableName
		}
	}
	return refTableName
}

func inferRelationFieldNameForMultiple(fk *ForeignKeyInfo, refTableName, tableName string) string {
	colName := fk.Columns[0]
	return fmt.Sprintf("%s_%s_%sTo%s", refTableName, tableName, colName, refTableName)
}

func generateRelationName(tableName string, fk *ForeignKeyInfo, refTableName string) string {
	colName := fk.Columns[0]
	return fmt.Sprintf("%s_%sTo%s", tableName, colName, refTableName)
}

func generateReverseRelations(tableName string, dbSchema *DatabaseSchema) []*parser.ModelField {
	relations := []*parser.ModelField{}

	fkCountMap := make(map[string]int)
	for _, table := range dbSchema.Tables {
		if table.Name == tableName {
			continue
		}
		for _, fk := range table.ForeignKeys {
			if fk.ReferencedTable == tableName {
				fkCountMap[table.Name]++
			}
		}
	}

	for _, table := range dbSchema.Tables {
		if table.Name == tableName {
			continue
		}
		tableFks := []*ForeignKeyInfo{}
		for _, fk := range table.ForeignKeys {
			if fk.ReferencedTable == tableName {
				tableFks = append(tableFks, fk)
			}
		}
		if len(tableFks) == 0 {
			continue
		}

		if len(tableFks) == 1 {
			fk := tableFks[0]
			fieldName := inferReverseRelationFieldName(table.Name, fk)
			relationField := &parser.ModelField{
				Name: fieldName,
				Type: &parser.FieldType{
					Name:       table.Name,
					IsOptional: false,
					IsArray:    true,
				},
				Attributes: []*parser.Attribute{
					{
						Name:      "relation",
						Arguments: []*parser.AttributeArgument{},
					},
				},
			}
			relations = append(relations, relationField)
		} else {
			for _, fk := range tableFks {
				fieldName := inferReverseRelationFieldNameForMultiple(table.Name, fk, tableName)
				relationName := generateRelationName(table.Name, fk, tableName)
				relationField := &parser.ModelField{
					Name: fieldName,
					Type: &parser.FieldType{
						Name:       table.Name,
						IsOptional: false,
						IsArray:    true,
					},
					Attributes: []*parser.Attribute{
						{
							Name: "relation",
							Arguments: []*parser.AttributeArgument{
								{Value: relationName},
							},
						},
					},
				}
				relations = append(relations, relationField)
			}
		}
	}

	return relations
}

func inferReverseRelationFieldName(refTableName string, fk *ForeignKeyInfo) string {
	return refTableName
}

func inferReverseRelationFieldNameForMultiple(refTableName string, fk *ForeignKeyInfo, targetTableName string) string {
	colName := fk.Columns[0]
	return fmt.Sprintf("%s_%s_%sTo%s", refTableName, refTableName, colName, targetTableName)
}

func mapOnDeleteAction(action string) string {
	action = strings.ToUpper(action)
	switch action {
	case "CASCADE":
		return "Cascade"
	case "SET NULL":
		return "SetNull"
	case "RESTRICT":
		return "Restrict"
	case "NO ACTION":
		return "NoAction"
	default:
		return "Cascade"
	}
}

func generateIndexes(tableName string, tableInfo *TableInfo) []*parser.Attribute {
	indexes := []*parser.Attribute{}

	for _, idx := range tableInfo.Indexes {
		// Build column list with sort order if needed
		columnList := make([]interface{}, 0, len(idx.Columns))
		// Create a map for quick lookup of column info by column name
		colInfoMap := make(map[string]IndexColumnInfo)
		for _, colInfo := range idx.ColumnInfos {
			colInfoMap[colInfo.ColumnName] = colInfo
		}
		// Use ColumnInfos if Columns is empty (fallback)
		if len(idx.Columns) == 0 && len(idx.ColumnInfos) > 0 {
			for _, colInfo := range idx.ColumnInfos {
				idx.Columns = append(idx.Columns, colInfo.ColumnName)
			}
		}
		for _, colName := range idx.Columns {
			if colInfo, exists := colInfoMap[colName]; exists && colInfo.SortOrder == "DESC" {
				columnList = append(columnList, map[string]interface{}{
					"name": colName,
					"sort": "Desc",
				})
			} else {
				columnList = append(columnList, colName)
			}
		}

		if idx.IsUnique {
			if len(idx.Columns) == 1 {
				colName := idx.Columns[0]
				if col, exists := tableInfo.Columns[colName]; exists && col.IsUnique {
					continue
				}
			}
			indexAttr := &parser.Attribute{
				Name: "unique",
				Arguments: []*parser.AttributeArgument{
					{Value: columnList},
				},
			}
			if idx.Name != "" && !strings.HasSuffix(idx.Name, "_pkey") {
				defaultUniqueName := fmt.Sprintf("%s_%s_key", tableName, strings.Join(idx.Columns, "_"))
				if idx.Name != defaultUniqueName {
					indexAttr.Arguments = append(indexAttr.Arguments, &parser.AttributeArgument{
						Name:  "map",
						Value: idx.Name,
					})
				}
			}
			indexes = append(indexes, indexAttr)
		} else {
			// Skip indexes with empty column lists
			if len(columnList) == 0 {
				// If we have ColumnInfos but no Columns, try to use ColumnInfos
				if len(idx.ColumnInfos) > 0 {
					for _, colInfo := range idx.ColumnInfos {
						if colInfo.SortOrder == "DESC" {
							columnList = append(columnList, map[string]interface{}{
								"name": colInfo.ColumnName,
								"sort": "Desc",
							})
						} else {
							columnList = append(columnList, colInfo.ColumnName)
						}
					}
				}
				// If still empty, skip this index
				if len(columnList) == 0 {
					continue
				}
			}
			indexAttr := &parser.Attribute{
				Name: "index",
				Arguments: []*parser.AttributeArgument{
					{Value: columnList},
				},
			}
			if idx.Name != "" {
				// Build default name from Columns or ColumnInfos
				colNames := idx.Columns
				if len(colNames) == 0 && len(idx.ColumnInfos) > 0 {
					colNames = make([]string, len(idx.ColumnInfos))
					for i, colInfo := range idx.ColumnInfos {
						colNames[i] = colInfo.ColumnName
					}
				}
				defaultIndexName := fmt.Sprintf("%s_%s_idx", tableName, strings.Join(colNames, "_"))
				if idx.Name != defaultIndexName {
					indexAttr.Arguments = append(indexAttr.Arguments, &parser.AttributeArgument{
						Name:  "map",
						Value: idx.Name,
					})
				}
			}
			indexes = append(indexes, indexAttr)
		}
	}

	return indexes
}
