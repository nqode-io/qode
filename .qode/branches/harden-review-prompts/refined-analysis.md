<!-- qode:iteration=2 score=25/25 -->

# Requirements Analysis — Harden Review Prompts (Iteration 2)

## 1. Problem Understanding

**Restated problem:** The AI-generated code and security reviews are systematically shallow. Reviewers produce vague praise ("looks good", "well-structured"), inflate scores without evidence, and miss real defects. The reviews pass quality gates on paper but fail to catch bugs before they ship. A secondary problem is prompt maintenance hygiene: the ticket references IDE command files that may duplicate review logic, creating two sources of truth.

**User need:** Developers relying on `qode review code` and `qode review security` need reviews they can actually trust as a shipping gate. The goal is to change the AI's *framing* before it reads a single line of code — forcing adversarial, evidence-based analysis rather than optimistic summarisation.

**Business value:** If reviews reliably catch defects and vulnerabilities, `qode check` becomes a genuine quality gate rather than a formality. This is the core value proposition of qode.

**Ambiguities resolved:**

1. The ticket says IDE command files "currently contain 'Review Standards' sections." Reading the current files:
   - `.claude/commands/qode-review-code.md` — no "Review Standards" section present; already at target state
   - `.claude/commands/qode-review-security.md` — already at target state
   - `.cursor/commands/qode-review-code.mdc` — already at target state
   - `.cursor/commands/qode-review-security.mdc` — already at target state
   - `claudeSlashCommands()` in `internal/ide/claudecode.go` — generates target state already
   - `slashCommands()` in `internal/ide/cursor.go` — generates target state already

   The cleanup described in ticket sections 3 and 4 is already complete. **The only substantive changes are the two template files and README.md.**

2. The ticket's "Correctness criterion" bullet belongs under `### 1. Correctness` in Review Criteria of `code.md.tmpl`.

3. The `### 6. Performance` section in `code.md.tmpl` is not mentioned by the ticket and must be preserved.

---

## 2. Technical Analysis

**Affected components:**

| File | Change type | Scope |
|------|-------------|-------|
| `internal/prompt/templates/review/code.md.tmpl` | Modify | Replace opening, add 2 sections, add 1 bullet, replace Rating |
| `internal/prompt/templates/review/security.md.tmpl` | Modify | Replace opening, add 3 sections, replace Rating |
| `README.md` | Modify | Add Reviews section before IDE Support |
| All IDE command files | No change needed | Already at target state (verified) |
| `internal/ide/claudecode.go` | No change needed | `claudeSlashCommands()` already generates target state |
| `internal/ide/cursor.go` | No change needed | `slashCommands()` already generates target state |

**Key technical decisions:**

1. **Template embedding** — Templates are embedded into the binary via `//go:embed templates` at `internal/prompt/engine.go:16`. There are no build constraints or conditional compilation. Editing the `.tmpl` files takes effect on the next `go build` / `go install`. No separate asset pipeline step is needed.

2. **Score-parsing compatibility** — The runner reads saved review files and extracts scores using the regex in `internal/scoring/extract.go:9`:
   ```go
   regexp.MustCompile(`(?i)total\s*score[:\s*]*(\d+(?:\.\d+)?)\s*/\s*10`)
   ```
   This matches `**Total Score: X.X/10**` case-insensitively with flexible spacing. The new rating tables preserve this exact line format. No changes to `internal/scoring/extract.go` or `internal/runner/runner.go` are needed, and no regression in score parsing will occur.

3. **`qode ide sync` not required** — Because `internal/ide/claudecode.go` and `cursor.go` already generate the correct review command content, running `qode ide sync` after this ticket produces identical files. No action required from users.

4. **Exact insertion points in `code.md.tmpl`:**
   - Line 4: Replace `"Review the diff below objectively."` with post-mortem framing (3 lines)
   - After line 4, before line 6 (`## Project Context`): insert `## Reviewer Stance` (8 lines)
   - Under `### 1. Correctness` (after line 35, the last existing bullet): add caller-output bullet
   - Before `## Output Format` (line 60): insert `## Unfinished Review Signals` (10 lines)
   - Lines 75–88 (`## Rating` through `**Total Score: X.X/10**`): replace with new 14-line constrained table

5. **Exact insertion points in `security.md.tmpl`:**
   - Line 4: Replace `"Review the diff below for security vulnerabilities."` with path-mapping framing (3 lines)
   - After line 4, before line 6 (`## Project Context`): insert `## Working Assumptions` (8 lines)
   - After line 70 (end of Dependency Issues section, `### Dependency Issues` block): insert `## Adversary Simulation` (9 lines)
   - Before `## Output Format` (line 72): insert `## Incomplete Review Signals` (8 lines)
   - Lines 88–100 (`## Rating` through `**Total Score: X.X/10**`): replace with new 14-line constrained table

6. **README.md insertion** — Insert `## Reviews` section between `## Scoring` (ends line 84) and `## IDE Support` (line 86). Section content from ticket verbatim.

**Patterns to follow:**
- Templates use raw Markdown; no new Go template variables to introduce
- README `##` level headers, pipe tables for structured data
- No test changes required for template content edits

---

## 3. Risk & Edge Cases

**Risk: Score regex regression (mitigated)** — Confirmed in `internal/scoring/extract.go:9`: the regex `(?i)total\s*score[:\s*]*(\d+(?:\.\d+)?)\s*/\s*10` matches the unchanged `**Total Score: X.X/10**` line. Rating table header and third column changes don't affect parsing. No regression.

**Risk: Template rendering breaks** — New Markdown sections are plain text inserted between existing Go template directives. No new `{{` delimiters introduced. Rendering is additive. Risk is low; verify with `qode review code --to-file` after changes.

**Risk: Score cap constraints not machine-enforced** — The "Critical finding → total ≤ 5.0" constraints are prompt instructions only. `internal/runner/runner.go:75` checks `result.CodeReview >= cfg.Review.MinCodeScore` but does not cross-reference the severity of findings in the review text. If an AI reviewer ignores the cap constraint, `qode check` can still pass with an improperly inflated score. This is a known and accepted limitation of prompt-only enforcement — runner changes are explicitly out of scope per the ticket.

**Risk: Adversary Simulation misread** — The section instructs the AI to roleplay exploit attempts. Output could alarm readers. Mitigation: README Reviews section explains the pattern explicitly.

**Edge case: Empty diff** — `{{.Diff}}` renders empty if no staged changes. The new framing sections (post-mortem opening, Reviewer Stance, Adversary Simulation) remain in the prompt. This is correct; the reviewer should state there is nothing to review.

**Edge case: Per-project template overrides** — Projects using `.qode/prompts/review/code.md.tmpl` overrides do not automatically inherit these changes. This is expected behaviour and is documented in the ticket's Additional Context.

**Edge case: `go:embed` and binary rebuilds** — Template changes require a new binary build. Projects using pre-built qode binaries will not see changes until they upgrade. No special handling needed; this is standard behaviour for embedded assets.

**Security considerations:** No new input surfaces, no external service calls, no storage changes. These are prompt text modifications only.

**Performance considerations:** None. Template rendering is synchronous and in-memory. The new sections add ~50 lines of text; negligible impact on stdout output time.

---

## 4. Completeness Check

**Acceptance criteria:**

| # | Criterion | Source |
|---|-----------|--------|
| AC1 | `code.md.tmpl` line 4 replaced with 3-line post-mortem framing | Ticket §1 |
| AC2 | `## Reviewer Stance` (3 interrogation bullets) inserted after opening, before `## Project Context` | Ticket §1 |
| AC3 | Bullet "Does each function's output enable its caller's next logical action?" added under `### 1. Correctness` | Ticket §1 |
| AC4 | `## Unfinished Review Signals` (4 signal patterns + per-file rule) inserted before `## Output Format` | Ticket §1 |
| AC5 | `## Rating` replaced with "What you verified" column + 4 scoring constraints | Ticket §1 |
| AC6 | `security.md.tmpl` line 4 replaced with 3-line path-mapping framing | Ticket §2 |
| AC7 | `## Working Assumptions` (3 trust bullets + conclusion line) inserted after opening, before `## Project Context` | Ticket §2 |
| AC8 | `## Adversary Simulation` (3-attempt table + controls requirement) inserted after Security Checklist | Ticket §2 |
| AC9 | `## Incomplete Review Signals` (4 signal patterns) inserted before `## Output Format` | Ticket §2 |
| AC10 | `## Rating` replaced with "Control or finding" column + 4 scoring constraints including "10.0 is not valid" | Ticket §2 |
| AC11 | `.claude/commands/qode-review-{code,security}.md` verified at target state — no edits | Ticket §3 (done) |
| AC12 | `.cursor/commands/qode-review-{code,security}.mdc` verified at target state — no edits | Ticket §3 (done) |
| AC13 | `claudeSlashCommands()` and `slashCommands()` verified to generate target state — no edits | Ticket §4 (done) |
| AC14 | `## Reviews` section added to `README.md` before `## IDE Support` | Ticket §5 |
| AC15 | `qode review code --to-file` output contains `## Reviewer Stance`, `## Unfinished Review Signals`, "What you verified" | Implicit |
| AC16 | `qode review security --to-file` output contains `## Working Assumptions`, `## Adversary Simulation`, `## Incomplete Review Signals`, "Control or finding" | Implicit |

**Implicit requirements not in ticket:**
- `### 6. Performance` section in `code.md.tmpl` must be preserved (not removed)
- `{{if .OutputPath}}` blocks at end of both templates must be preserved unchanged
- `{{if .Spec}}` block in `code.md.tmpl` must be preserved unchanged
- The `**Total Score: X.X/10**` line format must remain unchanged (runner dependency)

**Explicitly out of scope:**
- `internal/scoring/extract.go` — score regex is already compatible
- `internal/runner/runner.go` — no machine enforcement of score caps
- Per-project `.qode/prompts/review/` overrides
- Any IDE other than Claude Code and Cursor
- Test file additions

---

## 5. Actionable Implementation Plan

**Task 1 — Update `internal/prompt/templates/review/code.md.tmpl`** (1 commit)
- Replace line 4 opening with 3-line post-mortem framing
- Insert `## Reviewer Stance` block after opening, before `## Project Context`
- Add caller-output bullet under `### 1. Correctness` (after the last existing bullet in that section)
- Insert `## Unfinished Review Signals` block immediately before `## Output Format`
- Replace `## Rating` block (lines 75–88) with new constrained table preserving `**Total Score: X.X/10**` format

**Task 2 — Update `internal/prompt/templates/review/security.md.tmpl`** (1 commit)
- Replace line 4 opening with 3-line path-mapping framing
- Insert `## Working Assumptions` block after opening, before `## Project Context`
- Insert `## Adversary Simulation` block after the Dependency Issues section, before `## Output Format`
- Insert `## Incomplete Review Signals` block immediately before `## Output Format`
- Replace `## Rating` block (lines 88–100) with new constrained table preserving `**Total Score: X.X/10**` format

**Task 3 — Add Reviews section to `README.md`** (1 commit)
- Insert `## Reviews` section between `## Scoring` (line 84) and `## IDE Support` (line 86)
- Content: verbatim from ticket including code/security review bullet lists and override note

**Task 4 — Smoke test** (no commit)
- Run `qode review code --to-file` → confirm output contains all new section headers
- Run `qode review security --to-file` → confirm output contains all new section headers
- Confirm `**Total Score: X.X/10**` line is present and unchanged in both outputs

**Suggested commit messages:**
1. `feat: harden code review prompt with stance, signals, and constrained rating`
2. `feat: harden security review prompt with assumptions, adversary simulation, and constrained rating`
3. `docs: add Reviews section to README explaining hardened prompt behaviour`

**No prerequisite work.** All three tasks are independent and can be implemented in any order. Tasks 1 and 2 are the highest value and should be done first.
