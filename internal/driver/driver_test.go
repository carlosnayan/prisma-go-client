package driver

// Helper functions (getTestDatabaseURL, replaceDatabaseName, removeDatabaseFromURL) are in driver_test_helpers.go
//
// Driver-specific test functions are in separate files with build tags:
// - setupPostgreSQLTestDB, TestSQLDBAdapter_PostgreSQL: driver_test_pgx.go (requires build tag pgx)
// - setupMySQLTestDB, TestSQLDBAdapter_MySQL: driver_test_mysql.go (requires build tag mysql)
// - setupSQLiteTestDB, TestSQLDBAdapter_SQLite: driver_test_sqlite.go (requires build tag sqlite)

// TestSQLDBAdapter_SQLite is tested in driver_test_sqlite.go (requires build tag)
// TestPgxPoolAdapter is tested in driver_test_pgx.go (requires build tag)

