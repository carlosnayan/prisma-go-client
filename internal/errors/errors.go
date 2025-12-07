package errors

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ProductionMode indica se estamos em modo produção (oculta detalhes internos)
var ProductionMode = os.Getenv("ENV") == "production" || os.Getenv("ENV") == "prod"

// Tipos de erro específicos do prisma-go-client
var (
	// ErrNotFound é retornado quando um registro não é encontrado
	ErrNotFound = errors.New("record not found")

	// ErrValidation é retornado quando há erro de validação
	ErrValidation = errors.New("validation error")

	// ErrTooManyRows é retornado quando o resultado excede o limite de linhas
	ErrTooManyRows = errors.New("result set too large")

	// ErrInvalidInput é retornado quando a entrada é inválida
	ErrInvalidInput = errors.New("invalid input")

	// ErrPrimaryKeyRequired é retornado quando uma operação requer primary key mas não está definida
	ErrPrimaryKeyRequired = errors.New("primary key required")

	// ErrNoFieldsToUpdate é retornado quando não há campos para atualizar
	ErrNoFieldsToUpdate = errors.New("no fields to update")
)

// SanitizeError sanitiza uma mensagem de erro para não expor informações internas
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	if !ProductionMode {
		// Em desenvolvimento, retornar erro completo
		return err
	}

	errMsg := err.Error()

	// Remover nomes de tabelas e colunas
	errMsg = sanitizeTableNames(errMsg)
	errMsg = sanitizeColumnNames(errMsg)
	errMsg = sanitizeSQLDetails(errMsg)

	// Retornar erro sanitizado
	return fmt.Errorf("%s", errMsg)
}

// sanitizeTableNames remove nomes de tabelas das mensagens de erro
func sanitizeTableNames(msg string) string {
	// Padrões comuns que podem conter nomes de tabelas
	patterns := []string{
		"table",
		"relation",
		"FROM",
		"INTO",
		"UPDATE",
		"DELETE FROM",
	}

	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(pattern)) {
			return "database operation failed"
		}
	}

	return msg
}

// sanitizeColumnNames removes column names from error messages
func sanitizeColumnNames(msg string) string {
	patterns := []string{
		"column",
		"field",
		"SET",
		"WHERE",
	}

	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(pattern)) {
			return "database operation failed"
		}
	}

	return msg
}

// sanitizeSQLDetails removes SQL details from error messages
func sanitizeSQLDetails(msg string) string {
	if strings.Contains(strings.ToLower(msg), "sql") ||
		strings.Contains(strings.ToLower(msg), "syntax") ||
		strings.Contains(strings.ToLower(msg), "constraint") {
		return "database operation failed"
	}

	return msg
}

// WrapError wraps an error with a generic message in production
func WrapError(err error, genericMsg string) error {
	if err == nil {
		return nil
	}

	if ProductionMode {
		return fmt.Errorf("%s", genericMsg)
	}

	return fmt.Errorf("%s: %w", genericMsg, err)
}

// NewValidationError cria um novo erro de validação
func NewValidationError(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}

// NewNotFoundError cria um novo erro de não encontrado
func NewNotFoundError(resource string) error {
	if ProductionMode {
		return ErrNotFound
	}
	return fmt.Errorf("%w: %s", ErrNotFound, resource)
}

// IsNotFound verifica se o erro é do tipo NotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsValidation verifica se o erro é do tipo Validation
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}
