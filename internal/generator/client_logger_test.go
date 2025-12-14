package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func TestGenerateLoggerConfigHelper_ReadsDebugSection(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a minimal schema for testing
	schema := &parser.Schema{
		Datasources: []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{
						Name:  "provider",
						Value: "postgresql",
					},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "User",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
				},
			},
		},
	}

	// Generate client code
	err := GenerateClient(schema, outputDir)
	if err != nil {
		t.Fatalf("Failed to generate client: %v", err)
	}

	// Read the generated client file
	clientPath := filepath.Join(outputDir, "client.go")
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read generated client file: %v", err)
	}

	contentStr := string(content)

	// Verify configureLoggerFromConfig function exists
	if !strings.Contains(contentStr, "func configureLoggerFromConfig()") {
		t.Error("Generated client should contain configureLoggerFromConfig function")
	}

	if !strings.Contains(contentStr, "prisma.conf") {
		t.Error("configureLoggerFromConfig should search for prisma.conf")
	}

	if !strings.Contains(contentStr, "parseLogLevels") {
		t.Error("configureLoggerFromConfig should use parseLogLevels function")
	}

	if !strings.Contains(contentStr, "loadDotEnvFile") {
		t.Error("configureLoggerFromConfig should use loadDotEnvFile function")
	}

	if !strings.Contains(contentStr, "builder.SetLogLevels") {
		t.Error("configureLoggerFromConfig should call builder.SetLogLevels")
	}

	if !strings.Contains(contentStr, "if len(logLevels) > 0") {
		t.Error("configureLoggerFromConfig should check if log levels are specified")
	}
}

func TestGenerateLoggerConfigHelper_IsCalledInNewClient(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "generated")

	// Create a temporary go.mod file for module detection
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := "module test\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a minimal schema for testing
	schema := &parser.Schema{
		Datasources: []*parser.Datasource{
			{
				Name: "db",
				Fields: []*parser.Field{
					{
						Name:  "provider",
						Value: "postgresql",
					},
				},
			},
		},
		Models: []*parser.Model{
			{
				Name: "User",
				Fields: []*parser.ModelField{
					{
						Name: "id",
						Type: &parser.FieldType{Name: "String"},
						Attributes: []*parser.Attribute{
							{Name: "id"},
						},
					},
				},
			},
		},
	}

	// Generate client code
	err := GenerateClient(schema, outputDir)
	if err != nil {
		t.Fatalf("Failed to generate client: %v", err)
	}

	// Read the generated client file
	clientPath := filepath.Join(outputDir, "client.go")
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read generated client file: %v", err)
	}

	contentStr := string(content)

	// Verify NewClient calls configureLoggerFromConfig
	if !strings.Contains(contentStr, "func NewClient(") {
		t.Error("Generated client should contain NewClient function")
	}

	// Find NewClient function
	newClientStart := strings.Index(contentStr, "func NewClient(")
	if newClientStart == -1 {
		t.Fatal("NewClient function not found")
	}

	// Check that configureLoggerFromConfig is called early in NewClient
	newClientBody := contentStr[newClientStart:]
	if !strings.Contains(newClientBody, "configureLoggerFromConfig()") {
		t.Error("NewClient should call configureLoggerFromConfig()")
	}

	// Verify it's called before client initialization
	configureCall := strings.Index(newClientBody, "configureLoggerFromConfig()")
	clientInit := strings.Index(newClientBody, "client := &Client{")
	if configureCall == -1 || clientInit == -1 {
		t.Fatal("Could not find configureLoggerFromConfig or client initialization")
	}
	if configureCall > clientInit {
		t.Error("configureLoggerFromConfig should be called before client initialization")
	}
}
