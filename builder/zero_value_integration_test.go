package builder

import (
	"context"
	"reflect"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

func TestCreate_WithBooleanFalse(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_boolean (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						is_active BOOLEAN DEFAULT TRUE
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_boolean (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						is_active BOOLEAN DEFAULT TRUE
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_boolean (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL,
						is_active INTEGER DEFAULT 1
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			type TestRecord struct {
				ID       int    `db:"id"`
				Name     string `db:"name"`
				IsActive bool   `db:"is_active"`
			}

			builder := NewTableQueryBuilder(db, "test_boolean", []string{"id", "name", "is_active"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			fields := map[string]interface{}{
				"name":      "Test Disabled",
				"is_active": false,
			}

			created, err := builder.CreateFromFields(ctx, fields)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			if provider == "sqlite" {
				found, err := builder.FindFirst(ctx, Where{"name": "Test Disabled"})
				if err != nil {
					t.Fatalf("FindFirst failed: %v", err)
				}
				foundRecord := found.(TestRecord)
				if foundRecord.IsActive != false {
					t.Errorf("Expected IsActive=false, got IsActive=%v", foundRecord.IsActive)
				}
			} else {
				createdRecord := created.(TestRecord)
				if createdRecord.IsActive != false {
					t.Errorf("Expected IsActive=false, got IsActive=%v", createdRecord.IsActive)
				}
			}
		})
	}
}

func TestUpdate_WithZeroValues(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_update (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						count INT DEFAULT 100,
						is_active BOOLEAN DEFAULT TRUE
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_update (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						count INT DEFAULT 100,
						is_active BOOLEAN DEFAULT TRUE
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_update (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL,
						count INTEGER DEFAULT 100,
						is_active INTEGER DEFAULT 1
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			type TestRecord struct {
				ID       int    `db:"id"`
				Name     string `db:"name"`
				Count    int    `db:"count"`
				IsActive bool   `db:"is_active"`
			}

			builder := NewTableQueryBuilder(db, "test_update", []string{"id", "name", "count", "is_active"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			record := TestRecord{
				Name:     "Initial",
				Count:    100,
				IsActive: true,
			}

			created, err := builder.Create(ctx, record)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			var recordID int
			if provider == "sqlite" {
				found, _ := builder.FindFirst(ctx, Where{"name": "Initial"})
				recordID = found.(TestRecord).ID
			} else {
				recordID = created.(TestRecord).ID
			}

			updateFields := map[string]interface{}{
				"count":     0,
				"is_active": false,
			}

			updated, err := builder.UpdateFromFields(ctx, recordID, updateFields)
			if err != nil {
				t.Fatalf("Update failed: %v", err)
			}

			var updatedRecord TestRecord
			if provider == "sqlite" {
				found, _ := builder.FindFirst(ctx, Where{"id": recordID})
				updatedRecord = found.(TestRecord)
			} else {
				updatedRecord = updated.(TestRecord)
			}

			if updatedRecord.Count != 0 {
				t.Errorf("Expected Count=0, got Count=%d", updatedRecord.Count)
			}
			if updatedRecord.IsActive != false {
				t.Errorf("Expected IsActive=false, got IsActive=%v", updatedRecord.IsActive)
			}
		})
	}
}
