package builder

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	contextutil "github.com/carlosnayan/prisma-go-client/internal/context"
	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	"github.com/carlosnayan/prisma-go-client/internal/driver"
	"github.com/carlosnayan/prisma-go-client/internal/errors"
	"github.com/carlosnayan/prisma-go-client/internal/limits"
)

// DBTX is an alias for driver.DB for backward compatibility
type DBTX = driver.DB

// Result is an alias for driver.Result for use in generated code
type Result = driver.Result

// Rows is an alias for driver.Rows for use in generated code
type Rows = driver.Rows

// Row is an alias for driver.Row for use in generated code
type Row = driver.Row

// Tx is an alias for driver.Tx for use in generated code
type Tx = driver.Tx

// TableQueryBuilder provides a Prisma-like query builder for database tables
type TableQueryBuilder struct {
	db         DBTX
	table      string
	columns    []string
	primaryKey string
	hasDeleted bool
	modelType  reflect.Type
	dialect    dialect.Dialect
}

// NewTableQueryBuilder creates a new query builder for a table
func NewTableQueryBuilder(db DBTX, table string, columns []string) *TableQueryBuilder {
	return &TableQueryBuilder{
		db:      db,
		table:   table,
		columns: columns,
		dialect: dialect.GetDialect("postgresql"), // Default
	}
}

// SetDialect sets the database dialect
func (b *TableQueryBuilder) SetDialect(d dialect.Dialect) *TableQueryBuilder {
	b.dialect = d
	return b
}

// SetPrimaryKey defines the primary key column name
func (b *TableQueryBuilder) SetPrimaryKey(pk string) *TableQueryBuilder {
	b.primaryKey = pk
	return b
}

// SetHasDeleted indicates if the table has a deleted_at column for soft deletes
func (b *TableQueryBuilder) SetHasDeleted(has bool) *TableQueryBuilder {
	b.hasDeleted = has
	return b
}

// SetModelType defines the model type for automatic scanning
func (b *TableQueryBuilder) SetModelType(modelType reflect.Type) *TableQueryBuilder {
	b.modelType = modelType
	return b
}

// FindFirst finds the first record matching the where conditions
func (b *TableQueryBuilder) FindFirst(ctx context.Context, where Where) (interface{}, error) {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	query, args := b.buildQuery(where, nil, true)
	row := b.db.QueryRow(ctx, query, args...)

	if b.modelType == nil {
		return row, nil
	}

	return b.scanRow(row)
}

// FindMany finds multiple records matching the query options
func (b *TableQueryBuilder) FindMany(ctx context.Context, opts QueryOptions) (interface{}, error) {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	query, args := b.buildQuery(opts.Where, &opts, false)
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if b.modelType == nil {
		return rows, nil
	}

	return b.scanRows(rows)
}

// Count counts records matching the where conditions
func (b *TableQueryBuilder) Count(ctx context.Context, where Where) (int, error) {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	var parts []string
	var args []interface{}
	argIndex := 1

	quotedTable := b.dialect.QuoteIdentifier(b.table)
	parts = append(parts, fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable))

	if len(where) > 0 {
		whereClause, whereArgs := b.buildWhereFromMap(where, &argIndex)
		if whereClause != "" {
			parts = append(parts, "WHERE "+whereClause)
			args = append(args, whereArgs...)
		}
	}

	query := strings.Join(parts, " ")
	var count int
	err := b.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, errors.SanitizeError(err)
	}
	return count, nil
}

// Create inserts a new record and returns the created model
func (b *TableQueryBuilder) Create(ctx context.Context, data interface{}) (interface{}, error) {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data must be a struct")
	}

	var insertColumns []string
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
		if fieldName == b.primaryKey || fieldName == "created_at" || fieldName == "updated_at" {
			continue
		}

		insertColumns = append(insertColumns, fieldName)
		values = append(values, fmt.Sprintf("$%d", argIndex))
		args = append(args, fieldVal.Interface())
		argIndex++
	}

	hasCreatedAt := contains(b.columns, "created_at")
	hasUpdatedAt := contains(b.columns, "updated_at")

	returningColumns := make([]string, len(b.columns))
	copy(returningColumns, b.columns)

	if hasCreatedAt {
		insertColumns = append(insertColumns, "created_at")
		values = append(values, "NOW()")
	}
	if hasUpdatedAt {
		insertColumns = append(insertColumns, "updated_at")
		values = append(values, "NOW()")
	}

	quotedTable := b.dialect.QuoteIdentifier(b.table)
	quotedInsertCols := make([]string, len(insertColumns))
	for i, col := range insertColumns {
		quotedInsertCols[i] = b.dialect.QuoteIdentifier(col)
	}
	quotedReturnCols := make([]string, len(returningColumns))
	for i, col := range returningColumns {
		quotedReturnCols[i] = b.dialect.QuoteIdentifier(col)
	}
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		quotedTable,
		strings.Join(quotedInsertCols, ", "),
		strings.Join(values, ", "),
		strings.Join(quotedReturnCols, ", "),
	)

	row := b.db.QueryRow(ctx, query, args...)

	if b.modelType == nil {
		return row, nil
	}

	return b.scanRow(row)
}

// Update updates a record by primary key and returns the updated model
func (b *TableQueryBuilder) Update(ctx context.Context, id interface{}, data interface{}) (interface{}, error) {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	if b.primaryKey == "" {
		return nil, fmt.Errorf("%w: table %s", errors.ErrPrimaryKeyRequired, b.table)
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data must be a struct")
	}

	var updateColumns []string
	var args []interface{}
	argIndex := 1
	updatedAtAdded := false

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		fieldName := toSnakeCase(field.Name)
		quotedFieldName := b.dialect.QuoteIdentifier(fieldName)

		if fieldName == b.primaryKey || fieldName == "created_at" || fieldName == "deleted_at" {
			continue
		}

		if fieldName == "updated_at" {
			quotedUpdatedAt := b.dialect.QuoteIdentifier("updated_at")
			updateColumns = append(updateColumns, fmt.Sprintf("%s = NOW()", quotedUpdatedAt))
			updatedAtAdded = true
			continue
		}

		if fieldVal.IsZero() {
			continue
		}

		updateColumns = append(updateColumns, fmt.Sprintf("%s = $%d", quotedFieldName, argIndex))
		args = append(args, fieldVal.Interface())
		argIndex++
	}

	if len(updateColumns) == 0 {
		return nil, errors.ErrNoFieldsToUpdate
	}

	hasUpdatedAt := contains(b.columns, "updated_at")
	if hasUpdatedAt && !updatedAtAdded {
		quotedUpdatedAt := b.dialect.QuoteIdentifier("updated_at")
		updateColumns = append(updateColumns, fmt.Sprintf("%s = NOW()", quotedUpdatedAt))
	}

	quotedPK := b.dialect.QuoteIdentifier(b.primaryKey)
	whereClause := fmt.Sprintf("%s = $%d", quotedPK, argIndex)
	args = append(args, id)

	if b.hasDeleted {
		quotedDeletedAt := b.dialect.QuoteIdentifier("deleted_at")
		var sb strings.Builder
		sb.Grow(len(whereClause) + 20) // Pre-alocar espaço estimado
		sb.WriteString(whereClause)
		sb.WriteString(" AND ")
		sb.WriteString(quotedDeletedAt)
		sb.WriteString(" IS NULL")
		whereClause = sb.String()
	}

	quotedReturnCols := make([]string, len(b.columns))
	for i, col := range b.columns {
		quotedReturnCols[i] = b.dialect.QuoteIdentifier(col)
	}
	returningColumns := quotedReturnCols

	quotedTable := b.dialect.QuoteIdentifier(b.table)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s RETURNING %s",
		quotedTable,
		strings.Join(updateColumns, ", "),
		whereClause,
		strings.Join(returningColumns, ", "),
	)

	row := b.db.QueryRow(ctx, query, args...)

	if b.modelType == nil {
		return row, nil
	}

	return b.scanRow(row)
}

// Delete removes a record (soft delete if has deleted_at, otherwise hard delete)
func (b *TableQueryBuilder) Delete(ctx context.Context, id interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	if b.primaryKey == "" {
		return fmt.Errorf("%w: table %s", errors.ErrPrimaryKeyRequired, b.table)
	}

	var query string
	var args []interface{}

	quotedTable := b.dialect.QuoteIdentifier(b.table)
	quotedPK := b.dialect.QuoteIdentifier(b.primaryKey)
	quotedDeletedAt := b.dialect.QuoteIdentifier("deleted_at")
	if b.hasDeleted {
		query = fmt.Sprintf(
			"UPDATE %s SET %s = NOW() WHERE %s = $1 AND %s IS NULL",
			quotedTable,
			quotedDeletedAt,
			quotedPK,
			quotedDeletedAt,
		)
		args = []interface{}{id}
	} else {
		query = fmt.Sprintf(
			"DELETE FROM %s WHERE %s = $1",
			quotedTable,
			quotedPK,
		)
		args = []interface{}{id}
	}

	_, err := b.db.Exec(ctx, query, args...)
	return err
}

// buildQuery constructs the SQL query
func (b *TableQueryBuilder) buildQuery(where Where, opts *QueryOptions, single bool) (string, []interface{}) {
	var parts []string
	var args []interface{}
	argIndex := 1

	quotedColumns := make([]string, len(b.columns))
	for i, col := range b.columns {
		quotedColumns[i] = b.dialect.QuoteIdentifier(col)
	}
	columns := strings.Join(quotedColumns, ", ")
	quotedTable := b.dialect.QuoteIdentifier(b.table)
	parts = append(parts, fmt.Sprintf("SELECT %s FROM %s", columns, quotedTable))

	if len(where) > 0 {
		whereClause, whereArgs := b.buildWhereFromMap(where, &argIndex)
		if whereClause != "" {
			parts = append(parts, "WHERE "+whereClause)
			args = append(args, whereArgs...)
			argIndex += len(whereArgs)
		}
	}

	if opts != nil && len(opts.OrderBy) > 0 {
		var orderParts []string
		for _, order := range opts.OrderBy {
			quotedField := b.dialect.QuoteIdentifier(order.Field)
			orderDir := strings.ToUpper(strings.TrimSpace(order.Order))
			if orderDir != "ASC" && orderDir != "DESC" {
				orderDir = "ASC"
			}
			orderParts = append(orderParts, fmt.Sprintf("%s %s", quotedField, orderDir))
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}

	if !single {
		if opts != nil && (opts.Take != nil || opts.Skip != nil) {
			limit := 0
			offset := 0
			if opts.Take != nil {
				limit = *opts.Take
			}
			if opts.Skip != nil {
				offset = *opts.Skip
			}
			limitOffset := b.dialect.GetLimitOffsetSyntax(limit, offset)
			if limitOffset != "" {
				parts = append(parts, limitOffset)
			}
			// Note: GetLimitOffsetSyntax already includes the values in the SQL string,
			// so we don't need to add them to args
		}
	} else {
		parts = append(parts, "LIMIT 1")
	}

	return strings.Join(parts, " "), args
}

// buildWhereFromMap constructs the WHERE clause from a Prisma-style map
func (b *TableQueryBuilder) buildWhereFromMap(where Where, argIndex *int) (string, []interface{}) {
	var parts []string
	var args []interface{}

	for field, value := range where {
		quotedField := b.dialect.QuoteIdentifier(field)
		if op, ok := value.(WhereOperator); ok {
			switch op.GetOp() {
			case "IS NULL", "IS NOT NULL":
				parts = append(parts, fmt.Sprintf("%s %s", quotedField, op.GetOp()))
			case "IN", "NOT IN":
				if values, ok := op.GetValue().([]interface{}); ok {
					placeholders := make([]string, len(values))
					for i := range values {
						placeholders[i] = fmt.Sprintf("$%d", *argIndex)
						args = append(args, values[i])
						(*argIndex)++
					}
					parts = append(parts, fmt.Sprintf("%s %s (%s)", quotedField, op.GetOp(), strings.Join(placeholders, ", ")))
				}
			default:
				parts = append(parts, fmt.Sprintf("%s %s $%d", quotedField, op.GetOp(), *argIndex))
				args = append(args, op.GetValue())
				(*argIndex)++
			}
		} else if value == nil {
			parts = append(parts, fmt.Sprintf("%s IS NULL", quotedField))
		} else {
			parts = append(parts, fmt.Sprintf("%s = $%d", quotedField, *argIndex))
			args = append(args, value)
			(*argIndex)++
		}
	}

	return strings.Join(parts, " AND "), args
}

// buildColumnToFieldMap cria um mapa de nome de coluna para índice de campo
func buildColumnToFieldMap(modelType reflect.Type) map[string]int {
	columnToField := make(map[string]int)
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				jsonTag = jsonTag[:idx]
			}
			columnToField[jsonTag] = i
		}
	}
	return columnToField
}

// scanRow scans a single row into the model type
func (b *TableQueryBuilder) scanRow(row driver.Row) (interface{}, error) {
	if b.modelType == nil {
		err := fmt.Errorf("modelType not defined")
		return nil, errors.SanitizeError(err)
	}

	modelValue := reflect.New(b.modelType).Elem()

	columnToField := buildColumnToFieldMap(b.modelType)

	fields := make([]interface{}, len(b.columns))
	for i, colName := range b.columns {
		if fieldIdx, ok := columnToField[colName]; ok {
			field := modelValue.Field(fieldIdx)
			fields[i] = field.Addr().Interface()
		} else {
			var dummy interface{}
			fields[i] = &dummy
		}
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return modelValue.Interface(), nil
}

// scanRows scans multiple rows into a slice of models
func (b *TableQueryBuilder) scanRows(rows driver.Rows) (interface{}, error) {
	if b.modelType == nil {
		err := fmt.Errorf("modelType not defined")
		return nil, errors.SanitizeError(err)
	}

	columnToField := buildColumnToFieldMap(b.modelType)

	sliceType := reflect.SliceOf(b.modelType)
	initialCapacity := 16
	if len(b.columns) > 10 {
		initialCapacity = 32
	}
	if len(b.columns) > 20 {
		initialCapacity = 64
	}
	sliceValue := reflect.MakeSlice(sliceType, 0, initialCapacity)

	rowCount := 0
	fields := make([]interface{}, len(b.columns))

	for rows.Next() {
		if rowCount >= limits.MaxScanRows {
			return nil, fmt.Errorf("%w: maximum %d rows allowed", errors.ErrTooManyRows, limits.MaxScanRows)
		}

		modelValue := reflect.New(b.modelType).Elem()

		for i, colName := range b.columns {
			if fieldIdx, ok := columnToField[colName]; ok {
				field := modelValue.Field(fieldIdx)
				fields[i] = field.Addr().Interface()
			} else {
				var dummy interface{}
				fields[i] = &dummy
			}
		}

		if err := rows.Scan(fields...); err != nil {
			return nil, err
		}

		sliceValue = reflect.Append(sliceValue, modelValue)
		rowCount++

		if rowCount > 0 && rowCount%1000 == 0 {
			currentCap := sliceValue.Cap()
			if currentCap < rowCount*2 && currentCap < limits.MaxScanRows {
				newCap := rowCount * 2
				if newCap > limits.MaxScanRows {
					newCap = limits.MaxScanRows
				}
				newSlice := reflect.MakeSlice(sliceType, sliceValue.Len(), newCap)
				reflect.Copy(newSlice, sliceValue)
				sliceValue = newSlice
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sliceValue.Interface(), nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
