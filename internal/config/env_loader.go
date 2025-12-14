package config

import (
	"bufio"
	"os"
	"strings"
)

func loadDotEnv(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if len(value) > 0 {
			firstChar := value[0]
			if firstChar == '"' || firstChar == '\'' {
				endQuoteIdx := strings.Index(value[1:], string(firstChar))
				if endQuoteIdx != -1 {
					value = value[1 : endQuoteIdx+1]
				} else {
					value = strings.Trim(value, "\"'")
				}
			} else {
				if commentIdx := strings.Index(value, "#"); commentIdx != -1 {
					value = strings.TrimSpace(value[:commentIdx])
				}
			}
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}
