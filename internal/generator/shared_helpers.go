package generator

import (
	"fmt"
	"os"
)

// generateHelperFunctions generates common helper functions (toSnakeCase, contains)
// These functions are shared across generated builder code
func generateHelperFunctions(file *os.File) {
	fmt.Fprintf(file, "// Helper functions\n\n")

	fmt.Fprintf(file, "// contains checks if a string slice contains a specific item\n")
	fmt.Fprintf(file, "func contains(slice []string, item string) bool {\n")
	fmt.Fprintf(file, "\tfor _, s := range slice {\n")
	fmt.Fprintf(file, "\t\tif s == item {\n")
	fmt.Fprintf(file, "\t\t\treturn true\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn false\n")
	fmt.Fprintf(file, "}\n\n")

	fmt.Fprintf(file, "// toSnakeCase converts PascalCase to snake_case\n")
	fmt.Fprintf(file, "func toSnakeCase(s string) string {\n")
	fmt.Fprintf(file, "\tvar result strings.Builder\n")
	fmt.Fprintf(file, "\tfor i, r := range s {\n")
	fmt.Fprintf(file, "\t\tif i > 0 && r >= 'A' && r <= 'Z' {\n")
	fmt.Fprintf(file, "\t\t\tresult.WriteByte('_')\n")
	fmt.Fprintf(file, "\t\t}\n")
	fmt.Fprintf(file, "\t\tresult.WriteRune(r)\n")
	fmt.Fprintf(file, "\t}\n")
	fmt.Fprintf(file, "\treturn strings.ToLower(result.String())\n")
	fmt.Fprintf(file, "}\n\n")
}
