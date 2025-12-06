package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	// Validar key para prevenir injection
	if err := validateJSONKey(key); err != nil {
		return nil, fmt.Errorf("invalid JSON key: %w", err)
	}

	// Escapar identificadores e usar parâmetro preparado para a key
	quotedField := j.query.dialect.QuoteIdentifier(j.field)
	quotedTable := j.query.dialect.QuoteIdentifier(j.query.table)
	// Usar parâmetro preparado para a key JSON
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

	// Escapar identificadores
	quotedTable := j.query.dialect.QuoteIdentifier(j.query.table)
	quotedField := j.query.dialect.QuoteIdentifier(j.field)
	quotedPK := j.query.dialect.QuoteIdentifier(j.query.primaryKey)

	// Usar parâmetros preparados para key e value
	// jsonb_set precisa de uma string JSON válida, então usamos $2::jsonb
	query := fmt.Sprintf("UPDATE %s SET %s = jsonb_set(%s, ARRAY[$1], $2::jsonb) WHERE %s = $3",
		quotedTable, quotedField, quotedField, quotedPK)

	// Nota: isso requer um ID, então precisamos de uma forma melhor
	// Por enquanto, retornar erro se não houver condições WHERE
	if len(j.query.whereConditions) == 0 {
		return fmt.Errorf("JSON.Set requer uma condição WHERE ou ID")
	}

	// Construir WHERE clause se houver condições
	if len(j.query.whereConditions) > 0 {
		argIndex := 4 // Começar após os parâmetros do SET
		whereClause, whereArgs := j.query.buildWhereClause(&argIndex)
		if whereClause != "" {
			query += " AND " + whereClause
			// Combinar todos os argumentos
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
	// Usar parâmetro preparado para a key
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
	// Validar todas as keys antes de processar
	for _, key := range keys {
		if err := validateJSONKey(key); err != nil {
			// Se alguma key for inválida, retornar erro silenciosamente
			// (será tratado quando a query for executada)
			return &JSONField{
				field: "",
				query: j.query,
			}
		}
	}

	// Construir caminho JSON usando parâmetros preparados
	quotedField := j.query.dialect.QuoteIdentifier(j.field)
	path := quotedField
	for i, key := range keys {
		// Usar parâmetros preparados para todas as keys
		if i == len(keys)-1 {
			// Última key usa ->> para retornar texto
			path = fmt.Sprintf("%s->>$%d", path, i+1)
		} else {
			// Keys intermediárias usam -> para manter JSON
			path = fmt.Sprintf("%s->$%d", path, i+1)
		}
		// Adicionar key aos argumentos da query
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

	// Rejeitar caracteres perigosos que podem ser usados em SQL injection
	dangerousChars := []string{"'", "\"", "\\", ";", "--", "/*", "*/", "xp_", "sp_", "exec", "select", "insert", "update", "delete", "drop", "create", "alter"}
	keyLower := strings.ToLower(key)
	for _, char := range dangerousChars {
		if strings.Contains(keyLower, char) {
			return fmt.Errorf("JSON key contains dangerous characters: %s", char)
		}
	}

	// Limitar tamanho da key
	if len(key) > 255 {
		return fmt.Errorf("JSON key too long (max 255 characters)")
	}

	return nil
}
