# Tokyo

Tokyo is a CLI tool for managing configuration profiles for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) and [Codex](https://github.com/openai/codex).

## Why Tokyo?

If you use Claude Code or Codex with multiple accounts (work/personal) or different configurations, switching between them manually is tedious and error-prone. Tokyo makes it simple:

```bash
# Save your current work config
tokyo claude save work

# Later, switch to personal config
tokyo claude switch personal

# Check which profile is active
tokyo claude current
# => personal
```

## Installation

### Download Binary (Recommended)

Download the latest release from [GitHub Releases](https://github.com/user/tokyo/releases):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/user/tokyo/releases/latest/download/tokyo_Darwin_arm64.tar.gz
tar xzf tokyo_Darwin_arm64.tar.gz
sudo mv tokyo /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/user/tokyo/releases/latest/download/tokyo_Darwin_x86_64.tar.gz
tar xzf tokyo_Darwin_x86_64.tar.gz
sudo mv tokyo /usr/local/bin/

# Linux (x86_64)
curl -LO https://github.com/user/tokyo/releases/latest/download/tokyo_Linux_x86_64.tar.gz
tar xzf tokyo_Linux_x86_64.tar.gz
sudo mv tokyo /usr/local/bin/

# Linux (ARM64)
curl -LO https://github.com/user/tokyo/releases/latest/download/tokyo_Linux_arm64.tar.gz
tar xzf tokyo_Linux_arm64.tar.gz
sudo mv tokyo /usr/local/bin/
```

### Go Install

```bash
go install github.com/user/tokyo@latest
```

### Build from Source

```bash
git clone https://github.com/user/tokyo.git
cd tokyo
go build -o tokyo
sudo mv tokyo /usr/local/bin/
```

## Quick Start

### Claude Code

```bash
# Save current configuration as "work" profile
tokyo claude save work

# Create another profile
# (first, manually edit ~/.claude/settings.json for personal use)
tokyo claude save personal

# Switch between profiles
tokyo claude switch work
tokyo claude switch personal

# See current profile status
tokyo claude current
# => "work", "work (modified)", or "<custom>"

# List all saved profiles
tokyo claude list

# Delete a profile
tokyo claude delete old-profile
```

### Codex

```bash
tokyo codex save work
tokyo codex switch work
tokyo codex current
tokyo codex list
tokyo codex delete work
```

## Commands

| Command | Description |
|---------|-------------|
| `save <profile>` | Save current config as a named profile |
| `save <profile> --force` | Overwrite existing profile |
| `switch <profile>` | Switch to a saved profile |
| `current` | Show current profile status |
| `list` | List all saved profiles |
| `delete <profile>` | Delete a saved profile |

## How It Works

### Profile Storage

Tokyo stores profiles in `~/.config/tokyo/`:

```
~/.config/tokyo/
├── claude/
│   ├── profiles/
│   │   ├── work/
│   │   │   └── settings.json
│   │   └── personal/
│   │       └── settings.json
│   └── current.json
└── codex/
    ├── profiles/
    │   └── work/
    │       ├── config.toml
    │       └── auth.json
    └── current.json
```

### Managed Config Files

| Tool | Config Files |
|------|--------------|
| Claude Code | `~/.claude/settings.json` |
| Codex | `~/.codex/config.toml`, `~/.codex/auth.json` |

### Current Status

The `current` command shows:

| Output | Meaning |
|--------|---------|
| `work` | Active profile matches saved profile |
| `work (modified)` | Active profile has local changes |
| `<custom>` | No profile is active or config doesn't match any profile |

## Safety Features

- **Atomic switching**: Uses staging files and automatic rollback on failure
- **Safe save**: Won't overwrite existing profiles without `--force`
- **Modification detection**: Uses SHA-256 hashes to detect config changes
- **Security**: Rejects symlinks and non-regular files to prevent attacks

## Troubleshooting

### "profile not found"

The profile doesn't exist. Use `tokyo claude list` to see available profiles.

### "config file not found"

The tool's config file doesn't exist yet. Run the tool (Claude Code or Codex) at least once to create it.

### "symlink not allowed"

Tokyo doesn't support symlinked config files for security reasons. Use regular files only.

### Interrupted switch

If a switch is interrupted (e.g., by Ctrl+C), your config may be in an inconsistent state. Simply run `tokyo <tool> switch <profile>` again to restore consistency.

## License

MIT
