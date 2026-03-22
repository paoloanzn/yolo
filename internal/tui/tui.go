package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/paolo/yolo/internal/command"
	"github.com/paolo/yolo/internal/config"
	"github.com/paolo/yolo/internal/skill"
)

// Run presents the interactive TUI and returns user selections.
func Run(cfg *config.Config) (*command.Selections, error) {
	sel := &command.Selections{}

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
	promptOptions := make([]huh.Option[string], 0, len(cfg.SystemPrompts))
	for _, sp := range cfg.SystemPrompts {
		promptOptions = append(promptOptions, huh.NewOption(sp.Name, sp.Name))
	}
	if len(promptOptions) == 0 {
		promptOptions = append(promptOptions, huh.NewOption("Default (no override)", ""))
	}

	permDefault := cfg.Defaults.PermissionMode
	if permDefault == "" {
		permDefault = "bypassPermissions"
	}

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
	sel.Model = cfg.Defaults.Model
	sel.Effort = cfg.Defaults.Effort

	// Group 2: Individual skills from configured directories
	if len(cfg.SkillDirs) > 0 {
		skills, _ := skill.DiscoverAll(cfg)
		if len(skills) > 0 {
			skillOptions := make([]huh.Option[string], 0, len(skills))
			for _, sk := range skills {
				label := fmt.Sprintf("[%s] %s", sk.DirName, sk.Name)
				skillOptions = append(skillOptions, huh.NewOption(label, sk.Path))
			}
			groups = append(groups, huh.NewGroup(
				huh.NewNote().Title("Skills & Plugins"),
				huh.NewMultiSelect[string]().
					Title("Skills").
					Description("Select individual skills to load").
					Options(skillOptions...).
					Value(&sel.SkillPaths),
			))
		}
	}

	// Group 3: MCP configs
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

	// Group 4: Agents
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

// PrintCommand displays the constructed command in styled output.
func PrintCommand(args []string) {
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
