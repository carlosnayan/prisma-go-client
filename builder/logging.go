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
// nolint: unused // Mantido para uso futuro ou compatibilidade
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

// logQueryWithTiming loga query time e process time separadamente para transparência
// queryStart: início da chamada ao banco
// processStart: início de todo o processamento (incluindo construção da query)
// queryDuration: tempo de execução no banco (já calculado, pode ser diferente de time.Since(queryStart) para QueryRow)
func (q *Query) logQueryWithTiming(ctx context.Context, query string, args []interface{}, queryStart, processStart time.Time, queryDuration time.Duration) {
	logger := q.getLogger()
	if logger == nil {
		return
	}

	processDuration := time.Since(processStart)
	overheadDuration := processDuration - queryDuration

	// Log QUERY (já existente) - usa query time (tempo real no banco)
	logger.Query(query, args, queryDuration)

	// Log INFO com query time, overhead e tempo total
	queryType := detectQueryType(query)
	logger.Info("%s query: %v, overhead: %v (total: %v)", queryType, queryDuration, overheadDuration, processDuration)

	// Log WARN se query for lenta (> 1000ms)
	if queryDuration > 1000*time.Millisecond {
		logger.Warn("Slow query detected: %s took %v", queryType, queryDuration)
	}

	// Log WARN se overhead do ORM for muito grande (> 2x o tempo da query)
	if overheadDuration > 0 && queryDuration > 0 && overheadDuration > queryDuration*2 {
		logger.Warn("ORM overhead high: %v (query: %v, overhead: %v, %.1f%% overhead)",
			processDuration, queryDuration, overheadDuration,
			float64(overheadDuration)/float64(queryDuration)*100)
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
