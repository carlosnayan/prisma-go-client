package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRawScan_DriverTemplatesHaveColumnsMethod(t *testing.T) {
	driverTests := []struct {
		name     string
		template string
		rowsType string
	}{
		{
			name:     "PostgreSQL PgxRows",
			template: "templates/driver/postgresql_driver.tmpl",
			rowsType: "PgxRows",
		},
	}

	for _, tt := range driverTests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(tt.template)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", tt.template, err)
			}

			templateStr := string(content)

			columnsMethodSignature := "func (r *" + tt.rowsType + ") Columns() ([]string, error)"
			if !strings.Contains(templateStr, columnsMethodSignature) {
				t.Errorf("%s must have Columns() method for automatic struct scanning. Expected: %s", tt.rowsType, columnsMethodSignature)
			}

			if !strings.Contains(templateStr, "FieldDescriptions()") {
				t.Errorf("%s.Columns() must use FieldDescriptions() to get column names", tt.rowsType)
			}
		})
	}
}

func TestRawScan_ScanResultTemplateHasRequiredFunctions(t *testing.T) {
	templatePath := filepath.Join("templates", "raw", "scan_result.tmpl")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("Failed to read scan_result.tmpl: %v", err)
	}

	templateStr := string(content)

	requiredFunctions := []struct {
		signature   string
		description string
	}{
		{"func scanRows(rows Rows, dest interface{}) error", "scanRows for automatic slice scanning"},
		{"func buildFieldMap(t reflect.Type) map[string]int", "buildFieldMap for db tag mapping"},
		{"func extractColumnName(col string) string", "extractColumnName for alias handling"},
		{"func scanRow(row Row, dest interface{}) error", "scanRow for single struct scanning"},
	}

	for _, fn := range requiredFunctions {
		if !strings.Contains(templateStr, fn.signature) {
			t.Errorf("Missing %s: %s", fn.description, fn.signature)
		}
	}

	if !strings.Contains(templateStr, `field.Tag.Get("db")`) {
		t.Error("buildFieldMap must support db struct tags")
	}

	if !strings.Contains(templateStr, "rows.Columns()") {
		t.Error("scanRows must call rows.Columns() to get column names")
	}

	if !strings.Contains(templateStr, `" as "`) || !strings.Contains(templateStr, "LastIndex") {
		t.Error("extractColumnName must handle SQL AS aliases")
	}
}

func TestRawScan_AdaptersHaveFallbackSupport(t *testing.T) {
	templatePath := filepath.Join("templates", "raw", "adapters.tmpl")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("Failed to read adapters.tmpl: %v", err)
	}

	templateStr := string(content)

	if !strings.Contains(templateStr, "func (r *rowsAdapter) Columns() ([]string, error)") {
		t.Error("rowsAdapter must have Columns() method")
	}

	if !strings.Contains(templateStr, "func (r *rowAdapter) Columns() ([]string, error)") {
		t.Error("rowAdapter must have Columns() method")
	}

	if !strings.Contains(templateStr, `r.rows.(interface{ Columns() ([]string, error) })`) {
		t.Error("rowsAdapter.Columns() must try standard Columns() interface first")
	}

	if !strings.Contains(templateStr, "FieldDescriptions") {
		t.Error("rowsAdapter.Columns() must have FieldDescriptions fallback for pgx")
	}

	if !strings.Contains(templateStr, "reflect.ValueOf") {
		t.Error("rowsAdapter.Columns() must have reflection fallback for unknown types")
	}
}

func TestRawScan_GeneratedCodeHasAllComponents(t *testing.T) {
	rawPath := filepath.Join("..", "..", "prisma", "db", "raw", "raw.go")

	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		t.Skip("Generated code not found, run 'prisma generate' first")
		return
	}

	content, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("Failed to read generated raw.go: %v", err)
	}

	generatedCode := string(content)

	requiredComponents := []struct {
		content     string
		description string
	}{
		{"Columns() ([]string, error)", "Rows interface must have Columns method"},
		{"func scanRows(rows Rows, dest interface{}) error", "scanRows function"},
		{"func buildFieldMap(t reflect.Type) map[string]int", "buildFieldMap function"},
		{`field.Tag.Get("db")`, "db tag support in buildFieldMap"},
		{"func (r *rowsAdapter) Columns() ([]string, error)", "rowsAdapter.Columns method"},
	}

	for _, c := range requiredComponents {
		if !strings.Contains(generatedCode, c.content) {
			t.Errorf("Generated code missing: %s", c.description)
		}
	}
}

func TestRawScan_GeneratedDriverHasColumnsMethod(t *testing.T) {
	driverPath := filepath.Join("..", "..", "prisma", "db", "driver.go")

	if _, err := os.Stat(driverPath); os.IsNotExist(err) {
		t.Skip("Generated driver.go not found, run 'prisma generate' first")
		return
	}

	content, err := os.ReadFile(driverPath)
	if err != nil {
		t.Fatalf("Failed to read generated driver.go: %v", err)
	}

	generatedCode := string(content)

	if strings.Contains(generatedCode, "type PgxRows struct") {
		if !strings.Contains(generatedCode, "func (r *PgxRows) Columns() ([]string, error)") {
			t.Error("PgxRows wrapper must have Columns() method for automatic struct scanning")
		}

		if !strings.Contains(generatedCode, "FieldDescriptions()") {
			t.Error("PgxRows.Columns() must delegate to rows.FieldDescriptions()")
		}
	}
}

func TestRawScan_InterfacesHaveColumnsMethod(t *testing.T) {
	templatePath := filepath.Join("templates", "shared", "interfaces.tmpl")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("Failed to read interfaces.tmpl: %v", err)
	}

	templateStr := string(content)

	if !strings.Contains(templateStr, "Columns() ([]string, error)") {
		t.Error("Rows interface must declare Columns() method")
	}

	rowsInterfaceIdx := strings.Index(templateStr, "type Rows interface")
	rowInterfaceIdx := strings.Index(templateStr, "type Row interface")

	if rowsInterfaceIdx == -1 {
		t.Error("Rows interface not found in interfaces.tmpl")
		return
	}

	if rowInterfaceIdx == -1 {
		t.Error("Row interface not found in interfaces.tmpl")
		return
	}

	rowsSection := templateStr[rowsInterfaceIdx:rowInterfaceIdx]
	if !strings.Contains(rowsSection, "Columns() ([]string, error)") {
		t.Error("Rows interface must include Columns() method")
	}
}
