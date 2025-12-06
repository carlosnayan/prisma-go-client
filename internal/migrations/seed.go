package migrations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecuteSeed executa o comando de seed configurado
func ExecuteSeed(seedCommand string) error {
	if seedCommand == "" {
		return fmt.Errorf("comando de seed não configurado")
	}

	// Dividir comando em partes
	parts := strings.Fields(seedCommand)
	if len(parts) == 0 {
		return fmt.Errorf("comando de seed vazio")
	}

	// Primeiro elemento é o comando
	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// Criar comando
	cmd := exec.Command(cmdName, args...)

	// Configurar diretório de trabalho (raiz do projeto)
	wd, err := os.Getwd()
	if err == nil {
		// Procurar raiz do projeto (onde está prisma.conf)
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

	// Capturar stdout e stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Executar
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("erro ao executar seed: %w", err)
	}

	return nil
}
