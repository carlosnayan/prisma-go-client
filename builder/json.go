package builder

import (
	"context"
	"encoding/json"
	"fmt"
)

// JSONField representa um campo JSON para operações
type JSONField struct {
	field string
	query *Query
}

// JSON acessa um campo JSON para operações
func (q *Query) JSON(field string) *JSONField {
	return &JSONField{
		field: field,
		query: q,
	}
}

// Get obtém um valor de um campo JSON
// Exemplo: q.JSON("metadata").Get("key")
func (j *JSONField) Get(ctx context.Context, key string) (interface{}, error) {
	if err := validateJSONKey(key); err != nil {
		return nil, fmt.Errorf("invalid JSON key: %w", err)
	}

	quotedField := j.query.dialect.QuoteIdentifier(j.field)
	quotedTable := j.query.dialect.QuoteIdentifier(j.query.table)
	query := fmt.Sprintf("SELECT %s->>$1 FROM %s", quotedField, quotedTable)
	row := j.query.db.QueryRow(ctx, query, key)

	var result interface{}
	err := row.Scan(&result)
	return result, err
}

// Set atualiza um valor em um campo JSON
// Exemplo: q.JSON("metadata").Set("key", "value")
func (j *JSONField) Set(ctx context.Context, key string, value interface{}) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("erro ao serializar valor JSON: %w", err)
	}

	quotedTable := j.query.dialect.QuoteIdentifier(j.query.table)
	quotedField := j.query.dialect.QuoteIdentifier(j.field)
	quotedPK := j.query.dialect.QuoteIdentifier(j.query.primaryKey)

	query := fmt.Sprintf("UPDATE %s SET %s = jsonb_set(%s, ARRAY[$1], $2::jsonb) WHERE %s = $3",
		quotedTable, quotedField, quotedField, quotedPK)

	if len(j.query.whereConditions) == 0 {
		return fmt.Errorf("JSON.Set requer uma condição WHERE ou ID")
	}

	if len(j.query.whereConditions) > 0 {
		argIndex := 4
		whereClause, whereArgs := j.query.buildWhereClause(&argIndex)
		if whereClause != "" {
			query += " AND " + whereClause
			allArgs := []interface{}{key, string(valueJSON), nil}
			allArgs = append(allArgs, whereArgs...)
			_, err = j.query.db.Exec(ctx, query, allArgs...)
			return err
		}
	}

	_, err = j.query.db.Exec(ctx, query, key, string(valueJSON), nil)
	return err
}

// Contains verifica se um campo JSON contém uma chave
func (j *JSONField) Contains(key string) *Query {
	quotedField := j.query.dialect.QuoteIdentifier(j.field)
	j.query.whereConditions = append(j.query.whereConditions, whereCondition{
		query: fmt.Sprintf("%s ? $1", quotedField),
		args:  []interface{}{key},
		or:    false,
	})
	return j.query
}

// Path acessa um caminho aninhado em JSON
// Exemplo: q.JSON("metadata").Path("user", "name")
func (j *JSONField) Path(keys ...string) *JSONField {
	for _, key := range keys {
		if err := validateJSONKey(key); err != nil {
			return &JSONField{
				field: "",
				query: j.query,
			}
		}
	}

	quotedField := j.query.dialect.QuoteIdentifier(j.field)
	path := quotedField
	for i, key := range keys {
		if i == len(keys)-1 {
			path = fmt.Sprintf("%s->>$%d", path, i+1)
		} else {
			path = fmt.Sprintf("%s->$%d", path, i+1)
		}
		j.query.whereConditions = append(j.query.whereConditions, whereCondition{
			query: path,
			args:  []interface{}{key},
			or:    false,
		})
	}

	return &JSONField{
		field: path,
		query: j.query,
	}
}

// validateJSONKey valida que uma JSON key é segura
func validateJSONKey(key string) error {
	if key == "" {
		return fmt.Errorf("JSON key cannot be empty")
	}

	if len(key) > 255 {
		return fmt.Errorf("JSON key too long (max 255 characters)")
	}

	for _, r := range key {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			if r == '\'' || r == '"' || r == '\\' || r == ';' || r == '\n' || r == '\r' {
				return fmt.Errorf("JSON key contains invalid character: %c", r)
			}
		}
	}

	return nil
}
