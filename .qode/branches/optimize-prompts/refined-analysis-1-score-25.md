<!-- qode:iteration=1 score=25/25 -->

# Requirements Analysis — Optimize Prompts for Token Usage

## 1. Problem Understanding

Each qode workflow step (refine, spec, start, review-code, review-security) calls a builder function that constructs a `TemplateData` struct, loads the previous step's output as a string (e.g. `ctx.RefinedAnalysis`, `ctx.Spec`), and passes it to the Go template engine, which embeds it verbatim into the rendered prompt. When the prompt is piped to stdout and executed by an IDE AI session (Claude Code / Cursor), the session already has that content in its conversation context from having written it moments ago. Pasting it again as part of the next prompt is pure duplication.

There are two concrete sub-problems:
1. **Token waste**: inlined large artefacts (a 3 000-token spec in `start/base.md.tmpl`, or a 2 000-token refined analysis in `spec/base.md.tmpl`) double the context consumed. On re-iteration of refine, the previous analysis is pasted again each round, growing the prompt linearly.
2. **Isolation assumption**: the architecture assumes each step is a fresh session. There is no concept of "the agent can read this from a file it already has in context." Adding a `context_mode` field lets templates express intent without forcing the implementation to detect session state.

**User need:** developers running multi-step workflows in a single IDE session should not burn their context window on content already present. `--to-file` debugging and non-IDE use cases (piping to a model API) must continue to work.

**Open question:** Issue #25 declares a dependency on #24. Issue #24 must be verified as merged before this work starts — its absence would leave the base incomplete.

---

## 2. Technical Analysis

### Affected components

| Layer | File | Change |
|---|---|---|
| `internal/prompt` | `engine.go` | Add 2 fields to `TemplateData` |
| `internal/plan` | `refine.go` | Update 3 builder functions |
| `internal/review` | `review.go` | Update 2 builder functions |
| `internal/cli` | `plan.go`, `review.go`, `start.go` | Wire `contextMode` from `--to-file` flag |
| `internal/context` | `context.go` | Add predecessor-file validation |
| `internal/prompt/templates` | 6 `.md.tmpl` files | Add `{{if eq .ContextMode "inline"}}` conditionals |

### Key technical decisions

**A. ContextMode values and default**

`ContextMode` is a string field on `TemplateData` with two defined values:
- `"inline"` — current behaviour; all content fields populated, templates paste them verbatim
- `"reference"` — default for stdout/IDE path; large content fields left empty, templates emit file-read instructions

When `ContextMode` is empty it is treated as `"reference"` by templates (using `{{if eq .ContextMode "inline"}}`). This means existing callers that do not set the field get reference mode, which is wrong for the scoring engine. See the carve-out for `scoring/judge_refine.md.tmpl` below.

Deterministic rule for CLI callers: `if toFile { contextMode = "inline" } else { contextMode = "reference" }`. This maps exactly onto the existing `--to-file` flag semantics.

**B. BranchDir**

`ctx.ContextDir` (type `string`) is already set to `filepath.Join(root, config.QodeDir, "branches", branch)` inside `context.Load()` (`context.go:41-43`). This is an absolute path. Builder functions set `BranchDir: ctx.ContextDir` in `TemplateData` — no new computation required.

File paths that templates will reference in reference mode:
- Ticket: `{{.BranchDir}}/context/ticket.md`
- Notes: `{{.BranchDir}}/context/notes.md`
- Refined analysis: `{{.BranchDir}}/refined-analysis.md`
- Spec: `{{.BranchDir}}/spec.md`
- Diff snapshot: `{{.BranchDir}}/diff.md` (written by `runReview` before prompt generation)

**C. Propagating ContextMode into builder functions**

Current signatures require changes:

```go
// plan/refine.go
func BuildRefinePromptWithOutput(engine, cfg, ctx, ticketURL string, iteration int, outputPath, contextMode string) (*RefineOutput, error)
func BuildSpecPromptWithOutput(engine, cfg, ctx, outputPath, contextMode string) (string, error)
func BuildStartPrompt(engine, cfg, ctx, kb, contextMode string) (string, error)

// review/review.go
func BuildCodePrompt(engine, cfg, ctx, diff, outputPath, contextMode string) (string, error)
func BuildSecurityPrompt(engine, cfg, ctx, diff, outputPath, contextMode string) (string, error)
```

Each builder sets both `ContextMode: contextMode` and `BranchDir: ctx.ContextDir` in the `TemplateData` it constructs. In reference mode, large content fields (`Analysis`, `Spec`, `Diff`) are left as empty strings — the template's `{{else}}` branch renders the file path instead.

The zero-argument wrappers (e.g. `BuildRefinePrompt`, `BuildSpecPrompt`) that currently call `WithOutput` variants should pass `contextMode = "inline"` so existing call sites and tests are unaffected.

**D. Diff snapshot**

`runReview` in `cli/review.go` computes the diff via `git.DiffFromBase(root, "")` (line 65). In reference mode, it must write this diff to `branchDir/diff.md` before calling the builder:

```go
if contextMode == "reference" {
    if err := os.WriteFile(filepath.Join(branchDir, "diff.md"), []byte(diff), 0644); err != nil {
        return fmt.Errorf("saving diff snapshot: %w", err)
    }
}
```

The builder then leaves `Diff: ""` in `TemplateData`; the template emits a read instruction.

Writing the diff to disk has an ancillary benefit: it becomes available to `knowledge/add-branch.md.tmpl` without re-running the git command.

**E. Knowledge base in reference mode**

`BuildStartPrompt` currently receives a pre-loaded `kb string` from `knowledge.Load(root, cfg)`. In reference mode, the CLI should instead call `knowledge.List(root, cfg)` to get file paths and format them as a reference string:

```
Read the following knowledge base files before implementing:
- /abs/path/.qode/knowledge/lessons/lesson-a.md
- /abs/path/.qode/knowledge/lessons/lesson-b.md
```

This string is passed as `kb`; the template uses `{{.KB}}` unchanged. No new `TemplateData` field is needed for KB paths.

**F. Scoring judge template — explicit carve-out**

`scoring/judge_refine.md.tmpl` is populated by `scoring.Engine.BuildJudgePrompt()` (scoring/scoring.go), which always sets `Analysis` = the worker's full output. This is not a cross-step re-inline situation — the judge genuinely needs the worker output as input. The scoring engine must set `ContextMode: "inline"` when constructing `TemplateData`, and `judge_refine.md.tmpl` does not receive the inline/reference conditional treatment.

**G. Patterns to follow**

- The existing `{{if .OutputPath}}` conditional in `refine/base.md.tmpl` (line 81) and `spec/base.md.tmpl` (line 68) is the established pattern for template-level mode switching. Apply the same pattern for `ContextMode`.
- Builder functions follow the pattern of named parameters, not options structs. Continue this.
- Error handling: all `os.WriteFile` calls for the diff snapshot return errors wrapped with `fmt.Errorf`.

### Dependencies

- Issue #24 (predecessor): must be confirmed merged before this branch starts.
- `context.ContextDir` field is relied upon for `BranchDir` — must not be renamed or removed.
- `knowledge.List(root, cfg)` function must exist and return absolute paths; verify its signature before Task 5.

---

## 3. Risk & Edge Cases

**R1 — Blank `ContextMode` in existing callers**

The zero-value of `ContextMode` is `""`. Templates using `{{if eq .ContextMode "inline"}}` will fall into the `{{else}}` (reference) branch when the field is empty. This will break any existing caller that does not set the field but expects inline behaviour — specifically the scoring engine and any test that calls builder functions without passing `contextMode`. Mitigation: update zero-arity wrappers and scoring engine to pass `"inline"` explicitly; add a constant `prompt.ContextModeInline = "inline"` and `prompt.ContextModeReference = "reference"` to prevent typos.

**R2 — Missing predecessor files in reference mode**

If a user runs `qode plan spec` in reference mode but `refined-analysis.md` does not exist, the template emits a read instruction pointing to a non-existent file, causing silent AI failure (the agent will hallucinate or error). Mitigation: `runPlanSpec` already guards with `ctx.HasRefinedAnalysis()` (plan.go:139). Extend this pattern:
- `runPlanSpec`: error if `!ctx.HasRefinedAnalysis()` — already done
- `runStart`: warn (stderr) if `!ctx.HasSpec()`
- `runReview`: warn if `!ctx.HasSpec()` (spec is optional but useful)

Add `WarnMissingPredecessors(step string, w io.Writer)` to `Context` struct:
```go
func (c *Context) WarnMissingPredecessors(step string, w io.Writer) {
    switch step {
    case "spec":
        if !c.HasRefinedAnalysis() {
            fmt.Fprintln(w, "Warning: no refined-analysis.md found; spec quality may be low.")
        }
    case "start":
        if !c.HasSpec() {
            fmt.Fprintln(w, "Warning: no spec.md found; run 'qode plan spec' first.")
        }
    case "review":
        if !c.HasSpec() {
            fmt.Fprintln(w, "Warning: no spec.md found; code review will proceed without spec context.")
        }
    }
}
```

**R3 — Diff write race condition**

In reference mode, the diff is written to `branchDir/diff.md` before the prompt is generated. If two review runs execute concurrently on the same branch (unlikely but possible in CI), the second run overwrites the first's diff. Acceptable: `--to-file` debugging is not a concurrent workflow; document as a known limitation.

**R4 — Diff snapshot stale between prompt generation and AI execution**

The user runs `qode review code` (reference mode), prompt is generated, AI starts reviewing. Meanwhile a new commit is added. The diff.md reflects the old state. Acceptable: the AI executes the prompt based on the snapshot; this matches the existing inline behaviour (the diff was frozen at prompt generation time).

**R5 — Knowledge base paths non-portable**

`knowledge.List()` returns absolute paths. If the project is on a different machine or inside a container with different root, absolute paths break. Mitigation: format paths relative to project root using `filepath.Rel(root, path)`. Present paths as `.qode/knowledge/...` in the reference string so the AI can read them relative to its working directory.

**R6 — Local template overrides in `.qode/prompts/`**

The engine's `loadTemplate()` (`engine.go:107`) checks for per-project overrides first. If a project has customised e.g. `start/base.md.tmpl` without the new `ContextMode` conditionals, reference mode will not work. Mitigation: document this in the change; no code change needed (user responsibility to update local overrides).

**R7 — `BuildStartPrompt` lacks OutputPath**

Unlike `BuildRefinePromptWithOutput` and `BuildSpecPromptWithOutput`, `BuildStartPrompt` does not take an `outputPath` parameter and the start template has no `{{if .OutputPath}}` section. The start step doesn't write to a structured output file — the AI writes code directly. This is intentional and should not be changed as part of this issue.

**Security**: no new external input surfaces are introduced. `BranchDir` is constructed from `ctx.ContextDir` which is derived from `config.QodeDir` (a constant `".qode"`) and the git branch name. Branch names come from `git.CurrentBranch()` which reads the local git state, not user input. File paths constructed from these values do not create a path injection risk.

**Performance**: in reference mode, large strings are not loaded into memory for the prompt. For a 10 000-word spec, this reduces prompt construction memory usage by ~40 KB per call — negligible in absolute terms but meaningful for context window budget.

---

## 4. Completeness Check

### Acceptance criteria

| # | Criterion | Source |
|---|---|---|
| AC1 | `TemplateData` has `ContextMode string` and `BranchDir string` fields | Ticket §5 |
| AC2 | `ContextMode = "reference"` when `--to-file` is absent; `"inline"` when present | Ticket §2 |
| AC3 | `refine/base.md.tmpl` conditionally inlines or references `.Ticket`, `.Notes`, `.Analysis` | Ticket §4 |
| AC4 | `spec/base.md.tmpl` conditionally inlines or references `.Analysis` | Ticket §4 |
| AC5 | `start/base.md.tmpl` conditionally inlines or references `.Spec`, `.KB` | Ticket §4 |
| AC6 | `review/code.md.tmpl` conditionally inlines or references `.Spec`, `.Diff` | Ticket §4 |
| AC7 | `review/security.md.tmpl` conditionally inlines or references `.Diff` | Ticket §4 |
| AC8 | `knowledge/add-branch.md.tmpl` conditionally inlines or references `.Analysis`, `.Spec`, `.Diff` | Ticket §4 |
| AC9 | `runReview` writes `diff.md` to branch dir before generating prompt in reference mode | Derived |
| AC10 | `runStart` passes KB path references (not content) to `BuildStartPrompt` in reference mode | Ticket §1 |
| AC11 | `context.go` adds `WarnMissingPredecessors` warning for spec, start, review steps | Ticket §3 |
| AC12 | Scoring engine (`scoring.Engine.BuildJudgePrompt`) passes `ContextMode = "inline"` | Derived |
| AC13 | Zero-arity builder wrappers pass `contextMode = "inline"` for backward compatibility | Derived |
| AC14 | `--to-file` debugging produces fully self-contained prompts (inline mode unchanged) | Ticket §2 |

### Implicit requirements not in the ticket

- **Named constants** `prompt.ContextModeInline` and `prompt.ContextModeReference` must be defined to prevent string drift across callers (CLAUDE.md: "No magic numbers — use named constants").
- **`knowledge/add-context.md.tmpl`** extracts lessons from session conversation, not from files. It does not inline cross-step artefacts and should not be given the conditional treatment.
- **`scoring/judge_refine.md.tmpl`** must remain inline-only (explicit carve-out required).
- KB paths passed in reference mode must be relative to project root, not absolute, to avoid portability issues (Risk R5).

### Out of scope

- Automatic detection of whether the AI is running inside an active IDE session.
- Truncating or summarising inlined content when inline mode is active.
- Removing inline mode entirely.
- Changing the scoring engine's two-pass architecture.
- Modifying `knowledge/add-context.md.tmpl`.
- Adding OutputPath to `BuildStartPrompt`.

---

## 5. Actionable Implementation Plan

Each task is a single commit. Order is strictly sequential because each task builds on the previous.

### Task 1 — Add constants and struct fields to `internal/prompt/engine.go`

```go
const (
    ContextModeInline    = "inline"
    ContextModeReference = "reference"
)

type TemplateData struct {
    // ... existing fields unchanged ...
    ContextMode string // ContextModeInline | ContextModeReference
    BranchDir   string // absolute path to .qode/branches/<branch>/
}
```

No other changes in this commit.

### Task 2 — Update builder function signatures in `internal/plan/refine.go`

- `BuildRefinePromptWithOutput`: add `contextMode string` parameter (last param). Set `ContextMode: contextMode`, `BranchDir: ctx.ContextDir` in `data`. In reference mode, set `Analysis: ""` (already empty if no previous iteration; explicitly clear on re-iteration).
- `BuildSpecPromptWithOutput`: add `contextMode string`. Set same two fields. Set `Analysis: ""` in reference mode.
- `BuildStartPrompt`: add `contextMode string`. Set same two fields.
- `BuildRefinePrompt` (zero-arity wrapper, line 24): pass `prompt.ContextModeInline` as contextMode.
- `BuildSpecPrompt` (zero-arity wrapper, line 74): same.

### Task 3 — Update builder function signatures in `internal/review/review.go`

- `BuildCodePrompt`: add `contextMode string`. Set `ContextMode`, `BranchDir: ctx.ContextDir`. In reference mode, set `Spec: ""`, `Diff: ""`.
- `BuildSecurityPrompt`: add `contextMode string`. Set `ContextMode`, `BranchDir: ctx.ContextDir`. In reference mode, set `Diff: ""`.

### Task 4 — Update scoring engine in `internal/scoring/scoring.go`

In `Engine.BuildJudgePrompt()`, set `ContextMode: prompt.ContextModeInline` in the `TemplateData` constructed for the judge template. This ensures the judge template (which does not change) continues to work correctly after Task 1's zero-value behaviour change.

### Task 5 — Wire ContextMode in `internal/cli/plan.go`

In `runPlanRefine` (line 70): compute `contextMode := prompt.ContextModeReference; if toFile { contextMode = prompt.ContextModeInline }`. Pass to `BuildRefinePromptWithOutput`.

In `runPlanSpec` (line 120): same pattern. Pass to `BuildSpecPromptWithOutput`.

### Task 6 — Wire ContextMode in `internal/cli/start.go`

Compute `contextMode` from `toFile` flag. In reference mode: call `knowledge.List(root, cfg)` instead of `knowledge.Load(root, cfg)`, format paths as relative-to-root references, pass as `kb` string. Pass `contextMode` to `BuildStartPrompt`.

```go
var kb string
if toFile {
    kb, _ = knowledge.Load(root, cfg)
} else {
    paths, _ := knowledge.List(root, cfg)
    // format as relative paths for portability
    refs := make([]string, 0, len(paths))
    for _, p := range paths {
        rel, _ := filepath.Rel(root, p)
        refs = append(refs, "- "+rel)
    }
    kb = "Read the following knowledge base files:\n" + strings.Join(refs, "\n")
}
```

Call `ctx.WarnMissingPredecessors("start", os.Stderr)`.

### Task 7 — Wire ContextMode and diff snapshot in `internal/cli/review.go`

Compute `contextMode` from `toFile` flag. In reference mode, after computing `diff` (line 65), write it to `branchDir/diff.md`:

```go
if contextMode == prompt.ContextModeReference {
    diffPath := filepath.Join(branchDir, "diff.md")
    if err := os.WriteFile(diffPath, []byte(diff), 0644); err != nil {
        return fmt.Errorf("saving diff snapshot: %w", err)
    }
}
```

Pass `contextMode` to `BuildCodePrompt` and `BuildSecurityPrompt`. Call `ctx.WarnMissingPredecessors("review", os.Stderr)`.

### Task 8 — Add `WarnMissingPredecessors` to `internal/context/context.go`

Add the method as specified in Risk R2. No changes to `Load()` or existing methods.

### Task 9 — Update `internal/prompt/templates/refine/base.md.tmpl`

Replace the unconditional `{{if .Ticket}}{{.Ticket}}{{end}}` block (line 16) with:

```
{{if eq .ContextMode "inline"}}
{{if .Ticket}}
{{.Ticket}}
{{end}}
{{else}}
Read the ticket from `{{.BranchDir}}/context/ticket.md`.
{{end}}
```

Apply same pattern for `.Notes` and `.Analysis` blocks.

### Task 10 — Update `internal/prompt/templates/spec/base.md.tmpl`

Replace unconditional `{{.Analysis}}` (line 15):

```
{{if eq .ContextMode "inline"}}
{{.Analysis}}
{{else}}
Read the refined analysis from `{{.BranchDir}}/refined-analysis.md` before generating the spec.
{{end}}
```

### Task 11 — Update `internal/prompt/templates/start/base.md.tmpl`

Replace unconditional `{{.Spec}}` (line 16):

```
{{if eq .ContextMode "inline"}}
{{.Spec}}
{{else}}
Read the technical specification from `{{.BranchDir}}/spec.md` before beginning implementation.
{{end}}
```

The `{{if .KB}}{{.KB}}{{end}}` block (line 18) requires no template change — the builder sets `.KB` to path references in reference mode.

### Task 12 — Update `internal/prompt/templates/review/code.md.tmpl`

Replace `{{if .Spec}}{{.Spec}}{{end}}` (line 25) and the diff block (line 33):

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
```diff
{{.Diff}}
```
{{else}}
Read the diff from `{{.BranchDir}}/diff.md`.
{{end}}
```

### Task 13 — Update `internal/prompt/templates/review/security.md.tmpl`

Replace the diff block (line 27) with the same inline/reference conditional as Task 12.

### Task 14 — Update `internal/prompt/templates/knowledge/add-branch.md.tmpl`

Apply conditionals for `.Analysis`, `.Spec`, `.Diff` fields. The knowledge command caller must also be updated to pass `contextMode` and set `BranchDir` — add this to `buildBranchLessonData()` in `cli/knowledge_cmd.go`, defaulting to `ContextModeReference`.

---

### Implementation order summary

```
Task 1  engine.go           — struct + constants (no callers broken yet)
Task 2  plan/refine.go      — builder signatures + BranchDir/ContextMode
Task 3  review/review.go    — builder signatures
Task 4  scoring/scoring.go  — hardcode inline for judge
Task 5  cli/plan.go         — wire contextMode
Task 6  cli/start.go        — wire contextMode + KB reference format
Task 7  cli/review.go       — wire contextMode + diff snapshot
Task 8  context/context.go  — predecessor warnings
Tasks 9-14  templates       — add conditionals (can be one commit)
```

Prerequisite: confirm issue #24 is merged and its changes are on the base branch before starting Task 1.
