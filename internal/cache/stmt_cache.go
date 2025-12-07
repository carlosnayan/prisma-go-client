package cache

import (
	"context"
	"sync"
	"time"
)

const (
	// DefaultMaxQuerySize é o tamanho máximo padrão para uma query individual no cache (1MB)
	DefaultMaxQuerySize = 1024 * 1024
	// DefaultMaxTotalSize é o tamanho máximo total padrão para o cache (10MB)
	DefaultMaxTotalSize = 10 * 1024 * 1024
)

// StmtCache é um cache de prepared statements
type StmtCache struct {
	mu           sync.RWMutex
	queries      map[string]*CachedStmt
	maxSize      int
	maxQuerySize int // Tamanho máximo por query
	maxTotalSize int // Tamanho máximo total do cache
	currentSize  int // Tamanho atual do cache em bytes
	ttl          time.Duration
}

// CachedStmt representa um prepared statement em cache
type CachedStmt struct {
	Query       string
	LastUsed    time.Time
	AccessCount int64
	Size        int // Tamanho da query em bytes
}

// NewStmtCache cria um novo cache de prepared statements
func NewStmtCache(maxSize int, ttl time.Duration) *StmtCache {
	return &StmtCache{
		queries:      make(map[string]*CachedStmt),
		maxSize:      maxSize,
		maxQuerySize: DefaultMaxQuerySize,
		maxTotalSize: DefaultMaxTotalSize,
		currentSize:  0,
		ttl:          ttl,
	}
}

// NewStmtCacheWithLimits cria um cache com limites de memória customizados
func NewStmtCacheWithLimits(maxSize int, maxQuerySize, maxTotalSize int, ttl time.Duration) *StmtCache {
	return &StmtCache{
		queries:      make(map[string]*CachedStmt),
		maxSize:      maxSize,
		maxQuerySize: maxQuerySize,
		maxTotalSize: maxTotalSize,
		currentSize:  0,
		ttl:          ttl,
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

	querySize := len(query)

	// Verificar se a query individual excede o limite
	if querySize > c.maxQuerySize {
		// Query muito grande, não cachear
		return
	}

	// Se já existe, atualizar
	if existing, exists := c.queries[query]; exists {
		existing.LastUsed = time.Now()
		existing.AccessCount++
		return
	}

	// Verificar se adicionar esta query excederia o limite total
	// Se sim, evict até ter espaço
	for c.currentSize+querySize > c.maxTotalSize || len(c.queries) >= c.maxSize {
		if !c.evictLRU() {
			// Não há mais nada para remover
			break
		}
	}

	// Adicionar ao cache
	c.queries[query] = &CachedStmt{
		Query:       query,
		LastUsed:    time.Now(),
		AccessCount: 1,
		Size:        querySize,
	}
	c.currentSize += querySize
}

// evictLRU remove o item menos usado recentemente
// Retorna true se removeu algo, false caso contrário
func (c *StmtCache) evictLRU() bool {
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
		stmt := c.queries[oldestKey]
		c.currentSize -= stmt.Size
		delete(c.queries, oldestKey)
		return true
	}
	return false
}

// Cleanup remove queries expiradas do cache
func (c *StmtCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, stmt := range c.queries {
		if now.Sub(stmt.LastUsed) > c.ttl {
			c.currentSize -= stmt.Size
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
func (c *StmtCache) Stats() (size int, totalAccesses int64, currentSize int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	size = len(c.queries)
	for _, stmt := range c.queries {
		totalAccesses += stmt.AccessCount
	}
	return size, totalAccesses, c.currentSize
}
