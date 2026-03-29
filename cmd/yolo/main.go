package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/paolo/yolo/internal/command"
	"github.com/paolo/yolo/internal/config"
	"github.com/paolo/yolo/internal/export"
	"github.com/paolo/yolo/internal/skill"
	"github.com/paolo/yolo/internal/tui"
)

func main() {
	// Extract --export-dir before subcommand dispatch
	exportDir, args := export.ExtractExportDir(os.Args)

	if len(args) > 1 {
		switch args[1] {
		case "init":
			if err := runInit(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "config":
			fmt.Println(config.Path())
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

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sel, err := tui.Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cancelled.\n")
		os.Exit(0)
	}

	claudeArgs := command.Build(cfg, sel)

	var configOverride string
	if len(sel.SkillPaths) > 0 {
		override, err := skill.CreateConfigOverride(skill.DefaultClaudeDir(), sel.SkillPaths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create config override: %v\n", err)
		} else {
			configOverride = override
		}
	}

	tui.PrintCommand(claudeArgs)

	// Set up conversation export
	cwd, _ := os.Getwd()
	resolvedExportDir := export.ExportDir(exportDir)
	exportFilename := export.SessionFilename()
	exportPath := filepath.Join(resolvedExportDir, exportFilename)
	projectDir := export.ProjectDir(cwd)

	watcher := export.StartWatcher(projectDir, exportPath)

	exitCode := launchClaude(claudeArgs, configOverride)

	watcher.Stop()

	if exportPath != "" {
		if _, err := os.Stat(exportPath); err == nil {
			fmt.Fprintf(os.Stderr, "\nConversation exported to %s\n", exportPath)
		}
	}

	os.Exit(exitCode)
}

func runInit() error {
	path := config.Path()
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Config already exists at %s\n", path)
		fmt.Println("Delete it first if you want to regenerate.")
		return nil
	}
	if err := config.WriteDefault(); err != nil {
		return err
	}
	fmt.Printf("Config created at %s\n", path)
	fmt.Println("Edit it to add your system prompts, skill directories, and presets.")
	return nil
}

func runDryRun() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	sel, err := tui.Run(cfg)
	if err != nil {
		return fmt.Errorf("cancelled")
	}
	args := command.Build(cfg, sel)
	if len(sel.SkillPaths) > 0 {
		fmt.Println("CLAUDE_CONFIG_DIR=<temp-shadow-dir> \\")
	}
	fmt.Println(strings.Join(args, " "))
	return nil
}

// launchClaude runs claude as a subprocess with full stdio passthrough.
// Returns the exit code.
func launchClaude(args []string, configOverride string) int {
	binary, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: claude not found in PATH\n")
		return 1
	}

	cmd := exec.Command(binary, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := os.Environ()
	if configOverride != "" {
		filtered := make([]string, 0, len(env)+1)
		for _, e := range env {
			if !strings.HasPrefix(e, "CLAUDE_CONFIG_DIR=") {
				filtered = append(filtered, e)
			}
		}
		filtered = append(filtered, "CLAUDE_CONFIG_DIR="+configOverride)
		env = filtered
	}
	cmd.Env = env

	// Ignore SIGINT in the parent — Claude handles it directly from the terminal.
	// Forward SIGTERM so graceful shutdown propagates.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching claude: %v\n", err)
		return 1
	}

	go func() {
		for sig := range sigCh {
			if sig == syscall.SIGTERM {
				cmd.Process.Signal(syscall.SIGTERM)
			}
			// SIGINT: ignore in parent, Claude gets it from terminal directly
		}
	}()

	err = cmd.Wait()
	signal.Stop(sigCh)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func printHelp() {
	fmt.Println(`YOLO — Claude Code Launcher

Usage:
  yolo              Launch the TUI and configure a Claude Code session
  yolo init         Create a default config at ~/.yolo/config.yaml
  yolo config       Print the config file path
  yolo dry-run      Run the TUI but only print the command (don't launch)
  yolo help         Show this help

Options:
  --export-dir <path>  Custom directory for conversation exports
                       (default: ~/.yolo/exports/)

Conversation Export:
  Every session automatically exports conversation data to ~/.yolo/exports/.
  Each session gets its own file with a timestamped human-readable name
  (e.g. 2026-03-29_12-04-00_bold-keen-fox.jsonl).

Configuration:
  Edit ~/.yolo/config.yaml to customize:
  - System prompts to choose from
  - Skill/plugin directories
  - MCP server configurations
  - Agent definitions
  - Presets for quick launch`)
}
