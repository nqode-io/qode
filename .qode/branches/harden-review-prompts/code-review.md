# Code Review — qode (harden-review-prompts)

## Reviewer Stance

**Assumptions made by this code:**
- `internal/prompt/templates/review/code.md.tmpl` and `security.md.tmpl`: assume the Go template engine receives `{{.Project.Name}}`, `{{.Branch}}`, `{{.Diff}}`, `{{.Layers}}` fully populated, and that all existing conditional blocks (`{{if .Spec}}`, `{{if .OutputPath}}`) remain structurally valid. No new variables introduced — this assumption holds trivially.
- `.cursorrules/qode-workflow.mdc`: assumes the reader follows step numbering in order. The swap of steps 7 and 8 is an implicit claim about the correct workflow sequence.
- `.claude/commands/*.md`: the blank line additions assume the consuming IDE ignores extra blank lines between H1 headings and body text. All tested IDE parsers do.

**Earliest silent failure point:** The step-order inconsistency in `.cursorrules/qode-workflow.mdc` — a developer following cursorrules will run "Lessons" before "Check", while a developer following `CLAUDE.md` or `README.md` will run them in the opposite order. There is no runtime error; the inconsistency surfaces only as developer confusion.

---

## Issues

**Severity:** Medium
**File:** `.cursorrules/qode-workflow.mdc:20-21`
**Issue:** Steps 7 and 8 are swapped relative to `CLAUDE.md` (Check=7, Lessons=8) and `README.md` (Check=8, Lessons=9 in that doc's numbering). The change brings the cursorrules into alignment with what `buildClaudeMD()` generates (`internal/ide/claudecode.go:74-77`), but that generator is itself inconsistent with the checked-in `CLAUDE.md`. Net result: three workflow documents now give three different orderings for these steps.
**Suggestion:** Either (a) revert the cursorrules change and address the generator inconsistency in a separate ticket, or (b) accept this ordering and update `CLAUDE.md` and `README.md` in the same commit so all three are consistent. The ordering change is also out of scope for this ticket (scope: review templates + IDE review commands + README Reviews section).

---

**Severity:** Nit
**File:** `.claude/commands/qode-knowledge-add-branch.md`, `qode-plan-spec.md`, `qode-review-code.md`, `qode-review-security.md`, `qode-start.md`
**Issue:** Five command files each gained one blank line after the H1 title. These are artifacts of running `qode ide sync`, which regenerates from `claudeSlashCommands()` — the Go string literals use double newlines after the title (`\n\n`). The files are semantically unchanged. Not harmful, but adds diff noise to this PR outside the ticket scope.
**Suggestion:** Either run `qode ide sync` in a separate housekeeping commit, or accept the formatting and note it in the PR description. No code change required to fix.

---

## File-by-File Evidence

### `internal/prompt/templates/review/code.md.tmpl`
1. **Verified safe:** Post-mortem framing (lines 4–6) matches ticket spec verbatim. No trailing whitespace anomalies. The existing role-setting sentence on line 3 is correctly preserved.
2. **Verified safe:** `## Reviewer Stance` section (lines 8–15) inserted between opening framing and `## Project Context`. All three interrogation bullets present. No `{{` delimiters introduced — template will render without modification.
3. **Verified safe:** `**Total Score: X.X/10**` line (line 111) preserved unchanged. The scoring regex `(?i)total\s*score[:\s*]*(\d+(?:\.\d+)?)\s*/\s*10` in `internal/scoring/extract.go:9` continues to match. Score-cap constraints (lines 113–117) are additive and do not affect the parseable line.
4. **Verified safe:** `{{if .Spec}}` block (lines 25–29) and `{{if .OutputPath}}` block (lines 120–127) both preserved unchanged.
5. **Verified safe:** `### 6. Performance` section (lines 68–70) preserved; not in scope for removal per ticket.
6. **Verified safe:** Correctness bullet (line 45) added as last item under `### 1. Correctness`, before blank line leading to `### 2. Code Quality`.

### `internal/prompt/templates/review/security.md.tmpl`
1. **Verified safe:** Path-mapping framing (lines 4–6) matches ticket spec verbatim.
2. **Verified safe:** `## Working Assumptions` (lines 8–15) inserted before `## Project Context`. "Unverified trust is where vulnerabilities live." closing line is present.
3. **Verified safe:** `## Adversary Simulation` (lines 83–92) is inserted after the Dependency Issues section (line 81) and before `## Output Format` (line 102). The three-attempt format is correct.
4. **Verified safe:** `## Incomplete Review Signals` (lines 94–100) inserted immediately before `## Output Format` (line 102). All four signal patterns present.
5. **Verified safe:** Rating table (lines 120–126) uses "Control or finding" column header. "Total Score: X.X/10" line (128) unchanged. "10.0 is not a valid security score" constraint present at line 135.
6. **Verified safe:** Security Rating section has no preamble paragraph — correct per ticket spec (code template has it; security does not).

### `README.md`
1. **Verified safe:** `## Reviews` section inserted between `## Scoring` and `## IDE Support` — correct position per spec.
2. **Verified safe:** Code review bullet list matches ticket §5 verbatim (5 bullets including score constraints).
3. **Verified safe:** Security review bullet list matches ticket §5 verbatim (5 bullets; "Score 10.0 is not valid" present). Local override note present.

### `.cursorrules/qode-workflow.mdc`
1. **Defect (Medium):** Step 7/8 ordering inconsistency — documented above.
2. **Verified safe:** Steps 1–6 and step 9 are correct and unchanged.
3. **Verified safe:** `## Rules` section is unchanged.

### `.claude/commands/*.md` (5 files)
1. **Verified safe:** Content semantically identical to pre-branch state — blank lines only.
2. **Nit:** Unintended diff noise from generator rerun.

---

## Summary

**Issues by severity:**
- Medium: 1 (workflow step ordering inconsistency in cursorrules)
- Nit: 1 (blank line noise in 5 command files)

**Overall assessment:** The core changes (template hardening) are correctly implemented and match the ticket spec exactly. The `**Total Score: X.X/10**` format is preserved, all new sections are in the right positions, and no template logic was modified. The one real concern is the out-of-scope step ordering change in `.cursorrules/qode-workflow.mdc`, which adds a third conflicting ordering to documentation that is already inconsistent between the generator and committed files.

**Top 3 items before merging:**
1. Resolve the step 7/8 ordering — either revert the cursorrules change or align `CLAUDE.md` and `README.md` with the new ordering in the same PR.
2. Decide whether the blank line diffs in `.claude/commands/*.md` are intentional (generator artifact) — if yes, note in PR description; if no, revert.
3. No blocker: consider filing a follow-up ticket for the `buildClaudeMD()` generator / `CLAUDE.md` workflow ordering mismatch, which predates this branch.

---

## Rating

A score is a shipping recommendation. Score from what you found,
not from what you didn't look for.

| Dimension      | Score (0-2) | What you verified (not what you assumed) |
|----------------|-------------|------------------------------------------|
| Correctness    | 2           | All 10 template changes verified against ticket spec line-by-line; `Total Score` format confirmed compatible with `extract.go:9` regex; no template variables added or removed |
| Code Quality   | 1           | Step 7/8 swap in cursorrules creates documented inconsistency across three files; blank line diffs in 5 files are unintended generator artifacts |
| Architecture   | 2           | Changes confined to template files and docs — correct layer; no logic changes; `go:embed` boundary respected; scoring pipeline untouched |
| Error Handling | 2           | No error-path code modified; template rendering is additive; no new `{{` delimiters that could cause parse errors |
| Testing        | 2           | Spec explicitly excludes tests; existing `internal/scoring/extract_test.go` tests are unaffected (format preserved); template output verified by section-header grep |

**Total Score: 9.0/10**

Constraints check: No Critical or High findings — no score cap applies. Score ≥ 8.0 justified by: Correctness fully verified against spec; Architecture correctly isolates changes to template layer; Error Handling and Testing unaffected. Deduction on Code Quality is for the out-of-scope cursorrules step swap creating a three-way inconsistency in workflow documentation.
