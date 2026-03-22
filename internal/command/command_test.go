package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paolo/yolo/internal/config"
)

func containsSequence(args []string, key, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}

func containsArg(args []string, arg string) bool {
	for _, a := range args {
		if a == arg {
			return true
		}
	}
	return false
}

func TestBuild_MinimalCustom(t *testing.T) {
	cfg := &config.Config{}
	sel := &Selections{
		PermissionMode: "default",
	}

	args := Build(cfg, sel)

	if args[0] != "claude" {
		t.Errorf("first arg should be 'claude', got %q", args[0])
	}
	if !containsSequence(args, "--permission-mode", "default") {
		t.Error("expected --permission-mode default")
	}
}

func TestBuild_AllOptions(t *testing.T) {
	cfg := &config.Config{
		SystemPrompts: []config.SystemPrompt{
			{Name: "Engineer", Text: "You are an engineer."},
		},
		Agents: []config.AgentDef{
			{Name: "MyAgent", JSON: `{"agent": {}}`},
		},
	}
	sel := &Selections{
		PermissionMode: "bypassPermissions",
		SystemPrompt:   "Engineer",
		AppendPrompt:   "Be concise.",
		Model:          "opus",
		Effort:         "high",
		MCPConfigs:     []string{"/path/to/mcp.json"},
		Agent:          "MyAgent",
		AddDirs:        "/tmp /var",
		ExtraFlags:     "--verbose --debug",
	}

	args := Build(cfg, sel)

	checks := []struct {
		key, value string
	}{
		{"--permission-mode", "bypassPermissions"},
		{"--system-prompt", "You are an engineer."},
		{"--append-system-prompt", "Be concise."},
		{"--model", "opus"},
		{"--effort", "high"},
		{"--mcp-config", "/path/to/mcp.json"},
		{"--agents", `{"agent": {}}`},
	}

	for _, c := range checks {
		if !containsSequence(args, c.key, c.value) {
			t.Errorf("expected %s %s in args: %v", c.key, c.value, args)
		}
	}

	if !containsSequence(args, "--add-dir", "/tmp") {
		t.Error("expected --add-dir /tmp")
	}
	if !containsSequence(args, "--add-dir", "/var") {
		t.Error("expected --add-dir /var")
	}
	if !containsArg(args, "--verbose") {
		t.Error("expected --verbose in extra flags")
	}
	if !containsArg(args, "--debug") {
		t.Error("expected --debug in extra flags")
	}
}

func TestBuild_SkillsNotInArgs(t *testing.T) {
	cfg := &config.Config{}
	sel := &Selections{
		SkillPaths: []string{"/some/skill.md"},
	}

	args := Build(cfg, sel)

	for _, a := range args {
		if a == "--plugin-dir" {
			t.Error("--plugin-dir should not appear in args; skills are handled via CLAUDE_CONFIG_DIR")
		}
	}
}

func TestBuild_EmptyOptionalFields(t *testing.T) {
	cfg := &config.Config{}
	sel := &Selections{}

	args := Build(cfg, sel)

	if len(args) != 1 || args[0] != "claude" {
		t.Errorf("expected just ['claude'] for empty selections, got %v", args)
	}
}

func TestBuild_SystemPromptFromFile(t *testing.T) {
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "prompt.md")
	os.WriteFile(promptFile, []byte("file-based prompt"), 0644)

	cfg := &config.Config{
		SystemPrompts: []config.SystemPrompt{
			{Name: "FromFile", File: promptFile},
		},
	}
	sel := &Selections{SystemPrompt: "FromFile"}

	args := Build(cfg, sel)

	if !containsSequence(args, "--system-prompt", "file-based prompt") {
		t.Errorf("expected file-based prompt content in args: %v", args)
	}
}

func TestBuild_SystemPromptNotFound(t *testing.T) {
	cfg := &config.Config{
		SystemPrompts: []config.SystemPrompt{
			{Name: "Exists", Text: "hello"},
		},
	}
	sel := &Selections{SystemPrompt: "DoesNotExist"}

	args := Build(cfg, sel)

	if containsArg(args, "--system-prompt") {
		t.Error("should not include --system-prompt for unknown prompt name")
	}
}

func TestBuildPreset_Basic(t *testing.T) {
	cfg := &config.Config{
		SystemPrompts: []config.SystemPrompt{
			{Name: "Engineer", Text: "You are an engineer."},
		},
	}
	sel := &Selections{}
	preset := &config.Preset{
		Name:           "Fast",
		PermissionMode: "bypassPermissions",
		SystemPrompt:   "Engineer",
		AppendPrompt:   "Be fast.",
		Model:          "sonnet",
		Effort:         "low",
	}

	args := buildPreset(cfg, preset, sel)

	checks := []struct {
		key, value string
	}{
		{"--permission-mode", "bypassPermissions"},
		{"--system-prompt", "You are an engineer."},
		{"--append-system-prompt", "Be fast."},
		{"--model", "sonnet"},
		{"--effort", "low"},
	}
	for _, c := range checks {
		if !containsSequence(args, c.key, c.value) {
			t.Errorf("expected %s %s in args: %v", c.key, c.value, args)
		}
	}
}

func TestBuildPreset_ResolvesSkillDirs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "skill-one.md"), []byte("one"), 0644)
	os.MkdirAll(filepath.Join(dir, "skill-two"), 0755)

	cfg := &config.Config{
		SkillDirs: []config.SkillDir{
			{Name: "TestSkills", Path: dir},
		},
	}
	sel := &Selections{}
	preset := &config.Preset{
		SkillDirs: []string{"TestSkills"},
	}

	buildPreset(cfg, preset, sel)

	if len(sel.SkillPaths) != 2 {
		t.Fatalf("expected 2 skill paths resolved from preset, got %d: %v", len(sel.SkillPaths), sel.SkillPaths)
	}

	// Skills should NOT appear as --plugin-dir
	args := buildPreset(cfg, preset, &Selections{})
	for _, a := range args {
		if a == "--plugin-dir" {
			t.Error("--plugin-dir should not appear; skills handled via CLAUDE_CONFIG_DIR")
		}
	}
}

func TestBuildPreset_MCPAndAgent(t *testing.T) {
	cfg := &config.Config{
		MCPConfigs: []config.MCPConfig{
			{Name: "MCP1", Path: "/etc/mcp1.json"},
			{Name: "MCP2", Path: "/etc/mcp2.json"},
		},
		Agents: []config.AgentDef{
			{Name: "Bot", JSON: `{"bot": {}}`},
		},
	}
	sel := &Selections{}
	preset := &config.Preset{
		MCPConfigs: []string{"MCP1", "MCP2"},
		Agent:      "Bot",
	}

	args := buildPreset(cfg, preset, sel)

	if !containsSequence(args, "--mcp-config", "/etc/mcp1.json") {
		t.Error("expected --mcp-config /etc/mcp1.json")
	}
	if !containsSequence(args, "--mcp-config", "/etc/mcp2.json") {
		t.Error("expected --mcp-config /etc/mcp2.json")
	}
	if !containsSequence(args, "--agents", `{"bot": {}}`) {
		t.Error("expected --agents for Bot")
	}
}

func TestBuildPreset_AllowedToolsAndAddDirs(t *testing.T) {
	cfg := &config.Config{}
	sel := &Selections{}
	preset := &config.Preset{
		AllowedTools: []string{"Bash", "Edit"},
		AddDirs:      []string{"/tmp", "/var"},
		ExtraFlags:   []string{"--verbose"},
	}

	args := buildPreset(cfg, preset, sel)

	if !containsSequence(args, "--allowed-tools", "Bash") {
		t.Error("expected --allowed-tools Bash")
	}
	if !containsSequence(args, "--allowed-tools", "Edit") {
		t.Error("expected --allowed-tools Edit")
	}
	if !containsSequence(args, "--add-dir", "/tmp") {
		t.Error("expected --add-dir /tmp")
	}
	if !containsArg(args, "--verbose") {
		t.Error("expected --verbose in extra flags")
	}
}

func TestBuild_PresetDelegation(t *testing.T) {
	cfg := &config.Config{
		Presets: []config.Preset{
			{
				Name:           "YOLO",
				PermissionMode: "bypassPermissions",
			},
		},
	}
	sel := &Selections{
		UsePreset:  true,
		PresetName: "YOLO",
	}

	args := Build(cfg, sel)

	if !containsSequence(args, "--permission-mode", "bypassPermissions") {
		t.Errorf("expected preset to set --permission-mode bypassPermissions: %v", args)
	}
}

func TestBuild_UnknownPreset(t *testing.T) {
	cfg := &config.Config{
		Presets: []config.Preset{
			{Name: "Exists"},
		},
	}
	sel := &Selections{
		UsePreset:  true,
		PresetName: "DoesNotExist",
	}

	args := Build(cfg, sel)

	if len(args) != 1 || args[0] != "claude" {
		t.Errorf("expected fallback to ['claude'] for unknown preset, got %v", args)
	}
}
