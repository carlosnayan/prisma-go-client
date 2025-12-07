package main

import (
	"os"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/cmd"

	// Import database drivers
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
