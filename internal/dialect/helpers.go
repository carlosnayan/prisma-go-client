package dialect

import "strings"

// isSQLType checks if a type is already a SQL type (from @db.* attributes)
func isSQLType(typ string) bool {
	sqlTypes := []string{
		"TEXT", "VARCHAR", "CHAR", "DATE", "TIME", "TIMESTAMP", "TIMESTAMPTZ",
		"DECIMAL", "NUMERIC", "SMALLINT", "INTEGER", "INT", "BIGINT",
		"REAL", "DOUBLE PRECISION", "DOUBLE", "BOOLEAN", "BOOL",
		"JSON", "JSONB", "BYTEA", "BLOB", "UUID", "INET", "CIDR", "MONEY",
		"BIT", "VARBIT",
	}
	for _, sqlType := range sqlTypes {
		if strings.HasPrefix(typ, sqlType) {
			return true
		}
	}
	return false
}
