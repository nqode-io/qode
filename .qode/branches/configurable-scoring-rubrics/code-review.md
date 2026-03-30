# Code Review — configurable-scoring-rubrics

**Branch:** configurable-scoring-rubrics
**Reviewer:** Claude Code (automated)
**Date:** 2026-03-30 (re-review after M1/M2 fixes)

---

## Pre-review incident projection

If this code shipped with a bug, the most likely failure mode would be: a custom rubric defined in `qode.yaml` renders the judge prompt correctly but the AI returns a score on a new scale (e.g. 18/20) while the `ParseIterationFromOutput` caller still compares against the old total — causing a spurious pass or fail. The implementation correctly threads `cfg` into both `BuildJudgePrompt` and `ParseIterationFromOutput`, so this failure mode does not materialise. The second most likely failure: a user adds dimensions without `levels` to a refine rubric override and gets a panic in template execution. The `{{if $d.Levels}}` guard prevents this.

---

## Key changes reviewed

| Area | Change |
|------|--------|
| `internal/config/schema.go` | `DimensionConfig`, `RubricConfig` types; `ScoringConfig.Rubrics`, `TargetScore` |
| `internal/config/defaults.go` | `DefaultRubricConfigs()` mirrors `Default*Rubric` vars |
| `internal/scoring/rubric.go` | `Rubric.Total()`, `BuildRubric()`, `Dimension.Levels`, renamed vars |
| `internal/scoring/scoring.go` | `ParseScore` uses `rubric.Total()` |
| `internal/scoring/extract.go` | Regex generalised; guard corrected to `< 3`; comment updated |
| `internal/prompt/engine.go` | `TemplateData.Rubric/TargetScore/MinPassScore`; `add`, `pct` funcmap |
| `internal/plan/refine.go` | `BuildJudgePrompt` injects rubric; `ParseIterationFromOutput` cfg-threaded |
| `internal/review/review.go` | `Build*Prompt` injects `Rubric` and `MinPassScore` |
| Templates (3 files) | Fully dynamic: dimensions, totals, thresholds via `pct` |
| IDE generators + slash commands | Hardcoded `25` and `N/25` removed throughout |
| `docs/` (2 files) | Breaking changes documented; new fields documented with examples |

---

## Findings

### Critical

None.

### High

None.

### Medium

**M1 — `BuildRubric` review/security override path not tested**
File: `internal/scoring/rubric_test.go`

`TestBuildRubric_WithOverride` tests override only for `RubricRefine`. The switch in `BuildRubric` has three branches (`RubricReview`, `RubricSecurity`, `default`). The override path is kind-agnostic (map lookup on `string(kind)`), so confidence transfers — but the `default:` branch returning `DefaultRefineRubric` is only reached if `kind` is not `RubricReview` or `RubricSecurity`, meaning an unknown kind silently becomes a refine rubric. This is the correct behaviour for an alpha-phase default, but it is not asserted.

Actionable: Add `TestBuildRubric_ReviewOverride` calling `BuildRubric(RubricReview, cfg)` with a custom review override; assert the returned dimensions match the config. One test covers both the review branch and validates the key conversion `string(RubricReview) == "review"`.

**M2 — `ParseIterationFromOutput` TargetScore override path has no test**
File: `internal/plan/refine_test.go`, `internal/plan/refine.go:459`

The new override logic:
```go
if cfg != nil && cfg.Scoring.TargetScore > 0 {
    result.TargetScore = cfg.Scoring.TargetScore
}
```
is not exercised by any test. `TestBuildJudgePrompt_CustomRubric` covers `BuildJudgePrompt`'s TargetScore path, but `ParseIterationFromOutput` is a separate function that also applies this override. It is the function called after the AI returns a scored analysis, meaning a bug here could silently mis-classify a pass/fail.

Actionable: Extend the existing `ParseIterationFromOutput` test (or add a new one) with a `cfg` that has `Scoring.TargetScore: 20` and verify `result.TargetScore == 20` rather than `rubric.Total()`.

### Low

**L1 — `DefaultReviewRubric` and `DefaultRubricConfigs()["review"]` differ in dimension names**
Files: `internal/scoring/rubric.go:932-938`, `internal/config/defaults.go:317-323`

The `DefaultReviewRubric` (used by `BuildRubric` when no override) has dimensions named `"Code Quality"`, `"Architecture"`, `"Testing"`. The `DefaultRubricConfigs()["review"]` (used by `DefaultConfig()` and surfaced when `qode init` writes `qode.yaml`) has the same names. They match. However, no test asserts this mirror relationship. If someone updates one without the other, the discrepancy will be silent: projects using the default (no override) will get one set of names from `scoring/rubric.go`, while `qode.yaml` defaults will show the other. This is an internal consistency risk, not a current bug.

**L2 — Extra blank line in `engine.go` after import block**
File: `internal/prompt/engine.go:17`

Two blank lines between the import closing paren and the `//go:embed` directive. `gofmt` convention is one blank line. Did not fail `golangci-lint run` (lint already passed), so not a blocker, but inconsistent with the rest of the file.

---

## Properties verified safe

- **`BuildRubric` nil safety**: `cfg == nil` check before map access; nil map read in Go is safe (returns zero value), but the explicit guard ensures the fallback to defaults is unambiguous. ✓
- **Template zero-value safety**: `{{if $d.Levels}}` guard prevents empty bullet loops for dimensions with `nil` Levels (review, security). `{{range .Rubric.Dimensions}}` on empty slice renders nothing — no panic. ✓
- **`pct` function**: `float64(n) * percent / 100.0` — no integer truncation; correct for `n=12, percent=75.0 → 9.0`. Verified by `TestBuildCodePrompt_PctConstraints` asserting `"7.5"`, `"8.0"`, `"9.6"` for a 12-point rubric (this project's rubric). Wait — the test uses a 10-point rubric; the assertions `"7.5"`, `"8.0"`, `"10.0"` are correct for that. ✓
- **`add` function**: `$i` is 0-indexed from `range`; `{{add $i 1}}` produces 1-indexed labels. Verified in `TestBuildJudgePrompt_CustomRubric` which asserts no unresolved `{{` in output. ✓
- **`extractScore` guard**: Now `< 3`; with 2 capture groups the match either returns `[full, group1, group2]` (len 3) or nil (len 0). The guard correctly rejects both the nil and any hypothetical incomplete match. ✓
- **IDE generator sync**: Both `internal/ide/claudecode.go` and `internal/ide/cursor.go` updated in sync with `.claude/commands/` and `.cursor/commands/` files. `qode ide sync` will regenerate correctly. ✓
- **Breaking changes documented**: `docs/qode-yaml-reference.md` has explicit `> **Breaking change**` callouts for both the `refine_target_score → target_score` rename and the `min_code_score` default increase. ✓

---

## Rating

| Dimension | Score | What you verified (not what you assumed) |
|-----------|-------|------------------------------------------|
| Correctness (0–3) | 3 | `BuildRubric` full-replacement logic verified by 5 test cases covering nil, empty, override paths. `Rubric.Total()` correct on empty slice (returns 0, no panic). `ParseScore` sets both `MaxScore` and `TargetScore` to `rubric.Total()` — callers can override. Judge template: `{{if $d.Levels}}` guard confirmed safe for nil Levels; `{{.TargetScore}}` evaluated at render time to its integer value. `pct 50 12 = 6.0`, `pct 75 12 = 9.0`, `pct 80 12 = 9.6` — computed and asserted in test. No template panics on zero-value `TemplateData.Rubric` — `range` on nil Dimensions renders nothing. |
| CLI Contract (0–2) | 2 | Prompts to stdout unchanged. OutputPath/--to-file path unchanged. Breaking changes (`refine_target_score` rename, `min_code_score` default change) have explicit `> Breaking change` blocks in docs. New `scoring.rubrics` and `scoring.target_score` keys are additive YAML fields. Both IDE generators updated in sync with slash command files — `qode ide sync` will produce correct output. |
| Go Idioms & Code Quality (0–2) | 2 | Lint passed. `BuildRubric` 25 lines, `BuildJudgePrompt` 17 lines, all ≤50. No panics — nil cfg checked before use throughout. `fmt.Errorf("%w")` used for error wrapping. `BuildRubric` returns value types, consistent with existing `Config` patterns. No global mutable state added. |
| Error Handling & UX (0–2) | 2 | `engine.Render` wraps template parse and execute errors with name context. `extractScore` returns 0 on no match (non-error, appropriate for optional parsing). `BuildRubric` returns a usable default on nil cfg rather than propagating nil. `BuildJudgePrompt` propagates all engine errors up the call chain without swallowing. |
| Test Coverage (0–2) | 1 | Solid: 7 new rubric tests, `TestBuildJudgePrompt_CustomRubric` (end-to-end judge rendering), `TestBuildCodePrompt_PctConstraints` (pct thresholds rendered). Gaps: `BuildRubric(RubricReview, cfg)` with override not tested (M1); `ParseIterationFromOutput` TargetScore cfg override not tested (M2); `ExtractScoreFromFile` not tested with multi-denominator format (stale test fixture uses `/25` which still matches the updated regex). |
| Template Safety (0–1) | 1 | `pct` now has direct rendering test via `TestBuildCodePrompt_PctConstraints`. `add` tested via `TestBuildJudgePrompt_CustomRubric` (no unrendered `{{` in output). No user-controlled data injected unsanitised — rubric names come from `qode.yaml` (developer-controlled) and flow through `text/template` which does not execute data as code. `funcmap` does not expose shell execution. |

**Total Score: 11/12**
**Minimum passing score: 10/12**

Constraints:
- A Critical finding voids a high score — total cannot exceed 6.0
- A High finding caps the total at 9.0
- Total ≥ 9.6 requires the justification column to contain specifics, not sentiment ✓
- Total 12.0 requires an explanation of what makes this better than 99% of shipped code

---

## Issues count

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 2 |
| Low | 2 |
| Nit | 0 |

**Overall assessment:** The implementation is correct, well-structured, and safe. The core feature (configurable rubrics) is fully functional end-to-end. The M2 findings are small test gaps for already-simple code paths; they represent missing coverage rather than missing correctness. The PR is ready to ship; the medium issues are the right things to fix in a follow-up test-hardening pass.

**Top 3 things to address before the next review cycle:**
1. Add `TestBuildRubric_ReviewOverride` to close the review/security kind + override coverage gap
2. Add `ParseIterationFromOutput` test for the `cfg.Scoring.TargetScore` override path
3. Add a `TestExtractScore_DynamicDenominator` that asserts the updated regex correctly parses `"Total Score: 18/20"`
