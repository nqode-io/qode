<!-- qode:iteration=4 score=25/25 -->

# Requirements Analysis: Configurable Scoring Rubrics (Iteration 4)

## 1. Problem Understanding

The three scoring rubrics (`RefineRubric`, `ReviewRubric`, `SecurityRubric`) in `internal/scoring/rubric.go` are Go vars with hardcoded dimensions, weights, and a `MaxScore` field. Every qode project receives identical rubrics regardless of domain. There is no way to customise dimensions, weights, or descriptions without forking.

Two additional issues compound this: (1) `judge_refine.md.tmpl` hardcodes per-level score descriptions per dimension (e.g. "5: Perfect restatement...", "4: Good understanding...") that cannot be overridden; (2) `review/code.md.tmpl` lists "6. Performance" as a review criterion, but `ReviewRubric` has no Performance dimension — the rating table and the criteria are inconsistent.

**Resolved design decisions (from notes.md):**
- Target score is configurable via `scoring.target_score` in `qode.yaml` (not always derived from rubric max)
- Performance is a dimension in the default review rubric (weight 2), fixing the criterion/rubric mismatch
- Default rubric configs in `internal/config/defaults.go` must reflect the current hardcoded state of `internal/scoring/rubric.go` exactly, including `Levels` for refine dimensions

**User need:** Teams can define custom rubric dimensions, weights, descriptions, and per-score level descriptions in `qode.yaml` under `scoring.rubrics`, with totals computed dynamically.

**Business value:** qode is adoptable by projects with domain-specific quality standards without forking.

**Open questions:** None.

---

## 2. Technical Analysis

### Affected components

**`internal/config/schema.go`** — add types:
- `DimensionConfig{Name string, Weight int, Description string, Levels []string}` — `Levels` stores full label strings including score prefix, highest first (e.g. `"5: Perfect restatement, all ambiguities resolved"`); only meaningful for refine dimensions
- `RubricConfig{Dimensions []DimensionConfig}`
- `ScoringConfig`: add `Rubrics map[string]RubricConfig` (yaml: `rubrics,omitempty`); rename `RefineTargetScore int` → `TargetScore int` (yaml: `target_score,omitempty`)

**`internal/config/defaults.go`** — populate default rubric configs:
- Add `DefaultRubricConfigs() map[string]RubricConfig` returning all three rubric configs:
  - `"refine"`: 5 dimensions matching current `RefineRubric` exactly, each with `Levels []string` containing the full labelled strings from `judge_refine.md.tmpl` verbatim (e.g. `[]string{"5: Perfect restatement, all ambiguities resolved, user need crystal clear", "4: Good understanding, minor gaps", "3: Adequate but surface-level", "2: Partial understanding, significant gaps", "1: Mostly incorrect or too vague", "0: Missing or completely wrong"}`)
  - `"review"`: 5 existing dimensions + Performance (`Weight: 2`, `Description: "No obvious performance issues, unnecessary allocations, or blocking calls"`) — total 12
  - `"security"`: 5 dimensions matching current `SecurityRubric` exactly — total 10
- Update `DefaultConfig()`: set `Scoring.Rubrics = DefaultRubricConfigs()`; remove `Scoring.RefineTargetScore: 25` (zero-value of renamed `TargetScore` means "use rubric max"); change `Review.MinCodeScore: 8.0` → `10.0` (≈83% of new review max 12)
- Update `internal/config/config_test.go`: verify `Scoring.Rubrics` is non-nil with 3 keys; verify `Review.MinCodeScore == 10.0`

**`internal/scoring/rubric.go`** — make rubrics configurable:
- Rename vars: `RefineRubric → DefaultRefineRubric`, `ReviewRubric → DefaultReviewRubric`, `SecurityRubric → DefaultSecurityRubric`
- Add `Levels []string` to `Dimension` struct — stores full labelled strings (e.g. `"5: Perfect restatement..."`)
- Populate `DefaultRefineRubric` `Dimension.Levels` with verbatim labelled strings from `judge_refine.md.tmpl` (6 entries per dimension, highest first)
- Add Performance `Dimension{Name: "Performance", Weight: 2, Desc: "No obvious performance issues, unnecessary allocations, or blocking calls"}` to `DefaultReviewRubric` — do NOT set `MaxScore` on the var; `Total()` computes 12 from the 6 dimensions
- Remove `MaxScore int` from `Rubric` struct; add `func (r Rubric) Total() int` (zero-arg value receiver, callable as `{{.Rubric.Total}}` in Go templates)
- Add `func BuildRubric(kind RubricKind, cfg *config.Config) Rubric`: map lookup on `cfg.Scoring.Rubrics` (nil map read is safe in Go); if key present and `len(dims) > 0`, convert `[]DimensionConfig` → `[]Dimension`; otherwise return the corresponding `Default*Rubric`

**`internal/scoring/scoring.go`** — update score parsing:
- `ParseScore`: replace `rubric.MaxScore` with `rubric.Total()` for `Result.MaxScore`; set `result.TargetScore = rubric.Total()` as default (callers override with `cfg.Scoring.TargetScore` if non-zero, after calling `ParseScore`)
- `Result.Ready` computation unchanged (`TotalScore >= TargetScore`)

**`internal/scoring/extract.go`** — generalise regex:
- `totalScoreRe`: change `/10` → `/(\d+)` making denominator a captured group; `extractScore` returns numerator unchanged; existing `"Total Score: 8.5/10"` fixtures still match — verify with `go test ./internal/scoring/...`

**`internal/prompt/engine.go`** — extend template infrastructure:
- Add `"add": func(a, b int) int { return a + b }` to `e.funcMap` (no naming collision — `add` is not a built-in Go template function; existing funcmap only has `join`)
- Add `Rubric scoring.Rubric` field to `TemplateData`

**`internal/plan/refine.go`** — inject rubric and resolve cfg-threading:
- `BuildJudgePrompt(engine, cfg, ctx)`: add `data.Rubric = scoring.BuildRubric(scoring.RubricRefine, cfg)` before rendering
- `ParseIterationFromOutput`: add `cfg *config.Config` parameter; replace `scoring.RefineRubric` with `scoring.BuildRubric(scoring.RubricRefine, cfg)`; replace `result.TargetScore = 25` with `if cfg.Scoring.TargetScore > 0 { result.TargetScore = cfg.Scoring.TargetScore }`. Function has zero callers in production code (grep-confirmed) — signature change is safe

**`internal/review/review.go`** — inject rubric:
- `BuildCodePrompt`: add `data.Rubric = scoring.BuildRubric(scoring.RubricReview, cfg)`
- `BuildSecurityPrompt`: add `data.Rubric = scoring.BuildRubric(scoring.RubricSecurity, cfg)`

**`internal/prompt/templates/scoring/judge_refine.md.tmpl`** — render dynamically. The template source uses `{{"{{"}}` and `{{"}}"}}` as escape sequences that render literal `{{` and `}}` in the output (the judge receives them as pseudo-code instructions, not executed Go template syntax). Three hardcoded values to replace:

1. **Header line**: replace `(5 dimensions × 5 points = 25 maximum)` with `({{len .Rubric.Dimensions}} dimensions = {{.Rubric.Total}} maximum)` — evaluated at render time

2. **Dimension sections**: replace all 5 hardcoded blocks with a `{{range $i, $d := .Rubric.Dimensions}}` loop:
   ```
   {{range $i, $d := .Rubric.Dimensions}}
   ### Dimension {{add $i 1}}: {{$d.Name}} (0-{{$d.Weight}})
   {{if $d.Levels}}{{range $d.Levels}}- {{.}}
   {{end}}{{end}}
   **Score:** /{{$d.Weight}}
   **Justification:**

   {{end}}
   ```
   `Levels` entries are full labelled strings, so `- {{.}}` renders them verbatim with no arithmetic.

3. **Summary section** — `**Total Score:** /25` → `**Total Score:** /{{.Rubric.Total}}` — evaluated at render time, embeds the integer (e.g. `25`) into the output

4. **Recommendation line** — currently `{{"{{"}}if ge .Score 25{{"}}"}}`: change the hardcoded `25` to use Go template evaluation at render time by writing `{{"{{"}}if ge .Score {{.Rubric.Total}}{{"}}"}}` in the source template. At render time, `{{.Rubric.Total}}` is evaluated to its integer value (e.g. `25`) and the output is `{{if ge .Score 25}}` — a dynamically embedded number, not the string `.Rubric.Total`

**`internal/prompt/templates/review/code.md.tmpl`** — render rating table dynamically:
- Change table header from `| Dimension | Score (0-2) | What you verified... |` to `| Dimension | Score | What you verified... |`
- Replace 5 hardcoded body rows with `{{range .Rubric.Dimensions}}| {{.Name}} (0–{{.Weight}}) |             |                                          |{{"\n"}}{{end}}`
- Embedding `(0–{{.Weight}})` in the dimension name makes per-dimension max visible for variable-weight custom rubrics without requiring a dynamic table header

**`internal/prompt/templates/review/security.md.tmpl`** — same pattern:
- Header: `| Dimension | Score | Control or finding... |`
- Body: `{{range .Rubric.Dimensions}}| {{.Name}} (0–{{.Weight}}) | | |{{"\n"}}{{end}}`

**`internal/cli/plan.go`** and **`internal/cli/review.go`** — no direct rubric references; changes absorbed by `plan` and `review` packages.

**`docs/qode-yaml-reference.md`** — document new config fields.

### Key technical decisions

1. **`Levels` stores full labelled strings**: e.g. `"5: Perfect restatement, all ambiguities resolved"`. Template renders with `{{range $d.Levels}}- {{.}}{{"\n"}}{{end}}` — no arithmetic helpers needed. `add` funcmap is used only for 1-indexed dimension numbering.

2. **Rating table column header `Score` + per-row weight in dimension name**: `| Correctness (0–2) | | |` is the row format. Avoids dynamic header (which Go templates cannot change mid-table) while correctly showing per-dimension max for variable-weight rubrics.

3. **`TargetScore` replaces `RefineTargetScore`**: yaml key `target_score`, zero-value = use rubric max. Breaking change for any existing `refine_target_score` users; document in release notes.

4. **Performance in `DefaultReviewRubric`**: sixth dimension, total 12. `MinCodeScore` default → 10.0 (83%). Existing users with explicit `min_code_score` in their `qode.yaml` are unaffected.

5. **`BuildRubric` full-replacement**: no partial merge.

6. **`ParseScore` TargetScore override pattern**: `ParseScore` sets default from `rubric.Total()`; callers apply `cfg.Scoring.TargetScore` override post-call. Avoids threading `cfg` into `ParseScore`.

7. **`Rubric.MaxScore` removed cleanly**: `DefaultReviewRubric` does NOT set `MaxScore: 12` at declaration — the field is removed in the same commit. `Total()` computes 12 from the 6 dimensions automatically.

8. **Template escaping for dynamic Recommendation threshold**: In the `judge_refine.md.tmpl` source, `{{"{{"}}if ge .Score {{.Rubric.Total}}{{"}}"}}` causes `{{.Rubric.Total}}` to be evaluated to an integer at render time (e.g. `25`), producing the output `{{if ge .Score 25}}`. This is distinct from `{{"{{"}}if ge .Score .Rubric.Total{{"}}"}}` which would render the literal string `.Rubric.Total` in the output.

### Patterns to follow

- Config types in `schema.go`, defaults in `defaults.go`
- `Build*` functions in domain packages construct `TemplateData` and call engine
- Templates use `{{range .Rubric.Dimensions}}` — consistent with `{{range .Layers}}`
- `RubricKind` string constants — existing pattern

### Dependencies

- Depends on #24 (hardened review prompts). Template changes here are additive and do not conflict.

---

## 3. Risk & Edge Cases

**Risk 1 — `Rubric.MaxScore` field removal**: Accessed in `scoring.go:72` as `Result{MaxScore: rubric.MaxScore, TargetScore: rubric.MaxScore}` — both inside `scoring` package. Add `Total()`, update those two struct field initialisations, remove the field. Compiler catches all remaining callsites.

**Risk 2 — Empty `Levels` in judge template**: `{{if $d.Levels}}` guard prevents rendering empty bullets. Without the guard, `{{range nil}}` is a no-op in Go templates — guard is for clarity.

**Risk 3 — `MinCodeScore` default change (8.0 → 10.0) is a behaviour change**: Users with no explicit `min_code_score` see the threshold rise from 8/10 to 10/12 after upgrade. Document as a breaking change in release notes. Users with explicit `min_code_score` in `qode.yaml` are unaffected.

**Risk 4 — `TargetScore` YAML key rename**: Users who set `refine_target_score` in `qode.yaml` silently lose that value after upgrade (unmapped key is ignored by yaml.Unmarshal). Document in release notes. No migration path in V1.

**Risk 5 — Users who customise rubric total must manually update score thresholds**: If a team changes their review rubric to 8 dimensions × 2 = 16 points, `min_code_score: 10.0` is no longer proportionally meaningful (62.5% vs intended 83%). The config does not auto-scale. Document in `qode-yaml-reference.md`: "When changing rubric dimensions, update `min_code_score` and `min_security_score` accordingly."

**Risk 6 — `ParseScore` dimension-score pairing fragility (pre-existing)**: Index-based matching is fragile with variable rubrics. Pre-existing debt; document, do not fix in this PR.

**Risk 7 — `ParseIterationFromOutput` signature change**: Exported, zero callers in production, no test coverage. Add `cfg *config.Config` parameter. Safe.

**Risk 8 — `add` funcmap naming collision**: No collision — `add` is not a built-in Go template identifier. Existing funcmap has only `join`. Verified against `engine.go:28-38`.

**Risk 9 — `extract_test.go` fixture compatibility**: Generalised regex `(\d+(?:\.\d+)?)\s*/\s*(\d+)` matches `8.5/10` — denominator captured to group 2, ignored. Tests pass without fixture changes; confirm with `go test ./internal/scoring/...`.

**Risk 10 — Template syntax error in `judge_refine.md.tmpl` after changes**: The range loop, `add` call, and escaped-brace patterns are easy to mis-nest. Mitigated by a template rendering test in Commit 8 that exercises the full judge prompt with a real `TemplateData.Rubric`.

**Security**: YAML config from local filesystem only. No injection vectors.

**Performance**: `BuildRubric` and `Rubric.Total()` called once per command invocation. Negligible.

---

## 4. Completeness Check

### Acceptance criteria

1. `qode.yaml` accepts `scoring.rubrics.{refine,review,security}` with `dimensions[].{name,weight,description,levels}`
2. `scoring.target_score` overrides refine pass threshold; defaults to `rubric.Total()` when absent or zero
3. Projects without `rubrics` in `qode.yaml` behave identically to current behaviour
4. `Rubric.Total()` returns `sum(dimension.Weight)` for any rubric
5. `BuildRubric(kind, cfg)` returns config-defined rubric when present, defaults otherwise
6. `TemplateData.Rubric` is populated for judge-refine, code-review, and security-review prompts
7. `judge_refine.md.tmpl` renders all dimensions, per-level descriptions, Summary total, and Recommendation threshold from `Rubric` data — no hardcoded numeric values remain
8. `review/code.md.tmpl` rating table rows rendered from `Rubric.Dimensions`; each row shows `(0–N)` max in dimension column
9. `review/security.md.tmpl` rating table rows rendered from `Rubric.Dimensions`; same per-row max format
10. `DefaultReviewRubric` includes Performance dimension (weight 2); `Rubric.Total() == 12`; default `MinCodeScore == 10.0`
11. `DefaultRefineRubric` `Dimension.Levels` contain verbatim labelled strings from `judge_refine.md.tmpl`
12. `ExtractScoreFromFile` regex matches any `N/M` denominator
13. All existing tests pass; new tests cover `BuildRubric`, `Rubric.Total()`, default rubric structure, and template rendering with a custom rubric
14. `docs/qode-yaml-reference.md` updated with `scoring.rubrics`, `scoring.target_score`, and manual threshold update guidance

### Implicit requirements

- `Levels` entries are full labelled strings (e.g. `"5: Perfect restatement..."`); templates render with `- {{.}}` — no helper arithmetic required
- `DefaultConfig().Scoring.TargetScore == 0` (zero-value); callers interpret 0 as "use rubric max"
- `parseIterationFromAnalysis` in `context.go` already scans `score=%d/%d`; no change needed
- `review/code.md.tmpl` "6. Performance" review criteria section stays; rating table now aligns with it via rubric injection
- `ScoringConfig.RefineTargetScore` removed entirely; yaml key `refine_target_score` silently ignored after upgrade
- Users who change a rubric's total via custom dimensions must manually update `min_code_score` / `min_security_score`; the config does not auto-scale these values
- In the `judge_refine.md.tmpl` source, `{{.Rubric.Total}}` appears outside escaped braces (evaluated at render time) wherever a literal integer is needed in the output; it appears inside `{{"{{"}}...{{"}}"}}` escape sequences only as part of pseudo-code rendered to the AI judge

### Explicitly out of scope

- Per-layer rubrics
- Rubric inheritance / partial override
- External rubric YAML files
- Per-branch rubric overrides
- Auto-scaling `MinCodeScore`/`MinSecurityScore` when rubric max changes

---

## 5. Actionable Implementation Plan

**Commit 1: Add rubric config types and rename TargetScore — `internal/config/schema.go`**
- Add `DimensionConfig{Name string, Weight int, Description string, Levels []string}`
- Add `RubricConfig{Dimensions []DimensionConfig}`
- Add `Rubrics map[string]RubricConfig` to `ScoringConfig` (yaml: `rubrics,omitempty`)
- Rename field `RefineTargetScore int` → `TargetScore int` (yaml: `target_score,omitempty`)

**Commit 2: Populate default rubric configs — `internal/config/defaults.go`**
- Add `DefaultRubricConfigs() map[string]RubricConfig`:
  - `"refine"`: 5 dims with `Name`, `Weight`, `Description`, and `Levels []string` — each entry a full labelled string taken verbatim from `judge_refine.md.tmpl`
  - `"review"`: `Correctness`, `Code Quality`, `Architecture`, `Error Handling`, `Testing` (weight 2 each) + `Performance` (weight 2, desc `"No obvious performance issues, unnecessary allocations, or blocking calls"`)
  - `"security"`: `Injection Prevention`, `Auth & AuthZ`, `Data Exposure`, `Input Validation`, `Dependency Safety` (weight 2 each)
- Update `DefaultConfig()`: set `Scoring.Rubrics`, remove `Scoring.RefineTargetScore: 25`, change `Review.MinCodeScore: 8.0` → `10.0`
- Update `config_test.go`: assert `len(cfg.Scoring.Rubrics) == 3`, `cfg.Review.MinCodeScore == 10.0`

**Commit 3: Update `internal/scoring/rubric.go`**
- Rename vars to `Default*Rubric`
- Add `Levels []string` to `Dimension` struct
- Populate `DefaultRefineRubric.Dimensions[*].Levels` with labelled strings from `judge_refine.md.tmpl`
- Append `Dimension{Name: "Performance", Weight: 2, Desc: "..."}` to `DefaultReviewRubric.Dimensions`
- Remove `MaxScore int` from `Rubric` struct (no interim value — `Total()` replaces it entirely in the same commit)
- Add `func (r Rubric) Total() int { total := 0; for _, d := range r.Dimensions { total += d.Weight }; return total }`
- Add `func BuildRubric(kind RubricKind, cfg *config.Config) Rubric` — nil-safe map lookup, full-replacement

**Commit 4: Update scoring — `internal/scoring/scoring.go` and `internal/scoring/extract.go`**
- `ParseScore`: replace `rubric.MaxScore` with `rubric.Total()` in `Result` initialisation; `result.TargetScore = rubric.Total()`
- `totalScoreRe`: replace literal `/10` with `/(\d+)` (two capture groups); `extractScore` uses group 1 only
- Run `go test ./internal/scoring/...` to confirm no regressions

**Commit 5: Extend `TemplateData` and inject rubric — `engine.go`, `refine.go`, `review.go`**
- `internal/prompt/engine.go`: add `"add": func(a, b int) int { return a + b }` to `e.funcMap`; add `Rubric scoring.Rubric` to `TemplateData`
- `internal/plan/refine.go` `BuildJudgePrompt`: `data.Rubric = scoring.BuildRubric(scoring.RubricRefine, cfg)`
- `internal/plan/refine.go` `ParseIterationFromOutput`: add `cfg *config.Config`; use `BuildRubric`; replace `result.TargetScore = 25` with cfg-driven override
- `internal/review/review.go` `BuildCodePrompt`: `data.Rubric = scoring.BuildRubric(scoring.RubricReview, cfg)`
- `internal/review/review.go` `BuildSecurityPrompt`: `data.Rubric = scoring.BuildRubric(scoring.RubricSecurity, cfg)`

**Commit 6: Update `judge_refine.md.tmpl`** — four changes, all in one file:
1. Header: `({{len .Rubric.Dimensions}} dimensions = {{.Rubric.Total}} maximum)`
2. Dimension blocks: replace 5 hardcoded sections with `{{range $i, $d := .Rubric.Dimensions}}` loop rendering name, weight, `Levels` via `- {{.}}`, and `**Score:** /{{$d.Weight}}`
3. Summary: `**Total Score:** /{{.Rubric.Total}}` (evaluated at render time — `.Rubric.Total` is outside any escaped-brace sequence)
4. Recommendation: `{{"{{"}}if ge .Score {{.Rubric.Total}}{{"}}"}}` — `.Rubric.Total` is inside the outer template action and is evaluated to its integer value before the `{{"{{"}}` and `{{"}}"}}` wrap it, producing e.g. `{{if ge .Score 25}}` in the output

**Commit 7: Update `review/code.md.tmpl` and `review/security.md.tmpl`**
- `code.md.tmpl`: header `| Dimension | Score | What you verified (not what you assumed) |`; body `{{range .Rubric.Dimensions}}| {{.Name}} (0–{{.Weight}}) |             |                                          |{{"\n"}}{{end}}`
- `security.md.tmpl`: header `| Dimension | Score | Control or finding that determines this score |`; body `{{range .Rubric.Dimensions}}| {{.Name}} (0–{{.Weight}}) | | |{{"\n"}}{{end}}`

**Commit 8: Tests — `internal/scoring/rubric_test.go` (new), `internal/plan/refine_test.go`, `internal/prompt/engine_test.go`**
- `TestRubricTotal`: custom rubric with known weights sums correctly
- `TestBuildRubric_NoOverride`: nil-rubrics config returns `DefaultRefineRubric`
- `TestBuildRubric_WithOverride`: config with one dimension returns that single-dimension rubric
- `TestDefaultReviewRubricHasPerformance`: `DefaultReviewRubric.Dimensions[5].Name == "Performance"`
- `TestDefaultRefineRubricLevels`: first dimension has 6 `Levels` entries; `Levels[0]` starts with `"5:"`
- `refine_test.go`: judge prompt rendered with custom rubric config contains custom dimension name (not `"Problem Understanding"`)
- `engine_test.go` (or inline in `refine_test.go`): render `judge_refine.md.tmpl` with a 2-dimension `TemplateData.Rubric`; assert output contains both dimension names, no `{{` literal remains from unresolved template actions, and total `= 10 maximum` appears (2 dims × 5 = 10)

**Commit 9: `docs/qode-yaml-reference.md`**
- Document `scoring.target_score` (type int, default = rubric max, zero means use rubric max)
- Document `scoring.rubrics` with complete YAML example for all three rubric types, including `levels` on refine dimensions
- Note `refine_target_score` → `target_score` rename as breaking change
- Note `min_code_score` default change from 8.0 to 10.0 as breaking change
- Add guidance: "When changing rubric dimensions or weights, update `min_code_score`/`min_security_score` manually to maintain your intended pass threshold"

**Order:** 1→2→3→4→5→6→7→8→9. Commits 6 and 7 can be parallelised after Commit 5. Commit 8 test stubs can be drafted alongside Commits 3–5. Commit 9 is always last.
