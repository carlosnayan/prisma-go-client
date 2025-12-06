package builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
)

// Search creates a full-text search operator
// Example: q.Search("content", "golang prisma")
func (q *Query) Search(field string, query string) *Query {
	searchQuery := fmt.Sprintf("%s @@ to_tsquery($1)", field)
	q.whereConditions = append(q.whereConditions, whereCondition{
		query: searchQuery,
		args:  []interface{}{NormalizeTSQuery(query)},
		or:    false,
	})
	return q
}

// SearchInsensitive creates a case-insensitive full-text search
func (q *Query) SearchInsensitive(field string, query string) *Query {
	searchQuery := fmt.Sprintf("to_tsvector('simple', lower(%s)) @@ to_tsquery('simple', lower($1))", field)
	q.whereConditions = append(q.whereConditions, whereCondition{
		query: searchQuery,
		args:  []interface{}{NormalizeTSQuery(query)},
		or:    false,
	})
	return q
}

// Rank adds ordering by full-text search relevance
// Example: q.Search("content", "golang").Rank("content", "golang", "DESC")
func (q *Query) Rank(field string, query string, order string) *Query {
	if order == "" {
		order = "DESC"
	}

	rankExpr := fmt.Sprintf("ts_rank(to_tsvector(%s), to_tsquery($1))", field)

	q.orderBy = append(q.orderBy, OrderBy{
		Field: rankExpr,
		Order: strings.ToUpper(order),
	})

	hasSearch := false
	for _, cond := range q.whereConditions {
		if strings.Contains(cond.query, "to_tsquery") {
			hasSearch = true
			break
		}
	}

	if !hasSearch {
		q.Search(field, query)
	}

	return q
}

// NormalizeTSQuery normalizes a query for PostgreSQL to_tsquery
// Converts spaces to & (AND) and adds :* for prefix matching
func NormalizeTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	words := strings.Fields(query)

	wordsWithPrefix := make([]string, len(words))
	for i, word := range words {
		wordsWithPrefix[i] = word + ":*"
	}

	return strings.Join(wordsWithPrefix, " & ")
}

// SearchOp creates a full-text search operator for use in Where
// Example: q.Where(builder.Where{"content": builder.SearchOp("golang prisma")})
func SearchOp(query string) WhereOperator {
	return WhereOperator{
		op:    "FULLTEXT_SEARCH",
		value: query,
	}
}

// SearchOpWithConfig creates a full-text search operator with specific configuration
func SearchOpWithConfig(query string, config string) WhereOperator {
	return WhereOperator{
		op: "FULLTEXT_SEARCH_CONFIG",
		value: map[string]interface{}{
			"query":  query,
			"config": config,
		},
	}
}

// BuildFullTextIndex creates a full-text index (helper for migrations)
// Returns SQL to create a GIN index for full-text search
func BuildFullTextIndex(tableName string, fieldName string, indexName string) string {
	d := dialect.GetDialect("postgresql")
	quotedTable := d.QuoteIdentifier(tableName)
	quotedField := d.QuoteIdentifier(fieldName)

	if indexName == "" {
		indexName = fmt.Sprintf("idx_%s_%s_fulltext", tableName, fieldName)
	}
	quotedIndex := d.QuoteIdentifier(indexName)
	return fmt.Sprintf("CREATE INDEX %s ON %s USING GIN (to_tsvector('english', %s))",
		quotedIndex, quotedTable, quotedField)
}

// ExecuteFullTextSearch executes a full-text search and returns results with rank
func (q *Query) ExecuteFullTextSearch(ctx context.Context, field string, searchQuery string, dest interface{}) error {
	q.Search(field, searchQuery)

	if len(q.selectFields) == 0 {
		q.selectFields = append(q.selectFields, fmt.Sprintf("ts_rank(to_tsvector(%s), to_tsquery($1)) as rank", field))
	} else {
		q.selectFields = append(q.selectFields, fmt.Sprintf("ts_rank(to_tsvector(%s), to_tsquery($1)) as rank", field))
	}

	q.Rank(field, searchQuery, "DESC")

	return q.Find(ctx, dest)
}
