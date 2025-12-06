package migrations

import (
	"database/sql"
	"fmt"
	"strings"
)

// DatabaseSchema representa o schema atual do banco de dados
type DatabaseSchema struct {
	Tables map[string]*TableInfo
}

// TableInfo representa informações de uma tabela no banco
type TableInfo struct {
	Name    string
	Columns map[string]*ColumnInfo
	Indexes []*IndexInfo
}

// ColumnInfo representa informações de uma coluna
type ColumnInfo struct {
	Name         string
	Type         string
	IsNullable   bool
	IsPrimaryKey bool
	IsUnique     bool
	DefaultValue *string
}

// IndexInfo representa informações de um índice
type IndexInfo struct {
	Name      string
	TableName string
	Columns   []string
	IsUnique  bool
}

// IntrospectDatabase faz introspection do banco de dados
func IntrospectDatabase(db *sql.DB, provider string) (*DatabaseSchema, error) {
	schema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	switch provider {
	case "postgresql", "postgres":
		return introspectPostgreSQL(db, schema)
	case "mysql":
		return introspectMySQL(db, schema)
	case "sqlite":
		return introspectSQLite(db, schema)
	default:
		return nil, fmt.Errorf("provider não suportado para introspection: %s", provider)
	}
}

// introspectPostgreSQL faz introspection de PostgreSQL
func introspectPostgreSQL(db *sql.DB, schema *DatabaseSchema) (*DatabaseSchema, error) {
	// Obter lista de tabelas (excluindo tabelas do sistema)
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		AND table_name NOT LIKE '_prisma%'
		ORDER BY table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar tabelas: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("erro ao ler nome da tabela: %w", err)
		}
		tableNames = append(tableNames, name)
	}

	// Para cada tabela, obter colunas
	for _, tableName := range tableNames {
		table := &TableInfo{
			Name:    tableName,
			Columns: make(map[string]*ColumnInfo),
			Indexes: []*IndexInfo{},
		}

		// Obter colunas
		colsQuery := `
			SELECT 
				c.column_name,
				c.data_type,
				c.is_nullable,
				c.column_default,
				CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary_key
			FROM information_schema.columns c
			LEFT JOIN (
				SELECT ku.table_name, ku.column_name
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage ku
					ON tc.constraint_name = ku.constraint_name
					AND tc.table_schema = ku.table_schema
				WHERE tc.constraint_type = 'PRIMARY KEY'
				AND tc.table_schema = 'public'
			) pk ON c.table_name = pk.table_name AND c.column_name = pk.column_name
			WHERE c.table_schema = 'public'
			AND c.table_name = $1
			ORDER BY c.ordinal_position
		`

		colsRows, err := db.Query(colsQuery, tableName)
		if err != nil {
			return nil, fmt.Errorf("erro ao obter colunas da tabela %s: %w", tableName, err)
		}

		for colsRows.Next() {
			var colName, dataType, isNullable string
			var columnDefault sql.NullString
			var isPrimaryKey bool

			if err := colsRows.Scan(&colName, &dataType, &isNullable, &columnDefault, &isPrimaryKey); err != nil {
				colsRows.Close()
				return nil, fmt.Errorf("erro ao ler coluna: %w", err)
			}

			col := &ColumnInfo{
				Name:         colName,
				Type:         mapPostgreSQLType(dataType),
				IsNullable:   isNullable == "YES",
				IsPrimaryKey: isPrimaryKey,
				IsUnique:     false, // Será preenchido depois
			}

			if columnDefault.Valid {
				col.DefaultValue = &columnDefault.String
			}

			table.Columns[colName] = col
		}
		colsRows.Close()

		// Obter índices únicos
		idxQuery := `
			SELECT
				i.indexname,
				a.attname,
				ix.indisunique
			FROM pg_indexes i
			JOIN pg_index ix ON i.indexname = (SELECT relname FROM pg_class WHERE oid = ix.indexrelid)
			JOIN pg_attribute a ON a.attrelid = ix.indrelid AND a.attnum = ANY(ix.indkey)
			WHERE i.schemaname = 'public'
			AND i.tablename = $1
			AND i.indexname NOT LIKE '%_pkey'
		`

		idxRows, err := db.Query(idxQuery, tableName)
		if err == nil {
			indexMap := make(map[string]*IndexInfo)
			for idxRows.Next() {
				var idxName, colName string
				var isUnique bool
				if err := idxRows.Scan(&idxName, &colName, &isUnique); err == nil {
					if idx, exists := indexMap[idxName]; exists {
						idx.Columns = append(idx.Columns, colName)
					} else {
						indexMap[idxName] = &IndexInfo{
							Name:      idxName,
							TableName: tableName,
							Columns:   []string{colName},
							IsUnique:  isUnique,
						}
					}
				}
			}
			idxRows.Close()

			for _, idx := range indexMap {
				table.Indexes = append(table.Indexes, idx)
				// Marcar colunas como unique se o índice for único
				if idx.IsUnique && len(idx.Columns) == 1 {
					if col, exists := table.Columns[idx.Columns[0]]; exists {
						col.IsUnique = true
					}
				}
			}
		}

		schema.Tables[tableName] = table
	}

	return schema, nil
}

// introspectMySQL faz introspection de MySQL
func introspectMySQL(db *sql.DB, schema *DatabaseSchema) (*DatabaseSchema, error) {
	// TODO: Implementar introspection para MySQL
	return schema, fmt.Errorf("introspection MySQL ainda não implementada")
}

// introspectSQLite faz introspection de SQLite
func introspectSQLite(db *sql.DB, schema *DatabaseSchema) (*DatabaseSchema, error) {
	// TODO: Implementar introspection para SQLite
	return schema, fmt.Errorf("introspection SQLite ainda não implementada")
}

// mapPostgreSQLType mapeia tipo PostgreSQL para tipo Prisma
func mapPostgreSQLType(pgType string) string {
	pgType = strings.ToLower(pgType)
	
	switch {
	case strings.HasPrefix(pgType, "varchar"), pgType == "text", pgType == "char":
		return "String"
	case pgType == "integer" || pgType == "int" || pgType == "int4":
		return "Int"
	case pgType == "bigint" || pgType == "int8":
		return "BigInt"
	case pgType == "boolean" || pgType == "bool":
		return "Boolean"
	case pgType == "timestamp" || pgType == "timestamptz" || pgType == "date" || pgType == "time":
		return "DateTime"
	case pgType == "real" || pgType == "double precision" || pgType == "float8":
		return "Float"
	case strings.HasPrefix(pgType, "numeric") || strings.HasPrefix(pgType, "decimal"):
		return "Decimal"
	case pgType == "jsonb" || pgType == "json":
		return "Json"
	case pgType == "bytea":
		return "Bytes"
	default:
		return "String" // Default
	}
}

