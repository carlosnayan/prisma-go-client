package cli

import (
	"fmt"
	"os"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Short       string
	Long        string
	Usage       string
	Run         func(args []string) error
	Subcommands []*Command
	Flags       []*Flag
}

// Flag represents a command flag
type Flag struct {
	Name     string
	Short    string
	Long     string
	Usage    string
	Required bool
	Value    interface{} // *string, *bool, *int, etc.
}

// App represents the CLI application
type App struct {
	Name        string
	Version     string
	Description string
	Commands    []*Command
	GlobalFlags []*Flag
}

// NewApp creates a new CLI application
func NewApp(name, version, description string) *App {
	return &App{
		Name:        name,
		Version:     version,
		Description: description,
		Commands:    []*Command{},
		GlobalFlags: []*Flag{},
	}
}

// AddCommand adds a command to the app
func (a *App) AddCommand(cmd *Command) {
	a.Commands = append(a.Commands, cmd)
}

// AddGlobalFlag adds a global flag to the app
func (a *App) AddGlobalFlag(flag *Flag) {
	a.GlobalFlags = append(a.GlobalFlags, flag)
}

// Execute runs the CLI application
func (a *App) Execute() error {
	args := os.Args[1:]

	if len(args) == 0 {
		a.printUsage()
		return nil
	}

	// Check for version flag
	if args[0] == "--version" || args[0] == "-v" {
		fmt.Printf("%s version %s\n", a.Name, a.Version)
		return nil
	}

	// Check for help flag
	if args[0] == "--help" || args[0] == "-h" {
		a.printUsage()
		return nil
	}

	// Parse global flags
	globalArgs, remainingArgs := parseFlags(args, a.GlobalFlags)
	_ = globalArgs // Store parsed global flags if needed

	// Find and execute command
	if len(remainingArgs) == 0 {
		a.printUsage()
		return nil
	}

	cmdName := remainingArgs[0]
	cmdArgs := remainingArgs[1:]

	// Find command
	var cmd *Command
	for _, c := range a.Commands {
		if c.Name == cmdName {
			cmd = c
			break
		}
	}

	if cmd == nil {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmdName)
		a.printUsage()
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	// Check for subcommands
	if len(cmdArgs) > 0 && len(cmd.Subcommands) > 0 {
		subCmdName := cmdArgs[0]
		// Check if it's not a flag
		if subCmdName[0] != '-' {
			for _, subCmd := range cmd.Subcommands {
				if subCmd.Name == subCmdName {
					// Parse flags for subcommand
					_, subCmdArgs := parseFlags(cmdArgs[1:], subCmd.Flags)
					if subCmd.Run == nil {
						return fmt.Errorf("subcommand %s has no run function", subCmdName)
					}
					return subCmd.Run(subCmdArgs)
				}
			}
		}
	}

	// If command has subcommands but no run function, show usage
	if len(cmd.Subcommands) > 0 && cmd.Run == nil {
		cmd.PrintUsage()
		return nil
	}

	// Parse flags for command
	_, finalArgs := parseFlags(cmdArgs, cmd.Flags)
	if cmd.Run == nil {
		return fmt.Errorf("command %s has no run function", cmdName)
	}
	return cmd.Run(finalArgs)
}

// parseFlags parses flags from arguments and returns parsed flags and remaining args
func parseFlags(args []string, flags []*Flag) (map[string]interface{}, []string) {
	parsed := make(map[string]interface{})
	remaining := []string{}
	i := 0

	for i < len(args) {
		arg := args[i]

		// Check if it's a flag
		if len(arg) > 1 && arg[0] == '-' {
			flagName := arg[1:]
			if len(arg) > 2 && arg[1] == '-' {
				flagName = arg[2:]
			}

			// Find flag
			var flag *Flag
			for _, f := range flags {
				if f.Name == flagName || f.Short == flagName {
					flag = f
					break
				}
			}

			if flag != nil {
				// Check if it's a bool flag
				_, isBool := flag.Value.(*bool)

				// Get value
				if !isBool && i+1 < len(args) && args[i+1][0] != '-' {
					value := args[i+1]
					parsed[flag.Name] = value
					setFlagValue(flag, value)
					i += 2
				} else {
					// Boolean flag or no value provided
					parsed[flag.Name] = true
					setFlagValue(flag, "true")
					i++
				}
			} else {
				// Unknown flag, treat as argument
				remaining = append(remaining, arg)
				i++
			}
		} else {
			remaining = append(remaining, arg)
			i++
		}
	}

	// Check required flags
	for _, flag := range flags {
		if flag.Required {
			if _, ok := parsed[flag.Name]; !ok {
				fmt.Fprintf(os.Stderr, "Error: flag --%s is required\n", flag.Name)
				os.Exit(1)
			}
		}
	}

	return parsed, remaining
}

// setFlagValue sets the value of a flag
func setFlagValue(flag *Flag, value string) {
	switch v := flag.Value.(type) {
	case *string:
		*v = value
	case *bool:
		*v = value == "true" || value == "1"
	case *int:
		_, _ = fmt.Sscanf(value, "%d", v)
	default:
		// Try to set as string
		if s, ok := flag.Value.(*string); ok {
			*s = value
		}
	}
}

// printUsage prints the usage information
func (a *App) printUsage() {
	fmt.Printf("%s - %s\n\n", a.Name, a.Description)
	fmt.Printf("Usage:\n  %s [command] [flags] [arguments]\n\n", a.Name)

	if len(a.Commands) > 0 {
		fmt.Println("Commands:")
		for _, cmd := range a.Commands {
			fmt.Printf("  %-15s %s\n", cmd.Name, cmd.Short)
		}
		fmt.Println()
	}

	if len(a.GlobalFlags) > 0 {
		fmt.Println("Global Flags:")
		for _, flag := range a.GlobalFlags {
			short := ""
			if flag.Short != "" {
				short = fmt.Sprintf("-%s, ", flag.Short)
			}
			fmt.Printf("  %s--%s\t%s\n", short, flag.Name, flag.Usage)
		}
		fmt.Println()
	}

	fmt.Printf("Use '%s [command] --help' for more information about a command.\n", a.Name)
}

// PrintCommandUsage prints usage for a specific command
func (cmd *Command) PrintUsage() {
	if cmd.Long != "" {
		fmt.Println(cmd.Long)
		fmt.Println()
	}

	if cmd.Usage != "" {
		fmt.Printf("Usage:\n  %s\n\n", cmd.Usage)
	} else {
		fmt.Printf("Usage:\n  %s\n\n", cmd.Name)
	}

	if len(cmd.Flags) > 0 {
		fmt.Println("Flags:")
		for _, flag := range cmd.Flags {
			short := ""
			if flag.Short != "" {
				short = fmt.Sprintf("-%s, ", flag.Short)
			}
			required := ""
			if flag.Required {
				required = " (required)"
			}
			fmt.Printf("  %s--%s\t%s%s\n", short, flag.Name, flag.Usage, required)
		}
		fmt.Println()
	}

	if len(cmd.Subcommands) > 0 {
		fmt.Println("Subcommands:")
		for _, subCmd := range cmd.Subcommands {
			fmt.Printf("  %-15s %s\n", subCmd.Name, subCmd.Short)
		}
		fmt.Println()
	}
}
