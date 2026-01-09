# Project Guidelines

- Use English for all project content: code, comments, documentation, UI text, tests, and commit messages.

## Commands

```bash
# Build
go build

# Test
go test ./...

# Build with embedded web UI
npm ci --prefix web
npm run build --prefix web
go build -tags=embedui

# Release (uses GoReleaser)
goreleaser release --snapshot --clean
```

## Project Structure

```
cmd/          # Cobra CLI commands
pkg/          # Reusable packages (profile management)
api/          # API definitions
web/          # Svelte + Vite frontend (embedded via -tags=embedui)
docs/         # Documentation
```

## Tech Stack

- **CLI**: Go + Cobra
- **Frontend**: Svelte + TypeScript + Vite
- **Release**: GoReleaser
