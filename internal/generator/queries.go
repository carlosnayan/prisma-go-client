package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateQueries generates type-safe query builders for each model
func GenerateQueries(schema *parser.Schema, outputDir string) error {
	queriesDir := filepath.Join(outputDir, "queries")
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		return fmt.Errorf("failed to create queries directory: %w", err)
	}

	// Detect user module
	userModule, err := detectUserModule(outputDir)
	if err != nil {
		return fmt.Errorf("failed to detect user module: %w", err)
	}

	queryResultFile := filepath.Join(queriesDir, "query_result.go")
	if err := generateQueryResultFile(queryResultFile, userModule, outputDir); err != nil {
		return fmt.Errorf("failed to generate query_result.go: %w", err)
	}

	for _, model := range schema.Models {
		queryFile := filepath.Join(queriesDir, toSnakeCase(model.Name)+"_query.go")
		if err := generateQueryFile(queryFile, model, schema, userModule, outputDir); err != nil {
			return fmt.Errorf("failed to generate query for %s: %w", model.Name, err)
		}
	}

	return nil
}

// generateQueryResultFile generates the file with QueryResult type
func generateQueryResultFile(filePath string, userModule, outputDir string) error {
	file, err := createGeneratedFile(filePath, "queries")
	if err != nil {
		return err
	}
	defer file.Close()

	// Calculate local import path for builder
	builderPath, _, err := calculateLocalImportPath(userModule, outputDir)
	if err != nil {
		builderPath = "github.com/carlosnayan/prisma-go-client/db/builder"
	}

	fmt.Fprintf(file, "import (\n")
	fmt.Fprintf(file, "\t\"context\"\n")
	fmt.Fprintf(file, "\t%q\n", builderPath)
	fmt.Fprintf(file, ")\n\n")

	fmt.Fprintf(file, "// QueryResult represents a query that can be executed\n")
	fmt.Fprintf(file, "type QueryResult[T any] struct {\n")
	fmt.Fprintf(file, "\tquery *builder.Query\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes the query and returns results\n")
	fmt.Fprintf(file, "func (r *QueryResult[T]) Exec(ctx context.Context) ([]T, error) {\n")
	fmt.Fprintf(file, "\tvar results []T\n")
	fmt.Fprintf(file, "\terr := r.query.Find(ctx, &results)\n")
	fmt.Fprintf(file, "\treturn results, err\n")
	fmt.Fprintf(file, "}\n\n")

	return nil
}

// generateQueryFile generates the query builder file for a model
func generateQueryFile(filePath string, model *parser.Model, schema *parser.Schema, userModule, outputDir string) error {
	file, err := createGeneratedFile(filePath, "queries")
	if err != nil {
		return err
	}
	defer file.Close()

	// Determine required imports
	imports := determineQueryImports(userModule, outputDir)
	if len(imports) > 0 {
		// Separate stdlib and third-party imports
		stdlib := make([]string, 0, len(imports))
		thirdParty := make([]string, 0, len(imports))

		for _, imp := range imports {
			if isStdlibImport(imp) {
				stdlib = append(stdlib, imp)
			} else {
				thirdParty = append(thirdParty, imp)
			}
		}

		writeImportsWithGroups(file, stdlib, thirdParty)
		closeImports(file)
	}

	// Query struct
	pascalModelName := toPascalCase(model.Name)
	fmt.Fprintf(file, "// %sQuery is the query builder for model %s with fluent API\n", pascalModelName, model.Name)
	fmt.Fprintf(file, "// All methods from builder.Query are available via embedding\n")
	fmt.Fprintf(file, "// Example: q.Where(\"email = ?\", \"user@example.com\").First(ctx, &user)\n")
	fmt.Fprintf(file, "type %sQuery struct {\n", pascalModelName)
	fmt.Fprintf(file, "\t*builder.Query\n")
	fmt.Fprintf(file, "}\n\n")

	// Query builder methods
	generateQueryMethods(file, model, schema)

	return nil
}

// generateQueryMethods generates the query builder methods with fluent API
func generateQueryMethods(file *os.File, model *parser.Model, schema *parser.Schema) {
	pascalModelName := toPascalCase(model.Name)

	// First
	fmt.Fprintf(file, "// First finds the first record\n")
	fmt.Fprintf(file, "// Examples:\n")
	fmt.Fprintf(file, "//   q.Where(\"email = ?\", \"user@example.com\").First(ctx, &user)\n")
	fmt.Fprintf(file, "//   q.Where(builder.Where{\"id\": id}).First(ctx, &user)\n")
	fmt.Fprintf(file, "func (q *%sQuery) First(ctx context.Context, dest *models.%s) error {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn q.Query.First(ctx, dest)\n")
	fmt.Fprintf(file, "}\n\n")

	// FindFirst - removed to avoid conflict with Prisma-style FindFirst() builder
	// Use First() for direct execution or FindFirst() for builder pattern

	// Find
	fmt.Fprintf(file, "// Find finds all records\n")
	fmt.Fprintf(file, "// Examples:\n")
	fmt.Fprintf(file, "//   q.Where(\"active = ?\", true).Order(\"created_at DESC\").Take(10).Find(ctx, &users)\n")
	fmt.Fprintf(file, "//   q.Where(builder.Where{\"active\": true}).Find(ctx, &users)\n")
	fmt.Fprintf(file, "func (q *%sQuery) Find(ctx context.Context, dest *[]models.%s) error {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn q.Query.Find(ctx, dest)\n")
	fmt.Fprintf(file, "}\n\n")

	// FindMany - removed to avoid conflict with Prisma-style FindMany() builder
	// Use FindMany() builder pattern instead: q.FindMany().Where(...).Exec(ctx)

	generateWhereInputConverter(file, model, schema)
	generateApplyWhereInputHelper(file, model, schema)
	// Count - removed to avoid conflict with Prisma-style Count() builder
	// Use Count() builder pattern instead

	// Create - removed to avoid conflict with Prisma-style Create() builder
	// Use Create() builder pattern instead

	fmt.Fprintf(file, "// Save saves a record (create or update)\n")
	fmt.Fprintf(file, "// Example: q.Save(ctx, &user)\n")
	fmt.Fprintf(file, "func (q *%sQuery) Save(ctx context.Context, value *models.%s) error {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn q.Query.Save(ctx, value)\n")
	fmt.Fprintf(file, "}\n\n")

	// Update - removed to avoid conflict with Prisma-style Update() builder
	// Use Updates() for direct updates or Update() builder pattern

	fmt.Fprintf(file, "// Updates updates multiple columns\n")
	fmt.Fprintf(file, "// Example: q.Where(\"id = ?\", 1).Updates(ctx, map[string]interface{}{\"name\": \"New\", \"age\": 30})\n")
	fmt.Fprintf(file, "func (q *%sQuery) Updates(ctx context.Context, values map[string]interface{}) error {\n", pascalModelName)
	fmt.Fprintf(file, "\treturn q.Query.Updates(ctx, values)\n")
	fmt.Fprintf(file, "}\n\n")

	// Delete - removed to avoid conflict with Prisma-style Delete() builder
	// Use Delete() builder pattern instead

	// Generate Prisma-style builder methods
	generatePrismaBuilders(file, model, schema)
}

// generateWhereInputConverter generates function to convert WhereInput to builder.Where
func generateWhereInputConverter(file *os.File, model *parser.Model, schema *parser.Schema) {
	pascalModelName := toPascalCase(model.Name)

	fmt.Fprintf(file, "// Convert%sWhereInputToWhere converts WhereInput to builder.Where\n", pascalModelName)
	fmt.Fprintf(file, "func Convert%sWhereInputToWhere(where inputs.%sWhereInput) builder.Where {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tresult := builder.Where{}\n\n")

	for _, field := range model.Fields {
		if isRelation(field, schema) {
			continue
		}
		fieldName := toPascalCase(field.Name)
		// Get the actual database column name (check for @map attribute)
		dbFieldName := field.Name
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					dbFieldName = val
					break
				}
			}
		}
		filterType := getFilterType(field.Type)

		fmt.Fprintf(file, "\tif where.%s != nil {\n", fieldName)
		fmt.Fprintf(file, "\t\tfilter := where.%s\n", fieldName)
		generateFilterConverter(file, filterType, dbFieldName)
		fmt.Fprintf(file, "\t}\n\n")
	}

	fmt.Fprintf(file, "\t// Handle OR conditions\n")
	fmt.Fprintf(file, "\tif len(where.Or) > 0 {\n")
	fmt.Fprintf(file, "\t\torConditions := []builder.Where{}\n")
	fmt.Fprintf(file, "\t\tfor _, orWhere := range where.Or {\n")
	fmt.Fprintf(file, "\t\t\torConditions = append(orConditions, Convert%sWhereInputToWhere(orWhere))\n", pascalModelName)
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t\tfor _, orCond := range orConditions {\n")
	fmt.Fprintf(file, "\t\t\tfor k, v := range orCond {\n")
	fmt.Fprintf(file, "\t\t\t\tresult[k] = v\n")
	fmt.Fprintf(file, "\t\t\t}\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n\n")

	fmt.Fprintf(file, "\t// Handle AND conditions\n")
	fmt.Fprintf(file, "\tif len(where.And) > 0 {\n")
	fmt.Fprintf(file, "\t\tfor _, andWhere := range where.And {\n")
	fmt.Fprintf(file, "\t\t\tandMap := Convert%sWhereInputToWhere(andWhere)\n", pascalModelName)
	fmt.Fprintf(file, "\t\t\tfor k, v := range andMap {\n")
	fmt.Fprintf(file, "\t\t\t\tresult[k] = v\n")
	fmt.Fprintf(file, "\t\t\t}\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n\n")

	fmt.Fprintf(file, "\t// Handle NOT condition\n")
	fmt.Fprintf(file, "\tif where.Not != nil {\n")
	fmt.Fprintf(file, "\t\tnotMap := Convert%sWhereInputToWhere(*where.Not)\n", pascalModelName)
	fmt.Fprintf(file, "\t\t// For now, combine with AND\n")
	fmt.Fprintf(file, "\t\tfor k, v := range notMap {\n")
	fmt.Fprintf(file, "\t\t\tresult[k] = v\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n\n")

	fmt.Fprintf(file, "\treturn result\n")
	fmt.Fprintf(file, "}\n\n")
}

// generateApplyWhereInputHelper generates a helper function to apply WhereInput to a query
// This handles OR conditions correctly by using the Or() method
func generateApplyWhereInputHelper(file *os.File, model *parser.Model, schema *parser.Schema) {
	pascalModelName := toPascalCase(model.Name)

	fmt.Fprintf(file, "// apply%sWhereInput applies WhereInput to a query builder, handling OR conditions correctly\n", pascalModelName)
	fmt.Fprintf(file, "func apply%sWhereInput(query *builder.Query, where inputs.%sWhereInput) {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\t// Handle OR conditions first\n")
	fmt.Fprintf(file, "\tif len(where.Or) > 0 {\n")
	fmt.Fprintf(file, "\t\t// Apply first OR condition as regular WHERE\n")
	fmt.Fprintf(file, "\t\tfirstOrMap := Convert%sWhereInputToWhere(where.Or[0])\n", pascalModelName)
	fmt.Fprintf(file, "\t\tquery.Where(firstOrMap)\n")
	fmt.Fprintf(file, "\t\t// Apply remaining OR conditions using Or()\n")
	fmt.Fprintf(file, "\t\tfor i := 1; i < len(where.Or); i++ {\n")
	fmt.Fprintf(file, "\t\t\torMap := Convert%sWhereInputToWhere(where.Or[i])\n", pascalModelName)
	fmt.Fprintf(file, "\t\t\t// Build OR conditions from the map\n")
	fmt.Fprintf(file, "\t\t\tfor field, value := range orMap {\n")
	fmt.Fprintf(file, "\t\t\t\tif op, ok := value.(builder.WhereOperator); ok {\n")
	fmt.Fprintf(file, "\t\t\t\t\tquotedField := query.GetDialect().QuoteIdentifier(field)\n")
	fmt.Fprintf(file, "\t\t\t\t\tswitch op.GetOp() {\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \">\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s > ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \">=\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s >= ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"<\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s < ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"<=\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s <= ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"=\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s = ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"!=\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s != ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"LIKE\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s LIKE ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"ILIKE\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s ILIKE ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"IN\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tif values, ok := op.GetValue().([]interface{}); ok {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\tplaceholders := make([]string, len(values))\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\tfor j := range placeholders {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\t\tplaceholders[j] = \"?\"\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s IN (%%s)\", quotedField, strings.Join(placeholders, \", \")), values...)\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"NOT IN\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tif values, ok := op.GetValue().([]interface{}); ok {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\tplaceholders := make([]string, len(values))\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\tfor j := range placeholders {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\t\tplaceholders[j] = \"?\"\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s NOT IN (%%s)\", quotedField, strings.Join(placeholders, \", \")), values...)\n")
	fmt.Fprintf(file, "\t\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"IS NULL\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s IS NULL\", quotedField))\n")
	fmt.Fprintf(file, "\t\t\t\t\tcase \"IS NOT NULL\":\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s IS NOT NULL\", quotedField))\n")
	fmt.Fprintf(file, "\t\t\t\t\tdefault:\n")
	fmt.Fprintf(file, "\t\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s = ?\", quotedField), op.GetValue())\n")
	fmt.Fprintf(file, "\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t} else if value == nil {\n")
	fmt.Fprintf(file, "\t\t\t\t\tquotedField := query.GetDialect().QuoteIdentifier(field)\n")
	fmt.Fprintf(file, "\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s IS NULL\", quotedField))\n")
	fmt.Fprintf(file, "\t\t\t\t} else {\n")
	fmt.Fprintf(file, "\t\t\t\t\tquotedField := query.GetDialect().QuoteIdentifier(field)\n")
	fmt.Fprintf(file, "\t\t\t\t\tquery.Or(fmt.Sprintf(\"%%s = ?\", quotedField), value)\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t}\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\t// Handle regular conditions (non-OR)\n")
	fmt.Fprintf(file, "\t// Create a copy without Or field to avoid recursion\n")
	fmt.Fprintf(file, "\tregularWhere := where\n")
	fmt.Fprintf(file, "\tregularWhere.Or = nil\n")
	fmt.Fprintf(file, "\tregularMap := Convert%sWhereInputToWhere(regularWhere)\n", pascalModelName)
	fmt.Fprintf(file, "\tif len(regularMap) > 0 {\n")
	fmt.Fprintf(file, "\t\tquery.Where(regularMap)\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "}\n\n")
}

// generateFilterConverter generates code to convert a Filter type to WhereOperator
func generateFilterConverter(file *os.File, filterType, fieldName string) {
	switch filterType {
	case "StringFilter":
		fmt.Fprintf(file, "\t\tif filter.Contains != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Contains(*filter.Contains)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.StartsWith != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.StartsWith(*filter.StartsWith)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.EndsWith != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.EndsWith(*filter.EndsWith)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.ContainsInsensitive != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.ContainsInsensitive(*filter.ContainsInsensitive)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.NotEquals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotEquals(*filter.NotEquals)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif len(filter.In) > 0 {\n")
		fmt.Fprintf(file, "\t\t\tvalues := make([]interface{}, len(filter.In))\n")
		fmt.Fprintf(file, "\t\t\tfor i, v := range filter.In {\n")
		fmt.Fprintf(file, "\t\t\t\tvalues[i] = v\n")
		fmt.Fprintf(file, "\t\t\t}\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.In(values...)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif len(filter.NotIn) > 0 {\n")
		fmt.Fprintf(file, "\t\t\tvalues := make([]interface{}, len(filter.NotIn))\n")
		fmt.Fprintf(file, "\t\t\tfor i, v := range filter.NotIn {\n")
		fmt.Fprintf(file, "\t\t\t\tvalues[i] = v\n")
		fmt.Fprintf(file, "\t\t\t}\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotIn(values...)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNull != nil && *filter.IsNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNotNull != nil && *filter.IsNotNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNotNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	case "IntFilter":
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.NotEquals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotEquals(*filter.NotEquals)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Gt != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Gt(*filter.Gt)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Gte != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Gte(*filter.Gte)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Lt != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Lt(*filter.Lt)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Lte != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Lte(*filter.Lte)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif len(filter.In) > 0 {\n")
		fmt.Fprintf(file, "\t\t\tvalues := make([]interface{}, len(filter.In))\n")
		fmt.Fprintf(file, "\t\t\tfor i, v := range filter.In {\n")
		fmt.Fprintf(file, "\t\t\t\tvalues[i] = v\n")
		fmt.Fprintf(file, "\t\t\t}\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.In(values...)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif len(filter.NotIn) > 0 {\n")
		fmt.Fprintf(file, "\t\t\tvalues := make([]interface{}, len(filter.NotIn))\n")
		fmt.Fprintf(file, "\t\t\tfor i, v := range filter.NotIn {\n")
		fmt.Fprintf(file, "\t\t\t\tvalues[i] = v\n")
		fmt.Fprintf(file, "\t\t\t}\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotIn(values...)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNull != nil && *filter.IsNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNotNull != nil && *filter.IsNotNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNotNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	case "Int64Filter":
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.NotEquals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotEquals(*filter.NotEquals)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Gt != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Gt(*filter.Gt)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Gte != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Gte(*filter.Gte)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Lt != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Lt(*filter.Lt)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Lte != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Lte(*filter.Lte)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif len(filter.In) > 0 {\n")
		fmt.Fprintf(file, "\t\t\tvalues := make([]interface{}, len(filter.In))\n")
		fmt.Fprintf(file, "\t\t\tfor i, v := range filter.In {\n")
		fmt.Fprintf(file, "\t\t\t\tvalues[i] = v\n")
		fmt.Fprintf(file, "\t\t\t}\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.In(values...)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif len(filter.NotIn) > 0 {\n")
		fmt.Fprintf(file, "\t\t\tvalues := make([]interface{}, len(filter.NotIn))\n")
		fmt.Fprintf(file, "\t\t\tfor i, v := range filter.NotIn {\n")
		fmt.Fprintf(file, "\t\t\t\tvalues[i] = v\n")
		fmt.Fprintf(file, "\t\t\t}\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotIn(values...)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNull != nil && *filter.IsNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNotNull != nil && *filter.IsNotNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNotNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	case "FloatFilter":
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.NotEquals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotEquals(*filter.NotEquals)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Gt != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Gt(*filter.Gt)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Gte != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Gte(*filter.Gte)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Lt != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Lt(*filter.Lt)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Lte != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Lte(*filter.Lte)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif len(filter.In) > 0 {\n")
		fmt.Fprintf(file, "\t\t\tvalues := make([]interface{}, len(filter.In))\n")
		fmt.Fprintf(file, "\t\t\tfor i, v := range filter.In {\n")
		fmt.Fprintf(file, "\t\t\t\tvalues[i] = v\n")
		fmt.Fprintf(file, "\t\t\t}\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.In(values...)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif len(filter.NotIn) > 0 {\n")
		fmt.Fprintf(file, "\t\t\tvalues := make([]interface{}, len(filter.NotIn))\n")
		fmt.Fprintf(file, "\t\t\tfor i, v := range filter.NotIn {\n")
		fmt.Fprintf(file, "\t\t\t\tvalues[i] = v\n")
		fmt.Fprintf(file, "\t\t\t}\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotIn(values...)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNull != nil && *filter.IsNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNotNull != nil && *filter.IsNotNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNotNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	case "BooleanFilter":
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.NotEquals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotEquals(*filter.NotEquals)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNull != nil && *filter.IsNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNotNull != nil && *filter.IsNotNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNotNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	case "DateTimeFilter":
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.NotEquals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotEquals(*filter.NotEquals)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Gt != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Gt(*filter.Gt)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Gte != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Gte(*filter.Gte)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Lt != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Lt(*filter.Lt)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.Lte != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.Lte(*filter.Lte)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNull != nil && *filter.IsNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNotNull != nil && *filter.IsNotNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNotNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	case "JsonFilter":
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.NotEquals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotEquals(*filter.NotEquals)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNull != nil && *filter.IsNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNotNull != nil && *filter.IsNotNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNotNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	case "BytesFilter":
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.NotEquals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.NotEquals(*filter.NotEquals)\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNull != nil && *filter.IsNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
		fmt.Fprintf(file, "\t\tif filter.IsNotNull != nil && *filter.IsNotNull {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = builder.IsNotNull()\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	default:
		// For unknown filter types, try to use Equals if available
		fmt.Fprintf(file, "\t\tif filter.Equals != nil {\n")
		fmt.Fprintf(file, "\t\t\tresult[%q] = *filter.Equals\n", fieldName)
		fmt.Fprintf(file, "\t\t}\n")
	}
}

// generatePrismaBuilders generates Prisma-style builder methods for FindFirst, FindMany, Count, Delete, Update, Create
func generatePrismaBuilders(file *os.File, model *parser.Model, schema *parser.Schema) {
	pascalModelName := toPascalCase(model.Name)

	// FindFirst builder
	fmt.Fprintf(file, "// FindFirst returns a builder for finding a single %s record (Prisma-style)\n", pascalModelName)
	fmt.Fprintf(file, "// Example: tenant, err := q.FindFirst().Where(inputs.%sWhereInput{...}).Exec(ctx)\n", pascalModelName)
	fmt.Fprintf(file, "func (q *%sQuery) FindFirst() *%sFindFirstBuilder {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn &%sFindFirstBuilder{query: q}\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// %sFindFirstBuilder is a builder for finding a single %s record\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "type %sFindFirstBuilder struct {\n", pascalModelName)
	fmt.Fprintf(file, "\tquery     *%sQuery\n", pascalModelName)
	fmt.Fprintf(file, "\twhereInput *inputs.%sWhereInput\n", pascalModelName)
	fmt.Fprintf(file, "\tselectFields *inputs.%sSelect\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Where sets the where conditions\n")
	fmt.Fprintf(file, "func (b *%sFindFirstBuilder) Where(where inputs.%sWhereInput) *%sFindFirstBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.whereInput = &where\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Select sets which fields to return\n")
	fmt.Fprintf(file, "func (b *%sFindFirstBuilder) Select(selectFields inputs.%sSelect) *%sFindFirstBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.selectFields = &selectFields\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes the find first operation and returns the default model\n")
	fmt.Fprintf(file, "// Returns (*models.%s, error)\n", pascalModelName)
	fmt.Fprintf(file, "// For custom types, use ExecTyped[T]() instead\n")
	fmt.Fprintf(file, "func (b *%sFindFirstBuilder) Exec(ctx context.Context) (*models.%s, error) {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tif b.whereInput != nil {\n")
	fmt.Fprintf(file, "\t\tapply%sWhereInput(b.query.Query, *b.whereInput)\n", pascalModelName)
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tif b.selectFields != nil {\n")
	fmt.Fprintf(file, "\t\tvar selectedFields []string\n")
	for _, field := range model.Fields {
		if isRelation(field, schema) {
			continue
		}
		fieldName := toPascalCase(field.Name)
		// Get the actual database column name (check for @map attribute)
		columnName := field.Name
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					columnName = val
					break
				}
			}
		}
		fmt.Fprintf(file, "\t\tif b.selectFields.%s {\n", fieldName)
		fmt.Fprintf(file, "\t\t\tselectedFields = append(selectedFields, %q)\n", columnName)
		fmt.Fprintf(file, "\t\t}\n")
	}
	fmt.Fprintf(file, "\t\tif len(selectedFields) > 0 {\n")
	fmt.Fprintf(file, "\t\t\tb.query.Select(selectedFields...)\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tvar result models.%s\n", pascalModelName)
	fmt.Fprintf(file, "\terr := b.query.First(ctx, &result)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &result, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// ExecTyped executes the find first operation and scans the result into the provided type\n")
	fmt.Fprintf(file, "// dest must be a pointer to a struct with json or db tags for field mapping\n")
	fmt.Fprintf(file, "// Example: var dto *TenantsDTO; err := builder.ExecTyped(ctx, &dto)\n")
	fmt.Fprintf(file, "func (b *%sFindFirstBuilder) ExecTyped(ctx context.Context, dest interface{}) error {\n", pascalModelName)
	fmt.Fprintf(file, "\tif b.whereInput != nil {\n")
	fmt.Fprintf(file, "\t\twhereMap := Convert%sWhereInputToWhere(*b.whereInput)\n", pascalModelName)
	fmt.Fprintf(file, "\t\tb.query.Where(whereMap)\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tif b.selectFields != nil {\n")
	fmt.Fprintf(file, "\t\tvar selectedFields []string\n")
	for _, field := range model.Fields {
		if isRelation(field, schema) {
			continue
		}
		fieldName := toPascalCase(field.Name)
		// Get the actual database column name (check for @map attribute)
		columnName := field.Name
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					columnName = val
					break
				}
			}
		}
		fmt.Fprintf(file, "\t\tif b.selectFields.%s {\n", fieldName)
		fmt.Fprintf(file, "\t\t\tselectedFields = append(selectedFields, %q)\n", columnName)
		fmt.Fprintf(file, "\t\t}\n")
	}
	fmt.Fprintf(file, "\t\tif len(selectedFields) > 0 {\n")
	fmt.Fprintf(file, "\t\t\tb.query.Select(selectedFields...)\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\t// Validate dest is a pointer\n")
	fmt.Fprintf(file, "\tdestVal := reflect.ValueOf(dest)\n")
	fmt.Fprintf(file, "\tif destVal.Kind() != reflect.Ptr {\n")
	fmt.Fprintf(file, "\t\treturn fmt.Errorf(\"ExecTyped: dest must be a pointer (e.g., *TenantsDTO), got %%v\", destVal.Kind())\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\telemType := destVal.Elem().Type()\n")
	fmt.Fprintf(file, "\tif elemType.Kind() != reflect.Struct {\n")
	fmt.Fprintf(file, "\t\treturn fmt.Errorf(\"ExecTyped: dest must be a pointer to struct, got %%v\", elemType.Kind())\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\t// Scan into dest\n")
	fmt.Fprintf(file, "\terr := b.query.ScanFirst(ctx, dest, elemType)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn nil\n")
	fmt.Fprintf(file, "}\n\n")

	// FindMany builder
	fmt.Fprintf(file, "// FindMany returns a builder for finding multiple %s records (Prisma-style)\n", pascalModelName)
	fmt.Fprintf(file, "// Example: tenants, err := q.FindMany().Where(inputs.%sWhereInput{...}).Exec(ctx)\n", pascalModelName)
	fmt.Fprintf(file, "func (q *%sQuery) FindMany() *%sFindManyBuilder {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn &%sFindManyBuilder{query: q}\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// %sFindManyBuilder is a builder for finding multiple %s records\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "type %sFindManyBuilder struct {\n", pascalModelName)
	fmt.Fprintf(file, "\tquery       *%sQuery\n", pascalModelName)
	fmt.Fprintf(file, "\twhereInput  *inputs.%sWhereInput\n", pascalModelName)
	fmt.Fprintf(file, "\tselectFields *inputs.%sSelect\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Where sets the where conditions\n")
	fmt.Fprintf(file, "func (b *%sFindManyBuilder) Where(where inputs.%sWhereInput) *%sFindManyBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.whereInput = &where\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Select sets which fields to return\n")
	fmt.Fprintf(file, "func (b *%sFindManyBuilder) Select(selectFields inputs.%sSelect) *%sFindManyBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.selectFields = &selectFields\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes the find many operation and returns the default model\n")
	fmt.Fprintf(file, "// Returns ([]models.%s, error)\n", pascalModelName)
	fmt.Fprintf(file, "// For custom types, use ExecTyped[T]() instead\n")
	fmt.Fprintf(file, "func (b *%sFindManyBuilder) Exec(ctx context.Context) ([]models.%s, error) {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tif b.whereInput != nil {\n")
	fmt.Fprintf(file, "\t\tapply%sWhereInput(b.query.Query, *b.whereInput)\n", pascalModelName)
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tif b.selectFields != nil {\n")
	fmt.Fprintf(file, "\t\tvar selectedFields []string\n")
	for _, field := range model.Fields {
		if isRelation(field, schema) {
			continue
		}
		fieldName := toPascalCase(field.Name)
		// Get the actual database column name (check for @map attribute)
		columnName := field.Name
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					columnName = val
					break
				}
			}
		}
		fmt.Fprintf(file, "\t\tif b.selectFields.%s {\n", fieldName)
		fmt.Fprintf(file, "\t\t\tselectedFields = append(selectedFields, %q)\n", columnName)
		fmt.Fprintf(file, "\t\t}\n")
	}
	fmt.Fprintf(file, "\t\tif len(selectedFields) > 0 {\n")
	fmt.Fprintf(file, "\t\t\tb.query.Select(selectedFields...)\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tvar results []models.%s\n", pascalModelName)
	fmt.Fprintf(file, "\terr := b.query.Find(ctx, &results)\n")
	fmt.Fprintf(file, "\treturn results, err\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// ExecTyped executes the find many operation and scans the results into the provided slice\n")
	fmt.Fprintf(file, "// dest must be a pointer to a slice of structs with json or db tags for field mapping\n")
	fmt.Fprintf(file, "// Example: var dtos []TenantsDTO; err := builder.ExecTyped(ctx, &dtos)\n")
	fmt.Fprintf(file, "func (b *%sFindManyBuilder) ExecTyped(ctx context.Context, dest interface{}) error {\n", pascalModelName)
	fmt.Fprintf(file, "\tif b.whereInput != nil {\n")
	fmt.Fprintf(file, "\t\twhereMap := Convert%sWhereInputToWhere(*b.whereInput)\n", pascalModelName)
	fmt.Fprintf(file, "\t\tb.query.Where(whereMap)\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tif b.selectFields != nil {\n")
	fmt.Fprintf(file, "\t\tvar selectedFields []string\n")
	for _, field := range model.Fields {
		if isRelation(field, schema) {
			continue
		}
		fieldName := toPascalCase(field.Name)
		// Get the actual database column name (check for @map attribute)
		columnName := field.Name
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					columnName = val
					break
				}
			}
		}
		fmt.Fprintf(file, "\t\tif b.selectFields.%s {\n", fieldName)
		fmt.Fprintf(file, "\t\t\tselectedFields = append(selectedFields, %q)\n", columnName)
		fmt.Fprintf(file, "\t\t}\n")
	}
	fmt.Fprintf(file, "\t\tif len(selectedFields) > 0 {\n")
	fmt.Fprintf(file, "\t\t\tb.query.Select(selectedFields...)\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\t// Validate dest is a pointer to slice\n")
	fmt.Fprintf(file, "\tdestVal := reflect.ValueOf(dest)\n")
	fmt.Fprintf(file, "\tif destVal.Kind() != reflect.Ptr {\n")
	fmt.Fprintf(file, "\t\treturn fmt.Errorf(\"ExecTyped: dest must be a pointer to slice (e.g., *[]TenantsDTO), got %%v\", destVal.Kind())\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tsliceVal := destVal.Elem()\n")
	fmt.Fprintf(file, "\tif sliceVal.Kind() != reflect.Slice {\n")
	fmt.Fprintf(file, "\t\treturn fmt.Errorf(\"ExecTyped: dest must be a pointer to slice, got %%v\", sliceVal.Kind())\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\telemType := sliceVal.Type().Elem()\n")
	fmt.Fprintf(file, "\tif elemType.Kind() == reflect.Ptr {\n")
	fmt.Fprintf(file, "\t\telemType = elemType.Elem()\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tif elemType.Kind() != reflect.Struct {\n")
	fmt.Fprintf(file, "\t\treturn fmt.Errorf(\"ExecTyped: dest must be a slice of structs, got %%v\", elemType.Kind())\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\t// Scan into dest\n")
	fmt.Fprintf(file, "\terr := b.query.ScanFind(ctx, dest, elemType)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn nil\n")
	fmt.Fprintf(file, "}\n\n")

	// Count builder
	fmt.Fprintf(file, "// Count returns a builder for counting %s records (Prisma-style)\n", pascalModelName)
	fmt.Fprintf(file, "// Example: count, err := q.Count().Where(inputs.%sWhereInput{...}).Exec(ctx)\n", pascalModelName)
	fmt.Fprintf(file, "func (q *%sQuery) Count() *%sCountBuilder {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn &%sCountBuilder{query: q}\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// %sCountBuilder is a builder for counting %s records\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "type %sCountBuilder struct {\n", pascalModelName)
	fmt.Fprintf(file, "\tquery      *%sQuery\n", pascalModelName)
	fmt.Fprintf(file, "\twhereInput *inputs.%sWhereInput\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Where sets the where conditions\n")
	fmt.Fprintf(file, "func (b *%sCountBuilder) Where(where inputs.%sWhereInput) *%sCountBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.whereInput = &where\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes the count operation\n")
	fmt.Fprintf(file, "func (b *%sCountBuilder) Exec(ctx context.Context) (int64, error) {\n", pascalModelName)
	fmt.Fprintf(file, "\tif b.whereInput != nil {\n")
	fmt.Fprintf(file, "\t\twhereMap := Convert%sWhereInputToWhere(*b.whereInput)\n", pascalModelName)
	fmt.Fprintf(file, "\t\tb.query.Where(whereMap)\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn b.query.Query.Count(ctx)\n")
	fmt.Fprintf(file, "}\n\n")

	// Delete builder
	fmt.Fprintf(file, "// Delete returns a builder for deleting %s records (Prisma-style)\n", pascalModelName)
	fmt.Fprintf(file, "// Example: err := q.Delete().Where(inputs.%sWhereInput{...}).Exec(ctx)\n", pascalModelName)
	fmt.Fprintf(file, "func (q *%sQuery) Delete() *%sDeleteBuilder {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn &%sDeleteBuilder{query: q}\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// %sDeleteBuilder is a builder for deleting %s records\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "type %sDeleteBuilder struct {\n", pascalModelName)
	fmt.Fprintf(file, "\tquery      *%sQuery\n", pascalModelName)
	fmt.Fprintf(file, "\twhereInput *inputs.%sWhereInput\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Where sets the where conditions\n")
	fmt.Fprintf(file, "func (b *%sDeleteBuilder) Where(where inputs.%sWhereInput) *%sDeleteBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.whereInput = &where\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes the delete operation\n")
	fmt.Fprintf(file, "func (b *%sDeleteBuilder) Exec(ctx context.Context) error {\n", pascalModelName)
	fmt.Fprintf(file, "\tif b.whereInput == nil {\n")
	fmt.Fprintf(file, "\t\treturn fmt.Errorf(\"where condition is required for delete\")\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\twhereMap := Convert%sWhereInputToWhere(*b.whereInput)\n", pascalModelName)
	fmt.Fprintf(file, "\tb.query.Where(whereMap)\n")
	fmt.Fprintf(file, "\treturn b.query.Query.Delete(ctx, &models.%s{})\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	// Update builder
	fmt.Fprintf(file, "// Update returns a builder for updating %s records (Prisma-style)\n", pascalModelName)
	fmt.Fprintf(file, "// Example: err := q.Update().Where(inputs.%sWhereInput{...}).Data(inputs.%sUpdateInput{...}).Exec(ctx)\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "func (q *%sQuery) Update() *%sUpdateBuilder {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn &%sUpdateBuilder{query: q}\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// %sUpdateBuilder is a builder for updating %s records\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "type %sUpdateBuilder struct {\n", pascalModelName)
	fmt.Fprintf(file, "\tquery      *%sQuery\n", pascalModelName)
	fmt.Fprintf(file, "\twhereInput *inputs.%sWhereInput\n", pascalModelName)
	fmt.Fprintf(file, "\tdata       *inputs.%sUpdateInput\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Where sets the where conditions\n")
	fmt.Fprintf(file, "func (b *%sUpdateBuilder) Where(where inputs.%sWhereInput) *%sUpdateBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.whereInput = &where\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Data sets the data for updating\n")
	fmt.Fprintf(file, "func (b *%sUpdateBuilder) Data(data inputs.%sUpdateInput) *%sUpdateBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.data = &data\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes the update operation\n")
	fmt.Fprintf(file, "func (b *%sUpdateBuilder) Exec(ctx context.Context) error {\n", pascalModelName)
	fmt.Fprintf(file, "\tif b.whereInput == nil {\n")
	fmt.Fprintf(file, "\t\treturn fmt.Errorf(\"where condition is required for update\")\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tif b.data == nil {\n")
	fmt.Fprintf(file, "\t\treturn fmt.Errorf(\"data is required for update\")\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\twhereMap := Convert%sWhereInputToWhere(*b.whereInput)\n", pascalModelName)
	fmt.Fprintf(file, "\tb.query.Where(whereMap)\n")
	fmt.Fprintf(file, "\tupdateData := make(map[string]interface{})\n")
	for _, field := range model.Fields {
		if isAutoGenerated(field) || isPrimaryKey(field) || isRelation(field, schema) {
			continue
		}
		fieldName := toPascalCase(field.Name)
		// Get the actual database column name (check for @map attribute)
		dbFieldName := field.Name
		for _, attr := range field.Attributes {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				if val, ok := attr.Arguments[0].Value.(string); ok {
					dbFieldName = val
					break
				}
			}
		}
		fmt.Fprintf(file, "\tif b.data.%s != nil {\n", fieldName)
		fmt.Fprintf(file, "\t\tupdateData[%q] = *b.data.%s\n", dbFieldName, fieldName)
		fmt.Fprintf(file, "\t}\n")
	}
	fmt.Fprintf(file, "\treturn b.query.Updates(ctx, updateData)\n")
	fmt.Fprintf(file, "}\n\n")

	// Create builder
	fmt.Fprintf(file, "// Create returns a builder for creating %s records (Prisma-style)\n", pascalModelName)
	fmt.Fprintf(file, "// Example: tenant, err := q.Create().Data(inputs.%sCreateInput{...}).Exec(ctx)\n", pascalModelName)
	fmt.Fprintf(file, "func (q *%sQuery) Create() *%sCreateBuilder {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\treturn &%sCreateBuilder{query: q}\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// %sCreateBuilder is a builder for creating %s records\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "type %sCreateBuilder struct {\n", pascalModelName)
	fmt.Fprintf(file, "\tquery *%sQuery\n", pascalModelName)
	fmt.Fprintf(file, "\tdata  *inputs.%sCreateInput\n", pascalModelName)
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Data sets the data for creating\n")
	fmt.Fprintf(file, "func (b *%sCreateBuilder) Data(data inputs.%sCreateInput) *%sCreateBuilder {\n", pascalModelName, pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tb.data = &data\n")
	fmt.Fprintf(file, "\treturn b\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes the create operation\n")
	fmt.Fprintf(file, "func (b *%sCreateBuilder) Exec(ctx context.Context) (*models.%s, error) {\n", pascalModelName, pascalModelName)
	fmt.Fprintf(file, "\tif b.data == nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, fmt.Errorf(\"data is required for create\")\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\tresult := &models.%s{}\n", pascalModelName)
	for _, field := range model.Fields {
		if isAutoGenerated(field) || isRelation(field, schema) {
			continue
		}
		fieldName := toPascalCase(field.Name)
		// Check if field is optional (pointer) in both model and input
		isOptional := field.Type != nil && field.Type.IsOptional
		if isOptional {
			fmt.Fprintf(file, "\tif b.data.%s != nil {\n", fieldName)
			// Special handling for types that don't use pointers in models (json.RawMessage, []byte)
			// but are pointers in inputs
			if isNonPointerOptionalType(field.Type) {
				// Model doesn't use pointer, input does - need to dereference
				fmt.Fprintf(file, "\t\tresult.%s = *b.data.%s\n", fieldName, fieldName)
			} else {
				// Both model and input use pointers - copy pointer directly
				fmt.Fprintf(file, "\t\tresult.%s = b.data.%s\n", fieldName, fieldName)
			}
			fmt.Fprintf(file, "\t}\n")
		} else {
			// Field is required (not a pointer), assign directly
			fmt.Fprintf(file, "\tresult.%s = b.data.%s\n", fieldName, fieldName)
		}
	}

	// Use TableQueryBuilder.Create which returns the actual database result
	columns := getModelColumns(model, schema)
	primaryKey := getPrimaryKey(model)
	tableName := getTableName(model)
	fmt.Fprintf(file, "\t// Use TableQueryBuilder to get the actual result from database\n")
	fmt.Fprintf(file, "\tcolumns := []string{%s}\n", formatColumns(columns))
	fmt.Fprintf(file, "\ttableBuilder := builder.NewTableQueryBuilder(b.query.Query.GetDB(), %q, columns)\n", tableName)
	if primaryKey != "" {
		fmt.Fprintf(file, "\ttableBuilder.SetPrimaryKey(%q)\n", primaryKey)
	}
	fmt.Fprintf(file, "\ttableBuilder.SetDialect(b.query.Query.GetDialect())\n")
	fmt.Fprintf(file, "\ttableBuilder.SetModelType(reflect.TypeOf(models.%s{}))\n", pascalModelName)
	fmt.Fprintf(file, "\tcreated, err := tableBuilder.Create(ctx, result)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\t// Convert the result from interface{} to *models.%s\n", pascalModelName)
	fmt.Fprintf(file, "\tif createdModel, ok := created.(models.%s); ok {\n", pascalModelName)
	fmt.Fprintf(file, "\t\treturn &createdModel, nil\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\t// Fallback: if conversion fails, return the result we prepared\n")
	fmt.Fprintf(file, "\t// This should not happen, but provides a safety net\n")
	fmt.Fprintf(file, "\treturn result, nil\n")
	fmt.Fprintf(file, "}\n\n")
}

// isNonPointerOptionalType checks if a field type doesn't use pointers in models
// even when optional (json.RawMessage and []byte)
func isNonPointerOptionalType(fieldType *parser.FieldType) bool {
	if fieldType == nil {
		return false
	}
	// Json and Bytes types don't use pointers in models even when optional
	return fieldType.Name == "Json" || fieldType.Name == "Bytes"
}

// determineQueryImports determines which imports are needed for query files
func determineQueryImports(userModule, outputDir string) []string {
	// Calculate import paths for generated packages
	modelsPath, _, inputsPath, err := calculateImportPath(userModule, outputDir)
	if err != nil {
		// Fallback to old paths if detection fails
		modelsPath = "github.com/carlosnayan/prisma-go-client/db/models"
		inputsPath = "github.com/carlosnayan/prisma-go-client/db/inputs"
	}

	// Calculate local import path for builder (standalone package)
	builderPath, _, err := calculateLocalImportPath(userModule, outputDir)
	if err != nil {
		// Fallback to old path if detection fails
		builderPath = "github.com/carlosnayan/prisma-go-client/db/builder"
	}

	// context is always needed for all query methods
	// fmt is needed for fmt.Errorf in builders
	// reflect is needed for Scan() method
	// strings is needed for buildColumnToFieldMapForScan (strings.Index)
	// builder is always needed for Query embedding
	// models is always needed for type references
	// inputs is needed for WhereInput
	return []string{
		"context",
		"fmt",
		"reflect",
		"strings",
		builderPath,
		modelsPath,
		inputsPath,
	}
}
