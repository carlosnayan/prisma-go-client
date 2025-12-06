package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

// Config representa a configuração completa do Prisma (equivalente ao prisma.config.ts v7)
type Config struct {
	Schema     string            `toml:"schema"`     // Caminho para schema.prisma
	Migrations *MigrationsConfig `toml:"migrations"`  // Configuração de migrations
	Datasource *DatasourceConfig `toml:"datasource"` // Configuração do banco de dados
	Generator  *GeneratorConfig   `toml:"generator,omitempty"` // Configuração do gerador (opcional, pode estar no schema)
	Log        []string          `toml:"log,omitempty"` // Níveis de log: query, info, warn, error
}

// MigrationsConfig configura as migrations
type MigrationsConfig struct {
	Path string `toml:"path"` // Caminho para migrations (ex: "prisma/migrations")
	Seed string `toml:"seed,omitempty"` // Script de seed (ex: "go run prisma/seed.go")
}

// DatasourceConfig configura a fonte de dados
type DatasourceConfig struct {
	URL              string `toml:"url"` // URL do banco (pode usar env("DATABASE_URL") ou ${DATABASE_URL})
	ShadowDatabaseURL string `toml:"shadowDatabaseUrl,omitempty"`
}

// GeneratorConfig configura a geração de código
type GeneratorConfig struct {
	Provider        string   `toml:"provider"` // prisma-client-go
	Output          string   `toml:"output"`
	PreviewFeatures []string `toml:"previewFeatures,omitempty"`
}

// Load carrega a configuração do arquivo prisma.conf
func Load(configPath string) (*Config, error) {
	// Carregar arquivo .env se existir (ignora erro se não existir)
	// Procura .env na raiz do projeto, subindo os diretórios
	wd, _ := os.Getwd()
	if wd != "" {
		dir := wd
		for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			// Carregar .env encontrado (ignore errors - optional file)
			_ = godotenv.Load(envPath)
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Não encontrou .env, tenta carregar do diretório atual
			_ = godotenv.Load()
			break
		}
		dir = parent
	}
} else {
	// Fallback: tentar carregar .env do diretório atual (ignore errors - optional file)
	_ = godotenv.Load()
}

	if configPath == "" {
		// Procurar prisma.conf na raiz do projeto
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("erro ao obter diretório atual: %w", err)
		}

		// Procurar subindo os diretórios
		dir := wd
		for {
			configPath = filepath.Join(dir, "prisma.conf")
			if _, err := os.Stat(configPath); err == nil {
				break
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				return nil, fmt.Errorf("prisma.conf não encontrado")
			}
			dir = parent
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler prisma.conf: %w", err)
	}

	var config Config
	if _, err := toml.Decode(string(data), &config); err != nil {
		return nil, fmt.Errorf("erro ao parsear prisma.conf: %w", err)
	}

	// Expandir variáveis de ambiente
	if err := config.expandEnvVars(); err != nil {
		return nil, fmt.Errorf("erro ao expandir variáveis de ambiente: %w", err)
	}

	// Validar configuração
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuração inválida: %w", err)
	}

	return &config, nil
}

// expandEnvVars expande variáveis de ambiente no formato ${VAR}, $VAR ou env("VAR")
func (c *Config) expandEnvVars() error {
	if c.Datasource != nil {
		c.Datasource.URL = expandString(c.Datasource.URL)
		c.Datasource.ShadowDatabaseURL = expandString(c.Datasource.ShadowDatabaseURL)
	}

	if c.Migrations != nil {
		c.Migrations.Seed = expandString(c.Migrations.Seed)
	}

	return nil
}

// expandString expande variáveis de ambiente em uma string
// Suporta: ${VAR}, $VAR, env("VAR") e env('VAR')
func expandString(s string) string {
	// Primeiro, expandir formato env("VAR") ou env('VAR')
	for {
		var start int
		var endQuote string
		
		// Tentar encontrar env(" ou env('
		if idx := strings.Index(s, `env("`); idx != -1 {
			start = idx
			endQuote = `")`
		} else if idx := strings.Index(s, `env('`); idx != -1 {
			start = idx
			endQuote = `')`
		} else {
			break
		}
		
		end := strings.Index(s[start+5:], endQuote)
		if end == -1 {
			break
		}
		end += start + 5

		varName := s[start+5 : end]
		value := os.Getenv(varName)
		s = s[:start] + value + s[end+2:]
	}

	// Depois, expandir ${VAR} e $VAR
	s = os.ExpandEnv(s)

	// Expandir formato ${VAR} explicitamente (caso ExpandEnv não tenha pego)
	for {
		start := strings.Index(s, "${")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "}")
		if end == -1 {
			break
		}
		end += start

		varName := s[start+2 : end]
		value := os.Getenv(varName)
		s = s[:start] + value + s[end+1:]
	}

	return s
}

// Validate valida a configuração
func (c *Config) Validate() error {
	if c.Schema == "" {
		c.Schema = "prisma/schema.prisma" // Padrão
	}

	if c.Migrations == nil {
		c.Migrations = &MigrationsConfig{
			Path: "prisma/migrations", // Padrão
		}
	} else if c.Migrations.Path == "" {
		c.Migrations.Path = "prisma/migrations" // Padrão
	}

	if c.Datasource == nil {
		return fmt.Errorf("datasource é obrigatório")
	}

	if c.Datasource.URL == "" {
		return fmt.Errorf("datasource.url é obrigatório (use env(\"DATABASE_URL\") ou ${DATABASE_URL})")
	}

	// Validar se a URL foi expandida corretamente (não deve conter env(" ou ${ sem ser expandido)
	if strings.Contains(c.Datasource.URL, `env("`) || (strings.Contains(c.Datasource.URL, "${") && !strings.Contains(c.Datasource.URL, "://")) {
		// Se ainda contém env(" ou ${, significa que a variável não foi encontrada
		if strings.Contains(c.Datasource.URL, `env("DATABASE_URL")`) || c.Datasource.URL == "${DATABASE_URL}" {
			return fmt.Errorf("variável de ambiente DATABASE_URL não está definida")
		}
	}

	return nil
}

// GetSchemaPath retorna o caminho do schema.prisma
func (c *Config) GetSchemaPath() string {
	if c.Schema != "" {
		return c.Schema
	}
	// Padrão: prisma/schema.prisma
	if _, err := os.Stat("prisma/schema.prisma"); err == nil {
		return "prisma/schema.prisma"
	}
	// Fallback: schema.prisma na raiz
	if _, err := os.Stat("schema.prisma"); err == nil {
		return "schema.prisma"
	}
	return "prisma/schema.prisma"
}

// GetMigrationsPath retorna o caminho das migrations
func (c *Config) GetMigrationsPath() string {
	if c.Migrations != nil && c.Migrations.Path != "" {
		return c.Migrations.Path
	}
	return "prisma/migrations"
}

// GetDatabaseURL retorna a URL do banco de dados (já expandida)
func (c *Config) GetDatabaseURL() string {
	if c.Datasource != nil {
		return c.Datasource.URL
	}
	return ""
}

