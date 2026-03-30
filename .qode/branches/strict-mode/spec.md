# Technical Specification: Strict Mode — Enforce Step Ordering and Minimum Scores

**Issue:** nqode-io/qode#30
**Branch:** strict-mode
**Date:** 2026-03-30
**Analysis:** iteration 3, score 25/25

---

## 1. Feature Overview

qode's workflow is a linear pipeline — refine → judge → spec → start → review — but nothing today prevents a developer from skipping steps or proceeding despite low scores. A developer can run `/qode-start` without a spec, or generate a spec from a 15/25 analysis. The scoring system is advisory only.

Strict mode closes this gap by introducing a `StepGuard` (`internal/workflow/guard.go`) that checks prerequisites before each guarded command emits a prompt. When `scoring.strict: true` in `qode.yaml`, a blocked step returns a non-zero exit code with an actionable error on stderr. When `strict: false` (the default), the guard emits a structured `STOP.` instruction to stdout so the AI assistant halts and informs the user rather than silently proceeding.

The feature is backward compatible: `strict: false` is the zero value, and all guarded commands accept a `--force` flag to bypass score gates in exceptional cases.

**Success criteria:**
- `strict: true`: blocked step → stderr error message, exit 1, stdout empty.
- `strict: false`: blocked step → `STOP.` instruction on stdout, exit 0.
- `--to-file`: always writes prompt regardless of guard state (debugging is unconditional).
- `--force`: bypasses score/completeness gates; absent-file state remains a hard error.
- `qode workflow status` shows per-step live status with "Up next" guidance.

---

## 2. Scope

### In scope

- `scoring.strict: bool` field in `ScoringConfig` (defaults `false`).
- `internal/workflow/guard.go` — pure `CheckStep` function with prerequisite logic.
- Guards wired into `runPlanSpec`, `runStartCmd`, `runReview` (code + security).
- `--force` flag on all four guarded commands.
- Non-strict stop instruction (stdout) for blocked steps in non-strict mode.
- Strict error (stderr, exit 1) for blocked steps in strict mode.
- Four new `Context` helpers: `HasCodeReview`, `HasSecurityReview`, `CodeReviewScore`, `SecurityReviewScore`.
- Delete `WarnMissingPredecessors` and its 5 tests.
- `qode workflow` command: replace box-border diagram with numbered list.
- `qode workflow status` subcommand: live per-step status with "Up next".
- Documentation: `docs/qode-yaml-reference.md`.

### Out of scope

- No guard on `knowledge_cmd.go` (any subcommand) — optional, always safe.
- No guard on `qode check` (`/qode-check`) — orthogonal quality gate.
- No guard on `plan refine` or `plan judge` — these are always safe to re-run.
- No new `min_score` field on `RubricConfig` — existing thresholds cover all three gates.
- No "soft warnings with escalation" after N bypasses.
- No CI-only `--strict` flag on `qode check`.
- No change to `--to-file` behaviour.

### Assumptions

- Score is read from the `<!-- qode:iteration=N score=S/M -->` header written by `qode plan judge`. A score of 0 with an existing `refined-analysis.md` means the judge has not run yet (not the same as no file).
- Review scores are extracted from `code-review.md` / `security-review.md` via the existing `scoring.ExtractScoreFromFile` helper.
- `qode plan judge` already hard-errors when `refined-analysis.md` is absent (`plan.go:104`) — no change needed there.
- The guard is stdout-path only: `--to-file` is for prompt debugging and unconditionally bypasses the guard.

---

## 3. Architecture & Design

### Component diagram

```
qode.yaml
    │
    ▼
config.ScoringConfig
    │  .Strict bool (new)
    │
    ├──► internal/workflow/guard.go  (new)
    │        CheckStep(step, ctx, cfg) CheckResult
    │        refineMinScore(cfg) int
    │
    ├──► internal/context/context.go  (modified)
    │        HasCodeReview() bool        (new)
    │        HasSecurityReview() bool    (new)
    │        CodeReviewScore() float64   (new)
    │        SecurityReviewScore() float64 (new)
    │        WarnMissingPredecessors()   (deleted)
    │
    └──► internal/cli/  (modified)
             plan.go       runPlanSpec(toFile, force bool)
             start.go      RunE(force bool)
             review.go     runReview(kind, toFile, force bool)
             help.go       newWorkflowCmd() + show/status subcommands
             root.go       registration updated
```

### Affected layers

| Layer | Change type |
|---|---|
| `internal/config/schema.go` | Add `Strict bool` to `ScoringConfig` |
| `internal/context/context.go` | Add 4 helpers, delete `WarnMissingPredecessors` |
| `internal/context/context_test.go` | Delete 5 tests, add 8 tests |
| `internal/workflow/guard.go` | **New file** |
| `internal/workflow/guard_test.go` | **New file** |
| `internal/cli/plan.go` | Add `--force`, wire guard |
| `internal/cli/start.go` | Add `--force`, wire guard |
| `internal/cli/review.go` | Add `--force`, wire guard, strict diff-empty |
| `internal/cli/help.go` | Rename, restructure, add `status` subcommand |
| `internal/cli/root.go` | Update registration |
| `docs/qode-yaml-reference.md` | Document `scoring.strict` |

### Import graph (no cycles)

```
scoring  →  config
context  →  config, scoring
workflow →  context, config, scoring
cli      →  workflow, context, config, scoring, ...
```

Adding `scoring` to `context`'s imports is safe — `scoring` imports only `config`.

### Guard data flow

```
runPlanSpec(toFile=false, force=false)
    │
    ├── load root, cfg, branch, ctx
    │
    ├── [!toFile && !force]
    │       result := workflow.CheckStep("spec", ctx, cfg)
    │       if result.Blocked:
    │           if cfg.Scoring.Strict:
    │               return fmt.Errorf("%s", result.Message)   → exit 1, stderr
    │           else:
    │               fmt.Printf(stopInstruction, result.Message)  → exit 0, stdout
    │               return nil
    │
    └── [not blocked] → render prompt → stdout
```

### Non-strict stop instruction format

```
STOP. Do not continue with this prompt.

<reason>

Inform the user: "<user-facing message>" and wait for further instructions.
```

This is unambiguous to both Cursor composer (reads markdown, follows imperatives) and Claude Code (executes stdout as a prompt directly). Stderr is not used in non-strict mode so IDE integrations capturing stdout receive the instruction correctly.

---

## 4. API / Interface Contracts

### `workflow.CheckResult`

```go
package workflow

type CheckResult struct {
    Blocked bool
    Message string // actionable; includes slash command to run next
}
```

### `workflow.CheckStep`

```go
func CheckStep(step string, ctx *context.Context, cfg *config.Config) CheckResult
```

Pure function — no side effects, no I/O. All branching on strict/non-strict lives in the caller.

**Step prerequisites:**

| Step | Condition | Message |
|---|---|---|
| `"spec"` | `!ctx.HasRefinedAnalysis()` | "No refined-analysis.md found. Run /qode-plan-refine first." |
| `"spec"` | `ctx.LatestScore() == 0` (file present, unscored) | "refined-analysis.md is unscored. Run /qode-plan-judge first." |
| `"spec"` | `ctx.LatestScore() < refineMinScore(cfg)` | "Refine score is S/M, minimum required is T. Run /qode-plan-refine to improve." |
| `"start"` | `!ctx.HasSpec()` | "No spec.md found. Run /qode-plan-spec first." |
| `"review-code"` | (always passes) | — |
| `"review-security"` | (always passes) | — |

### `workflow.refineMinScore` (internal)

```go
func refineMinScore(cfg *config.Config) int {
    if cfg.Scoring.TargetScore > 0 {
        return cfg.Scoring.TargetScore
    }
    return scoring.BuildRubric(scoring.RubricRefine, cfg).Total()
}
```

Falls back to `scoring.BuildRubric(...).Total()` (25 for the default rubric), mirroring the existing `scoring.ParseScore` logic.

### New `Context` methods

```go
func (c *Context) HasCodeReview() bool
func (c *Context) HasSecurityReview() bool
func (c *Context) CodeReviewScore() float64
func (c *Context) SecurityReviewScore() float64
```

All four follow the existing `HasSpec()` / `LatestScore()` pattern.

### Modified CLI signatures

| Function | Current | New |
|---|---|---|
| `runPlanSpec` | `(toFile bool) error` | `(toFile, force bool) error` |
| `runReview` | `(kind string, toFile bool) error` | `(kind string, toFile, force bool) error` |
| `newStartCmd` | no `--force` flag | adds `--force` flag; `force` captured in closure |

### CLI flag additions

All four commands gain:
```
--force    bypass step guard checks
```

Flag style: `cmd.Flags().BoolVar(&force, "force", false, "bypass step guard checks")` — consistent with existing `--to-file`.

### `qode workflow status` output

```
1. Create branch - Completed.
2. Add context - Completed.
3. Refine requirements - 3 iterations, latest score: 25/25.
4. Generate spec - Completed.
5. Implement - Completed.
6. Test locally - Always done by the user.
7. Quality gates - Always done by the user.
8. Review - Code review passed with score: 11/12.
   Security review: not started.

Up next: Run /qode-review-security.
```

Step completion detection (in `qode workflow status`):

| Step | Detection |
|---|---|
| 1. Create branch | Always "Completed." |
| 2. Add context | `ctx.Ticket != ""` |
| 3. Refine requirements | `ctx.HasRefinedAnalysis()`, `len(ctx.Iterations)`, `ctx.LatestScore()` vs `refineMinScore(cfg)` |
| 4. Generate spec | `ctx.HasSpec()` |
| 5. Implement | `git.DiffFromBase(root, "") != ""` |
| 6. Test locally | Always "Always done by the user." |
| 7. Quality gates | Always "Always done by the user." |
| 8a. Code review | `ctx.HasCodeReview()` + `ctx.CodeReviewScore()` vs `cfg.Review.MinCodeScore` |
| 8b. Security review | `ctx.HasSecurityReview()` + `ctx.SecurityReviewScore()` vs `cfg.Review.MinSecurityScore` |
| 9. Capture lessons | Always optional — caption shown, no completion tracking. |

`qode plan judge` is a substep of step 3 (not a separate numbered step). When `refined-analysis.md` exists but `LatestScore() == 0`, step 3 shows: "1 iteration, unscored — run /qode-plan-judge."

---

## 5. Data Model Changes

### `internal/config/schema.go` — `ScoringConfig`

**Before:**
```go
type ScoringConfig struct {
    TargetScore int                     `yaml:"target_score,omitempty"`
    Rubrics     map[string]RubricConfig `yaml:"rubrics,omitempty"`
}
```

**After:**
```go
type ScoringConfig struct {
    TargetScore int                     `yaml:"target_score,omitempty"`
    Strict      bool                    `yaml:"strict,omitempty"`
    Rubrics     map[string]RubricConfig `yaml:"rubrics,omitempty"`
}
```

**Backward compatibility:** `Strict` zero value is `false` — existing `qode.yaml` files without this field behave identically.

**No migration required.** The field is optional (`omitempty`); no existing data is invalidated.

### `internal/config/defaults.go`

Add inline comment only — no code change:
```go
// Strict: false — backward compatible default
```

### No new database tables, collections, or file formats.

---

## 6. Implementation Tasks

Ordered by dependency. Each commit leaves CI green at every step.

- [ ] **Commit 1 — Schema:** `internal/config/schema.go`, `internal/config/defaults.go`
  - Add `Strict bool \`yaml:"strict,omitempty"\`` to `ScoringConfig` (after `TargetScore`, line 96)
  - Add comment `// Strict: false — backward compatible default` to `DefaultConfig()`

- [ ] **Commit 2 — Context helpers + remove `WarnMissingPredecessors`:** `internal/context/context.go`, `internal/context/context_test.go`
  - Add `import "github.com/nqode/qode/internal/scoring"`
  - Add `HasCodeReview()`, `HasSecurityReview()`, `CodeReviewScore()`, `SecurityReviewScore()`
  - Delete `WarnMissingPredecessors` (lines 132–149) and its 5 tests (lines 192–240: all `TestWarnMissingPredecessors_*`)
  - Add 8 new tests using existing `setupBranchDir`/`writeFile` pattern (`package context`, `t.TempDir()`-based):
    - `TestHasCodeReview_Present`, `TestHasCodeReview_Absent`
    - `TestHasSecurityReview_Present`, `TestHasSecurityReview_Absent`
    - `TestCodeReviewScore_ReturnsScore`, `TestCodeReviewScore_MissingFile`
    - `TestSecurityReviewScore_ReturnsScore`, `TestSecurityReviewScore_MissingFile`
  - Run `go test ./internal/context/...`

- [ ] **Commit 3 — New package: `internal/workflow/guard.go` + `guard_test.go`**
  - Define `CheckResult`, `CheckStep`, `refineMinScore` in `package workflow`
  - Table-driven `TestCheckStep` in `guard_test.go` (`package workflow`); build minimal `*context.Context` inline (no filesystem):
    - spec/no-analysis → blocked, message contains "refined-analysis.md"
    - spec/unscored (score=0, file present) → blocked, message contains "/qode-plan-judge"
    - spec/below-default-min (score=20) → blocked, message contains "/qode-plan-refine"
    - spec/meets-default-min (score=25) → not blocked
    - spec/custom-TargetScore=20, score=20 → not blocked
    - spec/custom-TargetScore=20, score=19 → blocked
    - start/no-spec → blocked
    - start/spec-present → not blocked
    - review-code → not blocked
    - review-security → not blocked
  - Run `go test ./internal/workflow/...`

- [ ] **Commit 4 — Wire guard into `runPlanSpec`:** `internal/cli/plan.go`, `internal/cli/plan_test.go` (new)
  - Add `force bool` parameter to `runPlanSpec`; wire from `newPlanSpecCmd` with `--force` flag
  - Remove `ctx.WarnMissingPredecessors("spec", os.Stderr)` (line 199)
  - Insert guard block after context load when `!toFile && !force`
  - Keep existing `HasRefinedAnalysis()` hard error (lines 201–205)
  - Add `internal/cli/plan_test.go` (`package cli`):
    - `TestRunPlanSpec_GuardBlocked_NoAnalysis`: strict=true, no `refined-analysis.md` → error returned
    - `TestRunPlanSpec_GuardBlocked_NonStrict`: strict=false, no analysis → nil returned, stdout contains "STOP"
    - `TestRunPlanSpec_Force_SkipsGuard`: no analysis, force=true → guard skipped (hard-error on absent file, not guard error)
    - `TestRunPlanSpec_Pass`: `refined-analysis.md` scored ≥ 25, strict=true → nil, prompt printed
  - Test infrastructure: set `flagRoot = t.TempDir()` (package-level var in `root.go`); write `.git/HEAD` stub so `git.CurrentBranch` resolves

- [ ] **Commit 5 — Wire guard into `start`:** `internal/cli/start.go`, `internal/cli/start_test.go` (new)
  - Add `force bool` to `newStartCmd` closure; wire `--force` flag
  - Remove `ctx.WarnMissingPredecessors("start", os.Stderr)` (line 46)
  - Insert guard block for step `"start"`
  - Add `internal/cli/start_test.go` (`package cli`):
    - `TestRunStart_GuardBlocked_NoSpec`: no `spec.md`, strict=true → error
    - `TestRunStart_GuardBlocked_NonStrict`: no `spec.md`, strict=false → nil, stdout contains "STOP"
    - `TestRunStart_Force_SkipsGuard`: no spec, force=true → passes guard (may fail later, but not on guard)

- [ ] **Commit 6 — Wire guard into `runReview` (strict diff-empty):** `internal/cli/review.go`, `internal/cli/review_test.go` (new)
  - Add `force bool` to `runReview`; thread through `newReviewCodeCmd` and `newReviewSecurityCmd`
  - Remove `ctx.WarnMissingPredecessors("review", os.Stderr)` (line 79)
  - Update diff-empty block (lines 69–72):
    ```go
    if diff == "" {
        if cfg.Scoring.Strict && !force {
            return fmt.Errorf("no changes detected: commit code first before running a review")
        }
        fmt.Fprintln(os.Stderr, "No changes detected. Commit some code first.")
        return nil
    }
    ```
  - Add `internal/cli/review_test.go` (`package cli`):
    - `TestRunReview_StrictEmptyDiff_Code`: strict=true, empty diff → `runReview("code", false, false)` returns error containing "no changes"
    - `TestRunReview_StrictEmptyDiff_Security`: same for `"security"`
    - `TestRunReview_NonStrict_EmptyDiff_ReturnsNil`: strict=false, empty diff → nil (soft warning)
    - `TestRunReview_Force_EmptyDiff_Proceeds`: strict=true, force=true, empty diff → nil (bypassed)

- [ ] **Commit 7 — `qode workflow` numbered list + `qode workflow status`:** `internal/cli/help.go`, `internal/cli/root.go`
  - Replace `workflowDiagram` constant with numbered-list format (no box borders)
  - Rename `newHelpWorkflowCmd` → `newWorkflowCmd`; update `root.go:68` registration
  - Add `show` subcommand (static numbered list)
  - Add `status` subcommand with step-detection logic per section 4
  - `qode workflow` (no subcommand) falls through to `show` or prints help

- [ ] **Commit 8 — Documentation:** `docs/qode-yaml-reference.md`
  - Add `strict: false` in full reference YAML under `scoring:` (after `target_score`)
  - Add `scoring.strict` field description table entry with: default value, strict=true effect, strict=false effect, `--force` override note

---

## 7. Testing Strategy

### Unit tests

**`internal/workflow/guard_test.go`** (`package workflow`)
- Table-driven `TestCheckStep` with 10 rows (see Commit 3 above)
- Constructs `*context.Context` inline with only the fields relevant to each row
- Uses `config.DefaultConfig()` for standard threshold; overrides `TargetScore` for custom cases
- No filesystem, no cobra — pure function tests

**`internal/context/context_test.go`** (additions to existing file, `package context`)
- 8 new tests for four helpers using existing `setupBranchDir`/`writeFile` infrastructure
- White-box tests in the same package; `t.TempDir()` for isolation

### CLI integration tests (white-box, `package cli`)

**`internal/cli/plan_test.go`** (new)
- Direct calls to `runPlanSpec(toFile, force)` (package-internal)
- `flagRoot = t.TempDir()` to make `resolveRoot()` deterministic
- `.git/HEAD` stub so `git.CurrentBranch` succeeds
- Captures stdout by temporarily replacing `os.Stdout` with a `*os.File` created via `os.Pipe()`
- 4 test functions (see Commit 4 above)

**`internal/cli/start_test.go`** (new)
- Same infrastructure pattern
- 3 test functions (see Commit 5 above)

**`internal/cli/review_test.go`** (new)
- Same infrastructure pattern
- 4 test functions (see Commit 6 above)

### Edge cases to test explicitly

| Scenario | Where |
|---|---|
| `refined-analysis.md` present but `LatestScore() == 0` (unscored) | `guard_test.go` |
| `scoring.target_score: 0` (not set) falls back to rubric total (25) | `guard_test.go` |
| `--force` with `strict: true` bypasses score gate | CLI tests |
| `--force` does NOT bypass absent `refined-analysis.md` (hard error) | CLI tests |
| `qode workflow status` on fresh branch (all steps unstarted) | manual / `status` subcommand |
| `review` with `force=true` and empty diff proceeds past diff-empty block | `review_test.go` |

### Test commands

```bash
go test ./internal/context/...
go test ./internal/workflow/...
go test ./internal/cli/...
go test ./...        # full suite before each commit
```

---

## 8. Security Considerations

### No authentication or authorization changes

The guard operates entirely on local branch context files and git state. No external network calls, credentials, or permissions are involved.

### Input validation

- `step` argument to `CheckStep`: only four values are used internally (`"spec"`, `"start"`, `"review-code"`, `"review-security"`). Unknown step names return `CheckResult{Blocked: false}` (safe default — no gate applied).
- `scoring.ExtractScoreFromFile`: reads a local file and parses a float. Returns 0 on any error — safe default (treated as "score not present").
- `--force` flag: boolean flag. No user-supplied string input.

### Data sensitivity

- Branch context files (`code-review.md`, `security-review.md`) contain AI-generated content from the developer's own repository. No new external data flows are introduced.
- Stop instruction written to stdout contains only the internal guard message — no secrets or credentials.

### No new network surface

The guard does not make any HTTP requests. `scoring.ExtractScoreFromFile` reads only local files.

---

## 9. Open Questions

None. All ambiguities were resolved during requirements refinement:

1. **`RubricConfig.MinScore` vs existing thresholds** — Resolved: reuse `ScoringConfig.TargetScore` (refine), `ReviewConfig.MinCodeScore`, `ReviewConfig.MinSecurityScore`. No duplicate config.
2. **Score = 0 with analysis present** — Resolved: distinct guard message directing user to run `/qode-plan-judge`, not `/qode-plan-refine`.
3. **`--force` semantics** — Resolved: bypasses score/completeness gates; absent-file hard errors remain in all modes.
4. **Exit-code convention with `SilenceErrors: true`** — Resolved: `SilenceErrors` suppresses cobra's own error printing but does not suppress `RunE` error propagation. `main.go:18–21` handles printing and `os.Exit(1)`. No changes to `main.go`.
5. **`qode plan judge` as distinct step** — Resolved: substep of step 3 in `qode workflow status`; not a separate numbered step.
6. **Import cycle risk** — Resolved: `context → scoring` is safe. Full import graph has no cycles.

---

*Spec generated by qode. Copy to nqode-io/qode#30 for team review.*
