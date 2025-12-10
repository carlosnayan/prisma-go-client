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
	"github.com/carlosnayan/prisma-go-client/internal/uuid"
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
	var primaryKeyValue interface{}
	var primaryKeyCol string
	var primaryKeyType reflect.Kind
	var primaryKeyIsZero bool

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		dbTag := field.Tag.Get("db")
		fieldName := dbTag
		if fieldName == "" {
			fieldName = toSnakeCase(field.Name)
		}

		if fieldName == b.primaryKey {
			primaryKeyCol = fieldName
			primaryKeyValue = fieldVal.Interface()
			primaryKeyType = fieldVal.Kind()
			primaryKeyIsZero = fieldVal.IsZero()
			continue
		}

		if fieldVal.IsZero() {
			continue
		}

		insertColumns = append(insertColumns, fieldName)
		values = append(values, b.dialect.GetPlaceholder(argIndex))
		args = append(args, fieldVal.Interface())
		argIndex++
	}

	if primaryKeyCol != "" {
		if !primaryKeyIsZero {
			insertColumns = append(insertColumns, primaryKeyCol)
			values = append(values, b.dialect.GetPlaceholder(argIndex))
			args = append(args, primaryKeyValue)
		} else if primaryKeyType == reflect.String {
			generatedUUID := uuid.GenerateUUID()
			primaryKeyValue = generatedUUID
			insertColumns = append(insertColumns, primaryKeyCol)
			values = append(values, b.dialect.GetPlaceholder(argIndex))
			args = append(args, generatedUUID)
		}
	}

	// returningColumns must contain ONLY columns from the model
	returningColumns := make([]string, len(b.columns))
	copy(returningColumns, b.columns)

	quotedTable := b.dialect.QuoteIdentifier(b.table)
	quotedInsertCols := make([]string, len(insertColumns))
	for i, col := range insertColumns {
		quotedInsertCols[i] = b.dialect.QuoteIdentifier(col)
	}
	quotedReturnCols := make([]string, len(returningColumns))
	for i, col := range returningColumns {
		quotedReturnCols[i] = b.dialect.QuoteIdentifier(col)
	}

	var row interface{}
	if b.dialect.SupportsReturning() {
		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
			quotedTable,
			strings.Join(quotedInsertCols, ", "),
			strings.Join(values, ", "),
			strings.Join(quotedReturnCols, ", "),
		)
		row = b.db.QueryRow(ctx, query, args...)
	} else {
		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			quotedTable,
			strings.Join(quotedInsertCols, ", "),
			strings.Join(values, ", "),
		)
		result, err := b.db.Exec(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		// SQLite não retorna o modelo criado, apenas confirma sucesso
		if b.dialect.Name() == "sqlite" {
			return nil, nil
		}

		if primaryKeyCol != "" && primaryKeyValue != nil {
			selectQuery := fmt.Sprintf(
				"SELECT %s FROM %s WHERE %s = %s LIMIT 1",
				strings.Join(quotedReturnCols, ", "),
				quotedTable,
				b.dialect.QuoteIdentifier(primaryKeyCol),
				b.dialect.GetPlaceholder(1),
			)
			row = b.db.QueryRow(ctx, selectQuery, primaryKeyValue)
		} else if primaryKeyCol != "" {
			if b.dialect.Name() == "mysql" {
				selectQuery := fmt.Sprintf(
					"SELECT %s FROM %s WHERE %s = LAST_INSERT_ID() LIMIT 1",
					strings.Join(quotedReturnCols, ", "),
					quotedTable,
					b.dialect.QuoteIdentifier(primaryKeyCol),
				)
				row = b.db.QueryRow(ctx, selectQuery)
			} else {
				lastInsertID, err := result.LastInsertId()
				if err != nil || lastInsertID == 0 {
					return nil, fmt.Errorf("cannot retrieve inserted record: primary key was auto-generated but LastInsertId() failed: %v", err)
				}
				selectQuery := fmt.Sprintf(
					"SELECT %s FROM %s WHERE %s = %s LIMIT 1",
					strings.Join(quotedReturnCols, ", "),
					quotedTable,
					b.dialect.QuoteIdentifier(primaryKeyCol),
					b.dialect.GetPlaceholder(1),
				)
				row = b.db.QueryRow(ctx, selectQuery, lastInsertID)
			}
		} else {
			return nil, fmt.Errorf("cannot retrieve inserted record: no primary key and dialect does not support RETURNING")
		}
	}

	if b.modelType == nil {
		return row, nil
	}

	if driverRow, ok := row.(driver.Row); ok {
		return b.scanRow(driverRow)
	}
	return nil, fmt.Errorf("invalid row type")
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

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		fieldName := toSnakeCase(field.Name)
		quotedFieldName := b.dialect.QuoteIdentifier(fieldName)

		if fieldName == b.primaryKey {
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

	quotedPK := b.dialect.QuoteIdentifier(b.primaryKey)
	whereClause := fmt.Sprintf("%s = $%d", quotedPK, argIndex)
	args = append(args, id)

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

// Delete removes a record (hard delete)
func (b *TableQueryBuilder) Delete(ctx context.Context, id interface{}) error {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	if b.primaryKey == "" {
		return fmt.Errorf("%w: table %s", errors.ErrPrimaryKeyRequired, b.table)
	}

	quotedTable := b.dialect.QuoteIdentifier(b.table)
	quotedPK := b.dialect.QuoteIdentifier(b.primaryKey)
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = $1",
		quotedTable,
		quotedPK,
	)
	args := []interface{}{id}

	_, err := b.db.Exec(ctx, query, args...)
	return err
}

// CreateMany inserts multiple records and returns the number of records created
func (b *TableQueryBuilder) CreateMany(ctx context.Context, data []interface{}, skipDuplicates bool) (*BatchPayload, error) {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	if len(data) == 0 {
		return &BatchPayload{Count: 0}, nil
	}

	// Process first record to determine columns
	firstVal := reflect.ValueOf(data[0])
	if firstVal.Kind() == reflect.Ptr {
		firstVal = firstVal.Elem()
	}
	if firstVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data must be a slice of structs")
	}

	// Determine columns from first record
	var insertColumns []string
	typ := firstVal.Type()
	columnMap := make(map[string]bool)
	var primaryKeyCol string
	var primaryKeyType reflect.Kind

	// First, identify primary key field
	if b.primaryKey != "" {
		for i := 0; i < firstVal.NumField(); i++ {
			field := typ.Field(i)
			dbTag := field.Tag.Get("db")
			fieldName := dbTag
			if fieldName == "" {
				fieldName = toSnakeCase(field.Name)
			}
			if fieldName == b.primaryKey {
				primaryKeyCol = fieldName
				primaryKeyType = firstVal.Field(i).Kind()
				break
			}
		}
	}

	// Collect all non-primary key columns that are not zero
	for i := 0; i < firstVal.NumField(); i++ {
		field := typ.Field(i)
		dbTag := field.Tag.Get("db")
		fieldName := dbTag
		if fieldName == "" {
			fieldName = toSnakeCase(field.Name)
		}
		if fieldName != b.primaryKey && !firstVal.Field(i).IsZero() {
			insertColumns = append(insertColumns, fieldName)
			columnMap[fieldName] = true
		}
	}

	// Add primary key if it's set or if it's a string type (for UUID generation)
	if primaryKeyCol != "" {
		primaryKeySet := false
		for i := 0; i < firstVal.NumField(); i++ {
			field := typ.Field(i)
			dbTag := field.Tag.Get("db")
			fieldName := dbTag
			if fieldName == "" {
				fieldName = toSnakeCase(field.Name)
			}
			if fieldName == primaryKeyCol {
				if !firstVal.Field(i).IsZero() {
					insertColumns = append(insertColumns, fieldName)
					primaryKeySet = true
				}
				break
			}
		}
		// If primary key is string type and not set, we'll generate UUID for each record
		if !primaryKeySet && primaryKeyType == reflect.String {
			insertColumns = append(insertColumns, primaryKeyCol)
		}
	}

	quotedTable := b.dialect.QuoteIdentifier(b.table)
	quotedInsertCols := make([]string, len(insertColumns))
	for i, col := range insertColumns {
		quotedInsertCols[i] = b.dialect.QuoteIdentifier(col)
	}

	// Batch size for large inserts
	batchSize := 1000
	totalCount := 0

	for batchStart := 0; batchStart < len(data); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(data) {
			batchEnd = len(data)
		}
		batch := data[batchStart:batchEnd]

		var valuesParts []string
		var allArgs []interface{}
		argIndex := 1

		for _, item := range batch {
			val := reflect.ValueOf(item)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			itemTyp := val.Type()

			var rowValues []string
			var rowArgs []interface{}

			for _, col := range insertColumns {
				if col == primaryKeyCol && primaryKeyType == reflect.String {
					// Check if primary key is set for this item
					found := false
					for i := 0; i < val.NumField(); i++ {
						field := itemTyp.Field(i)
						dbTag := field.Tag.Get("db")
						fieldName := dbTag
						if fieldName == "" {
							fieldName = toSnakeCase(field.Name)
						}
						if fieldName == col {
							fieldVal := val.Field(i)
							if !fieldVal.IsZero() {
								rowArgs = append(rowArgs, fieldVal.Interface())
								found = true
								break
							}
						}
					}
					if !found {
						rowArgs = append(rowArgs, uuid.GenerateUUID())
					}
				} else {
					// Find field by column name
					found := false
					for i := 0; i < val.NumField(); i++ {
						field := itemTyp.Field(i)
						dbTag := field.Tag.Get("db")
						fieldName := dbTag
						if fieldName == "" {
							fieldName = toSnakeCase(field.Name)
						}
						if fieldName == col {
							fieldVal := val.Field(i)
							rowArgs = append(rowArgs, fieldVal.Interface())
							found = true
							break
						}
					}
					if !found {
						rowArgs = append(rowArgs, nil)
					}
				}
				rowValues = append(rowValues, b.dialect.GetPlaceholder(argIndex))
				argIndex++
			}
			valuesParts = append(valuesParts, "("+strings.Join(rowValues, ", ")+")")
			allArgs = append(allArgs, rowArgs...)
		}

		onConflict := ""
		if skipDuplicates && b.dialect.Name() == "postgresql" {
			onConflict = " ON CONFLICT DO NOTHING"
		} else if skipDuplicates && b.dialect.Name() == "mysql" {
			onConflict = " ON DUPLICATE KEY UPDATE " + quotedInsertCols[0] + " = " + quotedInsertCols[0]
		}

		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s%s",
			quotedTable,
			strings.Join(quotedInsertCols, ", "),
			strings.Join(valuesParts, ", "),
			onConflict,
		)

		result, err := b.db.Exec(ctx, query, allArgs...)
		if err != nil {
			return &BatchPayload{Count: totalCount}, err
		}

		rowsAffected := result.RowsAffected()
		totalCount += int(rowsAffected)
	}

	return &BatchPayload{Count: totalCount}, nil
}

// UpdateMany updates multiple records matching the where conditions and returns the number of records updated
func (b *TableQueryBuilder) UpdateMany(ctx context.Context, where Where, data interface{}) (*BatchPayload, error) {
	ctx, cancel := contextutil.WithQueryTimeout(ctx)
	defer cancel()

	if len(where) == 0 {
		return nil, fmt.Errorf("where condition is required for UpdateMany (empty where would update all records)")
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

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		dbTag := field.Tag.Get("db")
		fieldName := dbTag
		if fieldName == "" {
			fieldName = toSnakeCase(field.Name)
		}
		quotedFieldName := b.dialect.QuoteIdentifier(fieldName)

		if fieldName == b.primaryKey {
			continue
		}

		if fieldVal.IsZero() {
			continue
		}

		updateColumns = append(updateColumns, fmt.Sprintf("%s = %s", quotedFieldName, b.dialect.GetPlaceholder(argIndex)))
		args = append(args, fieldVal.Interface())
		argIndex++
	}

	if len(updateColumns) == 0 {
		return nil, errors.ErrNoFieldsToUpdate
	}

	whereClause, whereArgs := b.buildWhereFromMap(where, &argIndex)
	if whereClause == "" {
		return nil, fmt.Errorf("where condition is required for UpdateMany")
	}
	args = append(args, whereArgs...)

	quotedTable := b.dialect.QuoteIdentifier(b.table)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		quotedTable,
		strings.Join(updateColumns, ", "),
		whereClause,
	)

	result, err := b.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected := result.RowsAffected()
	return &BatchPayload{Count: int(rowsAffected)}, nil
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
				orderDir = "ASC" // Default seguro
			}
			orderParts = append(orderParts, fmt.Sprintf("%s %s", quotedField, orderDir))
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}

	if !single {
		if opts != nil {
			limit := 0
			offset := 0
			hasLimit := false
			if opts.Take != nil {
				limit = *opts.Take
				hasLimit = true
			}
			if opts.Skip != nil {
				offset = *opts.Skip
				hasLimit = true
			}
			if hasLimit {
				limitOffset := b.dialect.GetLimitOffsetSyntax(limit, offset)
				if limitOffset != "" {
					parts = append(parts, limitOffset)
				}
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

func buildColumnToFieldMap(modelType reflect.Type, columns []string) map[string]int {
	columnToField := make(map[string]int)

	// Build a reverse map: field identifier -> field index
	// This allows us to quickly find fields by their various identifiers
	fieldMap := make(map[string]int)

	// First, build a map of all possible field identifiers to field indices
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")

		// Remove options from json tag (e.g., "id,omitempty" -> "id")
		if jsonTag != "" && jsonTag != "-" {
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				jsonTag = jsonTag[:idx]
			}
		}

		// Map all possible identifiers to this field index
		// Priority: dbTag > jsonTag > snake_case field name
		if dbTag != "" {
			fieldMap[dbTag] = i
		}
		if jsonTag != "" && jsonTag != "-" {
			fieldMap[jsonTag] = i
		}
		// Also map snake_case field name
		fieldName := toSnakeCase(field.Name)
		if fieldName != "" {
			fieldMap[fieldName] = i
		}
	}

	// Now iterate through columns and find matching fields
	// This ensures all columns are checked and mapped
	for _, col := range columns {
		if idx, ok := fieldMap[col]; ok {
			columnToField[col] = idx
		}
		// If column not found in fieldMap, it will not be in columnToField
		// and scanRow will use a dummy variable for it
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

	columnToField := buildColumnToFieldMap(b.modelType, b.columns)

	fields := make([]interface{}, len(b.columns))
	mappedCount := 0
	for i, colName := range b.columns {
		if fieldIdx, ok := columnToField[colName]; ok {
			field := modelValue.Field(fieldIdx)
			fields[i] = field.Addr().Interface()
			mappedCount++
		} else {
			var dummy interface{}
			fields[i] = &dummy
		}
	}

	err := row.Scan(fields...)
	if err != nil {
		// Log detailed error information for debugging
		return nil, fmt.Errorf("scan failed: %w (columns: %v, mapped: %d/%d)", err, b.columns, mappedCount, len(b.columns))
	}

	return modelValue.Interface(), nil
}

// scanRows scans multiple rows into a slice of models
func (b *TableQueryBuilder) scanRows(rows driver.Rows) (interface{}, error) {
	if b.modelType == nil {
		err := fmt.Errorf("modelType not defined")
		return nil, errors.SanitizeError(err)
	}

	columnToField := buildColumnToFieldMap(b.modelType, b.columns)

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

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := s[i-1]
			if prev >= 'a' && prev <= 'z' {
				result.WriteByte('_')
			} else if i < len(s)-1 {
				next := s[i+1]
				if next >= 'a' && next <= 'z' {
					result.WriteByte('_')
				}
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
