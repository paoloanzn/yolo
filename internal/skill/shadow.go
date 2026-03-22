package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultClaudeDir returns the path to ~/.claude.
func DefaultClaudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// CreateConfigOverride creates a shadow of the given Claude config directory
// that symlinks everything except the skills/ directory. Only the selected
// skills are symlinked into the shadow skills/ dir. Returns the temp dir path
// to be used as CLAUDE_CONFIG_DIR.
func CreateConfigOverride(claudeDir string, selectedPaths []string) (string, error) {
	if len(selectedPaths) == 0 {
		return "", nil
	}

	tmpDir, err := os.MkdirTemp("", "yolo-claude-config-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp config dir: %w", err)
	}

	// Symlink everything from the claude config dir except skills/
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", claudeDir, err)
	}

	for _, entry := range entries {
		if entry.Name() == "skills" {
			continue
		}
		src := filepath.Join(claudeDir, entry.Name())
		dst := filepath.Join(tmpDir, entry.Name())
		if err := os.Symlink(src, dst); err != nil {
			return "", fmt.Errorf("failed to symlink %s: %w", src, err)
		}
	}

	// Create skills/ with only the selected skills
	skillsDir := filepath.Join(tmpDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skills dir: %w", err)
	}

	for _, srcPath := range selectedPaths {
		name := filepath.Base(srcPath)
		dst := filepath.Join(skillsDir, name)
		if err := os.Symlink(srcPath, dst); err != nil {
			return "", fmt.Errorf("failed to symlink skill %s: %w", srcPath, err)
		}
	}

	return tmpDir, nil
}
