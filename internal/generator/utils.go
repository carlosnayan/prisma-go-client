package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateUtils generates utility files like uuid.go
func GenerateUtils(outputDir string) error {
	utilsDir := filepath.Join(outputDir, "utils")
	if err := os.MkdirAll(utilsDir, 0755); err != nil {
		return fmt.Errorf("failed to create utils directory: %w", err)
	}

	uuidFile := filepath.Join(utilsDir, "uuid.go")
	if err := generateUUIDFile(uuidFile); err != nil {
		return fmt.Errorf("failed to generate uuid.go: %w", err)
	}

	errorsFile := filepath.Join(utilsDir, "errors.go")
	if err := generateErrorsFile(errorsFile); err != nil {
		return fmt.Errorf("failed to generate errors.go: %w", err)
	}

	return nil
}

// generateErrorsFile generates the errors.go file with standardized Prisma errors
func generateErrorsFile(filePath string) error {
	return executeModelTemplate(filePath, "utils", "utils", "errors.tmpl", nil)
}

// generateUUIDFile generates the uuid.go file with GenerateUUID function using templates
func generateUUIDFile(filePath string) error {
	// Use executeModelTemplate to specify package name "utils"
	return executeModelTemplate(filePath, "utils", "utils", "uuid.tmpl", nil)
}
