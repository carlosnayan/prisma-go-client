package dialect

import (
	"fmt"
	"strings"
)

// PostgreSQLDialect implements the PostgreSQL dialect
type PostgreSQLDialect struct{}

func (d *PostgreSQLDialect) Name() string {
	return "postgresql"
}

func (d *PostgreSQLDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

func (d *PostgreSQLDialect) QuoteString(value string) string {
	// Escapar aspas simples
	escaped := strings.ReplaceAll(value, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

func (d *PostgreSQLDialect) MapType(prismaType string, isNullable bool) string {
	prismaTypeLower := strings.ToLower(prismaType)

	// Lista de tipos Prisma conhecidos - verificar primeiro para evitar confusão com tipos SQL
	prismaTypes := []string{"string", "int", "bigint", "boolean", "bool", "datetime", "float", "decimal", "json", "bytes", "uuid"}
	isPrismaType := false
	for _, pt := range prismaTypes {
		if prismaTypeLower == pt {
			isPrismaType = true
			break
		}
	}

	// Se não é um tipo Prisma conhecido, pode ser um tipo SQL (vem de @db.*)
	if !isPrismaType {
		prismaTypeUpper := strings.ToUpper(prismaType)
		if isSQLType(prismaTypeUpper) {
			return prismaType
		}
	}

	switch prismaTypeLower {
	case "string":
		return "TEXT"
	case "int":
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "boolean", "bool":
		return "BOOLEAN"
	case "datetime":
		return "TIMESTAMP"
	case "float":
		return "DOUBLE PRECISION"
	case "decimal":
		return "DECIMAL(65, 30)"
	case "json":
		return "JSONB"
	case "bytes":
		return "BYTEA"
	case "uuid":
		return "UUID"
	default:
		return "TEXT"
	}
}

func (d *PostgreSQLDialect) MapDefaultValue(value string) string {
	value = strings.ToLower(value)
	switch {
	case value == "autoincrement()" || value == "autoincrement":
		return "" // Será tratado como SERIAL
	case value == "now()" || value == "now":
		return "NOW()"
	case strings.HasPrefix(value, "uuid()") || strings.HasPrefix(value, "uuid"):
		return "gen_random_uuid()"
	case strings.HasPrefix(value, "cuid()") || strings.HasPrefix(value, "cuid"):
		return "gen_random_uuid()" // Fallback para UUID
	default:
		return value
	}
}

func (d *PostgreSQLDialect) GetPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (d *PostgreSQLDialect) GetAutoIncrementKeyword() string {
	return "SERIAL" // ou BIGSERIAL para bigint
}

func (d *PostgreSQLDialect) GetNowFunction() string {
	return "NOW()"
}

func (d *PostgreSQLDialect) GetDriverName() string {
	return "pgx"
}

func (d *PostgreSQLDialect) SupportsFullTextSearch() bool {
	return true
}

func (d *PostgreSQLDialect) GetFullTextSearchQuery(field string, query string) string {
	return fmt.Sprintf("%s @@ to_tsquery(%s)", d.QuoteIdentifier(field), d.QuoteString(query))
}

func (d *PostgreSQLDialect) SupportsJSON() bool {
	return true
}

func (d *PostgreSQLDialect) GetJSONContainsQuery(field string, value string) string {
	return fmt.Sprintf("%s @> %s::jsonb", d.QuoteIdentifier(field), d.QuoteString(value))
}

func (d *PostgreSQLDialect) GetLimitOffsetSyntax(limit, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	} else if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	} else if offset > 0 {
		return fmt.Sprintf("OFFSET %d", offset)
	}
	return ""
}
