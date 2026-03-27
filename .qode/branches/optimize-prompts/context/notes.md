# Notes

## Design Decision: Remove ContextMode Entirely

The `--to-file` flag invocations must behave identically to stdout output.
Both always use reference mode: templates emit file-read instructions instead
of inlining content. There is no inline mode.

**Consequences:**
- `ContextModeInline` and `ContextModeReference` constants are removed from `engine.go`
- `ContextMode string` field is removed from `TemplateData`
- All builder functions (`BuildRefinePromptWithOutput`, `BuildSpecPromptWithOutput`,
  `BuildStartPrompt`, `BuildCodePrompt`, `BuildSecurityPrompt`) no longer accept
  a `contextMode` parameter
- All templates use reference-mode content only (no `{{if eq .ContextMode "inline"}}` blocks)
- Large content fields (Ticket, Analysis, Spec, Diff) are only populated for the
  `knowledge/add-branch` command and scoring judge, where inline content is needed
  for retrospective lesson extraction across branches; all other templates reference
  files via `{{.BranchDir}}`
- The `--to-file` flag only controls whether the rendered prompt is saved to disk
  or printed to stdout — it does not change prompt content
- `BranchDir` field stays in `TemplateData` (templates need it for file-path references)

## Design Decision: Exclude .qode/ from Git Diff

`DiffFromBase` in `internal/git/git.go` passes `-- :(exclude).qode/` to all three
`git diff` invocations. `.qode/branches/` contains workflow artifacts (spec.md,
code-review.md, diff.md, etc.) that are committed as part of the workflow. Including
them in the review diff wastes tokens and adds noise.

This is portable: `exec.Command` bypasses shell interpretation so `:`, `(`, `)` are
passed literally to git. `:(exclude)` is a git core pathspec feature (since git 1.9),
supported identically on macOS, Linux, and Windows.

Any new `git diff` calls should carry `"--", ":(exclude).qode/"` as the final args.

## Design Decision: Remove Notes from TemplateData and Context

`Notes string` was a field that held the contents of `context/notes.md` for inline
embedding. Since all templates now use reference mode, there is no consumer of this
field. It has been removed from both `prompt.TemplateData` and `context.Context`.

The `refine/base.md.tmpl` template still instructs the AI to read
`{{.BranchDir}}/context/notes.md` directly, so the notes file is still effective.
