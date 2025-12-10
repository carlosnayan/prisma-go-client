package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type TableInfo struct {
	Name       string
	Columns    []ColumnInfo
	PrimaryKey string
	HasDeleted bool
}

type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
}

// GenerateQueries gera queries SQL para todas as tabelas a partir das migrations
func GenerateQueries(projectRoot string) error {
	migrationsDir := filepath.Join(projectRoot, "goose/migrations")
	queriesDir := filepath.Join(projectRoot, "sqlc/generated/queries")

	// Criar diretório queries se não existir
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório queries: %v", err)
	}

	// Ler todas as migrations
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
			// Se a tabela já existe, mesclar colunas (migrations podem adicionar colunas)
			if existing, ok := tables[table.Name]; ok {
				existing.Columns = mergeColumns(existing.Columns, table.Columns)
			} else {
				tables[table.Name] = table
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("erro ao ler migrations: %v", err)
	}

	// Gerar queries para cada tabela
	for tableName, table := range tables {
		if err := generateQueriesForTable(queriesDir, tableName, table); err != nil {
			return fmt.Errorf("erro ao gerar queries para %s: %v", tableName, err)
		}
	}

	fmt.Printf("✅ Queries geradas para %d tabelas\n", len(tables))
	return nil
}

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

		// Detectar CREATE TABLE
		createTableRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?["']?(\w+)["']?`)
		if matches := createTableRegex.FindStringSubmatch(trimmed); matches != nil {
			inCreateTable = true
			tableName = matches[1]
			tableContent.WriteString(line + "\n")
			continue
		}

		// Se estamos dentro de CREATE TABLE
		if inCreateTable {
			tableContent.WriteString(line + "\n")

			// Detectar fim da tabela
			if strings.HasPrefix(trimmed, ");") {
				break
			}
		}
	}

	if tableName == "" {
		return nil
	}

	// Parsear colunas do conteúdo completo
	columns, primaryKey, hasDeleted := parseTableColumns(tableContent.String())

	return &TableInfo{
		Name:       tableName,
		Columns:    columns,
		PrimaryKey: primaryKey,
		HasDeleted: hasDeleted,
	}
}

func parseTableColumns(statement string) ([]ColumnInfo, string, bool) {
	var columns []ColumnInfo
	var primaryKey string
	hasDeleted := false

	// Regex para capturar definições de coluna: "nome" TIPO [NOT NULL] [DEFAULT ...] [PRIMARY KEY]
	// Exemplo: "id_user" UUId PRIMARY KEY DEFAULT gen_random_uuid(),
	columnLineRegex := regexp.MustCompile(`^\s*["']?(\w+)["']?\s+(\w+(?:\([^)]+\))?)\s*([^,)]*)`)

	lines := strings.Split(statement, "\n")
	inTable := false
	parenDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detectar início da tabela
		if strings.Contains(trimmed, "CREATE TABLE") {
			inTable = true
			continue
		}

		if !inTable {
			continue
		}

		// Contar parênteses para detectar fim da tabela
		parenDepth += strings.Count(line, "(") - strings.Count(line, ")")
		if strings.HasPrefix(trimmed, ");") || (parenDepth < 0 && strings.Contains(trimmed, ")")) {
			break
		}

		// Pular linhas que não são definições de coluna
		if strings.HasPrefix(trimmed, "CONSTRAINT") ||
			strings.HasPrefix(trimmed, "--") ||
			trimmed == "" ||
			strings.HasPrefix(trimmed, "CREATE") {
			continue
		}

		// Remover vírgula final e limpar
		trimmed = strings.TrimSuffix(trimmed, ",")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			continue
		}

		// Tentar extrair coluna usando regex
		matches := columnLineRegex.FindStringSubmatch(trimmed)
		if len(matches) < 3 {
			continue
		}

		colName := strings.Trim(matches[1], `"'`)
		colType := matches[2]
		rest := matches[3]

		// Verificar se é uma coluna válida (não é palavra reservada)
		if isReservedWord(colName) {
			continue
		}

		// Verificar nullable
		nullable := !strings.Contains(rest, "NOT NULL")

		// Extrair default - capturar até encontrar vírgula ou fim de linha, mas respeitar parênteses
		var defaultValue string
		if strings.Contains(rest, "DEFAULT") {
			// Procurar por DEFAULT seguido do valor
			defaultPattern := regexp.MustCompile(`(?i)DEFAULT\s+([^,\n]+)`)
			if defaultMatch := defaultPattern.FindStringSubmatch(rest); defaultMatch != nil {
				defaultValue = strings.TrimSpace(defaultMatch[1])
			}
		}

		// Verificar se é primary key
		if strings.Contains(rest, "PRIMARY KEY") {
			primaryKey = colName
		}

		// Verificar se tem deleted_at
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

	// Ordenar por nome para consistência
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

func generateQueriesForTable(queriesDir, tableName string, table *TableInfo) error {
	fileName := fmt.Sprintf("%s/%s.sql", queriesDir, tableName)

	// Verificar se arquivo já existe (não sobrescrever queries customizadas)
	if _, err := os.Stat(fileName); err == nil {
		fmt.Printf("⚠️  Arquivo %s já existe, pulando...\n", fileName)
		return nil
	}

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo %s: %v", fileName, err)
	}
	defer file.Close()

	// Verificar quais colunas existem
	hasCreatedAt := false
	hasUpdatedAt := false
	for _, col := range table.Columns {
		if col.Name == "created_at" {
			hasCreatedAt = true
		}
		if col.Name == "updated_at" {
			hasUpdatedAt = true
		}
	}

	// Gerar queries padrão
	generateFindFirst(file, tableName, table, hasCreatedAt)
	generateFindMany(file, tableName, table, hasCreatedAt)
	generateCreate(file, tableName, table, hasCreatedAt, hasUpdatedAt)
	generateUpdate(file, tableName, table, hasUpdatedAt)
	generateDelete(file, tableName, table, hasUpdatedAt)

	fmt.Printf("✅ Gerado: %s\n", fileName)
	return nil
}

func generateFindFirst(w *os.File, tableName string, table *TableInfo, hasCreatedAt bool) {
	columns := getSelectColumns(table)

	fmt.Fprintf(w, "-- name: FindFirst%s :one\n", toPascalCase(tableName))
	fmt.Fprintf(w, "SELECT %s\n", columns)
	fmt.Fprintf(w, "FROM %s\n", tableName)
	fmt.Fprintf(w, "LIMIT 1;\n\n")
}

func generateFindMany(w *os.File, tableName string, table *TableInfo, hasCreatedAt bool) {
	columns := getSelectColumns(table)

	fmt.Fprintf(w, "-- name: FindMany%s :many\n", toPascalCase(tableName))
	fmt.Fprintf(w, "SELECT %s\n", columns)
	fmt.Fprintf(w, "FROM %s\n", tableName)
	fmt.Fprintf(w, "LIMIT $1 OFFSET $2;\n\n")
}

func generateCreate(w *os.File, tableName string, table *TableInfo, hasCreatedAt, hasUpdatedAt bool) {
	var insertColumns []string
	var values []string
	var returningColumns []string

	for _, col := range table.Columns {
		// Pular colunas com default ou que são geradas automaticamente
		// Primary key com gen_random_uuid ou SERIAL
		if col.Name == table.PrimaryKey {
			if strings.Contains(col.Default, "gen_random_uuid") ||
				strings.Contains(strings.ToUpper(col.Type), "SERIAL") {
				returningColumns = append(returningColumns, col.Name)
				continue // Não incluir no INSERT
			}
		}

		insertColumns = append(insertColumns, col.Name)
		values = append(values, fmt.Sprintf("$%d", len(insertColumns)))
		returningColumns = append(returningColumns, col.Name)
	}

	fmt.Fprintf(w, "-- name: Create%s :one\n", toPascalCase(tableName))
	fmt.Fprintf(w, "INSERT INTO %s (%s)\n", tableName, strings.Join(insertColumns, ", "))
	fmt.Fprintf(w, "VALUES (%s)\n", strings.Join(values, ", "))
	fmt.Fprintf(w, "RETURNING %s;\n\n", strings.Join(returningColumns, ", "))
}

func generateUpdate(w *os.File, tableName string, table *TableInfo, hasUpdatedAt bool) {
	var updateColumns []string
	returningMap := make(map[string]bool) // Para evitar duplicatas
	var returningColumns []string

	for _, col := range table.Columns {
		// Pular primary key
		if col.Name == table.PrimaryKey {
			continue
		}

		updateColumns = append(updateColumns, fmt.Sprintf("%s = $%d", col.Name, len(updateColumns)+1))
		if !returningMap[col.Name] {
			returningColumns = append(returningColumns, col.Name)
			returningMap[col.Name] = true
		}
	}

	// WHERE clause usando primary key
	whereParam := len(updateColumns) + 1

	fmt.Fprintf(w, "-- name: Update%s :one\n", toPascalCase(tableName))
	fmt.Fprintf(w, "UPDATE %s\n", tableName)
	fmt.Fprintf(w, "SET %s\n", strings.Join(updateColumns, ", "))
	fmt.Fprintf(w, "WHERE %s = $%d", table.PrimaryKey, whereParam)
	fmt.Fprintf(w, "\nRETURNING %s;\n\n", strings.Join(returningColumns, ", "))
}

func generateDelete(w *os.File, tableName string, table *TableInfo, hasUpdatedAt bool) {
	// Hard delete
	fmt.Fprintf(w, "-- name: Delete%s :exec\n", toPascalCase(tableName))
	fmt.Fprintf(w, "DELETE FROM %s\n", tableName)
	fmt.Fprintf(w, "WHERE %s = $1;\n\n", table.PrimaryKey)
}

func getSelectColumns(table *TableInfo) string {
	var columns []string
	for _, col := range table.Columns {
		columns = append(columns, col.Name)
	}
	return strings.Join(columns, ", ")
}
