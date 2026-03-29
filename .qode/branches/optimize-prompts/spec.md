> **Design change applied during implementation:** The `ContextMode` constants and inline mode described in this spec were removed after the spec was written. All prompts always use reference mode (file-read instructions). The `--to-file` flag only controls whether the rendered prompt is saved to disk or printed to stdout — it no longer changes prompt content. See `context/notes.md` for the full rationale.

# Technical Specification — Optimize Prompts for Token Usage

**Branch:** optimize-prompts
**Issue:** nqode-io/qode#25
**Depends on:** nqode-io/qode#24

---

## 1. Feature Overview

This change introduces a `ContextMode` rendering mode to qode's prompt pipeline. Currently every workflow step (refine, spec, start, review-code, review-security, knowledge extraction) embeds the full text of previous steps' artefacts verbatim inside the rendered prompt. When these prompts are executed inside a running IDE AI session (Claude Code / Cursor), the session already holds those artefacts in its conversation context — re-embedding them wastes tokens and causes linear context growth on re-iteration. Adding `ContextMode` lets templates emit file-read instructions instead of inlined content when running in an IDE session, while preserving the existing fully self-contained behaviour for `--to-file` debugging and non-IDE usage.

**Business value:** developers using the full multi-step workflow in a single IDE session can complete feature branches without exhausting their context window, reducing interruptions and improving AI output quality on large features.

**Success criteria:**
- All workflow prompts emitted to stdout omit previously-seen large artefacts and instead emit file-path read instructions.
- `--to-file` prompts remain byte-for-byte equivalent to current inline behaviour.
- All existing tests pass.
- `qode check` passes (tests + lint).

---

## 2. Scope

### In scope

- Add `ContextModeInline` / `ContextModeReference` constants and `ContextMode`, `BranchDir` fields to `prompt.TemplateData`
- Update 5 builder functions across `plan/refine.go` and `review/review.go` to accept and propagate `contextMode`
- Wire `contextMode` from the `--to-file` flag in `cli/plan.go`, `cli/review.go`, `cli/start.go`, and `cli/knowledge_cmd.go`
- Write the computed git diff to `<branchDir>/diff.md` in reference mode (before building review prompts)
- Switch knowledge base loading from `knowledge.Load` → `knowledge.List` + relative-path formatting in reference mode
- Add `context.Context.WarnMissingPredecessors(step string, w io.Writer)` called before prompt generation in start and review commands
- Update 6 templates with `{{if eq .ContextMode "inline"}}` conditionals: `refine/base.md.tmpl`, `spec/base.md.tmpl`, `start/base.md.tmpl`, `review/code.md.tmpl`, `review/security.md.tmpl`, `knowledge/add-branch.md.tmpl`
- Update `plan/refine_test.go` call sites for the new builder signatures

### Out of scope

- Auto-detecting whether the CLI runs inside an active IDE session
- Truncating or summarising inlined content in inline mode
- Removing inline mode entirely
- Modifying `scoring/scoring.go` or `scoring/judge_refine.md.tmpl`
- Modifying `knowledge/add-context.md.tmpl`
- Lazy-loading context fields (skipping file reads for reference mode)
- Adding `OutputPath` to `BuildStartPrompt`
- Changing the scoring engine's two-pass architecture

### Assumptions

- Issue #24 is merged to `main` before this branch is rebased. Verify: `git log --oneline main | grep -i '#24'`.
- `ctx.ContextDir` is an absolute path (confirmed: `context.Load` uses `filepath.Join(root, ...)` where `root` comes from `resolveRoot()` which calls `filepath.Abs`).
- `knowledge.List(root, cfg)` returns absolute paths (confirmed from source).
- Cross-volume KB paths (where `filepath.Rel` cannot produce a relative path) are not a supported configuration.

---

## 3. Architecture & Design

### Component diagram

```
CLI layer (cli/*.go)
  │  determines contextMode from --to-file flag
  │  writes diff.md snapshot (review only)
  │  chooses KB load strategy (start only)
  ▼
Builder layer (plan/refine.go, review/review.go)
  │  populates TemplateData{ContextMode, BranchDir, ...}
  │  omits large content fields in reference mode
  ▼
Engine layer (prompt/engine.go)
  │  renders template with TemplateData
  ▼
Template layer (templates/**/*.md.tmpl)
  │  {{if eq .ContextMode "inline"}} → inline content
  │  {{else}} → emit file-read instruction with absolute BranchDir path
  ▼
stdout → IDE AI session reads prompt, optionally reads files
```

### Affected layers

| Layer | Component | Change type |
|---|---|---|
| `internal/prompt` | `engine.go` | Modified — add 2 constants + 2 struct fields |
| `internal/plan` | `refine.go` | Modified — 5 function signatures, conditional field population |
| `internal/review` | `review.go` | Modified — 2 function signatures, conditional field population |
| `internal/cli` | `plan.go` | Modified — contextMode wiring in 2 functions |
| `internal/cli` | `review.go` | Modified — contextMode wiring + diff.md write |
| `internal/cli` | `start.go` | Modified — contextMode wiring + KB load strategy |
| `internal/cli` | `knowledge_cmd.go` | Modified — BranchDir + ContextMode in TemplateData |
| `internal/context` | `context.go` | Modified — add `WarnMissingPredecessors` method |
| `internal/prompt/templates` | 6 `.md.tmpl` files | Modified — add inline/reference conditionals |
| `internal/plan` | `refine_test.go` | Modified — update call sites for new signatures |

### New components

None. All changes are modifications to existing files.

### Data flow — reference mode (IDE session, no `--to-file`)

1. User runs `/qode-plan-spec` (IDE slash command → executes `qode plan spec`)
2. `runPlanSpec` sets `contextMode = ContextModeReference`
3. `BuildSpecPromptWithOutput` receives `contextMode`; sets `Analysis = ""` (does not read `ctx.RefinedAnalysis` into the field), sets `BranchDir = ctx.ContextDir`
4. Template renders: `{{else}}` branch emits `Read the refined analysis from '<BranchDir>/refined-analysis.md'`
5. Prompt written to stdout; IDE AI reads it and reads the file directly (already in session context → no token duplication)

### Data flow — inline mode (`--to-file` or debugging)

1. User runs `qode plan spec --to-file`
2. `runPlanSpec` sets `contextMode = ContextModeInline`
3. `BuildSpecPromptWithOutput` sets `Analysis = ctx.RefinedAnalysis` (full content)
4. Template renders: `{{if eq .ContextMode "inline"}}` branch embeds content verbatim
5. Self-contained prompt written to `.qode/branches/<branch>/.spec-prompt.md`

---

## 4. API / Interface Contracts

This feature has no HTTP endpoints, CLI commands, or config schema changes. All changes are to internal Go function signatures and template rendering behaviour.

### Modified function signatures

```go
// internal/prompt/engine.go — new exports
const (
    ContextModeInline    = "inline"
    ContextModeReference = "reference"
)

// TemplateData — two new fields appended
type TemplateData struct {
    Project     config.ProjectConfig
    Layers      []config.LayerConfig
    Branch      string
    Ticket      string
    Notes       string
    Analysis    string
    Spec        string
    Diff        string
    Extra       string
    KB          string
    Lessons     string
    OutputPath  string
    ContextMode string // NEW: ContextModeInline | ContextModeReference
    BranchDir   string // NEW: absolute path to .qode/branches/<branch>/
}

// internal/plan/refine.go
func BuildRefinePromptWithOutput(
    engine     *prompt.Engine,
    cfg        *config.Config,
    ctx        *context.Context,
    ticketURL  string,
    iteration  int,
    outputPath string,
    contextMode string,  // NEW (last param)
) (*RefineOutput, error)

func BuildSpecPromptWithOutput(
    engine      *prompt.Engine,
    cfg         *config.Config,
    ctx         *context.Context,
    outputPath  string,
    contextMode string,  // NEW (last param)
) (string, error)

func BuildStartPrompt(
    engine      *prompt.Engine,
    cfg         *config.Config,
    ctx         *context.Context,
    kb          string,
    contextMode string,  // NEW (last param)
) (string, error)

// internal/review/review.go
func BuildCodePrompt(
    engine      *prompt.Engine,
    cfg         *config.Config,
    ctx         *context.Context,
    diff        string,
    outputPath  string,
    contextMode string,  // NEW (last param)
) (string, error)

func BuildSecurityPrompt(
    engine      *prompt.Engine,
    cfg         *config.Config,
    ctx         *context.Context,
    diff        string,
    outputPath  string,
    contextMode string,  // NEW (last param)
) (string, error)

// internal/context/context.go — new method
func (c *Context) WarnMissingPredecessors(step string, w io.Writer)
```

### Unchanged zero-arity wrappers (backward compatibility)

`BuildRefinePrompt` and `BuildSpecPrompt` continue to exist with their current signatures. Both internally pass `prompt.ContextModeInline` to their `WithOutput` counterparts.

### Template output contract

In reference mode, templates emit plain-text read instructions using absolute paths:
```
Read the refined analysis from `/abs/path/.qode/branches/my-branch/refined-analysis.md` before generating the spec.
```

In inline mode, templates embed content exactly as today. No structural change to the output format.

---

## 5. Data Model Changes

This feature introduces no database tables, new config fields, or persistent schema changes.

### New files written at runtime

| File | Written by | When | Contents |
|---|---|---|---|
| `<branchDir>/diff.md` | `cli/review.go:runReview` | On `qode review code` or `qode review security` in reference mode | Raw output of `git.DiffFromBase(root, "")` |

This file is ephemeral (overwritten on each review run) and does not require migration or cleanup logic.

### Backward compatibility

- Adding fields to `TemplateData` is additive; all existing callers compile without modification because Go structs allow omitting new fields in composite literals.
- Zero-arity builder wrappers (`BuildRefinePrompt`, `BuildSpecPrompt`) pass `ContextModeInline`, preserving current inline behaviour for any external callers.
- `judge_refine.md.tmpl` is not modified; the scoring engine continues to work with `TemplateData{Analysis: workerOutput}` (all other fields zero-valued).
- The `--to-file` flag path is semantically unchanged: it still produces a fully self-contained prompt; the only difference is the code path now explicitly passes `ContextModeInline` instead of relying on default behaviour.

---

## 6. Implementation Tasks

- [ ] **Task 1** (`internal/prompt`): Add `ContextModeInline`, `ContextModeReference` constants and `ContextMode string`, `BranchDir string` fields to `TemplateData` in `engine.go`. No other changes. Project must compile after this commit.

- [ ] **Task 2** (`internal/plan`): Update `BuildRefinePromptWithOutput`, `BuildSpecPromptWithOutput`, `BuildStartPrompt` in `refine.go`:
  - Add `contextMode string` as last parameter
  - Set `ContextMode: contextMode` and `BranchDir: ctx.ContextDir` in each `TemplateData`
  - Conditionally populate `Analysis` / `Spec` fields (only in `ContextModeInline`)
  - Update zero-arity wrappers `BuildRefinePrompt` and `BuildSpecPrompt` to pass `prompt.ContextModeInline`

- [ ] **Task 3** (`internal/review`): Update `BuildCodePrompt`, `BuildSecurityPrompt` in `review.go`:
  - Add `contextMode string` as last parameter
  - Set `ContextMode: contextMode` and `BranchDir: ctx.ContextDir`
  - Conditionally populate `Spec` and `Diff` (only in `ContextModeInline`)

- [ ] **Task 4** (`internal/plan`): Update `refine_test.go` — add `prompt.ContextModeInline` as last argument to all builder function call sites. Ensure `go test ./internal/plan/...` passes.

- [ ] **Task 5** (`internal/cli`): Update `plan.go` — in `runPlanRefine` and `runPlanSpec`, compute `contextMode` from `toFile` flag and pass to the respective builder functions.

- [ ] **Task 6** (`internal/cli`): Update `start.go` — compute `contextMode` from `toFile` flag; switch KB loading to `knowledge.List` + relative-path formatting when `contextMode == ContextModeReference`; call `ctx.WarnMissingPredecessors("start", os.Stderr)`; pass `contextMode` to `BuildStartPrompt`.

- [ ] **Task 7** (`internal/cli`): Update `review.go` — compute `contextMode` from `toFile` flag; write `diff.md` to `branchDir` when `contextMode == ContextModeReference`; call `ctx.WarnMissingPredecessors("review", os.Stderr)`; pass `contextMode` to `BuildCodePrompt` and `BuildSecurityPrompt`.

- [ ] **Task 8** (`internal/context`): Add `WarnMissingPredecessors(step string, w io.Writer)` to `context.go`. Add `"io"` to imports. No changes to `Load()` or existing exported functions.

- [ ] **Task 9** (`internal/cli`): Update `knowledge_cmd.go` — in `buildBranchLessonData()`, add `ContextMode: prompt.ContextModeReference` and `BranchDir: ctx.ContextDir` to the `TemplateData` struct literal.

- [ ] **Task 10–15** (`internal/prompt/templates`): Update all 6 templates with `{{if eq .ContextMode "inline"}}` conditionals (one commit):
  - `refine/base.md.tmpl` — Ticket, Notes, Analysis blocks
  - `spec/base.md.tmpl` — Analysis block
  - `start/base.md.tmpl` — Spec block (KB block unchanged)
  - `review/code.md.tmpl` — Spec block and Diff block
  - `review/security.md.tmpl` — Diff block
  - `knowledge/add-branch.md.tmpl` — Analysis, Spec, Diff blocks

---

## 7. Testing Strategy

### Unit tests

**`internal/plan/refine_test.go`** (existing, updated):
- Update all call sites to pass `prompt.ContextModeInline` — ensures inline mode renders identically to current behaviour.
- Add tests for reference mode:
  - `BuildRefinePromptWithOutput` with `ContextModeReference`: assert rendered prompt does NOT contain `ctx.RefinedAnalysis` content; assert it contains the `BranchDir` path substring.
  - `BuildSpecPromptWithOutput` with `ContextModeReference`: assert `ctx.RefinedAnalysis` is not inlined.
  - `BuildStartPrompt` with `ContextModeReference`: assert `ctx.Spec` is not inlined.

**`internal/review/review_test.go`** (new or existing):
- `BuildCodePrompt` with `ContextModeInline`: assert diff is embedded in a `diff` code fence.
- `BuildCodePrompt` with `ContextModeReference`: assert diff is NOT in output; assert `diff.md` path reference appears.
- `BuildSecurityPrompt` with `ContextModeReference`: same for diff.

**`internal/context/context_test.go`** (existing):
- `WarnMissingPredecessors("start", w)` with `HasSpec() == false`: assert warning written to `w`.
- `WarnMissingPredecessors("start", w)` with `HasSpec() == true`: assert nothing written.
- `WarnMissingPredecessors("review", w)` with `HasSpec() == false`: assert warning written.

### Integration tests

- `qode plan spec` with no `refined-analysis.md` present: assert error exit with message mentioning `refined-analysis.md` (existing guard, no change required — verify it still fires).
- `qode plan spec --to-file` with `refined-analysis.md` present: assert output file contains the full analysis text (inline mode unchanged).
- `qode plan spec` (no `--to-file`) with `refined-analysis.md` present: assert stdout prompt contains file-path reference, not the full analysis text.

### E2E tests

- Full workflow run in reference mode: `refine` → `spec` → `start` → `review code` → `review security`. Each step's stdout prompt must contain a file-path reference rather than the full previous artefact.
- Full workflow run with `--to-file` on each step: each saved prompt file must be fully self-contained (no file-read instructions).

### Edge cases to test explicitly

- Reference mode on first refine iteration (no previous `refined-analysis.md`): prompt must not contain a hard file-read instruction for the analysis file; must contain "begin fresh" wording.
- Reference mode with zero KB files (`knowledge.List` returns empty slice): `BuildStartPrompt` receives empty `kb` string; `{{if .KB}}` guard in start template suppresses the KB section entirely.
- `--to-file` flag with `toFile = true`: `contextMode` must equal `ContextModeInline`; all existing inline tests must pass.
- `BuildRefinePrompt` (zero-arity wrapper): must render as inline without caller passing contextMode.

---

## 8. Security Considerations

### Authentication / authorisation

No changes. This feature modifies local file I/O and stdout rendering only; it introduces no network calls, authentication, or access control changes.

### Input validation

`BranchDir` is derived from:
1. `config.QodeDir` — a constant string `".qode"`
2. `git.CurrentBranch(root)` — reads `.git/HEAD`, a local file; not external user input
3. `root` — from `resolveRoot()` → `filepath.Abs(os.Getwd())`

No user-controlled strings are concatenated into file paths. No path traversal risk.

`diff.md` is written from `git.DiffFromBase(root, "")` output. This is local git data; no sanitisation required for file write.

### Data sensitivity

`diff.md` written to `<branchDir>/diff.md` is a local git diff. It remains in the `.qode/` directory, which is already gitignored (or should be — verify `.gitignore` includes `.qode/branches/`). No secrets are introduced.

KB reference paths written to prompts are relative project paths (`.qode/knowledge/...`). These are not sensitive.

---

## 9. Open Questions

| # | Question | Owner | Blocking? |
|---|---|---|---|
| OQ1 | Is issue #24 merged to `main`? Verify: `git log --oneline main \| grep -i '#24'`. If not, this branch must wait. | petar-stupar | Yes |
| OQ2 | Should `diff.md` be committed to the branch or remain gitignored? It's a useful debugging artefact but regenerated on each review run. Current assumption: gitignored (ephemeral). | petar-stupar | No |
| OQ3 | Should `qode review code` / `qode review security` in reference mode also reference `spec.md` from `BranchDir` for the security template? Currently `BuildSecurityPrompt` does not set `Spec`. The template has no `{{.Spec}}` section. Confirm this is intentional (security review is diff-only) or extend both. | petar-stupar | No |
| OQ4 | `WarnMissingPredecessors` warns but does not block. Should `runStart` hard-error if `spec.md` is absent (similar to how `runPlanSpec` errors on missing `refined-analysis.md`)? Analysis recommends warn-only; confirm preference. | petar-stupar | No |

---

*Spec generated by qode. Copy to nqode-io/qode#25 for team review.*
