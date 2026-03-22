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

// discoverSkills scans all configured skill directories and returns individual skill files.
func discoverSkills(cfg *Config) ([]Skill, error) {
	var skills []Skill

	for _, sd := range cfg.SkillDirs {
		dirPath := expandHome(sd.Path)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read skill dir %s: %v\n", dirPath, err)
			continue
		}

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
	}

	return skills, nil
}

// skillDisplayName turns a filename into a readable name.
// "my-cool-skill.md" -> "my-cool-skill"
func skillDisplayName(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}

// createSkillTempDir creates a temporary directory containing symlinks to the selected skill files.
// Returns the temp dir path. The caller (claude process) will inherit it and it'll be
// cleaned up when the OS cleans temp files.
func createSkillTempDir(selectedPaths []string) (string, error) {
	if len(selectedPaths) == 0 {
		return "", nil
	}

	tmpDir, err := os.MkdirTemp("", "yolo-skills-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp skill dir: %w", err)
	}

	for _, srcPath := range selectedPaths {
		filename := filepath.Base(srcPath)
		dstPath := filepath.Join(tmpDir, filename)
		if err := os.Symlink(srcPath, dstPath); err != nil {
			return "", fmt.Errorf("failed to symlink %s: %w", srcPath, err)
		}
	}

	return tmpDir, nil
}

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
