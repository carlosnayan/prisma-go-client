package builder

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	contextutil "github.com/carlosnayan/prisma-go-client/internal/context"
	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	"github.com/carlosnayan/prisma-go-client/internal/driver"
	"github.com/carlosnayan/prisma-go-client/internal/errors"
	"github.com/carlosnayan/prisma-go-client/internal/limits"
	"github.com/carlosnayan/prisma-go-client/internal/logger"
)

// Query represents a query builder with fluent (chainable) API
type Query struct {
	db             driver.DB
	table          string
	columns        []string
	primaryKey     string
	hasDeleted     bool
	modelType      reflect.Type
	includeDeleted bool            // Flag to include deleted records
	logger         *logger.Logger  // Logger for queries
	dialect        dialect.Dialect // Database dialect

	// Query state
	whereConditions []whereCondition
	orderBy         []OrderBy
	limit           *int
	offset          *int
	selectFields    []string
	groupBy         []string
	having          []whereCondition
	joins           []join
}

// whereCondition represents a WHERE condition
type whereCondition struct {
	query string
	args  []interface{}
	or    bool // if true, use OR instead of AND
}

// join represents a JOIN
type join struct {
	joinType string // "INNER", "LEFT", "RIGHT", "FULL"
	table    string
	on       string
	args     []interface{}
}

// NewQuery creates a new query builder with fluent API
func NewQuery(db DBTX, table string, columns []string) *Query {
	return &Query{
		db:              db,
		table:           table,
		columns:         columns,
		dialect:         dialect.GetDialect("postgresql"), // Default
		logger:          logger.GetDefaultLogger(),        // Use default logger
		whereConditions: []whereCondition{},
		orderBy:         []OrderBy{},
		joins:           []join{},
		selectFields:    []string{},
		groupBy:         []string{},
		having:          []whereCondition{},
	}
}

// SetDialect sets the database dialect
func (q *Query) SetDialect(d dialect.Dialect) *Query {
	q.dialect = d
	return q
}

// SetDialectFromProvider sets the dialect from provider name
func (q *Query) SetDialectFromProvider(provider string) *Query {
	q.dialect = dialect.GetDialect(provider)
	return q
}

// SetPrimaryKey sets the primary key
func (q *Query) SetPrimaryKey(pk string) *Query {
	q.primaryKey = pk
	return q
}

// SetHasDeleted indicates if the table has deleted_at
func (q *Query) SetHasDeleted(has bool) *Query {
	q.hasDeleted = has
	return q
}

// SetModelType sets the model type for automatic scanning
func (q *Query) SetModelType(modelType reflect.Type) *Query {
	q.modelType = modelType
	return q
}

// getLogger returns the logger, always getting the current default logger
// This ensures that if the logger is configured after Query creation, it will use the updated logger
func (q *Query) getLogger() *logger.Logger {
	// Always get the current default logger to ensure it's up to date
	currentLogger := logger.GetDefaultLogger()
	// Update q.logger if it's different (for efficiency, but always use current)
	if currentLogger != q.logger {
		q.logger = currentLogger
	}
	return q.logger
}

// Where adds a WHERE condition
// Supports two syntaxes:
//  1. Direct SQL: q.Where("name = ?", "jinzhu")
//  2. Prisma map: q.Where(builder.Where{"name": "jinzhu", "age": builder.Gt(18)})
func (q *Query) Where(condition interface{}, args ...interface{}) *Query {
	if queryStr, ok := condition.(string); ok {
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: queryStr,
			args:  args,
			or:    false,
		})
		return q
	}

	if whereMap, ok := condition.(Where); ok {
		for field, value := range whereMap {
			if op, ok := value.(WhereOperator); ok {
				q.addPrismaWhereCondition(field, op)
			} else if value == nil {
				quotedField := q.dialect.QuoteIdentifier(field)
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: fmt.Sprintf("%s IS NULL", quotedField),
					args:  []interface{}{},
					or:    false,
				})
			} else {
				quotedField := q.dialect.QuoteIdentifier(field)
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: fmt.Sprintf("%s = ?", quotedField),
					args:  []interface{}{value},
					or:    false,
				})
			}
		}
		return q
	}

	return q
}

// addPrismaWhereCondition adds a WHERE condition using Prisma operator
func (q *Query) addPrismaWhereCondition(field string, op WhereOperator) {
	quotedField := q.dialect.QuoteIdentifier(field)
	switch op.GetOp() {
	case ">":
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s > ?", quotedField),
			args:  []interface{}{op.GetValue()},
			or:    false,
		})
	case ">=":
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s >= ?", quotedField),
			args:  []interface{}{op.GetValue()},
			or:    false,
		})
	case "<":
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s < ?", quotedField),
			args:  []interface{}{op.GetValue()},
			or:    false,
		})
	case "<=":
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s <= ?", quotedField),
			args:  []interface{}{op.GetValue()},
			or:    false,
		})
	case "IN":
		if values, ok := op.GetValue().([]interface{}); ok {
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = "?"
			}
			q.whereConditions = append(q.whereConditions, whereCondition{
				query: fmt.Sprintf("%s IN (%s)", quotedField, strings.Join(placeholders, ", ")),
				args:  values,
				or:    false,
			})
		}
	case "NOT IN":
		if values, ok := op.GetValue().([]interface{}); ok {
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = "?"
			}
			q.whereConditions = append(q.whereConditions, whereCondition{
				query: fmt.Sprintf("%s NOT IN (%s)", quotedField, strings.Join(placeholders, ", ")),
				args:  values,
				or:    false,
			})
		}
	case "LIKE":
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s LIKE ?", quotedField),
			args:  []interface{}{op.GetValue()},
			or:    false,
		})
	case "ILIKE":
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s ILIKE ?", quotedField),
			args:  []interface{}{op.GetValue()},
			or:    false,
		})
	case "IS NULL":
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s IS NULL", quotedField),
			args:  []interface{}{},
			or:    false,
		})
	case "IS NOT NULL":
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s IS NOT NULL", quotedField),
			args:  []interface{}{},
			or:    false,
		})
	case "HAS":
		if q.dialect.SupportsJSON() {
			jsonValue := fmt.Sprintf(`["%v"]`, op.GetValue())
			query := q.dialect.GetJSONContainsQuery(field, jsonValue)
			q.whereConditions = append(q.whereConditions, whereCondition{
				query: query,
				args:  []interface{}{},
				or:    false,
			})
		} else {
			q.whereConditions = append(q.whereConditions, whereCondition{
				query: fmt.Sprintf("%s LIKE ?", q.dialect.QuoteIdentifier(field)),
				args:  []interface{}{fmt.Sprintf("%%%v%%", op.GetValue())},
				or:    false,
			})
		}
	case "HAS_EVERY":
		if q.dialect.SupportsJSON() {
			if values, ok := op.GetValue().([]interface{}); ok {
				jsonValue := fmt.Sprintf(`[%s]`, strings.Join(func() []string {
					result := make([]string, len(values))
					for i, v := range values {
						result[i] = fmt.Sprintf(`"%v"`, v)
					}
					return result
				}(), ", "))
				query := q.dialect.GetJSONContainsQuery(field, jsonValue)
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: query,
					args:  []interface{}{},
					or:    false,
				})
			}
		} else {
			if values, ok := op.GetValue().([]interface{}); ok {
				conditions := make([]string, len(values))
				for i := range values {
					conditions[i] = fmt.Sprintf("%s LIKE ?", q.dialect.QuoteIdentifier(field))
				}
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: fmt.Sprintf("(%s)", strings.Join(conditions, " AND ")),
					args: func() []interface{} {
						result := make([]interface{}, len(values))
						for i, v := range values {
							result[i] = fmt.Sprintf("%%%v%%", v)
						}
						return result
					}(),
					or: false,
				})
			}
		}
	case "HAS_SOME":
		if q.dialect.SupportsJSON() {
			if values, ok := op.GetValue().([]interface{}); ok {
				if q.dialect.Name() == "postgresql" {
					placeholders := make([]string, len(values))
					for i := range values {
						placeholders[i] = "?"
					}
					quotedField := q.dialect.QuoteIdentifier(field)
					q.whereConditions = append(q.whereConditions, whereCondition{
						query: fmt.Sprintf("%s ?| array[%s]", quotedField, strings.Join(placeholders, ", ")),
						args:  values,
						or:    false,
					})
				} else {
					conditions := make([]string, len(values))
					allArgs := make([]interface{}, 0)
					for i, v := range values {
						jsonValue := fmt.Sprintf(`"%v"`, v)
						conditions[i] = q.dialect.GetJSONContainsQuery(field, jsonValue)
					}
					q.whereConditions = append(q.whereConditions, whereCondition{
						query: fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")),
						args:  allArgs,
						or:    false,
					})
				}
			}
		} else {
			if values, ok := op.GetValue().([]interface{}); ok {
				conditions := make([]string, len(values))
				allArgs := make([]interface{}, len(values))
				for i, v := range values {
					conditions[i] = fmt.Sprintf("%s LIKE ?", q.dialect.QuoteIdentifier(field))
					allArgs[i] = fmt.Sprintf("%%%v%%", v)
				}
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")),
					args:  allArgs,
					or:    false,
				})
			}
		}
	case "IS_EMPTY":
		if q.dialect.SupportsJSON() {
			quotedField := q.dialect.QuoteIdentifier(field)
			if q.dialect.Name() == "postgresql" {
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: fmt.Sprintf("(jsonb_typeof(%s) = 'array' AND jsonb_array_length(%s) = 0) OR %s = '[]'::jsonb", quotedField, quotedField, quotedField),
					args:  []interface{}{},
					or:    false,
				})
			} else if q.dialect.Name() == "mysql" {
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: fmt.Sprintf("(JSON_TYPE(%s) = 'ARRAY' AND JSON_LENGTH(%s) = 0) OR %s = '[]'", quotedField, quotedField, quotedField),
					args:  []interface{}{},
					or:    false,
				})
			} else {
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: fmt.Sprintf("(json_array_length(%s) = 0 OR %s IS NULL)", quotedField, quotedField),
					args:  []interface{}{},
					or:    false,
				})
			}
		} else {
			quotedField := q.dialect.QuoteIdentifier(field)
			q.whereConditions = append(q.whereConditions, whereCondition{
				query: fmt.Sprintf("(%s IS NULL OR %s = '')", quotedField, quotedField),
				args:  []interface{}{},
				or:    false,
			})
		}
	case "FULLTEXT_SEARCH":
		if q.dialect.SupportsFullTextSearch() {
			if queryStr, ok := op.GetValue().(string); ok {
				if q.dialect.Name() == "postgresql" {
					queryStr = NormalizeTSQuery(queryStr)
				}
				query := q.dialect.GetFullTextSearchQuery(field, queryStr)
				args := []interface{}{}
				if strings.Contains(query, "?") || strings.Contains(query, "$") {
					args = []interface{}{queryStr}
				}
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: query,
					args:  args,
					or:    false,
				})
			}
		} else {
			if queryStr, ok := op.GetValue().(string); ok {
				quotedField := q.dialect.QuoteIdentifier(field)
				q.whereConditions = append(q.whereConditions, whereCondition{
					query: fmt.Sprintf("%s LIKE ?", quotedField),
					args:  []interface{}{fmt.Sprintf("%%%s%%", queryStr)},
					or:    false,
				})
			}
		}
	case "FULLTEXT_SEARCH_CONFIG":
		if q.dialect.SupportsFullTextSearch() && q.dialect.Name() == "postgresql" {
			if configMap, ok := op.GetValue().(map[string]interface{}); ok {
				if queryStr, ok := configMap["query"].(string); ok {
					queryStr = NormalizeTSQuery(queryStr)
					config := "english"
					if c, ok := configMap["config"].(string); ok {
						config = c
					}
					quotedField := q.dialect.QuoteIdentifier(field)
					q.whereConditions = append(q.whereConditions, whereCondition{
						query: fmt.Sprintf("to_tsvector('%s', %s) @@ to_tsquery('%s', $1)", config, quotedField, config),
						args:  []interface{}{queryStr},
						or:    false,
					})
				}
			}
		} else {
			if configMap, ok := op.GetValue().(map[string]interface{}); ok {
				if queryStr, ok := configMap["query"].(string); ok {
					quotedField := q.dialect.QuoteIdentifier(field)
					q.whereConditions = append(q.whereConditions, whereCondition{
						query: fmt.Sprintf("%s LIKE ?", quotedField),
						args:  []interface{}{fmt.Sprintf("%%%s%%", queryStr)},
						or:    false,
					})
				}
			}
		}
	default:
		quotedField := q.dialect.QuoteIdentifier(field)
		q.whereConditions = append(q.whereConditions, whereCondition{
			query: fmt.Sprintf("%s = ?", quotedField),
			args:  []interface{}{op.GetValue()},
			or:    false,
		})
	}
}

// Or adds an OR condition
func (q *Query) Or(query string, args ...interface{}) *Query {
	q.whereConditions = append(q.whereConditions, whereCondition{
		query: query,
		args:  args,
		or:    true,
	})
	return q
}

// Not adds a NOT condition
func (q *Query) Not(query string, args ...interface{}) *Query {
	q.whereConditions = append(q.whereConditions, whereCondition{
		query: fmt.Sprintf("NOT (%s)", query),
		args:  args,
		or:    false,
	})
	return q
}

// Select specifies which columns to select
// Example: q.Select("id", "name", "email")
func (q *Query) Select(fields ...string) *Query {
	remaining := limits.MaxSelectFields - len(q.selectFields)
	if remaining <= 0 {
		return q
	}
	if len(fields) > remaining {
		fields = fields[:remaining]
	}
	q.selectFields = append(q.selectFields, fields...)
	return q
}

// SelectAll clears Select and returns all fields
func (q *Query) SelectAll() *Query {
	q.selectFields = []string{}
	return q
}

// Order adds ORDER BY
func (q *Query) Order(order string) *Query {
	if len(q.orderBy) >= limits.MaxOrderByFields {
		return q
	}

	parts := strings.Fields(order)
	if len(parts) == 2 {
		q.orderBy = append(q.orderBy, OrderBy{
			Field: parts[0],
			Order: strings.ToUpper(parts[1]),
		})
	} else if len(parts) == 1 {
		q.orderBy = append(q.orderBy, OrderBy{
			Field: parts[0],
			Order: "ASC",
		})
	}
	return q
}

// Limit sets the LIMIT
func (q *Query) Limit(limit int) *Query {
	q.limit = &limit
	return q
}

// Offset sets the OFFSET
func (q *Query) Offset(offset int) *Query {
	q.offset = &offset
	return q
}

// Group adds GROUP BY
func (q *Query) Group(fields ...string) *Query {
	remaining := limits.MaxGroupByFields - len(q.groupBy)
	if remaining <= 0 {
		return q
	}
	if len(fields) > remaining {
		fields = fields[:remaining]
	}
	q.groupBy = append(q.groupBy, fields...)
	return q
}

// Having adds HAVING
func (q *Query) Having(query string, args ...interface{}) *Query {
	q.having = append(q.having, whereCondition{
		query: query,
		args:  args,
		or:    false,
	})
	return q
}

// Join adds a JOIN
func (q *Query) Join(joinType, table, on string, args ...interface{}) *Query {
	if len(q.joins) >= limits.MaxJoins {
		return q
	}
	q.joins = append(q.joins, join{
		joinType: joinType,
		table:    table,
		on:       on,
		args:     args,
	})
	return q
}

// InnerJoin adds an INNER JOIN
func (q *Query) InnerJoin(table, on string, args ...interface{}) *Query {
	return q.Join("INNER", table, on, args...)
}

// LeftJoin adds a LEFT JOIN
func (q *Query) LeftJoin(table, on string, args ...interface{}) *Query {
	return q.Join("LEFT", table, on, args...)
}

// RightJoin adds a RIGHT JOIN
func (q *Query) RightJoin(table, on string, args ...interface{}) *Query {
	return q.Join("RIGHT", table, on, args...)
}

// First executes the query and returns the first result
// Example: q.Where("email = ?", "user@example.com").First(ctx, &user)
func (q *Query) First(ctx context.Context, dest interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	processStart := time.Now()
	query, args := q.buildSelectQuery(true)
	
	// Medir query time: desde o início da chamada ao banco até o retorno do Scan
	// Para QueryRow, a execução real acontece durante o Scan()
	queryStart := time.Now()
	row := q.db.QueryRow(ctx, query, args...)

	var err error
	if q.modelType != nil {
		err = q.scanRowIntoModel(row, dest)
	} else {
		err = row.Scan(dest)
	}
	queryEnd := time.Now()
	queryDuration := queryEnd.Sub(queryStart)

	// Log com tempos separados (query time e process time)
	q.logQueryWithTiming(ctx, query, args, queryStart, processStart, queryDuration)

	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("SELECT query failed: %v", err)
		}
	}

	return err
}

// Find executes the query and returns all results
// Example: q.Where("active = ?", true).Order("created_at DESC").Find(ctx, &users)
func (q *Query) Find(ctx context.Context, dest interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	processStart := time.Now()
	query, args := q.buildSelectQuery(false)
	
	// Medir query time: desde o início da chamada ao banco até o retorno
	queryStart := time.Now()
	rows, err := q.db.Query(ctx, query, args...)
	queryEnd := time.Now()
	queryDuration := queryEnd.Sub(queryStart)
	
	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("SELECT query failed: %v", err)
		}
		return err
	}
	defer rows.Close()

	if q.modelType != nil {
		err = q.scanRowsIntoModel(rows, dest)
	} else {
		err = q.scanRowsDirect(rows, dest)
	}

	// Log com tempos separados (query time e process time)
	q.logQueryWithTiming(ctx, query, args, queryStart, processStart, queryDuration)

	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("SELECT query failed during scan: %v", err)
		}
	}

	return err
}

// FindFirst is an alias for First (compatibility)
func (q *Query) FindFirst(ctx context.Context, dest interface{}) error {
	return q.First(ctx, dest)
}

// FindMany is an alias for Find (compatibility)
func (q *Query) FindMany(ctx context.Context, dest interface{}) error {
	return q.Find(ctx, dest)
}

// Count executes COUNT(*)
func (q *Query) Count(ctx context.Context) (int64, error) {
	processStart := time.Now()
	query, args := q.buildCountQuery()
	
	// Medir query time: desde o início da chamada ao banco até o retorno do Scan
	// Para QueryRow, a execução real acontece durante o Scan()
	queryStart := time.Now()
	row := q.db.QueryRow(ctx, query, args...)
	var count int64
	err := row.Scan(&count)
	queryEnd := time.Now()
	queryDuration := queryEnd.Sub(queryStart)
	
	// Log com tempos separados (query time e process time)
	q.logQueryWithTiming(ctx, query, args, queryStart, processStart, queryDuration)
	
	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("COUNT query failed: %v", err)
		}
	}
	return count, err
}

// Create inserts a new record
func (q *Query) Create(ctx context.Context, value interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	processStart := time.Now()
	query, args := q.buildInsertQuery(value)
	
	// Medir query time: desde o início da chamada ao banco até o retorno
	queryStart := time.Now()
	_, err := q.db.Exec(ctx, query, args...)
	queryEnd := time.Now()
	queryDuration := queryEnd.Sub(queryStart)
	
	// Log com tempos separados (query time e process time)
	q.logQueryWithTiming(ctx, query, args, queryStart, processStart, queryDuration)
	
	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("INSERT query failed: %v", err)
		}
	}
	return errors.SanitizeError(err)
}

// Save updates or creates a record (upsert)
func (q *Query) Save(ctx context.Context, value interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	if q.primaryKey == "" {
		// Se não há primary key, apenas criar
		return q.Create(ctx, value)
	}

	processStart := time.Now()
	query, args := q.buildUpsertQuery(value)
	
	// Medir query time: desde o início da chamada ao banco até o retorno
	queryStart := time.Now()
	_, err := q.db.Exec(ctx, query, args...)
	queryEnd := time.Now()
	queryDuration := queryEnd.Sub(queryStart)
	
	// Log com tempos separados (query time e process time)
	q.logQueryWithTiming(ctx, query, args, queryStart, processStart, queryDuration)
	
	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("UPSERT query failed: %v", err)
		}
	}
	return errors.SanitizeError(err)
}

// Update updates records
func (q *Query) Update(ctx context.Context, column string, value interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	processStart := time.Now()
	query, args := q.buildUpdateQuery(column, value)
	
	// Medir query time: desde o início da chamada ao banco até o retorno
	queryStart := time.Now()
	_, err := q.db.Exec(ctx, query, args...)
	queryEnd := time.Now()
	queryDuration := queryEnd.Sub(queryStart)
	
	// Log com tempos separados (query time e process time)
	q.logQueryWithTiming(ctx, query, args, queryStart, processStart, queryDuration)
	
	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("UPDATE query failed: %v", err)
		}
	}
	return errors.SanitizeError(err)
}

// Updates updates multiple columns
func (q *Query) Updates(ctx context.Context, values map[string]interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	processStart := time.Now()
	query, args := q.buildUpdatesQuery(values)
	
	// Medir query time: desde o início da chamada ao banco até o retorno
	queryStart := time.Now()
	_, err := q.db.Exec(ctx, query, args...)
	queryEnd := time.Now()
	queryDuration := queryEnd.Sub(queryStart)
	
	// Log com tempos separados (query time e process time)
	q.logQueryWithTiming(ctx, query, args, queryStart, processStart, queryDuration)
	
	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("UPDATE query failed: %v", err)
		}
	}
	return errors.SanitizeError(err)
}

// Delete removes records
func (q *Query) Delete(ctx context.Context, value interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	processStart := time.Now()
	query, args := q.buildDeleteQuery()
	
	// Medir query time: desde o início da chamada ao banco até o retorno
	queryStart := time.Now()
	_, err := q.db.Exec(ctx, query, args...)
	queryEnd := time.Now()
	queryDuration := queryEnd.Sub(queryStart)
	
	// Log com tempos separados (query time e process time)
	q.logQueryWithTiming(ctx, query, args, queryStart, processStart, queryDuration)
	
	if err != nil {
		if logger := q.getLogger(); logger != nil {
			logger.Error("DELETE query failed: %v", err)
		}
	}
	return errors.SanitizeError(err)
}

// buildSelectQuery builds the SELECT query
func (q *Query) buildSelectQuery(single bool) (string, []interface{}) {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT
	parts = append(parts, "SELECT")
	if len(q.selectFields) > 0 {
		quotedFields := make([]string, len(q.selectFields))
		for i, field := range q.selectFields {
			quotedFields[i] = q.dialect.QuoteIdentifier(field)
		}
		parts = append(parts, strings.Join(quotedFields, ", "))
	} else {
		quotedColumns := make([]string, len(q.columns))
		for i, col := range q.columns {
			quotedColumns[i] = q.dialect.QuoteIdentifier(col)
		}
		parts = append(parts, strings.Join(quotedColumns, ", "))
	}

	// FROM
	parts = append(parts, "FROM", q.dialect.QuoteIdentifier(q.table))

	// JOINs
	for _, join := range q.joins {
		parts = append(parts, fmt.Sprintf("%s JOIN %s ON %s", join.joinType, q.dialect.QuoteIdentifier(join.table), join.on))
		args = append(args, join.args...)
		argIndex += len(join.args)
	}

	// WHERE
	if len(q.whereConditions) > 0 {
		whereClause, whereArgs := q.buildWhereClause(&argIndex)
		parts = append(parts, "WHERE", whereClause)
		args = append(args, whereArgs...)

		// Se tem deleted_at e não queremos incluir deletados, adicionar condição
		if q.hasDeleted && !q.includeDeleted {
			deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
			parts = append(parts, fmt.Sprintf("AND %s IS NULL", deletedAtField))
		}
	} else if q.hasDeleted && !q.includeDeleted {
		// Se tem deleted_at mas não tem WHERE, adicionar condição para não mostrar deletados
		deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
		parts = append(parts, fmt.Sprintf("WHERE %s IS NULL", deletedAtField))
	}

	// GROUP BY
	if len(q.groupBy) > 0 {
		quotedGroupBy := make([]string, len(q.groupBy))
		for i, field := range q.groupBy {
			quotedGroupBy[i] = q.dialect.QuoteIdentifier(field)
		}
		parts = append(parts, "GROUP BY", strings.Join(quotedGroupBy, ", "))
	}

	// HAVING
	if len(q.having) > 0 {
		havingClause, havingArgs := q.buildHavingClause(&argIndex)
		parts = append(parts, "HAVING", havingClause)
		args = append(args, havingArgs...)
	}

	// ORDER BY
	if len(q.orderBy) > 0 {
		var orderParts []string
		for _, order := range q.orderBy {
			orderParts = append(orderParts, fmt.Sprintf("%s %s", q.dialect.QuoteIdentifier(order.Field), order.Order))
		}
		parts = append(parts, "ORDER BY", strings.Join(orderParts, ", "))
	}

	if single {
		parts = append(parts, "LIMIT 1")
	} else if q.limit != nil || q.offset != nil {
		limit := 0
		offset := 0
		if q.limit != nil {
			limit = *q.limit
		}
		if q.offset != nil {
			offset = *q.offset
		}
		limitOffset := q.dialect.GetLimitOffsetSyntax(limit, offset)
		parts = append(parts, limitOffset)
		if q.limit != nil {
			args = append(args, *q.limit)
		}
		if q.offset != nil {
			args = append(args, *q.offset)
		}
	}

	return strings.Join(parts, " "), args
}

// buildWhereClause builds the WHERE clause
func (q *Query) buildWhereClause(argIndex *int) (string, []interface{}) {
	if len(q.whereConditions) == 0 {
		return "", nil
	}

	var parts []string
	var args []interface{}

	for i, cond := range q.whereConditions {
		if i > 0 {
			if cond.or {
				parts = append(parts, "OR")
			} else {
				parts = append(parts, "AND")
			}
		}

		query := cond.query
		var queryBuilder strings.Builder
		queryBuilder.Grow(len(query) + 100)

		argPos := 0
		for i := 0; i < len(query); i++ {
			if query[i] == '?' && argPos < len(cond.args) {
				arg := cond.args[argPos]
				if reflect.TypeOf(arg).Kind() == reflect.Slice {
					slice := reflect.ValueOf(arg)
					placeholders := make([]string, slice.Len())
					for j := 0; j < slice.Len(); j++ {
						placeholders[j] = q.dialect.GetPlaceholder(*argIndex)
						args = append(args, slice.Index(j).Interface())
						(*argIndex)++
					}
					queryBuilder.WriteString(fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")))
				} else {
					queryBuilder.WriteString(q.dialect.GetPlaceholder(*argIndex))
					args = append(args, arg)
					(*argIndex)++
				}
				argPos++
			} else {
				queryBuilder.WriteByte(query[i])
			}
		}
		query = queryBuilder.String()

		parts = append(parts, query)
	}

	return strings.Join(parts, " "), args
}

// buildHavingClause builds the HAVING clause (similar to WHERE)
func (q *Query) buildHavingClause(argIndex *int) (string, []interface{}) {
	return q.buildWhereClause(argIndex)
}

// buildCountQuery builds the COUNT query
func (q *Query) buildCountQuery() (string, []interface{}) {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, "SELECT COUNT(*) FROM", q.dialect.QuoteIdentifier(q.table))

	// JOINs
	for _, join := range q.joins {
		parts = append(parts, fmt.Sprintf("%s JOIN %s ON %s", join.joinType, q.dialect.QuoteIdentifier(join.table), join.on))
		args = append(args, join.args...)
		argIndex += len(join.args)
	}

	// WHERE
	if len(q.whereConditions) > 0 {
		whereClause, whereArgs := q.buildWhereClause(&argIndex)
		parts = append(parts, "WHERE", whereClause)
		args = append(args, whereArgs...)

		if q.hasDeleted && !q.includeDeleted {
			deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
			parts = append(parts, fmt.Sprintf("AND %s IS NULL", deletedAtField))
		}
	} else if q.hasDeleted && !q.includeDeleted {
		deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
		parts = append(parts, fmt.Sprintf("WHERE %s IS NULL", deletedAtField))
	}

	return strings.Join(parts, " "), args
}

// buildInsertQuery builds the INSERT query
func (q *Query) buildInsertQuery(value interface{}) (string, []interface{}) {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return "", nil
	}

	var columns []string
	var values []string
	var args []interface{}
	argIndex := 1

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if fieldVal.IsZero() {
			continue
		}

		fieldName := toSnakeCase(field.Name)
		if fieldName == q.primaryKey || fieldName == "created_at" || fieldName == "updated_at" {
			continue
		}

		columns = append(columns, fieldName)
		values = append(values, q.dialect.GetPlaceholder(argIndex))
		args = append(args, fieldVal.Interface())
		argIndex++
	}

	hasCreatedAt := contains(q.columns, "created_at")
	hasUpdatedAt := contains(q.columns, "updated_at")

	if hasCreatedAt {
		columns = append(columns, "created_at")
		values = append(values, q.dialect.GetNowFunction())
	}
	if hasUpdatedAt {
		columns = append(columns, "updated_at")
		values = append(values, q.dialect.GetNowFunction())
	}

	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = q.dialect.QuoteIdentifier(col)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		q.dialect.QuoteIdentifier(q.table),
		strings.Join(quotedColumns, ", "),
		strings.Join(values, ", "),
	)

	return query, args
}

// buildUpsertQuery builds an INSERT ... ON CONFLICT (upsert) query
func (q *Query) buildUpsertQuery(value interface{}) (string, []interface{}) {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return "", nil
	}

	var columns []string
	var values []string
	var args []interface{}
	argIndex := 1

	typ := val.Type()
	var primaryKeyValue interface{}
	var primaryKeyCol string

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		fieldName := toSnakeCase(field.Name)

		// Capturar primary key se existir
		if fieldName == q.primaryKey {
			primaryKeyCol = fieldName
			primaryKeyValue = fieldVal.Interface()
			continue
		}

		if fieldVal.IsZero() {
			continue
		}

		if fieldName == "created_at" || fieldName == "updated_at" {
			continue
		}

		columns = append(columns, fieldName)
		values = append(values, q.dialect.GetPlaceholder(argIndex))
		args = append(args, fieldVal.Interface())
		argIndex++
	}

	hasCreatedAt := contains(q.columns, "created_at")
	hasUpdatedAt := contains(q.columns, "updated_at")

	// Se há primary key, adicionar à lista de colunas
	if primaryKeyCol != "" && primaryKeyValue != nil {
		columns = append(columns, primaryKeyCol)
		values = append(values, q.dialect.GetPlaceholder(argIndex))
		args = append(args, primaryKeyValue)
		argIndex++
	}

	if hasCreatedAt {
		columns = append(columns, "created_at")
		values = append(values, q.dialect.GetNowFunction())
	}
	if hasUpdatedAt {
		columns = append(columns, "updated_at")
		values = append(values, q.dialect.GetNowFunction())
	}

	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = q.dialect.QuoteIdentifier(col)
	}

	quotedTable := q.dialect.QuoteIdentifier(q.table)
	insertPart := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quotedTable,
		strings.Join(quotedColumns, ", "),
		strings.Join(values, ", "),
	)

	// Construir parte de conflito baseado no dialect
	dialectName := q.dialect.Name()
	var conflictPart string

	if dialectName == "postgresql" || dialectName == "postgres" || dialectName == "sqlite" {
		// PostgreSQL e SQLite usam ON CONFLICT
		if primaryKeyCol != "" {
			quotedPK := q.dialect.QuoteIdentifier(primaryKeyCol)
			var updateParts []string
			for _, col := range columns {
				if col == primaryKeyCol || col == "created_at" {
					continue
				}
				quotedCol := q.dialect.QuoteIdentifier(col)
				if col == "updated_at" {
					updateParts = append(updateParts, fmt.Sprintf("%s = %s", quotedCol, q.dialect.GetNowFunction()))
				} else {
					updateParts = append(updateParts, fmt.Sprintf("%s = EXCLUDED.%s", quotedCol, quotedCol))
				}
			}
			conflictPart = fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET %s", quotedPK, strings.Join(updateParts, ", "))
		} else {
			// Sem primary key, apenas INSERT
			return insertPart, args
		}
	} else if dialectName == "mysql" || dialectName == "mariadb" {
		// MySQL usa ON DUPLICATE KEY UPDATE
		if primaryKeyCol != "" {
			var updateParts []string
			for _, col := range columns {
				if col == primaryKeyCol || col == "created_at" {
					continue
				}
				quotedCol := q.dialect.QuoteIdentifier(col)
				if col == "updated_at" {
					updateParts = append(updateParts, fmt.Sprintf("%s = %s", quotedCol, q.dialect.GetNowFunction()))
				} else {
					updateParts = append(updateParts, fmt.Sprintf("%s = VALUES(%s)", quotedCol, quotedCol))
				}
			}
			conflictPart = fmt.Sprintf("ON DUPLICATE KEY UPDATE %s", strings.Join(updateParts, ", "))
		} else {
			// Sem primary key, apenas INSERT
			return insertPart, args
		}
	} else {
		// Fallback: apenas INSERT
		return insertPart, args
	}

	query := fmt.Sprintf("%s %s", insertPart, conflictPart)
	return query, args
}

// buildUpdateQuery builds the UPDATE query
func (q *Query) buildUpdateQuery(column string, value interface{}) (string, []interface{}) {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("UPDATE %s SET %s = %s",
		q.dialect.QuoteIdentifier(q.table),
		q.dialect.QuoteIdentifier(column),
		q.dialect.GetPlaceholder(argIndex)))
	args = append(args, value)
	argIndex++

	hasUpdatedAt := contains(q.columns, "updated_at")
	if hasUpdatedAt {
		parts = append(parts, fmt.Sprintf(", %s = %s",
			q.dialect.QuoteIdentifier("updated_at"),
			q.dialect.GetNowFunction()))
	}

	// WHERE
	if len(q.whereConditions) > 0 {
		whereClause, whereArgs := q.buildWhereClause(&argIndex)
		parts = append(parts, "WHERE", whereClause)
		args = append(args, whereArgs...)

		if q.hasDeleted && !q.includeDeleted {
			deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
			parts = append(parts, fmt.Sprintf("AND %s IS NULL", deletedAtField))
		}
	} else if q.hasDeleted && !q.includeDeleted {
		deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
		parts = append(parts, fmt.Sprintf("WHERE %s IS NULL", deletedAtField))
	}

	return strings.Join(parts, " "), args
}

// buildUpdatesQuery builds the UPDATE query with multiple columns
func (q *Query) buildUpdatesQuery(values map[string]interface{}) (string, []interface{}) {
	var parts []string
	var args []interface{}
	argIndex := 1

	var setParts []string
	for col, val := range values {
		if col == "updated_at" {
			setParts = append(setParts, fmt.Sprintf("%s = %s",
				q.dialect.QuoteIdentifier("updated_at"),
				q.dialect.GetNowFunction()))
		} else {
			setParts = append(setParts, fmt.Sprintf("%s = %s",
				q.dialect.QuoteIdentifier(col),
				q.dialect.GetPlaceholder(argIndex)))
			args = append(args, val)
			argIndex++
		}
	}

	parts = append(parts, fmt.Sprintf("UPDATE %s SET %s",
		q.dialect.QuoteIdentifier(q.table),
		strings.Join(setParts, ", ")))

	// WHERE
	if len(q.whereConditions) > 0 {
		whereClause, whereArgs := q.buildWhereClause(&argIndex)
		parts = append(parts, "WHERE", whereClause)
		args = append(args, whereArgs...)

		if q.hasDeleted && !q.includeDeleted {
			deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
			parts = append(parts, fmt.Sprintf("AND %s IS NULL", deletedAtField))
		}
	} else if q.hasDeleted && !q.includeDeleted {
		deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
		parts = append(parts, fmt.Sprintf("WHERE %s IS NULL", deletedAtField))
	}

	return strings.Join(parts, " "), args
}

// buildDeleteQuery builds the DELETE query
func (q *Query) buildDeleteQuery() (string, []interface{}) {
	var parts []string
	var args []interface{}
	argIndex := 1

	if q.hasDeleted {
		parts = append(parts, fmt.Sprintf("UPDATE %s SET %s = %s",
			q.dialect.QuoteIdentifier(q.table),
			q.dialect.QuoteIdentifier("deleted_at"),
			q.dialect.GetNowFunction()))
	} else {
		parts = append(parts, fmt.Sprintf("DELETE FROM %s", q.dialect.QuoteIdentifier(q.table)))
	}

	// WHERE
	if len(q.whereConditions) > 0 {
		whereClause, whereArgs := q.buildWhereClause(&argIndex)
		parts = append(parts, "WHERE", whereClause)
		args = append(args, whereArgs...)
	} else if q.hasDeleted {
		parts = append(parts, fmt.Sprintf("WHERE %s IS NULL", q.dialect.QuoteIdentifier("deleted_at")))
	}

	return strings.Join(parts, " "), args
}

// scanRowIntoModel scans a row into a model
func (q *Query) scanRowIntoModel(row interface{}, dest interface{}) error {
	if driverRow, ok := row.(driver.Row); ok {
		destVal := reflect.ValueOf(dest)
		if destVal.Kind() != reflect.Ptr {
			return errors.SanitizeError(fmt.Errorf("dest must be a pointer"))
		}
		destVal = destVal.Elem()

		if q.modelType == nil {
			return errors.SanitizeError(fmt.Errorf("modelType not defined"))
		}

		modelValue := reflect.New(q.modelType).Elem()

		// Use selectFields if available (when Select() was called), otherwise use all columns
		columnsToScan := q.columns
		if len(q.selectFields) > 0 {
			columnsToScan = q.selectFields
		}

		fields := make([]interface{}, len(columnsToScan))
		for i, colName := range columnsToScan {
			field := findFieldByColumn(modelValue, colName)
			if field.IsValid() {
				fields[i] = field.Addr().Interface()
			} else {
				var dummy interface{}
				fields[i] = &dummy
			}
		}

		if err := driverRow.Scan(fields...); err != nil {
			if logger := q.getLogger(); logger != nil {
				logger.Error("Scan failed: %v (scanning %d fields: %v)", err, len(columnsToScan), columnsToScan)
			}
			return err
		}

		destVal.Set(modelValue)
		return nil
	}
	return errors.SanitizeError(fmt.Errorf("unsupported row type"))
}

// scanRowsIntoModel scans rows into a slice of models
func (q *Query) scanRowsIntoModel(rows interface{}, dest interface{}) error {
	if driverRows, ok := rows.(driver.Rows); ok {
		defer driverRows.Close()

		destVal := reflect.ValueOf(dest)
		if destVal.Kind() != reflect.Ptr {
			return errors.SanitizeError(fmt.Errorf("dest must be a pointer to slice"))
		}

		sliceVal := destVal.Elem()
		if sliceVal.Kind() != reflect.Slice {
			return errors.SanitizeError(fmt.Errorf("dest must be a pointer to slice"))
		}

		if q.modelType == nil {
			return errors.SanitizeError(fmt.Errorf("modelType not defined"))
		}

		sliceType := sliceVal.Type().Elem()
		if sliceType.Kind() == reflect.Ptr {
			sliceType = sliceType.Elem()
		}

		rowCount := 0

		for driverRows.Next() {
			if rowCount >= limits.MaxScanRows {
				return fmt.Errorf("result set too large: maximum %d rows allowed", limits.MaxScanRows)
			}

			modelValue := reflect.New(sliceType).Elem()

			// Use selectFields if available (when Select() was called), otherwise use all columns
			columnsToScan := q.columns
			if len(q.selectFields) > 0 {
				columnsToScan = q.selectFields
			}

			fields := make([]interface{}, len(columnsToScan))
			for i, colName := range columnsToScan {
				field := findFieldByColumn(modelValue, colName)
				if field.IsValid() {
					fields[i] = field.Addr().Interface()
				} else {
					var dummy interface{}
					fields[i] = &dummy
				}
			}

			if err := driverRows.Scan(fields...); err != nil {
				if logger := q.getLogger(); logger != nil {
					logger.Error("Scan failed: %v (scanning %d fields: %v)", err, len(columnsToScan), columnsToScan)
				}
				return err
			}

			rowCount++
			if destVal.Elem().Type().Elem().Kind() == reflect.Ptr {
				sliceVal.Set(reflect.Append(sliceVal, modelValue.Addr()))
			} else {
				sliceVal.Set(reflect.Append(sliceVal, modelValue))
			}
		}

		return driverRows.Err()
	}
	return errors.SanitizeError(fmt.Errorf("unsupported rows type"))
}

// scanRowsDirect performs direct scan (for simple cases)
func (q *Query) scanRowsDirect(rows interface{}, dest interface{}) error {
	return q.scanRowsIntoModel(rows, dest)
}

// findFieldByColumn finds a struct field by column name
func findFieldByColumn(modelValue reflect.Value, colName string) reflect.Value {
	typ := modelValue.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")

		// Verificar tags
		if dbTag == colName || jsonTag == colName {
			return modelValue.Field(i)
		}

		// Verificar nome do campo (snake_case)
		fieldName := toSnakeCase(field.Name)
		if fieldName == colName {
			return modelValue.Field(i)
		}
	}
	return reflect.Value{}
}
