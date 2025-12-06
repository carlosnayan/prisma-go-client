package dialect

import (
	"fmt"
	"strings"
)

// SQLiteDialect implements the SQLite dialect
type SQLiteDialect struct{}

func (d *SQLiteDialect) Name() string {
	return "sqlite"
}

func (d *SQLiteDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

func (d *SQLiteDialect) QuoteString(value string) string {
	// Escapar aspas simples
	escaped := strings.ReplaceAll(value, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

func (d *SQLiteDialect) MapType(prismaType string, isNullable bool) string {
	// SQLite tem tipos dinâmicos, mas mapeamos para tipos recomendados
	switch strings.ToLower(prismaType) {
	case "string":
		return "TEXT"
	case "int":
		return "INTEGER"
	case "bigint":
		return "INTEGER" // SQLite não diferencia
	case "boolean", "bool":
		return "INTEGER" // SQLite usa 0/1
	case "datetime":
		return "TEXT" // SQLite armazena como ISO8601 string
	case "float":
		return "REAL"
	case "decimal":
		return "NUMERIC"
	case "json":
		return "TEXT" // SQLite armazena JSON como TEXT
	case "bytes":
		return "BLOB"
	default:
		return "TEXT"
	}
}

func (d *SQLiteDialect) MapDefaultValue(value string) string {
	value = strings.ToLower(value)
	switch {
	case value == "autoincrement()" || value == "autoincrement":
		return "" // Será tratado como AUTOINCREMENT
	case value == "now()" || value == "now":
		return "datetime('now')"
	case strings.HasPrefix(value, "uuid()") || strings.HasPrefix(value, "uuid"):
		return "(lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6))))"
	case strings.HasPrefix(value, "cuid()") || strings.HasPrefix(value, "cuid"):
		return "(lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6))))"
	default:
		return value
	}
}

func (d *SQLiteDialect) GetPlaceholder(index int) string {
	return "?"
}

func (d *SQLiteDialect) GetAutoIncrementKeyword() string {
	return "AUTOINCREMENT"
}

func (d *SQLiteDialect) GetNowFunction() string {
	return "datetime('now')"
}

func (d *SQLiteDialect) GetDriverName() string {
	return "sqlite3"
}

func (d *SQLiteDialect) SupportsFullTextSearch() bool {
	return false // SQLite requer FTS extension
}

func (d *SQLiteDialect) GetFullTextSearchQuery(field string, query string) string {
	// SQLite não suporta full-text search nativamente sem FTS extension
	// Fallback para LIKE
	return fmt.Sprintf("%s LIKE %s", d.QuoteIdentifier(field), d.QuoteString("%"+query+"%"))
}

func (d *SQLiteDialect) SupportsJSON() bool {
	return true // SQLite 3.9+ tem suporte JSON
}

func (d *SQLiteDialect) GetJSONContainsQuery(field string, value string) string {
	return fmt.Sprintf("json_extract(%s, '$') = %s", d.QuoteIdentifier(field), d.QuoteString(value))
}

func (d *SQLiteDialect) GetLimitOffsetSyntax(limit, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	} else if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	} else if offset > 0 {
		return fmt.Sprintf("OFFSET %d", offset)
	}
	return ""
}
