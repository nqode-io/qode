# Code Review — strict-mode branch

**Reviewer:** Claude Sonnet 4.6
**Date:** 2026-03-30
**Diff:** `.qode/branches/strict-mode/diff.md`
**Spec:** `.qode/branches/strict-mode/spec.md`

---

## Pre-reading incident report

Fictional post-mortem (written before reading the diff):

> **Incident:** A developer on a project with `scoring.strict: true` has been blocked twice by the spec guard (score too low). They run `qode start --force` to unblock themselves, confident that `--force` only bypasses score gates. But their `spec.md` was deleted by a rebase accident the previous day. The command succeeds with exit 0. Their AI assistant reads the prompt, tries to open `spec.md`, reports it missing, and proceeds to generate a "best-effort" implementation from incomplete context. The implementation is committed and merged. Three days later the PM asks why the feature doesn't match the design. Root cause: `--force` silently bypassed the absent-spec check even though the spec says absent-file state is a hard error.

---

## Issues

### Medium

**M1 — `start.go` has no hard error for absent `spec.md`; `--force` silently generates an empty-spec prompt**
- **File:** `internal/cli/start.go:48-56`
- **Issue:** `plan.go:212-216` has an explicit hard error after the guard — even with `--force`, if `refined-analysis.md` is absent, the function returns `fmt.Errorf("no refined analysis")`. The spec (section 1, `--force` semantics) states: *"bypasses score/completeness gates; absent-file state remains a hard error."* For `start`, the only guard condition is `!ctx.HasSpec()` — an absent-file state. But there is no analogous post-guard hard error. With `force=true` and no `spec.md`, the code falls through to `plan.BuildStartPrompt`. The template references `spec.md` by path (not by inlining `ctx.Spec`), so the AI receives a well-formed prompt pointing at a file that does not exist.
- **Suggestion:**
  ```go
  // After the guard block (around line 56), before knowledge.List:
  if !ctx.HasSpec() {
      fmt.Fprintln(os.Stderr, "No spec.md found.")
      fmt.Fprintf(os.Stderr,
          "Run /qode-plan-spec first and save the output to:\n  .qode/branches/%s/spec.md\n",
          branch)
      return fmt.Errorf("no spec")
  }
  ```
  Update `TestRunStart_Force_SkipsGuard` to assert `err != nil` and that the error contains `"no spec"`.

---

### Low

**L1 — `TestRunReview_Force_EmptyDiff_Proceeds` test name misrepresents actual behaviour**
- **File:** `internal/cli/review_test.go:63-78`
- **Issue:** With `strict=true`, `force=true`, and empty diff, `runReview` does NOT proceed to generate a review prompt. It falls through to `review.go:77-79` (the non-strict path) which prints to stderr and returns nil — exit 0, no prompt. The test name claims it "Proceeds"; a developer reading it will infer that `--force` causes prompt generation on an empty diff, which it does not. The `--force` flag in this context only changes the exit code (nil vs error), not whether a review prompt is produced.
- **Suggestion:** Rename to `TestRunReview_Force_EmptyDiff_ReturnsNil` and add a comment: *`// force bypasses the strict-mode error but does not generate a prompt on empty diff.`* If force should actually produce a prompt regardless of diff (consistent with its semantics in other commands), move the nil-return before the `!force` check:
  ```go
  if diff == "" && !force {
      if cfg.Scoring.Strict {
          return fmt.Errorf("no changes detected: commit code first before running a review")
      }
      fmt.Fprintln(os.Stderr, "No changes detected. Commit some code first.")
      return nil
  }
  ```

**L2 — `TestRunPlanSpec_GuardBlocked_Unscored` missing at CLI level**
- **File:** `internal/cli/plan_test.go`
- **Issue:** `guard_test.go` covers `"spec/unscored"` at the unit level (score=0, file present → blocked, message contains `/qode-plan-judge`). No CLI-level test verifies that `runPlanSpec` surfaces this case. If the guard message format changes or the integration breaks for this specific path (the guard fires on `LatestScore() == 0`, which is a different code path from `!HasRefinedAnalysis()`), there is no regression protection.
- **Suggestion:** Add:
  ```go
  func TestRunPlanSpec_GuardBlocked_Unscored(t *testing.T) {
      root := setupPlanTestRoot(t, "test-branch")
      cfg := "project:\n  name: test\n  stack: go\nscoring:\n  strict: true\n"
      if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(cfg), 0644); err != nil {
          t.Fatalf("WriteFile qode.yaml: %v", err)
      }
      // Write analysis without a score header.
      writePlanFile(t, root, "test-branch", "refined-analysis.md",
          "<!-- qode:iteration=1 -->\n# Analysis\nContent.")
      err := runPlanSpec(false, false)
      if err == nil {
          t.Fatal("expected error for unscored analysis in strict mode")
      }
      if !strings.Contains(err.Error(), "qode-plan-judge") {
          t.Errorf("error should mention /qode-plan-judge, got: %v", err)
      }
  }
  ```

**L3 — `HasCodeReview`/`HasSecurityReview` silently suppress permission errors; undocumented API asymmetry with `HasSpec`**
- **File:** `internal/context/context.go:132-151`
- **Issue:** `HasCodeReview()` and `HasSecurityReview()` call `os.Stat` and return `false` for both "file absent" and "permission denied". A permission error will cause `qode workflow status` to display "Not started." instead of an error. Additionally, `HasSpec()` and `HasRefinedAnalysis()` check in-memory strings loaded at `Load()` time, while these four new methods check the filesystem at call time — a caller using both APIs in the same request can observe different filesystem states.
- **Suggestion:** Add a comment documenting both properties:
  ```go
  // HasCodeReview returns true if a code-review.md exists in the branch context.
  // Note: unlike HasSpec (in-memory check from Load), this checks the filesystem
  // at call time. Permission errors are treated as "not present".
  ```

---

### Nit

**N1 — `fmt.Errorf("%s", result.Message)` should be `errors.New`**
- **File:** `internal/cli/plan.go:205`, `internal/cli/start.go:51`
- **Issue:** `fmt.Errorf("%s", result.Message)` creates an error with no wrapping; the `%s` verb with a string arg is a no-op format. Idiomatic Go for a static string error is `errors.New(result.Message)`. This pattern is inconsistent with `errors.New` usage elsewhere in the codebase.
- **Suggestion:** `return errors.New(result.Message)` (add `"errors"` to imports in each file).

**N2 — `buildStatusLines` helper functions are untested**
- **File:** `internal/cli/help.go:78-191`
- **Issue:** `buildStatusLines`, `refineStatus`, `reviewStatus`, and `reviewItemStatus` are pure functions with 9+ branching paths (4 paths in `refineStatus` alone). No unit tests exist for any of them. The M1 fix in this review cycle changed the signature of `reviewItemStatus`, so these paths are particularly vulnerable to regression.
- **Note:** Not blocking, but a `help_test.go` with table-driven cases would give these much-needed coverage.

---

## Verified safe

- **`plan.go:202-216`** — Guard runs before hard-error. With `force=true`: guard skipped, hard-error fires on absent `refined-analysis.md`. With `force=false+strict=true+no-analysis`: guard error returned. With `force=false+strict=false+no-analysis`: STOP instruction to stdout, nil returned. All four combinations consistent with spec table.
- **`start.go:48-56`** — Guard fires when `!ctx.HasSpec()`. `toFile` unconditionally bypasses the guard at line 48 (guard block is `if !toFile && !force`). `force` captured correctly in closure from `newStartCmd` scope.
- **`review.go:73-79`** — Strict+empty diff returns `fmt.Errorf(...)` (exit 1) when `!force`. Non-strict+empty diff prints to stderr and returns nil (exit 0). `force=true` skips the strict error (but still returns nil without a prompt — see L1). All three paths verified.
- **`guard.go` — Pure function:** No I/O, no side effects. Verified by reading the entire file. `RefineMinScore` correctly falls back to `scoring.BuildRubric(scoring.RubricRefine, cfg).Total()` when `TargetScore==0`.
- **`scoring/rubric.go:BuildRubric`** — Falls back to `DefaultRefineRubric`/`DefaultReviewRubric`/`DefaultSecurityRubric` when `cfg==nil` or the rubric key is absent or has empty dimensions. Override is a full replacement (no partial merge) — documented in function comment. Verified correct for all three paths.
- **`scoring/scoring.go:ParseScore`** — Uses `rubric.Total()` for `MaxScore` and `TargetScore` initialization instead of hardcoded `rubric.MaxScore`. Dynamic total correct for custom rubrics.
- **`scoring/extract.go`** — Regex updated from `/\s*10` to `/\s*(\d+)`. Group index guard updated from `len(m) < 2` to `len(m) < 3`. `m[1]` is the score, `m[2]` is the denominator (captured but not used). Verified no index out-of-bounds.
- **`context.go:132-151`** — Four new methods follow the existing single-responsibility pattern. `CodeReviewScore` and `SecurityReviewScore` delegate to `scoring.ExtractScoreFromFile` which returns 0 on any error — safe default.
- **`schema.go:96-99`** — `Strict bool` is zero-value `false`, tagged `omitempty`. Existing `qode.yaml` files without this field load correctly. Backward compatibility preserved.
- **`help.go:reviewStatus`** — M1 fix applied: `codeMax` and `secMax` computed from `scoring.BuildRubric(RubricReview/RubricSecurity, cfg).Total()`. `reviewItemStatus` accepts `maxScore int` parameter. Verified the displayed maximum now matches any configured rubric.
- **`guard_test.go`** — 10 table rows: no-analysis, unscored, below-default-min, meets-default-min, custom-target-met, custom-target-not-met, start/no-spec, start/spec-present, review-code, review-security. All guard paths covered.
- **`plan_test.go`** — `TestRunPlanSpec_Pass` captures stdout and asserts `"Technical Specification"` is present (L3 fix from prior review). `captureStdout` checks `w.Close()`, `io.Copy()`, and `r.Close()` return values (errcheck lint fix).
- **Template safety** — `pct` and `add` funcmap functions verified safe (no shell execution, no user-controlled input). `TestBuildCodePrompt_PctConstraints` verifies rendered threshold values for a custom rubric. `judge_refine.md.tmpl` uses `{{range $i, $d := .Rubric.Dimensions}}` — renders 0 blocks on empty rubric (safe zero-value behavior).
- **Import graph** — `context → scoring` safe: `scoring` imports only `config`. `workflow → context, config, scoring` acyclic. Confirmed by `go build ./...` clean.
- **`DefaultRubricConfigs` mirror test** — `TestBuildRubric_DefaultReviewSecurityMirrorConfig` in `rubric_test.go` enforces that `config.DefaultRubricConfigs()` and `DefaultReviewRubric` stay in sync. Fragile but explicit.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 1 |
| Low | 3 |
| Nit | 2 |

**Top 3 to fix before merging:**
1. **M1** — Add a hard error in `start.go` for absent `spec.md`. One-line check; prevents `--force` from silently generating a prompt that references a non-existent file, consistent with the spec's stated `--force` semantics.
2. **L1** — Fix `TestRunReview_Force_EmptyDiff_Proceeds`: rename it and decide whether `--force` with empty diff should actually generate a prompt. The current behavior (force+empty-diff = exit 0, no prompt) is subtly inconsistent with how `--force` works in `plan spec` and `start`.
3. **L2** — Add `TestRunPlanSpec_GuardBlocked_Unscored`: the unscored path is a distinct guard condition that has no CLI-level regression protection.

---

## Rating

A score is a shipping recommendation. Score from what you found, not from what you didn't look for.

| Dimension | Score | What you verified (not what you assumed) |
|-----------|-------|------------------------------------------|
| Correctness (0–3) | 2 | Traced all 4 guarded command paths (plan.go:202-216, start.go:48-56, review.go:73-79, guard.go:34-68) against spec table. Guard/force/strict/toFile combinations all correct. `start.go` missing hard error for absent spec — contradicts spec's "absent-file hard errors remain" (M1). `review.go` force+empty-diff exits nil without generating prompt (L1). |
| CLI Contract (0–2) | 2 | STOP. goes to stdout (non-strict), error to stderr (strict). Verified by `captureStdout` pattern in tests. `--force` wired in all 4 commands. `--to-file` unconditionally bypasses guard at lines plan.go:202, start.go:48. `qode workflow status` now uses dynamic rubric totals (M1 fix applied). Slash commands updated with STOP. handling. |
| Go Idioms & Code Quality (0–2) | 2 | New functions all ≤50 lines (guard.go max ~25 lines). Named types (CheckResult, RubricKind). No global mutable state. `fmt.Errorf("%s", ...)` nit is the only idiom deviation (two occurrences, N1). `DefaultRubricConfigs()` in config layer duplicates scoring layer data — fragile but covered by a mirror test. |
| Error Handling & UX (0–2) | 2 | Guard messages include next-action slash command. `git.DiffFromBase` error wrapped with context. Strict vs non-strict correctly uses stderr/stdout. `HasCodeReview` os.Stat permission errors silently return false — acceptable given single-user local use (documented as L3). |
| Test Coverage (0–2) | 1 | CLI white-box tests: 4 (plan), 3 (start), 4 (review) = 11 new tests. `guard_test.go` has 10 table rows covering all guard paths. `TestRunPlanSpec_Pass` now asserts positive output (L3 fix applied). Missing: `TestRunPlanSpec_GuardBlocked_Unscored` (L2). `buildStatusLines` (4+ branching paths) has zero tests (N2). `TestRunStart_Force_SkipsGuard` does not verify absent-spec hard error. |
| Template Safety (0–1) | 1 | No new templates in strict-mode. Guard runs entirely before template engine invoked. `pct` and `add` funcmap functions verified in `TestBuildCodePrompt_PctConstraints`. Template zero-value handling safe (range on empty slice = 0 blocks). `TemplateData.Rubric` and `.MinPassScore` set before rendering — verified field names match template variables. |

**Total Score: 10.0/12**
**Minimum passing score: 10/12**

Constraints:
- A Critical finding voids a high score — total cannot exceed 6.0
- A High finding caps the total at 9.0
- Total ≥ 9.6 requires the justification column to contain specifics, not sentiment ✓
- Total 12.0 requires an explanation of what makes this better than 99% of shipped code
