# qode

AI-assisted developer workflow CLI by nQode.

Standardises how developers use AI coding assistants across client projects with varied tech stacks — Next.js+React, .NET+React, Angular+Java, and more.

## Installation

Download the latest binary for your platform from [GitHub Releases](https://github.com/nqode-io/qode/releases), then extract and add it to your PATH.

**Alternative — install from source** (requires Go 1.24+):
```bash
go install github.com/nqode/qode/cmd/qode@latest
```

Verify the installation:
```bash
qode --version
```

## Quick Start

```bash
# Onboard an existing project
cd your-project
qode init

# Set up IDE configs (Cursor + Claude Code)
qode ide setup

# Start a feature
qode branch create feat-user-dashboard
qode ticket fetch https://company.atlassian.net/browse/ENG-123
```

Do note that you will have to configure `.env` file with appropriate ticketing system API key or equivalent as described below.

## The Workflow

```
1. qode branch create <name>                 Create git branch + context folder
2. qode ticket fetch <url>                   Fetch ticket into context
   /qode-ticket-fetch <url>      (in IDE)    — or use the slash command
3. /qode-plan-refine             (in IDE)    Refine requirements — iterate to 25/25
4. /qode-plan-spec               (in IDE)    Generate tech spec
5. /qode-start                   (in IDE)    Run implementation prompt
6. /qode-review-code             (in IDE)    Code review
7. /qode-review-security         (in IDE)    Security review
8. qode check                                Run all quality gates
9. `/qode-knowledge-add-context` (in IDE)    (Recommended) Extract lessons learned
10. qode branch remove <name>                Cleanup
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

## Reviews

Code and security reviews use hardened prompts designed to prevent shallow outputs:

**Code review** (`/qode-review-code`):
- Reviewer reads the diff as if writing a post-mortem — find the failure before it ships
- Each file requires ≥ 3 documented items: defects, flagged concerns, or properties explicitly verified safe
- Score constraints: Critical finding → total ≤ 5.0 | High finding → total ≤ 7.5
- Scores ≥ 8.0 must cite specific properties verified, not the absence of bugs
- Score 10.0 requires justification against typical production-quality code

**Security review** (`/qode-review-security`):
- Reviewer maps every path from external input to persistent state or sensitive data
- Adversary Simulation section is required: three named exploit attempts with technique, target, and outcome
- Same score constraints apply
- Scores ≥ 8.0 must cite specific controls observed (e.g. parameterized queries at line X)
- Score 10.0 is not valid — complete security is not provable

Both prompts can be customised via `.qode/prompts/review/` local overrides.

## IDE Support

| IDE | What's generated |
|---|---|
| **Cursor** | `.cursorrules/*.mdc` rules + `.cursor/commands/*.mdc` slash commands |
| **Claude Code** | `CLAUDE.md` + `.claude/commands/*.md` slash commands |

Slash commands available in all IDEs: `/qode-ticket-fetch`, `/qode-plan-refine`, `/qode-plan-spec`, `/qode-start`, `/qode-review-code`, `/qode-review-security`, `/qode-knowledge-add-branch`

All configs are stack-aware. Use `qode ide sync` to regenerate after updating `qode.yaml`.

## Commands

```
qode init                                                      Detect stack, create qode.yaml
qode init --new                                                New project wizard
qode init --workspace                                          Multi-repo workspace setup

qode branch create <name>                                      Create feature branch + context folder
qode branch remove <name>                                      Clean up branch and context

qode ticket fetch <url>                                        Fetch ticket (Jira, Azure DevOps, Linear, GitHub Issues, Notion)

qode plan refine                                               Generate worker refinement prompt to stdout (use in IDE via /qode-plan-refine)
qode plan refine --to-file                                     Save worker prompt to file for debugging
qode plan judge                                                Generate judge scoring prompt to stdout (requires refined-analysis.md)
qode plan judge --to-file                                      Save judge prompt to file for debugging
qode plan spec                                                 Generate tech spec prompt to stdout (use in IDE via /qode-plan-spec)
qode plan spec --to-file                                       Save spec prompt to file for debugging

qode start                                                     Generate implementation prompt to stdout (use in IDE via /qode-start)
qode start --to-file                                           Save implementation prompt to file for debugging

qode review code                                               Generate code review prompt to stdout (use in IDE via /qode-review-code)
qode review security                                           Generate security review prompt to stdout (use in IDE via /qode-review-security)

qode check                                                     Run all quality gates per layer
qode check --layer <name>                                      Gates for a specific layer only
qode check --skip-tests                                        Reviews only, skip test execution

qode ide setup                                                 Generate IDE configs for all enabled IDEs
qode ide sync                                                  Regenerate configs from qode.yaml

qode knowledge add <path>                                      Add file to knowledge base
qode knowledge add-branch <name or comma separated names>      Add file to knowledge base
qode knowledge list                                            List knowledge base files
qode knowledge search <query>                                  Search knowledge base

qode workflow                                                  Show full workflow diagram
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

# GitHub Issues (public repos — no token required)
qode ticket fetch https://github.com/owner/repo/issues/42

# Notion
export NOTION_API_KEY=your-token
qode ticket fetch https://www.notion.so/workspace/My-Ticket-abc123de1234567890abcdef12345678
```

Credentials are auto-loaded from a `.env` file in the project root. See [docs/how-to-use-ticket-fetch.md](docs/how-to-use-ticket-fetch.md) for full setup instructions and token scope requirements.

## Further Reading

- [docs/versioning.md](docs/versioning.md) — Versioning strategy and release process
- [docs/how-to-use-ticket-fetch.md](docs/how-to-use-ticket-fetch.md) — Ticket system setup and token scopes
- [docs/qode-yaml-reference.md](docs/qode-yaml-reference.md) — Full `qode.yaml` configuration reference
- [docs/how-to-customise-prompts.md](docs/how-to-customise-prompts.md) — Override built-in prompt templates per project

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.
