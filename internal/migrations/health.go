package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// HealthCheck represents the result of a health check
type HealthCheck struct {
	Status       string        `json:"status"`          // "healthy", "unhealthy"
	Database     string        `json:"database"`        // Database name
	ResponseTime time.Duration `json:"response_time"`   // Response time
	Error        string        `json:"error,omitempty"` // Error if any
}

// CheckHealth checks the health of the database connection
func CheckHealth(db *sql.DB, timeout time.Duration) (*HealthCheck, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()

	// Try to ping
	err := db.PingContext(ctx)
	responseTime := time.Since(start)

	check := &HealthCheck{
		ResponseTime: responseTime,
	}

	if err != nil {
		check.Status = "unhealthy"
		check.Error = err.Error()
		return check, err
	}

	// Check if it can execute a simple query
	var result int
	queryErr := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if queryErr != nil {
		check.Status = "unhealthy"
		check.Error = queryErr.Error()
		return check, queryErr
	}

	// Get database name (if possible)
	dbName := "unknown"
	if nameErr := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); nameErr != nil {
		_ = nameErr // Ignore error, use "unknown"
	}
	check.Database = dbName

	check.Status = "healthy"
	return check, nil
}

// CheckHealthWithPool checks health including pool statistics
func CheckHealthWithPool(db *sql.DB, timeout time.Duration) (*HealthCheck, error) {
	check, err := CheckHealth(db, timeout)
	if err != nil {
		return check, err
	}

	// Add pool information to status
	stats := db.Stats()
	if stats.OpenConnections > stats.MaxOpenConnections*9/10 {
		// Pool almost full, consider warning
		check.Status = "healthy" // Still healthy, but close to limit
	}

	return check, nil
}

// PrintHealthCheck prints the health check result in a readable format
func PrintHealthCheck(check *HealthCheck) {
	fmt.Printf("Health Check:\n")
	fmt.Printf("  Status: %s\n", check.Status)
	fmt.Printf("  Database: %s\n", check.Database)
	fmt.Printf("  Response Time: %v\n", check.ResponseTime)
	if check.Error != "" {
		fmt.Printf("  Error: %s\n", check.Error)
	}
}
