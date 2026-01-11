//go:build pgx

package driver

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConfigurePgxPool(t *testing.T) {
	config, _ := pgxpool.ParseConfig("postgresql://localhost:5432/test")

	poolCfg := &PoolConfig{
		MaxConns:              50,
		MinConns:              10,
		MaxConnLifetime:       1 * time.Hour,
		MaxConnIdleTime:       15 * time.Minute,
		HealthCheckPeriod:     2 * time.Minute,
		MaxConnLifetimeJitter: 1 * time.Minute,
		ConnectTimeout:        10 * time.Second,
	}

	err := ConfigurePgxPool(config, poolCfg)
	if err != nil {
		t.Fatalf("ConfigurePgxPool failed: %v", err)
	}

	if config.MaxConns != 50 {
		t.Errorf("expected MaxConns 50, got %d", config.MaxConns)
	}
	if config.MinConns != 10 {
		t.Errorf("expected MinConns 10, got %d", config.MinConns)
	}
	if config.MaxConnLifetime != 1*time.Hour {
		t.Errorf("expected MaxConnLifetime 1h, got %v", config.MaxConnLifetime)
	}
	if config.MaxConnIdleTime != 15*time.Minute {
		t.Errorf("expected MaxConnIdleTime 15m, got %v", config.MaxConnIdleTime)
	}
	if config.HealthCheckPeriod != 2*time.Minute {
		t.Errorf("expected HealthCheckPeriod 2m, got %v", config.HealthCheckPeriod)
	}
	if config.MaxConnLifetimeJitter != 1*time.Minute {
		t.Errorf("expected MaxConnLifetimeJitter 1m, got %v", config.MaxConnLifetimeJitter)
	}
	if config.ConnConfig.ConnectTimeout != 10*time.Second {
		t.Errorf("expected ConnectTimeout 10s, got %v", config.ConnConfig.ConnectTimeout)
	}
}

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()
	if cfg.MaxConns != 25 {
		t.Errorf("expected default MaxConns 25, got %d", cfg.MaxConns)
	}
	if cfg.MinConns != 5 {
		t.Errorf("expected default MinConns 5, got %d", cfg.MinConns)
	}
}
