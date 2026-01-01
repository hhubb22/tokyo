# Tokyo

Tokyo is a small CLI for managing Claude Code and Codex configuration profiles.

## Features

- Save, switch, list, and delete profiles for Claude Code and Codex.
- Detect modified profiles via file hashes.
- Atomic switch with automatic rollback on failure.
- Safe `save` by default; use `--force` to overwrite.

## Quick Start

```bash
# Claude Code
./tokyo claude save work
./tokyo claude switch work
./tokyo claude current
./tokyo claude list
./tokyo claude delete work

# Codex
./tokyo codex save personal
./tokyo codex switch personal
./tokyo codex current
```

## Commands

```bash
tokyo claude switch <profile>
tokyo claude current
tokyo claude list
tokyo claude save <profile> [--force]
tokyo claude delete <profile>

tokyo codex switch <profile>
tokyo codex current
tokyo codex list
tokyo codex save <profile> [--force]
tokyo codex delete <profile>
```

## Config Locations

- Claude Code: `~/.claude/settings.json`
- Codex: `~/.codex/config.toml`, `~/.codex/auth.json`
- Profiles: `~/.config/tokyo/<tool>/profiles/<profile>`

## Notes

- `current` prints `profile` or `profile (modified)`; `<custom>` means no matching profile.
- `switch` uses a staging copy + rollback to keep configs consistent.
