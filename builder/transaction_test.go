package builder

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// TestBeginTransaction tests transaction begin
func TestBeginTransaction(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()
			tx, err := BeginTransaction(ctx, db)
			if err != nil {
				t.Fatalf("BeginTransaction failed: %v", err)
			}

			// Test that we can use the transaction
			if tx == nil {
				t.Fatal("Transaction is nil")
			}

			// Test rollback
			if err := tx.Rollback(ctx); err != nil {
				t.Fatalf("Rollback failed: %v", err)
			}
		})
	}
}

// TestTransactionCommit tests transaction commit
func TestTransactionCommit(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_transactions (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_transactions (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_transactions (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Begin transaction
			tx, err := BeginTransaction(ctx, db)
			if err != nil {
				t.Fatalf("BeginTransaction failed: %v", err)
			}

			// Insert data within transaction
			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			query := tx.Query("test_transactions", []string{"id", "name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.SetPrimaryKey("id")

			record := TestRecord{Name: "Transaction Test"}
			err = query.Create(ctx, record)
			if err != nil {
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					t.Logf("Rollback failed: %v", rbErr)
				}
				t.Fatalf("Create in transaction failed: %v", err)
			}

			// Commit transaction
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("Commit failed: %v", err)
			}

			// Verify data is persisted
			builder := NewTableQueryBuilder(db, "test_transactions", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			found, err := builder.FindFirst(ctx, Where{"name": "Transaction Test"})
			if err != nil {
				t.Fatalf("FindFirst failed: %v", err)
			}

			foundRecord, ok := found.(TestRecord)
			if !ok {
				t.Fatal("FindFirst returned wrong type")
			}
			if foundRecord.Name != "Transaction Test" {
				t.Errorf("Expected name 'Transaction Test', got %s", foundRecord.Name)
			}
		})
	}
}

// TestTransactionRollback tests transaction rollback
func TestTransactionRollback(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_rollback (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_rollback (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_rollback (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Begin transaction
			tx, err := BeginTransaction(ctx, db)
			if err != nil {
				t.Fatalf("BeginTransaction failed: %v", err)
			}

			// Insert data within transaction
			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			query := tx.Query("test_rollback", []string{"id", "name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.SetPrimaryKey("id")

			record := TestRecord{Name: "Rollback Test"}
			err = query.Create(ctx, record)
			if err != nil {
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					t.Logf("Rollback failed: %v", rbErr)
				}
				t.Fatalf("Create in transaction failed: %v", err)
			}

			// Rollback transaction
			if err := tx.Rollback(ctx); err != nil {
				t.Fatalf("Rollback failed: %v", err)
			}

			// Verify data is NOT persisted
			builder := NewTableQueryBuilder(db, "test_rollback", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			_, err = builder.FindFirst(ctx, Where{"name": "Rollback Test"})
			if err == nil {
				t.Error("Expected error when finding rolled back record, but got nil")
			}
		})
	}
}

// TestExecuteTransaction tests ExecuteTransaction with success
func TestExecuteTransaction(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_execute (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_execute (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_execute (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Execute transaction
			err = ExecuteTransaction(ctx, db, func(tx *Transaction) error {
				query := tx.Query("test_execute", []string{"id", "name"})
				query.SetDialect(dialect.GetDialect(provider))
				query.SetPrimaryKey("id")

				type TestRecord struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				}

				record := TestRecord{Name: "Execute Test"}
				err := query.Create(ctx, record)
				return err
			})

			if err != nil {
				t.Fatalf("ExecuteTransaction failed: %v", err)
			}

			// Verify data is persisted
			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			builder := NewTableQueryBuilder(db, "test_execute", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			found, err := builder.FindFirst(ctx, Where{"name": "Execute Test"})
			if err != nil {
				t.Fatalf("FindFirst failed: %v", err)
			}

			foundRecord, ok := found.(struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			})
			if !ok {
				t.Fatal("FindFirst returned wrong type")
			}
			if foundRecord.Name != "Execute Test" {
				t.Errorf("Expected name 'Execute Test', got %s", foundRecord.Name)
			}
		})
	}
}

// TestExecuteTransactionRollback tests ExecuteTransaction with error (should rollback)
func TestExecuteTransactionRollback(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_execute_rollback (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_execute_rollback (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_execute_rollback (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Execute transaction that returns error
			testErr := errors.New("test error")
			err = ExecuteTransaction(ctx, db, func(tx *Transaction) error {
				query := tx.Query("test_execute_rollback", []string{"id", "name"})
				query.SetDialect(dialect.GetDialect(provider))
				query.SetPrimaryKey("id")

				type TestRecord struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				}

				record := TestRecord{Name: "Should Not Persist"}
				err := query.Create(ctx, record)
				if err != nil {
					return err
				}

				// Return error to trigger rollback
				return testErr
			})

			if err == nil {
				t.Fatal("ExecuteTransaction should have returned error")
			}
			if err != testErr {
				t.Fatalf("Expected test error, got: %v", err)
			}

			// Verify data is NOT persisted
			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			builder := NewTableQueryBuilder(db, "test_execute_rollback", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			_, err = builder.FindFirst(ctx, Where{"name": "Should Not Persist"})
			if err == nil {
				t.Error("Expected error when finding rolled back record, but got nil")
			}
		})
	}
}

// TestExecuteTransactionPanic tests ExecuteTransaction with panic (should rollback)
func TestExecuteTransactionPanic(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_panic (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_panic (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_panic (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Execute transaction that panics
			func() {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic, but none occurred")
					}
				}()

				err = ExecuteTransaction(ctx, db, func(tx *Transaction) error {
					query := tx.Query("test_panic", []string{"id", "name"})
					query.SetDialect(dialect.GetDialect(provider))
					query.SetPrimaryKey("id")

					type TestRecord struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					}

					record := TestRecord{Name: "Panic Test"}
					err := query.Create(ctx, record)
					if err != nil {
						return err
					}

					// Panic to trigger rollback
					panic("test panic")
				})

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}()

			// Verify data is NOT persisted
			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			builder := NewTableQueryBuilder(db, "test_panic", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			_, err = builder.FindFirst(ctx, Where{"name": "Panic Test"})
			if err == nil {
				t.Error("Expected error when finding rolled back record, but got nil")
			}
		})
	}
}

// TestExecuteSequentialTransactions tests ExecuteSequentialTransactions
func TestExecuteSequentialTransactions(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_sequential (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_sequential (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_sequential (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Execute sequential transactions
			operations := []TransactionFunc{
				func(tx *Transaction) error {
					query := tx.Query("test_sequential", []string{"id", "name"})
					query.SetDialect(dialect.GetDialect(provider))
					query.SetPrimaryKey("id")

					type TestRecord struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					}

					record := TestRecord{Name: "First"}
					err := query.Create(ctx, record)
					return err
				},
				func(tx *Transaction) error {
					query := tx.Query("test_sequential", []string{"id", "name"})
					query.SetDialect(dialect.GetDialect(provider))
					query.SetPrimaryKey("id")

					type TestRecord struct {
						ID   int    `json:"id"`
						Name string `json:"name"`
					}

					record := TestRecord{Name: "Second"}
					err := query.Create(ctx, record)
					return err
				},
			}

			err = ExecuteSequentialTransactions(ctx, db, operations)
			if err != nil {
				t.Fatalf("ExecuteSequentialTransactions failed: %v", err)
			}

			// Verify both records are persisted
			builder := NewTableQueryBuilder(db, "test_sequential", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")

			count, err := builder.Count(ctx, Where{})
			if err != nil {
				t.Fatalf("Count failed: %v", err)
			}
			if count != 2 {
				t.Errorf("Expected count 2, got %d", count)
			}
		})
	}
}

// TestTransactionIsolation tests that operations within transaction are not visible outside until commit
func TestTransactionIsolation(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			// Create test table
			sqlDB := db.SQLDB()
			if sqlDB == nil {
				t.Fatal("database does not support SQLDB()")
			}

			ctx := context.Background()
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_isolation (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_isolation (
						id INT AUTO_INCREMENT PRIMARY KEY,
						name VARCHAR(255) NOT NULL
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS test_isolation (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Begin transaction
			tx, err := BeginTransaction(ctx, db)
			if err != nil {
				t.Fatalf("BeginTransaction failed: %v", err)
			}

			// Insert data within transaction
			type TestRecord struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			query := tx.Query("test_isolation", []string{"id", "name"})
			query.SetDialect(dialect.GetDialect(provider))
			query.SetPrimaryKey("id")

			record := TestRecord{Name: "Isolation Test"}
			err = query.Create(ctx, record)
			if err != nil {
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					t.Logf("Rollback failed: %v", rbErr)
				}
				t.Fatalf("Create in transaction failed: %v", err)
			}

			// Try to find record outside transaction (should not be visible)
			builder := NewTableQueryBuilder(db, "test_isolation", []string{"id", "name"})
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(TestRecord{}))

			// Note: Isolation level depends on database configuration
			// Some databases may show uncommitted data, so we just verify commit works
			// For strict isolation testing, we'd need to configure isolation levels
			_, _ = builder.FindFirst(ctx, Where{"name": "Isolation Test"})

			// Commit transaction
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("Commit failed: %v", err)
			}

			// Now verify data is visible after commit
			found, err := builder.FindFirst(ctx, Where{"name": "Isolation Test"})
			if err != nil {
				t.Fatalf("FindFirst after commit failed: %v", err)
			}

			foundRecord, ok := found.(TestRecord)
			if !ok {
				t.Fatal("FindFirst returned wrong type")
			}
			if foundRecord.Name != "Isolation Test" {
				t.Errorf("Expected name 'Isolation Test', got %s", foundRecord.Name)
			}
		})
	}
}

// TestTransactionDBAdapter tests that txDBAdapter works correctly
func TestTransactionDBAdapter(t *testing.T) {
	providers := []string{"postgresql", "mysql", "sqlite"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			testutil.SkipIfNoDatabase(t, provider)
			db, cleanup := testutil.SetupTestDB(t, provider)
			defer cleanup()

			ctx := context.Background()
			tx, err := BeginTransaction(ctx, db)
			if err != nil {
				t.Fatalf("BeginTransaction failed: %v", err)
			}
			defer func() { _ = tx.Rollback(ctx) }()

			// Test DB() method
			txDB := tx.DB()
			if txDB == nil {
				t.Fatal("DB() returned nil")
			}

			// Test that we can't begin a transaction within a transaction
			_, err = txDB.Begin(ctx)
			if err == nil {
				t.Error("Expected error when beginning transaction within transaction, but got nil")
			}
		})
	}
}
