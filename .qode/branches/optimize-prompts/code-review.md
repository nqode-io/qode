# Code Review — qode (optimize-prompts)

## Reviewer Stance

**Assumptions verified:**
- `ctx.HasRefinedAnalysis()` and `BuildJudgePrompt`'s `os.ReadFile` both target `ctx.ContextDir/refined-analysis.md` — path is identical.
- `branchDir` in `runPlanJudge` equals `ctx.ContextDir` — both use `filepath.Join(root, config.QodeDir, "branches", branch)`.
- `.gitignore` glob `.qode/branches/*/.*.md` excludes only dotfiles — does not affect committed files (`spec.md`, `code-review.md`, `refined-analysis.md`).
- `qode plan judge` is usable regardless of `cfg.Scoring.TwoPass` — behavioral change from before; see Medium issue.

**Earliest silent failure point:** `runPlanJudge` guards on `ctx.HasRefinedAnalysis()` at load time, but the file could be deleted before `BuildJudgePrompt` reads it. Purely theoretical; `BuildJudgePrompt`'s `os.ReadFile` error still surfaces the failure with context.

---

## Issues

**Severity:** Medium
**File:** `internal/plan/refine.go:110-116` (`BuildJudgePrompt`)
**Issue:** `BuildJudgePrompt` no longer checks `cfg.Scoring.TwoPass`. Previously, judge prompts were only generated when `two_pass: true`. Now `qode plan judge` works unconditionally. Projects that explicitly set `two_pass: false` can now run the judge command without any indication that it is intended to be disabled.
**Impact:** Low in practice — the slash command has always included both passes regardless of `TwoPass`, so users were not blocked before. But the config flag's meaning is now inconsistent: it no longer controls whether judge prompts can be generated.
**Suggestion:** Two valid paths: (a) add a guard in `runPlanJudge` — `if !cfg.Scoring.TwoPass { return fmt.Errorf("two-pass scoring is disabled; set scoring.two_pass: true in qode.yaml") }` — OR (b) accept that the flag only gated the old automatic judge generation (now removed) and document that `qode plan judge` is always available. The slash command evidence favours (b). Whichever is chosen, document it.

---

**Severity:** Nit
**File:** `internal/cli/plan.go:120` (`runPlanJudge`)
**Issue:** `branchDir := filepath.Join(root, config.QodeDir, "branches", branch)` is computed but equals `ctx.ContextDir`. Not wrong — consistent with `runPlanSpec` and `runReview`.
**Suggestion:** No change needed; consistency with surrounding functions outweighs the minor redundancy.

---

## File-by-File Evidence

### `internal/plan/refine.go`
1. **Verified safe:** `RefineOutput.JudgePrompt` removed cleanly; no remaining references in file.
2. **Verified safe:** `SaveIterationFiles` return signature reduced from `(workerPath, judgePath string, err error)` to `(workerPath string, err error)`. Only caller (`internal/cli/plan.go:runPlanRefine`) updated correctly.
3. **Verified safe:** `BuildJudgePrompt` reads `refined-analysis.md` from `ctx.ContextDir`, passes content inline to scoring engine. Path is identical to what `HasRefinedAnalysis` checks. `os.ReadFile` error wrapped with context.
4. **Verified safe:** `scoring` import still needed — `BuildJudgePrompt`, `ParseIterationFromOutput`, `SaveIterationResult` all use it.
5. **Verified safe:** `fmt` still needed — `BuildJudgePrompt` error wrap, `buildAnalysisHeader`.

### `internal/cli/plan.go`
1. **Verified safe:** `newPlanJudgeCmd` pattern matches `newPlanSpecCmd` exactly — same `Use`/`Short`/`Long`/`RunE`/`--to-file` structure.
2. **Verified safe:** `runPlanJudge` load sequence: root → cfg → branch → ctx → guard → engine → prompt → output. Same sequence as `runPlanSpec`.
3. **Verified safe:** Guard error message matches `runPlanSpec` pattern — actionable, names the expected file and how to produce it.
4. **Verified safe:** `--to-file` path saves to `branchDir/.refine-judge-prompt.md` via `writePromptToFile` — same helper used by all other `--to-file` commands.
5. **Verified safe:** `runPlanRefine --to-file` single-line stderr message correct after `SaveIterationFiles` signature change.

### `internal/ide/claudecode.go` and `cursor.go`
1. **Verified safe:** Judge pass changed from "run qode plan refine → read file → replace placeholder → execute modified prompt" to "run qode plan judge, use stdout as prompt". Three fewer manual steps; placeholder replacement eliminated.
2. **Verified safe:** Step numbering updated from 8 to 5 steps. Count is correct.
3. **Verified safe:** Both files are symmetric — same intent, same step count.

### `internal/plan/refine_test.go`
1. **Verified safe:** `TestBuildJudgePrompt_InlinesRefinedAnalysis` — creates `refined-analysis.md`, calls `BuildJudgePrompt`, asserts sentinel content appears in output. Correct.
2. **Verified safe:** `TestBuildJudgePrompt_ErrorsIfNoRefinedAnalysis` — no file, asserts error returned. Correct.
3. **Verified safe:** All existing tests unaffected — `RefineOutput.WorkerPrompt` and `Iteration` unchanged; `SaveIterationResult` and `ParseIterationFromOutput` signatures unchanged.

### `.gitignore`
1. **Verified safe:** `.qode/branches/*/.*.md` matches only dotfiles under branch dirs. Committed files (`spec.md`, `code-review.md`, `refined-analysis.md`) are unaffected. All debug prompt files (`.refine-prompt.md`, `.refine-judge-prompt.md`, `.spec-prompt.md`, `.code-review-prompt.md`, `.security-review-prompt.md`) are correctly excluded.

### `README.md` and `docs/how-to-customise-prompts.md`
1. **Verified safe:** `qode plan judge` and `qode plan judge --to-file` added to command reference with correct descriptions and prerequisite noted.
2. **Verified safe:** How-to-customise example shows `--to-file` usage in the correct code block.

---

## Summary

**Issues by severity:**
- Critical: 0
- High: 0
- Medium: 1 (`TwoPass` flag semantics inconsistency in `BuildJudgePrompt`)
- Nit: 1 (`branchDir` redundancy — acceptable as consistent with surrounding code)

**Overall:** The split is implemented cleanly and correctly. The core change — moving judge prompt generation into a dedicated `BuildJudgePrompt` function and `qode plan judge` command — is consistent with the existing builder pattern. The slash command simplification (stdout instead of file-read-replace-execute) is the most significant UX improvement. All tests pass; build is clean.

---

## Rating

| Dimension      | Score (0-2) | What you verified |
|----------------|-------------|------------------------------------------|
| Correctness    | 2           | `HasRefinedAnalysis` and `BuildJudgePrompt` use identical path; `SaveIterationFiles` signature updated at all call sites; `--to-file` saves via `writePromptToFile` helper; slash commands updated symmetrically in both IDE generators |
| Code Quality   | 1           | `TwoPass` behavioral change not addressed or documented; `branchDir` redundancy (acceptable per consistency); test missing negative assertion for judge inlining |
| Architecture   | 2           | Judge cleanly decoupled from worker; builder function signature consistent with spec/start/review builders; slash command reduced from 8 manual steps to 5; `.gitignore` glob correctly scoped |
| Error Handling | 2           | Pre-check guard in `runPlanJudge` gives actionable error message; `BuildJudgePrompt` wraps `os.ReadFile` error with context string |
| Testing        | 2           | Two new tests cover happy path and absent-file error path; existing tests unaffected; build and `go test ./internal/plan/... ./internal/ide/...` pass clean |

**Total Score: 9.0/10**

Constraints check: No Critical or High findings — no score cap applies. Score ≥ 8.0 justified by: Correctness verified by reading all call sites and tracing path construction; Architecture verified by comparing builder function signatures across the package; Error Handling verified by tracing both CLI guard and function-level error wrap. Deduction on Code Quality for the `TwoPass` flag semantic inconsistency (Medium finding) and minor test coverage gap.
