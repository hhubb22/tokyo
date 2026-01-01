# Tokyo CLI Design

Tokyo is a CLI tool for managing Claude Code and Codex configurations.

## Command Structure

### Claude Code Configuration Management

```bash
tokyo claude switch <profile>    # Switch Claude Code to a profile
tokyo claude current              # Show current Claude Code profile
tokyo claude list                 # List Claude Code profiles
tokyo claude save <profile>       # Save current Claude Code config as profile
tokyo claude delete <profile>     # Delete a Claude Code profile
```

### Codex Configuration Management

```bash
tokyo codex switch <profile>      # Switch Codex to a profile
tokyo codex current               # Show current Codex profile
tokyo codex list                  # List Codex profiles
tokyo codex save <profile>        # Save current Codex config as profile
tokyo codex delete <profile>      # Delete a Codex profile
```

## Usage Examples

### Managing Claude Code Configurations

```bash
# List all saved Claude Code profiles
tokyo claude list

# Switch to a saved profile
tokyo claude switch work

# Save current configuration as a profile (fails if exists, use --force to overwrite)
tokyo claude save work-updated

# Check current profile status
tokyo claude current
# Output: work (modified)

# Delete a profile
tokyo claude delete old-profile
```

### Managing Codex Configurations

```bash
# List all saved Codex profiles
tokyo codex list

# Switch to a saved profile
tokyo codex switch personal

# Save current configuration as a profile (fails if exists, use --force to overwrite)
tokyo codex save personal-v2

# Check current profile status
tokyo codex current
# Output: personal

# Delete a profile
tokyo codex delete old-profile
```

## Configuration Storage Structure

```
~/.config/tokyo/
├── claude/
│   ├── profiles/
│   │   ├── work/
│   │   ├── personal/
│   │   └── default/
│   └── current.json
└── codex/
    ├── profiles/
    │   ├── work/
    │   ├── personal/
    │   └── default/
    └── current.json
```

## Profile Status Display

When running `tokyo claude current` or `tokyo codex current`, the output shows:

- **Profile name only**: Configuration matches the saved profile exactly
  ```
  work
  ```

- **Profile name (modified)**: Configuration was based on this profile but has been modified
  ```
  work (modified)
  ```

- **<custom>**: Configuration has never been switched using tokyo, or doesn't match any saved profile
  ```
  <custom>
  ```

## Design Principles

1. **Separate management**: Claude Code and Codex configurations are managed independently
2. **Flexible workflow**: Users can modify configurations manually and save them later
3. **Safe by default**: `save` fails if a profile already exists unless `--force` is provided
4. **Status tracking**: Track which profile is active and detect modifications
5. **Simple and focused**: Core functionality only, no over-engineering

## Actual Configuration File Locations

- **Claude Code**: `~/.claude/settings.json`
- **Codex**: `~/.codex/config.toml` and `~/.codex/auth.json`

## Atomic Switch Flow

1. Resolve the target profile directory and validate required files exist.
2. Copy profile files into a temporary staging directory.
3. Back up current config files to a rollback directory.
4. Swap staged files into the live config location using atomic renames where possible.
5. If any step fails, restore from the rollback directory and report an error.
6. On success, update `current.json` and clean up temp/backup directories.

## Implementation Notes

- `current.json` stores the last switched profile name for comparison
- Profile detection: Compare current config files with saved profiles using file hashes or byte-for-byte equality
- Switching should be atomic: stage changes in a temp directory, back up current config, and roll back if any copy fails
- Each profile directory contains a complete copy of the tool's configuration files
