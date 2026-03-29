# YOLO

> **This is an experimental project. Expect rough edges, breaking changes, and minimal polish.**

A small Go CLI that wraps [Claude Code](https://claude.ai/claude-code) with a TUI launcher. Pick your system prompt, permission mode, model, skills, and other options from a menu before each session — instead of remembering flags every time.

## What it does

```
$ yolo
```

A form pops up where you select:

- **System prompt** — from a configurable list (inline or file-based)
- **Permission mode** — bypass, default, plan, auto, etc.
- **Model** — opus, sonnet, haiku
- **Effort level** — low through max
- **Individual skills** — cherry-pick from configured skill directories
- **MCP server configs**
- **Agents**
- **Extra flags** — anything else you want to pass through

Then it launches `claude` with the corresponding flags.

You can also define **presets** for one-click launch (e.g. "YOLO mode", "Planner", "Safe mode").

### Conversation export

Every session automatically exports the Claude Code conversation JSONL to `~/.yolo/exports/`. Each session gets its own file with a timestamped human-readable name:

```
~/.yolo/exports/2026-03-29_12-04-05_bold-keen-fox.jsonl
```

The export updates in the background after every turn, overwriting the same file. Concurrent sessions never collide.

To use a custom export directory for a session:

```bash
yolo --export-dir /path/to/exports
```

## Install

One-liner (downloads the latest release for your platform):

```bash
curl -fsSL https://raw.githubusercontent.com/paoloanzn/yolo/main/install.sh | sh
```

Or build from source:

```bash
git clone https://github.com/paoloanzn/yolo.git
cd yolo
make install   # builds and copies to ~/.local/bin
```

Requires `claude` in your PATH. Building from source requires Go 1.25+.

## Setup

```bash
yolo init      # creates ~/.yolo/config.yaml with defaults
```

Edit `~/.yolo/config.yaml` to add your system prompts, skill directories, presets, etc.

## Commands

| Command | Description |
|---|---|
| `yolo` | Launch the TUI, pick options, start Claude |
| `yolo init` | Create default config |
| `yolo dry-run` | Same TUI but only prints the command |
| `yolo config` | Print config file path |
| `yolo help` | Usage info |
| `yolo --export-dir <path>` | Use a custom export directory for this session |

## Project structure

```
cmd/yolo/main.go              — entry point, CLI dispatch, subprocess launch
internal/
  config/config.go            — types, YAML loading, prompt resolution
  command/command.go           — Selections type, CLI arg building
  export/export.go            — conversation export watcher, session file discovery
  export/words.go             — random word ID generator for filenames
  skill/skill.go              — skill discovery (files, folders, symlinks)
  skill/shadow.go             — shadow config dir for skill isolation
  tui/tui.go                  — interactive TUI form, styled output
```

## License

MIT
