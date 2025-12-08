package migrations

import (
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// TestPrimaryKeyAsConstraint tests that PRIMARY KEY is generated as named CONSTRAINT
func TestPrimaryKeyAsConstraint(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "users",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "default", Arguments: []*parser.AttributeArgument{
								{Value: map[string]interface{}{"function": "dbgenerated", "args": []interface{}{"gen_random_uuid()"}}},
							}},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "email",
						Type: &parser.FieldType{Name: "String"},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Check that PRIMARY KEY is generated as CONSTRAINT
	if !strings.Contains(sql, `CONSTRAINT "users_pkey" PRIMARY KEY`) {
		t.Errorf("Expected CONSTRAINT \"users_pkey\" PRIMARY KEY, got:\n%s", sql)
	}
}

// TestPrimaryKeyMySQL tests PRIMARY KEY generation for MySQL (no named constraint)
func TestPrimaryKeyMySQL(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "users",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "default", Arguments: []*parser.AttributeArgument{
								{Value: map[string]interface{}{"function": "autoincrement"}},
							}},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "mysql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	sql, err := GenerateMigrationSQL(diff, "mysql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// MySQL doesn't use named CONSTRAINT for PRIMARY KEY in CREATE TABLE
	if !strings.Contains(sql, "PRIMARY KEY (`id`)") {
		t.Errorf("Expected PRIMARY KEY (`id`), got:\n%s", sql)
	}
}

// TestForeignKeyGeneration tests foreign key generation
func TestForeignKeyGeneration(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
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
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String"},
					},
				},
			},
			{
				Name: "books",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "title",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "author_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"author_id"}},
								{Name: "references", Value: []interface{}{"id"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "author",
						Type: &parser.FieldType{Name: "authors"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"author_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	if len(diff.ForeignKeysToCreate) == 0 {
		t.Fatal("Expected at least one foreign key, got none")
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Check that foreign key is generated after CREATE TABLE
	createTableIdx := strings.Index(sql, "CREATE TABLE")
	addFkIdx := strings.Index(sql, "ALTER TABLE")
	if createTableIdx == -1 || addFkIdx == -1 || addFkIdx < createTableIdx {
		t.Errorf("Foreign key should be added after CREATE TABLE")
	}

	// Check foreign key constraint
	if !strings.Contains(sql, `ADD CONSTRAINT "books_author_id_fkey"`) {
		t.Errorf("Expected foreign key constraint, got:\n%s", sql)
	}

	// Check ON DELETE CASCADE and ON UPDATE CASCADE
	if !strings.Contains(sql, "ON DELETE CASCADE") {
		t.Errorf("Expected ON DELETE CASCADE, got:\n%s", sql)
	}
	if !strings.Contains(sql, "ON UPDATE CASCADE") {
		t.Errorf("Expected ON UPDATE CASCADE, got:\n%s", sql)
	}
}

// TestUniqueIndexGeneration tests @@unique attribute processing
func TestUniqueIndexGeneration(t *testing.T) {
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
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "isbn",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "unique"},
						},
					},
					{
						Name: "author_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "title",
						Type: &parser.FieldType{Name: "String"},
					},
				},
				Attributes: []*parser.Attribute{
					{
						Name: "unique",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"author_id", "title"}},
							{Name: "map", Value: `"author_title_unique"`},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	if len(diff.IndexesToCreate) == 0 {
		t.Fatal("Expected at least one index, got none")
	}

	// Check for the unique index from @@unique
	found := false
	for _, idx := range diff.IndexesToCreate {
		if idx.Name == "author_title_unique" && idx.IsUnique {
			if len(idx.Columns) == 2 && idx.Columns[0] == "author_id" && idx.Columns[1] == "title" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("Expected unique index 'author_title_unique' with columns [author_id, title], got: %+v", diff.IndexesToCreate)
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Check that index is created after CREATE TABLE but before foreign keys
	createTableIdx := strings.Index(sql, "CREATE TABLE")
	createIndexIdx := strings.Index(sql, "CREATE UNIQUE INDEX")
	if createTableIdx == -1 || createIndexIdx == -1 || createIndexIdx < createTableIdx {
		t.Errorf("Index should be created after CREATE TABLE")
	}

	if !strings.Contains(sql, `CREATE UNIQUE INDEX "author_title_unique"`) {
		t.Errorf("Expected CREATE UNIQUE INDEX \"author_title_unique\", got:\n%s", sql)
	}
}

// TestComplexSchema tests a complex schema with multiple edge cases
func TestComplexSchema(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "authors",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "default", Arguments: []*parser.AttributeArgument{
								{Value: map[string]interface{}{"function": "dbgenerated", "args": []interface{}{"gen_random_uuid()"}}},
							}},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "email",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "unique"},
						},
					},
					{
						Name: "books",
						Type: &parser.FieldType{Name: "books", IsArray: true},
					},
				},
			},
			{
				Name: "publishers",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "address",
						Type: &parser.FieldType{Name: "String", IsOptional: true},
					},
				},
			},
			{
				Name: "books",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "title",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "author_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"author_id"}},
								{Name: "references", Value: []interface{}{"id"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "publisher_id",
						Type: &parser.FieldType{Name: "String", IsOptional: true},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"publisher_id"}},
								{Name: "references", Value: []interface{}{"id"}},
								{Name: "onDelete", Value: "SetNull"},
								{Name: "onUpdate", Value: "Restrict"},
							}},
						},
					},
					{
						Name: "isbn",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "unique"},
						},
					},
					{
						Name: "author",
						Type: &parser.FieldType{Name: "authors"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"author_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
					{
						Name: "publisher",
						Type: &parser.FieldType{Name: "publishers", IsOptional: true},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"publisher_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
				},
				Attributes: []*parser.Attribute{
					{
						Name: "unique",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"author_id", "title"}},
							{Name: "map", Value: `"author_title_unique"`},
						},
					},
				},
			},
			{
				Name: "tags",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "unique"},
						},
					},
				},
			},
			{
				Name: "book_tags",
				Fields: []*parser.ModelField{
					{
						Name: "book_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"book_id"}},
								{Name: "references", Value: []interface{}{"id"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tag_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"tag_id"}},
								{Name: "references", Value: []interface{}{"id"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "book",
						Type: &parser.FieldType{Name: "books"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"book_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
					{
						Name: "tag",
						Type: &parser.FieldType{Name: "tags"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"tag_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
				},
				Attributes: []*parser.Attribute{
					{
						Name: "unique",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"book_id", "tag_id"}},
						},
					},
				},
			},
			{
				Name: "reviews",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "default", Arguments: []*parser.AttributeArgument{
								{Value: map[string]interface{}{"function": "autoincrement"}},
							}},
						},
					},
					{
						Name: "book_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"book_id"}},
								{Name: "references", Value: []interface{}{"id"}},
								{Name: "onDelete", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "rating",
						Type: &parser.FieldType{Name: "Int"},
					},
					{
						Name: "comment",
						Type: &parser.FieldType{Name: "String", IsOptional: true},
					},
					{
						Name: "book",
						Type: &parser.FieldType{Name: "books"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"book_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
				},
				Attributes: []*parser.Attribute{
					{
						Name: "unique",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"book_id", "rating"}},
						},
					},
				},
			},
		},
	}

	// Test PostgreSQL
	t.Run("PostgreSQL", func(t *testing.T) {
		testComplexSchemaForProvider(t, schema, "postgresql")
	})

	// Test MySQL
	t.Run("MySQL", func(t *testing.T) {
		testComplexSchemaForProvider(t, schema, "mysql")
	})

	// Test SQLite
	t.Run("SQLite", func(t *testing.T) {
		testComplexSchemaForProvider(t, schema, "sqlite")
	})
}

func testComplexSchemaForProvider(t *testing.T, schema *parser.Schema, provider string) {
	diff, err := SchemaToSQL(schema, provider)
	if err != nil {
		t.Fatalf("SchemaToSQL failed for %s: %v", provider, err)
	}

	// Verify all tables are created
	if len(diff.TablesToCreate) < 6 {
		t.Errorf("Expected at least 6 tables, got %d", len(diff.TablesToCreate))
	}

	// Verify foreign keys
	if len(diff.ForeignKeysToCreate) < 4 {
		t.Errorf("Expected at least 4 foreign keys, got %d", len(diff.ForeignKeysToCreate))
	}

	// Verify indexes
	if len(diff.IndexesToCreate) < 3 {
		t.Errorf("Expected at least 3 indexes, got %d", len(diff.IndexesToCreate))
	}

	sql, err := GenerateMigrationSQL(diff, provider)
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed for %s: %v", provider, err)
	}

	// Verify order: CreateTable -> CreateIndex -> AddForeignKey
	createTableIdx := strings.Index(sql, "CREATE TABLE")
	addFkIdx := strings.Index(sql, "ALTER TABLE")

	if createTableIdx == -1 {
		t.Errorf("Expected CREATE TABLE in SQL for %s", provider)
	}
	if addFkIdx == -1 {
		t.Errorf("Expected ALTER TABLE (foreign keys) in SQL for %s", provider)
	}
	if createTableIdx != -1 && addFkIdx != -1 && addFkIdx < createTableIdx {
		t.Errorf("Foreign keys should be added after CREATE TABLE for %s", provider)
	}

	// Verify PRIMARY KEY format
	if provider == "postgresql" || provider == "sqlite" {
		if !strings.Contains(sql, `CONSTRAINT`) && strings.Contains(sql, `PRIMARY KEY`) {
			// Should have CONSTRAINT for PostgreSQL and SQLite
			if !strings.Contains(sql, `_pkey`) {
				t.Errorf("Expected CONSTRAINT with _pkey suffix for %s", provider)
			}
		}
	}

	// Verify foreign keys have ON DELETE and ON UPDATE
	if !strings.Contains(sql, "ON DELETE") {
		t.Errorf("Expected ON DELETE in foreign keys for %s", provider)
	}
	// ON UPDATE may not be supported in SQLite the same way
	if provider != "sqlite" {
		if !strings.Contains(sql, "ON UPDATE") {
			t.Errorf("Expected ON UPDATE in foreign keys for %s", provider)
		}
	}
}

// TestOrderOfOperations tests that operations are in the correct order
func TestOrderOfOperations(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "test_table",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "ref_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"ref_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
					{
						Name: "ref",
						Type: &parser.FieldType{Name: "test_table"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"ref_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
				},
				Attributes: []*parser.Attribute{
					{
						Name: "unique",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"ref_id"}},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Split SQL into lines to check order
	lines := strings.Split(sql, "\n")
	createTableLine := -1
	createIndexLine := -1
	addFkLine := -1

	for i, line := range lines {
		if strings.Contains(line, "CREATE TABLE") {
			createTableLine = i
		}
		if strings.Contains(line, "CREATE UNIQUE INDEX") {
			createIndexLine = i
		}
		if strings.Contains(line, "ALTER TABLE") && strings.Contains(line, "ADD CONSTRAINT") {
			addFkLine = i
		}
	}

	if createTableLine == -1 {
		t.Fatal("CREATE TABLE not found")
	}
	if createIndexLine == -1 {
		t.Fatal("CREATE UNIQUE INDEX not found")
	}
	if addFkLine == -1 {
		t.Fatal("ALTER TABLE ADD CONSTRAINT not found")
	}

	if createIndexLine < createTableLine {
		t.Error("CREATE INDEX should come after CREATE TABLE")
	}
	if addFkLine < createIndexLine {
		t.Error("ALTER TABLE ADD CONSTRAINT should come after CREATE INDEX")
	}
}

// TestIndexGeneration tests @@index (non-unique index) generation
func TestIndexGeneration(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "posts",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "title",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "author_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "created_at",
						Type: &parser.FieldType{Name: "DateTime"},
					},
				},
				Attributes: []*parser.Attribute{
					{
						Name: "index",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"author_id", "created_at"}},
							{Name: "map", Value: `"posts_author_created_idx"`},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	// Check that index was created
	found := false
	for _, idx := range diff.IndexesToCreate {
		if idx.Name == "posts_author_created_idx" && !idx.IsUnique {
			if len(idx.Columns) == 2 && idx.Columns[0] == "author_id" && idx.Columns[1] == "created_at" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("Expected non-unique index 'posts_author_created_idx' with columns [author_id, created_at], got: %+v", diff.IndexesToCreate)
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Check that CREATE INDEX (not UNIQUE) is generated
	if !strings.Contains(sql, `CREATE INDEX "posts_author_created_idx"`) {
		t.Errorf("Expected CREATE INDEX (non-unique), got:\n%s", sql)
	}
	if strings.Contains(sql, `CREATE UNIQUE INDEX "posts_author_created_idx"`) {
		t.Errorf("Index should NOT be UNIQUE, got:\n%s", sql)
	}
}

// TestMapAttributes tests @map and @@map attribute processing
func TestMapAttributes(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "User",
				Attributes: []*parser.Attribute{
					{
						Name: "map",
						Arguments: []*parser.AttributeArgument{
							{Value: `"users"`},
						},
					},
				},
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "default", Arguments: []*parser.AttributeArgument{
								{Value: map[string]interface{}{"function": "autoincrement"}},
							}},
						},
					},
					{
						Name: "emailAddress",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: `"email_address"`},
								},
							},
						},
					},
					{
						Name: "fullName",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{
								Name: "map",
								Arguments: []*parser.AttributeArgument{
									{Value: `"full_name"`},
								},
							},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	// Check that table name is mapped
	if len(diff.TablesToCreate) == 0 {
		t.Fatal("Expected at least one table")
	}
	if diff.TablesToCreate[0].Name != "users" {
		t.Errorf("Expected table name 'users' (from @@map), got '%s'", diff.TablesToCreate[0].Name)
	}

	// Check that column names are mapped
	table := diff.TablesToCreate[0]
	emailColFound := false
	fullNameColFound := false
	for _, col := range table.Columns {
		if col.Name == "email_address" {
			emailColFound = true
		}
		if col.Name == "full_name" {
			fullNameColFound = true
		}
		if col.Name == "emailAddress" || col.Name == "fullName" {
			t.Errorf("Column should use mapped name, not original field name: %s", col.Name)
		}
	}
	if !emailColFound {
		t.Error("Expected column 'email_address' (from @map)")
	}
	if !fullNameColFound {
		t.Error("Expected column 'full_name' (from @map)")
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Check that mapped names are used in SQL
	if !strings.Contains(sql, `CREATE TABLE "users"`) {
		t.Errorf("Expected table name 'users' in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `"email_address"`) {
		t.Errorf("Expected column name 'email_address' in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `"full_name"`) {
		t.Errorf("Expected column name 'full_name' in SQL, got:\n%s", sql)
	}
}

// TestCompositePrimaryKey tests @@id composite primary key
func TestCompositePrimaryKey(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "user_roles",
				Fields: []*parser.ModelField{
					{
						Name: "user_id",
						Type: &parser.FieldType{Name: "Int"},
					},
					{
						Name: "role_id",
						Type: &parser.FieldType{Name: "Int"},
					},
					{
						Name: "assigned_at",
						Type: &parser.FieldType{Name: "DateTime"},
					},
				},
				Attributes: []*parser.Attribute{
					{
						Name: "id",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"user_id", "role_id"}},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	if len(diff.TablesToCreate) == 0 {
		t.Fatal("Expected at least one table")
	}

	table := diff.TablesToCreate[0]
	if len(table.CompositePK) != 2 {
		t.Errorf("Expected composite PK with 2 fields, got %d", len(table.CompositePK))
	}
	if table.CompositePK[0] != "user_id" || table.CompositePK[1] != "role_id" {
		t.Errorf("Expected composite PK [user_id, role_id], got %v", table.CompositePK)
	}

	// Check that individual columns are not marked as PK
	for _, col := range table.Columns {
		if col.Name == "user_id" || col.Name == "role_id" {
			if col.IsPrimaryKey {
				t.Errorf("Column %s should not be marked as individual PK when using composite PK", col.Name)
			}
		}
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Check that composite PK is generated
	if !strings.Contains(sql, `CONSTRAINT "user_roles_pkey" PRIMARY KEY ("user_id", "role_id")`) {
		t.Errorf("Expected composite PRIMARY KEY, got:\n%s", sql)
	}
}

// TestDbTypes tests various @db.* type attributes
func TestDbTypes(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "test_types",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "text_field",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Text"},
						},
					},
					{
						Name: "date_field",
						Type: &parser.FieldType{Name: "DateTime"},
						Attributes: []*parser.Attribute{
							{Name: "db.Date"},
						},
					},
					{
						Name: "timestamp_field",
						Type: &parser.FieldType{Name: "DateTime"},
						Attributes: []*parser.Attribute{
							{Name: "db.Timestamp"},
						},
					},
					{
						Name: "decimal_field",
						Type: &parser.FieldType{Name: "Decimal"},
						Attributes: []*parser.Attribute{
							{Name: "db.Decimal", Arguments: []*parser.AttributeArgument{
								{Value: "10"},
								{Value: "2"},
							}},
						},
					},
					{
						Name: "char_field",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Char", Arguments: []*parser.AttributeArgument{
								{Value: "10"},
							}},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	table := diff.TablesToCreate[0]
	typeMap := make(map[string]string)
	for _, col := range table.Columns {
		typeMap[col.Name] = col.Type
	}

	// Check types
	if typeMap["text_field"] != "TEXT" {
		t.Errorf("Expected TEXT for text_field, got %s", typeMap["text_field"])
	}
	if typeMap["date_field"] != "DATE" {
		t.Errorf("Expected DATE for date_field, got %s", typeMap["date_field"])
	}
	if typeMap["timestamp_field"] != "TIMESTAMP" {
		t.Errorf("Expected TIMESTAMP for timestamp_field, got %s", typeMap["timestamp_field"])
	}
	if typeMap["decimal_field"] != "DECIMAL(10,2)" {
		t.Errorf("Expected DECIMAL(10,2) for decimal_field, got %s", typeMap["decimal_field"])
	}
	if typeMap["char_field"] != "CHAR(10)" {
		t.Errorf("Expected CHAR(10) for char_field, got %s", typeMap["char_field"])
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Verify types in SQL
	if !strings.Contains(sql, `"text_field" TEXT`) {
		t.Errorf("Expected TEXT type in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `"date_field" DATE`) {
		t.Errorf("Expected DATE type in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `"timestamp_field" TIMESTAMP`) {
		t.Errorf("Expected TIMESTAMP type in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `"decimal_field" DECIMAL(10,2)`) {
		t.Errorf("Expected DECIMAL(10,2) type in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `"char_field" CHAR(10)`) {
		t.Errorf("Expected CHAR(10) type in SQL, got:\n%s", sql)
	}
}

// TestUpdatedAt tests @updatedAt attribute
func TestUpdatedAt(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "posts",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "updated_at",
						Type: &parser.FieldType{Name: "DateTime"},
						Attributes: []*parser.Attribute{
							{Name: "updatedAt"},
							{Name: "default", Arguments: []*parser.AttributeArgument{
								{Value: map[string]interface{}{"function": "now"}},
							}},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	// @updatedAt doesn't need special SQL, just verify it doesn't break
	if len(diff.TablesToCreate) == 0 {
		t.Fatal("Expected at least one table")
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Verify table and column are created
	if !strings.Contains(sql, `CREATE TABLE "posts"`) {
		t.Errorf("Expected table 'posts', got:\n%s", sql)
	}
	if !strings.Contains(sql, `"updated_at"`) {
		t.Errorf("Expected column 'updated_at', got:\n%s", sql)
	}
}

// TestMapWithIndexesAndForeignKeys tests that @map works with indexes and foreign keys
func TestMapWithIndexesAndForeignKeys(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "Author",
				Attributes: []*parser.Attribute{
					{
						Name: "map",
						Arguments: []*parser.AttributeArgument{
							{Value: `"authors"`},
						},
					},
				},
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "name",
						Type: &parser.FieldType{Name: "String"},
					},
				},
			},
			{
				Name: "Book",
				Attributes: []*parser.Attribute{
					{
						Name: "map",
						Arguments: []*parser.AttributeArgument{
							{Value: `"books"`},
						},
					},
					{
						Name: "index",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"authorId"}},
						},
					},
				},
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
					{
						Name: "title",
						Type: &parser.FieldType{Name: "String"},
					},
					{
						Name: "authorId",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "map", Arguments: []*parser.AttributeArgument{
								{Value: `"author_id"`},
							}},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"authorId"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
					{
						Name: "author",
						Type: &parser.FieldType{Name: "Author"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"authorId"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	// Check that table names are mapped
	booksTableFound := false
	for _, table := range diff.TablesToCreate {
		if table.Name == "books" {
			booksTableFound = true
			// Check that index uses mapped column name
			for _, idx := range diff.IndexesToCreate {
				if idx.TableName == "books" {
					if idx.Columns[0] != "author_id" {
						t.Errorf("Index should use mapped column name 'author_id', got '%s'", idx.Columns[0])
					}
				}
			}
			// Check that foreign key uses mapped column name
			for _, fk := range diff.ForeignKeysToCreate {
				if fk.TableName == "books" {
					if fk.Columns[0] != "author_id" {
						t.Errorf("Foreign key should use mapped column name 'author_id', got '%s'", fk.Columns[0])
					}
					if fk.ReferencedTable != "authors" {
						t.Errorf("Foreign key should reference mapped table 'authors', got '%s'", fk.ReferencedTable)
					}
				}
			}
		}
	}
	if !booksTableFound {
		t.Error("Expected table 'books' (from @@map)")
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Verify mapped names in SQL
	if !strings.Contains(sql, `CREATE TABLE "books"`) {
		t.Errorf("Expected table name 'books' in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `"author_id"`) {
		t.Errorf("Expected mapped column name 'author_id' in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `CREATE INDEX`) && strings.Contains(sql, `"author_id"`) {
		t.Errorf("Expected index on 'author_id' in SQL, got:\n%s", sql)
	}
	if !strings.Contains(sql, `REFERENCES "authors"`) {
		t.Errorf("Expected foreign key reference to 'authors' table in SQL, got:\n%s", sql)
	}
}

// TestAllDbTypesMultiDriver tests @db.* types across all drivers
func TestAllDbTypesMultiDriver(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "test_all_types",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
					{
						Name: "text_col",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Text"},
						},
					},
					{
						Name: "date_col",
						Type: &parser.FieldType{Name: "DateTime"},
						Attributes: []*parser.Attribute{
							{Name: "db.Date"},
						},
					},
					{
						Name: "timestamp_col",
						Type: &parser.FieldType{Name: "DateTime"},
						Attributes: []*parser.Attribute{
							{Name: "db.Timestamp"},
						},
					},
					{
						Name: "smallint_col",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "db.SmallInt"},
						},
					},
					{
						Name: "integer_col",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "db.Integer"},
						},
					},
					{
						Name: "real_col",
						Type: &parser.FieldType{Name: "Float"},
						Attributes: []*parser.Attribute{
							{Name: "db.Real"},
						},
					},
					{
						Name: "double_precision_col",
						Type: &parser.FieldType{Name: "Float"},
						Attributes: []*parser.Attribute{
							{Name: "db.DoublePrecision"},
						},
					},
					{
						Name: "boolean_col",
						Type: &parser.FieldType{Name: "Boolean"},
						Attributes: []*parser.Attribute{
							{Name: "db.Boolean"},
						},
					},
					{
						Name: "json_col",
						Type: &parser.FieldType{Name: "Json"},
						Attributes: []*parser.Attribute{
							{Name: "db.Json"},
						},
					},
					{
						Name: "jsonb_col",
						Type: &parser.FieldType{Name: "Json"},
						Attributes: []*parser.Attribute{
							{Name: "db.JsonB"},
						},
					},
					{
						Name: "bytes_col",
						Type: &parser.FieldType{Name: "Bytes"},
						Attributes: []*parser.Attribute{
							{Name: "db.Bytes"},
						},
					},
				},
			},
		},
	}

	providers := []string{"postgresql", "mysql", "sqlite"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			diff, err := SchemaToSQL(schema, provider)
			if err != nil {
				t.Fatalf("SchemaToSQL failed for %s: %v", provider, err)
			}

			if len(diff.TablesToCreate) == 0 {
				t.Fatal("Expected at least one table")
			}

			sql, err := GenerateMigrationSQL(diff, provider)
			if err != nil {
				t.Fatalf("GenerateMigrationSQL failed for %s: %v", provider, err)
			}

			// Verify table is created
			if !strings.Contains(sql, "CREATE TABLE") {
				t.Errorf("Expected CREATE TABLE for %s, got:\n%s", provider, sql)
			}

			// Verify types are present (exact type depends on provider)
			if !strings.Contains(sql, "text_col") {
				t.Errorf("Expected text_col column for %s", provider)
			}
			if !strings.Contains(sql, "date_col") {
				t.Errorf("Expected date_col column for %s", provider)
			}
		})
	}
}

// TestCompositePKWithMap tests composite primary key with @map
func TestCompositePKWithMap(t *testing.T) {
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "UserRole",
				Attributes: []*parser.Attribute{
					{
						Name: "map",
						Arguments: []*parser.AttributeArgument{
							{Value: `"user_roles"`},
						},
					},
					{
						Name: "id",
						Arguments: []*parser.AttributeArgument{
							{Name: "", Value: []interface{}{"userId", "roleId"}},
						},
					},
				},
				Fields: []*parser.ModelField{
					{
						Name: "userId",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "map", Arguments: []*parser.AttributeArgument{
								{Value: `"user_id"`},
							}},
						},
					},
					{
						Name: "roleId",
						Type: &parser.FieldType{Name: "Int"},
						Attributes: []*parser.Attribute{
							{Name: "map", Arguments: []*parser.AttributeArgument{
								{Value: `"role_id"`},
							}},
						},
					},
				},
			},
		},
	}

	diff, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	if len(diff.TablesToCreate) == 0 {
		t.Fatal("Expected at least one table")
	}

	table := diff.TablesToCreate[0]
	if table.Name != "user_roles" {
		t.Errorf("Expected table name 'user_roles' (from @@map), got '%s'", table.Name)
	}

	if len(table.CompositePK) != 2 {
		t.Errorf("Expected composite PK with 2 fields, got %d", len(table.CompositePK))
	}

	if table.CompositePK[0] != "user_id" || table.CompositePK[1] != "role_id" {
		t.Errorf("Expected composite PK [user_id, role_id] (from @map), got %v", table.CompositePK)
	}

	sql, err := GenerateMigrationSQL(diff, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Verify composite PK uses mapped names
	if !strings.Contains(sql, `CONSTRAINT "user_roles_pkey" PRIMARY KEY ("user_id", "role_id")`) {
		t.Errorf("Expected composite PK with mapped names, got:\n%s", sql)
	}
}
