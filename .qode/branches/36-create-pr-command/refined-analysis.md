<!-- qode:iteration=3 score=25/25 -->
# Refined Requirements Analysis — qode pr create

## 1. Problem Understanding

**Restated problem:** qode builds up rich branch context over the lifecycle of a feature branch — ticket, spec, diff, code-review, security-review — but has no command that uses this context to create the pull request. The developer must write a PR title and description manually from scratch, which is slow and produces lower-quality descriptions than an AI with full context could generate.

**User need:** After `qode check` passes and changes are pushed, the developer runs `qode pr create` (or the IDE slash command `/qode-pr-create`). The CLI renders a prompt that includes all available branch context. The AI reads the prompt and uses its configured MCP server (GitHub MCP) to create the PR directly, then saves the resulting PR URL to `.qode/branches/<branch>/context/pr-url.txt` for use by the subsequent `/qode-review-address` step (#31).

**Business value:** Eliminates the manual PR description step; produces consistently high-quality descriptions; closes the workflow gap between `qode check` and `#31`.

**Resolved ambiguities (from notes.md):**

1. **PR creation is MCP-only.** The Go CLI renders a prompt and prints it to stdout. The AI uses its MCP server to create the PR. There is no `--dry-run` flag, no `--url` flag, no GitHub API calls in Go, and no `gh pr create` fallback.
2. **Ticket naming errors corrected.** `internal/context/context.go` in the ticket → correct path is `internal/branchcontext/context.go`. `internal/ide/claudecode.go` / `internal/ide/cursor.go` → correct paths are `internal/scaffold/claudecode.go` / `internal/scaffold/cursor.go`.
3. **Base branch override uses `--base` flag.** Priority: `--base` CLI flag > `pr.base_branch` config field > `git.DefaultBranch(ctx, root)` (runs `git symbolic-ref refs/remotes/origin/HEAD --short`, strips `origin/`) > hard-coded fallback `"main"`.

**No remaining open questions.**

---

## 2. Technical Analysis

### Directory layout (confirmed by reading source)

`branchcontext.Context.ContextDir` is the **branch root** directory: `.qode/branches/<sanitized-branch>/`. It is NOT the `context/` subdirectory. The subdirectory for per-branch files (ticket, notes, pr-url.txt) is `<ContextDir>/context/`. Review files live directly in the branch root:

```
.qode/branches/<branch>/
├── context/
│   ├── ticket.md
│   ├── notes.md
│   └── pr-url.txt          ← NEW: written by AI after PR creation
├── refined-analysis.md
├── spec.md
├── code-review.md           ← filepath.Join(ctx.ContextDir, "code-review.md")
├── security-review.md       ← filepath.Join(ctx.ContextDir, "security-review.md")
└── diff.md
```

### Layers and components affected

Following the dependency layering in CLAUDE.md (only `cli` fans out; no circular deps):

| Layer | Package | Change |
|-------|---------|--------|
| Leaf | `git` | Add `DefaultBranch(ctx, root) (string, error)` |
| Mid-level | `config` | Add `PRConfig` struct to `schema.go`; add defaults to `defaults.go` |
| Domain | `branchcontext` | Add `PRURL string` to `Context`; load from `context/pr-url.txt` in `Load()`; add `HasPRURL() bool`; add `StorePRURL(root, branch, url string) error` |
| Domain | `prompt` | Add `BaseBranch string`, `CodeReview string`, `SecurityReview string`, `DraftPR bool` to `TemplateData`; add builder methods; add template `pr/create.md.tmpl` |
| Domain | `scaffold` | Add `internal/prompt/templates/scaffold/qode-pr-create.md.tmpl`; register `"qode-pr-create"` in `claudeCommands` / `cursorCommands` slices |
| Domain | `workflow` | Add `"pr"` case to `CheckStep()` in `guard.go` |
| Top-level | `cli` | New `internal/cli/pr.go`; register `newPrCmd()` in `root.go` |

### Key technical decisions

**1. MCP-only PR creation — no Go API calls.**
The CLI's only job is to render and print a prompt. The AI uses the GitHub MCP server to create the PR. No network code in Go.

**2. `--base` flag with explicit priority chain.**
`newPrCreateCmd()` exposes `--base string`. In `runPrCreate`:
```go
baseBranch := base  // --base flag
if baseBranch == "" { baseBranch = sess.Config.PR.BaseBranch }
if baseBranch == "" {
    if b, err := git.DefaultBranch(ctx, sess.Root); err == nil { baseBranch = b }
}
if baseBranch == "" { baseBranch = "main" }
```

**3. `BaseBranch string` as a dedicated `TemplateData` field — not `.Extra`.**
`.Extra` is reserved for user-provided text file contents (aggregated from `context/` extra files). Base branch is a structured, typed piece of data that the template uses directly. Add `BaseBranch string` to `TemplateData` and `WithBaseBranch(s string)` to `TemplateDataBuilder`. The template uses `{{.BaseBranch}}`.

**4. `DraftPR bool` as a dedicated `TemplateData` field.**
Resolves cleanly — template uses `{{if .DraftPR}}` to conditionally set the draft flag in MCP instructions.

**5. Review files read directly from `ContextDir`.**
`code-review.md` and `security-review.md` are at `filepath.Join(sess.Context.ContextDir, "code-review.md")` and `filepath.Join(sess.Context.ContextDir, "security-review.md")`. Read with `os.ReadFile`; treat missing file as empty string (no error). Template conditionally omits section with `{{if .CodeReview}}`.

**6. PR URL stored in `context/` subdirectory.**
`StorePRURL(root, branch, url string) error` writes to `filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch), "context", "pr-url.txt")` via `iokit.AtomicWrite`. `Context.Load()` reads from `filepath.Join(ctxSubDir, "pr-url.txt")` (using the existing `ctxSubDir` variable in `Load()`). `HasPRURL() bool` returns `c.PRURL != ""`.

**7. `BranchDir` in template = `sess.Context.ContextDir`.**
The template instructs the AI to write `{{.BranchDir}}/context/pr-url.txt`. The `BranchDir` field on `TemplateData` already exists and is set via `WithBranchDir(sess.Context.ContextDir)`.

**8. Workflow guard — spec only.**
`workflow.CheckStep("pr", ctx, cfg)` blocks when `ctx.Spec == ""`. Reviews are NOT required. Message: `"spec.md not found — run /qode-plan-spec first"`.

**9. Template location and override.**
Template name `"pr/create"`. Embedded at `internal/prompt/templates/pr/create.md.tmpl`. Local override at `.qode/prompts/pr/create.md.tmpl` (standard engine override).

**10. Scaffold template — same pattern as existing commands.**
`internal/prompt/templates/scaffold/qode-pr-create.md.tmpl` with `{{if eq .IDE "cursor"}}` Cursor frontmatter guard.

**11. `git.DefaultBranch` — new function.**
Does not yet exist in `internal/git/git.go`. Confirmed by grep: only `SanitizeBranchName` exists. New function:
```go
// DefaultBranch returns the default branch for origin, falling back to "main".
func DefaultBranch(ctx context.Context, root string) (string, error) {
    out, err := runGit(ctx, root, "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
    if err != nil || strings.TrimSpace(out) == "" {
        return "main", nil
    }
    return strings.TrimPrefix(strings.TrimSpace(out), "origin/"), nil
}
```

### Patterns and conventions

- `newXxxCmd()` / `runXxx(ctx context.Context, out, errOut io.Writer, ...)` — same as `internal/cli/review.go`
- `loadSessionCtx(ctx)` for all session bootstrapping
- `prompt.NewTemplateData(name, branch).WithXxx().Build()` fluent builder
- `iokit.AtomicWrite` for all file writes
- `t.Helper()`, `t.Parallel()`, table-driven tests, golden files for template output
- Sentinel errors wrapped with `%w`; `errors.Is` in tests

### Dependencies on other features / services

- `#31` (review address) requires `pr-url.txt` — this feature is the prerequisite.
- `branchcontext.HasCodeReview()` and `HasSecurityReview()` already exist (confirmed in source).
- No new external Go library dependencies.

---

## 3. Risk & Edge Cases

### Risks

| Risk | Mitigation |
|------|-----------|
| `origin/HEAD` not set (shallow clone, no default branch configured) | `git.DefaultBranch` falls back to `"main"` silently; `--base` flag allows override |
| `code-review.md` or `security-review.md` absent | `os.ReadFile` error ignored; empty string passed; template guards with `{{if .CodeReview}}` |
| `spec.md` absent | `workflow.CheckStep("pr")` blocks with actionable message before any I/O |
| Detached HEAD state | `loadSessionCtx` resolves branch via `git.CurrentBranch()`; returns error if detached |
| PR already exists for branch | Template instructs AI to check via MCP before creating; AI reports existing URL and stops |
| `context/pr-url.txt` path traversal | `StorePRURL` uses `git.SanitizeBranchName` (same as `branch.go`) |
| Very long diff | Template passes `{{.Diff}}` inline; AI handles contextually (same as review prompts) |

### Edge cases

- **Draft PR:** `cfg.PR.Draft` → `TemplateData.DraftPR bool` → `{{if .DraftPR}}...draft...{{end}}` in MCP instruction.
- **No ticket file:** `ctx.Ticket == ""`; template uses `{{if .Ticket}}` guard.
- **Both `--base` flag and `pr.base_branch` set:** `--base` flag wins (checked first).
- **`BaseBranch` empty after all fallbacks:** Hard-coded `"main"` ensures it is never empty in the template.

### Security considerations

- No shell command construction in Go for PR creation — no injection risk.
- `pr-url.txt` path validated via `git.SanitizeBranchName`.
- No credentials stored; MCP handles authentication.

### Performance

No performance concerns — file I/O only; no network calls from Go.

---

## 4. Completeness Check

### Acceptance criteria

1. `qode pr create [--base <branch>]` registered in the CLI.
2. `--base` flag resolved with priority: flag > `pr.base_branch` config > `git.DefaultBranch()` > `"main"`.
3. Rendered prompt includes: project, branch, base branch, ticket (conditional), spec, diff (conditional), code-review (conditional), security-review (conditional), PR description format, MCP creation instructions with "PR already exists" check.
4. Template instructs AI to save PR URL to `{{.BranchDir}}/context/pr-url.txt` after creation.
5. Template conditionally sets draft via `{{if .DraftPR}}`.
6. `/qode-pr-create` slash command written for Claude Code (`.claude/commands/`) and Cursor (`.cursor/commands/`).
7. `PRConfig{Template string, Draft bool, BaseBranch string}` added to `internal/config/schema.go`.
8. Defaults: `Template: "default"`, `Draft: false`, `BaseBranch: ""` in `internal/config/defaults.go`.
9. `branchcontext.Context.PRURL` loaded from `<ContextDir>/context/pr-url.txt` in `Load()`.
10. `branchcontext.HasPRURL() bool` accessor exists.
11. `branchcontext.StorePRURL(root, branch, url string) error` uses `iokit.AtomicWrite`.
12. `workflow.CheckStep("pr", ctx, cfg)` blocks when `ctx.Spec == ""`.
13. `TemplateData` gains `BaseBranch string`, `CodeReview string`, `SecurityReview string`, `DraftPR bool` with builder methods.
14. `git.DefaultBranch(ctx context.Context, root string) (string, error)` implemented in `internal/git/git.go`.
15. Unit tests: `TestDefaultConfig_PRDefaults`, `TestDefaultBranch_FallsBackToMain`, `TestContext_LoadPRURL`, `TestStorePRURL`, `TestCheckStep_PR_Blocked`, `TestCheckStep_PR_Passes`, `TestTemplateDataBuilder_PRFields`, golden file for `pr/create` template.
16. Integration test: `internal/cli/pr_test.go` behind `//go:build integration`.
17. `docs/qode-yaml-reference.md` documents `pr:` section.
18. `README.md` workflow updated with `qode pr create` as step 9.
19. `CLAUDE.md` is NOT modified (project constraint).

### Implicit requirements not stated in the ticket

- **`qode-pr-create` scaffold template required in `internal/prompt/templates/scaffold/`** — `SetupClaudeCode()` / `SetupCursor()` must include `"qode-pr-create"` in their command slices; the generated files are committed after `qode init`.
- **Template "PR already exists" guard** — prompt must instruct AI to check via MCP before creating a duplicate; golden file test verifies this instruction is present.
- **`git.DefaultBranch` does not yet exist** (confirmed by grep) — must be added in Task 2.
- **`BaseBranch` must never be empty when passed to template** — hard-coded `"main"` fallback in `runPrCreate` guarantees this.

### Out of scope

- `--dry-run` flag and any "print without creating" behavior.
- `--url` / remote URL override.
- Go-level GitHub / GitLab / Azure DevOps API calls.
- `gh pr create` / `glab mr create` CLI fallback.
- Updating existing PRs.
- PR comment resolution (`#31`).
- Codex integration (`#34`).

---

## 5. Actionable Implementation Plan

Tasks 1–5 are independent and can be done in parallel. Task 6 depends on Task 5. Task 7 depends on Tasks 1–6. Task 8 depends on Task 7. Task 9 is last.

### Task 1 — Config: add `PRConfig` struct and defaults
**Files:** `internal/config/schema.go`, `internal/config/defaults.go`
- Add `PR PRConfig \`yaml:"pr"\`` to `Config` struct.
- `PRConfig`: `Template string \`yaml:"template"\``, `Draft bool \`yaml:"draft"\``, `BaseBranch string \`yaml:"base_branch"\``.
- Defaults: `Template: "default"`, `Draft: false`, `BaseBranch: ""`.
- Unit test: `TestDefaultConfig_PRDefaults` in `internal/config/config_test.go` — assert all three fields.

### Task 2 — Git: add `DefaultBranch` helper
**Files:** `internal/git/git.go`
- Add `DefaultBranch(ctx context.Context, root string) (string, error)`:
  - Run `runGit(ctx, root, "symbolic-ref", "refs/remotes/origin/HEAD", "--short")`.
  - Strip `origin/` prefix; if error or empty, return `"main", nil`.
- Unit test: `TestDefaultBranch_FallsBackToMain` in `internal/git/git_test.go` — use a temp git repo without `origin/HEAD`; assert return value is `"main"`.

### Task 3 — Branchcontext: add PR URL storage/retrieval
**Files:** `internal/branchcontext/context.go`
- Add `PRURL string` field to `Context` struct.
- In `Load()`, after reading `ctxSubDir`: `ctx.PRURL = strings.TrimSpace(iokit.ReadFileOrString(filepath.Join(ctxSubDir, "pr-url.txt"), ""))`.
- Add `HasPRURL() bool`: `return c.PRURL != ""`.
- Add `StorePRURL(root, branch, url string) error`: path = `filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch), "context", "pr-url.txt")`; calls `iokit.AtomicWrite(path, []byte(url))`.
- Unit tests: `TestContext_LoadPRURL` (writes `pr-url.txt`, calls `Load`, asserts `PRURL`); `TestStorePRURL` (calls function, reads file back, asserts trimmed content).

### Task 4 — Workflow: add `"pr"` guard step
**Files:** `internal/workflow/guard.go`
- Add `case "pr":` to `CheckStep()`: `if ctx.Spec == "" { return CheckResult{Blocked: true, Message: "spec.md not found — run /qode-plan-spec first"} }`.
- Unit tests: `TestCheckStep_PR_Blocked` (empty `Spec`), `TestCheckStep_PR_Passes` (non-empty `Spec`).

### Task 5 — Prompt: extend `TemplateData` with PR-specific fields
**Files:** `internal/prompt/engine.go` (`TemplateData` struct), `internal/prompt/builder.go`
- Add to `TemplateData`: `BaseBranch string`, `CodeReview string`, `SecurityReview string`, `DraftPR bool`.
- Add builder methods to `TemplateDataBuilder`: `WithBaseBranch(s string)`, `WithCodeReview(s string)`, `WithSecurityReview(s string)`, `WithDraftPR(draft bool)`.
- Unit test: `TestTemplateDataBuilder_PRFields` — assert all four fields are set via builder.

### Task 6 — Template: create `internal/prompt/templates/pr/create.md.tmpl`
**Files:** `internal/prompt/templates/pr/create.md.tmpl` (new)

```
# PR Creation — {{.Project.Name}}

## Context
- **Branch:** {{.Branch}}
- **Base branch:** {{.BaseBranch}}

{{if .Ticket}}## Ticket
{{.Ticket}}
{{end}}
## Specification
{{.Spec}}

{{if .Diff}}## Diff
{{.Diff}}
{{end}}
{{if .CodeReview}}## Code Review
{{.CodeReview}}
{{end}}
{{if .SecurityReview}}## Security Review
{{.SecurityReview}}
{{end}}
## Your task

1. Check via MCP whether a PR already exists for branch `{{.Branch}}` targeting `{{.BaseBranch}}`. If one exists, report its URL and stop.
2. Generate a PR title and description in this format:
   ```
   ## Summary
   <2–4 bullets referencing the ticket>

   ## Changes
   <grouped by area with file-level callouts>

   ## Testing
   <what was tested; links to test files>

   ## Review notes
   <flags from code/security review>

   ## Ticket
   <link if available>
   ```
3. Create the PR via MCP targeting `{{.BaseBranch}}`{{if .DraftPR}} as a draft{{end}}.
4. After creation, write the PR URL to: `{{.BranchDir}}/context/pr-url.txt`
```

- Golden file test: `internal/prompt/templates/pr/create_test.go` — table-driven with subtests:
  - Ticket present → section rendered; ticket absent → section omitted
  - CodeReview present → section rendered; absent → omitted
  - SecurityReview present → section rendered; absent → omitted
  - `DraftPR: true` → "as a draft" present; `DraftPR: false` → absent
  - "PR already exists" check instruction present in all cases

### Task 7 — CLI: add `internal/cli/pr.go`
**Files:** `internal/cli/pr.go` (new), `internal/cli/root.go`

```go
func newPrCmd() *cobra.Command {
    cmd := &cobra.Command{Use: "pr", Short: "Pull request commands"}
    cmd.AddCommand(newPrCreateCmd())
    return cmd
}

func newPrCreateCmd() *cobra.Command {
    var base string
    cmd := &cobra.Command{
        Use:   "create",
        Short: "Generate a PR prompt and create via MCP",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runPrCreate(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), base)
        },
    }
    cmd.Flags().StringVar(&base, "base", "", "Base branch (overrides config and auto-detection)")
    return cmd
}

func runPrCreate(ctx context.Context, out, errOut io.Writer, base string) error {
    sess, err := loadSessionCtx(ctx)
    if err != nil {
        return err
    }
    if result := workflow.CheckStep("pr", sess.Context, sess.Config); result.Blocked {
        fmt.Fprintln(errOut, result.Message)
        return nil
    }

    // Resolve base branch: flag > config > git detection > "main".
    baseBranch := base
    if baseBranch == "" {
        baseBranch = sess.Config.PR.BaseBranch
    }
    if baseBranch == "" {
        if b, err := git.DefaultBranch(ctx, sess.Root); err == nil {
            baseBranch = b
        }
    }
    if baseBranch == "" {
        baseBranch = "main"
    }

    codeReview, _ := os.ReadFile(filepath.Join(sess.Context.ContextDir, "code-review.md"))
    secReview, _ := os.ReadFile(filepath.Join(sess.Context.ContextDir, "security-review.md"))

    data := prompt.NewTemplateData(sess.Engine.ProjectName(), sess.Branch).
        WithBranchDir(sess.Context.ContextDir).
        WithTicket(sess.Context.Ticket).
        WithSpec(sess.Context.Spec).
        WithDiff(sess.Context.Diff).
        WithBaseBranch(baseBranch).
        WithCodeReview(string(codeReview)).
        WithSecurityReview(string(secReview)).
        WithDraftPR(sess.Config.PR.Draft).
        Build()

    p, err := sess.Engine.Render("pr/create", data)
    if err != nil {
        return err
    }
    fmt.Fprint(out, p)
    return nil
}
```

Key correctness notes:
- `sess.Context.ContextDir` is the branch root (`.qode/branches/<branch>/`) — confirmed by reading source; `BranchDir` in template is set directly from it.
- Review files at `filepath.Join(sess.Context.ContextDir, "code-review.md")` — confirmed by `HasCodeReview()` implementation.
- `sess.Context.Diff` — check if this field exists; if not, read `diff.md` via `os.ReadFile(filepath.Join(sess.Context.ContextDir, "diff.md"))`.

Register: `rootCmd.AddCommand(newPrCmd())` in `internal/cli/root.go`.

Integration test: `internal/cli/pr_test.go` behind `//go:build integration` — creates temp project with `spec.md` and `ticket.md`, calls `runPrCreate`, asserts output contains `BaseBranch` sentinel and ticket content.

### Task 8 — Scaffold: add `qode-pr-create` template and register
**Files:** `internal/prompt/templates/scaffold/qode-pr-create.md.tmpl` (new), `internal/scaffold/claudecode.go`, `internal/scaffold/cursor.go`

- Scaffold template instructs IDE to run `qode pr create`, then use MCP to create the PR and write URL to `pr-url.txt`. Same `{{if eq .IDE "cursor"}}` frontmatter guard.
- Add `"qode-pr-create"` to `claudeCommands` slice in `internal/scaffold/claudecode.go`.
- Add `"qode-pr-create"` to `cursorCommands` slice in `internal/scaffold/cursor.go`.
- Existing `SetupClaudeCode` / `SetupCursor` tests verify the new command file is written.

### Task 9 — Docs, generated files
**Actions:**
- Run `go install ./cmd/qode/ && qode init` to regenerate `.claude/commands/qode-pr-create.md` and `.cursor/commands/qode-pr-create.mdc`; commit generated files.
- Update `docs/qode-yaml-reference.md`: add `pr:` section with `template`, `draft`, `base_branch` descriptions.
- Update `README.md` workflow section: insert `qode pr create` as step 9 between `git push` and `/qode-review-address`.
- Do NOT modify `CLAUDE.md`.

### Dependency graph

```
Task 1 (config)         ─┐
Task 2 (git)            ─┤
Task 3 (branchcontext)  ─┤→ Task 7 (CLI) → Task 8 (scaffold) → Task 9 (docs/generated)
Task 4 (workflow)       ─┤
Task 5 (TemplateData)   ─┤
                         └→ Task 6 (template) ─┘
```
