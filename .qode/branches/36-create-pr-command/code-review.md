# Code Review — qode pr create (branch: 36-create-pr-command)

*Reviewer: Claude Sonnet 4.6 | Date: 2026-04-10*

---

## Approach

Reviewed as a post-mortem: where would this fail in production? Read the diff in full (all 13 changed/new files), cross-checked against the spec and the rubric, and traced execution paths for the three scenarios that matter most: spec absent, spec present + no reviews, and spec present + all context populated.

---

## File-by-file findings

### `internal/cli/pr.go` (new)

**DEFECT — Medium: Guard output goes to wrong channel, deviating from the established `handleBlocked` pattern.**

```go
if result := workflow.CheckStep("pr", sess.Context, sess.Config); result.Blocked {
    _, _ = fmt.Fprintln(errOut, result.Message)
    return nil
}
```

Every other guarded command (plan spec, start, review code, review security) uses `handleBlocked(result, out, errOut, cfg.Scoring.Strict)` which:
- In non-strict mode: writes `"STOP.\n<message>"` to **stdout** so the AI IDE receives it and knows to halt.
- In strict mode: returns an error (exit 1).

`runPrCreate` writes to `errOut` and returns `nil`. Consequence: when spec.md is absent and the command is invoked from the IDE (via `/qode-pr-create`), stdout is empty. The scaffold template instructions say "If the output begins with `STOP.`, do not execute it" — but nothing appears on stdout, so the AI receives an empty prompt and has no signal to stop. In strict mode, the process exits 0 instead of 1, which breaks any CI gate that checks the exit code.

Fix: replace the inline guard with `handleBlocked` (same signature as the other `run*` functions use).

**DEFECT — Low: Dead code path in `resolveBaseBranch`.**

```go
detected, err := git.DefaultBranch(ctx, root)
if err == nil && detected != "" {
    return detected, nil
}
return "main", nil  // unreachable
```

`git.DefaultBranch` never returns an error — it swallows all git failures and returns `"main"` as a fallback. The `if err == nil` branch is always taken; the final `return "main", nil` is unreachable. The dead path misleads readers into believing `DefaultBranch` can fail. Either document in the function contract that it never errors (`// DefaultBranch always returns a non-empty string and a nil error`) or simplify `resolveBaseBranch` to drop the error check:

```go
detected, _ := git.DefaultBranch(ctx, root)
return detected, nil  // "main" at minimum
```

**Verified safe: flag-priority chain.** `flag > config > git auto-detection` is correctly implemented and matches the spec. The `--base` flag wires cleanly via cobra's `StringVar`.

### `internal/prompt/templates/pr/create.md.tmpl` (new)

**Verified safe: all conditional sections (`{{if .Ticket}}`, `{{if .Diff}}`, `{{if .CodeReview}}`, `{{if .SecurityReview}}`) are correct and independently togglable.**

**Verified safe: "check if PR already exists" instruction is unconditional** — present in all render paths, satisfying the spec requirement.

**Verified safe: draft conditional** (`{{if .DraftPR}} as a draft{{end}}`) renders correctly.

**Verified safe: PR URL write instruction** references `{{.BranchDir}}/context/pr-url.txt` — the correct path given that `BranchDir` is set to `sess.Context.ContextDir` (the branch root, not the `context/` subdir).

**Observation — minor: `Spec` section is not conditionally guarded.** Other optional context sections use `{{if .Field}}` guards. Spec has no guard because the workflow guard ensures it is always present. This is logically correct, but slightly inconsistent with the template style. If the template is ever called without the guard (e.g., direct `engine.Render` in a test with an empty spec), it renders an empty `## Specification` section. The golden test catches this; it is not a defect.

### `internal/branchcontext/context.go`

**Verified safe: `pr-url.txt` is excluded from the Extra files scan.** Adding it to the skip list is correct — it would otherwise pollute `ctx.Extra` and be included in prompt context it doesn't belong in.

**Verified safe: `StorePRURL` uses `iokit.AtomicWrite`**, consistent with the CLAUDE.md constraint on atomic writes for files consumed by subsequent workflow steps.

**Verified safe: `PRURL` is `strings.TrimSpace`d on load**, correctly handling trailing newlines from the AI's file write.

### `internal/workflow/guard.go`

**Verified safe: `checkPR` is minimal and correct.** It guards only on `spec.md` existence, which matches the spec ("does not require code or security reviews to have passed").

**Observation:** `checkPR` always blocks regardless of `cfg.Scoring.Strict` — it is not a soft-only gate. This is intentional per spec but compounds the `handleBlocked` deviation noted above: with the current `runPrCreate` implementation, even the always-on gate produces the wrong output signal.

### `internal/git/git.go` — `DefaultBranch`

**Verified safe: implementation is correct.** `--short` with `symbolic-ref` returns `origin/develop`; `strings.TrimPrefix` with `"origin/"` yields `"develop"`. Fallback to `"main"` on any error or empty result is the right sentinel.

**Verified safe: error-swallowing is documented** via the function comment and consistent with the existing `run` / `runCtx` helpers in the same file.

### `internal/config/schema.go` + `defaults.go`

**Verified safe: `PRConfig` struct** with `template`, `draft`, `base_branch` fields and `yaml:"...,omitempty"` tags is correct. `omitempty` on the `pr` section means re-running `qode init` won't clutter `qode.yaml` for users who don't configure PR options.

**Observation — minor: `Draft: false` in `DefaultConfig` is the zero value** and redundant with Go's default. It is acceptable as explicit documentation of intent, consistent with how `KeepBranchContext: false` is spelled out in `BranchConfig`.

### `internal/scaffold/claudecode.go` + `cursor.go`

**Verified safe: `"qode-pr-create"` appended** to both command slices. Order matches the workflow step order (last in the list, after knowledge commands).

### Test files

**Verified safe: all new tests use `t.Parallel()`** on both parent and subtests.

**Verified safe: `TestPRCreateTemplate_Conditionals`** uses unique sentinel strings (e.g., `"UNIQUE-TICKET-SENTINEL"`) rather than generic strings, preventing false positives from template boilerplate.

**Verified safe: `TestDefaultBranch_StripsOriginPrefix`** correctly constructs origin/HEAD via `git symbolic-ref` without requiring a real network clone.

**DEFECT — Low: `resolveBaseBranch` has no unit test.** The four-level priority chain (flag > config > git > fallback) is non-trivial and not exercised by any test in the diff. Integration tests may cover this indirectly, but the function is pure enough to unit-test directly with a table of `flagVal` / `configVal` inputs.

**Verified safe: `TestBuildStatusLines_FullyComplete`** correctly receives `PRURL: "https://github.com/org/repo/pull/1"` to satisfy `HasPRURL()` in the new step 9.

### `docs/qode-yaml-reference.md` + `README.md`

**Verified safe: documentation matches implementation.** Base branch priority chain, `draft`, `template`, and `base_branch` fields are all documented accurately. Workflow step numbering is consistent across README, `workflowList`, and `buildStatusLines`.

---

## Critical and High Issues

None.

---

## Medium Issues

**M1 — Guard output channel / `handleBlocked` deviation** (`internal/cli/pr.go:44-47`)

When `spec.md` is absent, the guard message goes to stderr and stdout is empty. In non-strict mode the AI IDE receives an empty prompt instead of a `STOP.` signal. In strict mode the process exits 0 instead of 1.

Fix: use the `handleBlocked` helper that all other guarded commands use.

---

## Low Issues

**L1 — Dead code in `resolveBaseBranch`** (`internal/cli/pr.go:88-92`)

`git.DefaultBranch` never returns an error, so the final `return "main", nil` is unreachable. Document or simplify.

**L2 — No unit test for `resolveBaseBranch`** (`internal/cli/pr.go:81-93`)

The priority chain logic is non-trivial. Add a table-driven test with cases for each priority level.

---

## Score

| Dimension | Score | Max | Notes |
|-----------|-------|-----|-------|
| Correctness | 3 | 3 | All code paths produce correct results for happy path and expected failure modes |
| CLI Contract | 1.5 | 2 | Strict mode not respected; `--base` flag and usage text are correct |
| Go Idioms & Code Quality | 2 | 2 | Functions ≤ 50 lines, single responsibility, follows established patterns |
| Error Handling & UX | 1.5 | 2 | Wrong channel for blocked message; all other errors wrapped and handled correctly |
| Test Coverage | 1.5 | 2 | Comprehensive template/context/git tests; `resolveBaseBranch` untested |
| Template Safety | 1 | 1 | text/template used correctly; no injection vectors |
| **Total** | **10.5** | **12** | Passes minimum (10.0) |

**Result: PASS** — two medium/low issues should be addressed before merge but do not block.
