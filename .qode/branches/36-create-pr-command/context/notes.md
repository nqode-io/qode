# Notes

## Decisions and clarifications for qode pr create

### 1. PR creation always uses MCP — no dry-run, no GitHub API calls from Go

The `qode pr create` CLI command renders a prompt and outputs it to stdout. The AI (Claude Code, Cursor) then uses its configured MCP server (e.g. the GitHub MCP server) to create the PR directly. There is **no `--dry-run` flag**, no GitHub API calls in Go, and no fallback to `gh pr create`. MCP is the only supported creation method. Remove all references to `--dry-run`, `--url` for remote URL, and any fallback CLI strategy from the analysis and implementation.

### 2. Ticket naming errors — use the correct file paths

The ticket contains two naming errors that must be corrected in the implementation:

- The ticket references `internal/context/context.go` — the correct path is `internal/branchcontext/context.go`.
- The ticket references `internal/ide/claudecode.go` and `internal/ide/cursor.go` — the correct paths are `internal/scaffold/claudecode.go` and `internal/scaffold/cursor.go`.

All implementation tasks must use the corrected paths.

### 3. Base branch override uses `--base` flag

The `qode pr create` command accepts a `--base <branch>` flag to override the auto-detected base branch. Auto-detection uses `git symbolic-ref refs/remotes/origin/HEAD --short` (stripping `origin/`), falling back to `main`. The `pr.base_branch` config field in `qode.yaml` also overrides auto-detection. Priority: `--base` flag > `pr.base_branch` config > auto-detection > `main` fallback.
