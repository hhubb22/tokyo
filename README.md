# Tokyo

Switch between Claude Code / Codex configurations in one command.

```bash
tokyo claude switch work
tokyo claude switch personal
```

## Why?

If you juggle multiple Claude Code or Codex accounts (work vs personal, different API keys, different settings), you know the pain of manually editing config files. Tokyo saves and restores entire configurations so you can switch contexts instantly.

## Install

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/hhubb22/tokyo/main/install.sh | bash
```

**Go:**

```bash
go install github.com/hhubb22/tokyo@latest
```

**From source:**

```bash
git clone https://github.com/hhubb22/tokyo.git
cd tokyo && go build && sudo mv tokyo /usr/local/bin/
```

## Usage

```bash
# Save current config as a profile
tokyo claude save work

# Switch to a different profile
tokyo claude switch personal

# Check what's active
tokyo claude current
# => work
# => work (modified)   # if you edited the config after switching
# => <custom>          # if no profile is active

# List saved profiles
tokyo claude list

# Delete a profile
tokyo claude delete old-profile

# Overwrite existing profile
tokyo claude save work --force
```

Same commands work for Codex:

```bash
tokyo codex save work
tokyo codex switch work
tokyo codex current
```

## What gets saved?

| Tool | Files |
|------|-------|
| Claude Code | `~/.claude/settings.json` |
| Codex | `~/.codex/config.toml`, `~/.codex/auth.json` |

Profiles are stored in `~/.config/tokyo/`.

## Common issues

**"profile not found"** — Run `tokyo claude list` to see what you have.

**"config file not found"** — Run Claude Code or Codex at least once to create the config file.

**"symlink not allowed"** — Tokyo only works with regular files, not symlinks.

**Interrupted switch** — Just run the switch command again.

## License

MIT
