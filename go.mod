module github.com/carlosnayan/prisma-go-client

go 1.24.0

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-sql-driver/mysql v1.9.3
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.6
	github.com/joho/godotenv v1.5.1
	github.com/mattn/go-sqlite3 v1.14.32
)

// Database drivers should be added by users based on their needs:
// - PostgreSQL: go get github.com/jackc/pgx/v5/pgxpool
// - MySQL: go get github.com/go-sql-driver/mysql
// - SQLite: go get github.com/mattn/go-sqlite3
//
// Note: Drivers are only needed for testing with build tags:
//   go test -tags="pgx,mysql,sqlite" ./...

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)
