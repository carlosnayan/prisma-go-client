//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/carlosnayan/prisma-go-client/generator"
)

func main() {
	fmt.Println("🚀 Iniciando geração de código...")
	fmt.Println()

	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Printf("❌ Erro ao encontrar diretório raiz do projeto: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📁 Diretório do projeto: %s\n", projectRoot)
	fmt.Println()

	fmt.Println("🔧 Passo 1: Gerando core (db.go)...")
	if err := generator.GenerateCore(projectRoot); err != nil {
		fmt.Printf("❌ Erro ao gerar core: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Core gerado")
	fmt.Println()

	fmt.Println("📦 Passo 2: Gerando models...")
	if err := generateModels(projectRoot); err != nil {
		fmt.Printf("❌ Erro ao gerar models: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Models gerados")
	fmt.Println()

	fmt.Println("🔧 Passo 3: Gerando wrappers...")
	if err := generator.GenerateWrappers(projectRoot); err != nil {
		fmt.Printf("❌ Erro ao gerar wrappers: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Wrappers gerados")
	fmt.Println()

	fmt.Println("🔧 Passo 4: Gerando tipos de input...")
	if err := generator.GenerateTypes(projectRoot); err != nil {
		fmt.Printf("❌ Erro ao gerar tipos: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Tipos gerados")
	fmt.Println()

	fmt.Println("🔧 Passo 5: Gerando src/db/db.go...")
	if err := generator.GenerateDB(projectRoot); err != nil {
		fmt.Printf("❌ Erro ao gerar src/db/db.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ src/db/db.go gerado")
	fmt.Println()

	fmt.Println("🎉 Geração de código concluída com sucesso!")
}

// findProjectRoot finds the project root directory by searching for sqlc.yaml
func findProjectRoot() (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}

	for {
		sqlcYaml := filepath.Join(dir, "sqlc.yaml")
		if _, err := os.Stat(sqlcYaml); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("não foi possível encontrar sqlc.yaml")
		}
		dir = parent
	}
}

// generateModels generates models in sqlc/generated/models from tables in migrations
func generateModels(projectRoot string) error {
	migrationsDir := filepath.Join(projectRoot, "goose/migrations")
	modelsDir := filepath.Join(projectRoot, "sqlc/generated/models")

	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório models: %v", err)
	}

	tables, err := parseMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("erro ao parsear migrations: %v", err)
	}

	for tableName, tableInfo := range tables {
		modelFile := filepath.Join(modelsDir, toSnakeCase(tableName)+".go")

		if _, err := os.Stat(modelFile); err == nil {
			fmt.Printf("⚠️  Arquivo %s já existe, pulando...\n", modelFile)
			continue
		}

		if err := generateModelFile(modelFile, tableName, tableInfo); err != nil {
			return fmt.Errorf("erro ao gerar model para %s: %v", tableName, err)
		}
		fmt.Printf("✅ Gerado: %s\n", modelFile)
	}

	return nil
}

// TableInfo contains information about a table
type TableInfo struct {
	Name       string
	Columns    []ColumnInfo
	PrimaryKey string
	HasDeleted bool
}

// ColumnInfo contains information about a column
type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
}

// parseMigrations reads all migrations and extracts table information
func parseMigrations(migrationsDir string) (map[string]*TableInfo, error) {
	tables := make(map[string]*TableInfo)

	err := filepath.Walk(migrationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".sql") {
			return nil
		}

		table := parseMigration(path)
		if table != nil {
			if existing, ok := tables[table.Name]; ok {
				existing.Columns = mergeColumns(existing.Columns, table.Columns)
			} else {
				tables[table.Name] = table
			}
		}

		return nil
	})

	return tables, err
}

// parseMigration parses a migration and returns table information
func parseMigration(filePath string) *TableInfo {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var inCreateTable bool
	var tableName string
	var tableContent strings.Builder
	var primaryKey string
	hasDeleted := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		createTableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?["']?(\w+)["']?`)
		if matches := createTableRegex.FindStringSubmatch(trimmed); matches != nil {
			inCreateTable = true
			tableName = matches[1]
			tableContent.WriteString(line + "\n")
			continue
		}

		if inCreateTable {
			tableContent.WriteString(line + "\n")

			if strings.HasPrefix(trimmed, ");") {
				inCreateTable = false
				break
			}
		}
	}

	if tableName == "" {
		return nil
	}

	columns, primaryKey, hasDeleted := parseTableColumns(tableContent.String(), primaryKey, hasDeleted)

	return &TableInfo{
		Name:       tableName,
		Columns:    columns,
		PrimaryKey: primaryKey,
		HasDeleted: hasDeleted,
	}
}

// parseTableColumns parses columns from a table definition
func parseTableColumns(statement string, existingPK string, existingHasDeleted bool) ([]ColumnInfo, string, bool) {
	var columns []ColumnInfo
	primaryKey := existingPK
	hasDeleted := existingHasDeleted

	columnLineRegex := regexp.MustCompile(`^\s*["']?(\w+)["']?\s+(\w+(?:\([^)]+\))?)\s*([^,)]*)`)

	lines := strings.Split(statement, "\n")
	inTable := false
	parenDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "CREATE TABLE") {
			inTable = true
			continue
		}

		if !inTable {
			continue
		}

		parenDepth += strings.Count(line, "(") - strings.Count(line, ")")
		if strings.HasPrefix(trimmed, ");") || (parenDepth < 0 && strings.Contains(trimmed, ")")) {
			break
		}

		if strings.HasPrefix(trimmed, "CONSTRAINT") ||
			strings.HasPrefix(trimmed, "--") ||
			trimmed == "" ||
			strings.HasPrefix(trimmed, "CREATE") {
			continue
		}

		trimmed = strings.TrimSuffix(trimmed, ",")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			continue
		}

		matches := columnLineRegex.FindStringSubmatch(trimmed)
		if len(matches) < 3 {
			continue
		}

		colName := strings.Trim(matches[1], `"'`)
		colType := matches[2]
		rest := matches[3]

		if isReservedWord(colName) {
			continue
		}

		nullable := !strings.Contains(rest, "NOT NULL")

		var defaultValue string
		if strings.Contains(rest, "DEFAULT") {
			defaultPattern := regexp.MustCompile(`(?i)DEFAULT\s+([^,\n]+)`)
			if defaultMatch := defaultPattern.FindStringSubmatch(rest); defaultMatch != nil {
				defaultValue = strings.TrimSpace(defaultMatch[1])
			}
		}

		if strings.Contains(rest, "PRIMARY KEY") {
			primaryKey = colName
		}

		if colName == "deleted_at" {
			hasDeleted = true
		}

		columns = append(columns, ColumnInfo{
			Name:     colName,
			Type:     colType,
			Nullable: nullable,
			Default:  defaultValue,
		})
	}

	return columns, primaryKey, hasDeleted
}

// isReservedWord checks if a word is reserved
func isReservedWord(word string) bool {
	reserved := []string{"CREATE", "TABLE", "IF", "NOT", "EXISTS", "PRIMARY", "KEY", "FOREIGN", "CONSTRAINT", "CHECK", "DEFAULT", "NULL", "REFERENCES", "ON", "DELETE", "CASCADE", "UPDATE", "NO", "ACTION", "INDEX", "UNIQUE", "WHERE", "IS", "AND", "OR"}
	upper := strings.ToUpper(word)
	for _, r := range reserved {
		if upper == r {
			return true
		}
	}
	return false
}

// mergeColumns merges columns from different migrations
func mergeColumns(existing, new []ColumnInfo) []ColumnInfo {
	columnMap := make(map[string]ColumnInfo)
	for _, col := range existing {
		columnMap[col.Name] = col
	}
	for _, col := range new {
		columnMap[col.Name] = col
	}

	var result []ColumnInfo
	for _, col := range columnMap {
		result = append(result, col)
	}

	return result
}

// generateModelFile generates the model file for a table
func generateModelFile(filePath, tableName string, tableInfo *TableInfo) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	structName := toPascalCase(tableName)

	fmt.Fprintf(file, "package models\n\n")

	hasTime := false
	hasJSON := false
	for _, col := range tableInfo.Columns {
		if strings.Contains(strings.ToUpper(col.Type), "TIMESTAMP") ||
			strings.Contains(strings.ToUpper(col.Type), "DATE") {
			hasTime = true
		}
		if strings.Contains(strings.ToUpper(col.Type), "JSON") {
			hasJSON = true
		}
	}

	if hasTime || hasJSON {
		fmt.Fprintf(file, "import (\n")
		if hasTime {
			fmt.Fprintf(file, "\t\"time\"\n")
		}
		if hasJSON {
			fmt.Fprintf(file, "\t\"encoding/json\"\n")
		}
		fmt.Fprintf(file, ")\n\n")
	}

	fmt.Fprintf(file, "type %s struct {\n", structName)

	for _, col := range tableInfo.Columns {
		fieldName := toPascalCase(col.Name)
		goType := sqlTypeToGoType(col.Type, col.Nullable)
		jsonTag := toSnakeCase(col.Name)

		tags := fmt.Sprintf("`json:\"%s\"`", jsonTag)

		fmt.Fprintf(file, "\t%s %s %s\n", fieldName, goType, tags)
	}

	fmt.Fprintf(file, "}\n")

	return nil
}

// sqlTypeToGoType converts a SQL type to a Go type
func sqlTypeToGoType(sqlType string, nullable bool) string {
	sqlTypeUpper := strings.ToUpper(sqlType)

	if idx := strings.Index(sqlTypeUpper, "("); idx != -1 {
		sqlTypeUpper = sqlTypeUpper[:idx]
	}

	var goType string
	switch sqlTypeUpper {
	case "UUId":
		goType = "string"
	case "TEXT", "VARCHAR", "CHAR":
		goType = "string"
	case "INTEGER", "INT", "BIGINT", "SMALLINT":
		goType = "int"
	case "BOOLEAN", "BOOL":
		goType = "bool"
	case "TIMESTAMP", "TIMESTAMPTZ", "DATE":
		goType = "time.Time"
	case "JSONB", "JSON":
		goType = "json.RawMessage"
	case "DECIMAL", "NUMERIC", "FLOAT", "DOUBLE", "REAL":
		goType = "float64"
	default:
		goType = "string"
	}

	if nullable && goType != "json.RawMessage" {
		return "*" + goType
	}

	return goType
}

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
		}
	}
	return result.String()
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
