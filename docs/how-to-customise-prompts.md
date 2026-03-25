# How to Customise Prompts

qode ships with built-in prompt templates embedded in the binary. When you run `qode init`, all templates are automatically copied to `.qode/prompts/` so you can edit them directly.

## How it works

`qode init` copies all built-in templates into `.qode/prompts/`. qode checks this directory first when rendering prompts — if a file exists there, it is used instead of the built-in version.

Existing files are never overwritten, so running `qode init` again is safe.

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

## Template variables

Templates use Go's `text/template` syntax. The data struct passed in has these fields:

| Field | Type | Description |
|---|---|---|
| `Project` | `config.ProjectConfig` | Project name, topology |
| `Layers` | `[]config.LayerConfig` | Stack layers from qode.yaml |
| `Branch` | `string` | Current git branch name |
| `Ticket` | `string` | Contents of context/ticket.md |
| `Notes` | `string` | Contents of context/notes.md |
| `Analysis` | `string` | Current refined analysis |
| `Spec` | `string` | Generated spec |
| `KB` | `string` | Knowledge base fragments |

## Committing overrides

Prompt templates in `.qode/prompts/` should be committed to your repository so all team members use the same customised prompts.
