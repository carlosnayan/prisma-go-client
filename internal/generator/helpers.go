package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateHelpers generates type-safe helper functions in the db package
func GenerateHelpers(schema *parser.Schema, outputDir string) error {
	// Detect user module
	userModule, err := detectUserModule(outputDir)
	if err != nil {
		return fmt.Errorf("failed to detect user module: %w", err)
	}

	helpersFile := filepath.Join(outputDir, "helpers.go")
	return generateHelpersFile(helpersFile, userModule, outputDir, schema)
}

// generateHelpersFile generates the file with type-safe helper functions
func generateHelpersFile(filePath string, userModule, outputDir string, schema *parser.Schema) error {
	file, err := createGeneratedFile(filePath, "db")
	if err != nil {
		return err
	}
	defer file.Close()

	// Calculate import path for inputs
	_, _, inputsPath, err := calculateImportPath(userModule, outputDir)
	if err != nil {
		// Fallback
		inputsPath = "github.com/carlosnayan/prisma-go-client/db/inputs"
	}

	// Determine which filter types are needed based on schema
	neededFilters := determineNeededFilters(schema)

	// Build imports list based on needed filters
	imports := make([]string, 0, 3)
	if neededFilters["DateTimeFilter"] {
		imports = append(imports, "time")
	}
	if neededFilters["JsonFilter"] {
		imports = append(imports, "encoding/json")
	}

	if len(imports) > 0 || inputsPath != "" {
		fmt.Fprintf(file, "import (\n")
		for _, imp := range imports {
			fmt.Fprintf(file, "\t%q\n", imp)
		}
		if inputsPath != "" {
			fmt.Fprintf(file, "\tinputs %q\n", inputsPath)
		}
		fmt.Fprintf(file, ")\n\n")
	}

	// Generate helpers only for needed filter types
	if neededFilters["StringFilter"] {
		generateStringHelpers(file)
	}

	if neededFilters["IntFilter"] {
		generateIntHelpers(file)
	}

	if neededFilters["Int64Filter"] {
		generateInt64Helpers(file)
	}

	if neededFilters["FloatFilter"] {
		generateFloatHelpers(file)
	}

	if neededFilters["BooleanFilter"] {
		generateBooleanHelpers(file)
	}

	if neededFilters["DateTimeFilter"] {
		generateDateTimeHelpers(file)
	}

	if neededFilters["JsonFilter"] {
		generateJsonHelpers(file)
	}

	if neededFilters["BytesFilter"] {
		generateBytesHelpers(file)
	}

	return nil
}

// determineNeededFilters determines which filter types are needed based on the schema
func determineNeededFilters(schema *parser.Schema) map[string]bool {
	neededFilters := make(map[string]bool)

	// Check all models for field types to determine which filters are actually needed
	for _, model := range schema.Models {
		for _, field := range model.Fields {
			if field.Type == nil {
				continue
			}

			// Check if it's a relation using the same logic as inputs.go
			if isRelationForHelpers(field, schema) {
				continue
			}

			filterType := getFilterTypeForHelpers(field.Type)
			neededFilters[filterType] = true
		}
	}

	return neededFilters
}

// getFilterTypeForHelpers returns the appropriate Filter type for a field type
// Uses the shared getFilterType function from inputs.go
func getFilterTypeForHelpers(fieldType *parser.FieldType) string {
	return getFilterType(fieldType)
}

// isRelationForHelpers checks if a field is a relationship (non-primitive type)
// Uses the shared isRelation function from inputs.go
// Note: This function requires schema to be passed, but for backward compatibility
// it accepts nil schema (will treat enums as relations in that case)
func isRelationForHelpers(field *parser.ModelField, schema *parser.Schema) bool {
	return isRelation(field, schema)
}

func generateStringHelpers(file *os.File) {
	fmt.Fprintf(file, "// String helper functions for StringFilter\n\n")
	fmt.Fprintf(file, "func Contains(value string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{Contains: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func StartsWith(value string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{StartsWith: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func EndsWith(value string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{EndsWith: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func ContainsInsensitive(value string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{ContainsInsensitive: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func StartsWithInsensitive(value string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{StartsWithInsensitive: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func EndsWithInsensitive(value string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{EndsWithInsensitive: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func String(value string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{Equals: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func StringIn(values ...string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{In: values}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func StringNotIn(values ...string) *inputs.StringFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.StringFilter{NotIn: values}\n")
	fmt.Fprintf(file, "}\n\n")
}

func generateIntHelpers(file *os.File) {
	fmt.Fprintf(file, "// Int helper functions for IntFilter\n\n")
	fmt.Fprintf(file, "func Int(value int) *inputs.IntFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.IntFilter{Equals: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func IntGt(value int) *inputs.IntFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.IntFilter{Gt: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func IntGte(value int) *inputs.IntFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.IntFilter{Gte: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func IntLt(value int) *inputs.IntFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.IntFilter{Lt: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func IntLte(value int) *inputs.IntFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.IntFilter{Lte: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func IntIn(values ...int) *inputs.IntFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.IntFilter{In: values}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func IntNotIn(values ...int) *inputs.IntFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.IntFilter{NotIn: values}\n")
	fmt.Fprintf(file, "}\n\n")
}

func generateInt64Helpers(file *os.File) {
	fmt.Fprintf(file, "// Int64 helper functions for Int64Filter\n\n")
	fmt.Fprintf(file, "func Int64(value int64) *inputs.Int64Filter {\n")
	fmt.Fprintf(file, "\treturn &inputs.Int64Filter{Equals: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func Int64Gt(value int64) *inputs.Int64Filter {\n")
	fmt.Fprintf(file, "\treturn &inputs.Int64Filter{Gt: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func Int64Gte(value int64) *inputs.Int64Filter {\n")
	fmt.Fprintf(file, "\treturn &inputs.Int64Filter{Gte: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func Int64Lt(value int64) *inputs.Int64Filter {\n")
	fmt.Fprintf(file, "\treturn &inputs.Int64Filter{Lt: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func Int64Lte(value int64) *inputs.Int64Filter {\n")
	fmt.Fprintf(file, "\treturn &inputs.Int64Filter{Lte: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func Int64In(values ...int64) *inputs.Int64Filter {\n")
	fmt.Fprintf(file, "\treturn &inputs.Int64Filter{In: values}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func Int64NotIn(values ...int64) *inputs.Int64Filter {\n")
	fmt.Fprintf(file, "\treturn &inputs.Int64Filter{NotIn: values}\n")
	fmt.Fprintf(file, "}\n\n")
}

func generateFloatHelpers(file *os.File) {
	fmt.Fprintf(file, "// Float helper functions for FloatFilter\n\n")
	fmt.Fprintf(file, "func Float(value float64) *inputs.FloatFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.FloatFilter{Equals: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func FloatGt(value float64) *inputs.FloatFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.FloatFilter{Gt: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func FloatGte(value float64) *inputs.FloatFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.FloatFilter{Gte: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func FloatLt(value float64) *inputs.FloatFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.FloatFilter{Lt: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func FloatLte(value float64) *inputs.FloatFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.FloatFilter{Lte: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func FloatIn(values ...float64) *inputs.FloatFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.FloatFilter{In: values}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func FloatNotIn(values ...float64) *inputs.FloatFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.FloatFilter{NotIn: values}\n")
	fmt.Fprintf(file, "}\n\n")
}

func generateBooleanHelpers(file *os.File) {
	fmt.Fprintf(file, "// Boolean helper functions for BooleanFilter\n\n")
	fmt.Fprintf(file, "func Bool(value bool) *inputs.BooleanFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.BooleanFilter{Equals: &value}\n")
	fmt.Fprintf(file, "}\n\n")
}

func generateDateTimeHelpers(file *os.File) {
	fmt.Fprintf(file, "// DateTime helper functions for DateTimeFilter\n\n")
	fmt.Fprintf(file, "func DateTime(value time.Time) *inputs.DateTimeFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.DateTimeFilter{Equals: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func DateTimeGt(value time.Time) *inputs.DateTimeFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.DateTimeFilter{Gt: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func DateTimeGte(value time.Time) *inputs.DateTimeFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.DateTimeFilter{Gte: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func DateTimeLt(value time.Time) *inputs.DateTimeFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.DateTimeFilter{Lt: &value}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "func DateTimeLte(value time.Time) *inputs.DateTimeFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.DateTimeFilter{Lte: &value}\n")
	fmt.Fprintf(file, "}\n\n")
}

func generateJsonHelpers(file *os.File) {
	fmt.Fprintf(file, "// Json helper functions for JsonFilter\n\n")
	fmt.Fprintf(file, "func Json(value json.RawMessage) *inputs.JsonFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.JsonFilter{Equals: &value}\n")
	fmt.Fprintf(file, "}\n\n")
}

func generateBytesHelpers(file *os.File) {
	fmt.Fprintf(file, "// Bytes helper functions for BytesFilter\n\n")
	fmt.Fprintf(file, "func Bytes(value []byte) *inputs.BytesFilter {\n")
	fmt.Fprintf(file, "\treturn &inputs.BytesFilter{Equals: &value}\n")
	fmt.Fprintf(file, "}\n\n")
}
