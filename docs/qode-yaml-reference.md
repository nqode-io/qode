# qode.yaml Reference

Full configuration reference for `qode.yaml`.

> **Rubric customisation is in a separate file.** Scoring dimensions (`refine`, `review`, `security` rubrics) live in `.qode/scoring.yaml`, not here. See [scoring-yaml-reference.md](scoring-yaml-reference.md).

## Minimal example

`qode init` generates a `qode.yaml` with sensible defaults. No manual editing is required to get started.

```yaml
qode_version: "0.1"
review:
  min_code_score: 10
  min_security_score: 8
ide:
  cursor:
    enabled: true
  claude_code:
    enabled: true
knowledge:
  path: .qode/knowledge
```

> **Re-running `qode init` regenerates `qode.yaml` with these defaults.** Any customisations you have made to `qode.yaml` (score thresholds, `scoring.strict`, `ticket_system`, etc.) will be reset. Re-add them after running `qode init`, or run it only when you want a clean reset.

## Full reference

```yaml
qode_version: "0.1"          # written by qode init; informational

ticket_system:
  type: jira                 # jira | azure-devops | linear | github | notion | manual
  url: https://company.atlassian.net
  project_key: ENG
  auth:
    method: token
    env_var: JIRA_API_TOKEN

review:
  min_code_score: 10.0
  min_security_score: 8.0

scoring:
  target_score: 25        # override pass threshold for /qode-plan-refine
  strict: false           # enforce step ordering; exit 1 when a gate fails
  # Rubric dimensions are not configured here — edit .qode/scoring.yaml instead.

ide:
  cursor:
    enabled: true
  claude_code:
    enabled: true

knowledge:
  path: .qode/knowledge
```

## Field descriptions

### `qode_version`

Written by `qode init`. Identifies the qode configuration format version. Currently informational; version enforcement is planned for a future release.

### `ticket_system.type`

| Value | Env vars required |
|---|---|
| `jira` | `JIRA_EMAIL`, `JIRA_API_TOKEN` |
| `azure-devops` | `AZURE_DEVOPS_PAT` |
| `linear` | `LINEAR_API_KEY` |
| `github` | `GITHUB_TOKEN` (private repos only) |
| `notion` | `NOTION_API_KEY` |
| `manual` | None — edit `context/ticket.md` directly |

### `review.min_code_score` / `review.min_security_score`

Minimum scores for code and security reviews. Will be enforced by strict mode (see [#30](https://github.com/nqode-io/qode/issues/30)).

Defaults: `min_code_score: 10.0`, `min_security_score: 8.0`.

> **Breaking change (v0.x → configurable-rubrics):** `min_code_score` default changed from `8.0` to `10.0` because the default code review rubric now has a maximum of 12 (Performance dimension added, weight 2). If you had `min_code_score: 8.0` explicitly set, it remains valid. If you relied on the default, update your threshold to match the new maximum.

> **Tip:** When you override `scoring.rubrics.review` dimensions, update `min_code_score` to an appropriate fraction of your new total. Similarly for `scoring.rubrics.security` and `min_security_score`.

### `scoring.strict`

When `true`, guarded commands exit with a non-zero status code and print an error to stderr when a prerequisite is not met. When `false` (the default), the AI assistant receives a `STOP.` instruction on stdout and halts gracefully without an error exit.

| Mode | Gate fails | stdout | stderr | Exit code |
|---|---|---|---|---|
| `strict: false` | `STOP.` instruction | stop message | — | 0 |
| `strict: true` | error | — | `Error: <message>` | 1 |

**Guarded commands and their prerequisites:**

| Command | Prerequisite |
|---|---|
| `qode plan spec` / `/qode-plan-spec` | `refined-analysis.md` exists and score ≥ `target_score` |
| `qode start` / `/qode-start` | `spec.md` exists |
| `qode review code` / `/qode-review-code` | Uncommitted diff is non-empty (strict only) |
| `qode review security` / `/qode-review-security` | Uncommitted diff is non-empty (strict only) |

All guarded commands accept `--force` to bypass score and completeness gates. Absent-file errors (e.g. no `refined-analysis.md` at all) are always hard errors regardless of `--force` or `strict` mode.

Default: `false` (backward compatible — existing workflows are unaffected).

### `scoring.target_score`

Override the pass threshold for `/qode-plan-refine`. When not set, the threshold defaults to the total weight of the `refine` rubric (25 by default).

> **Breaking change (v0.x → configurable-rubrics):** `scoring.refine_target_score` was renamed to `scoring.target_score`. Update your `qode.yaml` if you had this field set.

### `scoring.rubrics`

Rubric dimensions are **not configured in `qode.yaml`**. They live in `.qode/scoring.yaml` so that re-running `qode init` never overwrites them. See [scoring-yaml-reference.md](scoring-yaml-reference.md) for the full rubric format and field reference.
