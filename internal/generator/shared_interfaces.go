package generator

import (
	"fmt"
	"os"
)

// generateDBInterfaces generates the common database interfaces (DB, Result, Rows, Row)
// These interfaces are shared between raw and builder packages
func generateDBInterfaces(file *os.File) {
	fmt.Fprintf(file, "// DB is the main database interface that abstracts different database drivers\n")
	fmt.Fprintf(file, "type DB interface {\n")
	fmt.Fprintf(file, "\t// Exec executes a query that doesn't return rows\n")
	fmt.Fprintf(file, "\tExec(ctx context.Context, sql string, args ...interface{}) (Result, error)\n\n")
	fmt.Fprintf(file, "\t// Query executes a query that returns multiple rows\n")
	fmt.Fprintf(file, "\tQuery(ctx context.Context, sql string, args ...interface{}) (Rows, error)\n\n")
	fmt.Fprintf(file, "\t// QueryRow executes a query that returns a single row\n")
	fmt.Fprintf(file, "\tQueryRow(ctx context.Context, sql string, args ...interface{}) Row\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Result represents the result of an Exec operation\n")
	fmt.Fprintf(file, "type Result interface {\n")
	fmt.Fprintf(file, "\t// RowsAffected returns the number of rows affected\n")
	fmt.Fprintf(file, "\tRowsAffected() int64\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Rows represents a set of query results\n")
	fmt.Fprintf(file, "type Rows interface {\n")
	fmt.Fprintf(file, "\t// Close closes the rows iterator\n")
	fmt.Fprintf(file, "\tClose()\n\n")
	fmt.Fprintf(file, "\t// Err returns any error that occurred during iteration\n")
	fmt.Fprintf(file, "\tErr() error\n\n")
	fmt.Fprintf(file, "\t// Next prepares the next result row for reading\n")
	fmt.Fprintf(file, "\tNext() bool\n\n")
	fmt.Fprintf(file, "\t// Scan copies the columns in the current row into the values pointed at by dest\n")
	fmt.Fprintf(file, "\tScan(dest ...interface{}) error\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Row represents a single row result\n")
	fmt.Fprintf(file, "type Row interface {\n")
	fmt.Fprintf(file, "\t// Scan copies the columns in the current row into the values pointed at by dest\n")
	fmt.Fprintf(file, "\tScan(dest ...interface{}) error\n")
	fmt.Fprintf(file, "}\n\n")
}

// generateTxInterface generates the Tx interface (only for builder package)
func generateTxInterface(file *os.File) {
	fmt.Fprintf(file, "// Tx represents a database transaction\n")
	fmt.Fprintf(file, "type Tx interface {\n")
	fmt.Fprintf(file, "\t// Commit commits the transaction\n")
	fmt.Fprintf(file, "\tCommit(ctx context.Context) error\n\n")
	fmt.Fprintf(file, "\t// Rollback rolls back the transaction\n")
	fmt.Fprintf(file, "\tRollback(ctx context.Context) error\n\n")
	fmt.Fprintf(file, "\t// Exec executes a query that doesn't return rows\n")
	fmt.Fprintf(file, "\tExec(ctx context.Context, sql string, args ...interface{}) (Result, error)\n\n")
	fmt.Fprintf(file, "\t// Query executes a query that returns multiple rows\n")
	fmt.Fprintf(file, "\tQuery(ctx context.Context, sql string, args ...interface{}) (Rows, error)\n\n")
	fmt.Fprintf(file, "\t// QueryRow executes a query that returns a single row\n")
	fmt.Fprintf(file, "\tQueryRow(ctx context.Context, sql string, args ...interface{}) Row\n")
	fmt.Fprintf(file, "}\n\n")
}
