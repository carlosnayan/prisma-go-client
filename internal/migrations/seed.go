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

	seedCommand = strings.TrimSpace(seedCommand)

	if idx := strings.Index(seedCommand, "#"); idx != -1 {
		seedCommand = strings.TrimSpace(seedCommand[:idx])
	}

	if len(seedCommand) >= 2 {
		if (seedCommand[0] == '"' && seedCommand[len(seedCommand)-1] == '"') ||
			(seedCommand[0] == '\'' && seedCommand[len(seedCommand)-1] == '\'') {
			seedCommand = seedCommand[1 : len(seedCommand)-1]
		}
	}

	seedCommand = strings.ReplaceAll(seedCommand, `\"`, `"`)
	seedCommand = strings.ReplaceAll(seedCommand, `\'`, `'`)

	parts := strings.Fields(seedCommand)
	if len(parts) == 0 {
		return fmt.Errorf("seed command is empty")
	}

	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	cmd := exec.Command(cmdName, args...)

	wd, err := os.Getwd()
	if err == nil {
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

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error executing seed: %w", err)
	}

	return nil
}
