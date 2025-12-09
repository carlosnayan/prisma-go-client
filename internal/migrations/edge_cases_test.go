package migrations

import (
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

type MigrationTestCase struct {
	Name           string
	Schema         *parser.Schema
	DBSchema       *DatabaseSchema
	ExpectedSQL    []string
	UnexpectedSQL  []string
	ExpectedErrors []string
}

func runMigrationTest(t *testing.T, tc MigrationTestCase) {
	t.Helper()
	t.Run(tc.Name, func(t *testing.T) {
		diff, err := CompareSchema(tc.Schema, tc.DBSchema, "postgresql")
		if err != nil {
			t.Fatalf("CompareSchema failed: %v", err)
		}

		sql, err := GenerateMigrationSQL(diff, "postgresql")
		if err != nil {
			t.Fatalf("GenerateMigrationSQL failed: %v", err)
		}

		t.Logf("Generated SQL:\n%s", sql)

		for _, expected := range tc.ExpectedSQL {
			if !strings.Contains(sql, expected) {
				t.Errorf("SQL missing expected content: %s", expected)
			}
		}

		for _, unexpected := range tc.UnexpectedSQL {
			if strings.Contains(sql, unexpected) {
				t.Errorf("SQL contains unexpected content: %s", unexpected)
			}
		}
	})
}

func TestEdgeCases_Robust(t *testing.T) {
	tests := []MigrationTestCase{
		{
			Name: "Drops",
			Schema: &parser.Schema{
				Models: []*parser.Model{},
			},
			DBSchema: &DatabaseSchema{
				Tables: map[string]*TableInfo{
					"reviews": {
						Name: "reviews",
						Columns: map[string]*ColumnInfo{
							"id":      {Name: "id", Type: "text", IsPrimaryKey: true, IsNullable: false},
							"comment": {Name: "comment", Type: "text", IsNullable: true},
							"rating":  {Name: "rating", Type: "int", IsNullable: false},
						},
						Indexes: []*IndexInfo{
							{
								Name:      "idx_reviews_rating",
								TableName: "reviews",
								Columns:   []string{"rating"},
								IsUnique:  false,
							},
						},
					},
				},
			},
			ExpectedSQL: []string{
				`DROP TABLE "reviews";`,
			},
		},
		{
			Name: "Alter Drops",
			Schema: &parser.Schema{
				Models: []*parser.Model{
					{
						Name: "reviews",
						Fields: []*parser.ModelField{
							{Name: "id", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "id"}}},
						},
					},
				},
			},
			DBSchema: &DatabaseSchema{
				Tables: map[string]*TableInfo{
					"reviews": {
						Name: "reviews",
						Columns: map[string]*ColumnInfo{
							"id":      {Name: "id", Type: "text", IsPrimaryKey: true, IsNullable: false},
							"comment": {Name: "comment", Type: "text", IsNullable: true},
							"rating":  {Name: "rating", Type: "int", IsNullable: false},
						},
						Indexes: []*IndexInfo{
							{
								Name:      "idx_reviews_rating",
								TableName: "reviews",
								Columns:   []string{"rating"},
								IsUnique:  false,
							},
						},
					},
				},
			},
			ExpectedSQL: []string{
				`ALTER TABLE "reviews" DROP COLUMN "comment";`,
				`ALTER TABLE "reviews" DROP COLUMN "rating";`,
				`DROP INDEX "idx_reviews_rating";`,
			},
		},
		{
			Name: "Complex Schema",
			Schema: &parser.Schema{
				Models: []*parser.Model{
					{
						Name: "genres",
						Fields: []*parser.ModelField{
							{Name: "id", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "id"}, {Name: "default", Arguments: []*parser.AttributeArgument{{Value: map[string]interface{}{"function": "uuid"}}}}}},
							{Name: "name", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "unique"}}},
						},
					},
					{
						Name: "book_genres",
						Fields: []*parser.ModelField{
							{Name: "book_id", Type: &parser.FieldType{Name: "String"}},
							{Name: "genre_id", Type: &parser.FieldType{Name: "String"}},
						},
						Attributes: []*parser.Attribute{
							{Name: "id", Arguments: []*parser.AttributeArgument{{Value: []interface{}{"book_id", "genre_id"}}}},
						},
					},
				},
			},
			DBSchema: &DatabaseSchema{Tables: make(map[string]*TableInfo)},
			ExpectedSQL: []string{
				`CREATE UNIQUE INDEX "genres_name_key" ON "genres" ("name");`,
				`CONSTRAINT "book_genres_pkey" PRIMARY KEY ("book_id", "genre_id")`,
			},
		},
		{
			Name: "Reproduction: Full Schema Regression",
			Schema: &parser.Schema{
				Enums: []*parser.Enum{
					{Name: "BookCategoryType", Values: []*parser.EnumValue{{Name: "FICTION"}, {Name: "NON_FICTION"}}},
				},
				Models: []*parser.Model{
					{
						Name: "publishers",
						Fields: []*parser.ModelField{
							{Name: "id_publisher", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "id"}, {Name: "db.Uuid"}}},
						},
					},
					{
						Name: "book_categories",
						Fields: []*parser.ModelField{
							{
								Name: "id_category",
								Type: &parser.FieldType{Name: "String"},
								Attributes: []*parser.Attribute{
									{Name: "id"},
									{Name: "default", Arguments: []*parser.AttributeArgument{{Name: "value", Value: map[string]interface{}{"function": "dbgenerated", "args": []interface{}{"gen_random_uuid()"}}}}},
									{Name: "db.Uuid"},
								},
							},
							{Name: "id_publisher", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "db.Uuid"}}},
							{Name: "type", Type: &parser.FieldType{Name: "BookCategoryType"}},
							{Name: "metadata", Type: &parser.FieldType{Name: "Json"}},
							{
								Name: "created_at",
								Type: &parser.FieldType{Name: "DateTime"},
								Attributes: []*parser.Attribute{
									{Name: "default", Arguments: []*parser.AttributeArgument{{Name: "value", Value: map[string]interface{}{"function": "now"}}}},
									{Name: "db.Timestamp", Arguments: []*parser.AttributeArgument{{Name: "value", Value: "3"}}},
								},
							},
							{
								Name: "updated_at",
								Type: &parser.FieldType{Name: "DateTime"},
								Attributes: []*parser.Attribute{
									{Name: "default", Arguments: []*parser.AttributeArgument{{Name: "value", Value: map[string]interface{}{"function": "now"}}}},
									{Name: "db.Timestamp", Arguments: []*parser.AttributeArgument{{Name: "value", Value: "3"}}},
								},
							},
							{
								Name: "deleted_at",
								Type: &parser.FieldType{Name: "DateTime", IsOptional: true},
								Attributes: []*parser.Attribute{
									{Name: "db.Timestamp", Arguments: []*parser.AttributeArgument{{Name: "value", Value: "3"}}},
								},
							},
							{
								Name: "publisher",
								Type: &parser.FieldType{Name: "publishers"},
								Attributes: []*parser.Attribute{
									{Name: "relation", Arguments: []*parser.AttributeArgument{{Name: "fields", Value: []interface{}{"id_publisher"}}, {Name: "references", Value: []interface{}{"id_publisher"}}, {Name: "onDelete", Value: "Cascade"}}},
								},
							},
						},
						Attributes: []*parser.Attribute{
							{Name: "unique", Arguments: []*parser.AttributeArgument{{Name: "fields", Value: []interface{}{"id_publisher", "type"}}, {Name: "map", Value: "idx_book_categories_publisher_type_unique"}}},
							{Name: "index", Arguments: []*parser.AttributeArgument{{Name: "fields", Value: []interface{}{"id_publisher"}}, {Name: "map", Value: "idx_book_categories_id_publisher"}}},
							{Name: "index", Arguments: []*parser.AttributeArgument{{Name: "fields", Value: []interface{}{"type"}}, {Name: "map", Value: "idx_book_categories_type"}}},
						},
					},
					{
						Name: "editions",
						Fields: []*parser.ModelField{
							{Name: "id_edition", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "id"}, {Name: "default", Arguments: []*parser.AttributeArgument{{Value: map[string]interface{}{"function": "uuid"}}}}}},
							{Name: "id_publisher", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "unique"}, {Name: "db.Uuid"}}},
							// ... other fields simplified
						},
					},
				},
			},
			DBSchema: &DatabaseSchema{
				Tables: map[string]*TableInfo{
					"publishers": {
						Name: "publishers",
						Columns: map[string]*ColumnInfo{
							"id_publisher": {Name: "id_publisher", Type: "uuid", IsPrimaryKey: true, IsNullable: false},
						},
					},
					"book_categories": {
						Name: "book_categories",
						Columns: map[string]*ColumnInfo{
							"id_category":  {Name: "id_category", Type: "uuid", IsPrimaryKey: true, IsNullable: false},
							"id_publisher": {Name: "id_publisher", Type: "uuid", IsNullable: false},
							"type":         {Name: "type", Type: "text", IsNullable: false}, // User enum is text in DB
							"metadata":     {Name: "metadata", Type: "json", IsNullable: false},
							"created_at":   {Name: "created_at", Type: "timestamp", IsNullable: false},
							"updated_at":   {Name: "updated_at", Type: "timestamp", IsNullable: false},
							"deleted_at":   {Name: "deleted_at", Type: "timestamp", IsNullable: true},
						},
						Indexes: []*IndexInfo{
							{Name: "idx_book_categories_publisher_type_unique", TableName: "book_categories", Columns: []string{"id_publisher", "type"}, IsUnique: true},
							{Name: "idx_book_categories_id_publisher", TableName: "book_categories", Columns: []string{"id_publisher"}, IsUnique: false},
							{Name: "idx_book_categories_type", TableName: "book_categories", Columns: []string{"type"}, IsUnique: false},
						},
					},
					// editions is missing in DB (so it will be created)
				},
			},
			ExpectedSQL: []string{
				`CREATE TABLE "editions"`,
			},
			UnexpectedSQL: []string{
				`DROP INDEX "idx_book_categories_type"`,
			},
		},
	}

	for _, tc := range tests {
		runMigrationTest(t, tc)
	}
}
