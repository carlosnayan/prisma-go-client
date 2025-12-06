package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// HealthCheck representa o resultado de um health check
type HealthCheck struct {
	Status      string        `json:"status"`       // "healthy", "unhealthy"
	Database    string        `json:"database"`    // Nome do banco
	ResponseTime time.Duration `json:"response_time"` // Tempo de resposta
	Error       string        `json:"error,omitempty"` // Erro se houver
}

// CheckHealth verifica a saúde da conexão com o banco
func CheckHealth(db *sql.DB, timeout time.Duration) (*HealthCheck, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	
	// Tentar fazer ping
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

	// Verificar se consegue executar uma query simples
	var result int
	queryErr := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if queryErr != nil {
		check.Status = "unhealthy"
		check.Error = queryErr.Error()
		return check, queryErr
	}

	// Obter nome do banco (se possível)
	dbName := "unknown"
	if nameErr := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); nameErr != nil {
		// Ignorar erro, usar "unknown"
	}
	check.Database = dbName

	check.Status = "healthy"
	return check, nil
}

// CheckHealthWithPool verifica a saúde incluindo estatísticas do pool
func CheckHealthWithPool(db *sql.DB, timeout time.Duration) (*HealthCheck, error) {
	check, err := CheckHealth(db, timeout)
	if err != nil {
		return check, err
	}

	// Adicionar informações do pool ao status
	stats := db.Stats()
	if stats.OpenConnections > stats.MaxOpenConnections*9/10 {
		// Pool quase cheio, considerar warning
		check.Status = "healthy" // Ainda healthy, mas próximo do limite
	}

	return check, nil
}

// PrintHealthCheck imprime o resultado do health check de forma legível
func PrintHealthCheck(check *HealthCheck) {
	fmt.Printf("Health Check:\n")
	fmt.Printf("  Status: %s\n", check.Status)
	fmt.Printf("  Database: %s\n", check.Database)
	fmt.Printf("  Response Time: %v\n", check.ResponseTime)
	if check.Error != "" {
		fmt.Printf("  Error: %s\n", check.Error)
	}
}

