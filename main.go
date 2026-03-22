package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			if err := runInit(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "config":
			fmt.Println(configPath())
			return
		case "dry-run":
			if err := runDryRun(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sel, err := runTUI(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cancelled.\n")
		os.Exit(0)
	}

	args := buildCommand(cfg, sel)

	// If skills were selected, create a config override so Claude only sees those skills
	var configOverride string
	if len(sel.SkillPaths) > 0 {
		override, err := createConfigOverride(sel.SkillPaths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create config override: %v\n", err)
		} else {
			configOverride = override
		}
	}

	printCommand(args)
	launchClaude(args, configOverride)
}

func runInit() error {
	path := configPath()
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Config already exists at %s\n", path)
		fmt.Println("Delete it first if you want to regenerate.")
		return nil
	}
	if err := writeDefaultConfig(); err != nil {
		return err
	}
	fmt.Printf("Config created at %s\n", path)
	fmt.Println("Edit it to add your system prompts, skill directories, and presets.")
	return nil
}

func runDryRun() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	sel, err := runTUI(cfg)
	if err != nil {
		return fmt.Errorf("cancelled")
	}
	args := buildCommand(cfg, sel)
	if len(sel.SkillPaths) > 0 {
		fmt.Println("CLAUDE_CONFIG_DIR=<temp-shadow-dir> \\")
	}
	fmt.Println(strings.Join(args, " "))
	return nil
}

func launchClaude(args []string, configOverride string) {
	binary, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: claude not found in PATH\n")
		os.Exit(1)
	}

	env := os.Environ()
	if configOverride != "" {
		// Filter out any existing CLAUDE_CONFIG_DIR and set our override
		filtered := make([]string, 0, len(env)+1)
		for _, e := range env {
			if !strings.HasPrefix(e, "CLAUDE_CONFIG_DIR=") {
				filtered = append(filtered, e)
			}
		}
		filtered = append(filtered, "CLAUDE_CONFIG_DIR="+configOverride)
		env = filtered
	}

	// Replace the current process with claude
	if err := syscall.Exec(binary, args, env); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching claude: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`YOLO — Claude Code Launcher

Usage:
  yolo              Launch the TUI and configure a Claude Code session
  yolo init         Create a default config at ~/.yolo/config.yaml
  yolo config       Print the config file path
  yolo dry-run      Run the TUI but only print the command (don't launch)
  yolo help         Show this help

Configuration:
  Edit ~/.yolo/config.yaml to customize:
  - System prompts to choose from
  - Skill/plugin directories
  - MCP server configurations
  - Agent definitions
  - Presets for quick launch`)
}
