package contextutil

import (
	"context"
	"time"
)

// DefaultTimeout é o timeout padrão para operações de banco de dados
var DefaultTimeout = 30 * time.Second

// WithTimeout cria um contexto com timeout, usando o timeout padrão se não especificado
func WithTimeout(ctx context.Context, timeout ...time.Duration) (context.Context, context.CancelFunc) {
	t := DefaultTimeout
	if len(timeout) > 0 && timeout[0] > 0 {
		t = timeout[0]
	}
	return context.WithTimeout(ctx, t)
}

// WithQueryTimeout cria um contexto com timeout para queries (5 segundos)
func WithQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}

// WithTransactionTimeout cria um contexto com timeout para transações (30 segundos)
func WithTransactionTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 30*time.Second)
}

// WithMigrationTimeout cria um contexto com timeout para migrations (5 minutos)
func WithMigrationTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Minute)
}

