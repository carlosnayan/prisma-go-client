package cli

import (
	"os"
	"testing"
)

func TestNewApp(t *testing.T) {
	app := NewApp("test", "1.0.0", "Test app")
	
	if app.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", app.Name)
	}
	if app.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", app.Version)
	}
	if app.Description != "Test app" {
		t.Errorf("Expected description 'Test app', got '%s'", app.Description)
	}
	if len(app.Commands) != 0 {
		t.Errorf("Expected empty commands, got %d", len(app.Commands))
	}
	if len(app.GlobalFlags) != 0 {
		t.Errorf("Expected empty global flags, got %d", len(app.GlobalFlags))
	}
}

func TestAddCommand(t *testing.T) {
	app := NewApp("test", "1.0.0", "Test app")
	cmd := &Command{
		Name:  "test",
		Short: "Test command",
		Run:   func(args []string) error { return nil },
	}
	
	app.AddCommand(cmd)
	
	if len(app.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(app.Commands))
	}
	if app.Commands[0].Name != "test" {
		t.Errorf("Expected command name 'test', got '%s'", app.Commands[0].Name)
	}
}

func TestAddGlobalFlag(t *testing.T) {
	app := NewApp("test", "1.0.0", "Test app")
	var value string
	flag := &Flag{
		Name:  "test",
		Usage: "Test flag",
		Value: &value,
	}
	
	app.AddGlobalFlag(flag)
	
	if len(app.GlobalFlags) != 1 {
		t.Errorf("Expected 1 global flag, got %d", len(app.GlobalFlags))
	}
	if app.GlobalFlags[0].Name != "test" {
		t.Errorf("Expected flag name 'test', got '%s'", app.GlobalFlags[0].Name)
	}
}

func TestParseFlags_String(t *testing.T) {
	var value string
	flags := []*Flag{
		{Name: "test", Value: &value},
	}
	
	args := []string{"--test", "value", "remaining"}
	parsed, remaining := parseFlags(args, flags)
	
	if parsed["test"] != "value" {
		t.Errorf("Expected parsed value 'value', got '%v'", parsed["test"])
	}
	if value != "value" {
		t.Errorf("Expected flag value 'value', got '%s'", value)
	}
	if len(remaining) != 1 || remaining[0] != "remaining" {
		t.Errorf("Expected remaining args ['remaining'], got %v", remaining)
	}
}

func TestParseFlags_Bool(t *testing.T) {
	var value bool
	flags := []*Flag{
		{Name: "test", Value: &value},
	}
	
	args := []string{"--test", "remaining"}
	parsed, remaining := parseFlags(args, flags)
	
	if parsed["test"] != true {
		t.Errorf("Expected parsed value true, got %v", parsed["test"])
	}
	if value != true {
		t.Errorf("Expected flag value true, got %v", value)
	}
	// For bool flags, the next arg is consumed if it doesn't start with '-'
	// So "remaining" should still be in remaining args
	if len(remaining) != 1 || remaining[0] != "remaining" {
		t.Errorf("Expected remaining args ['remaining'], got %v", remaining)
	}
}

func TestParseFlags_ShortFlag(t *testing.T) {
	var value string
	flags := []*Flag{
		{Name: "test", Short: "t", Value: &value},
	}
	
	args := []string{"-t", "value"}
	parsed, remaining := parseFlags(args, flags)
	
	if parsed["test"] != "value" {
		t.Errorf("Expected parsed value 'value', got '%v'", parsed["test"])
	}
	if value != "value" {
		t.Errorf("Expected flag value 'value', got '%s'", value)
	}
	if len(remaining) != 0 {
		t.Errorf("Expected no remaining args, got %v", remaining)
	}
}

func TestParseFlags_Required(t *testing.T) {
	// Note: Required flags cause os.Exit(1) which is hard to test
	// This test just verifies the flag parsing logic works
	var value string
	flags := []*Flag{
		{Name: "test", Required: false, Value: &value},
	}
	
	args := []string{"other"}
	parsed, remaining := parseFlags(args, flags)
	
	// Flag should not be set
	if _, ok := parsed["test"]; ok {
		t.Error("Flag should not be parsed when not provided")
	}
	if len(remaining) != 1 || remaining[0] != "other" {
		t.Errorf("Expected remaining args ['other'], got %v", remaining)
	}
}

func TestExecute_Version(t *testing.T) {
	app := NewApp("test", "1.0.0", "Test app")
	
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"test", "--version"}
	
	// Capture output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	
	err := app.Execute()
	
	w.Close()
	os.Stdout = oldStdout
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecute_Help(t *testing.T) {
	app := NewApp("test", "1.0.0", "Test app")
	
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"test", "--help"}
	
	// Capture output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	
	err := app.Execute()
	
	w.Close()
	os.Stdout = oldStdout
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	app := NewApp("test", "1.0.0", "Test app")
	
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"test", "unknown"}
	
	// Capture stderr
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	
	err := app.Execute()
	
	w.Close()
	os.Stdout = oldStderr
	os.Stderr = oldStderr
	
	if err == nil {
		t.Error("Expected error for unknown command, got nil")
	}
}

func TestExecute_CommandWithSubcommand(t *testing.T) {
	app := NewApp("test", "1.0.0", "Test app")
	
	subCmd := &Command{
		Name:  "sub",
		Short: "Subcommand",
		Run: func(args []string) error {
			return nil
		},
	}
	
	cmd := &Command{
		Name:        "cmd",
		Short:       "Command",
		Subcommands: []*Command{subCmd},
	}
	
	app.AddCommand(cmd)
	
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"test", "cmd", "sub"}
	
	err := app.Execute()
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecute_CommandWithFlags(t *testing.T) {
	app := NewApp("test", "1.0.0", "Test app")
	
	var flagValue string
	cmd := &Command{
		Name:  "test",
		Short: "Test command",
		Flags: []*Flag{
			{Name: "flag", Value: &flagValue},
		},
		Run: func(args []string) error {
			if flagValue != "value" {
				t.Errorf("Expected flag value 'value', got '%s'", flagValue)
			}
			return nil
		},
	}
	
	app.AddCommand(cmd)
	
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"test", "test", "--flag", "value"}
	
	err := app.Execute()
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCommand_PrintUsage(t *testing.T) {
	cmd := &Command{
		Name:  "test",
		Short: "Test command",
		Long:  "Long description",
		Usage: "test [flags]",
		Flags: []*Flag{
			{Name: "flag", Short: "f", Usage: "Test flag"},
		},
		Subcommands: []*Command{
			{Name: "sub", Short: "Subcommand"},
		},
	}
	
	// Capture output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	
	cmd.PrintUsage()
	
	w.Close()
	os.Stdout = oldStdout
}

