package migrations

import (
	"database/sql"
	"fmt"
	"time"
)

// PoolConfig configures the connection pool
type PoolConfig struct {
	MaxOpenConns    int           // Maximum number of open connections
	MaxIdleConns    int           // Maximum number of idle connections
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection
	ConnMaxIdleTime time.Duration // Maximum time a connection can be idle
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// ConfigurePool configures the database connection pool
func ConfigurePool(db *sql.DB, config *PoolConfig) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

// ConnectDatabaseWithPool connects to the database and configures the pool
func ConnectDatabaseWithPool(url string, poolConfig *PoolConfig) (*sql.DB, error) {
	db, err := ConnectDatabase(url)
	if err != nil {
		return nil, err
	}

	ConfigurePool(db, poolConfig)
	return db, nil
}

// GetPoolStats returns connection pool statistics
func GetPoolStats(db *sql.DB) (openConns, idleConns int, err error) {
	stats := db.Stats()
	return stats.OpenConnections, stats.Idle, nil
}

// PrintPoolStats prints pool statistics (useful for debugging)
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
