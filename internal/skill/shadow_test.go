package skill

import (
	"os"
	"path/filepath"
	"testing"
)

// setupFakeClaudeDir creates a fake ~/.claude-like directory for testing
// createConfigOverride without depending on the real ~/.claude.
func setupFakeClaudeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme":"dark"}`), 0644)
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Instructions"), 0644)
	os.WriteFile(dir+".json", []byte(`{"oauth":{"accessToken":"test-token"}}`), 0600)
	os.MkdirAll(filepath.Join(dir, "plugins"), 0755)
	os.MkdirAll(filepath.Join(dir, "sessions"), 0755)

	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)
	os.WriteFile(filepath.Join(skillsDir, "default-skill-a.md"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(skillsDir, "default-skill-b.md"), []byte("b"), 0644)
	os.MkdirAll(filepath.Join(skillsDir, "default-skill-c"), 0755)

	return dir
}

func TestCreateConfigOverride_EmptySelection(t *testing.T) {
	result, err := CreateConfigOverride("/fake", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("expected empty string for nil selection, got %q", result)
	}

	result, err = CreateConfigOverride("/fake", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("expected empty string for empty selection, got %q", result)
	}
}

func TestCreateConfigOverride_ShadowStructure(t *testing.T) {
	fakeClaudeDir := setupFakeClaudeDir(t)

	skillDir := t.TempDir()
	skillA := filepath.Join(skillDir, "skill-a")
	skillB := filepath.Join(skillDir, "skill-b.md")
	os.MkdirAll(skillA, 0755)
	os.WriteFile(filepath.Join(skillA, "index.md"), []byte("skill a"), 0644)
	os.WriteFile(skillB, []byte("skill b"), 0644)

	overrideDir, err := CreateConfigOverride(fakeClaudeDir, []string{skillA, skillB})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(overrideDir)
	defer os.Remove(overrideDir + ".json")

	if _, err := os.Stat(overrideDir); err != nil {
		t.Fatalf("override dir does not exist: %v", err)
	}

	// Everything except skills/ should be symlinked
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
		target, _ := os.Readlink(shadowPath)
		if target != filepath.Join(fakeClaudeDir, name) {
			t.Errorf("symlink %q -> %q, want -> %q", name, target, filepath.Join(fakeClaudeDir, name))
		}
	}

	authShadowPath := overrideDir + ".json"
	authInfo, err := os.Lstat(authShadowPath)
	if err != nil {
		t.Fatalf("expected auth file to exist next to shadow dir: %v", err)
	}
	if authInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected auth file to be a symlink")
	}
	authTarget, err := os.Readlink(authShadowPath)
	if err != nil {
		t.Fatalf("failed to read auth symlink: %v", err)
	}
	if authTarget != fakeClaudeDir+".json" {
		t.Fatalf("auth symlink -> %q, want -> %q", authTarget, fakeClaudeDir+".json")
	}

	// skills/ should exist as a real directory (not a symlink)
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

	// Only selected skills in shadow skills/
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

	// Symlinks point to correct targets
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

	skillDir := t.TempDir()
	onlySkill := filepath.Join(skillDir, "only-this-one.md")
	os.WriteFile(onlySkill, []byte("the one"), 0644)

	overrideDir, err := CreateConfigOverride(fakeClaudeDir, []string{onlySkill})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(overrideDir)
	defer os.Remove(overrideDir + ".json")

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
	_, err := CreateConfigOverride("/nonexistent/claude/dir", []string{"/some/skill"})
	if err == nil {
		t.Error("expected error for nonexistent claude dir")
	}
}
