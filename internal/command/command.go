package command

import (
	"strings"

	"github.com/paolo/yolo/internal/config"
	"github.com/paolo/yolo/internal/skill"
)

// Selections holds the user's choices from the TUI or preset resolution.
type Selections struct {
	UsePreset      bool
	PresetName     string
	SystemPrompt   string
	AppendPrompt   string
	PermissionMode string
	Model          string
	Effort         string
	SkillPaths     []string // individual skill file/folder paths
	MCPConfigs     []string
	Agent          string
	AddDirs        string
	ExtraFlags     string
}

// Build constructs the claude CLI args from config and user selections.
func Build(cfg *config.Config, sel *Selections) []string {
	args := []string{"claude"}

	if sel.UsePreset {
		for _, p := range cfg.Presets {
			if p.Name == sel.PresetName {
				return buildPreset(cfg, &p, sel)
			}
		}
		return args
	}

	if sel.PermissionMode != "" {
		args = append(args, "--permission-mode", sel.PermissionMode)
	}

	if sel.SystemPrompt != "" {
		for _, sp := range cfg.SystemPrompts {
			if sp.Name == sel.SystemPrompt {
				text, err := config.ResolvePromptText(sp)
				if err == nil && text != "" {
					args = append(args, "--system-prompt", text)
				}
				break
			}
		}
	}

	if sel.AppendPrompt != "" {
		args = append(args, "--append-system-prompt", sel.AppendPrompt)
	}

	if sel.Model != "" {
		args = append(args, "--model", sel.Model)
	}

	if sel.Effort != "" {
		args = append(args, "--effort", sel.Effort)
	}

	// Skills are handled via CLAUDE_CONFIG_DIR override, not CLI flags

	for _, mc := range sel.MCPConfigs {
		args = append(args, "--mcp-config", mc)
	}

	if sel.Agent != "" {
		for _, a := range cfg.Agents {
			if a.Name == sel.Agent {
				args = append(args, "--agents", a.JSON)
				break
			}
		}
	}

	if sel.AddDirs != "" {
		for _, d := range strings.Fields(sel.AddDirs) {
			args = append(args, "--add-dir", d)
		}
	}

	if sel.ExtraFlags != "" {
		args = append(args, strings.Fields(sel.ExtraFlags)...)
	}

	return args
}

func buildPreset(cfg *config.Config, p *config.Preset, sel *Selections) []string {
	args := []string{"claude"}

	if p.PermissionMode != "" {
		args = append(args, "--permission-mode", p.PermissionMode)
	}

	if p.SystemPrompt != "" {
		for _, sp := range cfg.SystemPrompts {
			if sp.Name == p.SystemPrompt {
				text, err := config.ResolvePromptText(sp)
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

	// Resolve preset skill dirs into individual skill paths for config override
	for _, sdName := range p.SkillDirs {
		for _, sd := range cfg.SkillDirs {
			if sd.Name == sdName {
				skills, _ := skill.DiscoverInDir(sd)
				for _, sk := range skills {
					sel.SkillPaths = append(sel.SkillPaths, sk.Path)
				}
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
