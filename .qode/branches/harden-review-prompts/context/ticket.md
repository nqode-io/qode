# Harden review prompts with enforcement mechanisms and slim IDE commands

**## Problem / motivation**

The code and security review prompts produce shallow, praise-leaning outputs. Scores inflate without evidence, issues go unfound, and reviews become a formality rather than a gate. Additionally, the slash command files for both Claude Code and Cursor duplicate review logic that belongs exclusively in the prompt templates, creating a maintenance split where changing one doesn't change the other.

**## Proposed solution**

### 1. `internal/prompt/templates/review/code.md.tmpl`

Replace the opening line and add two new sections before Output Format. Replace the Rating section entirely.

**Opening** — replace `"Review the diff below objectively."` with:
```
Before reading a single line: write the incident report. This code shipped,
caused a production failure, and you are explaining what went wrong.
Now read the diff and find it.
```

**Reviewer Stance** — insert after opening, before Project Context:
```markdown
## Reviewer Stance

Before scoring, interrogate each file:
- What does this code assume about its inputs, callers, and environment?
  List those assumptions. Which ones are unverified?
- If a downstream caller receives this output, can they accomplish their goal?
  Are there missing fields, missing timing info, or ambiguous states?
- Where is the earliest point this could fail silently?
```

**Correctness criterion** — add bullet:
```
- Does each function's output enable its caller's next logical action?
```

**Unfinished Review Signals** — insert before Output Format:
```markdown
## Unfinished Review Signals

If you find yourself writing any of the following, the review is not done:
- "Looks good overall" / "No major issues"
- "Well-structured" / "Clean implementation" (without citing what you verified)
- "Could consider..." — decide: is it a defect or isn't it?
- A score above 8 with a justification column left vague

For each file reviewed: document at least 3 specific items — defects found,
concerns flagged, or properties explicitly verified as safe (with the reason).
```

**Rating section** — replace entirely:
```markdown
## Rating

A score is a shipping recommendation. Score from what you found,
not from what you didn't look for.

| Dimension      | Score (0-2) | What you verified (not what you assumed) |
|----------------|-------------|------------------------------------------|
| Correctness    |             |                                          |
| Code Quality   |             |                                          |
| Architecture   |             |                                          |
| Error Handling |             |                                          |
| Testing        |             |                                          |

**Total Score: X.X/10**

Constraints:
- A Critical finding voids a high score — total cannot exceed 5.0
- A High finding caps the total at 7.5
- Total ≥ 8.0 requires the justification column to contain specifics, not sentiment
- Total 10.0 requires an explanation of what makes this better than 99% of shipped code
```

---

### 2. `internal/prompt/templates/review/security.md.tmpl`

**Opening** — replace `"Review the diff below for security vulnerabilities."` with:
```
Read this diff as a map: trace every path from external input to persistent
state, external service, or sensitive data. Your job is to find where that
map leads somewhere it shouldn't.
```

**Working Assumptions** — insert after opening, before Project Context:
```markdown
## Working Assumptions

Before the checklist: identify what this code trusts.
- Which inputs arrive from outside the trust boundary?
- What does this code assume the caller has already validated?
- Which of those assumptions are enforced vs. merely expected?

Unverified trust is where vulnerabilities live.
```

**Adversary Simulation** — insert after Security Checklist:
```markdown
## Adversary Simulation

Inhabit the role of someone actively attempting to exploit this change.
Describe three attempts — be specific about technique, target, and outcome:

1. **Attempt:** [what you'd try] | **Target:** [function/endpoint] | **Result:** [would succeed / blocked by X]
2. **Attempt:** ...
3. **Attempt:** ...

If all three fail, explain what controls stopped each one.
```

**Incomplete Review Signals** — insert before Output Format:
```markdown
## Incomplete Review Signals

These phrases mean the review isn't finished:
- "No significant security issues found"
- "Implementation appears secure"
- "Low overall risk"
- A score above 8 citing only the absence of bugs, not the presence of controls
```

**Rating section** — replace entirely:
```markdown
## Rating

| Dimension             | Score (0-2) | Control or finding that determines this score |
|-----------------------|-------------|------------------------------------------------|
| Injection Prevention  |             |                                                |
| Auth & Access Control |             |                                                |
| Data Protection       |             |                                                |
| Input Validation      |             |                                                |
| Dependency Security   |             |                                                |

**Total Score: X.X/10**

Constraints:
- A Critical vulnerability voids a high score — total cannot exceed 5.0
- A High vulnerability caps the total at 7.5
- Total ≥ 8.0 requires citing specific controls observed (e.g. parameterized queries
  at line X, input allowlist at line Y) — not just the absence of known bugs
- Total 10.0 is not a valid security score; complete security is not provable
```

---

### 3. IDE command files — remove duplicated logic

The slash command files currently contain "Review Standards" sections that duplicate what the templates already enforce. Strip them. The commands already follow the correct pattern (run CLI, use stdout as prompt) — only the redundant standards sections need removing.

**`.claude/commands/qode-review-code.md`** — target state:
```markdown
# Code Review — qode

Run this command and use its stdout output as your prompt:
  qode review code

After completing the review:
- Save to: .qode/branches/$(git branch --show-current)/code-review.md
- List all Critical and High issues clearly
- Provide specific, actionable fix suggestions
```

**`.claude/commands/qode-review-security.md`** — target state:
```markdown
# Security Review — qode

Run this command and use its stdout output as your prompt:
  qode review security

After completing the review:
- Save to: .qode/branches/$(git branch --show-current)/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue
```

**`.cursor/commands/qode-review-code.mdc`** — target state:
```
---
description: Code review for qode
---

Run this command and use its stdout output as your prompt:
  qode review code

After completing the review, save it to:
  .qode/branches/$(git branch --show-current)/code-review.md
```

**`.cursor/commands/qode-review-security.mdc`** — target state:
```
---
description: Security review for qode
---

Run this command and use its stdout output as your prompt:
  qode review security

After completing the review, save it to:
  .qode/branches/$(git branch --show-current)/security-review.md
```

---

### 4. IDE generator Go files

`internal/ide/claudecode.go` — `claudeSlashCommands()`: ensure `qode-review-code` and `qode-review-security` string literals match the target state above. Remove any "Review Standards" content if present.

`internal/ide/cursor.go` — `slashCommands()`: same.

---

### 5. `README.md`

Add a **Reviews** section before IDE Support:

```markdown
## Reviews

Code and security reviews use hardened prompts designed to prevent shallow outputs:

**Code review** (`/qode-review-code`):
- Reviewer reads the diff as if writing a post-mortem — find the failure before it ships
- Each file requires ≥ 3 documented items: defects, flagged concerns, or properties explicitly verified safe
- Score constraints: Critical finding → total ≤ 5.0 | High finding → total ≤ 7.5
- Scores ≥ 8.0 must cite specific properties verified, not the absence of bugs
- Score 10.0 requires justification against typical production-quality code

**Security review** (`/qode-review-security`):
- Reviewer maps every path from external input to persistent state or sensitive data
- Adversary Simulation section is required: three named exploit attempts with technique, target, and outcome
- Same score constraints apply
- Scores ≥ 8.0 must cite specific controls observed (e.g. parameterized queries at line X)
- Score 10.0 is not valid — complete security is not provable

Both prompts can be customised via `.qode/prompts/review/` local overrides.
```

---

**## Alternatives considered**

Duplicating enforcement in both the templates and the slash commands (attempted as a first pass). Rejected because it creates two sources of truth — any future prompt change would need to be mirrored in four command files and two Go generators.

**## Additional context**

The correct repo (`nqode-io/qode`) already has the current state: `qode review code` outputs the rendered prompt to stdout; slash commands run the CLI and treat its stdout as the next prompt. Template changes in this ticket apply on top of that existing architecture.
