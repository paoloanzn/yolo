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

```bash
git clone https://github.com/paoloanzn/yolo.git
cd yolo
make install   # builds and copies to ~/.local/bin
```

Requires Go 1.21+ and `claude` in your PATH.

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

## License

MIT
