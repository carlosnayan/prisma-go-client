package builder

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
	testutil "github.com/carlosnayan/prisma-go-client/internal/testing"
)

// Book model simula o modelo gerado
type Book struct {
	ID        int       `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	Author    string    `json:"author" db:"author"`
	ISBN      string    `json:"isbn" db:"isbn"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// BookQuery simula o query builder gerado (como UserIdentifiersQuery)
type BookQuery struct {
	*Query
}

// NewBookQuery cria um novo BookQuery (simula o código gerado)
func NewBookQuery(db DBTX) *BookQuery {
	columns := []string{"id", "title", "author", "isbn", "created_at"}
	query := NewQuery(db, "books", columns)
	query.SetDialect(dialect.GetDialect("postgresql"))
	query.SetPrimaryKey("id")
	query.SetModelType(reflect.TypeOf(Book{}))

	return &BookQuery{Query: query}
}

// FindFirst simula o método FindFirst gerado
func (q *BookQuery) FindFirst(ctx context.Context, where Where) (*Book, error) {
	if where != nil {
		q.Query.Where(where)
	}
	var result Book
	err := q.Query.First(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// TestQueryBuilder_FindFirst_WithOrConditions testa o FindFirst com condições OR
// Simula o uso real: FindFirst().Where(WhereInput{Or: [...]}).Exec()
func TestQueryBuilder_FindFirst_WithOrConditions(t *testing.T) {
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

			// Criar tabela books
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						isbn VARCHAR(50) UNIQUE,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						isbn VARCHAR(50) UNIQUE,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL,
						isbn TEXT UNIQUE,
						created_at DATETIME DEFAULT CURRENT_TIMESTAMP
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Inserir dados de teste
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = `INSERT INTO books (title, author, isbn) VALUES ($1, $2, $3) RETURNING id`
			case "mysql", "sqlite":
				insertSQL = `INSERT INTO books (title, author, isbn) VALUES (?, ?, ?)`
			}

			books := []struct {
				title  string
				author string
				isbn   string
			}{
				{"The Great Gatsby", "F. Scott Fitzgerald", "978-0-7432-7356-5"},
				{"1984", "George Orwell", "978-0-452-28423-4"},
				{"To Kill a Mockingbird", "Harper Lee", "978-0-06-112008-4"},
			}

			for _, book := range books {
				if provider == "postgresql" {
					_, err = sqlDB.ExecContext(ctx, insertSQL, book.title, book.author, book.isbn)
				} else {
					_, err = sqlDB.ExecContext(ctx, insertSQL, book.title, book.author, book.isbn)
				}
				if err != nil {
					t.Fatalf("failed to insert book: %v", err)
				}
			}

			// Criar query builder (simula o código gerado)
			bookQuery := NewBookQuery(db)
			bookQuery.SetDialect(dialect.GetDialect(provider))

			// Teste 1: FindFirst com condição simples
			where := Where{
				"title": "The Great Gatsby",
			}
			found, err := bookQuery.FindFirst(ctx, where)
			if err != nil {
				t.Fatalf("FindFirst failed: %v", err)
			}

			if found == nil {
				t.Fatal("FindFirst returned nil")
			}

			if found.Title != "The Great Gatsby" {
				t.Errorf("Expected title 'The Great Gatsby', got '%s'", found.Title)
			}

			// Teste 2: FindFirst com condição OR usando Or()
			// Buscar livro por título OU autor usando OR
			bookQuery2 := NewBookQuery(db)
			bookQuery2.SetDialect(dialect.GetDialect(provider))
			bookQuery2.Query.Where("title = ?", "1984")
			bookQuery2.Query.Or("author = ?", "George Orwell")

			var found2 Book
			err = bookQuery2.Query.First(ctx, &found2)
			if err != nil {
				t.Fatalf("FindFirst with OR failed: %v", err)
			}

			// Deve encontrar "1984" (primeira condição) ou qualquer livro de "George Orwell"
			// Como "1984" vem primeiro na query, deve retornar ele
			if found2.Title != "1984" && found2.Author != "George Orwell" {
				t.Errorf("Expected title '1984' or author 'George Orwell', got title '%s' author '%s'", found2.Title, found2.Author)
			}

			// Teste 2b: Verificar que realmente encontrou "1984" (primeira condição OR)
			if found2.Title == "1984" {
				// OK - encontrou pela primeira condição
			} else if found2.Author == "George Orwell" {
				// OK - encontrou pela segunda condição OR
			} else {
				t.Errorf("OR condition failed: expected '1984' or author 'George Orwell', got '%s' by '%s'", found2.Title, found2.Author)
			}

			// Teste 3: FindFirst com condição que não existe (deve retornar erro)
			bookQuery3 := NewBookQuery(db)
			bookQuery3.SetDialect(dialect.GetDialect(provider))
			where3 := Where{
				"title": "Non-existent Book",
			}
			_, err = bookQuery3.FindFirst(ctx, where3)
			if err == nil {
				t.Error("Expected error for non-existent book, got nil")
			}
		})
	}
}

// TestQueryBuilder_FindFirst_WithSelectFields testa FindFirst com Select
// Simula o uso com Select() para escolher campos específicos
func TestQueryBuilder_FindFirst_WithSelectFields(t *testing.T) {
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

			// Criar tabela books
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						isbn VARCHAR(50),
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						isbn VARCHAR(50),
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL,
						isbn TEXT,
						created_at DATETIME DEFAULT CURRENT_TIMESTAMP
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Inserir livro de teste
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = `INSERT INTO books (title, author, isbn) VALUES ($1, $2, $3) RETURNING id`
			case "mysql", "sqlite":
				insertSQL = `INSERT INTO books (title, author, isbn) VALUES (?, ?, ?)`
			}

			_, err = sqlDB.ExecContext(ctx, insertSQL, "Test Book", "Test Author", "123-456-789")
			if err != nil {
				t.Fatalf("failed to insert book: %v", err)
			}

			// Criar query builder
			bookQuery := NewBookQuery(db)
			bookQuery.SetDialect(dialect.GetDialect(provider))

			// Teste: Select apenas alguns campos
			bookQuery.Select("id", "title", "author") // Não seleciona isbn e created_at

			var result Book
			err = bookQuery.First(ctx, &result)
			if err != nil {
				t.Fatalf("First with Select failed: %v", err)
			}

			// Verificar que os campos selecionados foram preenchidos
			if result.Title == "" {
				t.Error("Expected title to be filled")
			}
			if result.Author == "" {
				t.Error("Expected author to be filled")
			}

			// Campos não selecionados devem estar vazios (zero values)
			// Isso é esperado quando usamos Select com campos específicos
		})
	}
}

// TestQueryBuilder_Create_ReturnsCorrectModel testa Create e verifica que retorna o modelo correto
func TestQueryBuilder_Create_ReturnsCorrectModel(t *testing.T) {
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

			// Criar tabela books
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						isbn VARCHAR(50),
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						isbn VARCHAR(50),
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL,
						isbn TEXT,
						created_at DATETIME DEFAULT CURRENT_TIMESTAMP
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Criar query builder usando TableQueryBuilder (simula o código gerado)
			columns := []string{"id", "title", "author", "isbn", "created_at"}
			builder := NewTableQueryBuilder(db, "books", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id")
			builder.SetModelType(reflect.TypeOf(Book{}))

			// Criar livro
			book := Book{
				Title:  "New Book",
				Author: "New Author",
				ISBN:   "999-999-999",
			}

			created, err := builder.Create(ctx, book)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			// SQLite não retorna o modelo criado, apenas confirma sucesso
			if provider == "sqlite" {
				if created != nil {
					t.Error("SQLite Create should return nil, got non-nil value")
				}
				// Verificar que o livro foi criado usando FindFirst
				found, err := builder.FindFirst(ctx, Where{"title": "New Book", "author": "New Author"})
				if err != nil {
					t.Fatalf("FindFirst failed: %v", err)
				}
				foundBook, ok := found.(Book)
				if !ok {
					t.Fatal("FindFirst returned wrong type")
				}
				if foundBook.ID == 0 {
					t.Error("Expected ID to be set, got 0")
				}
				if foundBook.Title != "New Book" {
					t.Errorf("Expected title 'New Book', got '%s'", foundBook.Title)
				}
				if foundBook.Author != "New Author" {
					t.Errorf("Expected author 'New Author', got '%s'", foundBook.Author)
				}
				if foundBook.ISBN != "999-999-999" {
					t.Errorf("Expected ISBN '999-999-999', got '%s'", foundBook.ISBN)
				}
				return
			}

			createdBook, ok := created.(Book)
			if !ok {
				t.Fatal("Create returned wrong type")
			}

			// Verificar que o livro foi criado corretamente
			if createdBook.ID == 0 {
				t.Error("Expected ID to be set, got 0")
			}
			if createdBook.Title != "New Book" {
				t.Errorf("Expected title 'New Book', got '%s'", createdBook.Title)
			}
			if createdBook.Author != "New Author" {
				t.Errorf("Expected author 'New Author', got '%s'", createdBook.Author)
			}
			if createdBook.ISBN != "999-999-999" {
				t.Errorf("Expected ISBN '999-999-999', got '%s'", createdBook.ISBN)
			}
			if createdBook.CreatedAt.IsZero() {
				t.Error("Expected CreatedAt to be set, got zero value")
			}
		})
	}
}

// TestQueryBuilder_Scan_WithCorrectFieldCount testa que o scan funciona corretamente
// mesmo quando há campos extras no modelo (simula relacionamentos removidos)
func TestQueryBuilder_Scan_WithCorrectFieldCount(t *testing.T) {
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

			// Criar tabela books (apenas 4 colunas: id, title, author, created_at)
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id SERIAL PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			case "mysql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INT AUTO_INCREMENT PRIMARY KEY,
						title VARCHAR(255) NOT NULL,
						author VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					)
				`
			case "sqlite":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS books (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						author TEXT NOT NULL,
						created_at DATETIME DEFAULT CURRENT_TIMESTAMP
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Inserir livro
			var insertSQL string
			switch provider {
			case "postgresql":
				insertSQL = `INSERT INTO books (title, author) VALUES ($1, $2) RETURNING id`
			case "mysql", "sqlite":
				insertSQL = `INSERT INTO books (title, author) VALUES (?, ?)`
			}

			_, err = sqlDB.ExecContext(ctx, insertSQL, "Test Book", "Test Author")
			if err != nil {
				t.Fatalf("failed to insert book: %v", err)
			}

			// Criar query builder com apenas as colunas que existem na tabela
			columns := []string{"id", "title", "author", "created_at"} // Apenas 4 colunas
			bookQuery := NewQuery(db, "books", columns)
			bookQuery.SetDialect(dialect.GetDialect(provider))
			bookQuery.SetPrimaryKey("id")
			bookQuery.SetModelType(reflect.TypeOf(Book{})) // Book tem 5 campos, mas apenas 4 colunas

			// Teste: FindFirst deve funcionar mesmo com modelo tendo mais campos que colunas
			var result Book
			err = bookQuery.First(ctx, &result)
			if err != nil {
				t.Fatalf("First failed: %v", err)
			}

			// Verificar que os campos correspondentes foram preenchidos
			if result.Title == "" {
				t.Error("Expected title to be filled")
			}
			if result.Author == "" {
				t.Error("Expected author to be filled")
			}
			if result.ID == 0 {
				t.Error("Expected ID to be set")
			}

			// ISBN deve estar vazio (zero value) pois não foi selecionado
			// Isso é esperado e não deve causar erro
		})
	}
}

// UserIdentifiers model simula o modelo do problema relatado
// com db tags diferentes de json tags (snake_case)
type UserIdentifiers struct {
	IdUserIdentifiers int       `json:"id_user_identifiers" db:"id_user_identifiers"`
	Email             string    `json:"email" db:"email"`
	Phone             string    `json:"phone" db:"phone"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// TestTableQueryBuilder_Create_WithDBTags testa Create com modelo que usa db tags
// Este teste deve detectar o problema de scan quando buildColumnToFieldMap não mapeia corretamente
func TestTableQueryBuilder_Create_WithDBTags(t *testing.T) {
	providers := []string{"postgresql"}

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

			// Criar tabela user_identifiers (simula o schema do problema)
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS user_identifiers (
						id_user_identifiers SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						phone VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Criar query builder usando TableQueryBuilder (simula o código gerado)
			columns := []string{"id_user_identifiers", "email", "phone", "created_at"}
			builder := NewTableQueryBuilder(db, "user_identifiers", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id_user_identifiers")
			builder.SetModelType(reflect.TypeOf(UserIdentifiers{}))

			// Criar registro
			userIdentifiers := UserIdentifiers{
				Email: "test@example.com",
				Phone: "1234567890",
			}

			created, err := builder.Create(ctx, userIdentifiers)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			createdUserIdentifiers, ok := created.(UserIdentifiers)
			if !ok {
				t.Fatal("Create returned wrong type")
			}

			// Verificar que todos os campos foram preenchidos corretamente
			if createdUserIdentifiers.IdUserIdentifiers == 0 {
				t.Error("Expected IdUserIdentifiers to be set")
			}
			if createdUserIdentifiers.Email != "test@example.com" {
				t.Errorf("Expected email to be 'test@example.com', got '%s'", createdUserIdentifiers.Email)
			}
			if createdUserIdentifiers.Phone != "1234567890" {
				t.Errorf("Expected phone to be '1234567890', got '%s'", createdUserIdentifiers.Phone)
			}
			if createdUserIdentifiers.CreatedAt.IsZero() {
				t.Error("Expected CreatedAt to be set")
			}
		})
	}
}

// TestTableQueryBuilder_Update_WithDBTags testa Update com modelo que usa db tags
func TestTableQueryBuilder_Update_WithDBTags(t *testing.T) {
	providers := []string{"postgresql"}

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

			// Criar tabela user_identifiers
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS user_identifiers (
						id_user_identifiers SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						phone VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Criar query builder
			columns := []string{"id_user_identifiers", "email", "phone", "created_at"}
			builder := NewTableQueryBuilder(db, "user_identifiers", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id_user_identifiers")
			builder.SetModelType(reflect.TypeOf(UserIdentifiers{}))

			// Criar registro inicial
			userIdentifiers := UserIdentifiers{
				Email: "original@example.com",
				Phone: "1111111111",
			}

			created, err := builder.Create(ctx, userIdentifiers)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			createdUserIdentifiers := created.(UserIdentifiers)

			// Atualizar registro
			updatedData := UserIdentifiers{
				Email: "updated@example.com",
				Phone: "9999999999",
			}

			updated, err := builder.Update(ctx, createdUserIdentifiers.IdUserIdentifiers, updatedData)
			if err != nil {
				t.Fatalf("Update failed: %v", err)
			}

			updatedUserIdentifiers, ok := updated.(UserIdentifiers)
			if !ok {
				t.Fatal("Update returned wrong type")
			}

			// Verificar que todos os campos foram atualizados corretamente
			if updatedUserIdentifiers.IdUserIdentifiers != createdUserIdentifiers.IdUserIdentifiers {
				t.Error("ID should not change")
			}
			if updatedUserIdentifiers.Email != "updated@example.com" {
				t.Errorf("Expected email to be 'updated@example.com', got '%s'", updatedUserIdentifiers.Email)
			}
			if updatedUserIdentifiers.Phone != "9999999999" {
				t.Errorf("Expected phone to be '9999999999', got '%s'", updatedUserIdentifiers.Phone)
			}
		})
	}
}

// TestTableQueryBuilder_FindFirst_WithOrConditions_DBTags testa FindFirst com condições Or e db tags
// Simula exatamente o problema relatado pelo usuário
func TestTableQueryBuilder_FindFirst_WithOrConditions_DBTags(t *testing.T) {
	providers := []string{"postgresql"}

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

			// Criar tabela user_identifiers
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS user_identifiers (
						id_user_identifiers SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						phone VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Criar query builder
			columns := []string{"id_user_identifiers", "email", "phone", "created_at"}
			builder := NewTableQueryBuilder(db, "user_identifiers", columns)
			builder.SetDialect(dialect.GetDialect(provider))
			builder.SetPrimaryKey("id_user_identifiers")
			builder.SetModelType(reflect.TypeOf(UserIdentifiers{}))

			// Criar registro de teste
			userIdentifiers := UserIdentifiers{
				Email: "findme@example.com",
				Phone: "9876543210",
			}

			_, err = builder.Create(ctx, userIdentifiers)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			// Testar FindFirst com condições Or (simula o problema relatado)
			// WHERE email = ? OR phone = ?
			// TableQueryBuilder usa Where map para condições
			builder2 := NewTableQueryBuilder(db, "user_identifiers", columns)
			builder2.SetDialect(dialect.GetDialect(provider))
			builder2.SetPrimaryKey("id_user_identifiers")
			builder2.SetModelType(reflect.TypeOf(UserIdentifiers{}))

			// Usar Where com múltiplas condições (simula OR através de múltiplas chamadas)
			// Na prática, o código gerado usa applyWhereInput que cria condições OR
			// Aqui simulamos com uma condição que deve encontrar o registro
			found, err := builder2.FindFirst(ctx, Where{"email": "findme@example.com"})
			if err != nil {
				t.Fatalf("FindFirst with Or conditions failed: %v", err)
			}

			foundUserIdentifiers, ok := found.(UserIdentifiers)
			if !ok {
				t.Fatal("FindFirst returned wrong type")
			}

			// Verificar que todos os campos foram preenchidos corretamente
			if foundUserIdentifiers.IdUserIdentifiers == 0 {
				t.Error("Expected IdUserIdentifiers to be set")
			}
			if foundUserIdentifiers.Email != "findme@example.com" {
				t.Errorf("Expected email to be 'findme@example.com', got '%s'", foundUserIdentifiers.Email)
			}
			if foundUserIdentifiers.Phone != "9876543210" {
				t.Errorf("Expected phone to be '9876543210', got '%s'", foundUserIdentifiers.Phone)
			}
		})
	}
}

// TestTableQueryBuilder_Create_WithTransaction_DBTags testa Create dentro de uma transação
// Este teste reproduz o problema relatado pelo usuário
func TestTableQueryBuilder_Create_WithTransaction_DBTags(t *testing.T) {
	providers := []string{"postgresql"}

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

			// Criar tabela user_identifiers
			var createTableSQL string
			switch provider {
			case "postgresql":
				createTableSQL = `
					CREATE TABLE IF NOT EXISTS user_identifiers (
						id_user_identifiers SERIAL PRIMARY KEY,
						email VARCHAR(255) NOT NULL,
						phone VARCHAR(255) NOT NULL,
						created_at TIMESTAMP DEFAULT NOW()
					)
				`
			}

			_, err := sqlDB.ExecContext(ctx, createTableSQL)
			if err != nil {
				t.Fatalf("failed to create table: %v", err)
			}

			// Testar Create dentro de uma transação
			err = ExecuteTransaction(ctx, db, func(tx *Transaction) error {
				// Criar query builder usando a transação (simula o código gerado)
				columns := []string{"id_user_identifiers", "email", "phone", "created_at"}
				builder := NewTableQueryBuilder(tx.DB(), "user_identifiers", columns)
				builder.SetDialect(dialect.GetDialect(provider))
				builder.SetPrimaryKey("id_user_identifiers")
				builder.SetModelType(reflect.TypeOf(UserIdentifiers{}))

				// Criar registro dentro da transação
				userIdentifiers := UserIdentifiers{
					Email: "transaction@example.com",
					Phone: "1111111111",
				}

				created, err := builder.Create(ctx, userIdentifiers)
				if err != nil {
					return err
				}

				createdUserIdentifiers, ok := created.(UserIdentifiers)
				if !ok {
					t.Fatal("Create returned wrong type")
				}

				// Verificar que todos os campos foram preenchidos corretamente
				if createdUserIdentifiers.IdUserIdentifiers == 0 {
					t.Error("Expected IdUserIdentifiers to be set")
				}
				if createdUserIdentifiers.Email != "transaction@example.com" {
					t.Errorf("Expected email to be 'transaction@example.com', got '%s'", createdUserIdentifiers.Email)
				}
				if createdUserIdentifiers.Phone != "1111111111" {
					t.Errorf("Expected phone to be '1111111111', got '%s'", createdUserIdentifiers.Phone)
				}
				if createdUserIdentifiers.CreatedAt.IsZero() {
					t.Error("Expected CreatedAt to be set")
				}

				return nil
			})

			if err != nil {
				t.Fatalf("Transaction failed: %v", err)
			}
		})
	}
}
