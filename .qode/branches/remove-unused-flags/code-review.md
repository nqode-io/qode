# Code Review — remove-unused-flags

**Branch:** remove-unused-flags
**Reviewer:** qode AI review
**Date:** 2026-03-29

---

## Pre-Review Incident Report

*This code shipped. Production failure. What went wrong?*

`qode branch remove my-feature --keep-branch-context` completed with exit 0. The engineer saw "Deleted git branch: my-feature" and assumed the operation succeeded fully. Two weeks later they discover `.qode/branches/my-feature/` is gone — the context folder was silently deleted. The flag had no effect because the engineer accidentally invoked a version of the binary built before the flag was added. Since the new flag produces no confirmation message when it *works*, there is also no confirmation message when it silently doesn't work. Silent success is indistinguishable from silent failure.

---

## Review

### 1. Correctness

**Verified: `config.Load` in `branch remove` does not regress on missing `qode.yaml`**

The call to `config.Load(root)` at `internal/cli/branch.go:100` was the highest-risk addition. Confirmed via `internal/config/config.go:31`:
```go
if err := mergeFromFile(projectPath, &cfg); err != nil && !os.IsNotExist(err) {
```
`IsNotExist` is explicitly swallowed — `config.Load` returns defaults if `qode.yaml` is absent. `branch remove` therefore works correctly in a directory without a config file: `cfg.Branch.KeepBranchContext` is `false` (the default), matching prior behaviour. **No regression.**

**Verified: `keepCtx` logic is correct**

`keepCtx := cfg.Branch.KeepBranchContext || flagKeepBranchCtx` — config OR flag → keep. Absence of both → delete. This is the intended unidirectional design. Correct.

**Verified: `RangeArgs(1, 2)` and positional `base` extraction**

`args[0]` = name, `args[1]` (when present) = base. `git.CreateBranch(root, name, "")` calls `git checkout -b name` when base is empty — already tested in `internal/git/git.go`. Edge case of three args correctly rejected by cobra. Correct.

**Verified: `filterLayers` removal is complete**

`filterLayers` was only ever called from `newCheckCmd` under the `--layer` flag. Both were removed together. Confirmed no other call site exists. Correct.

**Verified: `CheckOptions` removal is complete**

One struct with one field. One call site in `check.go`. Both updated together. `runLayerCheck` condition now `if layer.Test.Unit == ""` — identical behaviour to `if false || layer.Test.Unit == ""`. Correct.

---

### 2. Issues Found

---

**Severity:** Medium
**File:** `internal/cli/branch.go:119–126`
**Issue:** No user feedback when `--keep-branch-context` is active. When `keepCtx` is true, the code silently skips the removal block. The only output the user sees is "Deleted git branch: X". There is no message confirming the context folder was preserved. This is the exact failure mode from the incident report above: silent success is indistinguishable from unintended behaviour.
**Suggestion:**
```go
keepCtx := cfg.Branch.KeepBranchContext || flagKeepBranchCtx
if !keepCtx {
    if err := os.RemoveAll(branchDir); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("removing context: %w", err)
    }
    fmt.Printf("Removed context for branch: %s\n", name)
} else {
    fmt.Printf("Kept context for branch: %s\n", name)
}
```

---

**Severity:** Low
**File:** `internal/cli/branch.go:98`
**Issue:** `var flagKeepBranchCtx bool` uses a `flag`-prefixed name for a local variable. The rest of the codebase uses `flag`-prefix only for package-level persistent flag vars (`flagRoot`, previously `flagVerbose` in `root.go`). Local flag vars in command functions use no prefix: `var keepBranch bool` (old), `var skipTests bool`, `var toFile bool`, `var layerName string`. The inconsistency is a minor readability issue — suggests this is a persistent global when it isn't.
**Suggestion:** Rename to `var keepBranchCtx bool`.

---

**Severity:** Nit
**File:** `internal/config/config_test.go:89`
**Issue:** `cfg.Branch.KeepBranchContext != false` is a tautological comparison. Comparing a bool to `false` with `!=` is equivalent to just using the bool value directly.
**Suggestion:**
```go
if cfg.Branch.KeepBranchContext {
    t.Error("expected Branch.KeepBranchContext false by default")
}
```

---

**Severity:** Nit
**File:** `internal/cli/start.go:48`
**Issue:** `paths, _ := knowledge.List(root, cfg)` — swallowing the error is intentional per spec, but the blank identifier gives no hint to future maintainers that this is a deliberate choice. If a future developer adds verbose logging back, they might not realise this was a considered decision.
**Suggestion:** Add a comment:
```go
paths, _ := knowledge.List(root, cfg) // non-fatal: prompt generated without KB if listing fails
```
Same pattern applies to `knowledge_cmd.go:221`.

---

### 3. Properties Verified Safe

**`config.Load` graceful on missing `qode.yaml`** — checked `config.go:31`. `IsNotExist` is swallowed. `branch remove` does not regress. ✅

**Dead code removal is complete** — `runInitNew`, `runInitWorkspace`, `scaffoldProject`, `readLine`, `pickChoice`, `applyIDEFlags` are all unexported (`internal/cli/init.go`). Go build would fail if any were referenced outside the package. Build passes. ✅

**`BranchConfig.KeepBranchContext` omitempty behaviour** — bool zero value (`false`) is omitted from YAML. `TestBranchConfig_OmitEmpty` covers this. Existing `qode.yaml` files without a `branch:` key continue to work. ✅

**`init.go` imports cleaned** — `strings` removed from imports after the wizard functions that used `strings.ToUpper`, `strings.ReplaceAll`, `strings.TrimSpace` were deleted. Build verification confirms no unused imports. ✅

**`ide.go` refactor is safe** — the `return &cobra.Command{...}` form is valid cobra usage. The `cmd` local variable was only needed for `cmd.Flags().Changed(...)` calls, which are now deleted. The `newIDESyncCmd()` pattern already used the direct-return form. ✅

---

## Summary

| Severity | Count |
|---|---|
| Critical | 0 |
| High | 0 |
| Medium | 1 |
| Low | 1 |
| Nit | 2 |

**Overall assessment:** The change correctly implements all 8 tasks from the spec. The highest-risk new dependency — `config.Load` in `branch remove` — is safe because `config.Load` gracefully handles a missing `qode.yaml`. All dead code has been deleted and the build is clean. The one actionable issue is the missing confirmation message when `--keep-branch-context` keeps the context folder.

**Top 3 things to fix before merging:**

1. **Add feedback message when context is kept** (`branch.go`) — without it, `--keep-branch-context` succeeds silently and is indistinguishable from flag-not-working.
2. **Rename `flagKeepBranchCtx` → `keepBranchCtx`** (`branch.go`) — minor but keeps naming conventions consistent.
3. **Add intentional-swallow comment** to `start.go:48` and `knowledge_cmd.go:221` — documents that the blank identifier is a deliberate design choice.

---

## Rating

| Dimension      | Score (0-2) | What you verified                                                                                                              |
|----------------|-------------|--------------------------------------------------------------------------------------------------------------------------------|
| Correctness    | 2           | config.Load graceful on missing qode.yaml (config.go:31); keepCtx OR logic correct; RangeArgs(1,2) correct; filterLayers + CheckOptions removal both complete with no orphaned call sites |
| Code Quality   | 1           | Missing feedback message on context-keep is a real usability gap; flagKeepBranchCtx naming violates local var convention; tautological test comparison |
| Architecture   | 2           | BranchConfig fits naturally in config hierarchy alongside KnowledgeConfig; runner simplification matches existing pattern; init.go rewrite preserves all runInitExisting logic |
| Error Handling | 2           | All new error paths in branch remove explicitly returned; blank identifier swallowing verified intentional per spec; config.Load error correctly returned before name/branchDir are used |
| Testing        | 1           | 3 BranchConfig tests cover default value, YAML round-trip, and omitempty — good for the schema change; no CLI-level tests for --keep-branch-context or positional base arg (no CLI tests existed before this PR either) |

**Total Score: 8.0/10**
