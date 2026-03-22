package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadConfig_ValidYAML(t *testing.T) {
	configYAML := `
defaults:
  permission_mode: bypassPermissions
  model: opus
  effort: high

system_prompts:
  - name: "Test Prompt"
    text: "You are a test assistant."
  - name: "File Prompt"
    file: "~/.yolo/prompts/test.md"

skill_dirs:
  - name: "My Skills"
    path: "~/.claude/skills"
  - name: "Work Skills"
    path: "/opt/skills"

mcp_configs:
  - name: "Local MCP"
    path: "/etc/mcp.json"

agents:
  - name: "Reviewer"
    json: '{"reviewer": {"description": "Reviews code"}}'

presets:
  - name: "Fast"
    permission_mode: bypassPermissions
    model: sonnet
    effort: low
    skill_dirs:
      - "My Skills"
    mcp_configs:
      - "Local MCP"
    agent: "Reviewer"
    allowed_tools:
      - "Bash"
    add_dirs:
      - "/tmp"
    extra_flags:
      - "--verbose"
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(configYAML), &cfg); err != nil {
		t.Fatalf("failed to parse valid config: %v", err)
	}

	// Defaults
	if cfg.Defaults.PermissionMode != "bypassPermissions" {
		t.Errorf("expected permission_mode bypassPermissions, got %q", cfg.Defaults.PermissionMode)
	}
	if cfg.Defaults.Model != "opus" {
		t.Errorf("expected model opus, got %q", cfg.Defaults.Model)
	}
	if cfg.Defaults.Effort != "high" {
		t.Errorf("expected effort high, got %q", cfg.Defaults.Effort)
	}

	// System prompts
	if len(cfg.SystemPrompts) != 2 {
		t.Fatalf("expected 2 system prompts, got %d", len(cfg.SystemPrompts))
	}
	if cfg.SystemPrompts[0].Name != "Test Prompt" || cfg.SystemPrompts[0].Text != "You are a test assistant." {
		t.Errorf("unexpected system prompt: %+v", cfg.SystemPrompts[0])
	}
	if cfg.SystemPrompts[1].File != "~/.yolo/prompts/test.md" {
		t.Errorf("expected file prompt path, got %q", cfg.SystemPrompts[1].File)
	}

	// Skill dirs
	if len(cfg.SkillDirs) != 2 {
		t.Fatalf("expected 2 skill dirs, got %d", len(cfg.SkillDirs))
	}
	if cfg.SkillDirs[0].Name != "My Skills" || cfg.SkillDirs[0].Path != "~/.claude/skills" {
		t.Errorf("unexpected skill dir: %+v", cfg.SkillDirs[0])
	}

	// MCP configs
	if len(cfg.MCPConfigs) != 1 || cfg.MCPConfigs[0].Name != "Local MCP" {
		t.Errorf("unexpected mcp configs: %+v", cfg.MCPConfigs)
	}

	// Agents
	if len(cfg.Agents) != 1 || cfg.Agents[0].Name != "Reviewer" {
		t.Errorf("unexpected agents: %+v", cfg.Agents)
	}

	// Presets
	if len(cfg.Presets) != 1 {
		t.Fatalf("expected 1 preset, got %d", len(cfg.Presets))
	}
	p := cfg.Presets[0]
	if p.Name != "Fast" || p.Model != "sonnet" || p.Effort != "low" {
		t.Errorf("unexpected preset core: %+v", p)
	}
	if len(p.SkillDirs) != 1 || p.SkillDirs[0] != "My Skills" {
		t.Errorf("unexpected preset skill_dirs: %v", p.SkillDirs)
	}
	if len(p.AllowedTools) != 1 || p.AllowedTools[0] != "Bash" {
		t.Errorf("unexpected preset allowed_tools: %v", p.AllowedTools)
	}
	if len(p.ExtraFlags) != 1 || p.ExtraFlags[0] != "--verbose" {
		t.Errorf("unexpected preset extra_flags: %v", p.ExtraFlags)
	}
}

func TestLoadConfig_EmptyDefaults(t *testing.T) {
	configYAML := `
system_prompts: []
skill_dirs: []
presets: []
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(configYAML), &cfg); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if cfg.Defaults.PermissionMode != "" || cfg.Defaults.Model != "" || cfg.Defaults.Effort != "" {
		t.Errorf("expected empty defaults, got %+v", cfg.Defaults)
	}
}

func TestResolvePromptText_InlineText(t *testing.T) {
	sp := SystemPrompt{Name: "test", Text: "hello world"}
	text, err := resolvePromptText(sp)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello world" {
		t.Errorf("expected 'hello world', got %q", text)
	}
}

func TestResolvePromptText_FromFile(t *testing.T) {
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("file prompt content"), 0644); err != nil {
		t.Fatal(err)
	}

	sp := SystemPrompt{Name: "test", File: promptFile}
	text, err := resolvePromptText(sp)
	if err != nil {
		t.Fatal(err)
	}
	if text != "file prompt content" {
		t.Errorf("expected 'file prompt content', got %q", text)
	}
}

func TestResolvePromptText_MissingFile(t *testing.T) {
	sp := SystemPrompt{Name: "test", File: "/nonexistent/prompt.md"}
	_, err := resolvePromptText(sp)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestResolvePromptText_Empty(t *testing.T) {
	sp := SystemPrompt{Name: "test"}
	text, err := resolvePromptText(sp)
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestResolvePromptText_TextTakesPrecedence(t *testing.T) {
	sp := SystemPrompt{Name: "test", Text: "inline", File: "/some/file"}
	text, err := resolvePromptText(sp)
	if err != nil {
		t.Fatal(err)
	}
	if text != "inline" {
		t.Errorf("expected inline text to take precedence, got %q", text)
	}
}
