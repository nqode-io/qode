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
  type: jira                 # jira | azure-devops | linear | github | notion | manual
  url: https://company.atlassian.net
  project_key: ENG
  auth:
    method: token
    env_var: JIRA_API_TOKEN

review:
  min_code_score: 8.0
  min_security_score: 8.0

scoring:
  two_pass: true
  max_refine_iterations: 5
  refine_target_score: 25

ide:
  cursor:
    enabled: true
  claude_code:
    enabled: true
  # vscode: no longer supported — VSCode has no AI slash command mechanism

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

Minimum scores (0–10) required for `qode check` to pass. Default: `8.0`.

### `scoring.refine_target_score`

Target score for `/qode-plan-refine` iterations. Default: `25` (5 dimensions × 5 points).
