<!-- qode:iteration=2 score=25/25 -->

# Requirements Analysis — Optimize Prompts for Token Usage

## 1. Problem Understanding

Each qode workflow step calls a builder function that populates a `TemplateData` struct with the previous step's full output (`ctx.RefinedAnalysis`, `ctx.Spec`, a live-computed `diff` string, concatenated KB content), then renders a template that embeds this content verbatim. When the rendered prompt is piped to stdout and executed inside a running IDE AI session (Claude Code / Cursor), that session already has the previous step's output in its conversation context — it wrote those files. Re-embedding the content duplicates tokens the model already holds, wastes context window space, and causes linear prompt growth on re-iteration of the refine step (each iteration re-inlines the previous analysis).

Two distinct sub-problems must be solved independently:
1. **Token waste**: large artefacts (spec ≈ 2 000–5 000 tokens, refined analysis ≈ 1 500–3 000 tokens) are embedded even when the model already holds them. On refine re-iteration, prompt size grows with each pass.
2. **Isolation assumption**: the current design has no mechanism to express "use context already in the conversation." The fix is a `ContextMode` field that lets templates switch between inlining and emitting file-read instructions, without requiring session-detection logic.

**User need:** developers running the full workflow in one IDE session should not consume context window on content the model already has. The `--to-file` debug path and non-IDE usage (piping to an API) must produce fully self-contained prompts unchanged.

**Dependency:** Issue #24 is declared as a prerequisite. Confirmed in git log: PR #38 (`harden-review-prompts`) is the most recent merged PR; the main branch must be checked to confirm #24 is also merged before starting.

---

## 2. Technical Analysis

### Affected components

| Package | File | Change |
|---|---|---|
| `internal/prompt` | `engine.go` | Add 2 fields + 2 constants to `TemplateData` |
| `internal/plan` | `refine.go` | Add `contextMode string` to 3 builder function signatures |
| `internal/review` | `review.go` | Add `contextMode string` to 2 builder function signatures |
| `internal/cli` | `plan.go` | Wire `contextMode` from `toFile` flag in `runPlanRefine`, `runPlanSpec` |
| `internal/cli` | `review.go` | Wire `contextMode`; write `diff.md` snapshot in reference mode |
| `internal/cli` | `start.go` | Wire `contextMode`; switch KB load strategy |
| `internal/cli` | `knowledge_cmd.go` | Wire `contextMode` in `buildBranchLessonData()` |
| `internal/context` | `context.go` | Add `WarnMissingPredecessors` method |
| `internal/prompt/templates` | 6 `.md.tmpl` files | Add `{{if eq .ContextMode "inline"}}` conditionals |

`scoring/scoring.go` and `scoring/judge_refine.md.tmpl` are **not changed** — see §F below.

### Key technical decisions

**A. ContextMode values and default behaviour**

Two string constants defined in `internal/prompt/engine.go`:
```go
const (
    ContextModeInline    = "inline"
    ContextModeReference = "reference"
)
```

Templates use `{{if eq .ContextMode "inline"}}` to branch. The zero-value `""` falls into the `{{else}}` (reference) branch — this is acceptable because all CLI callers explicitly set the mode, and the scoring engine is explicitly excluded (§F).

CLI rule: `contextMode = ContextModeInline` when `toFile == true`; `contextMode = ContextModeReference` otherwise. Identical semantics: `--to-file` means "produce a self-contained debugging artefact."

**B. BranchDir derivation**

`context.Load()` (`context.go:41`) already sets `ctx.ContextDir = filepath.Join(root, config.QodeDir, "branches", branch)`. This is an absolute path. Builder functions set `BranchDir: ctx.ContextDir` in `TemplateData` — no new computation. Templates reference:

| Artefact | Path |
|---|---|
| Ticket | `{{.BranchDir}}/context/ticket.md` |
| Notes | `{{.BranchDir}}/context/notes.md` |
| Refined analysis | `{{.BranchDir}}/refined-analysis.md` |
| Spec | `{{.BranchDir}}/spec.md` |
| Diff snapshot | `{{.BranchDir}}/diff.md` (written by `runReview`) |

**C. Builder function signature changes**

Add `contextMode string` as the last parameter to each builder. In reference mode, explicitly omit large content fields rather than setting them empty — use a conditional assignment pattern:

```go
// plan/refine.go — BuildSpecPromptWithOutput
analysis := ""
if contextMode == ContextModeInline {
    analysis = ctx.RefinedAnalysis
}
data := prompt.TemplateData{
    Project:     cfg.Project,
    Layers:      cfg.Layers(),
    Branch:      ctx.Branch,
    Analysis:    analysis,
    BranchDir:   ctx.ContextDir,
    ContextMode: contextMode,
    OutputPath:  outputPath,
}
```

Apply the same pattern for `Spec` in `BuildStartPrompt` and `BuildCodePrompt`, and `Diff` in `BuildCodePrompt`/`BuildSecurityPrompt`.

Zero-arity wrappers (`BuildRefinePrompt` line 24, `BuildSpecPrompt` line 74) pass `ContextModeInline` to preserve existing behaviour for non-CLI callers and tests.

Updated signatures:
```go
// plan/refine.go
func BuildRefinePromptWithOutput(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, ticketURL string, iteration int, outputPath, contextMode string) (*RefineOutput, error)
func BuildSpecPromptWithOutput(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, outputPath, contextMode string) (string, error)
func BuildStartPrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, kb, contextMode string) (string, error)

// review/review.go
func BuildCodePrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, diff, outputPath, contextMode string) (string, error)
func BuildSecurityPrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, diff, outputPath, contextMode string) (string, error)
```

**D. Diff snapshot**

`runReview` (`cli/review.go:65`) calls `git.DiffFromBase(root, "")` to compute `diff`. In reference mode, write to `branchDir/diff.md` before building the prompt:

```go
branchDir := filepath.Join(root, config.QodeDir, "branches", branch) // line 84, already computed
if contextMode == prompt.ContextModeReference {
    if err := os.WriteFile(filepath.Join(branchDir, "diff.md"), []byte(diff), 0644); err != nil {
        return fmt.Errorf("saving diff snapshot: %w", err)
    }
}
```

Side benefit: `knowledge/add-branch.md.tmpl` can reference this file without re-computing the diff.

**E. Knowledge base in reference mode**

`knowledge.List(root, cfg)` (`knowledge.go:45`) signature confirmed: returns `([]string, error)` where paths are absolute (`filepath.Join(root, p)` for config paths; `filepath.Join(root, config.QodeDir, "knowledge")` for auto-discovered files). In reference mode, `cli/start.go` calls `List` instead of `Load` and formats relative paths:

```go
contextMode := prompt.ContextModeReference
if toFile {
    contextMode = prompt.ContextModeInline
}

var kb string
if toFile {
    kb, err = knowledge.Load(root, cfg)
    if err != nil && flagVerbose {
        fmt.Fprintf(os.Stderr, "Warning: knowledge base: %v\n", err)
    }
} else {
    paths, listErr := knowledge.List(root, cfg)
    if listErr != nil && flagVerbose {
        fmt.Fprintf(os.Stderr, "Warning: knowledge base: %v\n", listErr)
    }
    refs := make([]string, 0, len(paths))
    for _, p := range paths {
        rel, _ := filepath.Rel(root, p)
        refs = append(refs, "- "+rel)
    }
    if len(refs) > 0 {
        kb = "Read the following knowledge base files:\n" + strings.Join(refs, "\n")
    }
}
```

Error handling mirrors the existing verbose-flag pattern at `start.go:46-48`. The `{{if .KB}}` guard in `start/base.md.tmpl` (line 18) handles the empty-KB case (no knowledge files present) for both modes without a template change.

**F. Scoring engine — no change required**

`scoring.Engine.BuildJudgePrompt()` (`scoring.go:60-66`) constructs `prompt.TemplateData{Analysis: workerOutput}` with all other fields at zero value. `scoring/judge_refine.md.tmpl` uses only `{{.Analysis}}` — it has no `ContextMode` conditionals and will not receive them. Zero-value `ContextMode` in the judge template has no effect. **No change to `scoring/scoring.go` or `scoring/judge_refine.md.tmpl` is required.**

**G. `knowledge/add-context.md.tmpl` — excluded**

This template extracts lessons from the current session's conversation history, not from branch context files. It does not inline cross-step artefacts. No changes required.

**H. Patterns**

- Existing `{{if .OutputPath}}` conditional (refine/base.md.tmpl:81, spec/base.md.tmpl:68) is the established template mode-switch precedent. Apply `{{if eq .ContextMode "inline"}}` with identical syntax.
- Builder functions use named parameters, not option structs. Continue this pattern.
- All `os.WriteFile` calls for the diff snapshot return errors via `fmt.Errorf("...: %w", err)`.
- Knowledge error handling follows the existing `flagVerbose` pattern from `start.go:46`.

### Dependencies

- Issue #24 must be merged. Verify with `git log --oneline main | grep '#24'` before Task 1.
- `ctx.ContextDir` is the authoritative source for `BranchDir`; if this field is renamed, update builder functions.
- `knowledge.List(root, cfg)` confirmed to return absolute paths — relative formatting via `filepath.Rel` is correct.

---

## 3. Risk & Edge Cases

**R1 — Zero-value ContextMode breaks callers that expect inline**

`TemplateData{Analysis: workerOutput}` with unset `ContextMode` will render the `{{else}}` (reference) branch. The scoring engine is safe because `judge_refine.md.tmpl` has no ContextMode conditional. However, any test that calls a builder function without passing `contextMode` and checks the rendered output will break. Mitigation: zero-arity wrappers `BuildRefinePrompt` and `BuildSpecPrompt` pass `ContextModeInline` explicitly; update `plan/refine_test.go` wherever builder functions are called directly.

**R2 — Missing predecessor files produce silent AI failure in reference mode**

If `refined-analysis.md` is absent when `qode plan spec` runs in reference mode, the template emits `Read ... from {{.BranchDir}}/refined-analysis.md` pointing to a non-existent file. The AI will either error or hallucinate. Mitigation:

`runPlanSpec` already calls `ctx.HasRefinedAnalysis()` and returns an error (plan.go:139). Extend to `runStart` (warn) and `runReview` (warn) via `WarnMissingPredecessors`:

```go
// internal/context/context.go — new method
func (c *Context) WarnMissingPredecessors(step string, w io.Writer) {
    switch step {
    case "spec":
        if !c.HasRefinedAnalysis() {
            fmt.Fprintln(w, "Warning: no refined-analysis.md — spec quality may be low. Run 'qode plan refine' first.")
        }
    case "start":
        if !c.HasSpec() {
            fmt.Fprintln(w, "Warning: no spec.md — run 'qode plan spec' first.")
        }
    case "review":
        if !c.HasSpec() {
            fmt.Fprintln(w, "Warning: no spec.md — code review proceeds without spec context.")
        }
    }
}
```

Import `io` in `context.go` (currently not imported).

**R3 — Refine template: first iteration has no previous analysis file**

On the first refine iteration, `refined-analysis.md` does not exist (or contains only the placeholder text). The existing template guards with `{{if .Analysis}}` — in inline mode this works because `ctx.RefinedAnalysis` is empty on the first pass. In reference mode, the template must not emit a read instruction for a non-existent file. The template conditional must check `ContextMode` only when there actually was a previous iteration:

```
{{if eq .ContextMode "inline"}}
{{if .Analysis}}
## Previous Analysis (Iteration N-1)
{{.Analysis}}
{{end}}
{{else if .BranchDir}}
{{/* Only emit reference if a previous iteration could exist (Iteration > 1) */}}
If you have already refined this analysis in a previous iteration, read
`{{.BranchDir}}/refined-analysis.md` for context. Otherwise begin fresh.
{{end}}
```

The "begin fresh vs. continue" phrasing avoids a hard file dependency on the first pass.

**R4 — Diff write race condition**

Two concurrent `qode review code` runs on the same branch will overwrite each other's `diff.md`. Acceptable: the review workflow is human-driven and sequential. Document as a known limitation.

**R5 — Diff snapshot stale if commits land between generation and AI execution**

Diff is frozen at `git.DiffFromBase` call time; matches existing inline behaviour. Acceptable.

**R6 — KB paths non-portable across machines/containers**

`knowledge.List()` returns absolute paths. `filepath.Rel(root, p)` converts to project-relative paths (e.g. `.qode/knowledge/lessons/foo.md`). If `root` and `p` are on different volumes, `filepath.Rel` returns `p` unchanged (absolute). Mitigation: this edge case (cross-volume KB paths) is not a supported configuration; document as limitation.

**R7 — Local template overrides in `.qode/prompts/`**

`engine.loadTemplate()` (engine.go:107) prefers local overrides. If a project has a custom `start/base.md.tmpl` without `ContextMode` conditionals, reference mode silently falls back to inline behaviour (the `{{if eq .ContextMode "inline"}}` block is absent, so the template uses whatever it was written to use). Mitigation: release notes must document the required template changes for projects with overrides.

**R8 — Tests for changed builder signatures**

`internal/plan/refine_test.go` calls builder functions directly. Adding `contextMode string` parameter breaks compilation. All test call sites must be updated to pass `prompt.ContextModeInline` to preserve existing test semantics.

**Security**: `BranchDir` is derived from `config.QodeDir` (constant `.qode`) and `git.CurrentBranch()` (reads `.git/HEAD`, not user input). No path traversal risk. No new external inputs.

**Performance**: in reference mode, `ctx.RefinedAnalysis` and `ctx.Spec` are still loaded by `context.Load()` (they are always read from disk). An optional optimisation — skip loading these fields in reference mode — is out of scope for this issue; the memory savings are negligible for CLI usage.

---

## 4. Completeness Check

### Acceptance criteria

| # | Criterion | Source |
|---|---|---|
| AC1 | `TemplateData` gains `ContextMode string` and `BranchDir string` | Ticket §5 |
| AC2 | `ContextModeInline = "inline"` and `ContextModeReference = "reference"` constants defined | Implicit (CLAUDE.md: no magic strings) |
| AC3 | `ContextMode = ContextModeReference` when `--to-file` absent; `ContextModeInline` when present | Ticket §2 |
| AC4 | `refine/base.md.tmpl`: conditional for `.Ticket`, `.Notes`, `.Analysis` | Ticket §4 |
| AC5 | `spec/base.md.tmpl`: conditional for `.Analysis` | Ticket §4 |
| AC6 | `start/base.md.tmpl`: conditional for `.Spec` | Ticket §4 |
| AC7 | `review/code.md.tmpl`: conditional for `.Spec`, `.Diff` | Ticket §4 |
| AC8 | `review/security.md.tmpl`: conditional for `.Diff` | Ticket §4 |
| AC9 | `knowledge/add-branch.md.tmpl`: conditional for `.Analysis`, `.Spec`, `.Diff` | Ticket §4 |
| AC10 | `runReview` writes `diff.md` to branch dir in reference mode before building prompt | Derived from §1, §4 |
| AC11 | `runStart` passes KB path references to `BuildStartPrompt` in reference mode | Ticket §1 |
| AC12 | `context.go` gains `WarnMissingPredecessors(step, w)` called by start and review CLI | Ticket §3 |
| AC13 | Zero-arity builder wrappers and tests updated to pass `ContextModeInline` | Derived (backward compat) |
| AC14 | `--to-file` produces fully self-contained inline prompts unchanged | Ticket §2 |
| AC15 | `scoring/scoring.go` and `judge_refine.md.tmpl` unchanged | Derived (§F) |
| AC16 | `knowledge/add-context.md.tmpl` unchanged | Derived (§G) |

### Implicit requirements not stated in the ticket

- Named constants for ContextMode values (AC2 above).
- `WarnMissingPredecessors` needs `import "io"` added to `context.go`.
- The refine template's first-iteration case must not emit a hard file reference (Risk R3).
- KB error handling in `start.go` must mirror the existing `flagVerbose` pattern.
- Test files (`plan/refine_test.go`) must be updated for the new signature.
- `buildBranchLessonData()` in `cli/knowledge_cmd.go` must pass `contextMode` and `BranchDir` to `knowledge/add-branch.md.tmpl`.

### Out of scope

- Auto-detecting whether the CLI is running inside an IDE session.
- Truncating or summarising inlined content.
- Removing inline mode.
- Modifying the scoring engine's two-pass architecture.
- Lazy-loading (skipping `context.Load` fields) in reference mode.
- Adding `OutputPath` to `BuildStartPrompt`.
- Modifying `knowledge/add-context.md.tmpl`.

---

## 5. Actionable Implementation Plan

Each task is one commit. Tasks 1–8 must be sequential (each depends on the previous compiling). Tasks 9–14 (templates) can be done in a single commit.

### Task 1 — `internal/prompt/engine.go`: add constants and struct fields

```go
const (
    ContextModeInline    = "inline"
    ContextModeReference = "reference"
)

// In TemplateData, add after OutputPath:
ContextMode string // ContextModeInline | ContextModeReference; controls inline vs reference rendering
BranchDir   string // absolute path to .qode/branches/<branch>/; used by templates in reference mode
```

No callers change in this commit; project still compiles.

### Task 2 — `internal/plan/refine.go`: update builder signatures

- `BuildRefinePromptWithOutput`: add `contextMode string` (last param). Set `ContextMode: contextMode`, `BranchDir: ctx.ContextDir`. Conditionally set `Analysis`:
  ```go
  analysis := ""
  if contextMode == prompt.ContextModeInline {
      analysis = ctx.RefinedAnalysis
  }
  ```
- `BuildSpecPromptWithOutput`: same pattern for `Analysis`.
- `BuildStartPrompt`: same pattern for `Spec`; `KB` is already set by the caller based on mode.
- `BuildRefinePrompt` (line 24, zero-arity): pass `prompt.ContextModeInline`.
- `BuildSpecPrompt` (line 74, zero-arity): pass `prompt.ContextModeInline`.

### Task 3 — `internal/review/review.go`: update builder signatures

- `BuildCodePrompt`: add `contextMode string`. Set `ContextMode`, `BranchDir: ctx.ContextDir`. Conditionally set `Spec` and `Diff`:
  ```go
  spec, diff := "", ""
  if contextMode == prompt.ContextModeInline {
      spec, diff = ctx.Spec, diffArg
  }
  ```
- `BuildSecurityPrompt`: same for `Diff` only.

### Task 4 — `internal/plan/refine_test.go`: update test call sites

For every call to `BuildRefinePromptWithOutput`, `BuildSpecPromptWithOutput`, `BuildStartPrompt`: add `prompt.ContextModeInline` as the last argument. This keeps all tests green while the new parameter is present.

### Task 5 — `internal/cli/plan.go`: wire ContextMode

In `runPlanRefine` (line 70):
```go
contextMode := prompt.ContextModeReference
if toFile {
    contextMode = prompt.ContextModeInline
}
```
Pass to `BuildRefinePromptWithOutput`. Apply same pattern in `runPlanSpec` (line 120) for `BuildSpecPromptWithOutput`.

### Task 6 — `internal/cli/start.go`: wire ContextMode and KB reference

Compute `contextMode` from `toFile`. Switch KB loading strategy as shown in §E above. Call `ctx.WarnMissingPredecessors("start", os.Stderr)` after `context.Load`. Pass `contextMode` to `BuildStartPrompt`.

### Task 7 — `internal/cli/review.go`: wire ContextMode and diff snapshot

Compute `contextMode` from `toFile`. After `git.DiffFromBase` (line 65), write `diff.md` in reference mode as shown in §D above. Pass `contextMode` to `BuildCodePrompt`/`BuildSecurityPrompt`. Call `ctx.WarnMissingPredecessors("review", os.Stderr)`.

### Task 8 — `internal/context/context.go`: add `WarnMissingPredecessors`

Add `import "io"`. Add method as specified in Risk R2. No changes to `Load()` or existing methods.

### Task 9 — `internal/cli/knowledge_cmd.go`: wire ContextMode in `buildBranchLessonData`

Add `ContextMode: prompt.ContextModeReference` and `BranchDir: ctx.ContextDir` to the `TemplateData` constructed in `buildBranchLessonData()`. Default to reference mode (no `--to-file` flag exists for knowledge commands).

### Tasks 10–15 — Templates (one commit)

**Task 10 — `refine/base.md.tmpl`**: Replace Ticket, Notes, Analysis blocks with `{{if eq .ContextMode "inline"}}` conditionals. For Analysis specifically, use the soft first-iteration wording from Risk R3 in the `{{else}}` branch.

**Task 11 — `spec/base.md.tmpl`**: Replace unconditional `{{.Analysis}}` (line 15):
```
{{if eq .ContextMode "inline"}}
{{.Analysis}}
{{else}}
Read the refined analysis from `{{.BranchDir}}/refined-analysis.md` before generating the spec.
{{end}}
```

**Task 12 — `start/base.md.tmpl`**: Replace unconditional `{{.Spec}}` (line 16):
```
{{if eq .ContextMode "inline"}}
{{.Spec}}
{{else}}
Read the technical specification from `{{.BranchDir}}/spec.md` before beginning implementation.
{{end}}
```
`{{if .KB}}{{.KB}}{{end}}` (line 18) requires no change.

**Task 13 — `review/code.md.tmpl`**: Replace the `{{if .Spec}}` block (line 25) and diff block (line 33):
```
{{if eq .ContextMode "inline"}}
{{if .Spec}}
## Spec Summary

{{.Spec}}
{{end}}
{{else}}
## Spec Summary

Read the technical specification from `{{.BranchDir}}/spec.md`.
{{end}}

## Changes to Review

{{if eq .ContextMode "inline"}}
` + "```diff" + `
{{.Diff}}
` + "```" + `
{{else}}
Read the diff from `{{.BranchDir}}/diff.md`.
{{end}}
```

**Task 14 — `review/security.md.tmpl`**: Replace the diff block (line 27) with the same inline/reference conditional as Task 13.

**Task 15 — `knowledge/add-branch.md.tmpl`**: Apply conditionals for `.Analysis`, `.Spec`, `.Diff` fields using the same pattern. Reference paths: `{{.BranchDir}}/refined-analysis.md`, `{{.BranchDir}}/spec.md`, `{{.BranchDir}}/diff.md`.

---

### Implementation order summary

```
Task 1   engine.go            — constants + struct fields (compiles, no callers affected)
Task 2   plan/refine.go       — builder signatures, conditional field population
Task 3   review/review.go     — builder signatures, conditional field population
Task 4   plan/refine_test.go  — fix compilation of tests
Task 5   cli/plan.go          — wire contextMode
Task 6   cli/start.go         — wire contextMode + KB reference format
Task 7   cli/review.go        — wire contextMode + diff snapshot
Task 8   context/context.go   — WarnMissingPredecessors
Task 9   knowledge_cmd.go     — wire contextMode in buildBranchLessonData
Tasks 10-15  templates         — all conditional blocks (one commit)
```

**Prerequisite**: run `git log --oneline main | grep -i '#24'` to confirm issue #24 is merged before beginning Task 1.
