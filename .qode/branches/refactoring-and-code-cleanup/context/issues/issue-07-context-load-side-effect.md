# Issue #7: Context Sub-directory Created Silently on Load

## Summary

`context.Load()` calls `os.MkdirAll` on the context sub-directory as a side effect of reading. The error is silently discarded (`_ = os.MkdirAll(...)`). A function named `Load` should only read state — not create directories. This side effect masks missing state, can create unexpected orphaned directories, and hides initialization logic from callers.

## Affected Files

**Source of the problem:**
- `internal/context/context.go` lines 48–50 — the silent `os.MkdirAll` call

**Callers of `context.Load()` (7 call sites across 5 files):**
- `internal/cli/start.go:43`
- `internal/cli/help.go:60`
- `internal/cli/review.go:82`
- `internal/cli/plan.go:102, 152, 197`
- `internal/cli/knowledge_cmd.go:181`

**Existing correct explicit initialization (reference):**
- `internal/cli/branch.go:67` — explicit `os.MkdirAll` in `newBranchCreateCmd()`, the only place that should create the directory
- `internal/plan/refine.go:89, 123` — explicit directory creation in save functions

## Current State

```go
// context.go lines 46–50
func Load(root, branch string) (*Context, error) {
    dir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))

    ctxSubDir := filepath.Join(dir, "context")
    _ = os.MkdirAll(ctxSubDir, 0755)  // silent side effect — error discarded
    // ...
}
```

The rest of `Load` handles missing files gracefully via `readFileOr()` and error-suppressed `os.ReadDir()`. The `MkdirAll` is inconsistent: it mutates the filesystem and silently drops its error.

## Proposed Fix

**Step 1:** Remove lines 48–50 from `context.go`. `Load` already handles missing files gracefully — the sub-directory does not need to exist for a read.

**Step 2:** Add an explicit initialization function:

```go
// EnsureContextDir ensures the context sub-directory exists for a branch.
// Call this before writing any files to the context directory.
func EnsureContextDir(root, branch string) error {
    dir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
    return os.MkdirAll(filepath.Join(dir, "context"), 0755)
}
```

**Step 3:** Add `EnsureContextDir()` calls in the callers that write to the context directory (before the write, not before the read):
- `internal/cli/review.go` — before writing `diff.md`
- `internal/plan/refine.go` — before saving `refined-analysis.md`

`internal/cli/branch.go` already creates the directory correctly and needs no change.

## Impact

- **Principle of least surprise**: `Load` becomes a pure read; filesystem mutations are explicit
- **Error propagation**: directory creation errors surface instead of being silently lost
- **Testability**: tests can call `Load` without unexpected filesystem mutations
- **Explicitness**: it becomes clear in code review which paths create directories and which merely read
