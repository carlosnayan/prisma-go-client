package generator

import (
	"os"
	"path/filepath"
)

// generateDBInterfaces generates the common database interfaces (DB, Result, Rows, Row)
// These interfaces are shared between raw and builder packages
func generateDBInterfaces(file *os.File) error {
	_, templatesDir, err := getTemplatesDir("shared")
	if err != nil {
		return err
	}
	tmplPath := filepath.Join(templatesDir, "interfaces.tmpl")
	return executeTemplate(file, tmplPath, nil)
}
