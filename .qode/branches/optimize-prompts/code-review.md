# Code Review — qode (optimize-prompts)

## Reviewer Stance

**What this code assumes:**
- `ctx.ContextDir` is always an absolute path — confirmed: `context.Load` uses `filepath.Join(root, ...)` where `root` comes from `filepath.Abs(os.Getwd())`.
- `branchDir` exists when `diff.md` is written in `cli/review.go` — confirmed: `context.Load` calls `os.MkdirAll(<branchDir>/context/, 0755)` which creates `<branchDir>` as a side effect.
- `branches[0]` is always set in `buildBranchLessonData` — confirmed: guarded by `cobra.MinimumNArgs(1)`.
- `kind` in `runReview` is always "code" or "security" — hardcoded at both call sites.

**Earliest silent failure point:** If `BranchDir` is empty in a rendered template, file-path references become bare filenames. All callers set `BranchDir: ctx.ContextDir`, and `ctx.ContextDir` is always set by `context.Load`. Not a current risk, but relevant for future external callers.

---

## Issues

**Severity:** Low
**File:** `internal/plan/refine_test.go:140-165`
**Issue:** `TestBuildRefinePromptWithOutput_OmitsAnalysisAndTicket` only asserts the negative (no inline content) but does not assert the positive (that file-path references to `ticket.md` and `refined-analysis.md` appear). Compare to `TestBuildSpecPromptWithOutput_OmitsAnalysis` (checks both sentinel absence and `refined-analysis.md` presence) and `TestBuildCodePrompt_OmitsDiffAndSpec` (checks both `spec.md` and `diff.md` references).
**Suggestion:** Add two assertions:
```go
if !strings.Contains(out.WorkerPrompt, "ticket.md") {
    t.Error("prompt must reference ticket.md")
}
if !strings.Contains(out.WorkerPrompt, "refined-analysis.md") {
    t.Error("prompt must reference refined-analysis.md")
}
```

---

**Severity:** Nit
**File:** `internal/cli/knowledge_cmd.go:238-240`
**Issue:** Alignment inconsistency in struct literal introduced when `BranchDir` and `Lessons` were added: `Lessons:   lessonsStr` uses 3 spaces after the colon while `Analysis:    allAnalysis.String()` uses 4 spaces.
**Suggestion:** Align consistently — `Lessons:     lessonsStr,` / `BranchDir:   branchDir,`.

---

## File-by-File Evidence

### `internal/prompt/engine.go`
1. **Verified safe:** `TemplateData` fields correctly documented with inline comments naming callers. `BranchDir` added with clear description. `ContextMode` constants and field cleanly removed — no dead code left.
2. **Verified safe:** `Notes string` field removed. Notes are referenced via template file-read instruction, not inlined. No template references `{{.Notes}}`.
3. **Verified safe:** Remaining content fields (`Ticket`, `Analysis`, `Spec`, `Diff`) retained correctly — still used by `knowledge/add-branch.md.tmpl` and `scoring/judge_refine.md.tmpl`.

### `internal/context/context.go`
1. **Verified safe:** `WarnMissingPredecessors` uses `_, _ = fmt.Fprintln(w, ...)` — satisfies errcheck linter. Warning text for each case is specific and actionable.
2. **Verified safe:** All three cases match exactly where the method is called: `"spec"` in `runPlanSpec`, `"start"` in `runStart`, `"review"` in `runReview`.
3. **Verified safe:** `notes.md` excluded from Extra scan (lines 61-62) — correct, notes are referenced by path in templates, not aggregated.

### `internal/plan/refine.go`
1. **Verified safe:** `BuildRefinePromptWithOutput` omits Ticket and Analysis — sets only `Extra`, `Branch`, `OutputPath`, `BranchDir`. Correct for reference mode.
2. **Verified safe:** `BuildSpecPromptWithOutput` omits Analysis; `BuildStartPrompt` omits Spec. Both set `BranchDir: ctx.ContextDir`.
3. **Verified safe:** Zero-arity wrappers `BuildRefinePrompt` and `BuildSpecPrompt` preserved with original signatures.

### `internal/review/review.go`
1. **Verified safe:** Both builders omit Spec and Diff; set `Branch`, `OutputPath`, `BranchDir`. `outputPath` is always non-empty at the call site — ensures AI writes output to the review file regardless of `--to-file`.
2. **Verified safe:** `diff string` and `contextMode string` parameters fully removed — no dead parameters remain.

### `internal/cli/review.go`
1. **Verified safe:** `ctx.WarnMissingPredecessors("review", os.Stderr)` called at line 79, before prompt building.
2. **Verified safe:** `diff.md` written unconditionally (line 84) — not conditional on any flag. Always exists when the review prompt's file-read instruction fires.
3. **Verified safe:** Empty diff guard (lines 69-72) returns early before `diff.md` write — no empty file written.

### `internal/cli/plan.go`
1. **Verified safe:** `ctx.WarnMissingPredecessors("spec", os.Stderr)` called at line 139 — before the hard-error guard. Correct order: warn first, then hard-fail if blocking.
2. **Verified safe:** `contextMode` computation removed cleanly from both `runPlanRefine` and `runPlanSpec`.

### `internal/cli/start.go`
1. **Verified safe:** `ctx.WarnMissingPredecessors("start", os.Stderr)` called before KB loading and prompt building.
2. **Verified safe:** KB loading uses `knowledge.List` + `filepath.Rel` unconditionally. If list errors and `flagVerbose` is false, `kb` stays empty; `{{if .KB}}` guard handles this correctly.

### `internal/cli/knowledge_cmd.go`
1. **Verified safe:** `buildBranchLessonData` correctly sets inline content for `knowledge/add-branch` — this template still needs inline content, not reference mode.
2. **Defect (Nit):** Alignment inconsistency noted above.

### `internal/plan/refine_test.go`
1. **Verified safe:** All three tests correctly verify reference mode: no inline sentinel, file path reference present. Pattern is consistent with other test files.
2. **Defect (Low):** `TestBuildRefinePromptWithOutput_OmitsAnalysisAndTicket` missing positive assertions — noted above.

### `internal/review/review_test.go` (new)
1. **Verified safe:** `TestBuildCodePrompt_OmitsDiffAndSpec` checks three things: spec sentinel not inlined, `spec.md` referenced, `diff.md` referenced. Complete.
2. **Verified safe:** `TestBuildSecurityPrompt_OmitsDiff` checks `diff.md` referenced. Correct.

### `internal/context/context_test.go`
1. **Verified safe:** 5 `WarnMissingPredecessors` tests cover: start with no spec (warns), start with spec (silent), review with no spec (warns), spec with no analysis (warns), unknown step (silent). All cases match the implementation.

### Templates
1. **Verified safe:** `refine/base.md.tmpl` — ticket.md has fallback: "If the file is absent, infer requirements from the branch name and notes." Notes and analysis use soft phrasing. No hard file-read dependency.
2. **Verified safe:** `spec/base.md.tmpl` — references `{{.BranchDir}}/refined-analysis.md`. ✓
3. **Verified safe:** `start/base.md.tmpl` — references `{{.BranchDir}}/spec.md`. ✓
4. **Verified safe:** `review/code.md.tmpl` — references `{{.BranchDir}}/spec.md` and `{{.BranchDir}}/diff.md`. ✓
5. **Verified safe:** `review/security.md.tmpl` — references `{{.BranchDir}}/diff.md`. ✓

### `docs/how-to-customise-prompts.md`
1. **Verified safe:** Table updated — `Notes` removed, `BranchDir` added with accurate description. Inline fields documented as `knowledge/add-branch`-only.
2. **Verified safe:** New "Referencing context files" section gives concrete examples for all five reference paths.

### `.qode/branches/optimize-prompts/spec.md`
1. **Verified safe:** Design change note prepended accurately describes that ContextMode was removed mid-implementation and references `context/notes.md` for rationale.

---

## Summary

**Issues by severity:**
- Low: 1 (missing positive assertions in refine test)
- Nit: 1 (alignment inconsistency in knowledge_cmd.go struct literal)

No Critical or High findings.

**Top 3 before merging:**
1. Add positive assertions to `TestBuildRefinePromptWithOutput_OmitsAnalysisAndTicket` to match the testing pattern established by the other builders (Low — does not affect correctness but closes a gap in verification).
2. Fix alignment in `buildBranchLessonData` struct literal (Nit).
3. No blockers — both items are minor and do not affect correctness or runtime behavior.

---

## Rating

A score is a shipping recommendation. Score from what you found,
not from what you didn't look for.

| Dimension      | Score (0-2) | What you verified (not what you assumed) |
|----------------|-------------|------------------------------------------|
| Correctness    | 2           | All 5 builders verified to set BranchDir and omit inline content; WarnMissingPredecessors wired to all three commands; diff.md written unconditionally; branchDir existence guaranteed by context.Load; ticket.md fallback present in template |
| Code Quality   | 1           | Alignment inconsistency in knowledge_cmd.go struct literal (new to this branch); all functions well under 50 lines; dead code (ContextMode, Notes field) cleanly removed |
| Architecture   | 2           | ContextMode removed cleanly across 12+ files with no dead code; CLI/builder/template layer separation preserved; diff.md write correctly placed in CLI layer; knowledge/add-branch correctly exempted from reference mode |
| Error Handling | 2           | `_, _ = fmt.Fprintln` satisfies errcheck throughout WarnMissingPredecessors; error wrapping with %w in review.go diff write; empty-diff early return prevents writing empty file; filepath.Rel fallback to absolute path present |
| Testing        | 2           | 5 new WarnMissingPredecessors tests cover all switch cases; 2 new review_test.go tests verify positive and negative for each builder; 3 refine tests verify reference mode; one minor gap (missing positive assertions in refine test) does not undermine overall coverage |

**Total Score: 9.0/10**

Constraints check: No Critical or High findings — no score cap applies. Score ≥ 8.0 justified by: Correctness fully verified for all modified builders, CLI commands, and templates; Architecture cleanly removes ContextMode with no residue; Error Handling satisfies linter constraints explicitly. Deduction on Code Quality for the alignment inconsistency introduced in the new `BranchDir`/`Lessons` fields in `knowledge_cmd.go`.
