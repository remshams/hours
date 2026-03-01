# Agent Guidelines

## Version Control

This repository uses **JJ (Jujutsu)** for version control, not git directly. Use `jj` commands instead of `git` for local operations.

## Making Changesets

- Keep changesets small and focused on a single concern
- Write meaningful descriptions using imperative mood (e.g., "Add feature", "Fix bug")

Before creating a changeset, ensure all checks pass:

```bash
# Build
go build -v ./...

# Tests
go test -v ./...

# Live CLI tests (requires: go install .)
cd tests && ./test.sh

# Linter
golangci-lint run
```

## Pull Request Descriptions

When asked to provide a PR description, **do not create a markdown file**. Instead, output markdown-formatted text directly in the chat for copying into GitHub.
