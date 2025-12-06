package cache

import (
	"context"
	"sync"
	"time"
)

// StmtCache é um cache de prepared statements
type StmtCache struct {
	mu      sync.RWMutex
	queries map[string]*CachedStmt
	maxSize int
	ttl     time.Duration
}

// CachedStmt representa um prepared statement em cache
type CachedStmt struct {
	Query      string
	LastUsed   time.Time
	AccessCount int64
}

// NewStmtCache cria um novo cache de prepared statements
func NewStmtCache(maxSize int, ttl time.Duration) *StmtCache {
	return &StmtCache{
		queries: make(map[string]*CachedStmt),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// DefaultStmtCache retorna um cache com configurações padrão
func DefaultStmtCache() *StmtCache {
	return NewStmtCache(100, 5*time.Minute)
}

// Get retorna uma query do cache se existir e não estiver expirada
func (c *StmtCache) Get(query string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stmt, exists := c.queries[query]
	if !exists {
		return "", false
	}

	// Verificar se expirou
	if time.Since(stmt.LastUsed) > c.ttl {
		return "", false
	}

	// Atualizar estatísticas
	stmt.LastUsed = time.Now()
	stmt.AccessCount++

	return stmt.Query, true
}

// Put adiciona uma query ao cache
func (c *StmtCache) Put(query string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Se cache está cheio, remover o menos usado
	if len(c.queries) >= c.maxSize {
		c.evictLRU()
	}

	c.queries[query] = &CachedStmt{
		Query:       query,
		LastUsed:    time.Now(),
		AccessCount: 1,
	}
}

// evictLRU remove o item menos usado recentemente
func (c *StmtCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, stmt := range c.queries {
		if first || stmt.LastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = stmt.LastUsed
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.queries, oldestKey)
	}
}

// Cleanup remove queries expiradas do cache
func (c *StmtCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, stmt := range c.queries {
		if now.Sub(stmt.LastUsed) > c.ttl {
			delete(c.queries, key)
		}
	}
}

// StartCleanup inicia uma goroutine que limpa o cache periodicamente
func (c *StmtCache) StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				c.Cleanup()
			}
		}
	}()
}

// Stats retorna estatísticas do cache
func (c *StmtCache) Stats() (size int, totalAccesses int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	size = len(c.queries)
	for _, stmt := range c.queries {
		totalAccesses += stmt.AccessCount
	}
	return size, totalAccesses
}

