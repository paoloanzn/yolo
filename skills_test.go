package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestSkillDisplayName(t *testing.T) {
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
		got := skillDisplayName(tt.input)
		if got != tt.want {
			t.Errorf("skillDisplayName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"~/.claude/skills", filepath.Join(home, ".claude/skills")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}
	for _, tt := range tests {
		got := expandHome(tt.input)
		if got != tt.want {
			t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// setupSkillDir creates a temp skill directory with a mix of entry types.
func setupSkillDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Regular file
	os.WriteFile(filepath.Join(dir, "skill-file.md"), []byte("# Skill"), 0644)

	// Directory (folder-based skill)
	os.MkdirAll(filepath.Join(dir, "skill-folder", "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "skill-folder", "index.md"), []byte("# Folder Skill"), 0644)

	// Symlink to a file
	target := filepath.Join(dir, "skill-file.md")
	os.Symlink(target, filepath.Join(dir, "skill-symlink.md"))

	// Hidden entries (should be skipped)
	os.WriteFile(filepath.Join(dir, ".hidden-file"), []byte("hidden"), 0644)
	os.MkdirAll(filepath.Join(dir, ".hidden-dir"), 0755)

	return dir
}

// setupFakeClaudeDir creates a fake ~/.claude-like directory for testing
// createConfigOverride without depending on the real ~/.claude.
func setupFakeClaudeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Simulate typical ~/.claude contents
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme":"dark"}`), 0644)
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Instructions"), 0644)
	os.MkdirAll(filepath.Join(dir, "plugins"), 0755)
	os.MkdirAll(filepath.Join(dir, "sessions"), 0755)

	// Default skills directory with some skills
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)
	os.WriteFile(filepath.Join(skillsDir, "default-skill-a.md"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(skillsDir, "default-skill-b.md"), []byte("b"), 0644)
	os.MkdirAll(filepath.Join(skillsDir, "default-skill-c"), 0755)

	return dir
}

func TestDiscoverSkillsInDir_FindsAllTypes(t *testing.T) {
	dir := setupSkillDir(t)
	sd := SkillDir{Name: "Test", Path: dir}

	skills, err := discoverSkillsInDir(sd)
	if err != nil {
		t.Fatal(err)
	}

	// Should find: skill-file.md, skill-folder, skill-symlink.md (3 entries)
	// Should NOT find: .hidden-file, .hidden-dir
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

func TestDiscoverSkillsInDir_SkipsHiddenEntries(t *testing.T) {
	dir := setupSkillDir(t)
	sd := SkillDir{Name: "Test", Path: dir}

	skills, err := discoverSkillsInDir(sd)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range skills {
		if s.Name == "" || s.Name[0] == '.' {
			t.Errorf("found hidden entry that should have been skipped: %q", s.Name)
		}
	}
}

func TestDiscoverSkillsInDir_SetsCorrectPaths(t *testing.T) {
	dir := setupSkillDir(t)
	sd := SkillDir{Name: "MyDir", Path: dir}

	skills, err := discoverSkillsInDir(sd)
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

func TestDiscoverSkillsInDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	sd := SkillDir{Name: "Empty", Path: dir}

	skills, err := discoverSkillsInDir(sd)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills in empty dir, got %d", len(skills))
	}
}

func TestDiscoverSkillsInDir_NonexistentDir(t *testing.T) {
	sd := SkillDir{Name: "Missing", Path: "/nonexistent/dir"}
	_, err := discoverSkillsInDir(sd)
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestDiscoverSkills_MultipleDirectories(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	os.WriteFile(filepath.Join(dir1, "skill-a.md"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir2, "skill-b.md"), []byte("b"), 0644)

	cfg := &Config{
		SkillDirs: []SkillDir{
			{Name: "Dir1", Path: dir1},
			{Name: "Dir2", Path: dir2},
		},
	}

	skills, err := discoverSkills(cfg)
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

func TestDiscoverSkills_SkipsBadDirectories(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "good-skill.md"), []byte("ok"), 0644)

	cfg := &Config{
		SkillDirs: []SkillDir{
			{Name: "Bad", Path: "/nonexistent/dir"},
			{Name: "Good", Path: dir},
		},
	}

	skills, err := discoverSkills(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].DirName != "Good" {
		t.Errorf("expected 1 skill from Good dir, got %+v", skills)
	}
}

func TestDiscoverSkills_WithTildeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".yolo-test-skills-"+t.Name())
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	os.WriteFile(filepath.Join(dir, "test-skill.md"), []byte("test"), 0644)

	cfg := &Config{
		SkillDirs: []SkillDir{
			{Name: "Home", Path: "~/.yolo-test-skills-" + t.Name()},
		},
	}

	skills, err := discoverSkills(cfg)
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

func TestCreateConfigOverride_EmptySelection(t *testing.T) {
	result, err := createConfigOverride("/fake", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("expected empty string for nil selection, got %q", result)
	}

	result, err = createConfigOverride("/fake", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("expected empty string for empty selection, got %q", result)
	}
}

func TestCreateConfigOverride_ShadowStructure(t *testing.T) {
	fakeClaudeDir := setupFakeClaudeDir(t)

	// Create skills to select
	skillDir := t.TempDir()
	skillA := filepath.Join(skillDir, "skill-a")
	skillB := filepath.Join(skillDir, "skill-b.md")
	os.MkdirAll(skillA, 0755)
	os.WriteFile(filepath.Join(skillA, "index.md"), []byte("skill a"), 0644)
	os.WriteFile(skillB, []byte("skill b"), 0644)

	overrideDir, err := createConfigOverride(fakeClaudeDir, []string{skillA, skillB})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(overrideDir)

	// 1. Override dir exists
	if _, err := os.Stat(overrideDir); err != nil {
		t.Fatalf("override dir does not exist: %v", err)
	}

	// 2. Everything except skills/ should be symlinked
	for _, name := range []string{"settings.json", "CLAUDE.md", "plugins", "sessions"} {
		shadowPath := filepath.Join(overrideDir, name)
		info, err := os.Lstat(shadowPath)
		if err != nil {
			t.Errorf("expected %q to exist in shadow dir: %v", name, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected %q to be a symlink in shadow dir", name)
		}
		// Verify symlink target
		target, _ := os.Readlink(shadowPath)
		if target != filepath.Join(fakeClaudeDir, name) {
			t.Errorf("symlink %q -> %q, want -> %q", name, target, filepath.Join(fakeClaudeDir, name))
		}
	}

	// 3. skills/ should exist as a real directory (not a symlink)
	skillsShadow := filepath.Join(overrideDir, "skills")
	info, err := os.Lstat(skillsShadow)
	if err != nil {
		t.Fatalf("skills/ should exist in shadow dir: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("skills/ in shadow dir should NOT be a symlink")
	}
	if !info.IsDir() {
		t.Error("skills/ in shadow dir should be a directory")
	}

	// 4. Only selected skills in shadow skills/
	shadowSkills, _ := os.ReadDir(skillsShadow)
	if len(shadowSkills) != 2 {
		names := make([]string, len(shadowSkills))
		for i, e := range shadowSkills {
			names[i] = e.Name()
		}
		t.Fatalf("expected 2 skills in shadow, got %d: %v", len(shadowSkills), names)
	}

	expectedNames := map[string]bool{"skill-a": true, "skill-b.md": true}
	for _, entry := range shadowSkills {
		if !expectedNames[entry.Name()] {
			t.Errorf("unexpected skill in shadow: %q", entry.Name())
		}
		linfo, _ := os.Lstat(filepath.Join(skillsShadow, entry.Name()))
		if linfo.Mode()&os.ModeSymlink == 0 {
			t.Errorf("skill %q should be a symlink", entry.Name())
		}
	}

	// 5. Symlinks point to correct targets
	for _, srcPath := range []string{skillA, skillB} {
		name := filepath.Base(srcPath)
		link := filepath.Join(skillsShadow, name)
		target, err := os.Readlink(link)
		if err != nil {
			t.Errorf("failed to read symlink %q: %v", name, err)
			continue
		}
		if target != srcPath {
			t.Errorf("symlink %q -> %q, want -> %q", name, target, srcPath)
		}
	}
}

func TestCreateConfigOverride_NoDefaultSkillsLeak(t *testing.T) {
	fakeClaudeDir := setupFakeClaudeDir(t)

	// Select only one skill
	skillDir := t.TempDir()
	onlySkill := filepath.Join(skillDir, "only-this-one.md")
	os.WriteFile(onlySkill, []byte("the one"), 0644)

	overrideDir, err := createConfigOverride(fakeClaudeDir, []string{onlySkill})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(overrideDir)

	// The shadow skills/ should contain ONLY our selected skill,
	// NOT the 3 default skills from the fake claude dir
	shadowSkills, _ := os.ReadDir(filepath.Join(overrideDir, "skills"))
	if len(shadowSkills) != 1 {
		names := make([]string, len(shadowSkills))
		for i, e := range shadowSkills {
			names[i] = e.Name()
		}
		t.Errorf("expected exactly 1 skill (no defaults leaked), got %d: %v", len(shadowSkills), names)
	}
	if len(shadowSkills) == 1 && shadowSkills[0].Name() != "only-this-one.md" {
		t.Errorf("expected 'only-this-one.md', got %q", shadowSkills[0].Name())
	}
}

func TestCreateConfigOverride_NonexistentClaudeDir(t *testing.T) {
	_, err := createConfigOverride("/nonexistent/claude/dir", []string{"/some/skill"})
	if err == nil {
		t.Error("expected error for nonexistent claude dir")
	}
}
