package cmd

import (
	"fmt"
	"os"

	"github.com/carlosnayan/prisma-go-client/cli"
	"github.com/carlosnayan/prisma-go-client/internal/migrations"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

var (
	diffFrom string
	diffTo   string
	diffOut  string
)

var migrateDiffCmd = &cli.Command{
	Name:  "diff",
	Short: "Compare two schemas and generate a migration",
	Long: `Compares two schema.prisma files and generates a SQL migration
representing the differences between them.

Examples:
  prisma migrate diff --from schema1.prisma --to schema2.prisma
  prisma migrate diff --from schema.prisma --to database
  prisma migrate diff --from database --to schema.prisma`,
	Flags: []*cli.Flag{
		{
			Name:     "from",
			Usage:    "Source schema (.prisma file or 'database')",
			Value:    &diffFrom,
			Required: true,
		},
		{
			Name:     "to",
			Usage:    "Target schema (.prisma file or 'database')",
			Value:    &diffTo,
			Required: true,
		},
		{
			Name:  "out",
			Usage: "Output file for migration SQL (optional)",
			Value: &diffOut,
		},
	},
	Run: runMigrateDiff,
}

func runMigrateDiff(args []string) error {
	if err := checkProjectRoot(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Parse source schema
	var fromSchema *parser.Schema
	if diffFrom == "database" {
		// Introspect database
		dbURL := cfg.GetDatabaseURL()
		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL not configured")
		}

		db, err := migrations.ConnectDatabase(dbURL)
		if err != nil {
			return fmt.Errorf("error connecting to database: %w", err)
		}
		defer db.Close()

		provider := migrations.DetectProvider(dbURL)
		fmt.Printf("Introspecting database (%s)...\n", provider)

		dbSchema, err := migrations.IntrospectDatabase(db, provider)
		if err != nil {
			return fmt.Errorf("error introspecting database: %w", err)
		}

		fromSchema, err = migrations.GenerateSchemaFromDatabase(dbSchema, provider)
		if err != nil {
			return fmt.Errorf("error generating schema from database: %w", err)
		}
	} else {
		// Parse schema.prisma file
		if _, err := os.Stat(diffFrom); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", diffFrom)
		}

		schema, errors, err := parser.ParseFile(diffFrom)
		if err != nil {
			return fmt.Errorf("error parsing schema: %w", err)
		}
		if len(errors) > 0 {
			fmt.Println("Warnings in source schema:")
			for _, e := range errors {
				fmt.Printf("  %s\n", e)
			}
		}
		fromSchema = schema
	}

	// Parse target schema
	var toSchema *parser.Schema
	if diffTo == "database" {
		// Introspect database
		dbURL := cfg.GetDatabaseURL()
		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL not configured")
		}

		db, err := migrations.ConnectDatabase(dbURL)
		if err != nil {
			return fmt.Errorf("error connecting to database: %w", err)
		}
		defer db.Close()

		provider := migrations.DetectProvider(dbURL)
		fmt.Printf("Introspecting database (%s)...\n", provider)

		dbSchema, err := migrations.IntrospectDatabase(db, provider)
		if err != nil {
			return fmt.Errorf("error introspecting database: %w", err)
		}

		toSchema, err = migrations.GenerateSchemaFromDatabase(dbSchema, provider)
		if err != nil {
			return fmt.Errorf("error generating schema from database: %w", err)
		}
	} else {
		// Parse schema.prisma file
		if _, err := os.Stat(diffTo); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", diffTo)
		}

		schema, errors, err := parser.ParseFile(diffTo)
		if err != nil {
			return fmt.Errorf("error parsing schema: %w", err)
		}
		if len(errors) > 0 {
			fmt.Println("Warnings in target schema:")
			for _, e := range errors {
				fmt.Printf("  %s\n", e)
			}
		}
		toSchema = schema
	}

	// Get provider
	provider := migrations.GetProviderFromSchema(toSchema)
	if provider == "" {
		provider = migrations.GetProviderFromSchema(fromSchema)
	}
	if provider == "" {
		provider = "postgresql" // Default
	}

	// Convert fromSchema to DatabaseSchema for comparison
	fromDbSchema, err := schemaToDatabaseSchema(fromSchema, provider)
	if err != nil {
		return fmt.Errorf("error converting source schema: %w", err)
	}

	// Compare schemas (toSchema vs fromDbSchema)
	fmt.Println("Comparing schemas...")
	diff, err := migrations.CompareSchema(toSchema, fromDbSchema, provider)
	if err != nil {
		return fmt.Errorf("error comparing schemas: %w", err)
	}

	// Check if there are changes
	hasChanges := len(diff.TablesToCreate) > 0 ||
		len(diff.TablesToAlter) > 0 ||
		len(diff.TablesToDrop) > 0 ||
		len(diff.IndexesToCreate) > 0 ||
		len(diff.IndexesToDrop) > 0

	if !hasChanges {
		fmt.Println("No differences found between schemas.")
		return nil
	}

	// Generate migration SQL
	fmt.Println("Generating migration SQL...")
	sql, err := migrations.GenerateMigrationSQL(diff, provider)
	if err != nil {
		return fmt.Errorf("error generating SQL: %w", err)
	}

	// Write output
	if diffOut != "" {
		if err := os.WriteFile(diffOut, []byte(sql), 0644); err != nil {
			return fmt.Errorf("error writing file: %w", err)
		}
		fmt.Printf("Migration generated: %s\n", diffOut)
	} else {
		fmt.Println("\n=== SQL Migration ===")
		fmt.Println(sql)
	}

	return nil
}

// schemaToDatabaseSchema converte um parser.Schema para DatabaseSchema
// Isso é necessário para usar CompareSchema que espera DatabaseSchema
func schemaToDatabaseSchema(schema *parser.Schema, provider string) (*migrations.DatabaseSchema, error) {
	// Criar um DatabaseSchema vazio
	dbSchema := &migrations.DatabaseSchema{
		Tables: make(map[string]*migrations.TableInfo),
	}

	// Converter models para tabelas
	for _, model := range schema.Models {
		tableInfo := &migrations.TableInfo{
			Name:    model.Name,
			Columns: make(map[string]*migrations.ColumnInfo),
			Indexes: []*migrations.IndexInfo{},
		}

		// Converter campos para colunas
		for _, field := range model.Fields {
			defaultVal := ""
			hasDefault := false

			// Verificar atributos
			isPrimaryKey := false
			isUnique := false
			for _, attr := range field.Attributes {
				switch attr.Name {
				case "id":
					isPrimaryKey = true
				case "unique":
					isUnique = true
				case "default":
					if len(attr.Arguments) > 0 {
						if str, ok := attr.Arguments[0].Value.(string); ok {
							defaultVal = str
							hasDefault = true
						}
					}
				}
			}

			var defaultPtr *string
			if hasDefault {
				defaultPtr = &defaultVal
			}

			colInfo := &migrations.ColumnInfo{
				Name:         field.Name,
				Type:         mapPrismaTypeToSQLType(field.Type.Name, provider),
				IsNullable:   field.Type.IsOptional,
				IsPrimaryKey: isPrimaryKey,
				IsUnique:     isUnique,
				DefaultValue: defaultPtr,
			}

			tableInfo.Columns[field.Name] = colInfo
		}

		dbSchema.Tables[model.Name] = tableInfo
	}

	return dbSchema, nil
}

// mapPrismaTypeToSQLType mapeia tipo Prisma para tipo SQL
func mapPrismaTypeToSQLType(prismaType, provider string) string {
	// Usar a função existente de migrations
	// Por enquanto, mapeamento simples (mesmo usado em diff.go)
	switch prismaType {
	case "String":
		if provider == "mysql" {
			return "VARCHAR(255)"
		}
		return "TEXT"
	case "Int":
		return "INTEGER"
	case "BigInt":
		return "BIGINT"
	case "Boolean":
		if provider == "sqlite" {
			return "INTEGER"
		}
		return "BOOLEAN"
	case "DateTime":
		if provider == "sqlite" {
			return "TEXT"
		}
		return "TIMESTAMP"
	case "Float":
		return "DOUBLE PRECISION"
	case "Decimal":
		return "DECIMAL"
	case "Json":
		if provider == "sqlite" {
			return "TEXT"
		}
		return "JSONB"
	case "Bytes":
		if provider == "mysql" {
			return "BLOB"
		}
		return "BYTEA"
	default:
		return "TEXT"
	}
}
