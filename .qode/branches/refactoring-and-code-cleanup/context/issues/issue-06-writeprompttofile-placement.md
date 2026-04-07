# Issue #6: `writePromptToFile` Lives in `internal/cli/root.go`

## Summary

`writePromptToFile` is a shared utility used by multiple command files across the CLI package, but it is defined in `internal/cli/root.go` (lines 111–131) alongside root command setup code. `root.go` should contain only root command definition and wiring, not shared utilities. This violates single responsibility and makes the utility harder to discover.

## Affected Files

**Definition:**
- `internal/cli/root.go` lines 111–131 — `writePromptToFile` function

**Callers:**
- `internal/cli/knowledge_cmd.go:165` — `runKnowledgeAddBranch()` saves lesson extraction prompt
- `internal/cli/start.go:90` — saves implementation prompt to `.start-prompt.md`
- `internal/cli/plan.go:127` — `runPlanJudge()` saves judge prompt to `.refine-judge-prompt.md`
- `internal/cli/plan.go:233` — `runPlanSpec()` saves spec prompt to `.spec-prompt.md`
- `internal/cli/review.go:114` — `runReview()` saves code/security review prompts

## Current State

```go
// writePromptToFile atomically writes content to path, creating parent dirs as needed.
func writePromptToFile(path, content string) error {
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return err
    }
    tmp, err := os.CreateTemp(filepath.Dir(path), ".qode-prompt-*")
    if err != nil {
        return err
    }
    tmpName := tmp.Name()
    defer func() { _ = os.Remove(tmpName) }()
    if _, err := tmp.WriteString(content); err != nil {
        _ = tmp.Close()
        return err
    }
    if err := tmp.Close(); err != nil {
        return err
    }
    return os.Rename(tmpName, path)
}
```

The function uses an atomic temp-file-then-rename pattern and is called by 5 sites across 4 command files. `resolveRoot()` (lines 100–109) has the same placement problem — it is called by nearly every command file but also lives in `root.go`.

## Proposed Fix

Create `internal/cli/util.go` and move `writePromptToFile` (and `resolveRoot`) there:

```
internal/cli/
  root.go     ← root command setup only
  util.go     ← writePromptToFile, resolveRoot (new file)
  plan.go
  review.go
  ...
```

- Move `writePromptToFile` (lines 111–131) from `root.go` to `util.go`
- Move `resolveRoot` (lines 100–109) from `root.go` to `util.go`
- Update imports in `root.go` accordingly (remove `"path/filepath"` if no longer needed there)
- No changes to any callers — same package, same symbol names

Zero behavior change. All callers continue to work without modification.

## Impact

- `root.go` shrinks by ~30 lines and focuses solely on command wiring
- Utilities are discoverable in one place (`util.go`)
- Low risk: same package, no API changes, no dependency changes
