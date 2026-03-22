package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type Selections struct {
	UsePreset      bool
	PresetName     string
	SystemPrompt   string
	AppendPrompt   string
	PermissionMode string
	Model          string
	Effort         string
	SkillDirs      []string
	MCPConfigs     []string
	Agent          string
	AddDirs        string
	ExtraFlags     string
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6600")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)
)

func runTUI(cfg *Config) (*Selections, error) {
	sel := &Selections{}

	// Phase 1: Ask if user wants a preset or custom config
	hasPresets := len(cfg.Presets) > 0
	launchMode := "custom"

	if hasPresets {
		presetOptions := []huh.Option[string]{
			huh.NewOption("Custom configuration", "custom"),
		}
		for _, p := range cfg.Presets {
			presetOptions = append(presetOptions, huh.NewOption(p.Name, p.Name))
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("🚀 YOLO — Claude Code Launcher").
					Description("Configure and launch Claude Code with the settings you need."),
				huh.NewSelect[string]().
					Title("Launch mode").
					Options(presetOptions...).
					Value(&launchMode),
			),
		)

		if err := form.Run(); err != nil {
			return nil, err
		}
	}

	if launchMode != "custom" {
		sel.UsePreset = true
		sel.PresetName = launchMode
		return sel, nil
	}

	// Phase 2: Custom configuration
	// System prompt selection
	promptOptions := make([]huh.Option[string], 0, len(cfg.SystemPrompts))
	for _, sp := range cfg.SystemPrompts {
		promptOptions = append(promptOptions, huh.NewOption(sp.Name, sp.Name))
	}
	if len(promptOptions) == 0 {
		promptOptions = append(promptOptions, huh.NewOption("Default (no override)", ""))
	}

	// Permission mode
	permDefault := cfg.Defaults.PermissionMode
	if permDefault == "" {
		permDefault = "bypassPermissions"
	}

	// Model
	modelDefault := cfg.Defaults.Model

	// Effort
	effortDefault := cfg.Defaults.Effort

	groups := []*huh.Group{}

	// Group 1: Core settings
	coreGroup := huh.NewGroup(
		huh.NewNote().
			Title("Core Settings"),
		huh.NewSelect[string]().
			Title("System Prompt").
			Description("Choose a system prompt for this session").
			Options(promptOptions...).
			Value(&sel.SystemPrompt),
		huh.NewSelect[string]().
			Title("Permission Mode").
			Options(
				huh.NewOption("Bypass All (yolo mode)", "bypassPermissions"),
				huh.NewOption("Default", "default"),
				huh.NewOption("Accept Edits", "acceptEdits"),
				huh.NewOption("Plan Mode", "plan"),
				huh.NewOption("Auto", "auto"),
				huh.NewOption("Don't Ask", "dontAsk"),
			).
			Value(&sel.PermissionMode),
		huh.NewSelect[string]().
			Title("Model").
			Description("Leave default to use Claude's default model").
			Options(
				huh.NewOption("Default", ""),
				huh.NewOption("Opus", "opus"),
				huh.NewOption("Sonnet", "sonnet"),
				huh.NewOption("Haiku", "haiku"),
			).
			Value(&sel.Model),
		huh.NewSelect[string]().
			Title("Effort Level").
			Options(
				huh.NewOption("Default", ""),
				huh.NewOption("Low", "low"),
				huh.NewOption("Medium", "medium"),
				huh.NewOption("High", "high"),
				huh.NewOption("Max", "max"),
			).
			Value(&sel.Effort),
	)
	groups = append(groups, coreGroup)

	// Set defaults
	sel.PermissionMode = permDefault
	sel.Model = modelDefault
	sel.Effort = effortDefault

	// Group 2: Skill dirs (if any configured)
	if len(cfg.SkillDirs) > 0 {
		skillOptions := make([]huh.Option[string], 0, len(cfg.SkillDirs))
		for _, sd := range cfg.SkillDirs {
			skillOptions = append(skillOptions, huh.NewOption(sd.Name, sd.Path))
		}
		groups = append(groups, huh.NewGroup(
			huh.NewNote().Title("Skills & Plugins"),
			huh.NewMultiSelect[string]().
				Title("Skill Directories").
				Description("Select skill directories to load").
				Options(skillOptions...).
				Value(&sel.SkillDirs),
		))
	}

	// Group 3: MCP configs (if any configured)
	if len(cfg.MCPConfigs) > 0 {
		mcpOptions := make([]huh.Option[string], 0, len(cfg.MCPConfigs))
		for _, mc := range cfg.MCPConfigs {
			mcpOptions = append(mcpOptions, huh.NewOption(mc.Name, mc.Path))
		}
		groups = append(groups, huh.NewGroup(
			huh.NewNote().Title("MCP Servers"),
			huh.NewMultiSelect[string]().
				Title("MCP Configurations").
				Description("Select MCP server configs to load").
				Options(mcpOptions...).
				Value(&sel.MCPConfigs),
		))
	}

	// Group 4: Agents (if any configured)
	if len(cfg.Agents) > 0 {
		agentOptions := []huh.Option[string]{
			huh.NewOption("None", ""),
		}
		for _, a := range cfg.Agents {
			agentOptions = append(agentOptions, huh.NewOption(a.Name, a.Name))
		}
		groups = append(groups, huh.NewGroup(
			huh.NewNote().Title("Agents"),
			huh.NewSelect[string]().
				Title("Agent").
				Description("Select an agent definition to use").
				Options(agentOptions...).
				Value(&sel.Agent),
		))
	}

	// Group 5: Extra options
	groups = append(groups, huh.NewGroup(
		huh.NewNote().Title("Additional Options"),
		huh.NewInput().
			Title("Append to system prompt").
			Description("Additional text to append to the system prompt (optional)").
			Value(&sel.AppendPrompt),
		huh.NewInput().
			Title("Additional directories").
			Description("Space-separated paths to allow tool access to (optional)").
			Value(&sel.AddDirs),
		huh.NewInput().
			Title("Extra flags").
			Description("Any additional CLI flags to pass to claude (optional)").
			Value(&sel.ExtraFlags),
	))

	form := huh.NewForm(groups...)
	if err := form.Run(); err != nil {
		return nil, err
	}

	return sel, nil
}

func buildCommand(cfg *Config, sel *Selections) []string {
	args := []string{"claude"}

	if sel.UsePreset {
		for _, p := range cfg.Presets {
			if p.Name == sel.PresetName {
				return buildPresetCommand(cfg, &p)
			}
		}
		return args
	}

	// Permission mode
	if sel.PermissionMode != "" {
		args = append(args, "--permission-mode", sel.PermissionMode)
	}

	// System prompt
	if sel.SystemPrompt != "" {
		for _, sp := range cfg.SystemPrompts {
			if sp.Name == sel.SystemPrompt {
				text, err := resolvePromptText(sp)
				if err == nil && text != "" {
					args = append(args, "--system-prompt", text)
				}
				break
			}
		}
	}

	// Append prompt
	if sel.AppendPrompt != "" {
		args = append(args, "--append-system-prompt", sel.AppendPrompt)
	}

	// Model
	if sel.Model != "" {
		args = append(args, "--model", sel.Model)
	}

	// Effort
	if sel.Effort != "" {
		args = append(args, "--effort", sel.Effort)
	}

	// Skill dirs
	for _, dir := range sel.SkillDirs {
		args = append(args, "--plugin-dir", dir)
	}

	// MCP configs
	for _, mc := range sel.MCPConfigs {
		args = append(args, "--mcp-config", mc)
	}

	// Agent
	if sel.Agent != "" {
		for _, a := range cfg.Agents {
			if a.Name == sel.Agent {
				args = append(args, "--agents", a.JSON)
				break
			}
		}
	}

	// Add dirs
	if sel.AddDirs != "" {
		dirs := strings.Fields(sel.AddDirs)
		for _, d := range dirs {
			args = append(args, "--add-dir", d)
		}
	}

	// Extra flags
	if sel.ExtraFlags != "" {
		flags := strings.Fields(sel.ExtraFlags)
		args = append(args, flags...)
	}

	return args
}

func buildPresetCommand(cfg *Config, p *Preset) []string {
	args := []string{"claude"}

	if p.PermissionMode != "" {
		args = append(args, "--permission-mode", p.PermissionMode)
	}

	if p.SystemPrompt != "" {
		for _, sp := range cfg.SystemPrompts {
			if sp.Name == p.SystemPrompt {
				text, err := resolvePromptText(sp)
				if err == nil && text != "" {
					args = append(args, "--system-prompt", text)
				}
				break
			}
		}
	}

	if p.AppendPrompt != "" {
		args = append(args, "--append-system-prompt", p.AppendPrompt)
	}

	if p.Model != "" {
		args = append(args, "--model", p.Model)
	}

	if p.Effort != "" {
		args = append(args, "--effort", p.Effort)
	}

	for _, sdName := range p.SkillDirs {
		for _, sd := range cfg.SkillDirs {
			if sd.Name == sdName {
				args = append(args, "--plugin-dir", sd.Path)
				break
			}
		}
	}

	for _, mcName := range p.MCPConfigs {
		for _, mc := range cfg.MCPConfigs {
			if mc.Name == mcName {
				args = append(args, "--mcp-config", mc.Path)
				break
			}
		}
	}

	if p.Agent != "" {
		for _, a := range cfg.Agents {
			if a.Name == p.Agent {
				args = append(args, "--agents", a.JSON)
				break
			}
		}
	}

	for _, t := range p.AllowedTools {
		args = append(args, "--allowed-tools", t)
	}

	for _, d := range p.AddDirs {
		args = append(args, "--add-dir", d)
	}

	args = append(args, p.ExtraFlags...)

	return args
}

func printCommand(args []string) {
	fmt.Println()
	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	fmt.Println(labelStyle.Render("Launching:"))
	fmt.Println(cmdStyle.Render(strings.Join(args, " ")))
	fmt.Println()
}
