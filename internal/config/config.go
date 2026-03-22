package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type SystemPrompt struct {
	Name string `yaml:"name"`
	Text string `yaml:"text"`
	File string `yaml:"file"`
}

type SkillDir struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type MCPConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type AgentDef struct {
	Name string `yaml:"name"`
	JSON string `yaml:"json"`
}

type Preset struct {
	Name           string   `yaml:"name"`
	SystemPrompt   string   `yaml:"system_prompt"`
	AppendPrompt   string   `yaml:"append_prompt"`
	SkillDirs      []string `yaml:"skill_dirs"`
	MCPConfigs     []string `yaml:"mcp_configs"`
	Model          string   `yaml:"model"`
	Effort         string   `yaml:"effort"`
	PermissionMode string   `yaml:"permission_mode"`
	AllowedTools   []string `yaml:"allowed_tools"`
	AddDirs        []string `yaml:"add_dirs"`
	Agent          string   `yaml:"agent"`
	ExtraFlags     []string `yaml:"extra_flags"`
}

type Config struct {
	Defaults struct {
		PermissionMode string `yaml:"permission_mode"`
		Model          string `yaml:"model"`
		Effort         string `yaml:"effort"`
	} `yaml:"defaults"`
	SystemPrompts []SystemPrompt `yaml:"system_prompts"`
	SkillDirs     []SkillDir     `yaml:"skill_dirs"`
	MCPConfigs    []MCPConfig    `yaml:"mcp_configs"`
	Agents        []AgentDef     `yaml:"agents"`
	Presets       []Preset       `yaml:"presets"`
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".yolo")
}

func Path() string {
	return filepath.Join(Dir(), "config.yaml")
}

func Load() (*Config, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found at %s — run 'yolo init' to create one", path)
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}

func ResolvePromptText(sp SystemPrompt) (string, error) {
	if sp.Text != "" {
		return sp.Text, nil
	}
	if sp.File != "" {
		path := ExpandHome(sp.File)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file %s: %w", path, err)
		}
		return string(data), nil
	}
	return "", nil
}

func WriteDefault() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	defaultConfig := `# YOLO - Claude Code Launcher Configuration
# Place this file at ~/.yolo/config.yaml

# Default settings applied to every session unless overridden
defaults:
  permission_mode: bypassPermissions  # default, acceptEdits, bypassPermissions, plan, auto
  model: ""                           # leave empty for Claude's default, or set e.g. "opus", "sonnet"
  effort: ""                          # low, medium, high, max, or empty for default

# System prompts to choose from
# Each can use inline "text" or reference a "file" path
system_prompts:
  - name: "Default (no override)"
    text: ""
  - name: "Senior Engineer"
    text: "You are an expert senior software engineer. Write production-quality, well-tested code. Prioritize correctness, performance, and maintainability."
  - name: "Rapid Prototyper"
    text: "You are helping me rapidly prototype. Favor speed over perfection. Use simple solutions, skip tests unless asked, and get to a working demo fast."
  - name: "Code Reviewer"
    text: "You are a thorough code reviewer. Focus on finding bugs, security issues, and suggesting improvements. Be constructive but rigorous."
  # - name: "Custom from file"
  #   file: "~/.yolo/prompts/custom.md"

# Skill/plugin directories to optionally load
skill_dirs: []
  # - name: "My custom skills"
  #   path: "~/.claude/skills"

# MCP server configurations to optionally load
mcp_configs: []
  # - name: "Local MCP servers"
  #   path: "~/.claude/mcp.json"

# Agent definitions (JSON strings)
agents: []
  # - name: "Reviewer Agent"
  #   json: '{"reviewer": {"description": "Reviews code", "prompt": "You are a code reviewer"}}'

# Presets: pre-configured combinations for quick launch
presets:
  - name: "YOLO (bypass all)"
    permission_mode: bypassPermissions
  - name: "Safe mode (default perms)"
    permission_mode: default
  - name: "Planner"
    permission_mode: plan
    append_prompt: "Start by understanding the full scope of what needs to be done before writing any code."
`

	return os.WriteFile(Path(), []byte(defaultConfig), 0644)
}

func ExpandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
