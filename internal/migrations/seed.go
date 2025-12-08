package migrations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecuteSeed executes the configured seed command
func ExecuteSeed(seedCommand string) error {
	if seedCommand == "" {
		return fmt.Errorf("seed command not configured")
	}

	// Split command into parts
	parts := strings.Fields(seedCommand)
	if len(parts) == 0 {
		return fmt.Errorf("seed command is empty")
	}

	// First element is the command
	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// Create command
	cmd := exec.Command(cmdName, args...)

	// Set working directory (project root)
	wd, err := os.Getwd()
	if err == nil {
		// Find project root (where prisma.conf is)
		dir := wd
		for {
			configPath := filepath.Join(dir, "prisma.conf")
			if _, err := os.Stat(configPath); err == nil {
				cmd.Dir = dir
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				cmd.Dir = wd
				break
			}
			dir = parent
		}
	}

	// Capture stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Execute
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error executing seed: %w", err)
	}

	return nil
}
