package migrations

import (
	"database/sql"
	"fmt"
	"time"
)

// PoolConfig configura o pool de conexões
type PoolConfig struct {
	MaxOpenConns    int           // Número máximo de conexões abertas
	MaxIdleConns    int           // Número máximo de conexões ociosas
	ConnMaxLifetime time.Duration // Tempo máximo de vida de uma conexão
	ConnMaxIdleTime time.Duration // Tempo máximo que uma conexão pode ficar ociosa
}

// DefaultPoolConfig retorna configuração padrão do pool
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// ConfigurePool configura o pool de conexões do banco
func ConfigurePool(db *sql.DB, config *PoolConfig) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

// ConnectDatabaseWithPool conecta ao banco e configura o pool
func ConnectDatabaseWithPool(url string, poolConfig *PoolConfig) (*sql.DB, error) {
	db, err := ConnectDatabase(url)
	if err != nil {
		return nil, err
	}

	ConfigurePool(db, poolConfig)
	return db, nil
}

// GetPoolStats retorna estatísticas do pool de conexões
func GetPoolStats(db *sql.DB) (openConns, idleConns int, err error) {
	stats := db.Stats()
	return stats.OpenConnections, stats.Idle, nil
}

// PrintPoolStats imprime estatísticas do pool (útil para debug)
func PrintPoolStats(db *sql.DB) {
	stats := db.Stats()
	fmt.Printf("Pool Stats:\n")
	fmt.Printf("  Open Connections: %d\n", stats.OpenConnections)
	fmt.Printf("  Idle Connections: %d\n", stats.Idle)
	fmt.Printf("  In Use: %d\n", stats.InUse)
	fmt.Printf("  Wait Count: %d\n", stats.WaitCount)
	fmt.Printf("  Wait Duration: %v\n", stats.WaitDuration)
	fmt.Printf("  Max Idle Closed: %d\n", stats.MaxIdleClosed)
	fmt.Printf("  Max Idle Time Closed: %d\n", stats.MaxIdleTimeClosed)
	fmt.Printf("  Max Lifetime Closed: %d\n", stats.MaxLifetimeClosed)
}
