package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosnayan/prisma-go-client/cli"
)

var (
	providerFlag string
	databaseFlag string
)

var initCmd = &cli.Command{
	Name:  "init",
	Short: "Initialize a new Prisma project",
	Long: `Creates the initial structure of a Prisma project:
  - prisma.conf file with default configuration
  - schema.prisma file with basic example
  - prisma/migrations/ directory`,
	Flags: []*cli.Flag{
		{
			Name:  "provider",
			Short: "p",
			Usage: "Database provider (postgresql, mysql, sqlite)",
			Value: &providerFlag,
		},
		{
			Name:  "database",
			Short: "d",
			Usage: "Database connection URL",
			Value: &databaseFlag,
		},
	},
	Run: runInit,
}

func runInit(args []string) error {
	fmt.Println("Initializing Prisma project...")
	fmt.Println()

	// Check if prisma.conf already exists
	if _, err := os.Stat("prisma.conf"); err == nil {
		return fmt.Errorf("prisma.conf already exists in this directory. Use 'prisma init --force' to overwrite")
	}

	// Create prisma directory if it doesn't exist
	if err := os.MkdirAll("prisma/migrations", 0755); err != nil {
		return fmt.Errorf("error creating prisma/migrations directory: %w", err)
	}

	provider := providerFlag
	if provider == "" {
		// TODO: Interactive mode to choose provider
		provider = "postgresql"
	}

	// Create prisma.conf
	configContent := generateConfig(provider)
	if err := os.WriteFile("prisma.conf", []byte(configContent), 0644); err != nil {
		return fmt.Errorf("error creating prisma.conf: %w", err)
	}
	fmt.Println("Created prisma.conf")

	// Create schema.prisma
	schemaContent := generateSchema(provider)
	schemaPath := filepath.Join("prisma", "schema.prisma")
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		return fmt.Errorf("error creating schema.prisma: %w", err)
	}
	fmt.Printf("Created %s\n", schemaPath)

	fmt.Println()
	fmt.Println("Project initialized successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Configure the DATABASE_URL environment variable:")
	fmt.Println("     export DATABASE_URL=\"postgresql://user:password@localhost:5432/mydb\"")
	fmt.Println("  2. Edit prisma/schema.prisma with your models")
	fmt.Println("  3. Run 'prisma generate' to generate types")
	fmt.Println("  4. Run 'prisma migrate dev' to create migrations")

	return nil
}

func generateConfig(provider string) string {
	// Sempre usar variável de ambiente para a URL
	// O usuário deve definir DATABASE_URL no ambiente
	url := `env("DATABASE_URL")`

	// Se o usuário forneceu uma URL via flag, podemos usar diretamente
	// mas o padrão é usar env("DATABASE_URL")
	if databaseFlag != "" {
		url = databaseFlag
	}

	// Escapar corretamente para TOML
	urlEscaped := url
	if url == `env("DATABASE_URL")` {
		// Usar formato que o TOML aceita: aspas simples ou formato alternativo
		urlEscaped = `env('DATABASE_URL')`
	}

	return fmt.Sprintf(`# Prisma for Go Configuration
# Equivalent to Prisma v7's prisma.config.ts
# Database URL is read from the DATABASE_URL environment variable

schema = "prisma/schema.prisma"

[migrations]
path = "prisma/migrations"
# seed = "go run prisma/seed.go"  # Uncomment and configure if needed

[datasource]
url = %q
`, urlEscaped)
}

func generateSchema(provider string) string {
	return `// Database schema
// Database URL is configured in prisma.conf, not here

datasource db {
  provider = "` + provider + `"
}

generator client {
  provider = "prisma-client-go"
  output   = "../db"
}
`
}
