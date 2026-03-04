# How to Customise Prompts

qode ships with built-in prompt templates embedded in the binary. You can override any template per project without modifying the qode source.

## How it works

qode looks for prompt templates in two locations (in order):

1. `.qode/prompts/` — **your project-local overrides** (checked first)
2. `internal/prompt/templates/` — built-in templates compiled into the binary (fallback)

If a local override exists for a template, it is used instead of the built-in.

## Override a template

Create the override file at `.qode/prompts/<name>.md.tmpl`, mirroring the name used in `internal/prompt/templates/`:

```bash
mkdir -p .qode/prompts/refine
cp internal/prompt/templates/refine/base.md.tmpl .qode/prompts/refine/base.md.tmpl
# Edit .qode/prompts/refine/base.md.tmpl to your requirements
```

## Template naming

Local override paths mirror the structure under `internal/prompt/templates/`:

| Template | Source path | Override path |
|---|---|---|
| Refinement worker | `internal/prompt/templates/refine/base.md.tmpl` | `.qode/prompts/refine/base.md.tmpl` |
| Refinement judge | `internal/prompt/templates/scoring/judge_refine.md.tmpl` | `.qode/prompts/scoring/judge_refine.md.tmpl` |
| Spec generation | `internal/prompt/templates/spec/base.md.tmpl` | `.qode/prompts/spec/base.md.tmpl` |
| Implementation | `internal/prompt/templates/start/base.md.tmpl` | `.qode/prompts/start/base.md.tmpl` |
| Code review | `internal/prompt/templates/review/code.md.tmpl` | `.qode/prompts/review/code.md.tmpl` |
| Security review | `internal/prompt/templates/review/security.md.tmpl` | `.qode/prompts/review/security.md.tmpl` |

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

Local prompt overrides in `.qode/prompts/` should be committed to your repository so all team members use the same customised prompts.
