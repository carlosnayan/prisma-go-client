package migrations

import (
	"fmt"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateSchemaFromDatabase gera um schema.prisma a partir do banco de dados introspectado
func GenerateSchemaFromDatabase(dbSchema *DatabaseSchema, provider string) (*parser.Schema, error) {
	schema := &parser.Schema{
		Datasources: []*parser.Datasource{},
		Generators:  []*parser.Generator{},
		Models:      []*parser.Model{},
		Enums:       []*parser.Enum{},
	}

	// Criar datasource
	datasource := &parser.Datasource{
		Name: "db",
		Fields: []*parser.Field{
			{Name: "provider", Value: provider},
		},
	}
	schema.Datasources = append(schema.Datasources, datasource)

	// Criar generator padrão
	generator := &parser.Generator{
		Name: "client",
		Fields: []*parser.Field{
			{Name: "provider", Value: "prisma-client-go"},
			{Name: "output", Value: "../db"},
		},
	}
	schema.Generators = append(schema.Generators, generator)

	// Converter tabelas para models
	for tableName, tableInfo := range dbSchema.Tables {
		model := convertTableToModel(tableName, tableInfo, provider)
		schema.Models = append(schema.Models, model)
	}

	return schema, nil
}

// convertTableToModel converte uma tabela do banco em um model Prisma
func convertTableToModel(tableName string, tableInfo *TableInfo, provider string) *parser.Model {
	model := &parser.Model{
		Name:       toPascalCase(tableName),
		Fields:     []*parser.ModelField{},
		Attributes: []*parser.Attribute{},
	}

	// Converter colunas para campos
	for colName, colInfo := range tableInfo.Columns {
		field := &parser.ModelField{
			Name:       toCamelCase(colName),
			Type:       convertTypeToPrisma(colInfo.Type, colInfo.IsNullable),
			Attributes: []*parser.Attribute{},
		}

		// Adicionar atributos
		if colInfo.IsPrimaryKey {
			field.Attributes = append(field.Attributes, &parser.Attribute{
				Name:      "id",
				Arguments: []*parser.AttributeArgument{},
			})

			// Verificar se tem default autoincrement
			if colInfo.DefaultValue != nil && strings.Contains(*colInfo.DefaultValue, "nextval") {
				field.Attributes = append(field.Attributes, &parser.Attribute{
					Name: "default",
					Arguments: []*parser.AttributeArgument{
						{Value: "autoincrement()"},
					},
				})
			}
		}

		if colInfo.IsUnique && !colInfo.IsPrimaryKey {
			field.Attributes = append(field.Attributes, &parser.Attribute{
				Name:      "unique",
				Arguments: []*parser.AttributeArgument{},
			})
		}

		if colInfo.DefaultValue != nil && !colInfo.IsPrimaryKey {
			defaultVal := *colInfo.DefaultValue
			// Limpar default value (remover ::type, nextval, etc.)
			if !strings.Contains(defaultVal, "nextval") {
				// Tentar extrair valor literal
				if strings.HasPrefix(defaultVal, "'") && strings.HasSuffix(defaultVal, "'") {
					defaultVal = strings.Trim(defaultVal, "'")
					field.Attributes = append(field.Attributes, &parser.Attribute{
						Name: "default",
						Arguments: []*parser.AttributeArgument{
							{Value: fmt.Sprintf("%q", defaultVal)},
						},
					})
				} else if defaultVal == "now()" || defaultVal == "CURRENT_TIMESTAMP" {
					field.Attributes = append(field.Attributes, &parser.Attribute{
						Name: "default",
						Arguments: []*parser.AttributeArgument{
							{Value: "now()"},
						},
					})
				}
			}
		}

		// Adicionar @map se o nome da coluna for diferente do campo
		if colName != field.Name {
			field.Attributes = append(field.Attributes, &parser.Attribute{
				Name: "map",
				Arguments: []*parser.AttributeArgument{
					{Value: fmt.Sprintf("%q", colName)},
				},
			})
		}

		model.Fields = append(model.Fields, field)
	}

	return model
}

// convertTypeToPrisma converte tipo do banco para tipo Prisma
func convertTypeToPrisma(dbType string, isNullable bool) *parser.FieldType {
	prismaType := mapDatabaseTypeToPrisma(dbType)
	return &parser.FieldType{
		Name:       prismaType,
		IsOptional: isNullable,
		IsArray:    false,
	}
}

// mapDatabaseTypeToPrisma mapeia tipo do banco para Prisma
func mapDatabaseTypeToPrisma(dbType string) string {
	dbType = strings.ToLower(dbType)

	switch {
	case strings.HasPrefix(dbType, "varchar"), dbType == "text", dbType == "char":
		return "String"
	case dbType == "integer" || dbType == "int" || dbType == "int4":
		return "Int"
	case dbType == "bigint" || dbType == "int8":
		return "BigInt"
	case dbType == "boolean" || dbType == "bool":
		return "Boolean"
	case dbType == "timestamp" || dbType == "timestamptz" || dbType == "date" || dbType == "time":
		return "DateTime"
	case dbType == "real" || dbType == "double precision" || dbType == "float8":
		return "Float"
	case strings.HasPrefix(dbType, "numeric") || strings.HasPrefix(dbType, "decimal"):
		return "Decimal"
	case dbType == "jsonb" || dbType == "json":
		return "Json"
	case dbType == "bytea":
		return "Bytes"
	default:
		return "String" // Default
	}
}

// toPascalCase converte snake_case para PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	result := ""
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return result
}

// toCamelCase converte snake_case para camelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	result := ""
	for i, part := range parts {
		if len(part) > 0 {
			if i == 0 {
				result += strings.ToLower(part)
			} else {
				result += strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
			}
		}
	}
	return result
}
