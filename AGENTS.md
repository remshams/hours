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

## CodeRabbit Reviews

- CodeRabbit does not review pull requests automatically. Each PR gets one CodeRabbit review.
- Request it only when the PR is finished: your own review rounds are done, CI passes, and the branch is rebased onto its base. For a stack, wait until every layer is finished, then request one review per layer. Use `gh pr comment <n> --body "@coderabbitai review"`.
- Fix all actionable findings in one push and add a regression test for every fix that changes behavior. Reply on each thread with the commit that fixes it, or the reason for declining.
- Before replying, review the fix commits against the commit CodeRabbit reviewed, and check that each "Fixed in <sha>" reply points to a commit that addresses that finding.
- Do not request another review, except when a fix changes production behavior for a Major or Critical finding. Then request one incremental review with `@coderabbitai review`.
- If CodeRabbit replies that the rate limit is exceeded, wait the stated time, then trigger the request again. Count from when its message was posted; the number in it does not count down.

## Pull Request Descriptions

When asked to provide a PR description, **do not create a markdown file**. Instead, output markdown-formatted text directly in the chat for copying into GitHub.
