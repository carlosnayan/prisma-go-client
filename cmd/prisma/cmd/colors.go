package cmd

import (
	"os"
	"runtime"
)

// ANSI color codes
const (
	Reset = "\033[0m"
	Gray  = "\033[90m" // Bright black
	Cyan  = "\033[36m"
	Red   = "\033[31m"
	Green = "\033[32m"
)

var (
	// colorsEnabled is cached result of supportsColor()
	colorsEnabled = -1
)

// supportsColor checks if the terminal supports colors
func supportsColor() bool {
	// Cache the result
	if colorsEnabled != -1 {
		return colorsEnabled == 1
	}

	// Check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = 0
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "dumb" {
		colorsEnabled = 0
		return false
	}

	// On Windows, check if we're in a terminal that supports ANSI
	// Modern Windows 10+ terminals support ANSI colors
	if runtime.GOOS == "windows" {
		// Check if we're in a modern terminal (Windows Terminal, PowerShell, etc.)
		// If TERM is not set or is set to something other than "dumb", assume colors are supported
		if term == "" {
			// On Windows, if TERM is not set, check for other indicators
			// For now, assume modern Windows terminals support colors
			colorsEnabled = 1
			return true
		}
	}

	// Check if stdout is a terminal
	// This is a simple check - in most cases, if we got here, colors are likely supported
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		colorsEnabled = 0
		return false
	}

	// Check if it's a character device (terminal)
	mode := fileInfo.Mode()
	if (mode & os.ModeCharDevice) != 0 {
		colorsEnabled = 1
		return true
	}

	// Default: assume colors are supported if TERM is set and not "dumb"
	if term != "" {
		colorsEnabled = 1
		return true
	}

	colorsEnabled = 0
	return false
}

// Info returns text colored in gray for informational messages
func Info(text string) string {
	if !supportsColor() {
		return text
	}
	return Gray + text + Reset
}

// Prompt returns text colored in cyan for prompt symbols
func Prompt(text string) string {
	if !supportsColor() {
		return text
	}
	return Cyan + text + Reset
}

// Warning returns text colored in red for warnings and critical messages
func Warning(text string) string {
	if !supportsColor() {
		return text
	}
	return Red + text + Reset
}

// PromptText returns text colored in gray for prompt text
func PromptText(text string) string {
	if !supportsColor() {
		return text
	}
	return Gray + text + Reset
}

// Success returns text colored in green for success messages
func Success(text string) string {
	if !supportsColor() {
		return text
	}
	return Green + text + Reset
}

// MigrationName returns text colored in cyan for migration names
func MigrationName(text string) string {
	if !supportsColor() {
		return text
	}
	return Cyan + text + Reset
}
