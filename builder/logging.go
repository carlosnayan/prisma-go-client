package builder

import (
	"context"
	"strings"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/logger"
)

// detectQueryType detecta o tipo de query SQL (SELECT, INSERT, UPDATE, DELETE)
func detectQueryType(query string) string {
	trimmed := strings.TrimSpace(query)
	upper := strings.ToUpper(trimmed)
	
	if strings.HasPrefix(upper, "SELECT") {
		return "SELECT"
	}
	if strings.HasPrefix(upper, "INSERT") {
		return "INSERT"
	}
	if strings.HasPrefix(upper, "UPDATE") {
		return "UPDATE"
	}
	if strings.HasPrefix(upper, "DELETE") {
		return "DELETE"
	}
	
	return "UNKNOWN"
}

// logQuery loga uma query SQL se o logging estiver habilitado
func (q *Query) logQuery(ctx context.Context, query string, args []interface{}, start time.Time) {
	logger := q.getLogger()
	if logger == nil {
		return
	}
	duration := time.Since(start)
	
	// Log QUERY (já existente)
	logger.Query(query, args, duration)
	
	// Log INFO com tipo de query e tempo
	queryType := detectQueryType(query)
	logger.Info("%s executed in %v", queryType, duration)
	
	// Log WARN se query for lenta (> 1000ms)
	if duration > 1000*time.Millisecond {
		logger.Warn("Slow query detected: %s took %v", queryType, duration)
	}
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
