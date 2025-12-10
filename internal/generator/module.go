package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// detectUserModule detects the user's Go module name by finding and reading go.mod
func detectUserModule(outputDir string) (string, error) {
	// Find go.mod by traversing up from outputDir
	dir := outputDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Read go.mod
			data, err := os.ReadFile(goModPath)
			if err != nil {
				return "", fmt.Errorf("failed to read go.mod: %w", err)
			}
			// Extract module name (first line after "module ")
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
					// Remove any comments
					if idx := strings.Index(moduleName, "//"); idx != -1 {
						moduleName = strings.TrimSpace(moduleName[:idx])
					}
					return moduleName, nil
				}
			}
			return "", fmt.Errorf("module declaration not found in go.mod")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found")
}

// findModuleRoot finds the directory containing go.mod
func findModuleRoot(outputDir string) (string, error) {
	dir := outputDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found")
}

// calculateImportPath calculates the import path for generated packages
// Returns the full import path like "userModule/generated/models"
func calculateImportPath(userModule, outputDir string) (modelsPath, queriesPath, inputsPath string, err error) {
	moduleRoot, err := findModuleRoot(outputDir)
	if err != nil {
		return "", "", "", err
	}

	// Calculate relative path from module root to outputDir
	relPath, err := filepath.Rel(moduleRoot, outputDir)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Normalize path to use forward slashes (even on Windows)
	importBase := filepath.ToSlash(relPath)
	if importBase == "." {
		importBase = ""
	}

	// Build import paths
	if importBase != "" {
		modelsPath = fmt.Sprintf("%s/%s/models", userModule, importBase)
		queriesPath = fmt.Sprintf("%s/%s/queries", userModule, importBase)
		inputsPath = fmt.Sprintf("%s/%s/inputs", userModule, importBase)
	} else {
		modelsPath = fmt.Sprintf("%s/models", userModule)
		queriesPath = fmt.Sprintf("%s/queries", userModule)
		inputsPath = fmt.Sprintf("%s/inputs", userModule)
	}

	return modelsPath, queriesPath, inputsPath, nil
}

// calculateLocalImportPath calculates the import path for local packages (builder, raw)
// Returns the full import path like "userModule/generated/builder" and "userModule/generated/raw"
func calculateLocalImportPath(userModule, outputDir string) (builderPath, rawPath string, err error) {
	moduleRoot, err := findModuleRoot(outputDir)
	if err != nil {
		return "", "", err
	}

	// Calculate relative path from module root to outputDir
	relPath, err := filepath.Rel(moduleRoot, outputDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Normalize path to use forward slashes (even on Windows)
	importBase := filepath.ToSlash(relPath)
	if importBase == "." {
		importBase = ""
	}

	// Build import paths
	if importBase != "" {
		builderPath = fmt.Sprintf("%s/%s/builder", userModule, importBase)
		rawPath = fmt.Sprintf("%s/%s/raw", userModule, importBase)
	} else {
		builderPath = fmt.Sprintf("%s/builder", userModule)
		rawPath = fmt.Sprintf("%s/raw", userModule)
	}

	return builderPath, rawPath, nil
}

// calculateUtilsImportPath calculates the import path for utils package
// Returns the full import path like "userModule/generated/utils"
func calculateUtilsImportPath(userModule, outputDir string) (string, error) {
	moduleRoot, err := findModuleRoot(outputDir)
	if err != nil {
		return "", err
	}

	// Calculate relative path from module root to outputDir
	relPath, err := filepath.Rel(moduleRoot, outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Normalize path to use forward slashes (even on Windows)
	importBase := filepath.ToSlash(relPath)
	if importBase == "." {
		importBase = ""
	}

	// Build import path
	var utilsPath string
	if importBase != "" {
		utilsPath = fmt.Sprintf("%s/%s/utils", userModule, importBase)
	} else {
		utilsPath = fmt.Sprintf("%s/utils", userModule)
	}

	return utilsPath, nil
}
