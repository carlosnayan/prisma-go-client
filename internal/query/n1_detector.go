package query

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// N1Detector detecta padrões de N+1 queries
type N1Detector struct {
	mu         sync.RWMutex
	queries    map[string]*QueryInfo
	maxSize    int           // Número máximo de padrões a rastrear (previne crescimento ilimitado)
	threshold  int           // Número mínimo de queries similares para alertar
	timeWindow time.Duration // Janela de tempo para análise
}

// QueryInfo armazena informações sobre uma query
type QueryInfo struct {
	Pattern    string
	Count      int
	FirstSeen  time.Time
	LastSeen   time.Time
	TableNames []string
}

const (
	// MaxTableNames limita o número de nomes de tabelas rastreados por padrão
	MaxTableNames = 10
	// DefaultMaxQueries é o número padrão máximo de padrões de query a rastrear
	DefaultMaxQueries = 1000
)

// NewN1Detector cria um novo detector de N+1 queries
func NewN1Detector(threshold int, timeWindow time.Duration) *N1Detector {
	return &N1Detector{
		queries:    make(map[string]*QueryInfo),
		maxSize:    DefaultMaxQueries,
		threshold:  threshold,
		timeWindow: timeWindow,
	}
}

// NewN1DetectorWithMaxSize cria um detector com tamanho máximo customizado
func NewN1DetectorWithMaxSize(threshold int, timeWindow time.Duration, maxSize int) *N1Detector {
	return &N1Detector{
		queries:    make(map[string]*QueryInfo),
		maxSize:    maxSize,
		threshold:  threshold,
		timeWindow: timeWindow,
	}
}

// DefaultN1Detector returns a detector with default settings
func DefaultN1Detector() *N1Detector {
	return NewN1Detector(5, 1*time.Second)
}

// Record records an executed query
func (d *N1Detector) Record(query string, tableName string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pattern := normalizeQuery(query)

	info, exists := d.queries[pattern]
	if !exists {
		if len(d.queries) >= d.maxSize {
			d.evictOldest()
		}

		info = &QueryInfo{
			Pattern:    pattern,
			Count:      0,
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
			TableNames: []string{},
		}
		d.queries[pattern] = info
	}

	info.Count++
	info.LastSeen = time.Now()

	// Adicionar nome da tabela se não existir (com limite para prevenir crescimento ilimitado)
	found := false
	for _, tn := range info.TableNames {
		if tn == tableName {
			found = true
			break
		}
	}
	if !found && len(info.TableNames) < MaxTableNames {
		info.TableNames = append(info.TableNames, tableName)
	}
}

// Check checks for N+1 query patterns and returns alerts
func (d *N1Detector) Check() []N1Alert {
	d.mu.Lock()
	defer d.mu.Unlock()

	var alerts []N1Alert
	now := time.Now()

	for pattern, info := range d.queries {
		if now.Sub(info.FirstSeen) > d.timeWindow {
			delete(d.queries, pattern)
			continue
		}

		if info.Count >= d.threshold {
			alerts = append(alerts, N1Alert{
				Pattern:    pattern,
				Count:      info.Count,
				TableNames: info.TableNames,
				TimeWindow: now.Sub(info.FirstSeen),
			})
		}
	}

	return alerts
}

// N1Alert represents an N+1 query alert
type N1Alert struct {
	Pattern    string
	Count      int
	TableNames []string
	TimeWindow time.Duration
}

// String returns a string representation of the alert
func (a N1Alert) String() string {
	return fmt.Sprintf("N+1 Query detected: pattern '%s' executed %d times in %v on tables %v",
		a.Pattern, a.Count, a.TimeWindow, a.TableNames)
}

// evictOldest removes the oldest query pattern from the map
func (d *N1Detector) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, info := range d.queries {
		if first || info.FirstSeen.Before(oldestTime) {
			oldestKey = key
			oldestTime = info.FirstSeen
			first = false
		}
	}

	if oldestKey != "" {
		delete(d.queries, oldestKey)
	}
}

// normalizeQuery normalizes a query to create a pattern
func normalizeQuery(query string) string {
	if len(query) > 50 {
		return query[:50] + "..."
	}
	return query
}

// StartMonitoring inicia monitoramento contínuo de N+1 queries
func (d *N1Detector) StartMonitoring(ctx context.Context, interval time.Duration, callback func([]N1Alert)) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				alerts := d.Check()
				if len(alerts) > 0 && callback != nil {
					callback(alerts)
				}
			}
		}
	}()
}
