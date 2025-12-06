package builder

import (
	"context"
	"fmt"
)

// Restore restaura um registro soft-deleted
func (q *Query) Restore(ctx context.Context, id interface{}) error {
	if !q.hasDeleted {
		return fmt.Errorf("tabela não tem suporte a soft delete (deleted_at não encontrado)")
	}

	if q.primaryKey == "" {
		return fmt.Errorf("primary key não definida")
	}

	deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
	tableName := q.dialect.QuoteIdentifier(q.table)
	primaryKeyField := q.dialect.QuoteIdentifier(q.primaryKey)
	query := fmt.Sprintf("UPDATE %s SET %s = NULL WHERE %s = %s AND %s IS NOT NULL",
		tableName, deletedAtField, primaryKeyField, q.dialect.GetPlaceholder(1), deletedAtField)
	_, err := q.db.Exec(ctx, query, id)
	return err
}

// ForceDelete remove permanentemente um registro (ignora soft delete)
func (q *Query) ForceDelete(ctx context.Context, id interface{}) error {
	if q.primaryKey == "" {
		return fmt.Errorf("primary key não definida")
	}

	tableName := q.dialect.QuoteIdentifier(q.table)
	primaryKeyField := q.dialect.QuoteIdentifier(q.primaryKey)
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s", tableName, primaryKeyField, q.dialect.GetPlaceholder(1))
	_, err := q.db.Exec(ctx, query, id)
	return err
}

// IncludeDeleted modifica a query para incluir registros deletados
func (q *Query) IncludeDeleted() *Query {
	q.includeDeleted = true
	return q
}

// OnlyDeleted modifica a query para retornar apenas registros deletados
func (q *Query) OnlyDeleted() *Query {
	if !q.hasDeleted {
		return q
	}

	// Adicionar condição para apenas deletados
	deletedAtField := q.dialect.QuoteIdentifier("deleted_at")
	q.whereConditions = append(q.whereConditions, whereCondition{
		query: fmt.Sprintf("%s IS NOT NULL", deletedAtField),
		args:  []interface{}{},
		or:    false,
	})
	return q
}
