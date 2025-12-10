package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosnayan/prisma-go-client/internal/migrations"
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

// GenerateDriver generates the driver.go file based on the provider in the schema
func GenerateDriver(schema *parser.Schema, outputDir string) error {
	driverFile := filepath.Join(outputDir, "driver.go")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Get provider from schema
	provider := migrations.GetProviderFromSchema(schema)
	provider = strings.ToLower(provider)

	// Detect user module for local imports
	userModule, err := detectUserModule(outputDir)
	if err != nil {
		// Fallback
		userModule = ""
	}

	file, err := createGeneratedFile(driverFile, "db")
	if err != nil {
		return err
	}
	defer file.Close()

	// Generate imports based on provider
	generateDriverImports(file, provider, userModule, outputDir)

	// Generate driver code based on provider
	switch provider {
	case "postgresql":
		generatePostgreSQLDriver(file)
	case "mysql":
		generateMySQLDriver(file)
	case "sqlite":
		generateSQLiteDriver(file)
	default:
		// Default to PostgreSQL
		generatePostgreSQLDriver(file)
	}

	return nil
}

// generateDriverImports generates the import block based on the provider
func generateDriverImports(file *os.File, provider string, userModule, outputDir string) {
	// Calculate local import path for builder
	builderPath, _, err := calculateLocalImportPath(userModule, outputDir)
	if err != nil || builderPath == "" {
		// Fallback to old path if detection fails
		builderPath = "github.com/carlosnayan/prisma-go-client/db/builder"
	}

	fmt.Fprintf(file, "import (\n")
	fmt.Fprintf(file, "\t\"context\"\n")
	fmt.Fprintf(file, "\t\"database/sql\"\n")
	fmt.Fprintf(file, "\t\"fmt\"\n")
	fmt.Fprintf(file, "\t\"os\"\n")
	fmt.Fprintf(file, "\t\"path/filepath\"\n")
	fmt.Fprintf(file, "\t\"strings\"\n\n")
	fmt.Fprintf(file, "\t%q\n", builderPath)
	fmt.Fprintf(file, "\t\"github.com/BurntSushi/toml\"\n")

	switch provider {
	case "postgresql":
		fmt.Fprintf(file, "\t\"github.com/jackc/pgx/v5\"\n")
		fmt.Fprintf(file, "\t\"github.com/jackc/pgx/v5/pgconn\"\n")
		fmt.Fprintf(file, "\t\"github.com/jackc/pgx/v5/pgxpool\"\n")
	case "mysql":
		fmt.Fprintf(file, "\t_ \"github.com/go-sql-driver/mysql\"\n")
	case "sqlite":
		fmt.Fprintf(file, "\t_ \"github.com/mattn/go-sqlite3\"\n")
	default:
		// Default to PostgreSQL
		fmt.Fprintf(file, "\t\"github.com/jackc/pgx/v5\"\n")
		fmt.Fprintf(file, "\t\"github.com/jackc/pgx/v5/pgconn\"\n")
		fmt.Fprintf(file, "\t\"github.com/jackc/pgx/v5/pgxpool\"\n")
	}

	fmt.Fprintf(file, ")\n\n")
}

// generatePostgreSQLDriver generates driver code for PostgreSQL using pgx
func generatePostgreSQLDriver(file *os.File) {

	// PgxPoolAdapter type
	fmt.Fprintf(file, "// PgxPoolAdapter adapts *pgxpool.Pool to builder.DBTX\n")
	fmt.Fprintf(file, "type PgxPoolAdapter struct {\n")
	fmt.Fprintf(file, "\tpool *pgxpool.Pool\n")
	fmt.Fprintf(file, "}\n\n")

	// NewPgxPoolDriver function
	fmt.Fprintf(file, "// NewPgxPoolDriver creates a new driver adapter from a pgxpool.Pool\n")
	fmt.Fprintf(file, "// This allows you to use pgx pool with the Prisma client without\n")
	fmt.Fprintf(file, "// importing internal packages.\n")
	fmt.Fprintf(file, "func NewPgxPoolDriver(pool *pgxpool.Pool) builder.DBTX {\n")
	fmt.Fprintf(file, "\treturn &PgxPoolAdapter{pool: pool}\n")
	fmt.Fprintf(file, "}\n\n")

	// NewPgxPoolFromURL function - creates pool with PgBouncer-compatible settings
	fmt.Fprintf(file, "// NewPgxPoolFromURL creates a new pgx pool from a database URL\n")
	fmt.Fprintf(file, "// with PgBouncer-compatible settings (prepared statements disabled).\n")
	fmt.Fprintf(file, "// This is the recommended way to create a pool for use with NewPgxPoolDriver.\n")
	fmt.Fprintf(file, "// Example: pool, err := db.NewPgxPoolFromURL(ctx, databaseURL)\n")
	fmt.Fprintf(file, "func NewPgxPoolFromURL(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {\n")
	fmt.Fprintf(file, "\tcfg, err := pgxpool.ParseConfig(databaseURL)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, fmt.Errorf(\"error parsing database URL: %%w\", err)\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\t// Disable prepared statements for PgBouncer compatibility\n")
	fmt.Fprintf(file, "\tcfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol\n\n")
	fmt.Fprintf(file, "\tpool, err := pgxpool.NewWithConfig(ctx, cfg)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, fmt.Errorf(\"error creating pool: %%w\", err)\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\treturn pool, nil\n")
	fmt.Fprintf(file, "}\n\n")

	// Implement DBTX interface methods
	fmt.Fprintf(file, "// Exec executes a query that doesn't return rows\n")
	fmt.Fprintf(file, "func (a *PgxPoolAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error) {\n")
	fmt.Fprintf(file, "\tresult, err := a.pool.Exec(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &PgxResult{result: result}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Query executes a query that returns multiple rows\n")
	fmt.Fprintf(file, "func (a *PgxPoolAdapter) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error) {\n")
	fmt.Fprintf(file, "\trows, err := a.pool.Query(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &PgxRows{rows: rows}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// QueryRow executes a query that returns a single row\n")
	fmt.Fprintf(file, "func (a *PgxPoolAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row {\n")
	fmt.Fprintf(file, "\treturn &PgxRow{row: a.pool.QueryRow(ctx, sql, args...)}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Begin starts a transaction\n")
	fmt.Fprintf(file, "func (a *PgxPoolAdapter) Begin(ctx context.Context) (builder.Tx, error) {\n")
	fmt.Fprintf(file, "\ttx, err := a.pool.Begin(ctx)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &PgxTx{tx: tx}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// SQLDB returns nil as pgxpool.Pool doesn't provide *sql.DB directly\n")
	fmt.Fprintf(file, "func (a *PgxPoolAdapter) SQLDB() *sql.DB {\n")
	fmt.Fprintf(file, "\treturn nil\n")
	fmt.Fprintf(file, "}\n\n")

	// PgxResult type
	fmt.Fprintf(file, "// PgxResult wraps pgconn.CommandTag\n")
	fmt.Fprintf(file, "type PgxResult struct {\n")
	fmt.Fprintf(file, "\tresult pgconn.CommandTag\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// RowsAffected returns the number of rows affected\n")
	fmt.Fprintf(file, "func (r *PgxResult) RowsAffected() int64 {\n")
	fmt.Fprintf(file, "\treturn r.result.RowsAffected()\n")
	fmt.Fprintf(file, "}\n\n")
	fmt.Fprintf(file, "// LastInsertId returns the integer generated by the database\n")
	fmt.Fprintf(file, "// PostgreSQL doesn't support LastInsertId in the same way as MySQL/SQLite\n")
	fmt.Fprintf(file, "// This returns 0 and an error indicating that RETURNING should be used instead\n")
	fmt.Fprintf(file, "func (r *PgxResult) LastInsertId() (int64, error) {\n")
	fmt.Fprintf(file, "\treturn 0, fmt.Errorf(\"PostgreSQL does not support LastInsertId, use RETURNING clause instead\")\n")
	fmt.Fprintf(file, "}\n\n")

	// PgxRows type
	fmt.Fprintf(file, "// PgxRows wraps pgx.Rows\n")
	fmt.Fprintf(file, "type PgxRows struct {\n")
	fmt.Fprintf(file, "\trows pgx.Rows\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Close closes the rows iterator\n")
	fmt.Fprintf(file, "func (r *PgxRows) Close() {\n")
	fmt.Fprintf(file, "\tr.rows.Close()\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Err returns any error that occurred during iteration\n")
	fmt.Fprintf(file, "func (r *PgxRows) Err() error {\n")
	fmt.Fprintf(file, "\treturn r.rows.Err()\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Next prepares the next result row for reading\n")
	fmt.Fprintf(file, "func (r *PgxRows) Next() bool {\n")
	fmt.Fprintf(file, "\treturn r.rows.Next()\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Scan copies the columns in the current row into the values pointed at by dest\n")
	fmt.Fprintf(file, "func (r *PgxRows) Scan(dest ...interface{}) error {\n")
	fmt.Fprintf(file, "\treturn r.rows.Scan(dest...)\n")
	fmt.Fprintf(file, "}\n\n")

	// PgxRow type
	fmt.Fprintf(file, "// PgxRow wraps pgx.Row\n")
	fmt.Fprintf(file, "type PgxRow struct {\n")
	fmt.Fprintf(file, "\trow pgx.Row\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Scan copies the columns in the current row into the values pointed at by dest\n")
	fmt.Fprintf(file, "func (r *PgxRow) Scan(dest ...interface{}) error {\n")
	fmt.Fprintf(file, "\treturn r.row.Scan(dest...)\n")
	fmt.Fprintf(file, "}\n\n")

	// PgxTx type
	fmt.Fprintf(file, "// PgxTx wraps pgx.Tx\n")
	fmt.Fprintf(file, "type PgxTx struct {\n")
	fmt.Fprintf(file, "\ttx pgx.Tx\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Commit commits the transaction\n")
	fmt.Fprintf(file, "func (t *PgxTx) Commit(ctx context.Context) error {\n")
	fmt.Fprintf(file, "\treturn t.tx.Commit(ctx)\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Rollback rolls back the transaction\n")
	fmt.Fprintf(file, "func (t *PgxTx) Rollback(ctx context.Context) error {\n")
	fmt.Fprintf(file, "\treturn t.tx.Rollback(ctx)\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes a query that doesn't return rows\n")
	fmt.Fprintf(file, "func (t *PgxTx) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error) {\n")
	fmt.Fprintf(file, "\tresult, err := t.tx.Exec(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &PgxResult{result: result}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Query executes a query that returns multiple rows\n")
	fmt.Fprintf(file, "func (t *PgxTx) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error) {\n")
	fmt.Fprintf(file, "\trows, err := t.tx.Query(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &PgxRows{rows: rows}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// QueryRow executes a query that returns a single row\n")
	fmt.Fprintf(file, "func (t *PgxTx) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row {\n")
	fmt.Fprintf(file, "\trow := t.tx.QueryRow(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\treturn &PgxRow{row: row}\n")
	fmt.Fprintf(file, "}\n\n")

	// NewSQLDriver function
	fmt.Fprintf(file, "// NewSQLDriver creates a new driver adapter from a *sql.DB\n")
	fmt.Fprintf(file, "// This allows you to use database/sql with the Prisma client without\n")
	fmt.Fprintf(file, "// importing internal packages.\n")
	fmt.Fprintf(file, "// Note: For PostgreSQL, prefer NewPgxPoolDriver for better performance.\n")
	fmt.Fprintf(file, "func NewSQLDriver(db *sql.DB) builder.DBTX {\n")
	fmt.Fprintf(file, "\treturn &SQLDBAdapter{db: db}\n")
	fmt.Fprintf(file, "}\n\n")

	// getDatabaseURLFromConfig reads DATABASE_URL from prisma.conf
	fmt.Fprintf(file, "// getDatabaseURLFromConfig reads DATABASE_URL from prisma.conf\n")
	fmt.Fprintf(file, "func getDatabaseURLFromConfig() (string, error) {\n")
	fmt.Fprintf(file, "\t// Look for prisma.conf in project root\n")
	fmt.Fprintf(file, "\twd, err := os.Getwd()\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\t// Search up directories for prisma.conf\n")
	fmt.Fprintf(file, "\tdir := wd\n")
	fmt.Fprintf(file, "\tfor {\n")
	fmt.Fprintf(file, "\t\tconfigPath := filepath.Join(dir, \"prisma.conf\")\n")
	fmt.Fprintf(file, "\t\tif _, err := os.Stat(configPath); err == nil {\n")
	fmt.Fprintf(file, "\t\t\t// Read and parse prisma.conf\n")
	fmt.Fprintf(file, "\t\t\tdata, err := os.ReadFile(configPath)\n")
	fmt.Fprintf(file, "\t\t\tif err != nil {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Parse TOML config\n")
	fmt.Fprintf(file, "\t\t\ttype Config struct {\n")
	fmt.Fprintf(file, "\t\t\t\tDatasource struct {\n")
	fmt.Fprintf(file, "\t\t\t\t\tURL string `toml:\"url\"`\n")
	fmt.Fprintf(file, "\t\t\t\t} `toml:\"datasource\"`\n")
	fmt.Fprintf(file, "\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\tvar cfg Config\n")
	fmt.Fprintf(file, "\t\t\tif _, err := toml.Decode(string(data), &cfg); err != nil {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Expand environment variables in URL (support env(\"DATABASE_URL\") or ${DATABASE_URL})\n")
	fmt.Fprintf(file, "\t\t\turl := cfg.Datasource.URL\n")
	fmt.Fprintf(file, "\t\t\tif url == \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Handle env(\"VAR\") or env('VAR') format\n")
	fmt.Fprintf(file, "\t\t\tif strings.HasPrefix(url, \"env(\\\"\") {\n")
	fmt.Fprintf(file, "\t\t\t\tend := strings.Index(url, \"\\\")\")\n")
	fmt.Fprintf(file, "\t\t\t\tif end > 5 {\n")
	fmt.Fprintf(file, "\t\t\t\t\tvarName := url[5:end]\n")
	fmt.Fprintf(file, "\t\t\t\t\tif value := os.Getenv(varName); value != \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\treturn value, nil\n")
	fmt.Fprintf(file, "\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t} else if strings.HasPrefix(url, \"env('\") {\n")
	fmt.Fprintf(file, "\t\t\t\tend := strings.Index(url, \"')\")\n")
	fmt.Fprintf(file, "\t\t\t\tif end > 5 {\n")
	fmt.Fprintf(file, "\t\t\t\t\tvarName := url[5:end]\n")
	fmt.Fprintf(file, "\t\t\t\t\tif value := os.Getenv(varName); value != \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\treturn value, nil\n")
	fmt.Fprintf(file, "\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Handle ${VAR} format\n")
	fmt.Fprintf(file, "\t\t\tif strings.Contains(url, \"${\") {\n")
	fmt.Fprintf(file, "\t\t\t\turl = os.ExpandEnv(url)\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\treturn url, nil\n")
	fmt.Fprintf(file, "\t\t}\n\n")
	fmt.Fprintf(file, "\t\tparent := filepath.Dir(dir)\n")
	fmt.Fprintf(file, "\t\tif parent == dir {\n")
	fmt.Fprintf(file, "\t\t\t// Reached root, not found\n")
	fmt.Fprintf(file, "\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t\tdir = parent\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "}\n\n")

	// SetupClient function
	fmt.Fprintf(file, "// SetupClient creates a new Prisma client from DATABASE_URL\n")
	fmt.Fprintf(file, "// The database URL can be provided in two ways:\n")
	fmt.Fprintf(file, "// 1. As an optional parameter to SetupClient() (highest priority)\n")
	fmt.Fprintf(file, "// 2. In prisma.conf file under [datasource] url\n")
	fmt.Fprintf(file, "// Example: client, pool, err := db.SetupClient(ctx)\n")
	fmt.Fprintf(file, "// Example with explicit URL: client, pool, err := db.SetupClient(ctx, \"postgresql://...\")\n")
	fmt.Fprintf(file, "func SetupClient(ctx context.Context, databaseURL ...string) (*Client, *pgxpool.Pool, error) {\n")
	fmt.Fprintf(file, "\tvar url string\n")
	fmt.Fprintf(file, "\tvar err error\n\n")
	fmt.Fprintf(file, "\t// Priority 1: Explicit parameter\n")
	fmt.Fprintf(file, "\tif len(databaseURL) > 0 && databaseURL[0] != \"\" {\n")
	fmt.Fprintf(file, "\t\turl = databaseURL[0]\n")
	fmt.Fprintf(file, "\t} else {\n")
	fmt.Fprintf(file, "\t\t// Priority 2: Read from prisma.conf\n")
	fmt.Fprintf(file, "\t\turl, err = getDatabaseURLFromConfig()\n")
	fmt.Fprintf(file, "\t\tif err != nil {\n")
	fmt.Fprintf(file, "\t\t\treturn nil, nil, fmt.Errorf(\"error reading prisma.conf: %%w\", err)\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tif url == \"\" {\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, fmt.Errorf(\"DATABASE_URL is not set. Provide it via SetupClient parameter or in prisma.conf [datasource] url\")\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tpool, err := NewPgxPoolFromURL(ctx, url)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, err\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tif err := pool.Ping(ctx); err != nil {\n")
	fmt.Fprintf(file, "\t\tpool.Close()\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, fmt.Errorf(\"error pinging database: %%w\", err)\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tdbDriver := NewPgxPoolDriver(pool)\n")
	fmt.Fprintf(file, "\tclient := NewClient(dbDriver)\n\n")
	fmt.Fprintf(file, "\treturn client, pool, nil\n")
	fmt.Fprintf(file, "}\n\n")

	// SQLDBAdapter type (shared with MySQL and SQLite)
	generateSQLDBAdapter(file)
}

// generateMySQLDriver generates driver code for MySQL
func generateMySQLDriver(file *os.File) {

	fmt.Fprintf(file, "// NewSQLDriver creates a new driver adapter from a *sql.DB\n")
	fmt.Fprintf(file, "// This allows you to use database/sql with the Prisma client without\n")
	fmt.Fprintf(file, "// importing internal packages.\n")
	fmt.Fprintf(file, "func NewSQLDriver(db *sql.DB) builder.DBTX {\n")
	fmt.Fprintf(file, "\treturn &SQLDBAdapter{db: db}\n")
	fmt.Fprintf(file, "}\n\n")

	// getDatabaseURLFromConfig reads DATABASE_URL from prisma.conf
	fmt.Fprintf(file, "// getDatabaseURLFromConfig reads DATABASE_URL from prisma.conf\n")
	fmt.Fprintf(file, "func getDatabaseURLFromConfig() (string, error) {\n")
	fmt.Fprintf(file, "\t// Look for prisma.conf in project root\n")
	fmt.Fprintf(file, "\twd, err := os.Getwd()\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\t// Search up directories for prisma.conf\n")
	fmt.Fprintf(file, "\tdir := wd\n")
	fmt.Fprintf(file, "\tfor {\n")
	fmt.Fprintf(file, "\t\tconfigPath := filepath.Join(dir, \"prisma.conf\")\n")
	fmt.Fprintf(file, "\t\tif _, err := os.Stat(configPath); err == nil {\n")
	fmt.Fprintf(file, "\t\t\t// Read and parse prisma.conf\n")
	fmt.Fprintf(file, "\t\t\tdata, err := os.ReadFile(configPath)\n")
	fmt.Fprintf(file, "\t\t\tif err != nil {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Parse TOML config\n")
	fmt.Fprintf(file, "\t\t\ttype Config struct {\n")
	fmt.Fprintf(file, "\t\t\t\tDatasource struct {\n")
	fmt.Fprintf(file, "\t\t\t\t\tURL string `toml:\"url\"`\n")
	fmt.Fprintf(file, "\t\t\t\t} `toml:\"datasource\"`\n")
	fmt.Fprintf(file, "\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\tvar cfg Config\n")
	fmt.Fprintf(file, "\t\t\tif _, err := toml.Decode(string(data), &cfg); err != nil {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Expand environment variables in URL (support env(\"DATABASE_URL\") or ${DATABASE_URL})\n")
	fmt.Fprintf(file, "\t\t\turl := cfg.Datasource.URL\n")
	fmt.Fprintf(file, "\t\t\tif url == \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Handle env(\"VAR\") or env('VAR') format\n")
	fmt.Fprintf(file, "\t\t\tif strings.HasPrefix(url, \"env(\\\"\") {\n")
	fmt.Fprintf(file, "\t\t\t\tend := strings.Index(url, \"\\\")\")\n")
	fmt.Fprintf(file, "\t\t\t\tif end > 5 {\n")
	fmt.Fprintf(file, "\t\t\t\t\tvarName := url[5:end]\n")
	fmt.Fprintf(file, "\t\t\t\t\tif value := os.Getenv(varName); value != \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\treturn value, nil\n")
	fmt.Fprintf(file, "\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t} else if strings.HasPrefix(url, \"env('\") {\n")
	fmt.Fprintf(file, "\t\t\t\tend := strings.Index(url, \"')\")\n")
	fmt.Fprintf(file, "\t\t\t\tif end > 5 {\n")
	fmt.Fprintf(file, "\t\t\t\t\tvarName := url[5:end]\n")
	fmt.Fprintf(file, "\t\t\t\t\tif value := os.Getenv(varName); value != \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\treturn value, nil\n")
	fmt.Fprintf(file, "\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Handle ${VAR} format\n")
	fmt.Fprintf(file, "\t\t\tif strings.Contains(url, \"${\") {\n")
	fmt.Fprintf(file, "\t\t\t\turl = os.ExpandEnv(url)\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\treturn url, nil\n")
	fmt.Fprintf(file, "\t\t}\n\n")
	fmt.Fprintf(file, "\t\tparent := filepath.Dir(dir)\n")
	fmt.Fprintf(file, "\t\tif parent == dir {\n")
	fmt.Fprintf(file, "\t\t\t// Reached root, not found\n")
	fmt.Fprintf(file, "\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t\tdir = parent\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "}\n\n")

	// SetupClient function
	fmt.Fprintf(file, "// SetupClient creates a new Prisma client from DATABASE_URL\n")
	fmt.Fprintf(file, "// The database URL can be provided in two ways:\n")
	fmt.Fprintf(file, "// 1. As an optional parameter to SetupClient() (highest priority)\n")
	fmt.Fprintf(file, "// 2. In prisma.conf file under [datasource] url\n")
	fmt.Fprintf(file, "// Example: client, db, err := db.SetupClient(ctx)\n")
	fmt.Fprintf(file, "// Example with explicit URL: client, db, err := db.SetupClient(ctx, \"mysql://...\")\n")
	fmt.Fprintf(file, "func SetupClient(ctx context.Context, databaseURL ...string) (*Client, *sql.DB, error) {\n")
	fmt.Fprintf(file, "\tvar url string\n")
	fmt.Fprintf(file, "\tvar err error\n\n")
	fmt.Fprintf(file, "\t// Priority 1: Explicit parameter\n")
	fmt.Fprintf(file, "\tif len(databaseURL) > 0 && databaseURL[0] != \"\" {\n")
	fmt.Fprintf(file, "\t\turl = databaseURL[0]\n")
	fmt.Fprintf(file, "\t} else {\n")
	fmt.Fprintf(file, "\t\t// Priority 2: Read from prisma.conf\n")
	fmt.Fprintf(file, "\t\turl, err = getDatabaseURLFromConfig()\n")
	fmt.Fprintf(file, "\t\tif err != nil {\n")
	fmt.Fprintf(file, "\t\t\treturn nil, nil, fmt.Errorf(\"error reading prisma.conf: %%w\", err)\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tif url == \"\" {\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, fmt.Errorf(\"DATABASE_URL is not set. Provide it via SetupClient parameter or in prisma.conf [datasource] url\")\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tdb, err := sql.Open(\"mysql\", url)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, fmt.Errorf(\"error opening database: %%w\", err)\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tif err := db.PingContext(ctx); err != nil {\n")
	fmt.Fprintf(file, "\t\tdb.Close()\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, fmt.Errorf(\"error pinging database: %%w\", err)\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tdbDriver := NewSQLDriver(db)\n")
	fmt.Fprintf(file, "\tclient := NewClient(dbDriver)\n\n")
	fmt.Fprintf(file, "\treturn client, db, nil\n")
	fmt.Fprintf(file, "}\n\n")

	// SQLDBAdapter type
	generateSQLDBAdapter(file)
}

// generateSQLiteDriver generates driver code for SQLite
func generateSQLiteDriver(file *os.File) {

	fmt.Fprintf(file, "// NewSQLDriver creates a new driver adapter from a *sql.DB\n")
	fmt.Fprintf(file, "// This allows you to use database/sql with the Prisma client without\n")
	fmt.Fprintf(file, "// importing internal packages.\n")
	fmt.Fprintf(file, "func NewSQLDriver(db *sql.DB) builder.DBTX {\n")
	fmt.Fprintf(file, "\treturn &SQLDBAdapter{db: db}\n")
	fmt.Fprintf(file, "}\n\n")

	// getDatabaseURLFromConfig reads DATABASE_URL from prisma.conf
	fmt.Fprintf(file, "// getDatabaseURLFromConfig reads DATABASE_URL from prisma.conf\n")
	fmt.Fprintf(file, "func getDatabaseURLFromConfig() (string, error) {\n")
	fmt.Fprintf(file, "\t// Look for prisma.conf in project root\n")
	fmt.Fprintf(file, "\twd, err := os.Getwd()\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\t// Search up directories for prisma.conf\n")
	fmt.Fprintf(file, "\tdir := wd\n")
	fmt.Fprintf(file, "\tfor {\n")
	fmt.Fprintf(file, "\t\tconfigPath := filepath.Join(dir, \"prisma.conf\")\n")
	fmt.Fprintf(file, "\t\tif _, err := os.Stat(configPath); err == nil {\n")
	fmt.Fprintf(file, "\t\t\t// Read and parse prisma.conf\n")
	fmt.Fprintf(file, "\t\t\tdata, err := os.ReadFile(configPath)\n")
	fmt.Fprintf(file, "\t\t\tif err != nil {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Parse TOML config\n")
	fmt.Fprintf(file, "\t\t\ttype Config struct {\n")
	fmt.Fprintf(file, "\t\t\t\tDatasource struct {\n")
	fmt.Fprintf(file, "\t\t\t\t\tURL string `toml:\"url\"`\n")
	fmt.Fprintf(file, "\t\t\t\t} `toml:\"datasource\"`\n")
	fmt.Fprintf(file, "\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\tvar cfg Config\n")
	fmt.Fprintf(file, "\t\t\tif _, err := toml.Decode(string(data), &cfg); err != nil {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", err\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Expand environment variables in URL (support env(\"DATABASE_URL\") or ${DATABASE_URL})\n")
	fmt.Fprintf(file, "\t\t\turl := cfg.Datasource.URL\n")
	fmt.Fprintf(file, "\t\t\tif url == \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Handle env(\"VAR\") or env('VAR') format\n")
	fmt.Fprintf(file, "\t\t\tif strings.HasPrefix(url, \"env(\\\"\") {\n")
	fmt.Fprintf(file, "\t\t\t\tend := strings.Index(url, \"\\\")\")\n")
	fmt.Fprintf(file, "\t\t\t\tif end > 5 {\n")
	fmt.Fprintf(file, "\t\t\t\t\tvarName := url[5:end]\n")
	fmt.Fprintf(file, "\t\t\t\t\tif value := os.Getenv(varName); value != \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\treturn value, nil\n")
	fmt.Fprintf(file, "\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t} else if strings.HasPrefix(url, \"env('\") {\n")
	fmt.Fprintf(file, "\t\t\t\tend := strings.Index(url, \"')\")\n")
	fmt.Fprintf(file, "\t\t\t\tif end > 5 {\n")
	fmt.Fprintf(file, "\t\t\t\t\tvarName := url[5:end]\n")
	fmt.Fprintf(file, "\t\t\t\t\tif value := os.Getenv(varName); value != \"\" {\n")
	fmt.Fprintf(file, "\t\t\t\t\t\treturn value, nil\n")
	fmt.Fprintf(file, "\t\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t\t\t}\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\t// Handle ${VAR} format\n")
	fmt.Fprintf(file, "\t\t\tif strings.Contains(url, \"${\") {\n")
	fmt.Fprintf(file, "\t\t\t\turl = os.ExpandEnv(url)\n")
	fmt.Fprintf(file, "\t\t\t}\n\n")
	fmt.Fprintf(file, "\t\t\treturn url, nil\n")
	fmt.Fprintf(file, "\t\t}\n\n")
	fmt.Fprintf(file, "\t\tparent := filepath.Dir(dir)\n")
	fmt.Fprintf(file, "\t\tif parent == dir {\n")
	fmt.Fprintf(file, "\t\t\t// Reached root, not found\n")
	fmt.Fprintf(file, "\t\t\treturn \"\", nil\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t\tdir = parent\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "}\n\n")

	// SetupClient function
	fmt.Fprintf(file, "// SetupClient creates a new Prisma client from DATABASE_URL\n")
	fmt.Fprintf(file, "// The database URL can be provided in two ways:\n")
	fmt.Fprintf(file, "// 1. As an optional parameter to SetupClient() (highest priority)\n")
	fmt.Fprintf(file, "// 2. In prisma.conf file under [datasource] url\n")
	fmt.Fprintf(file, "// Example: client, db, err := db.SetupClient(ctx)\n")
	fmt.Fprintf(file, "// Example with explicit URL: client, db, err := db.SetupClient(ctx, \"file:./db.sqlite\")\n")
	fmt.Fprintf(file, "func SetupClient(ctx context.Context, databaseURL ...string) (*Client, *sql.DB, error) {\n")
	fmt.Fprintf(file, "\tvar url string\n")
	fmt.Fprintf(file, "\tvar err error\n\n")
	fmt.Fprintf(file, "\t// Priority 1: Explicit parameter\n")
	fmt.Fprintf(file, "\tif len(databaseURL) > 0 && databaseURL[0] != \"\" {\n")
	fmt.Fprintf(file, "\t\turl = databaseURL[0]\n")
	fmt.Fprintf(file, "\t} else {\n")
	fmt.Fprintf(file, "\t\t// Priority 2: Read from prisma.conf\n")
	fmt.Fprintf(file, "\t\turl, err = getDatabaseURLFromConfig()\n")
	fmt.Fprintf(file, "\t\tif err != nil {\n")
	fmt.Fprintf(file, "\t\t\treturn nil, nil, fmt.Errorf(\"error reading prisma.conf: %%w\", err)\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tif url == \"\" {\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, fmt.Errorf(\"DATABASE_URL is not set. Provide it via SetupClient parameter or in prisma.conf [datasource] url\")\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tdb, err := sql.Open(\"sqlite3\", url)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, fmt.Errorf(\"error opening database: %%w\", err)\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tif err := db.PingContext(ctx); err != nil {\n")
	fmt.Fprintf(file, "\t\tdb.Close()\n")
	fmt.Fprintf(file, "\t\treturn nil, nil, fmt.Errorf(\"error pinging database: %%w\", err)\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\tdbDriver := NewSQLDriver(db)\n")
	fmt.Fprintf(file, "\tclient := NewClient(dbDriver)\n\n")
	fmt.Fprintf(file, "\treturn client, db, nil\n")
	fmt.Fprintf(file, "}\n\n")

	// SQLDBAdapter type
	generateSQLDBAdapter(file)
}

// generateSQLDBAdapter generates the SQLDBAdapter implementation (shared for MySQL and SQLite)
func generateSQLDBAdapter(file *os.File) {
	// SQLDBAdapter type
	fmt.Fprintf(file, "// SQLDBAdapter adapts *sql.DB to builder.DBTX\n")
	fmt.Fprintf(file, "type SQLDBAdapter struct {\n")
	fmt.Fprintf(file, "\tdb *sql.DB\n")
	fmt.Fprintf(file, "}\n\n")

	// Implement DBTX interface methods
	fmt.Fprintf(file, "// Exec executes a query that doesn't return rows\n")
	fmt.Fprintf(file, "func (a *SQLDBAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error) {\n")
	fmt.Fprintf(file, "\tresult, err := a.db.ExecContext(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &SQLResult{result: result}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Query executes a query that returns multiple rows\n")
	fmt.Fprintf(file, "func (a *SQLDBAdapter) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error) {\n")
	fmt.Fprintf(file, "\trows, err := a.db.QueryContext(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &SQLRows{rows: rows}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// QueryRow executes a query that returns a single row\n")
	fmt.Fprintf(file, "func (a *SQLDBAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row {\n")
	fmt.Fprintf(file, "\trow := a.db.QueryRowContext(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\treturn &SQLRow{row: row}\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Begin starts a transaction\n")
	fmt.Fprintf(file, "func (a *SQLDBAdapter) Begin(ctx context.Context) (builder.Tx, error) {\n")
	fmt.Fprintf(file, "\ttx, err := a.db.BeginTx(ctx, nil)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &SQLTx{tx: tx}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// SQLDB returns the underlying *sql.DB\n")
	fmt.Fprintf(file, "func (a *SQLDBAdapter) SQLDB() *sql.DB {\n")
	fmt.Fprintf(file, "\treturn a.db\n")
	fmt.Fprintf(file, "}\n\n")

	// SQLResult type
	fmt.Fprintf(file, "// SQLResult wraps sql.Result\n")
	fmt.Fprintf(file, "type SQLResult struct {\n")
	fmt.Fprintf(file, "\tresult sql.Result\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// RowsAffected returns the number of rows affected\n")
	fmt.Fprintf(file, "func (r *SQLResult) RowsAffected() int64 {\n")
	fmt.Fprintf(file, "\trows, _ := r.result.RowsAffected()\n")
	fmt.Fprintf(file, "\treturn rows\n")
	fmt.Fprintf(file, "}\n\n")
	fmt.Fprintf(file, "// LastInsertId returns the integer generated by the database\n")
	fmt.Fprintf(file, "func (r *SQLResult) LastInsertId() (int64, error) {\n")
	fmt.Fprintf(file, "\treturn r.result.LastInsertId()\n")
	fmt.Fprintf(file, "}\n\n")

	// SQLRows type
	fmt.Fprintf(file, "// SQLRows wraps sql.Rows\n")
	fmt.Fprintf(file, "type SQLRows struct {\n")
	fmt.Fprintf(file, "\trows *sql.Rows\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Close closes the rows iterator\n")
	fmt.Fprintf(file, "func (r *SQLRows) Close() {\n")
	fmt.Fprintf(file, "\tr.rows.Close()\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Err returns any error that occurred during iteration\n")
	fmt.Fprintf(file, "func (r *SQLRows) Err() error {\n")
	fmt.Fprintf(file, "\treturn r.rows.Err()\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Next prepares the next result row for reading\n")
	fmt.Fprintf(file, "func (r *SQLRows) Next() bool {\n")
	fmt.Fprintf(file, "\treturn r.rows.Next()\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Scan copies the columns in the current row into the values pointed at by dest\n")
	fmt.Fprintf(file, "func (r *SQLRows) Scan(dest ...interface{}) error {\n")
	fmt.Fprintf(file, "\treturn r.rows.Scan(dest...)\n")
	fmt.Fprintf(file, "}\n\n")

	// SQLRow type
	fmt.Fprintf(file, "// SQLRow wraps sql.Row\n")
	fmt.Fprintf(file, "type SQLRow struct {\n")
	fmt.Fprintf(file, "\trow *sql.Row\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Scan copies the columns in the current row into the values pointed at by dest\n")
	fmt.Fprintf(file, "func (r *SQLRow) Scan(dest ...interface{}) error {\n")
	fmt.Fprintf(file, "\treturn r.row.Scan(dest...)\n")
	fmt.Fprintf(file, "}\n\n")

	// SQLTx type
	fmt.Fprintf(file, "// SQLTx wraps sql.Tx\n")
	fmt.Fprintf(file, "type SQLTx struct {\n")
	fmt.Fprintf(file, "\ttx *sql.Tx\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Commit commits the transaction\n")
	fmt.Fprintf(file, "func (t *SQLTx) Commit(ctx context.Context) error {\n")
	fmt.Fprintf(file, "\treturn t.tx.Commit()\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Rollback rolls back the transaction\n")
	fmt.Fprintf(file, "func (t *SQLTx) Rollback(ctx context.Context) error {\n")
	fmt.Fprintf(file, "\treturn t.tx.Rollback()\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Exec executes a query that doesn't return rows\n")
	fmt.Fprintf(file, "func (t *SQLTx) Exec(ctx context.Context, sql string, args ...interface{}) (builder.Result, error) {\n")
	fmt.Fprintf(file, "\tresult, err := t.tx.ExecContext(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &SQLResult{result: result}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// Query executes a query that returns multiple rows\n")
	fmt.Fprintf(file, "func (t *SQLTx) Query(ctx context.Context, sql string, args ...interface{}) (builder.Rows, error) {\n")
	fmt.Fprintf(file, "\trows, err := t.tx.QueryContext(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\tif err != nil {\n")
	fmt.Fprintf(file, "\t\treturn nil, err\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn &SQLRows{rows: rows}, nil\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// QueryRow executes a query that returns a single row\n")
	fmt.Fprintf(file, "func (t *SQLTx) QueryRow(ctx context.Context, sql string, args ...interface{}) builder.Row {\n")
	fmt.Fprintf(file, "\trow := t.tx.QueryRowContext(ctx, sql, args...)\n")
	fmt.Fprintf(file, "\treturn &SQLRow{row: row}\n")
	fmt.Fprintf(file, "}\n")
}
