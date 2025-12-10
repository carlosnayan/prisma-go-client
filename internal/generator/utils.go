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

	return nil
}

// generateUUIDFile generates the uuid.go file with GenerateUUID function
func generateUUIDFile(filePath string) error {
	file, err := createGeneratedFile(filePath, "utils")
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "import (\n")
	fmt.Fprintf(file, "\t\"math/rand\"\n")
	fmt.Fprintf(file, "\t\"time\"\n")
	fmt.Fprintf(file, ")\n\n")
	fmt.Fprintf(file, "var rng = rand.New(rand.NewSource(time.Now().UnixNano()))\n\n")
	fmt.Fprintf(file, "// GenerateUUID generates a UUID v4 without external dependencies\n")
	fmt.Fprintf(file, "// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx\n")
	fmt.Fprintf(file, "// where x is any hexadecimal digit and y is one of 8, 9, a, or b\n")
	fmt.Fprintf(file, "func GenerateUUID() string {\n")
	fmt.Fprintf(file, "\tuuid := make([]byte, 36)\n")
	fmt.Fprintf(file, "\ttemplate := \"xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx\"\n\n")
	fmt.Fprintf(file, "\tfor i, c := range template {\n")
	fmt.Fprintf(file, "\t\tswitch c {\n")
	fmt.Fprintf(file, "\t\tcase 'x':\n")
	fmt.Fprintf(file, "\t\t\tuuid[i] = \"0123456789abcdef\"[rng.Intn(16)]\n")
	fmt.Fprintf(file, "\t\tcase 'y':\n")
	fmt.Fprintf(file, "\t\t\tuuid[i] = \"89ab\"[rng.Intn(4)]\n")
	fmt.Fprintf(file, "\t\tcase '-':\n")
	fmt.Fprintf(file, "\t\t\tuuid[i] = '-'\n")
	fmt.Fprintf(file, "\t\tdefault:\n")
	fmt.Fprintf(file, "\t\t\tuuid[i] = byte(c)\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n\n")
	fmt.Fprintf(file, "\treturn string(uuid)\n")
	fmt.Fprintf(file, "}\n")

	return nil
}
