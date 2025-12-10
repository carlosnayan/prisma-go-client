package migrations

import (
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// TestForeignKeyOnDeleteChange tests detection of onDelete changes
func TestForeignKeyOnDeleteChange(t *testing.T) {
	// Schema with onDelete: Cascade
	schema1 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Schema with onDelete: Restrict (changed)
	schema2 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Restrict"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Create initial database schema from schema1
	diff1, err := SchemaToSQL(schema1, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	// Simulate database state (introspect from schema1)
	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	// Create tables in dbSchema
	for _, table := range diff1.TablesToCreate {
		tableInfo := &TableInfo{
			Name:        table.Name,
			Columns:     make(map[string]*ColumnInfo),
			Indexes:     []*IndexInfo{},
			ForeignKeys: []*ForeignKeyInfo{},
		}
		for _, col := range table.Columns {
			tableInfo.Columns[col.Name] = &ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsNullable:   col.IsNullable,
				IsPrimaryKey: col.IsPrimaryKey,
				IsUnique:     col.IsUnique,
				DefaultValue: &col.DefaultValue,
			}
		}
		dbSchema.Tables[table.Name] = tableInfo
	}

	// Add foreign keys to database schema
	for _, fk := range diff1.ForeignKeysToCreate {
		dbTable := dbSchema.Tables[fk.TableName]
		if dbTable != nil {
			dbTable.ForeignKeys = append(dbTable.ForeignKeys, &ForeignKeyInfo{
				Name:              fk.Name,
				TableName:         fk.TableName,
				Columns:           fk.Columns,
				ReferencedTable:   fk.ReferencedTable,
				ReferencedColumns: fk.ReferencedColumns,
				OnDelete:          fk.OnDelete,
				OnUpdate:          fk.OnUpdate,
			})
		}
	}

	// Compare schema2 with database (should detect onDelete change)
	diff2, err := CompareSchema(schema2, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Should detect FK alteration
	if len(diff2.ForeignKeysToAlter) == 0 {
		// Check if it was added to create instead
		if len(diff2.ForeignKeysToCreate) > 0 {
			t.Logf("FK was added to ForeignKeysToCreate instead. FK: %+v", diff2.ForeignKeysToCreate[0])
		}
		t.Fatal("Expected ForeignKeysToAlter to contain the FK with changed onDelete, got none")
	}

	// Should not create new FK (it already exists, just needs alteration)
	if len(diff2.ForeignKeysToCreate) > 0 {
		t.Errorf("Expected no new FKs to create, got %d", len(diff2.ForeignKeysToCreate))
	}

	// Check that the altered FK has correct onDelete
	found := false
	for _, fk := range diff2.ForeignKeysToAlter {
		if fk.TableName == "users" && strings.Contains(fk.Name, "id_tenant") {
			if fk.OnDelete != "RESTRICT" {
				t.Errorf("Expected OnDelete RESTRICT, got %s", fk.OnDelete)
			}
			if fk.OnUpdate != "CASCADE" {
				t.Errorf("Expected OnUpdate CASCADE, got %s", fk.OnUpdate)
			}
			found = true
		}
	}
	if !found {
		t.Error("Expected to find FK alteration for users.id_tenant")
	}

	// Generate SQL and verify it contains DROP and ADD
	sql, err := GenerateMigrationSQL(diff2, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	// Should contain DROP CONSTRAINT for the FK
	if !strings.Contains(sql, `DROP CONSTRAINT`) {
		t.Error("Expected SQL to contain DROP CONSTRAINT for altered FK")
	}

	// Should contain ADD CONSTRAINT with new onDelete
	if !strings.Contains(sql, `ON DELETE RESTRICT`) {
		t.Errorf("Expected SQL to contain ON DELETE RESTRICT, got:\n%s", sql)
	}
}

// TestForeignKeyOnUpdateChange tests detection of onUpdate changes
func TestForeignKeyOnUpdateChange(t *testing.T) {
	// Schema with onUpdate: Cascade
	schema1 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Schema with onUpdate: NoAction (changed)
	schema2 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "NoAction"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Create initial database schema
	diff1, err := SchemaToSQL(schema1, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	for _, table := range diff1.TablesToCreate {
		tableInfo := &TableInfo{
			Name:        table.Name,
			Columns:     make(map[string]*ColumnInfo),
			Indexes:     []*IndexInfo{},
			ForeignKeys: []*ForeignKeyInfo{},
		}
		for _, col := range table.Columns {
			tableInfo.Columns[col.Name] = &ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsNullable:   col.IsNullable,
				IsPrimaryKey: col.IsPrimaryKey,
				IsUnique:     col.IsUnique,
				DefaultValue: &col.DefaultValue,
			}
		}
		dbSchema.Tables[table.Name] = tableInfo
	}

	for _, fk := range diff1.ForeignKeysToCreate {
		dbTable := dbSchema.Tables[fk.TableName]
		if dbTable != nil {
			dbTable.ForeignKeys = append(dbTable.ForeignKeys, &ForeignKeyInfo{
				Name:              fk.Name,
				TableName:         fk.TableName,
				Columns:           fk.Columns,
				ReferencedTable:   fk.ReferencedTable,
				ReferencedColumns: fk.ReferencedColumns,
				OnDelete:          fk.OnDelete,
				OnUpdate:          fk.OnUpdate,
			})
		}
	}

	// Compare schema2 with database
	diff2, err := CompareSchema(schema2, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Should detect FK alteration
	if len(diff2.ForeignKeysToAlter) == 0 {
		t.Fatal("Expected ForeignKeysToAlter to contain the FK with changed onUpdate, got none")
	}

	// Check that the altered FK has correct onUpdate
	found := false
	for _, fk := range diff2.ForeignKeysToAlter {
		if fk.TableName == "users" && strings.Contains(fk.Name, "id_tenant") {
			if fk.OnDelete != "CASCADE" {
				t.Errorf("Expected OnDelete CASCADE, got %s", fk.OnDelete)
			}
			if fk.OnUpdate != "NO ACTION" {
				t.Errorf("Expected OnUpdate NO ACTION, got %s", fk.OnUpdate)
			}
			found = true
		}
	}
	if !found {
		t.Error("Expected to find FK alteration for users.id_tenant")
	}

	// Generate SQL and verify
	sql, err := GenerateMigrationSQL(diff2, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	if !strings.Contains(sql, `ON UPDATE "NO ACTION"`) && !strings.Contains(sql, `ON UPDATE NO ACTION`) {
		t.Errorf("Expected SQL to contain ON UPDATE NO ACTION, got:\n%s", sql)
	}
}

// TestForeignKeyBothAttributesChange tests detection when both onDelete and onUpdate change
func TestForeignKeyBothAttributesChange(t *testing.T) {
	// Schema with Cascade/Cascade
	schema1 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Schema with Restrict/NoAction (both changed)
	schema2 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Restrict"},
								{Name: "onUpdate", Value: "NoAction"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Create initial database schema
	diff1, err := SchemaToSQL(schema1, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	for _, table := range diff1.TablesToCreate {
		tableInfo := &TableInfo{
			Name:        table.Name,
			Columns:     make(map[string]*ColumnInfo),
			Indexes:     []*IndexInfo{},
			ForeignKeys: []*ForeignKeyInfo{},
		}
		for _, col := range table.Columns {
			tableInfo.Columns[col.Name] = &ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsNullable:   col.IsNullable,
				IsPrimaryKey: col.IsPrimaryKey,
				IsUnique:     col.IsUnique,
				DefaultValue: &col.DefaultValue,
			}
		}
		dbSchema.Tables[table.Name] = tableInfo
	}

	for _, fk := range diff1.ForeignKeysToCreate {
		dbTable := dbSchema.Tables[fk.TableName]
		if dbTable != nil {
			dbTable.ForeignKeys = append(dbTable.ForeignKeys, &ForeignKeyInfo{
				Name:              fk.Name,
				TableName:         fk.TableName,
				Columns:           fk.Columns,
				ReferencedTable:   fk.ReferencedTable,
				ReferencedColumns: fk.ReferencedColumns,
				OnDelete:          fk.OnDelete,
				OnUpdate:          fk.OnUpdate,
			})
		}
	}

	// Compare schema2 with database
	diff2, err := CompareSchema(schema2, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Should detect FK alteration
	if len(diff2.ForeignKeysToAlter) == 0 {
		t.Fatal("Expected ForeignKeysToAlter to contain the FK with changed onDelete and onUpdate, got none")
	}

	// Check that the altered FK has correct values
	found := false
	for _, fk := range diff2.ForeignKeysToAlter {
		if fk.TableName == "users" && strings.Contains(fk.Name, "id_tenant") {
			if fk.OnDelete != "RESTRICT" {
				t.Errorf("Expected OnDelete RESTRICT, got %s", fk.OnDelete)
			}
			if fk.OnUpdate != "NO ACTION" {
				t.Errorf("Expected OnUpdate NO ACTION, got %s", fk.OnUpdate)
			}
			found = true
		}
	}
	if !found {
		t.Error("Expected to find FK alteration for users.id_tenant")
	}

	// Generate SQL and verify
	sql, err := GenerateMigrationSQL(diff2, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	if !strings.Contains(sql, `ON DELETE RESTRICT`) {
		t.Errorf("Expected SQL to contain ON DELETE RESTRICT, got:\n%s", sql)
	}
	if !strings.Contains(sql, `ON UPDATE "NO ACTION"`) && !strings.Contains(sql, `ON UPDATE NO ACTION`) {
		t.Errorf("Expected SQL to contain ON UPDATE NO ACTION, got:\n%s", sql)
	}
}

// TestForeignKeyNoChange tests that identical FK doesn't generate alteration
func TestForeignKeyNoChange(t *testing.T) {
	// Schema with Cascade/Cascade
	schema := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Create initial database schema
	diff1, err := SchemaToSQL(schema, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	for _, table := range diff1.TablesToCreate {
		tableInfo := &TableInfo{
			Name:        table.Name,
			Columns:     make(map[string]*ColumnInfo),
			Indexes:     []*IndexInfo{},
			ForeignKeys: []*ForeignKeyInfo{},
		}
		for _, col := range table.Columns {
			tableInfo.Columns[col.Name] = &ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsNullable:   col.IsNullable,
				IsPrimaryKey: col.IsPrimaryKey,
				IsUnique:     col.IsUnique,
				DefaultValue: &col.DefaultValue,
			}
		}
		dbSchema.Tables[table.Name] = tableInfo
	}

	for _, fk := range diff1.ForeignKeysToCreate {
		dbTable := dbSchema.Tables[fk.TableName]
		if dbTable != nil {
			dbTable.ForeignKeys = append(dbTable.ForeignKeys, &ForeignKeyInfo{
				Name:              fk.Name,
				TableName:         fk.TableName,
				Columns:           fk.Columns,
				ReferencedTable:   fk.ReferencedTable,
				ReferencedColumns: fk.ReferencedColumns,
				OnDelete:          fk.OnDelete,
				OnUpdate:          fk.OnUpdate,
			})
		}
	}

	// Compare same schema with database (should detect no changes)
	diff2, err := CompareSchema(schema, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Should NOT detect FK alteration
	if len(diff2.ForeignKeysToAlter) > 0 {
		t.Errorf("Expected no FK alterations for identical schema, got %d", len(diff2.ForeignKeysToAlter))
	}

	// Should NOT create new FK
	if len(diff2.ForeignKeysToCreate) > 0 {
		t.Errorf("Expected no new FKs to create, got %d", len(diff2.ForeignKeysToCreate))
	}
}

// TestForeignKeyRemovedFromSchema tests detection when FK is removed from schema
func TestForeignKeyRemovedFromSchema(t *testing.T) {
	// Schema with FK
	schema1 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Schema without FK (removed)
	schema2 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							// Relation removed
						},
					},
					// Tenant relation field removed
				},
			},
		},
	}

	// Create initial database schema
	diff1, err := SchemaToSQL(schema1, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	for _, table := range diff1.TablesToCreate {
		tableInfo := &TableInfo{
			Name:        table.Name,
			Columns:     make(map[string]*ColumnInfo),
			Indexes:     []*IndexInfo{},
			ForeignKeys: []*ForeignKeyInfo{},
		}
		for _, col := range table.Columns {
			tableInfo.Columns[col.Name] = &ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsNullable:   col.IsNullable,
				IsPrimaryKey: col.IsPrimaryKey,
				IsUnique:     col.IsUnique,
				DefaultValue: &col.DefaultValue,
			}
		}
		dbSchema.Tables[table.Name] = tableInfo
	}

	for _, fk := range diff1.ForeignKeysToCreate {
		dbTable := dbSchema.Tables[fk.TableName]
		if dbTable != nil {
			dbTable.ForeignKeys = append(dbTable.ForeignKeys, &ForeignKeyInfo{
				Name:              fk.Name,
				TableName:         fk.TableName,
				Columns:           fk.Columns,
				ReferencedTable:   fk.ReferencedTable,
				ReferencedColumns: fk.ReferencedColumns,
				OnDelete:          fk.OnDelete,
				OnUpdate:          fk.OnUpdate,
			})
		}
	}

	// Compare schema2 with database (FK removed from schema)
	diff2, err := CompareSchema(schema2, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Should detect FK for removal
	if len(diff2.ForeignKeysToDrop) == 0 {
		t.Fatal("Expected ForeignKeysToDrop to contain the removed FK, got none")
	}

	// Generate SQL and verify
	sql, err := GenerateMigrationSQL(diff2, "postgresql")
	if err != nil {
		t.Fatalf("GenerateMigrationSQL failed: %v", err)
	}

	if !strings.Contains(sql, `DROP CONSTRAINT`) {
		t.Error("Expected SQL to contain DROP CONSTRAINT for removed FK")
	}
}

// TestForeignKeyDefaultValues tests that default values (Cascade/Cascade) are handled correctly
func TestForeignKeyDefaultValues(t *testing.T) {
	// Schema without explicit onDelete/onUpdate (should default to Cascade)
	schema1 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								// No onDelete/onUpdate - should default to Cascade
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Create initial database schema
	diff1, err := SchemaToSQL(schema1, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	for _, table := range diff1.TablesToCreate {
		tableInfo := &TableInfo{
			Name:        table.Name,
			Columns:     make(map[string]*ColumnInfo),
			Indexes:     []*IndexInfo{},
			ForeignKeys: []*ForeignKeyInfo{},
		}
		for _, col := range table.Columns {
			tableInfo.Columns[col.Name] = &ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsNullable:   col.IsNullable,
				IsPrimaryKey: col.IsPrimaryKey,
				IsUnique:     col.IsUnique,
				DefaultValue: &col.DefaultValue,
			}
		}
		dbSchema.Tables[table.Name] = tableInfo
	}

	for _, fk := range diff1.ForeignKeysToCreate {
		dbTable := dbSchema.Tables[fk.TableName]
		if dbTable != nil {
			// Store with CASCADE (default)
			dbTable.ForeignKeys = append(dbTable.ForeignKeys, &ForeignKeyInfo{
				Name:              fk.Name,
				TableName:         fk.TableName,
				Columns:           fk.Columns,
				ReferencedTable:   fk.ReferencedTable,
				ReferencedColumns: fk.ReferencedColumns,
				OnDelete:          "CASCADE", // Default
				OnUpdate:          "CASCADE", // Default
			})
		}
	}

	// Compare same schema with database (should detect no changes)
	diff2, err := CompareSchema(schema1, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Should NOT detect FK alteration (defaults match)
	if len(diff2.ForeignKeysToAlter) > 0 {
		t.Errorf("Expected no FK alterations when defaults match, got %d", len(diff2.ForeignKeysToAlter))
	}
}

// TestForeignKeyFieldsReferencesChange tests detection when fields/references change
func TestForeignKeyFieldsReferencesChange(t *testing.T) {
	// Schema with fields: [id_tenant], references: [id_tenant]
	schema1 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
				Fields: []*parser.ModelField{
					{
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
							{Name: "db.Uuid"},
						},
					},
				},
			},
			{
				Name: "users",
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
						Name: "id_tenant",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"id_tenant"}},
								{Name: "references", Value: []interface{}{"id_tenant"}},
							}},
						},
					},
				},
			},
		},
	}

	// Schema with fields: [tenant_id], references: [id] (changed)
	schema2 := &parser.Schema{
		Models: []*parser.Model{
			{
				Name: "tenants",
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
			{
				Name: "users",
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
						Name: "tenant_id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "db.Uuid"},
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"tenant_id"}},
								{Name: "references", Value: []interface{}{"id"}},
								{Name: "onDelete", Value: "Cascade"},
								{Name: "onUpdate", Value: "Cascade"},
							}},
						},
					},
					{
						Name: "tenant",
						Type: &parser.FieldType{Name: "tenants"},
						Attributes: []*parser.Attribute{
							{Name: "relation", Arguments: []*parser.AttributeArgument{
								{Name: "fields", Value: []interface{}{"tenant_id"}},
								{Name: "references", Value: []interface{}{"id"}},
							}},
						},
					},
				},
			},
		},
	}

	// Create initial database schema
	diff1, err := SchemaToSQL(schema1, "postgresql")
	if err != nil {
		t.Fatalf("SchemaToSQL failed: %v", err)
	}

	dbSchema := &DatabaseSchema{
		Tables: make(map[string]*TableInfo),
	}

	for _, table := range diff1.TablesToCreate {
		tableInfo := &TableInfo{
			Name:        table.Name,
			Columns:     make(map[string]*ColumnInfo),
			Indexes:     []*IndexInfo{},
			ForeignKeys: []*ForeignKeyInfo{},
		}
		for _, col := range table.Columns {
			tableInfo.Columns[col.Name] = &ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsNullable:   col.IsNullable,
				IsPrimaryKey: col.IsPrimaryKey,
				IsUnique:     col.IsUnique,
				DefaultValue: &col.DefaultValue,
			}
		}
		dbSchema.Tables[table.Name] = tableInfo
	}

	for _, fk := range diff1.ForeignKeysToCreate {
		dbTable := dbSchema.Tables[fk.TableName]
		if dbTable != nil {
			dbTable.ForeignKeys = append(dbTable.ForeignKeys, &ForeignKeyInfo{
				Name:              fk.Name,
				TableName:         fk.TableName,
				Columns:           fk.Columns,
				ReferencedTable:   fk.ReferencedTable,
				ReferencedColumns: fk.ReferencedColumns,
				OnDelete:          fk.OnDelete,
				OnUpdate:          fk.OnUpdate,
			})
		}
	}

	// Compare schema2 with database
	diff2, err := CompareSchema(schema2, dbSchema, "postgresql")
	if err != nil {
		t.Fatalf("CompareSchema failed: %v", err)
	}

	// Should detect old FK for removal and new FK for creation
	// (since fields/references changed, it's a different FK)
	if len(diff2.ForeignKeysToDrop) == 0 {
		t.Error("Expected ForeignKeysToDrop to contain the old FK with different fields/references")
	}
	if len(diff2.ForeignKeysToCreate) == 0 {
		t.Error("Expected ForeignKeysToCreate to contain the new FK with different fields/references")
	}
}
