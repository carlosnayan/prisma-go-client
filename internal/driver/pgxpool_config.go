//go:build pgx

package driver

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig configura o pool de conexões pgx
type PoolConfig struct {
	MaxConns              int32         // Número máximo de conexões no pool
	MinConns              int32         // Número mínimo de conexões no pool
	MaxConnLifetime       time.Duration // Tempo máximo de vida de uma conexão
	MaxConnIdleTime       time.Duration // Tempo máximo que uma conexão pode ficar ociosa
	HealthCheckPeriod     time.Duration // Período entre health checks
	MaxConnLifetimeJitter time.Duration // Jitter para MaxConnLifetime
}

// DefaultPoolConfig retorna configuração padrão otimizada para produção
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConns:              25,
		MinConns:              5,
		MaxConnLifetime:       30 * time.Minute,
		MaxConnIdleTime:       5 * time.Minute,
		HealthCheckPeriod:     1 * time.Minute,
		MaxConnLifetimeJitter: 30 * time.Second,
	}
}

// ConfigurePgxPool configura um pool pgx com as opções especificadas
func ConfigurePgxPool(config *pgxpool.Config, poolConfig *PoolConfig) error {
	if poolConfig == nil {
		poolConfig = DefaultPoolConfig()
	}

	config.MaxConns = poolConfig.MaxConns
	config.MinConns = poolConfig.MinConns
	config.MaxConnLifetime = poolConfig.MaxConnLifetime
	config.MaxConnIdleTime = poolConfig.MaxConnIdleTime
	config.HealthCheckPeriod = poolConfig.HealthCheckPeriod
	config.MaxConnLifetimeJitter = poolConfig.MaxConnLifetimeJitter

	return nil
}

// NewPgxPoolWithConfig cria um novo pool pgx com configuração customizada
func NewPgxPoolWithConfig(ctx context.Context, databaseURL string, poolConfig *PoolConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	if err := ConfigurePgxPool(config, poolConfig); err != nil {
		return nil, err
	}

	return pgxpool.NewWithConfig(ctx, config)
}
