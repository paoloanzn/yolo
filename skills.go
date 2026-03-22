package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a single skill file discovered inside a skill directory.
type Skill struct {
	Name    string // display name derived from filename
	Path    string // absolute path to the skill file
	DirName string // name of the parent SkillDir (for grouping in the TUI)
}

// discoverSkills scans all configured skill directories and returns individual skill entries.
func discoverSkills(cfg *Config) ([]Skill, error) {
	var skills []Skill
	for _, sd := range cfg.SkillDirs {
		found, err := discoverSkillsInDir(sd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read skill dir %s: %v\n", sd.Path, err)
			continue
		}
		skills = append(skills, found...)
	}
	return skills, nil
}

// discoverSkillsInDir scans a single skill directory and returns individual skills.
func discoverSkillsInDir(sd SkillDir) ([]Skill, error) {
	dirPath := expandHome(sd.Path)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		absPath := filepath.Join(dirPath, entry.Name())
		skills = append(skills, Skill{
			Name:    skillDisplayName(entry.Name()),
			Path:    absPath,
			DirName: sd.Name,
		})
	}
	return skills, nil
}

// skillDisplayName turns a filename into a readable name.
// "my-cool-skill.md" -> "my-cool-skill"
func skillDisplayName(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}

// createConfigOverride creates a shadow of the given Claude config directory
// that symlinks everything except the skills/ directory. Only the selected
// skills are symlinked into the shadow skills/ dir. Returns the temp dir path
// to be used as CLAUDE_CONFIG_DIR.
func createConfigOverride(claudeDir string, selectedPaths []string) (string, error) {
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

// defaultClaudeDir returns the path to ~/.claude.
func defaultClaudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
