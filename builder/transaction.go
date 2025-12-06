package builder

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/carlosnayan/prisma-go-client/internal/driver"
	"github.com/carlosnayan/prisma-go-client/internal/errors"
	contextutil "github.com/carlosnayan/prisma-go-client/internal/context"
)

// Transaction represents a database transaction
type Transaction struct {
	tx driver.Tx
}

// TransactionFunc is a function that executes within a transaction
type TransactionFunc func(*Transaction) error

// BeginTransaction starts a new database transaction
func BeginTransaction(ctx context.Context, db DBTX) (*Transaction, error) {
	// Add timeout for transactions
	ctx, cancel := contextutil.WithTransactionTimeout(ctx)
	defer cancel()

	// driver.DB already has Begin method that returns driver.Tx
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, errors.WrapError(err, "failed to begin transaction")
	}
	return &Transaction{tx: tx}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// Query creates a new Query using the transaction
// Note: driver.Tx does not implement driver.DB directly, so we need to create an adapter
func (t *Transaction) Query(table string, columns []string) *Query {
	// Create an adapter that implements driver.DB using driver.Tx
	txAdapter := &txDBAdapter{tx: t.tx}
	return NewQuery(txAdapter, table, columns)
}

// txDBAdapter adapts driver.Tx to driver.DB
type txDBAdapter struct {
	tx driver.Tx
}

// TxDBAdapter is exported for use in generated code
type TxDBAdapter = txDBAdapter

// NewTxDBAdapter creates a new adapter from a Transaction
func (t *Transaction) DB() driver.DB {
	return &txDBAdapter{tx: t.tx}
}

func (a *txDBAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (driver.Result, error) {
	return a.tx.Exec(ctx, sql, args...)
}

func (a *txDBAdapter) Query(ctx context.Context, sql string, args ...interface{}) (driver.Rows, error) {
	return a.tx.Query(ctx, sql, args...)
}

func (a *txDBAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) driver.Row {
	return a.tx.QueryRow(ctx, sql, args...)
}

func (a *txDBAdapter) Begin(ctx context.Context) (driver.Tx, error) {
	return nil, fmt.Errorf("cannot begin a transaction within a transaction")
}

func (a *txDBAdapter) SQLDB() *sql.DB {
	return nil
}

// ExecuteTransaction executes a function within a transaction
// If the function returns an error, the transaction is automatically rolled back
func ExecuteTransaction(ctx context.Context, db DBTX, fn TransactionFunc) error {
	tx, err := BeginTransaction(ctx, db)
	if err != nil {
		return err
	}
	
	// Always rollback at the end, unless commit is successful
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
	}()
	
	// Execute function
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	
	// Commit if everything went well
	return tx.Commit(ctx)
}

// ExecuteSequentialTransactions executes multiple operations in sequence within a transaction
func ExecuteSequentialTransactions(ctx context.Context, db DBTX, operations []TransactionFunc) error {
	return ExecuteTransaction(ctx, db, func(tx *Transaction) error {
		for _, op := range operations {
			if err := op(tx); err != nil {
				return err
			}
		}
		return nil
	})
}

