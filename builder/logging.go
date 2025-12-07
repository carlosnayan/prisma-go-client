package builder

import (
	"context"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/logger"
)

// logQuery loga uma query SQL se o logging estiver habilitado
func (q *Query) logQuery(ctx context.Context, query string, args []interface{}, start time.Time) {
	if q.logger == nil {
		return
	}
	duration := time.Since(start)
	q.logger.Query(query, args, duration)
}

// setLogger define o logger para a query
func (q *Query) SetLogger(l *logger.Logger) *Query {
	q.logger = l
	return q
}

// SetLogLevels configura os níveis de log do logger padrão
// Esta é uma função pública que pode ser usada no código gerado
func SetLogLevels(levels []string) {
	logger.SetLogLevels(levels)
}
