package export

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ExtractExportDir scans args for --export-dir <path> and returns the value
// plus the remaining args with --export-dir removed.
func ExtractExportDir(args []string) (string, []string) {
	var exportDir string
	remaining := make([]string, 0, len(args))
	skip := false
	for i, a := range args {
		if skip {
			skip = false
			continue
		}
		if a == "--export-dir" && i+1 < len(args) {
			exportDir = args[i+1]
			skip = true
			continue
		}
		if strings.HasPrefix(a, "--export-dir=") {
			exportDir = strings.TrimPrefix(a, "--export-dir=")
			continue
		}
		remaining = append(remaining, a)
	}
	return exportDir, remaining
}

const defaultExportDir = ".yolo/exports"
const pollInterval = 3 * time.Second

// ExportDir returns the resolved export directory.
// If custom is non-empty it is used as-is, otherwise ~/.yolo/exports/.
func ExportDir(custom string) string {
	if custom != "" {
		return custom
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultExportDir)
}

// SessionFilename returns a filename like "2026-03-29_12-04-00_bold-keen-fox.jsonl".
func SessionFilename() string {
	ts := time.Now().Format("2006-01-02_15-04-05")
	name := RandomName()
	return fmt.Sprintf("%s_%s.jsonl", ts, name)
}

// ProjectDir returns the Claude session directory for the given working directory.
// Claude Code stores sessions in ~/.claude/projects/{encoded-cwd}/ where the
// encoding replaces every "/" with "-".
func ProjectDir(cwd string) string {
	home, _ := os.UserHomeDir()
	encoded := strings.ReplaceAll(cwd, "/", "-")
	return filepath.Join(home, ".claude", "projects", encoded)
}

// latestJSONL finds the most recently modified .jsonl file in dir.
func latestJSONL(dir string) (string, time.Time, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", time.Time{}, err
	}

	var best string
	var bestMod time.Time

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(bestMod) {
			best = filepath.Join(dir, e.Name())
			bestMod = info.ModTime()
		}
	}
	if best == "" {
		return "", time.Time{}, fmt.Errorf("no .jsonl files in %s", dir)
	}
	return best, bestMod, nil
}

// copyFile copies src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// Watcher polls the Claude project directory for session file changes
// and copies the latest JSONL to the export path.
type Watcher struct {
	projectDir string
	exportPath string
	lastMod    time.Time
	stop       chan struct{}
	wg         sync.WaitGroup
}

// StartWatcher begins polling in the background.
// Call Stop() to terminate the watcher and perform a final export.
func StartWatcher(projectDir, exportPath string) *Watcher {
	w := &Watcher{
		projectDir: projectDir,
		exportPath: exportPath,
		stop:       make(chan struct{}),
	}
	w.wg.Add(1)
	go w.loop()
	return w
}

func (w *Watcher) loop() {
	defer w.wg.Done()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Do an initial check immediately
	w.tryExport()

	for {
		select {
		case <-ticker.C:
			w.tryExport()
		case <-w.stop:
			// Final export before shutting down
			w.tryExport()
			return
		}
	}
}

func (w *Watcher) tryExport() {
	src, mod, err := latestJSONL(w.projectDir)
	if err != nil {
		return
	}
	if !mod.After(w.lastMod) {
		return
	}
	if err := copyFile(src, w.exportPath); err != nil {
		return
	}
	w.lastMod = mod
}

// Stop signals the watcher to perform a final export and shut down.
// It blocks until the watcher goroutine has exited.
func (w *Watcher) Stop() {
	close(w.stop)
	w.wg.Wait()
}
