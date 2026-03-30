# How to Customise Prompts

qode ships with built-in prompt templates embedded in the binary. When you run `qode init`, all templates are automatically copied to `.qode/prompts/` so you can edit them directly.

## How it works

`qode init` copies all built-in templates into `.qode/prompts/`. qode checks this directory first when rendering prompts — if a file exists there, it is used instead of the built-in version.

Existing files are always overwritten on each `qode init` run to keep them in sync with the embedded defaults. Back up or commit any local edits before re-running `qode init`.

## Customise a template

Edit the template file directly in `.qode/prompts/`:

```bash
# Edit the refinement worker prompt
$EDITOR .qode/prompts/refine/base.md.tmpl
```

To preview the rendered output after editing, use the `--to-file` flag:

```bash
qode plan refine --to-file
# → saves to .qode/branches/<branch>/.refine-prompt.md

qode plan judge --to-file
# → saves to .qode/branches/<branch>/.refine-judge-prompt.md
# requires refined-analysis.md to exist in the branch directory
```

## Template files

All templates are located under `.qode/prompts/`:

| Template | Path |
|---|---|
| Refinement worker | `.qode/prompts/refine/base.md.tmpl` |
| Refinement judge | `.qode/prompts/scoring/judge_refine.md.tmpl` |
| Spec generation | `.qode/prompts/spec/base.md.tmpl` |
| Implementation | `.qode/prompts/start/base.md.tmpl` |
| Code review | `.qode/prompts/review/code.md.tmpl` |
| Security review | `.qode/prompts/review/security.md.tmpl` |
| Lessons from branch | `.qode/prompts/knowledge/add-branch.md.tmpl` |
| Lessons from session | `.qode/prompts/knowledge/add-context.md.tmpl` |

## Template variables

Templates use Go's `text/template` syntax. The following fields are available on the data struct:

| Field | Type | Description |
|---|---|---|
| `Project` | `config.ProjectConfig` | Project name and topology from `qode.yaml` |
| `Layers` | `[]config.LayerConfig` | Stack layers from `qode.yaml` |
| `Branch` | `string` | Current git branch name |
| `BranchDir` | `string` | Absolute path to `.qode/branches/<branch>/` — use this to reference context files |
| `OutputPath` | `string` | When set, templates should append file-write instructions so the AI saves its output directly |
| `KB` | `string` | Knowledge base file references (set for `start` only) |
| `Extra` | `string` | Assembled extra context such as code reviews (set for `refine` and `knowledge/add-branch`) |
| `Lessons` | `string` | Existing lesson summaries for deduplication (set for `knowledge/add-branch` only) |
| `Ticket` | `string` | Ticket content inlined (set for `knowledge/add-branch` only) |
| `Analysis` | `string` | Refined analysis inlined (set for `knowledge/add-branch` only) |
| `Spec` | `string` | Spec content inlined (set for `knowledge/add-branch` only) |
| `Diff` | `string` | Git diff inlined (set for `knowledge/add-branch` only) |
| `Rubric` | `scoring.Rubric` | Active scoring rubric with `.Dimensions` and `.Total` — set for `scoring/judge_refine`, `review/code`, and `review/security`; reflects any `scoring.rubrics` override from `qode.yaml` |
| `TargetScore` | `int` | Pass threshold for the refine judge — defaults to `Rubric.Total()`, overridden by `scoring.target_score` in `qode.yaml` |
| `MinPassScore` | `float64` | Minimum score to pass review — sourced from `review.min_code_score` (code) or `review.min_security_score` (security) in `qode.yaml` |

## Template functions

The following functions are available inside templates in addition to Go's built-in `text/template` functions (e.g. `printf`, `len`):

| Function | Signature | Description |
|---|---|---|
| `add` | `add a b int → int` | Adds two integers. Useful for 1-based dimension counters: `{{add $i 1}}` |
| `pct` | `pct percent float64, n int → float64` | Returns `n × percent / 100`. Use with `printf "%.1f"` for formatted thresholds: `{{printf "%.1f" (pct 75 .Rubric.Total)}}` |
| `join` | `join sep string, items []string → string` | Joins a string slice with the given separator |

### Referencing context files

For the main workflow templates (`refine`, `spec`, `start`, `review`) the AI reads context files directly rather than having content inlined. Use `BranchDir` to construct paths:

```
Read the ticket from `{{.BranchDir}}/context/ticket.md`.
Read the notes from `{{.BranchDir}}/context/notes.md` (if the file exists).
Read the refined analysis from `{{.BranchDir}}/refined-analysis.md`.
Read the spec from `{{.BranchDir}}/spec.md`.
Read the diff from `{{.BranchDir}}/diff.md`.
```

## Committing overrides

Prompt templates in `.qode/prompts/` should be committed to your repository so all team members use the same customised prompts.
