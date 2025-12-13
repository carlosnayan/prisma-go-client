package errors

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
)

var ProductionMode = os.Getenv("ENV") == "production" || os.Getenv("ENV") == "prod"

type PrismaError struct {
	Code    string
	Message string
	cause   error
}

func (e *PrismaError) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *PrismaError) Unwrap() error {
	return e.cause
}

func (e *PrismaError) Is(target error) bool {
	if t, ok := target.(*PrismaError); ok {
		return e.Code == t.Code
	}
	return false
}

var (
	ErrNotFound             = &PrismaError{Code: "P2025", Message: "Record not found"}
	ErrUniqueConstraint     = &PrismaError{Code: "P2002", Message: "Unique constraint violation"}
	ErrForeignKeyConstraint = &PrismaError{Code: "P2003", Message: "Foreign key constraint violation"}
	ErrNullConstraint       = &PrismaError{Code: "P2011", Message: "Not null constraint violation"}
	ErrValueTooLong         = &PrismaError{Code: "P2000", Message: "Value too long for column"}
	ErrInvalidInput         = &PrismaError{Code: "P2007", Message: "Invalid input value"}
	ErrRawQueryFailed       = &PrismaError{Code: "P2010", Message: "Raw query failed"}
	ErrValueOutOfRange      = &PrismaError{Code: "P2020", Message: "Value out of range"}
	ErrTableNotFound        = &PrismaError{Code: "P2021", Message: "Table does not exist"}
	ErrColumnNotFound       = &PrismaError{Code: "P2022", Message: "Column not found"}
	ErrDeadlock             = &PrismaError{Code: "P2034", Message: "Transaction write conflict or deadlock"}

	ErrAuthenticationFailed = &PrismaError{Code: "P1000", Message: "Authentication failed"}
	ErrConnectionFailed     = &PrismaError{Code: "P1001", Message: "Database not reachable"}
	ErrDatabaseNotFound     = &PrismaError{Code: "P1003", Message: "Database does not exist"}
	ErrTimeout              = &PrismaError{Code: "P1008", Message: "Operation timeout"}
	ErrConnectionClosed     = &PrismaError{Code: "P1017", Message: "Connection closed"}
	ErrTooManyConnections   = &PrismaError{Code: "P2037", Message: "Too many connections"}

	ErrValidation         = &PrismaError{Code: "P2007", Message: "Validation error"}
	ErrTooManyRows        = &PrismaError{Code: "P2000", Message: "Result set too large"}
	ErrPrimaryKeyRequired = &PrismaError{Code: "P2007", Message: "Primary key required"}
	ErrNoFieldsToUpdate   = &PrismaError{Code: "P2007", Message: "No fields to update"}
)

type OperationType string

const (
	OpFindMany   OperationType = "FindMany"
	OpFindFirst  OperationType = "FindFirst"
	OpFindUnique OperationType = "FindUnique"
	OpQuery      OperationType = "Query"
	OpQueryRow   OperationType = "QueryRow"
	OpExec       OperationType = "Exec"
	OpCreate     OperationType = "Create"
	OpUpdate     OperationType = "Update"
	OpDelete     OperationType = "Delete"
)

func NewPrismaError(code, message string, cause error) *PrismaError {
	return &PrismaError{Code: code, Message: message, cause: cause}
}

func WrapPrismaError(sentinel *PrismaError, cause error) *PrismaError {
	return &PrismaError{Code: sentinel.Code, Message: sentinel.Message, cause: cause}
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsUniqueConstraint(err error) bool {
	return errors.Is(err, ErrUniqueConstraint)
}

func IsForeignKeyConstraint(err error) bool {
	return errors.Is(err, ErrForeignKeyConstraint)
}

func IsNullConstraint(err error) bool {
	return errors.Is(err, ErrNullConstraint)
}

func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

func IsConnectionFailed(err error) bool {
	return errors.Is(err, ErrConnectionFailed)
}

func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}

func isNoRows(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "no rows") ||
		strings.Contains(errStr, "ErrNoRows")
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "duplicate entry") ||
		strings.Contains(errStr, "unique_violation") ||
		strings.Contains(errStr, "23505") ||
		strings.Contains(errStr, "1062")
}

func isForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "foreign key constraint") ||
		strings.Contains(errStr, "foreign_key_violation") ||
		strings.Contains(errStr, "23503") ||
		strings.Contains(errStr, "1452")
}

func isNullViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not null constraint") ||
		strings.Contains(errStr, "null_violation") ||
		strings.Contains(errStr, "not-null constraint") ||
		strings.Contains(errStr, "23502") ||
		strings.Contains(errStr, "1048")
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "timed out") ||
		strings.Contains(errStr, "deadline exceeded")
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable")
}

func MapDriverError(err error, op OperationType) error {
	if err == nil {
		return nil
	}

	if isNoRows(err) {
		switch op {
		case OpFindMany, OpQuery:
			return nil
		default:
			return WrapPrismaError(ErrNotFound, err)
		}
	}

	if isUniqueViolation(err) {
		return WrapPrismaError(ErrUniqueConstraint, err)
	}

	if isForeignKeyViolation(err) {
		return WrapPrismaError(ErrForeignKeyConstraint, err)
	}

	if isNullViolation(err) {
		return WrapPrismaError(ErrNullConstraint, err)
	}

	if isTimeout(err) {
		return WrapPrismaError(ErrTimeout, err)
	}

	if isConnectionError(err) {
		return WrapPrismaError(ErrConnectionFailed, err)
	}

	return WrapPrismaError(ErrRawQueryFailed, err)
}

func SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	if !ProductionMode {
		return err
	}

	errMsg := err.Error()
	errMsg = sanitizeTableNames(errMsg)
	errMsg = sanitizeColumnNames(errMsg)
	errMsg = sanitizeSQLDetails(errMsg)

	return fmt.Errorf("%s", errMsg)
}

func sanitizeTableNames(msg string) string {
	patterns := []string{"table", "relation", "FROM", "INTO", "UPDATE", "DELETE FROM"}
	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(pattern)) {
			return "database operation failed"
		}
	}
	return msg
}

func sanitizeColumnNames(msg string) string {
	patterns := []string{"column", "field", "SET", "WHERE"}
	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(pattern)) {
			return "database operation failed"
		}
	}
	return msg
}

func sanitizeSQLDetails(msg string) string {
	if strings.Contains(strings.ToLower(msg), "sql") ||
		strings.Contains(strings.ToLower(msg), "syntax") ||
		strings.Contains(strings.ToLower(msg), "constraint") {
		return "database operation failed"
	}
	return msg
}

func WrapError(err error, genericMsg string) error {
	if err == nil {
		return nil
	}
	if ProductionMode {
		return fmt.Errorf("%s", genericMsg)
	}
	return fmt.Errorf("%s: %w", genericMsg, err)
}

func NewValidationError(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}

func NewNotFoundError(resource string) error {
	if ProductionMode {
		return ErrNotFound
	}
	return fmt.Errorf("%w: %s", ErrNotFound, resource)
}
