package migrations

import (
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// SchemaDiff representa diferenças entre schema e banco
type SchemaDiff struct {
	TablesToCreate  []TableDefinition
	TablesToAlter   []TableAlteration
	TablesToDrop    []string
	IndexesToCreate []IndexDefinition
	IndexesToDrop   []string
}

// TableDefinition representa uma tabela a ser criada
type TableDefinition struct {
	Name    string
	Columns []ColumnDefinition
}

// ColumnDefinition representa uma coluna
type ColumnDefinition struct {
	Name         string
	Type         string
	IsNullable   bool
	IsPrimaryKey bool
	IsUnique     bool
	DefaultValue string
}

// TableAlteration representa alterações em uma tabela
type TableAlteration struct {
	TableName    string
	AddColumns   []ColumnDefinition
	DropColumns  []string
	AlterColumns []ColumnAlteration
}

// ColumnAlteration representa alteração em uma coluna
type ColumnAlteration struct {
	ColumnName  string
	NewType     string
	NewNullable bool
}

// IndexDefinition representa um índice
type IndexDefinition struct {
	Name      string
	TableName string
	Columns   []string
	IsUnique  bool
}

// needsUUIDExtension verifica se a migration precisa da extensão pgcrypto para gen_random_uuid()
func needsUUIDExtension(diff *SchemaDiff) bool {
	// Verificar em tabelas a serem criadas
	for _, table := range diff.TablesToCreate {
		for _, col := range table.Columns {
			if strings.Contains(strings.ToLower(col.DefaultValue), "gen_random_uuid") {
				return true
			}
		}
	}

	// Verificar em colunas a serem adicionadas
	for _, alter := range diff.TablesToAlter {
		for _, col := range alter.AddColumns {
			if strings.Contains(strings.ToLower(col.DefaultValue), "gen_random_uuid") {
				return true
			}
		}
	}

	return false
}

// GenerateMigrationSQL gera SQL de migration baseado nas diferenças
func GenerateMigrationSQL(diff *SchemaDiff, provider string) (string, error) {
	var sql strings.Builder
	d := dialect.GetDialect(provider)

	// Se for PostgreSQL e precisar de gen_random_uuid(), criar extensão
	if provider == "postgresql" && needsUUIDExtension(diff) {
		sql.WriteString("-- Enable pgcrypto extension for gen_random_uuid()\n")
		sql.WriteString("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";\n\n")
	}

	// Criar tabelas
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

			if col.IsUnique && !col.IsPrimaryKey {
				colDef += " UNIQUE"
			}

			columns = append(columns, colDef)
		}

		sql.WriteString(strings.Join(columns, ",\n"))

		if len(primaryKeys) > 0 {
			quotedPKs := make([]string, len(primaryKeys))
			for i, pk := range primaryKeys {
				quotedPKs[i] = d.QuoteIdentifier(pk)
			}
			sql.WriteString(fmt.Sprintf(",\n  PRIMARY KEY (%s)", strings.Join(quotedPKs, ", ")))
		}

		sql.WriteString("\n);\n\n")
	}

	// Alterar tabelas
	for _, alter := range diff.TablesToAlter {
		// Adicionar colunas
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

		// Remover colunas
		for _, colName := range alter.DropColumns {
			sql.WriteString(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;\n",
				d.QuoteIdentifier(alter.TableName),
				d.QuoteIdentifier(colName)))
		}

		sql.WriteString("\n")
	}

	// Criar índices
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

	// Remover índices
	for _, idxName := range diff.IndexesToDrop {
		sql.WriteString(fmt.Sprintf("DROP INDEX %s;\n", quoteIdentifier(idxName, provider)))
	}

	return sql.String(), nil
}

// SchemaToSQL converte um schema Prisma para SQL (cria tudo do zero)
// Use CompareSchema para detectar mudanças incrementais
func SchemaToSQL(schema *parser.Schema, provider string) (*SchemaDiff, error) {
	diff := &SchemaDiff{}

	for _, model := range schema.Models {
		table := TableDefinition{
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
						// Extrair valor padrão
						col.DefaultValue = extractDefaultValue(attr.Arguments[0])
					}
				}
			}

			table.Columns = append(table.Columns, col)
		}

		diff.TablesToCreate = append(diff.TablesToCreate, table)
	}

	return diff, nil
}

// extractDefaultValue extrai valor padrão de um argumento
func extractDefaultValue(arg *parser.AttributeArgument) string {
	if str, ok := arg.Value.(string); ok {
		return fmt.Sprintf("'%s'", strings.ReplaceAll(str, "'", "''"))
	}

	// Se for função (autoincrement, now, dbgenerated, etc.)
	if m, ok := arg.Value.(map[string]interface{}); ok {
		if fn, ok := m["function"].(string); ok {
			switch fn {
			case "autoincrement":
				return "" // Será tratado como SERIAL/BIGSERIAL
			case "now":
				return "CURRENT_TIMESTAMP"
			case "dbgenerated":
				// dbgenerated("gen_random_uuid()") -> extrair o argumento
				if args, ok := m["args"].([]interface{}); ok && len(args) > 0 {
					if sqlStr, ok := args[0].(string); ok {
						// Retornar o SQL diretamente, sem aspas
						return sqlStr
					}
				}
				return ""
			case "uuid":
				return "gen_random_uuid()"
			}
		}
	}

	return ""
}

// mapTypeToSQL mapeia tipo Prisma para SQL
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
		// Se começar com VARCHAR, retornar como está (já vem do @db.VarChar)
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

// quoteIdentifier coloca aspas em identificadores SQL
func quoteIdentifier(name string, provider string) string {
	switch provider {
	case "postgresql":
		return fmt.Sprintf(`"%s"`, name)
	case "mysql":
		return fmt.Sprintf("`%s`", name)
	case "sqlite":
		return fmt.Sprintf(`"%s"`, name)
	default:
		return fmt.Sprintf(`"%s"`, name)
	}
}
