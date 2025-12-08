package dialect

import (
	"fmt"
	"strings"
)

// MySQLDialect implements the MySQL dialect
type MySQLDialect struct{}

func (d *MySQLDialect) Name() string {
	return "mysql"
}

func (d *MySQLDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", name)
}

func (d *MySQLDialect) QuoteString(value string) string {
	// Escapar aspas simples e barras
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

func (d *MySQLDialect) MapType(prismaType string, isNullable bool) string {
	prismaTypeUpper := strings.ToUpper(prismaType)
	
	// Se já é um tipo SQL (vem de @db.*), adaptar para MySQL se necessário
	if isSQLType(prismaTypeUpper) {
		// Adaptar tipos específicos do PostgreSQL para MySQL
		if strings.HasPrefix(prismaTypeUpper, "BYTEA") {
			return "BLOB"
		}
		if strings.HasPrefix(prismaTypeUpper, "JSONB") {
			return "JSON"
		}
		if strings.HasPrefix(prismaTypeUpper, "TIMESTAMPTZ") {
			return "TIMESTAMP"
		}
		if strings.HasPrefix(prismaTypeUpper, "DOUBLE PRECISION") {
			return "DOUBLE"
		}
		if strings.HasPrefix(prismaTypeUpper, "BOOLEAN") || strings.HasPrefix(prismaTypeUpper, "BOOL") {
			return "TINYINT(1)"
		}
		// Tipos PostgreSQL não suportados em MySQL
		if strings.HasPrefix(prismaTypeUpper, "INET") || strings.HasPrefix(prismaTypeUpper, "CIDR") ||
			strings.HasPrefix(prismaTypeUpper, "MONEY") || strings.HasPrefix(prismaTypeUpper, "BIT") ||
			strings.HasPrefix(prismaTypeUpper, "VARBIT") {
			return "VARCHAR(255)" // Fallback
		}
		return prismaType
	}
	
	switch strings.ToLower(prismaType) {
	case "string":
		return "VARCHAR(191)" // MySQL tem limite menor por padrão
	case "int":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "boolean", "bool":
		return "TINYINT(1)"
	case "datetime":
		return "DATETIME"
	case "float":
		return "DOUBLE"
	case "decimal":
		return "DECIMAL(65, 30)"
	case "json":
		return "JSON"
	case "bytes":
		return "BLOB"
	default:
		return "VARCHAR(191)"
	}
}

func (d *MySQLDialect) MapDefaultValue(value string) string {
	value = strings.ToLower(value)
	switch {
	case value == "autoincrement()" || value == "autoincrement":
		return "" // Será tratado como AUTO_INCREMENT
	case value == "now()" || value == "now":
		return "CURRENT_TIMESTAMP"
	case strings.HasPrefix(value, "uuid()") || strings.HasPrefix(value, "uuid"):
		return "UUID()"
	case strings.HasPrefix(value, "cuid()") || strings.HasPrefix(value, "cuid"):
		return "UUID()" // Fallback para UUID
	default:
		return value
	}
}

func (d *MySQLDialect) GetPlaceholder(index int) string {
	return "?"
}

func (d *MySQLDialect) GetAutoIncrementKeyword() string {
	return "AUTO_INCREMENT"
}

func (d *MySQLDialect) GetNowFunction() string {
	return "NOW()"
}

func (d *MySQLDialect) GetDriverName() string {
	return "mysql"
}

func (d *MySQLDialect) SupportsFullTextSearch() bool {
	return true
}

func (d *MySQLDialect) GetFullTextSearchQuery(field string, query string) string {
	return fmt.Sprintf("MATCH(%s) AGAINST(%s IN BOOLEAN MODE)", d.QuoteIdentifier(field), d.QuoteString(query))
}

func (d *MySQLDialect) SupportsJSON() bool {
	return true
}

func (d *MySQLDialect) GetJSONContainsQuery(field string, value string) string {
	return fmt.Sprintf("JSON_CONTAINS(%s, %s)", d.QuoteIdentifier(field), d.QuoteString(value))
}

func (d *MySQLDialect) GetLimitOffsetSyntax(limit, offset int) string {
	if limit > 0 && offset > 0 {
		// MySQL suporta LIMIT offset, limit
		return fmt.Sprintf("LIMIT %d, %d", offset, limit)
	} else if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	} else if offset > 0 {
		// MySQL não suporta OFFSET sem LIMIT, usar LIMIT grande
		return fmt.Sprintf("LIMIT 18446744073709551615 OFFSET %d", offset)
	}
	return ""
}
