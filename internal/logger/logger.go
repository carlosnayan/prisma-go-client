package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// LogLevel representa o nível de log
type LogLevel int

const (
	LogLevelQuery LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String retorna a representação em string do nível de log
func (l LogLevel) String() string {
	switch l {
	case LogLevelQuery:
		return "query"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	default:
		return "unknown"
	}
}

// Logger gerencia logs do Prisma
type Logger struct {
	levels   map[LogLevel]bool
	writer   io.Writer
	queryLog bool
}

var defaultLogger *Logger

func init() {
	defaultLogger = &Logger{
		levels:   make(map[LogLevel]bool),
		writer:   os.Stdout,
		queryLog: false,
	}
}

// NewLogger cria um novo logger
func NewLogger(levels []string, writer io.Writer) *Logger {
	logger := &Logger{
		levels: make(map[LogLevel]bool),
		writer: writer,
	}

	// Parsear níveis de log
	for _, level := range levels {
		level = strings.ToLower(strings.TrimSpace(level))
		switch level {
		case "query":
			logger.levels[LogLevelQuery] = true
			logger.queryLog = true
		case "info":
			logger.levels[LogLevelInfo] = true
		case "warn", "warning":
			logger.levels[LogLevelWarn] = true
		case "error":
			logger.levels[LogLevelError] = true
		}
	}

	return logger
}

// SetDefaultLogger define o logger padrão
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// GetDefaultLogger retorna o logger padrão
func GetDefaultLogger() *Logger {
	return defaultLogger
}

// Query loga uma query SQL
func (l *Logger) Query(query string, args []interface{}, duration time.Duration) {
	if !l.queryLog || !l.levels[LogLevelQuery] {
		return
	}

	// Formatar query com argumentos
	formattedQuery := formatQuery(query, args)
	
	// Logar com timestamp e duração
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(l.writer, "[%s] [QUERY] %s (took %v)\n", timestamp, formattedQuery, duration)
}

// Info loga uma mensagem informativa
func (l *Logger) Info(format string, args ...interface{}) {
	if !l.levels[LogLevelInfo] {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(l.writer, "[%s] [INFO] %s\n", timestamp, fmt.Sprintf(format, args...))
}

// Warn loga um aviso
func (l *Logger) Warn(format string, args ...interface{}) {
	if !l.levels[LogLevelWarn] {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(l.writer, "[%s] [WARN] %s\n", timestamp, fmt.Sprintf(format, args...))
}

// Error loga um erro
func (l *Logger) Error(format string, args ...interface{}) {
	if !l.levels[LogLevelError] {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(l.writer, "[%s] [ERROR] %s\n", timestamp, fmt.Sprintf(format, args...))
}

// formatQuery formata uma query SQL com seus argumentos
func formatQuery(query string, args []interface{}) string {
	if len(args) == 0 {
		return query
	}

	// Substituir placeholders por valores
	formatted := query
	argIndex := 0
	
	// Para PostgreSQL ($1, $2, ...)
	if strings.Contains(query, "$") {
		for i := 1; argIndex < len(args); i++ {
			placeholder := fmt.Sprintf("$%d", i)
			if strings.Contains(formatted, placeholder) {
				formatted = strings.Replace(formatted, placeholder, formatArg(args[argIndex]), 1)
				argIndex++
			}
		}
	} else {
		// Para MySQL/SQLite (?)
		for argIndex < len(args) {
			if strings.Contains(formatted, "?") {
				formatted = strings.Replace(formatted, "?", formatArg(args[argIndex]), 1)
				argIndex++
			} else {
				break
			}
		}
	}

	return formatted
}

// formatArg formata um argumento para exibição
// Sanitiza dados sensíveis para prevenir vazamento em logs
func formatArg(arg interface{}) string {
	switch v := arg.(type) {
	case string:
		// Sanitizar strings que podem conter senhas, tokens, etc.
		if isSensitiveData(v) {
			return "'***REDACTED***'"
		}
		// Limitar tamanho para logs
		if len(v) > 100 {
			return fmt.Sprintf("'%s...' (truncated)", v[:100])
		}
		return fmt.Sprintf("'%s'", v)
	case []byte:
		// Sempre redactar bytes (podem conter dados binários sensíveis)
		if len(v) > 0 {
			return "'***REDACTED***'"
		}
		return "''"
	case nil:
		return "NULL"
	default:
		// Para outros tipos, converter para string e verificar
		str := fmt.Sprintf("%v", v)
		if isSensitiveData(str) {
			return "***REDACTED***"
		}
		return str
	}
}

// isSensitiveData verifica se uma string pode conter dados sensíveis
func isSensitiveData(s string) bool {
	s = strings.ToLower(s)
	sensitiveKeywords := []string{
		"password", "passwd", "pwd",
		"secret", "token", "key",
		"api_key", "apikey", "access_token",
		"refresh_token", "auth", "authorization",
		"credential", "private", "private_key",
		"ssn", "social_security", "credit_card",
		"cvv", "pin", "otp",
	}
	
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	
	// Verificar padrões comuns de tokens (JWT, API keys, etc.)
	if len(s) > 20 && (strings.HasPrefix(s, "eyj") || // JWT
		strings.HasPrefix(s, "sk_") || // Stripe keys
		strings.HasPrefix(s, "pk_") ||
		strings.HasPrefix(s, "ghp_") || // GitHub tokens
		strings.HasPrefix(s, "xoxb-") || // Slack tokens
		strings.HasPrefix(s, "xoxp-")) {
		return true
	}
	
	return false
}

// Funções globais para facilitar uso
func Query(query string, args []interface{}, duration time.Duration) {
	defaultLogger.Query(query, args, duration)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// SetLogLevels configura os níveis de log do logger padrão
func SetLogLevels(levels []string) {
	defaultLogger = NewLogger(levels, os.Stdout)
}

// SetLogWriter configura o writer do logger padrão
func SetLogWriter(writer io.Writer) {
	if defaultLogger == nil {
		defaultLogger = NewLogger([]string{"info", "warn", "error"}, writer)
	} else {
		defaultLogger.writer = writer
	}
}

// FileLogger cria um logger que escreve em arquivo
func FileLogger(filename string, levels []string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo de log: %w", err)
	}
	return NewLogger(levels, file), nil
}

