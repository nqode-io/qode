# qode.yaml Reference

Full configuration reference for `qode.yaml`.

## Minimal example (single repo)

```yaml
project:
  name: my-project
  topology: single
  layers:
    - name: default
      path: .
      stack: go
```

## Composite stack example (monorepo)

A real nQode project with React frontend, Next.js BFF, and .NET backend:

```yaml
project:
  name: insurance-portal
  topology: monorepo
  layers:
    - name: frontend
      path: ./frontend
      stack: react
      test:
        unit: "npx vitest run"
        lint: "npx eslint ."
    - name: bff
      path: ./bff
      stack: nextjs
      test:
        unit: "npm test"
        lint: "npm run lint"
    - name: api
      path: ./backend
      stack: dotnet
      test:
        unit: "dotnet test"
        lint: "dotnet format --verify-no-changes"
```

## Full reference

```yaml
project:
  name: my-project
  description: Optional description
  topology: monorepo        # monorepo | multirepo | single (auto-detected)
  layers:
    - name: frontend
      path: ./frontend
      stack: react           # react | angular | nextjs | vue | svelte |
                             # dotnet | java | python | go | typescript
      test:
        unit: "npm test"
        e2e: "npx playwright test"
        lint: "npm run lint"
        build: "npm run build"
        coverage:
          enabled: true
          min_percentage: 80

ticket_system:
  mode: mcp                  # "mcp" (recommended) | "api" (default, legacy)
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
  rubrics:
    refine:               # override dimensions for judge_refine scoring
      dimensions:
        - name: Problem Understanding
          weight: 5
          levels:
            - "5: Perfect restatement of the problem in engineering terms"
            - "4: Mostly correct with minor gaps"
            - "3: Partial understanding; some aspects missed"
            - "2: Surface-level; misses key constraints"
            - "1: Vague or mostly incorrect"
            - "0: No meaningful understanding shown"
    review:               # override dimensions for code review scoring
      dimensions:
        - name: Correctness
          weight: 3
        - name: Code Quality
          weight: 2
        - name: Architecture
          weight: 2
        - name: Error Handling
          weight: 2
        - name: Testing
          weight: 1
        - name: Performance
          weight: 2
    security:             # override dimensions for security review scoring
      dimensions:
        - name: Injection Prevention
          weight: 3
        - name: Authentication & Authorisation
          weight: 3
        - name: Data Exposure
          weight: 2
        - name: Input Validation
          weight: 2
        - name: Cryptography
          weight: 2

ide:
  cursor:
    enabled: true
  claude_code:
    enabled: true

knowledge:
  auto_discover: true
  paths:
    - docs/architecture/
    - tests/**/README.md

architecture:
  clean_code:
    max_function_lines: 50
```

## Field descriptions

### `project.topology`

| Value | When to use |
|---|---|
| `single` | One repo, one tech stack |
| `monorepo` | Multiple stacks in subdirectories of the same repo |
| `multirepo` | Separate repos managed as a workspace (`qode init --workspace`) |

Auto-detected from directory structure when running `qode init`.

### `project.layers[].stack`

Supported values: `react`, `angular`, `nextjs`, `vue`, `svelte`, `dotnet`, `java`, `python`, `go`, `typescript`

### `ticket_system.mode`

| Value | Behaviour |
|---|---|
| `""` or `"api"` | (Default) Uses qode's built-in HTTP clients. Fetches title and description only. |
| `"mcp"` | Uses IDE MCP servers via `/qode-ticket-fetch` slash command. Fetches full ticket context including comments, attachments, and linked resources. `qode ticket fetch` becomes a no-op. |

See [docs/how-to-use-ticket-fetch.md](how-to-use-ticket-fetch.md) for MCP server setup.

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

Override the default scoring dimensions for any of the three rubrics: `refine`, `review`, `security`.

Each rubric entry accepts a `dimensions` list. Omitting a rubric (or providing an empty `dimensions` list) falls back to the built-in default for that rubric.

**`dimensions` fields:**

| Field | Required | Description |
|---|---|---|
| `name` | yes | Display name of the dimension |
| `weight` | yes | Maximum points for this dimension |
| `description` | no | Short description (informational) |
| `levels` | no | Ordered score descriptions, highest first (e.g. `"5: Excellent..."`) — only used by `refine` rubric |

The `refine` rubric uses `levels` to show labelled score bands in the judge prompt. The `review` and `security` rubrics do not use `levels`.
