package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRandomName_Format(t *testing.T) {
	name := RandomName()
	parts := strings.Split(name, "-")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in %q, got %d", name, len(parts))
	}

	// All parts should be non-empty lowercase
	for i, p := range parts {
		if p == "" {
			t.Errorf("part %d is empty in %q", i, name)
		}
		if p != strings.ToLower(p) {
			t.Errorf("part %d (%q) is not lowercase in %q", i, p, name)
		}
	}
}

func TestRandomName_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name := RandomName()
		seen[name] = true
	}
	// With ~105 adjectives and ~70 nouns, collision in 100 samples is extremely unlikely
	if len(seen) < 90 {
		t.Errorf("expected mostly unique names in 100 samples, got %d unique", len(seen))
	}
}

func TestSessionFilename_Format(t *testing.T) {
	fn := SessionFilename()

	if !strings.HasSuffix(fn, ".jsonl") {
		t.Errorf("expected .jsonl suffix, got %q", fn)
	}

	// Should contain a timestamp and a random name separated by underscore
	// Format: 2006-01-02_15-04-05_adj-adj-noun.jsonl
	withoutExt := strings.TrimSuffix(fn, ".jsonl")
	parts := strings.SplitN(withoutExt, "_", 3)
	if len(parts) != 3 {
		t.Fatalf("expected 3 underscore-separated parts in %q, got %d", fn, len(parts))
	}

	// First part: date (YYYY-MM-DD)
	if len(parts[0]) != 10 {
		t.Errorf("expected 10-char date, got %q", parts[0])
	}

	// Second part: time (HH-MM-SS)
	if len(parts[1]) != 8 {
		t.Errorf("expected 8-char time, got %q", parts[1])
	}

	// Third part: random name (adj-adj-noun)
	nameParts := strings.Split(parts[2], "-")
	if len(nameParts) != 3 {
		t.Errorf("expected 3-part random name, got %q", parts[2])
	}
}

func TestExportDir_Default(t *testing.T) {
	dir := ExportDir("")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".yolo", "exports")
	if dir != expected {
		t.Errorf("ExportDir(\"\") = %q, want %q", dir, expected)
	}
}

func TestExportDir_Custom(t *testing.T) {
	dir := ExportDir("/tmp/my-exports")
	if dir != "/tmp/my-exports" {
		t.Errorf("ExportDir(\"/tmp/my-exports\") = %q, want /tmp/my-exports", dir)
	}
}

func TestProjectDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := ProjectDir("/Users/paolo/Fun/yolo")
	expected := filepath.Join(home, ".claude", "projects", "-Users-paolo-Fun-yolo")
	if dir != expected {
		t.Errorf("ProjectDir = %q, want %q", dir, expected)
	}
}

func TestLatestJSONL(t *testing.T) {
	dir := t.TempDir()

	// Create two JSONL files with different mod times
	older := filepath.Join(dir, "old-session.jsonl")
	newer := filepath.Join(dir, "new-session.jsonl")
	os.WriteFile(older, []byte(`{"type":"init"}`+"\n"), 0644)
	// Ensure different mod times
	os.Chtimes(older, time.Now().Add(-10*time.Second), time.Now().Add(-10*time.Second))
	os.WriteFile(newer, []byte(`{"type":"init"}`+"\n"+`{"type":"message"}`+"\n"), 0644)

	// Also create a non-JSONL file that should be ignored
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir.jsonl"), 0755) // directory, should be ignored

	path, _, err := latestJSONL(dir)
	if err != nil {
		t.Fatal(err)
	}
	if path != newer {
		t.Errorf("latestJSONL = %q, want %q", path, newer)
	}
}

func TestLatestJSONL_NoFiles(t *testing.T) {
	dir := t.TempDir()
	_, _, err := latestJSONL(dir)
	if err == nil {
		t.Error("expected error for empty dir")
	}
}

func TestCopyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source.jsonl")
	content := `{"type":"init"}` + "\n" + `{"type":"message"}` + "\n"
	os.WriteFile(src, []byte(content), 0644)

	// Destination in a nested directory that doesn't exist yet
	dst := filepath.Join(t.TempDir(), "nested", "deep", "export.jsonl")

	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("copied content = %q, want %q", string(data), content)
	}
}

func TestWatcher_ExportsOnChange(t *testing.T) {
	projectDir := t.TempDir()
	exportDir := t.TempDir()
	exportPath := filepath.Join(exportDir, "session.jsonl")

	// Create initial session file
	sessionFile := filepath.Join(projectDir, "abc123.jsonl")
	os.WriteFile(sessionFile, []byte(`{"turn":1}`+"\n"), 0644)

	w := StartWatcher(projectDir, exportPath)

	// Give the watcher time to do initial export
	time.Sleep(500 * time.Millisecond)

	// Verify initial export happened
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("export file should exist after initial poll: %v", err)
	}
	if !strings.Contains(string(data), "turn") {
		t.Errorf("export should contain session data, got %q", string(data))
	}

	// Modify the session file
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(sessionFile, []byte(`{"turn":1}`+"\n"+`{"turn":2}`+"\n"), 0644)

	// Wait for watcher to pick up the change (poll interval is 3s, but Stop does final export)
	w.Stop()

	data, err = os.ReadFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "turn\":2") {
		t.Errorf("export should contain updated data after Stop(), got %q", string(data))
	}
}

func TestWatcher_NoProjectDir(t *testing.T) {
	exportDir := t.TempDir()
	exportPath := filepath.Join(exportDir, "session.jsonl")

	// Non-existent project dir - watcher should not crash
	w := StartWatcher("/nonexistent/project/dir", exportPath)
	time.Sleep(200 * time.Millisecond)
	w.Stop()

	// Export file should not exist
	if _, err := os.Stat(exportPath); !os.IsNotExist(err) {
		t.Error("export file should not exist when project dir doesn't exist")
	}
}

func TestExtractExportDir_Variants(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantDir   string
		wantArgs  []string
	}{
		{
			name:     "no flag",
			args:     []string{"yolo"},
			wantDir:  "",
			wantArgs: []string{"yolo"},
		},
		{
			name:     "flag with space",
			args:     []string{"yolo", "--export-dir", "/tmp/exports"},
			wantDir:  "/tmp/exports",
			wantArgs: []string{"yolo"},
		},
		{
			name:     "flag with equals",
			args:     []string{"yolo", "--export-dir=/tmp/exports"},
			wantDir:  "/tmp/exports",
			wantArgs: []string{"yolo"},
		},
		{
			name:     "flag before subcommand",
			args:     []string{"yolo", "--export-dir", "/tmp/exports", "dry-run"},
			wantDir:  "/tmp/exports",
			wantArgs: []string{"yolo", "dry-run"},
		},
		{
			name:     "flag after subcommand",
			args:     []string{"yolo", "dry-run", "--export-dir", "/tmp/exports"},
			wantDir:  "/tmp/exports",
			wantArgs: []string{"yolo", "dry-run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, args := ExtractExportDir(tt.args)
			if dir != tt.wantDir {
				t.Errorf("exportDir = %q, want %q", dir, tt.wantDir)
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", args, tt.wantArgs)
			}
			for i := range args {
				if args[i] != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, args[i], tt.wantArgs[i])
				}
			}
		})
	}
}
