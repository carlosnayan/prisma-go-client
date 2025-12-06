package dialect

import (
	"strings"
)

// Dialect representa um dialeto de banco de dados
// Abstrai as diferenças entre PostgreSQL, MySQL, SQLite, etc.
type Dialect interface {
	// Name retorna o nome do dialeto (ex: "postgresql", "mysql", "sqlite")
	Name() string

	// QuoteIdentifier cita um identificador (tabela, coluna, etc.)
	// PostgreSQL: "table_name", MySQL: `table_name`, SQLite: "table_name"
	QuoteIdentifier(name string) string

	// QuoteString cita uma string literal
	// PostgreSQL: 'value', MySQL: 'value', SQLite: 'value'
	QuoteString(value string) string

	// MapType mapeia um tipo Prisma para tipo SQL do banco
	// Exemplo: "String" -> "VARCHAR(255)" (MySQL) ou "TEXT" (PostgreSQL)
	MapType(prismaType string, isNullable bool) string

	// MapDefaultValue mapeia um valor default do Prisma para SQL
	// Exemplo: "autoincrement()" -> "AUTO_INCREMENT" (MySQL) ou "SERIAL" (PostgreSQL)
	MapDefaultValue(value string) string

	// GetPlaceholder retorna o placeholder para parâmetros
	// PostgreSQL: $1, $2, MySQL: ?, ?, SQLite: ?, ?
	GetPlaceholder(index int) string

	// GetAutoIncrementKeyword retorna a palavra-chave para auto incremento
	// PostgreSQL: SERIAL/BIGSERIAL, MySQL: AUTO_INCREMENT, SQLite: AUTOINCREMENT
	GetAutoIncrementKeyword() string

	// GetNowFunction retorna a função para obter data/hora atual
	// PostgreSQL: NOW(), MySQL: NOW(), SQLite: datetime('now')
	GetNowFunction() string

	// GetDriverName retorna o nome do driver Go para database/sql
	// PostgreSQL: "pgx", MySQL: "mysql", SQLite: "sqlite3"
	GetDriverName() string

	// SupportsFullTextSearch indica se o banco suporta busca full-text
	SupportsFullTextSearch() bool

	// GetFullTextSearchQuery retorna a query de busca full-text
	// PostgreSQL: field @@ to_tsquery('query')
	// MySQL: MATCH(field) AGAINST('query' IN BOOLEAN MODE)
	GetFullTextSearchQuery(field string, query string) string

	// SupportsJSON indica se o banco suporta campos JSON
	SupportsJSON() bool

	// GetJSONContainsQuery retorna query para verificar se JSON contém valor
	// PostgreSQL: field @> 'value'::jsonb
	// MySQL: JSON_CONTAINS(field, 'value')
	GetJSONContainsQuery(field string, value string) string

	// GetLimitOffsetSyntax retorna a sintaxe LIMIT/OFFSET
	// PostgreSQL: LIMIT n OFFSET m, MySQL: LIMIT m, n (ou LIMIT n OFFSET m)
	GetLimitOffsetSyntax(limit, offset int) string
}

// GetDialect retorna o dialeto apropriado para o provider
func GetDialect(provider string) Dialect {
	provider = strings.ToLower(provider)
	
	switch provider {
	case "postgresql", "postgres":
		return &PostgreSQLDialect{}
	case "mysql", "mariadb":
		return &MySQLDialect{}
	case "sqlite":
		return &SQLiteDialect{}
	default:
		// Default para PostgreSQL
		return &PostgreSQLDialect{}
	}
}

