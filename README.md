# qode

AI-assisted developer workflow CLI for nQode.

Standardises how developers use AI coding assistants across client projects with varied tech stacks — Next.js+React, .NET+React, Angular+Java, and more.

## Quick Start

```bash
# Install
go install github.com/nqode/qode/cmd/qode@latest

# Add to PATH if not already
export PATH="$PATH:$HOME/go/bin"

# Onboard an existing project
cd your-project
qode init

# Set up IDE configs (Cursor + VS Code + Claude Code)
qode ide setup

# Start a feature
qode branch create feat-user-dashboard
qode ticket fetch https://company.atlassian.net/browse/ENG-123
```

## The Workflow

```
1. qode branch create <name>          Create git branch + context folder
2. qode ticket fetch <url>            Fetch ticket into context (or edit manually)
3. /qode-plan-refine  (in IDE)        Refine requirements — iterate to 25/25
4. /qode-plan-spec    (in IDE)        Generate tech spec
5. qode start                         Generate implementation prompt
6. (code in Cursor/Claude Code)
7. /qode-review-code  (in IDE)        Code review
8. /qode-review-security (in IDE)     Security review
9. qode check                         Run all quality gates
10. git commit && git push
```

Run `qode workflow` for the full diagram.

## Supported Tech Stacks

| Stack | Detected by |
|---|---|
| React | `react` in package.json |
| Next.js | `next` in package.json |
| Angular | `angular.json` |
| Vue / Svelte | package.json deps |
| .NET | `*.sln`, `*.csproj` |
| Java / Kotlin | `pom.xml`, `build.gradle` |
| Python | `pyproject.toml`, `requirements.txt` |
| Go | `go.mod` |
| TypeScript | `tsconfig.json` |

## Composite Stack Example

A real nQode project with React frontend, Next.js BFF, and .NET backend:

```yaml
# qode.yaml
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

## Scoring

qode uses a **two-pass scoring system** that eliminates self-scoring bias:

1. **Worker pass** — AI produces the analysis without self-scoring
2. **Judge pass** — Separate AI instance scores independently against a rubric

**Refinement rubric (25 points):**
- Problem Understanding (5)
- Technical Analysis (5)
- Risk & Edge Cases (5)
- Completeness (5)
- Actionability (5)

Target: 25/25 before generating a spec.

## IDE Support

| IDE | What's generated |
|---|---|
| **Cursor** | `.cursorrules/*.mdc` rules + `.cursor/commands/*.mdc` slash commands |
| **VS Code** | `.vscode/launch.json`, `tasks.json`, `settings.json`, `extensions.json` |
| **Claude Code** | `CLAUDE.md` + `.claude/commands/*.md` slash commands |

All configs are stack-aware. Existing files are preserved — qode only adds/updates its own sections.

## Commands

```
qode init                      Detect stack, create qode.yaml
qode init --new                New project wizard
qode init --workspace          Multi-repo workspace setup

qode branch create <name>      Create feature branch + context folder
qode branch list               List active branches
qode branch focus <name>       Switch to branch, show context
qode branch remove <name>      Clean up branch and context

qode ticket fetch <url>        Fetch ticket (Jira, Azure DevOps, Linear)

qode plan refine [url]         Generate worker + judge prompts (25/25 target)
qode plan spec                 Generate tech spec from refined analysis
qode plan status               Show iteration scores for current branch

qode start                     Generate implementation prompt

qode review code               Code review prompt
qode review security           Security review prompt
qode review all                Both reviews

qode check                     Run all quality gates per layer
qode check --layer <name>      Gates for a specific layer only
qode check --skip-tests        Reviews only, skip test execution

qode ide setup                 Generate IDE configs for all enabled IDEs
qode ide sync                  Regenerate configs from qode.yaml

qode knowledge add <path>      Add file to knowledge base
qode knowledge list            List knowledge base files
qode knowledge search <query>  Search knowledge base

qode config show               Show resolved config
qode config detect             Show auto-detected stacks
qode config validate           Validate qode.yaml

qode workflow                  Show full workflow diagram
```

## Project Types

### Single repo (most common)
```
my-project/
├── qode.yaml
├── src/
└── package.json
```
```bash
cd my-project && qode init
```

### Monorepo
```
enterprise-app/
├── qode.yaml
├── frontend/   (React)
├── backend/    (.NET)
└── shared/     (TypeScript)
```
```bash
cd enterprise-app && qode init   # auto-detects topology: monorepo
```

### Multi-repo workspace
```
workspace/
├── qode-workspace.yaml
├── client-frontend/
├── client-backend/
└── client-shared/
```
```bash
cd workspace && qode init --workspace
```

## Ticket System Setup

```bash
# Jira
export JIRA_EMAIL=you@company.com
export JIRA_API_TOKEN=your-token
qode ticket fetch https://company.atlassian.net/browse/ENG-123

# Azure DevOps
export AZURE_DEVOPS_PAT=your-pat
qode ticket fetch https://dev.azure.com/org/project/_workitems/edit/456

# Linear
export LINEAR_API_KEY=your-key
qode ticket fetch https://linear.app/team/ENG-123
```

## Customising Prompts

Override any built-in prompt per-project:

```bash
mkdir -p .qode/prompts/refine
# Create .qode/prompts/refine/base.md.tmpl with your custom content
# qode will use your file instead of the built-in template
```

## qode.yaml Reference

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
  type: jira                 # jira | azure-devops | linear | manual
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
  vscode:
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

## Contributing

```bash
git clone https://github.com/nqode/qode
cd qode
go test ./...
go install ./cmd/qode/
```
