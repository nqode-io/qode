<!-- qode:iteration=1 score=25/25 -->

# Requirements Analysis — Harden Review Prompts

## 1. Problem Understanding

**Restated problem:** The AI-generated code and security reviews are systematically shallow. Reviewers produce vague praise ("looks good", "well-structured"), inflate scores without evidence, and miss real defects. The reviews pass quality gates on paper but fail to catch bugs before they ship. A secondary problem is prompt maintenance hygiene: the ticket references IDE command files that may duplicate review logic, creating two sources of truth.

**User need:** Developers relying on `qode review code` and `qode review security` need reviews they can actually trust as a shipping gate. The goal is to change the AI's *framing* before it reads a single line of code — forcing adversarial, evidence-based analysis rather than optimistic summarisation.

**Business value:** If reviews reliably catch defects and vulnerabilities, `qode check` becomes a genuine quality gate rather than a formality. This is the core value proposition of qode.

**Ambiguities / open questions:**

1. The ticket says IDE command files "currently contain 'Review Standards' sections that duplicate what templates already enforce." However, reading the current files:
   - `.claude/commands/qode-review-code.md` — no "Review Standards" section present
   - `.claude/commands/qode-review-security.md` — no "Review Standards" section present
   - `.cursor/commands/qode-review-code.mdc` — no "Review Standards" section present
   - `.cursor/commands/qode-review-security.mdc` — no "Review Standards" section present
   - `internal/ide/claudecode.go` `claudeSlashCommands()` — review commands have no "Review Standards" content
   - `internal/ide/cursor.go` `slashCommands()` — review commands have no "Review Standards" content

   These files already match the ticket's "target state" exactly. The cleanup described in sections 3 and 4 of the ticket appears already complete (possibly done in PR #22/#23). **The only substantive changes are to the two template files and README.md.**

2. The ticket adds a "Correctness criterion" bullet to the code template but doesn't specify which existing section it belongs to. Context implies it goes under `### 1. Correctness` in Review Criteria.

3. The code template currently has a `### 6. Performance` section not mentioned in the ticket's proposed changes. It should be preserved — nothing in the ticket removes it.

---

## 2. Technical Analysis

**Affected components:**

| File | Change type | Scope |
|------|-------------|-------|
| `internal/prompt/templates/review/code.md.tmpl` | Modify | Replace opening, add 2 sections, add 1 bullet, replace Rating |
| `internal/prompt/templates/review/security.md.tmpl` | Modify | Replace opening, add 3 sections, replace Rating |
| `README.md` | Modify | Add Reviews section before IDE Support |
| `.claude/commands/qode-review-code.md` | No change needed | Already at target state |
| `.claude/commands/qode-review-security.md` | No change needed | Already at target state |
| `.cursor/commands/qode-review-code.mdc` | No change needed | Already at target state |
| `.cursor/commands/qode-review-security.mdc` | No change needed | Already at target state |
| `internal/ide/claudecode.go` | No change needed | `claudeSlashCommands()` already generates target state |
| `internal/ide/cursor.go` | No change needed | `slashCommands()` already generates target state |

**Key technical decisions:**

1. **Template-only enforcement** — All review logic lives in the embedded templates under `internal/prompt/templates/review/`. This is already the correct architecture; the ticket confirms it. No changes to `internal/dispatch/` or `internal/runner/` are needed.

2. **Template structure for `code.md.tmpl`** — The file uses Go template syntax (`{{.Project.Name}}`, `{{.Diff}}`, etc.) and has conditional blocks (`{{if .Spec}}`, `{{if .OutputPath}}`). New sections must be inserted as plain Markdown between existing template directives, not inside conditionals.

3. **Exact insertion points in `code.md.tmpl`:**
   - Line 4: Replace `"Review the diff below objectively."` with the post-mortem framing
   - After line 4 (opening), before line 6 (`## Project Context`): insert `## Reviewer Stance`
   - Under `### 1. Correctness` (line 31): add the new bullet about caller output
   - Before `## Output Format` (line 60): insert `## Unfinished Review Signals`
   - Lines 75–88 (`## Rating` through `**Total Score: X.X/10**`): replace entirely

4. **Exact insertion points in `security.md.tmpl`:**
   - Line 4: Replace `"Review the diff below for security vulnerabilities."` with path-mapping framing
   - After line 4, before line 6 (`## Project Context`): insert `## Working Assumptions`
   - After `## Security Checklist` block (line 70, after dependency section): insert `## Adversary Simulation`
   - Before `## Output Format` (line 72): insert `## Incomplete Review Signals`
   - Lines 88–100 (`## Rating` through `**Total Score: X.X/10**`): replace entirely

5. **Rating table change significance** — The new rating tables add a constraint column ("What you verified" / "Control or finding") replacing the vague "Justification" column, and add explicit scoring caps (Critical → ≤5.0, High → ≤7.5). This is a behavioural change enforced by prompt framing, not code logic — the runner at `internal/runner/` still parses the `Total Score: X.X/10` line by regex to check against `cfg.Review.MinCodeScore` / `cfg.Review.MinSecurityScore`.

6. **`README.md` insertion point** — The "Reviews" section should be inserted between the `## Scoring` section (line 72) and `## IDE Support` (line 86). The ticket says "before IDE Support" which matches this location.

**Patterns to follow:**

- Templates use raw Markdown; no Go logic should be added — only template variable references
- README sections use `##` level headers with pipe tables for structured data
- No test changes are needed for template content changes (templates are rendered by the prompt engine; behaviour is tested by running `qode review code`)

---

## 3. Risk & Edge Cases

**Risk: Regression in `Total Score` parsing** — `internal/runner/` parses the score from the reviewer's output. The new rating table header changes from `| Dimension | Score (0-2) | Justification |` to `| Dimension | Score (0-2) | What you verified ... |`. This does not affect the `**Total Score: X.X/10**` line format which is what the runner parses. No runtime regression.

**Risk: Template rendering breaks** — The new Markdown sections are plain text inserted between existing template directives. No new template variables are introduced. Risk is low, but the output of `qode review code` should be verified after changes.

**Risk: Adversary Simulation section is misread as actual threats** — The section asks the AI to simulate three exploit attempts. This is a prompt-engineering technique to force adversarial thinking. The wording is clear ("Inhabit the role of..."), but the output could alarm readers unfamiliar with the pattern. Mitigation: the README Reviews section explains the technique.

**Risk: Score cap rules are not machine-enforced** — The constraints ("Critical finding voids a high score") are communicated to the AI via prompt text, not enforced by code in `internal/runner/`. If the AI ignores the constraint and scores 9.0 despite a Critical finding, `qode check` would pass. This is a prompt-engineering limitation, not a code bug. It is explicitly in-scope per the ticket (prompt hardening, not runner changes).

**Edge case: Projects with no diff** — If there are no staged changes, `{{.Diff}}` renders empty. New sections (Reviewer Stance, Adversary Simulation) remain in the prompt regardless. This is correct behaviour; the reviewer should note the absence of changes.

**Edge case: Per-project overrides** — Projects can override templates via `.qode/prompts/review/`. A project with a custom `code.md.tmpl` will not receive the hardened prompts automatically. This is expected and documented (ticket mentions this under "Both prompts can be customised via `.qode/prompts/review/` local overrides").

**Security considerations:** None material. These are prompt text changes; no new input surfaces, no new external calls, no data stored differently.

---

## 4. Completeness Check

**Acceptance criteria:**

| # | Criterion | Source |
|---|-----------|--------|
| AC1 | `code.md.tmpl` opening replaced with post-mortem framing | Ticket §1 |
| AC2 | `## Reviewer Stance` section inserted before Project Context in code template | Ticket §1 |
| AC3 | Correctness bullet "Does each function's output enable its caller's next logical action?" added under `### 1. Correctness` | Ticket §1 |
| AC4 | `## Unfinished Review Signals` inserted before Output Format in code template | Ticket §1 |
| AC5 | Rating section in code template replaced with new table + constraints | Ticket §1 |
| AC6 | `security.md.tmpl` opening replaced with path-mapping framing | Ticket §2 |
| AC7 | `## Working Assumptions` section inserted before Project Context in security template | Ticket §2 |
| AC8 | `## Adversary Simulation` inserted after Security Checklist in security template | Ticket §2 |
| AC9 | `## Incomplete Review Signals` inserted before Output Format in security template | Ticket §2 |
| AC10 | Rating section in security template replaced with new table + constraints | Ticket §2 |
| AC11 | IDE command files already at target state — verify, no edit required | Ticket §3 (already done) |
| AC12 | `internal/ide/claudecode.go` and `cursor.go` already generate target state — verify, no edit required | Ticket §4 (already done) |
| AC13 | `## Reviews` section added to README.md before `## IDE Support` | Ticket §5 |
| AC14 | `qode review code` stdout output verified to contain all new sections | Implicit |
| AC15 | `qode review security` stdout output verified to contain all new sections | Implicit |

**Implicit requirements not in ticket:**

- The existing `### 6. Performance` section in `code.md.tmpl` must be preserved — nothing in the ticket removes it
- The `{{if .OutputPath}}` block at the end of both templates must be preserved unchanged
- The `{{if .Spec}}` block in `code.md.tmpl` must be preserved unchanged

**Explicitly out of scope:**

- Changes to `internal/runner/` score parsing or enforcement logic
- Per-project `.qode/prompts/review/` overrides — those inherit this behaviour unless manually overridden
- Changes to other IDE generators (JetBrains, etc.) not mentioned in ticket
- Test file changes

---

## 5. Actionable Implementation Plan

Tasks are ordered to minimise context switching and allow easy review of each commit.

**Task 1 — Update `internal/prompt/templates/review/code.md.tmpl`**
- Replace line 4 opening with post-mortem framing
- Insert `## Reviewer Stance` block (5 lines) after opening, before `## Project Context`
- Add correctness caller bullet under `### 1. Correctness`
- Insert `## Unfinished Review Signals` block before `## Output Format`
- Replace `## Rating` section (lines 75–88) with new constrained table

**Task 2 — Update `internal/prompt/templates/review/security.md.tmpl`**
- Replace line 4 opening with path-mapping framing
- Insert `## Working Assumptions` block after opening, before `## Project Context`
- Insert `## Adversary Simulation` block after the Security Checklist (after Dependency Issues section, before `## Output Format`)
- Insert `## Incomplete Review Signals` block before `## Output Format`
- Replace `## Rating` section (lines 88–100) with new constrained table

**Task 3 — Verify IDE command files (no edits expected)**
- Confirm `.claude/commands/qode-review-code.md`, `.claude/commands/qode-review-security.md`, `.cursor/commands/qode-review-code.mdc`, `.cursor/commands/qode-review-security.mdc` all match target state — they already do based on current file reads
- Confirm `claudeSlashCommands()` in `internal/ide/claudecode.go` and `slashCommands()` in `internal/ide/cursor.go` already generate the correct output — confirmed

**Task 4 — Update `README.md`**
- Insert `## Reviews` section between `## Scoring` and `## IDE Support` (between lines 84 and 86) with the content from the ticket

**Task 5 — Smoke test**
- Run `qode review code` and confirm output contains: `## Reviewer Stance`, `## Unfinished Review Signals`, new Rating table header "What you verified"
- Run `qode review security` and confirm output contains: `## Working Assumptions`, `## Adversary Simulation`, `## Incomplete Review Signals`, new Rating table header "Control or finding"

**Suggested commit order:**
1. `feat: harden code review prompt with stance, signals, and constrained rating`
2. `feat: harden security review prompt with assumptions, adversary simulation, and constrained rating`
3. `docs: add Reviews section to README explaining prompt hardening`
