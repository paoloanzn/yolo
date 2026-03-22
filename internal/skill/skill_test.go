package skill

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/paolo/yolo/internal/config"
)

func TestDisplayName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-cool-skill.md", "my-cool-skill"},
		{"deploy.yaml", "deploy"},
		{"no-extension", "no-extension"},
		{"dots.in.name.md", "dots.in.name"},
		{".hidden", ""},
	}
	for _, tt := range tests {
		got := displayName(tt.input)
		if got != tt.want {
			t.Errorf("displayName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// setupSkillDir creates a temp skill directory with a mix of entry types.
func setupSkillDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "skill-file.md"), []byte("# Skill"), 0644)

	os.MkdirAll(filepath.Join(dir, "skill-folder", "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "skill-folder", "index.md"), []byte("# Folder Skill"), 0644)

	target := filepath.Join(dir, "skill-file.md")
	os.Symlink(target, filepath.Join(dir, "skill-symlink.md"))

	os.WriteFile(filepath.Join(dir, ".hidden-file"), []byte("hidden"), 0644)
	os.MkdirAll(filepath.Join(dir, ".hidden-dir"), 0755)

	return dir
}

func TestDiscoverInDir_FindsAllTypes(t *testing.T) {
	dir := setupSkillDir(t)
	sd := config.SkillDir{Name: "Test", Path: dir}

	skills, err := DiscoverInDir(sd)
	if err != nil {
		t.Fatal(err)
	}

	if len(skills) != 3 {
		names := make([]string, len(skills))
		for i, s := range skills {
			names[i] = s.Name
		}
		t.Fatalf("expected 3 skills, got %d: %v", len(skills), names)
	}

	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	sort.Strings(names)

	expected := []string{"skill-file", "skill-folder", "skill-symlink"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("skill[%d] name = %q, want %q", i, names[i], want)
		}
	}
}

func TestDiscoverInDir_SkipsHiddenEntries(t *testing.T) {
	dir := setupSkillDir(t)
	sd := config.SkillDir{Name: "Test", Path: dir}

	skills, err := DiscoverInDir(sd)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range skills {
		if s.Name == "" || s.Name[0] == '.' {
			t.Errorf("found hidden entry that should have been skipped: %q", s.Name)
		}
	}
}

func TestDiscoverInDir_SetsCorrectPaths(t *testing.T) {
	dir := setupSkillDir(t)
	sd := config.SkillDir{Name: "MyDir", Path: dir}

	skills, err := DiscoverInDir(sd)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range skills {
		if s.DirName != "MyDir" {
			t.Errorf("DirName = %q, want %q", s.DirName, "MyDir")
		}
		if !filepath.IsAbs(s.Path) {
			t.Errorf("Path should be absolute, got %q", s.Path)
		}
		if filepath.Dir(s.Path) != dir {
			t.Errorf("Path parent should be %q, got %q", dir, filepath.Dir(s.Path))
		}
	}
}

func TestDiscoverInDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	sd := config.SkillDir{Name: "Empty", Path: dir}

	skills, err := DiscoverInDir(sd)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills in empty dir, got %d", len(skills))
	}
}

func TestDiscoverInDir_NonexistentDir(t *testing.T) {
	sd := config.SkillDir{Name: "Missing", Path: "/nonexistent/dir"}
	_, err := DiscoverInDir(sd)
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestDiscoverAll_MultipleDirectories(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	os.WriteFile(filepath.Join(dir1, "skill-a.md"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir2, "skill-b.md"), []byte("b"), 0644)

	cfg := &config.Config{
		SkillDirs: []config.SkillDir{
			{Name: "Dir1", Path: dir1},
			{Name: "Dir2", Path: dir2},
		},
	}

	skills, err := DiscoverAll(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills across dirs, got %d", len(skills))
	}

	dirNames := map[string]bool{}
	for _, s := range skills {
		dirNames[s.DirName] = true
	}
	if !dirNames["Dir1"] || !dirNames["Dir2"] {
		t.Errorf("expected skills from both dirs, got dir names: %v", dirNames)
	}
}

func TestDiscoverAll_SkipsBadDirectories(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "good-skill.md"), []byte("ok"), 0644)

	cfg := &config.Config{
		SkillDirs: []config.SkillDir{
			{Name: "Bad", Path: "/nonexistent/dir"},
			{Name: "Good", Path: dir},
		},
	}

	skills, err := DiscoverAll(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].DirName != "Good" {
		t.Errorf("expected 1 skill from Good dir, got %+v", skills)
	}
}

func TestDiscoverAll_WithTildeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".yolo-test-skills-"+t.Name())
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	os.WriteFile(filepath.Join(dir, "test-skill.md"), []byte("test"), 0644)

	cfg := &config.Config{
		SkillDirs: []config.SkillDir{
			{Name: "Home", Path: "~/.yolo-test-skills-" + t.Name()},
		},
	}

	skills, err := DiscoverAll(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill with tilde path, got %d", len(skills))
	}
	if skills[0].Path != filepath.Join(dir, "test-skill.md") {
		t.Errorf("expected expanded path, got %q", skills[0].Path)
	}
}
