# Ticket

## nqode-io/qode#25 — Optimize prompts for token usage

**State:** open | **Author:** petar-stupar | **Label:** enhancement | **Depends on:** nqode-io/qode#24

---

## Problem / Motivation

Every workflow step (refine, spec, start, review-code, review-security) embeds the full text of previous step outputs directly into the prompt template via TemplateData fields (`.Ticket`, `.Analysis`, `.Spec`, `.Diff`, `.KB`). For example, `spec/base.md.tmpl` inlines `{{.Analysis}}` (the entire refined analysis), and `start/base.md.tmpl` inlines `{{.Spec}}` (the entire spec). When running multiple steps in the same Claude Code or Cursor session, this duplicates content already present in the conversation context, wasting tokens and potentially hitting context window limits on large features.

Additionally, the current design assumes each step runs in isolation. There is no mechanism to detect whether context from a previous step is already available in the active session or whether files need to be re-read.

---

## Proposed Solution

### 1. Reference files instead of inlining content

Update templates to emit file-path references with read instructions (e.g., "Read the refined analysis from `.qode/branches/<branch>/refined-analysis.md`") rather than pasting the full content inline via `{{.Analysis}}`. The AI agent can then decide whether to read the file (if starting a fresh session) or rely on context already in the conversation.

### 2. Add a `context_mode` field to TemplateData

Introduce a rendering mode — `inline` (current behaviour, for backward compatibility and `--to-file` debugging) vs `reference` (default for IDE slash commands). The templates use `{{if eq .ContextMode "inline"}}` to conditionally inline or reference.

### 3. Ensure output files are always written

Each step must reliably produce its output file (refined-analysis.md, spec.md, code-review.md, security-review.md) so that subsequent steps can reference them. Add validation in `context.Load()` to warn if expected predecessor files are missing.

### 4. Update all templates

Modify `refine/base.md.tmpl`, `spec/base.md.tmpl`, `start/base.md.tmpl`, `review/code.md.tmpl`, `review/security.md.tmpl`, `scoring/judge_refine.md.tmpl`, and both knowledge templates to support the reference mode.

### 5. Update TemplateData struct

In `internal/prompt/engine.go`, add `ContextMode string` and `BranchDir string` (the absolute path to the branch context folder, so templates can emit correct file paths) to the TemplateData struct.

---

## Files to Modify

- `internal/prompt/engine.go` — add `ContextMode` and `BranchDir` to TemplateData
- `internal/prompt/templates/refine/base.md.tmpl` — conditional inline/reference for `.Analysis`, `.Ticket`, `.Notes`
- `internal/prompt/templates/spec/base.md.tmpl` — conditional inline/reference for `.Analysis`
- `internal/prompt/templates/start/base.md.tmpl` — conditional inline/reference for `.Spec`, `.KB`
- `internal/prompt/templates/review/code.md.tmpl` — conditional inline/reference for `.Spec`, `.Diff`
- `internal/prompt/templates/review/security.md.tmpl` — same as code review
- `internal/cli/plan.go`, `internal/cli/review.go`, `internal/cli/start.go` — set ContextMode when building TemplateData
- `internal/context/context.go` — add file-existence validation helpers

---

## Alternatives Considered

- **Always inline, but truncate to a summary.** Reduces tokens but loses detail; the AI may produce worse output without the full context.
- **Session-detection heuristic.** Detect whether running inside an active Claude/Cursor session and auto-switch modes. Rejected because there is no reliable way to detect this from a CLI tool writing to stdout.
- **Remove inlining entirely.** Would break `--to-file` debugging and non-IDE usage where there is no file-reading agent.
