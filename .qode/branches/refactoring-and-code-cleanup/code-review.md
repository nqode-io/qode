# Code Review — qode

**Branch:** refactoring-and-code-cleanup
**Date:** 2026-04-08

---

## Incident Report (pre-read)

`qode knowledge add-branch feature/old-work` is called in a post-mortems session. The lesson extraction prompt saves to `.qode/branches/main/.knowledge-add-branch-prompt.md` instead of `.qode/branches/feature--old-work/`. The branch field in the rendered prompt also reads "main" — the current branch — not "feature/old-work". The extracted lesson is attributed to the wrong branch, and the file lands in the wrong place. The on-call dev merges and the issue goes unnoticed until someone notices lesson files accumulating under the wrong branch directories.

---

## What this refactoring achieves

The diff introduces five clean structural improvements:

1. **`Session` struct** — eliminates boilerplate `resolveRoot / config.Load / git.CurrentBranch / branchcontext.Load / prompt.NewEngine` in every command handler.
2. **`iokit` package** — consolidates `os.MkdirAll + os.WriteFile` patterns into named helpers with consistent parent-dir creation.
3. **`io.Writer` injection** — all `run*` functions now accept `out, errOut io.Writer` instead of writing directly to `os.Stdout/os.Stderr`, making them testable.
4. **`prompt.Renderer` interface** — decouples plan/review packages from the concrete `*prompt.Engine`.
5. **`config.Validate`** — adds structural validation at load time.

Two correctness bugs were introduced alongside these improvements.

---

## Issues

### High

---

**Severity:** High
**File:** [internal/iokit/iokit.go:310-327](internal/iokit/iokit.go)
**Issue:** `AtomicWrite` accepts a `perm os.FileMode` parameter but never uses it. `os.CreateTemp` creates the temp file with `0600`. After `os.Rename`, the final file is `0600` regardless of what was passed. Every call site passes `0644` (or `0600` for diff.md), but all prompt files — `.refine-prompt.md`, `.spec-prompt.md`, `.start-prompt.md`, `.knowledge-add-branch-prompt.md` — land as `0600`. On multi-user CI systems or when `git diff` tools open these files, the wrong permissions cause access failures.

**Suggestion:**
```go
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
    if err := EnsureDir(filepath.Dir(path)); err != nil {
        return err
    }
    tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
    if err != nil {
        return fmt.Errorf("create temp: %w", err)
    }
    defer func() { _ = os.Remove(tmp.Name()) }()
    if _, err := tmp.Write(data); err != nil {
        _ = tmp.Close()
        return err
    }
    if err := tmp.Close(); err != nil {
        return err
    }
    if err := os.Chmod(tmp.Name(), perm); err != nil {
        return err
    }
    return os.Rename(tmp.Name(), path)
}
```
Also add a `TestAtomicWrite_VerifyPermissions` test mirroring the existing `TestWriteFile_VerifyPermissions`.

---

**Severity:** High
**File:** [internal/cli/knowledge_cmd.go:216-222](internal/cli/knowledge_cmd.go)
**Issue:** `buildBranchLessonData` was changed to use `currentBranch` for both `Branch` in `TemplateData` and the `branchDir` path, replacing the previous `branches[0]`. This is a behavioral regression. `qode knowledge add-branch feature/shipped-work` used to:
- set `Branch: "feature/shipped-work"` in the rendered prompt
- save `--to-file` output to `.qode/branches/feature--shipped-work/.knowledge-add-branch-prompt.md`

Now it does both using the current git branch name. The `branches` argument list — the whole point of the command — is used only for iterating context files, not for naming the output. Running the command on any non-current branch produces a mislabeled prompt.

**Suggestion:** Restore the original semantics. `currentBranch` should only be the fallback when `args` contains only the current branch:
```go
// branchDir and Branch should reference the first target branch, not the current one.
// currentBranch is only needed to construct the session; remove it from TemplateData.
branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branches[0]))

return prompt.TemplateData{
    Project:  prompt.TemplateProject{Name: engine.ProjectName()},
    Branch:   branches[0],
    ...
```
If the intent was to always use the current branch for output location, that needs to be documented and the function signature should not accept `currentBranch` as a separate parameter at all.

---

### Medium

---

**Severity:** Medium
**File:** [internal/cli/session.go:13](internal/cli/session.go)
**Issue:** `Session.Engine` is typed as `*prompt.Engine`, not `prompt.Renderer`. The `Renderer` interface was introduced this very PR to decouple callers from the concrete engine — but `Session` leaks the concrete type. Any code that receives a `*Session` (e.g., future command extensions or tests) cannot substitute a mock renderer without also having a real engine. The interface is partially pointless.

**Suggestion:**
```go
type Session struct {
    Root    string
    Config  *config.Config
    Branch  string
    Context *branchcontext.Context
    Engine  prompt.Renderer  // changed from *prompt.Engine
}
```
`loadSession` still returns a `*prompt.Engine` internally; it satisfies `prompt.Renderer`, so the assignment is fine.

---

**Severity:** Medium
**File:** [internal/log/log.go:507](internal/log/log.go)
**Issue:** `Logger` is a package-level `*slog.Logger` initialized only by `log.Init()`. If any test calls a function that reaches `log.Warn(...)` (e.g., the branch-not-found warning in `buildBranchLessonData`) without first calling `log.Init()`, the program panics with a nil pointer dereference. `main.go` calls `log.Init()` — tests do not.

**Suggestion:** Initialize `Logger` at package level with a default (discard in tests is fine):
```go
var Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

func Init() {
    // overrides with real handler
    Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
```
Or use `slog.Default()` as the fallback, which is always non-nil.

---

**Severity:** Medium
**File:** [internal/scaffold/claudecode.go:3550](internal/scaffold/claudecode.go), [internal/scaffold/cursor.go:3739](internal/scaffold/cursor.go)
**Issue:** `SetupClaudeCode` and `SetupCursor` still call `fmt.Printf(...)` directly (writes to `os.Stdout`), even though every other refactored function was converted to accept `out io.Writer`. This inconsistency means scaffold output cannot be suppressed or captured in tests, and these functions cannot follow the same testing pattern established by the rest of the PR.

**Suggestion:** Pass an `out io.Writer` to both functions and use `fmt.Fprintf(out, ...)`. Both call sites (`init.go`) already pass `out` around and can thread it through.

---

### Low

---

**Severity:** Low
**File:** [internal/cli/plan.go:1285](internal/cli/plan.go) (`runPlanJudge`)
**Issue:** `runPlanJudge` does not apply `flagStrict` to the session config, while `runPlanSpec`, `runReview`, and `runStart` all do. If a user passes `--strict` to `qode plan judge`, it has no effect. This is inconsistent behavior — `--strict` is a persistent flag registered on `rootCmd`.

**Suggestion:** Add the same pattern present in the other handlers:
```go
if flagStrict {
    sess.Config.Scoring.Strict = true
}
```

---

**Severity:** Low
**File:** [internal/cli/integration_test.go:782](internal/cli/integration_test.go)
**Issue:** `setupProject` only resets `flagRoot` in its `t.Cleanup`. `flagStrict` (and any other persistent flags modified during a test) can leak into subsequent tests because `rootCmd` is a package-level var reused across tests. Since integration tests run with `-tags integration` and may be run with `-parallel`, a test that sets `--strict` could poison a later test.

**Suggestion:** In `setupProject`'s cleanup:
```go
t.Cleanup(func() {
    flagRoot = ""
    flagStrict = false
})
```

---

**Severity:** Low
**File:** [internal/cli/init_test.go:650-655](internal/cli/init_test.go)
**Issue:** `TestRunInitExisting_NoDetectionOutput` now trivially passes. It calls `runInitExisting(&bytes.Buffer{}, dir)` and then asserts stdout is empty — which it always will be because `runInitExisting` no longer writes to `os.Stdout` at all. The test claims to verify "no detection output" but exercises nothing meaningful after this refactoring.

**Suggestion:** Either delete this test (the behavior it was guarding against — detection output leaking to stdout — cannot happen anymore by construction), or rewrite it to assert the `bytes.Buffer` contains specific expected output and `os.Stdout` remains clean.

---

### Nit

---

**Severity:** Nit
**File:** [internal/prompt/templates/scaffold/qode-knowledge-add-branch.claude.md.tmpl](internal/prompt/templates/scaffold/qode-knowledge-add-branch.claude.md.tmpl)
**Issue:** Extra blank line between the heading and `Run this command...` (lines 2 and 3 in the template). All other `.claude.md.tmpl` files have the command body directly after the heading.

**Suggestion:** Remove the extra blank line.

---

**Severity:** Nit
**File:** [.gitignore](.gitignore)
**Issue:** Missing newline at end of file (shown as `\ No newline at end of file` in the diff).

**Suggestion:** Add a trailing newline.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 2 |
| Medium   | 3 |
| Low      | 3 |
| Nit      | 2 |

**Total: 10 issues**

### Top 3 before merging

1. **Fix `AtomicWrite` permissions** — prompt files land as `0600` silently; add the `Chmod` call and a permissions test.
2. **Restore `buildBranchLessonData` semantics** — `qode knowledge add-branch <branch>` saves to the wrong directory and renders the wrong branch name; revert `currentBranch` to `branches[0]` for `Branch` and `branchDir`.
3. **Protect against `log.Logger` nil panic** — initialize `Logger` to a default non-nil value at declaration so tests that reach warn paths don't panic.

---

## Rating

| Dimension | Score | What you verified |
|-----------|-------|-------------------|
| Correctness (0–3) | 2.0 | `AtomicWrite`: confirmed `perm` unused by reading all 52 lines — temp file created with `0600`, `perm` arg passed but never applied. `buildBranchLessonData`: traced diff at lines 1197-1222 — `Branch` and `branchDir` both changed from `branches[0]` to `currentBranch`; confirmed no existing test covers multi-branch argument. All other logic paths (guard checks, error propagation, session loading) correct. |
| CLI Contract (0–2) | 1.5 | STOP/strict/force paths verified in `plan_test.go`, `review_test.go`, and new integration tests. Non-strict empty-diff returns nil confirmed. `knowledge add-branch` CLI contract broken: target branch is silently ignored for output location and template Branch field. |
| Go Idioms & Code Quality (0–2) | 1.5 | `io.Writer` injection consistent across all 7 run* functions. `Session` struct eliminates ~15 lines of boilerplate per command. `iokit` helpers idiomatic and correctly handle parent-dir creation. `Session.Engine` typed as `*prompt.Engine` instead of `prompt.Renderer` — interface introduced this PR is underutilised. `scaffold` still writes to `os.Stdout` directly. |
| Error Handling & UX (0–2) | 1.5 | `ParseIterationFromOutput` errors previously swallowed with `_ =`; now propagated with context — verified at lines 2658-2669. Warn-path migration to slog correct. `log.Logger` nil at test time is a latent panic: no nil-guard or default initialization at declaration site. |
| Test Coverage (0–2) | 1.5 | `iokit` package: 8 tests covering all 4 functions — complete. `config/validate`: 9 tests covering all validation rules. Integration tests: 6 scenarios covering plan, spec, review paths. Gaps: `AtomicWrite` permissions untested; `TestRunInitExisting_NoDetectionOutput` trivially passes; `flagStrict` leak possible in integration suite. |
| Template Safety (0–1) | 1.0 | `judge_refine.md.tmpl` Go template syntax correctly escaped with `{{"{{"}}` and `{{"}}"}}`. Scaffold templates use `{{.Project.Name}}` only — no user-supplied data interpolated without escaping. |

**Total Score: 9.0/12**
**Minimum passing score: 10/12**

> Score capped at 9.0 by two High findings (one correctness bug in `AtomicWrite`, one behavioral regression in `buildBranchLessonData`). The structural improvements in this PR are solid and the `io.Writer` injection pattern is well-executed — these two specific fixes are all that stand between this and a passing score.
