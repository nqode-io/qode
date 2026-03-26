# Technical Specification — Harden Review Prompts

*Branch: harden-review-prompts | Generated: 2026-03-26*

---

## 1. Feature Overview

This feature hardens the AI-generated code and security review prompts in qode to prevent shallow, praise-leaning outputs. Currently, the prompts produce reviews that inflate scores without evidence, miss defects, and treat `qode check` as a formality rather than a real quality gate.

The change introduces adversarial framing, structured interrogation sections, per-file evidence requirements, and explicit score-cap constraints directly into the two review prompt templates (`code.md.tmpl` and `security.md.tmpl`). No application logic changes are needed — all enforcement is via prompt engineering applied to the embedded Go template files.

**Business value:** Reviews become trustworthy signals rather than checkbox artefacts. `qode check` can be relied upon as a real shipping gate. The change applies to all projects using qode's default templates with no migration or configuration required.

**Success criteria:**
- `qode review code` and `qode review security` output contains all new sections (Reviewer Stance / Working Assumptions, Unfinished / Incomplete Review Signals, Adversary Simulation, constrained Rating tables)
- `**Total Score: X.X/10**` line format is unchanged — `internal/scoring/extract.go` regex continues to parse scores correctly
- README accurately documents the hardened prompt behaviour and score constraints

---

## 2. Scope

### In scope
- `internal/prompt/templates/review/code.md.tmpl` — opening, Reviewer Stance, Correctness bullet, Unfinished Review Signals, Rating section
- `internal/prompt/templates/review/security.md.tmpl` — opening, Working Assumptions, Adversary Simulation, Incomplete Review Signals, Rating section
- `README.md` — new `## Reviews` section before `## IDE Support`

### Out of scope
- `internal/scoring/extract.go` — regex already compatible with unchanged `**Total Score: X.X/10**` format
- `internal/runner/runner.go` — no machine enforcement of score-cap constraints
- `.claude/commands/qode-review-{code,security}.md` — already at target state
- `.cursor/commands/qode-review-{code,security}.mdc` — already at target state
- `internal/ide/claudecode.go` / `cursor.go` — generators already produce correct output
- Per-project `.qode/prompts/review/` overrides
- Any IDE other than Claude Code and Cursor
- Test file additions

### Assumptions
- `qode ide sync` does not need to be run after this change — the IDE generator Go files are unchanged and already produce the correct command content
- Template changes require a new binary build; projects on pre-built binaries will see the change on next upgrade (standard `go:embed` behaviour)
- The score-cap rules ("Critical → total ≤ 5.0") are enforced by prompt framing only, not by code — this is the accepted design for this ticket

---

## 3. Architecture & Design

### Component overview

```
qode review code
      │
      ▼
internal/prompt/engine.go
  //go:embed templates          ← templates compiled into binary
      │
      ▼
internal/prompt/templates/review/code.md.tmpl   ← MODIFIED
      │  (rendered with branch diff, project name, spec)
      ▼
stdout  ──►  IDE reads as prompt  ──►  AI reviewer produces output
                                              │
                                              ▼
                              .qode/branches/<branch>/code-review.md
                                              │
                                              ▼
                              internal/scoring/extract.go
                                (regex: Total Score X.X/10)
                                              │
                                              ▼
                              internal/runner/runner.go
                                (compare to cfg.Review.MinCodeScore)
```

Same flow for `security.md.tmpl` / `security-review.md`.

### Affected layers

| Layer | File | Change |
|-------|------|--------|
| Prompt templates | `internal/prompt/templates/review/code.md.tmpl` | Modify — add sections, replace Rating |
| Prompt templates | `internal/prompt/templates/review/security.md.tmpl` | Modify — add sections, replace Rating |
| Documentation | `README.md` | Modify — add Reviews section |

### New vs modified components

All changes are modifications to existing files. No new files are created.

### Data flow impact

The prompt text grows by ~50 lines per template. The `**Total Score: X.X/10**` output line is preserved, maintaining compatibility with the scoring pipeline. No data stored differently.

---

## 4. API / Interface Contracts

No HTTP endpoints, CLI flags, or public Go APIs change.

**`qode review code` stdout contract (modified):**

The rendered prompt gains three new top-level sections and a modified Rating section:

| Section | Position in output | New/Modified |
|---------|-------------------|--------------|
| Post-mortem framing (3 lines) | After `"You are a senior software engineer..."`, replaces `"Review the diff below objectively."` | Modified |
| `## Reviewer Stance` | After opening, before `## Project Context` | New |
| Correctness caller-output bullet | Under `### 1. Correctness` | New |
| `## Unfinished Review Signals` | Before `## Output Format` | New |
| `## Rating` (constrained table) | Replaces existing Rating section | Modified |

**`qode review security` stdout contract (modified):**

| Section | Position in output | New/Modified |
|---------|-------------------|--------------|
| Path-mapping framing (3 lines) | Replaces `"Review the diff below for security vulnerabilities."` | Modified |
| `## Working Assumptions` | After opening, before `## Project Context` | New |
| `## Adversary Simulation` | After Security Checklist, before `## Output Format` | New |
| `## Incomplete Review Signals` | Before `## Output Format` | New |
| `## Rating` (constrained table) | Replaces existing Rating section | Modified |

**Score line format — unchanged:**
```
**Total Score: X.X/10**
```
Parsed by regex `(?i)total\s*score[:\s*]*(\d+(?:\.\d+)?)\s*/\s*10` in `internal/scoring/extract.go:9`.

---

## 5. Data Model Changes

None. No database tables, files schemas, or config structures change. The `qode.yaml` fields `review.min_code_score` and `review.min_security_score` are unchanged.

---

## 6. Implementation Tasks

- [ ] **Task 1 (templates):** Update `internal/prompt/templates/review/code.md.tmpl`
  - Replace line 4 `"Review the diff below objectively."` with 3-line post-mortem framing
  - Insert `## Reviewer Stance` block (3 interrogation bullets) after opening, before `## Project Context`
  - Add bullet `"Does each function's output enable its caller's next logical action?"` under `### 1. Correctness`
  - Insert `## Unfinished Review Signals` block (4 signal patterns + per-file requirement) before `## Output Format`
  - Replace `## Rating` block (lines 75–88) with new table: "What you verified" column + 4 scoring constraints; preserve `**Total Score: X.X/10**` line

- [ ] **Task 2 (templates):** Update `internal/prompt/templates/review/security.md.tmpl`
  - Replace line 4 `"Review the diff below for security vulnerabilities."` with 3-line path-mapping framing
  - Insert `## Working Assumptions` block (3 trust bullets + "Unverified trust is where vulnerabilities live.") after opening, before `## Project Context`
  - Insert `## Adversary Simulation` block (3-attempt format + controls explanation) after Dependency Issues section, before `## Output Format`
  - Insert `## Incomplete Review Signals` block (4 signal patterns) before `## Output Format`
  - Replace `## Rating` block (lines 88–100) with new table: "Control or finding" column + 4 constraints including "10.0 is not a valid security score"; preserve `**Total Score: X.X/10**` line

- [ ] **Task 3 (docs):** Update `README.md`
  - Insert `## Reviews` section between end of `## Scoring` (line 84) and `## IDE Support` (line 86)
  - Include code review bullet list, security review bullet list, and local override note as specified in ticket §5

- [ ] **Task 4 (verification):** Smoke test
  - Run `qode review code --to-file` and confirm output contains: `## Reviewer Stance`, `## Unfinished Review Signals`, `What you verified`, `**Total Score: X.X/10**`
  - Run `qode review security --to-file` and confirm output contains: `## Working Assumptions`, `## Adversary Simulation`, `## Incomplete Review Signals`, `Control or finding`, `**Total Score: X.X/10**`

---

## 7. Testing Strategy

### Unit tests
Not applicable. Template content changes have no unit-testable logic. The prompt engine's rendering is tested separately by existing engine tests; no new variables or conditional logic are introduced.

### Integration / smoke tests (manual, Task 4)
- `qode review code --to-file` output: grep for each new section header
- `qode review security --to-file` output: grep for each new section header
- Confirm `**Total Score: X.X/10**` present and correctly extracted by `internal/scoring/extract.go`

### Edge cases to verify
- Run with empty staged diff — confirm output renders (no template errors) and new sections appear in output
- Run in a project with a `.qode/prompts/review/code.md.tmpl` local override — confirm default template changes are NOT applied (override takes precedence, as expected)

### Regression check
- Existing `internal/scoring/extract_test.go` tests should pass unchanged — the `**Total Score: X.X/10**` format is preserved

---

## 8. Security Considerations

No authentication, authorisation, or input validation changes. The templates are prompt text embedded in the binary — they do not accept user input, make external calls, or write to storage directly.

The `## Adversary Simulation` section instructs the AI to simulate three exploit attempts. This is a prompt-engineering technique; the output is analysis text, not executable code. No new attack surface is introduced.

---

## 9. Open Questions

None. All ambiguities from the ticket have been resolved during requirements refinement:

1. ✅ IDE command files and Go generators are already at target state — no changes needed
2. ✅ The "Correctness criterion" bullet belongs under `### 1. Correctness`
3. ✅ `### 6. Performance` section is preserved (not in scope for removal)
4. ✅ Score-parsing regex in `internal/scoring/extract.go:9` is compatible with unchanged `**Total Score: X.X/10**` format
5. ✅ `qode ide sync` does not need to be run after this change

---

*Spec generated by qode. Copy to GitHub issue #24 for team review.*
