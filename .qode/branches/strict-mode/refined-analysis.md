<!-- qode:iteration=3 score=25/25 -->

# Requirements Analysis: Strict Mode — Enforce Step Ordering and Minimum Scores (Iteration 3)

## 1. Problem Understanding

qode's workflow is a linear pipeline: refine → judge → spec → start → review → knowledge. Today nothing enforces this order. A developer can invoke `/qode-start` without a spec, or `/qode-plan-spec` from a 15/25 analysis. The scoring system exists precisely to gate quality — but it is advisory only. Strict mode closes that gap.

The user need is twofold:
1. **Correctness guarantee**: teams that trust qode's scores need the tooling to enforce minimums so low-quality work cannot silently progress.
2. **Actionable feedback**: when a gate fails, the AI assistant must stop and give the developer a specific, fixable instruction — not silently proceed.

The key clarifications from `notes.md` change the ticket's original proposal in three important ways:
- `knowledge_cmd.go` and `/qode-check` are **not guarded** (optional/always-safe steps).
- The guard affects **stdout only**: `--to-file` is for prompt-template debugging and retains current behaviour unconditionally.
- `qode plan status` is **replaced** by a new `qode workflow status` subcommand with a numbered-list format. The `qode workflow` command changes from a box diagram to a numbered list.

Issue comment (re: #39): `qode plan judge` is a distinct step between `plan refine` and `plan spec`. `runPlanJudge` at `internal/cli/plan.go:104` already hard-errors when `refined-analysis.md` is absent. The score gate belongs in `runPlanSpec`, reading the `<!-- qode:iteration=N score=S/M -->` header written by the judge.

## 2. Technical Analysis

### 2a. Stack-agnosticism and workspace topology

The guard is **fully stack-agnostic**. It operates exclusively on branch context files (`refined-analysis.md`, `spec.md`, `code-review.md`, `security-review.md`) and git diff output — none of which differ by stack. `internal/context.Load(root, branch)` resolves to `.qode/branches/<branch>/` regardless of which stack layers are configured. No per-stack branching is needed in the guard.

In a **multirepo workspace**, each repo has its own `qode.yaml` and its own `.qode/branches/` directory. Each `qode` invocation resolves the nearest `qode.yaml` via `config.FindRoot` (called in `cmd/qode/main.go:27`). The guard reads from that repo's branch context directory — the same scoping already used by all existing commands. No special workspace handling is needed.

**IDE integrations (Cursor and Claude Code)** both consume qode's stdout as the prompt text to execute. The stop instruction for non-strict mode must be unambiguous in both:
- Cursor composer: interprets markdown; follows imperative instructions.
- Claude Code: executes stdout as a prompt directly.

Both will halt on:
```
STOP. Do not continue with this prompt.

<reason>

Inform the user: "<user-facing message>" and wait for further instructions.
```
This is emitted to stdout (replacing the normal prompt). Stderr is not used, so IDE integrations capturing stdout receive the instruction correctly.

### 2b. Affected components

**`internal/config/schema.go`** (`ScoringConfig` struct, lines 94–98)
- Add `Strict bool \`yaml:"strict,omitempty"\`` to `ScoringConfig` after `TargetScore`.
- **No** new `MinScore int` field on `RubricConfig`. Existing thresholds cover all three gates:
  - Refine: `ScoringConfig.TargetScore` (from `scoring.target_score`)
  - Code review: `ReviewConfig.MinCodeScore` (default 10.0/12)
  - Security review: `ReviewConfig.MinSecurityScore` (default 8.0/10)
  Adding `min_score` to `RubricConfig` would duplicate these and create split configuration. Reusing existing fields is correct.

**`internal/config/defaults.go`** (`DefaultConfig`)
- `Strict: false` is the zero value — no code change needed; add inline comment `// Strict: false — backward compatible default` for documentation clarity.

**`internal/context/context.go`**
- Add four helpers following the `HasSpec()` / `LatestScore()` pattern:
  - `HasCodeReview() bool` — `os.Stat(filepath.Join(c.ContextDir, "code-review.md"))` succeeds
  - `HasSecurityReview() bool` — `os.Stat(filepath.Join(c.ContextDir, "security-review.md"))` succeeds
  - `CodeReviewScore() float64` — `scoring.ExtractScoreFromFile(filepath.Join(c.ContextDir, "code-review.md"))`
  - `SecurityReviewScore() float64` — `scoring.ExtractScoreFromFile(filepath.Join(c.ContextDir, "security-review.md"))`
- Add import `"github.com/nqode/qode/internal/scoring"`. No import cycle: `scoring` imports only `config`; `context` imports `config`; adding `scoring` to `context`'s imports is safe.
- Delete `WarnMissingPredecessors` method (lines 133–149 in current file).
- **Delete its 5 tests** in `internal/context/context_test.go` (lines 192–240: `TestWarnMissingPredecessors_Start_NoSpec`, `_Start_HasSpec`, `_Review_NoSpec`, `_Spec_NoAnalysis`, `_Unknown_NoOutput`).
- Add new tests for the four helpers in `internal/context/context_test.go` using the existing `setupBranchDir`/`writeFile` helpers (white-box `package context` style, `t.TempDir()`-based):
  - `TestHasCodeReview_Present` / `TestHasCodeReview_Absent`
  - `TestHasSecurityReview_Present` / `TestHasSecurityReview_Absent`
  - `TestCodeReviewScore_ReturnsScore` / `TestCodeReviewScore_MissingFile`
  - `TestSecurityReviewScore_ReturnsScore` / `TestSecurityReviewScore_MissingFile`

**New `internal/workflow/guard.go`**
```go
package workflow

type CheckResult struct {
    Blocked bool
    Message string // actionable; includes the slash command to run next
}

func CheckStep(step string, ctx *context.Context, cfg *config.Config) CheckResult
```
Step prerequisites:
- `"spec"`:
  1. `!ctx.HasRefinedAnalysis()` → blocked: "No refined-analysis.md found. Run /qode-plan-refine first."
  2. `ctx.LatestScore() == 0` (file present, no score header) → blocked: "refined-analysis.md is unscored. Run /qode-plan-judge first."
  3. `ctx.LatestScore() < refineMinScore(cfg)` → blocked: "Refine score is S/M, minimum required is T. Run /qode-plan-refine to improve."
- `"start"`: `!ctx.HasSpec()` → blocked: "No spec.md found. Run /qode-plan-spec first."
- `"review-code"`, `"review-security"`: always `CheckResult{Blocked: false}`.

Internal helper:
```go
func refineMinScore(cfg *config.Config) int {
    if cfg.Scoring.TargetScore > 0 {
        return cfg.Scoring.TargetScore
    }
    return scoring.BuildRubric(scoring.RubricRefine, cfg).Total()
}
```
`CheckStep` is pure — no side effects. All branching on strict/non-strict lives in the caller.

**New `internal/workflow/guard_test.go`** (`package workflow`, white-box style)
- Table-driven `TestCheckStep` covering:
  - `"spec"` / no analysis → blocked, message contains "refined-analysis.md"
  - `"spec"` / analysis present, score 0 → blocked, message contains "/qode-plan-judge"
  - `"spec"` / analysis present, score below default min (25) → blocked, message contains "/qode-plan-refine"
  - `"spec"` / analysis present, score meets default min → not blocked
  - `"spec"` / analysis present, score meets custom `TargetScore: 20` → not blocked at 20, blocked at 19
  - `"start"` / no spec → blocked
  - `"start"` / spec present → not blocked
  - `"review-code"` → always not blocked
  - `"review-security"` → always not blocked

**`internal/cli/plan.go`** (`runPlanSpec`, line 180)
- Add `force bool` parameter; thread from `newPlanSpecCmd` flag `--force` (`BoolVar`, consistent with existing `--to-file` pattern).
- After context load, before engine use, when `!toFile && !force`:
  ```go
  if result := workflow.CheckStep("spec", ctx, cfg); result.Blocked {
      if cfg.Scoring.Strict {
          return fmt.Errorf("%s", result.Message)
      }
      fmt.Printf("STOP. Do not continue with this prompt.\n\n%s\n\nInform the user: %q and wait for further instructions.\n", result.Message, result.Message)
      return nil
  }
  ```
- Remove `ctx.WarnMissingPredecessors("spec", os.Stderr)` (line 199).
- Keep existing `HasRefinedAnalysis()` hard error (lines 201–205). `--force` bypasses score gates only, not absent-file state.

**`internal/cli/start.go`** (`RunE`)
- Add `--force` flag; same guard pattern for step `"start"`.
- Remove `ctx.WarnMissingPredecessors("start", os.Stderr)` (line 46).

**`internal/cli/review.go`** (`runReview`, line 51)
- Add `force bool` to `runReview` signature; thread through both `newReviewCodeCmd` and `newReviewSecurityCmd`.
- `cfg` is already loaded before the diff-empty check (line 56 before line 65); order is correct.
- Update diff-empty block (lines 66–70):
  ```go
  if diff == "" {
      if cfg.Scoring.Strict && !force {
          return fmt.Errorf("no changes detected: commit code first before running a review")
      }
      fmt.Fprintln(os.Stderr, "No changes detected. Commit some code first.")
      return nil
  }
  ```
- Remove `ctx.WarnMissingPredecessors("review", os.Stderr)` (line 79).

**`internal/cli/help.go`**
- Replace `workflowDiagram` with numbered-list format (no box borders).
- Convert `newHelpWorkflowCmd` to `newWorkflowCmd` returning a parent `workflow` cobra.Command.
- Update `root.go:68` registration from `newHelpWorkflowCmd()` to `newWorkflowCmd()`.
- Add `show` subcommand (static list).
- Add `status` subcommand: loads root, cfg, current branch, context, git diff; prints live step status.

`qode workflow status` output format (per notes):
```
1. Create branch - Completed.
2. Add context - Completed.
3. Refine requirements - 4 iterations, latest score: 25/25.
4. Generate spec - Completed.
5. Implement - Completed.
6. Test locally - Always done by the user.
7. Quality gates - Always done by the user.
8. Review - Code review passed with score: 11/12.

Up next: Complete review step by running /qode-review-security.
```

Step completion detection:
- Step 1: always "Completed."
- Step 2: `ctx.Ticket != ""`
- Step 3: `ctx.HasRefinedAnalysis()`, `len(ctx.Iterations)`, `ctx.LatestScore()` vs `refineMinScore(cfg)` (call exported helper from `workflow` package or inline)
- Step 4: `ctx.HasSpec()`
- Step 5: `git.DiffFromBase(root, "") != ""`
- Steps 6–7: always "Always done by the user."
- Step 8a: `ctx.HasCodeReview()` + `ctx.CodeReviewScore()` vs `cfg.Review.MinCodeScore`
- Step 8b: `ctx.HasSecurityReview()` + `ctx.SecurityReviewScore()` vs `cfg.Review.MinSecurityScore`
- Step 9: "Always optional — capture lessons with /qode-knowledge-add-context."

`qode plan judge` as a substep of step 3, not a separate numbered step (consistent with notes showing 9 numbered steps).

**`docs/qode-yaml-reference.md`**
- Add `strict: false` in the full reference YAML under `scoring:` (after `target_score`).
- Add `scoring.strict` to the Field descriptions table.

### 2c. Exit-code convention

`root.go:48` sets `SilenceErrors: true`. This prevents cobra from printing errors but does **not** suppress the `error` return from `RunE`. The error propagates to `cli.Execute()` → `cmd/qode/main.go:18`, which prints `Error: <message>` to stderr and calls `os.Exit(1)`. Verified at `main.go:18–21`.

- Strict mode, guard blocked: `return fmt.Errorf("%s", result.Message)` → exit 1, stderr message. Correct.
- Non-strict mode, guard blocked: `return nil` → exit 0, stop instruction on stdout. Correct.
- `--to-file`, any state: guard skipped, prompt written to file, exit 0. Correct.
- `--force`, any mode: guard skipped, prompt printed normally. Correct.

No changes to `cmd/qode/main.go` required.

### 2d. Key technical decisions

1. **Reuse existing score thresholds** — no new `RubricConfig.MinScore` to avoid duplicate config.
2. **Guard is stdout-path only** — `--to-file` bypasses guard unconditionally.
3. **`--force` bypasses score gates only** — absent-file state is a hard error in all modes.
4. **Stop instruction format verified for both IDEs** — `STOP.` imperative + `<reason>` + `Inform the user:`.
5. **`CheckStep` is pure** — no side effects; testable without cobra or filesystem.
6. **`WarnMissingPredecessors` and its 5 tests deleted** — no dead code left.

## 3. Risk & Edge Cases

**Score = 0 with analysis present** — worker ran but judge has not. `LatestScore()` returns 0. Guard distinguishes this from no-file: message "refined-analysis.md is unscored — run /qode-plan-judge first." Tested in `guard_test.go`.

**`scoring.target_score` = 0 (not set)** — `refineMinScore` falls back to `scoring.BuildRubric(scoring.RubricRefine, cfg).Total()` = 25 for default rubric. Mirrors `scoring.ParseScore` logic.

**`--force` with `strict: true`** — guard entirely skipped; strict flag irrelevant when force is set.

**`qode workflow status` on a fresh branch** — `ctx.Iterations` empty, `LatestScore()` returns 0, `HasRefinedAnalysis()` false. All steps show "Not started." or "Always done by user." No panic; all nil/empty checks covered by existing helpers.

**`plan judge` as substep** — treated as part of step 3 (Refine requirements), not a separate numbered step. `qode workflow status` shows "3. Refine requirements - N iterations, latest score: S/M." A score of 0 on an existing file means the judge hasn't run; status shows "3. Refine requirements - 1 iteration, unscored — run /qode-plan-judge."

**`review.min_code_score` default 10.0/max 12** — `float64` comparison. Guard message includes max: "Code review score is X/12, minimum required is 10.0."

**Import cycle** — `internal/context` adding `internal/scoring`: safe. `scoring → config`; `context → config + scoring`; `workflow → context + config + scoring`. No cycle.

**`--force` flag name** — no existing `--force` flag in any `internal/cli/*.go` command. Safe to add.

**`WarnMissingPredecessors` deletion** — 3 callers (all in guarded commands); 5 tests in `context_test.go` (lines 192–240) must be deleted alongside the method. No external callers outside the `cli` package.

## 4. Completeness Check

**Acceptance criteria:**
- [ ] `scoring.strict: false` by default (zero value, backward compatible)
- [ ] `strict: true`: guarded command → stderr `Error: <message>`, exit 1, stdout empty
- [ ] `strict: false`: guarded command → stop instruction on stdout, exit 0
- [ ] `--to-file`: always writes prompt regardless of guard state (both modes)
- [ ] `--force` on `plan spec`, `start`, `review code`, `review security`: bypasses score gate; not file-existence gate
- [ ] `plan spec` blocked: no analysis; score 0 (unscored); score below threshold
- [ ] `start` blocked: no spec.md
- [ ] `review code`/`review security` strict: diff-empty → hard error; non-strict: existing soft warning
- [ ] `HasCodeReview()`, `HasSecurityReview()`, `CodeReviewScore()`, `SecurityReviewScore()` on `Context`
- [ ] `WarnMissingPredecessors` and its 5 tests deleted
- [ ] `qode workflow` prints numbered list (no box borders)
- [ ] `qode workflow status` shows per-step live status with "Up next"
- [ ] `docs/qode-yaml-reference.md` documents `scoring.strict`
- [ ] `root.go` registration updated from `newHelpWorkflowCmd()` to `newWorkflowCmd()`
- [ ] All inline Go comments updated; no TODO comments committed

**Implicit requirements:**
- Stop instruction unambiguous to both Cursor and Claude Code.
- `qode workflow status` "Up next" correctly identifies first incomplete step.
- Guard `Message` includes both reason and remediation slash command.
- `qode workflow status` treats `plan judge` as a substep of step 3.

**Explicitly out of scope:**
- No guard on `knowledge_cmd.go` (any subcommand)
- No guard on `/qode-check`
- No guard on `plan refine` or `plan judge`
- No "soft warnings with escalation" after N bypasses
- No CI-only `--strict` flag on `qode check`
- No change to `--to-file` behaviour

## 5. Actionable Implementation Plan

### Commit 1 — Schema: add `Strict` to `ScoringConfig`
Files: `internal/config/schema.go`, `internal/config/defaults.go`
- Add `Strict bool \`yaml:"strict,omitempty"\`` to `ScoringConfig` (after `TargetScore`, line 96)
- Add comment `// Strict: false — backward compatible default` in `DefaultConfig()`

### Commit 2 — Context: add review artifact helpers, remove `WarnMissingPredecessors`
File: `internal/context/context.go`, `internal/context/context_test.go`
- Add `import "github.com/nqode/qode/internal/scoring"`
- Implement `HasCodeReview()`, `HasSecurityReview()`, `CodeReviewScore()`, `SecurityReviewScore()`
- Delete `WarnMissingPredecessors` method (lines 133–149)
- Delete its 5 tests in `context_test.go` (lines 192–240: all `TestWarnMissingPredecessors_*`)
- Add 8 new tests for the four helpers in `context_test.go` using existing `setupBranchDir`/`writeFile` pattern:
  - `TestHasCodeReview_Present`: write `code-review.md` to branchDir, call `Load`, assert `HasCodeReview() == true`
  - `TestHasCodeReview_Absent`: no file, assert `HasCodeReview() == false`
  - `TestHasSecurityReview_Present` / `_Absent`: same for `security-review.md`
  - `TestCodeReviewScore_ReturnsScore`: write `code-review.md` with `**Total Score: 10.0/12**`, assert `CodeReviewScore() == 10.0`
  - `TestCodeReviewScore_MissingFile`: no file, assert `CodeReviewScore() == 0`
  - `TestSecurityReviewScore_ReturnsScore` / `_MissingFile`: same for security
- Run `go test ./internal/context/...` to confirm

### Commit 3 — New package: `internal/workflow/guard.go` + `guard_test.go`
- Define `CheckResult`, `CheckStep`, `refineMinScore` in `package workflow`
- Table-driven `TestCheckStep` in `guard_test.go` (`package workflow`):
  - Build minimal `*context.Context` structs inline (no filesystem needed — `ContextDir`, `RefinedAnalysis`, `Spec`, `Iterations` fields set directly)
  - Use `config.DefaultConfig()` for standard threshold; override `TargetScore` for custom-threshold cases
  - Rows: spec/no-analysis, spec/unscored, spec/below-min, spec/meets-min, spec/custom-threshold-met, spec/custom-threshold-not-met, start/no-spec, start/spec-present, review-code, review-security
- Run `go test ./internal/workflow/...`

### Commit 4 — Wire guard into `runPlanSpec`
File: `internal/cli/plan.go`
- Add `force bool` parameter to `runPlanSpec`; add `--force` flag in `newPlanSpecCmd` (`cmd.Flags().BoolVar(&force, "force", false, "bypass step guard checks")`)
- Remove `ctx.WarnMissingPredecessors("spec", os.Stderr)` (line 199)
- Insert guard block (after context load, when `!toFile && !force`)
- Keep existing `HasRefinedAnalysis()` hard error (lines 201–205)
- Add CLI tests in new file `internal/cli/plan_test.go` (`package cli`):
  - Call `runPlanSpec(toFile, force)` directly (package-internal function, same package test)
  - Set `flagRoot = t.TempDir()` (package-level var in `root.go`) so `resolveRoot()` returns the temp dir
  - Mock git branch via writing `.git/HEAD` to the temp dir so `git.CurrentBranch` resolves correctly
  - `TestRunPlanSpec_GuardBlocked_NoAnalysis`: temp dir with no `refined-analysis.md`; cfg strict=true; assert error returned
  - `TestRunPlanSpec_GuardBlocked_NonStrict`: same setup, strict=false; assert nil returned; capture stdout and assert contains "STOP"
  - `TestRunPlanSpec_Force_SkipsGuard`: no analysis, force=true; assert reaches engine (returns error about missing analysis file, not guard error — distinguishable by message)
  - `TestRunPlanSpec_Pass`: temp dir with `refined-analysis.md` scored ≥ 25, `spec.md` output path writable; strict=true; assert nil (prompt printed)

### Commit 5 — Wire guard into `start`
File: `internal/cli/start.go`
- Add `force bool`; add `--force` flag
- Remove `ctx.WarnMissingPredecessors("start", os.Stderr)` (line 46)
- Insert guard block (step `"start"`)
- Add `internal/cli/start_test.go` (`package cli`):
  - `TestRunStart_GuardBlocked_NoSpec`: no `spec.md`, strict=true → error
  - `TestRunStart_GuardBlocked_NonStrict`: no `spec.md`, strict=false → nil, stdout contains "STOP"
  - `TestRunStart_Force_SkipsGuard`: no spec, force=true → proceeds past guard (may fail later, but not on guard)

### Commit 6 — Wire guard into `runReview` (strict diff-empty)
File: `internal/cli/review.go`
- Add `force bool` to `runReview` signature; thread through `newReviewCodeCmd` and `newReviewSecurityCmd`
- Remove `ctx.WarnMissingPredecessors("review", os.Stderr)` (line 79)
- Update diff-empty block: `cfg.Scoring.Strict && !force` → return error
- Add `internal/cli/review_test.go` (`package cli`):
  - `TestRunReview_StrictEmptyDiff_Code`: temp dir, `cfg.Scoring.Strict=true`, no commits (empty diff) → `runReview("code", false, false)` returns non-nil error containing "no changes"
  - `TestRunReview_StrictEmptyDiff_Security`: same for `"security"`
  - `TestRunReview_NonStrict_EmptyDiff_ReturnsNil`: strict=false, empty diff → returns nil (soft warning)
  - `TestRunReview_Force_EmptyDiff_Proceeds`: strict=true, force=true, empty diff → returns nil (bypassed, proceeds to context load)

### Commit 7 — `qode workflow` numbered list + `qode workflow status`
Files: `internal/cli/help.go`, `internal/cli/root.go`
- Replace `workflowDiagram` with numbered-list string (no border characters)
- Rename `newHelpWorkflowCmd` → `newWorkflowCmd`; update registration in `root.go:68`
- Add `show` subcommand (static numbered list)
- Add `status` subcommand with step-detection logic as specified

### Commit 8 — Documentation
File: `docs/qode-yaml-reference.md`
- Add `strict: false` to full reference YAML under `scoring:` (after `target_score`)
- Add `scoring.strict` field description with default, effect in strict and non-strict modes, and `--force` override

### Order rationale
Commits 1–3: foundational, zero behaviour change. Commits 4–6: guard wiring, each independently testable and reviewable. Commit 7: additive UI. Commit 8: documentation only. Each commit has its own tests so CI stays green at every step.
