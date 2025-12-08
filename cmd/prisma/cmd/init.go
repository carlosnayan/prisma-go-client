package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosnayan/prisma-go-client/cli"
)

var (
	providerFlag string
	databaseFlag string
)

// Supported providers
var supportedProviders = []string{
	"postgresql",
	"mysql",
	"sqlite",
}

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

	// Validate that we're not in an existing Prisma project
	if err := validateNoExistingProject(); err != nil {
		return err
	}

	// Get provider
	provider := providerFlag
	if provider != "" {
		if !isValidProvider(provider) {
			return fmt.Errorf(`provider "%s" is invalid or not supported. Try again with %s`,
				provider, strings.Join(supportedProviders, ", "))
		}
	} else {
		provider = promptForProvider()
	}

	// Normalize provider name
	provider = normalizeProvider(provider)

	// Create prisma directory if it doesn't exist
	if err := os.MkdirAll("prisma/migrations", 0755); err != nil {
		return fmt.Errorf("error creating prisma/migrations directory: %w", err)
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
	fmt.Printf("     export DATABASE_URL=\"%s\"\n", getDefaultURL(provider))
	fmt.Println("  2. Edit prisma/schema.prisma with your models")
	fmt.Println("  3. Run 'prisma generate' to generate types")
	fmt.Println("  4. Run 'prisma migrate dev' to create migrations")

	return nil
}

// validateNoExistingProject checks if we're already in a Prisma project
func validateNoExistingProject() error {
	// Check for schema.prisma in root
	if _, err := os.Stat("schema.prisma"); err == nil {
		return fmt.Errorf(`file %s already exists in your project.
Please try again in a project that is not yet using Prisma.`, "schema.prisma")
	}

	// Check for prisma folder
	if _, err := os.Stat("prisma"); err == nil {
		return fmt.Errorf(`a folder called %s already exists in your project.
Please try again in a project that is not yet using Prisma.`, "prisma")
	}

	// Check for prisma/schema.prisma
	schemaPath := filepath.Join("prisma", "schema.prisma")
	if _, err := os.Stat(schemaPath); err == nil {
		return fmt.Errorf(`file %s already exists in your project.
Please try again in a project that is not yet using Prisma.`, schemaPath)
	}

	// Check for prisma.conf
	if _, err := os.Stat("prisma.conf"); err == nil {
		return fmt.Errorf(`file %s already exists in this directory.
Please try again in a project that is not yet using Prisma.`, "prisma.conf")
	}

	return nil
}

// isValidProvider checks if a provider is supported
func isValidProvider(provider string) bool {
	provider = strings.ToLower(provider)
	for _, p := range supportedProviders {
		if p == provider || p == normalizeProvider(provider) {
			return true
		}
	}
	return false
}

// normalizeProvider normalizes provider names (e.g., "postgres" -> "postgresql")
func normalizeProvider(provider string) string {
	provider = strings.ToLower(provider)
	switch provider {
	case "postgres", "postgresql":
		return "postgresql"
	case "mysql", "mariadb":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite"
	default:
		return provider
	}
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

[debug]
log = ["warn","error"]
`, urlEscaped)
}

func generateSchema(provider string) string {
	// Base schema with comments consistent with official Prisma
	schema := `// This is your Prisma schema file,
// learn more about it in the docs: https://pris.ly/d/prisma-schema

generator client {
  provider = "prisma-client-go"
  output   = "./generated"
}

datasource db {
  provider = "` + provider + `"
}
`

	return schema
}

// getDefaultURL returns a default connection URL for the provider
func getDefaultURL(provider string) string {
	switch provider {
	case "postgresql":
		return "postgresql://johndoe:randompassword@localhost:5432/mydb?schema=public"
	case "mysql":
		return "mysql://johndoe:randompassword@localhost:3306/mydb"
	case "sqlite":
		return "file:./dev.db"
	default:
		return "postgresql://johndoe:randompassword@localhost:5432/mydb?schema=public"
	}
}

// promptForProvider prompts the user to choose a database provider interactively
func promptForProvider() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Select a database provider:")
	fmt.Println("  1) PostgreSQL")
	fmt.Println("  2) MySQL")
	fmt.Println("  3) SQLite")
	fmt.Print("Enter choice (1-3) [default: 1]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input, defaulting to PostgreSQL")
		return "postgresql"
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "postgresql"
	}

	switch input {
	case "1", "postgresql", "postgres":
		return "postgresql"
	case "2", "mysql":
		return "mysql"
	case "3", "sqlite":
		return "sqlite"
	default:
		fmt.Printf("Invalid choice '%s', defaulting to PostgreSQL\n", input)
		return "postgresql"
	}
}
