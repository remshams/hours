# Agent Guidelines

## Version Control

This repository uses **JJ (Jujutsu)** for version control, not git directly. Use `jj` commands instead of `git` for local operations.

## Making Changesets

- Keep changesets small and focused on a single concern
- Write meaningful descriptions using imperative mood (e.g., "Add feature", "Fix bug")

This repo stays as a single Go module with multiple binaries under `cmd/`.
When validating the split client/server layout, prefer the explicit
`./cmd/hours` and `./cmd/hours-server` targets over the legacy root install
path.

Before creating a changeset, ensure all checks pass:

```bash
# Build
go build -v ./...

# Tests
go test -v ./...

# Build explicit client/server binaries from the single go.mod
go build -o /tmp/hours ./cmd/hours
go build -o /tmp/hours-server ./cmd/hours-server

# Smoke-check both final binaries
/tmp/hours --help
/tmp/hours-server --help

# Live CLI tests against a freshly built client binary
HOURS_BIN=/tmp/hours ./tests/test.sh

# Linter
golangci-lint run
```

## Pull Request Descriptions

When asked to provide a PR description, **do not create a markdown file**. Instead, output markdown-formatted text directly in the chat for copying into GitHub.
