# Code Review — Remove Unused Commands

**Branch:** remove-unused-commands  
**Reviewer:** AI Code Review  
**Date:** 2026-03-27

---

## Pre-mortem

This change shipped and caused an incident. What went wrong?

The most plausible failure for a deletion PR like this:
1. A command factory was removed but its `AddCommand` registration was not updated (or vice versa) — causing a nil pointer panic at startup.
2. A package import became orphaned after deleting config_cmd.go, causing a compile error in CI after merge.
3. `git.CheckoutBranch` was deleted but another caller existed that wasn't caught.

**Reading the diff to find it:** None of the above occurred. The diff shows clean atomic removals. See correctness section for per-file verification.

---

## Reviewer Stance

**What does this code assume?**
- That the six removed commands have no callers outside `internal/cli/` — verified by the analysis (detect still used by init.go, CheckoutBranch sole caller was newBranchFocusCmd).
- That `go build ./...` is a sufficient correctness gate — true for this PR since it catches all dead symbol references at compile time.

**Earliest silent failure point:** If a removed command was referenced in a shell script or CI pipeline outside this repository. Not detectable from within the codebase — and explicitly acknowledged as "very low" risk in the spec.

---

## File-by-File Review

### `internal/cli/plan.go`

1. **Verified safe:** `newPlanStatusCmd()` removed completely (lines 172–223 of original); registration on line 21 updated atomically in the same edit. No half-removed state.
2. **Verified safe:** All six imports (`fmt`, `os`, `filepath`, `config`, `gocontext`, `git`, `plan`, `prompt`, `cobra`) are still consumed by the remaining `runPlanRefine` and `runPlanSpec` functions. `go vet` would have caught any orphan.
3. **Verified safe:** The removed function contained `ctx.LatestScore()` and `ctx.Iterations` range — these methods and fields survive in `internal/context/context.go` because other callers reference them.

### `internal/cli/branch.go`

1. **Verified safe:** Both `newBranchListCmd()` and `newBranchFocusCmd()` fully removed; `AddCommand` call updated in the same diff hunk. The `newBranchCreateCmd` and `newBranchRemoveCmd` registrations are unaffected.
2. **Verified safe:** `safeBranchDir` helper is still called by both remaining commands — not accidentally deleted along with the removed functions.
3. **Noted (no action required):** The deleted `newBranchListCmd` contained `current, _ := git.CurrentBranch(root)` — silently discarding an error. This antipattern is now gone. The deletion actually improved error handling discipline in this file.

### `internal/cli/config_cmd.go`

1. **Verified safe:** Entire file deleted. `newConfigCmd` reference removed from `root.go` atomically.
2. **Verified safe:** `detect.Composite` (imported only in this file) is still imported by `internal/cli/init.go:75` — no package orphan.
3. **Verified safe:** `gopkg.in/yaml.v3` was imported only in this file within `internal/cli/`. Checked: no other file in `internal/cli/` imports `yaml`. `go build ./...` confirms no issue (the yaml package remains in go.mod as it's used elsewhere, e.g. `internal/config/`).

### `internal/cli/root.go`

1. **Verified safe:** `newConfigCmd()` removed from the `AddCommand` list. The surrounding commands (`newTicketCmd()`, `newKnowledgeCmd()`) retain correct ordering.
2. **Verified safe:** No other reference to `newConfigCmd` anywhere in the codebase — single definition, single registration, both removed.
3. **Verified safe:** The `Long` description string does not mention `config`, `plan status`, `branch list`, or `branch focus` — confirmed by reading root.go. No doc drift.

### `internal/cli/help.go`

1. **Verified safe:** One line removed from the `workflowDiagram` string constant. The surrounding box-drawing characters form a valid ASCII box after removal — the `├──` line above and the new `├──` opener below connect correctly.
2. **Verified safe:** No other reference to `qode plan status` in `help.go`.
3. **Nit:** The STEP 3 box is now 3 lines shorter than before, which is intentional — the workflow diagram is still readable.

### `internal/git/git.go`

1. **Verified safe:** `CheckoutBranch` removed. Confirmed via grep: the only caller was `newBranchFocusCmd` in `internal/cli/branch.go`, which is also deleted. No other callers in any `*.go` file.
2. **Verified safe:** No import changes required — `CheckoutBranch` used no packages that weren't already used by `CreateBranch` and `DeleteBranch`.
3. **Verified safe:** The remaining `run()` helper and surrounding functions are unaffected. File compiles and vets clean.

### `README.md`

1. **Verified safe:** 6 rows removed from the command reference table. The table remains syntactically valid (backtick-fenced block, consistent column spacing for remaining rows).
2. **Verified safe:** `qode branch create` and `qode branch remove` rows remain; only `list` and `focus` deleted. `qode plan refine` and `qode plan spec` rows remain; only `status` deleted. The `config` block (3 rows + blank line) removed cleanly.
3. **Nit:** There is now a double blank line between the knowledge commands block and `qode workflow` (the removed config block had a trailing blank). This is cosmetic and does not affect rendered output.

---

## Issues Found

**Severity:** Nit  
**File:** [README.md:149](README.md#L149)  
**Issue:** After removing the config block, there is an extra blank line before `qode workflow` in the fenced code block.  
**Suggestion:** Remove the extra blank line so there is a single blank between `qode knowledge search` and `qode workflow`.

No Critical, High, Medium, or Low issues found.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| Nit | 1 |

**Top 3 things to address before merging:**
1. (Nit) Fix the extra blank line in README.md before `qode workflow`.
2. Consider running `qode --help`, `qode plan --help`, and `qode branch --help` manually as a smoke test to confirm removed commands no longer appear in output.
3. No further issues — the deletion is clean, atomic, and compiler-verified.

---

## Rating

| Dimension      | Score (0-2) | What you verified (not what you assumed) |
|----------------|-------------|------------------------------------------|
| Correctness    | 2           | Every AddCommand call site updated atomically with its factory removal; `go build ./... && go vet ./...` pass; grep confirms no remaining references to deleted symbols in production code |
| Code Quality   | 2           | Factory functions removed entirely per project convention; no orphaned imports; no empty stubs left behind; trailing blank cleaned from plan.go |
| Architecture   | 2           | Deletion follows dependency chain correctly: CheckoutBranch removed with its sole CLI caller; config_cmd.go deleted as a unit rather than gutted; detect package correctly identified as surviving via init.go |
| Error Handling | 2           | No new error paths introduced; the deleted code contained two silent error discards (`_, _`) that are now gone — deletion improved this dimension |
| Testing        | 1           | No test files existed for the deleted commands (correct — nothing to delete); `go build` is compile-time verification of absence; no runtime smoke test asserting `qode plan status` returns an unknown-command error |

**Total Score: 9.0/10**

**Justification for 9.0:** This is a well-executed, compiler-verified deletion PR. The score is not 10 because there is no automated runtime assertion that the removed commands are absent from the CLI. A shell-level smoke test (or a Cobra integration test asserting the command tree) would close the remaining gap. The single Nit (double blank line in README) is trivially fixable. No High or Critical findings — the constraints do not cap this score.
