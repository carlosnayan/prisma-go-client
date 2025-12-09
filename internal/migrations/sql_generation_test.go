package migrations

import (
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func TestGenerateMigrationSQL_Generic(t *testing.T) {
	// Setup generic schema: Books <-> Authors
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "books",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "default", Arguments: []*parser.AttributeArgument{{Value: map[string]interface{}{"function": "uuid"}}}},
						},
					},
					{
						Name: "author_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "unique"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "title",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "published_at",
						Type: &parser.FieldType{Name: "DateTime", IsOptional: true},
					},
					{
						Name: "created_at",
						Type: &parser.FieldType{Name: "DateTime", IsOptional: true},
						Attributes: []*parser.Attribute{
							{Name: "default", Arguments: []*parser.AttributeArgument{{Value: map[string]interface{}{"function": "now"}}}},
						},
					},
					{
						Name: "updated_at",
						Type: &parser.FieldType{Name: "DateTime", IsOptional: true},
						Attributes: []*parser.Attribute{
							{Name: "default", Arguments: []*parser.AttributeArgument{{Value: map[string]interface{}{"function": "now"}}}},
						},
					},
				},
			},
			{
				Name: "authors",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
		},
	}

	// Manually add the relation to books
	booksModel := schema.Models[0]
	booksModel.Fields = append(booksModel.Fields, &parser.ModelField{
		Name: "author",
		Type: &parser.FieldType{Name: "authors"},
		Attributes: []*parser.Attribute{
			{
				Name: "relation",
				Arguments: []*parser.AttributeArgument{
					{Name: "fields", Value: []interface{}{"author_id"}},
					{Name: "references", Value: []interface{}{"id"}},
					{Name: "onDelete", Value: "Cascade"},
				},
			},
		},
	})

	// Convert schema to SQL
	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	// Generate SQL
	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Log SQL for debugging
	t.Logf("Generated SQL:\n%s", sql)

	// Assertions

	// 1. Check for gen_random_uuid() usage (Should NOT be present)
	if strings.Contains(sql, "gen_random_uuid()") {
		t.Errorf("SQL contains 'gen_random_uuid()', but should not")
	}

	// 2. Check for pgcrypto extension (Should NOT be present)
	if strings.Contains(sql, "CREATE EXTENSION") {
		t.Errorf("SQL contains 'CREATE EXTENSION'")
	}

	// 3. Check for separate CREATE UNIQUE INDEX
	if !strings.Contains(sql, `CREATE UNIQUE INDEX "books_author_id_key"`) {
		t.Errorf("SQL missing 'CREATE UNIQUE INDEX \"books_author_id_key\"'")
	}

	// 4. Check for Table Creation
	if !strings.Contains(sql, `CREATE TABLE "books"`) {
		t.Error("SQL missing CREATE TABLE \"books\"")
	}

	// 5. Check foreign key
	if !strings.Contains(sql, `ALTER TABLE "books" ADD CONSTRAINT "books_author_id_fkey" FOREIGN KEY ("author_id") REFERENCES "authors" ("id")`) {
		t.Error("SQL missing correct FOREIGN KEY constraint")
	}
}

func TestCompareSchema_Generic(t *testing.T) {
	// Setup generic schema: Books <-> Authors (Same as above)
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "books",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "default", Arguments: []*parser.AttributeArgument{{Value: map[string]interface{}{"function": "uuid"}}}},
						},
					},
					{
						Name: "author_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "unique"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "title",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "published_at",
						Type: &parser.FieldType{Name: "DateTime", IsOptional: true},
					},
					{
						Name: "created_at",
						Type: &parser.FieldType{Name: "DateTime", IsOptional: true},
						Attributes: []*parser.Attribute{
							{Name: "default", Arguments: []*parser.AttributeArgument{{Value: map[string]interface{}{"function": "now"}}}},
						},
					},
					{
						Name: "updated_at",
						Type: &parser.FieldType{Name: "DateTime", IsOptional: true},
						Attributes: []*parser.Attribute{
							{Name: "default", Arguments: []*parser.AttributeArgument{{Value: map[string]interface{}{"function": "now"}}}},
						},
					},
				},
			},
			{
				Name: "authors",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
		},
	}

	// Manually add the relation to books
	booksModel := schema.Models[0]
	booksModel.Fields = append(booksModel.Fields, &parser.ModelField{
		Name: "author",
		Type: &parser.FieldType{Name: "authors"},
		Attributes: []*parser.Attribute{
			{
				Name: "relation",
				Arguments: []*parser.AttributeArgument{
					{Name: "fields", Value: []interface{}{"author_id"}},
					{Name: "references", Value: []interface{}{"id"}},
					{Name: "onDelete", Value: "Cascade"},
				},
			},
		},
	})

	// Setup empty database schema (simulating fresh DB)
	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	// Compare schema (this mimics migrate dev's diffing process)
	diff, err := CompareSchema(schema, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Generate SQL
	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Log SQL for debugging
	t.Logf("Generated SQL (CompareSchema):\n%s", sql)

	// Assertions

	// 1. Check for gen_random_uuid() usage (Should NOT be present)
	if strings.Contains(sql, "gen_random_uuid()") {
		t.Errorf("SQL contains 'gen_random_uuid()', but should not")
	}

	// 2. Check for pgcrypto extension (Should NOT be present)
	if strings.Contains(sql, "CREATE EXTENSION") {
		t.Errorf("SQL contains 'CREATE EXTENSION'")
	}

	// 3. Check for separate CREATE UNIQUE INDEX
	if !strings.Contains(sql, `CREATE UNIQUE INDEX "books_author_id_key"`) {
		t.Errorf("SQL missing 'CREATE UNIQUE INDEX \"books_author_id_key\"'")
	}

	// 4. Check for comments
	if !strings.Contains(sql, "-- CreateTable") {
		t.Errorf("SQL missing '-- CreateTable' comment")
	}
	if !strings.Contains(sql, "-- CreateIndex") {
		t.Errorf("SQL missing '-- CreateIndex' comment")
	}
	if !strings.Contains(sql, "-- AddForeignKey") {
		t.Errorf("SQL missing '-- AddForeignKey' comment")
	}
}

func TestCompareSchema_CompositeIndexes(t *testing.T) {
	// Setup schema with composite/named indexes
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "books",
				Fields: []*parser.ModelField{
					{Name: "id", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "id"}}},
				},
			},
			{
				Name: "authors",
				Fields: []*parser.ModelField{
					{Name: "id", Type: &parser.FieldType{Name: "String"}, Attributes: []*parser.Attribute{{Name: "id"}}},
				},
			},
			{
				Name: "chatbot_variables",
				Fields: []*parser.ModelField{
					{
						Name: "id_chatbot_flow",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String"},
					},
				},
				Attributes: []*parser.Attribute{
					{
						Name: "unique",
						Arguments: []*parser.AttributeArgument{
							{Value: []interface{}{"id_chatbot_flow", "name"}},
							{Name: "map", Value: "chatbot_variables_unique_name_per_flow"},
						},
					},
					{
						Name: "index",
						Arguments: []*parser.AttributeArgument{
							{Value: []interface{}{"id_chatbot_flow"}},
							{Name: "map", Value: "idx_chatbot_variables_id_chatbot_flow"},
						},
					},
				},
			},
		},
	}

	// Setup empty database schema
	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	// Compare
	diff, err := CompareSchema(schema, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Generate SQL
	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	t.Logf("Generated SQL (CompositeIndexes):\n%s", sql)

	// Assertions
	// Expected:
	// -- CreateIndex
	// CREATE INDEX "idx_chatbot_variables_id_chatbot_flow" ON "chatbot_variables"("id_chatbot_flow");
	// -- CreateIndex
	// CREATE UNIQUE INDEX "chatbot_variables_unique_name_per_flow" ON "chatbot_variables"("id_chatbot_flow", "name");

	if !strings.Contains(sql, `CREATE INDEX "idx_chatbot_variables_id_chatbot_flow" ON "chatbot_variables" ("id_chatbot_flow")`) {
		t.Errorf("SQL missing named index 'idx_chatbot_variables_id_chatbot_flow'")
	}

	if !strings.Contains(sql, `CREATE UNIQUE INDEX "chatbot_variables_unique_name_per_flow" ON "chatbot_variables" ("id_chatbot_flow", "name")`) {
		t.Errorf("SQL missing named unique index 'chatbot_variables_unique_name_per_flow'")
	}
}
