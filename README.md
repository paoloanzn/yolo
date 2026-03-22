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

## Project structure

```
cmd/yolo/main.go              — entry point, CLI dispatch
internal/
  config/config.go            — types, YAML loading, prompt resolution
  skill/skill.go              — skill discovery (files, folders, symlinks)
  skill/shadow.go             — shadow config dir for skill isolation
  command/command.go           — Selections type, CLI arg building
  tui/tui.go                  — interactive TUI form, styled output
```

## License

MIT
