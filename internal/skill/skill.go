package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paolo/yolo/internal/config"
)

// Skill represents a single skill entry discovered inside a skill directory.
type Skill struct {
	Name    string // display name derived from filename
	Path    string // absolute path to the skill file or folder
	DirName string // name of the parent SkillDir (for grouping in the TUI)
}

// DiscoverAll scans all configured skill directories and returns individual skill entries.
func DiscoverAll(cfg *config.Config) ([]Skill, error) {
	var skills []Skill
	for _, sd := range cfg.SkillDirs {
		found, err := DiscoverInDir(sd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read skill dir %s: %v\n", sd.Path, err)
			continue
		}
		skills = append(skills, found...)
	}
	return skills, nil
}

// DiscoverInDir scans a single skill directory and returns individual skills.
func DiscoverInDir(sd config.SkillDir) ([]Skill, error) {
	dirPath := config.ExpandHome(sd.Path)
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
			Name:    displayName(entry.Name()),
			Path:    absPath,
			DirName: sd.Name,
		})
	}
	return skills, nil
}

// displayName turns a filename into a readable name by stripping the extension.
func displayName(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}
