package builder

import (
	"context"
	"fmt"
	"strings"
)

// AggregateResult representa o resultado de uma agregação
type AggregateResult struct {
	Count *int64
	Sum   *float64
	Avg   *float64
	Min   *interface{}
	Max   *interface{}
}

// Aggregate executa uma agregação (COUNT, SUM, AVG, MIN, MAX)
func (q *Query) Aggregate(ctx context.Context, field string, aggType string) (interface{}, error) {
	var query string
	var args []interface{}
	argIndex := 1

	// Construir SELECT com agregação
	quotedTable := q.dialect.QuoteIdentifier(q.table)
	aggFunc := strings.ToUpper(aggType)
	switch aggFunc {
	case "COUNT":
		if field == "*" || field == "" {
			query = fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)
		} else {
			quotedField := q.dialect.QuoteIdentifier(field)
			query = fmt.Sprintf("SELECT COUNT(%s) FROM %s", quotedField, quotedTable)
		}
	case "SUM", "AVG", "MIN", "MAX":
		quotedField := q.dialect.QuoteIdentifier(field)
		query = fmt.Sprintf("SELECT %s(%s) FROM %s", aggFunc, quotedField, quotedTable)
	default:
		return nil, fmt.Errorf("tipo de agregação não suportado: %s", aggType)
	}

	// Adicionar JOINs
	for _, join := range q.joins {
		quotedJoinTable := q.dialect.QuoteIdentifier(join.table)
		// join.on já deve estar construído com identificadores escapados
		query += fmt.Sprintf(" %s JOIN %s ON %s", join.joinType, quotedJoinTable, join.on)
		args = append(args, join.args...)
		argIndex += len(join.args)
	}

	// Adicionar WHERE
	if len(q.whereConditions) > 0 {
		whereClause, whereArgs := q.buildWhereClause(&argIndex)
		query += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	// Adicionar GROUP BY
	if len(q.groupBy) > 0 {
		quotedGroupBy := make([]string, len(q.groupBy))
		for i, field := range q.groupBy {
			quotedGroupBy[i] = q.dialect.QuoteIdentifier(field)
		}
		query += " GROUP BY " + strings.Join(quotedGroupBy, ", ")
	}

	// Adicionar HAVING
	if len(q.having) > 0 {
		havingClause, havingArgs := q.buildHavingClause(&argIndex)
		query += " HAVING " + havingClause
		args = append(args, havingArgs...)
	}

	// Executar query
	row := q.db.QueryRow(ctx, query, args...)

	var result interface{}
	err := row.Scan(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Count executa COUNT(*)
func (q *Query) CountAggregate(ctx context.Context) (int64, error) {
	result, err := q.Aggregate(ctx, "*", "COUNT")
	if err != nil {
		return 0, err
	}
	if count, ok := result.(int64); ok {
		return count, nil
	}
	return 0, fmt.Errorf("resultado inesperado do COUNT")
}

// Sum executa SUM(field)
func (q *Query) Sum(ctx context.Context, field string) (float64, error) {
	result, err := q.Aggregate(ctx, field, "SUM")
	if err != nil {
		return 0, err
	}
	if sum, ok := result.(float64); ok {
		return sum, nil
	}
	return 0, fmt.Errorf("resultado inesperado do SUM")
}

// Avg executa AVG(field)
func (q *Query) Avg(ctx context.Context, field string) (float64, error) {
	result, err := q.Aggregate(ctx, field, "AVG")
	if err != nil {
		return 0, err
	}
	if avg, ok := result.(float64); ok {
		return avg, nil
	}
	return 0, fmt.Errorf("resultado inesperado do AVG")
}

// Min executa MIN(field)
func (q *Query) Min(ctx context.Context, field string) (interface{}, error) {
	return q.Aggregate(ctx, field, "MIN")
}

// Max executa MAX(field)
func (q *Query) Max(ctx context.Context, field string) (interface{}, error) {
	return q.Aggregate(ctx, field, "MAX")
}
