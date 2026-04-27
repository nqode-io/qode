# qode

AI-assisted developer workflow CLI by nQode.

[![CI](https://github.com/nqode-io/qode/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nqode-io/qode/actions/workflows/ci.yml)
[![Release](https://github.com/nqode-io/qode/actions/workflows/release.yml/badge.svg?branch=main)](https://github.com/nqode-io/qode/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/nqode-io/qode?include_prereleases&sort=semver)](https://github.com/nqode-io/qode/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/nqode-io/qode)](https://goreportcard.com/report/github.com/nqode-io/qode)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Standardises how developers use AI coding assistants across client projects with varied tech stacks — Next.js+React, .NET+React, Angular+Java, and more.

## Installation

### macOS

```bash
brew install nqode-io/tap/qode
```

### Linux

```bash
curl -sSfL https://raw.githubusercontent.com/nqode-io/qode/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/nqode-io/qode/main/install.ps1 | iex
```

Windows SmartScreen may block the first run of `qode.exe`. Click **More info** → **Run anyway**.

### Verify and self-update

```bash
qode --version
```

Re-run the one-liner for your platform to upgrade to the latest release.

### Alternative — install from source

Requires Go 1.24+:

```bash
go install github.com/nqode-io/qode/cmd/qode@latest
```

### Advanced — supply-chain verification

Tagged releases sign `checksums.txt` with [cosign](https://github.com/sigstore/cosign) keyless OIDC. To verify, download the checksums and signature artifacts from the release page, then run `cosign verify-blob` from the directory containing them:

```bash
# Replace with the tag you want to verify
TAG=v0.1.0-beta
BASE="https://github.com/nqode-io/qode/releases/download/${TAG}"
curl -sSfLO "${BASE}/checksums.txt"
curl -sSfLO "${BASE}/checksums.txt.sig"
curl -sSfLO "${BASE}/checksums.txt.pem"

cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/nqode-io/qode/\.github/workflows/release\.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --signature checksums.txt.sig \
  --certificate checksums.txt.pem \
  checksums.txt
```

## Quick Start

```bash
# Onboard an existing project (writes qode.yaml and generates IDE configs)
cd your-project
qode init

# Start a feature
qode context init feat-user-dashboard --auto-switch
# Then use /qode-ticket-fetch <url> in your IDE to fetch the ticket via MCP
```

## The Workflow

Before beginning, manually create a new branch for your work.

```markdown
1.  qode context init <name>                 Create a named work context
2.  /qode-ticket-fetch <url>      (in IDE)   Fetch ticket via MCP into context
3.  /qode-plan-refine             (in IDE)   Refine requirements — worker + scoring pass
4.  /qode-plan-spec               (in IDE)   Generate tech spec
5.  /qode-start                   (in IDE)   Run implementation prompt
6.  Test locally                  (manual)   Verify the change behaves as expected
7.  /qode-check                   (in IDE)   Run quality gates (tests + lint)
8.  /qode-review-code             (in IDE)   Code review
    /qode-review-security         (in IDE)   Security review
9.  /qode-pr-create               (in IDE)   Create pull request via MCP
10. /qode-pr-resolve              (in IDE)   Resolve PR review comments via MCP
11. /qode-knowledge-add-context   (in IDE)   Capture lessons learned (optional)
12. qode context remove                      Cleanup
```

Run `qode workflow` for the full diagram. `qode workflow status` shows live completion status for the active context.

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

qode supports three IDEs out of the box. All receive the same slash-command catalog; only the on-disk format differs.

|                       | Cursor                           | Claude Code                       | Codex                            |
| --------------------- | -------------------------------- | --------------------------------- | -------------------------------- |
| Generated assets      | `.cursor/commands/*.mdc`         | `.claude/commands/*.md`           | `.codex/commands/*.md`           |
| Enable in `qode.yaml` | `ide.cursor.enabled: true`       | `ide.claude_code.enabled: true`   | `ide.codex.enabled: true`        |
| Regenerate            | Run `qode init` after toggling   | Run `qode init` after toggling    | Run `qode init` after toggling   |

Slash commands available in all IDEs:

- `/qode-ticket-fetch <url>` — fetch ticket via MCP
- `/qode-plan-refine` — refine requirements (worker + scoring pass)
- `/qode-plan-spec` — generate tech spec
- `/qode-start` — run implementation prompt
- `/qode-check` — quality gates (tests + lint)
- `/qode-review-code` — code review
- `/qode-review-security` — security review
- `/qode-pr-create` — create pull request via MCP
- `/qode-pr-resolve` — resolve PR review comments via MCP
- `/qode-knowledge-add-context` — capture lessons learned

Run `qode init` after toggling enablement in `qode.yaml` to regenerate the IDE assets.

## Commands

```markdown
qode init                                                      Initialise qode: write qode.yaml, create .qode/ dirs, generate IDE configs

qode context init <name> [--auto-switch]                        Create a new context (--auto-switch to activate it)
qode context switch <name>                                     Switch the active context
qode context clear [name]                                      Clear a context's files, reinitialising stubs
qode context remove [name]                                     Remove a context directory
qode context reset                                             Clear the active context selection

qode plan refine                                               Generate worker refinement prompt to stdout (use in IDE via /qode-plan-refine)
qode plan refine --to-file                                     Save worker prompt to file for debugging
qode plan judge                                                Generate judge scoring prompt to stdout (requires refined-analysis.md)
qode plan judge --to-file                                      Save judge prompt to file for debugging
qode plan spec                                                 Generate tech spec prompt to stdout (use in IDE via /qode-plan-spec)
qode plan spec --force                                         Bypass score gate (prerequisite check still applies)
qode plan spec --to-file                                       Save spec prompt to file for debugging

qode start                                                     Generate implementation prompt to stdout (use in IDE via /qode-start)
qode start --force                                             Bypass spec prerequisite gate
qode start --to-file                                           Save implementation prompt to file for debugging

qode review code                                               Generate code review prompt to stdout (use in IDE via /qode-review-code)
qode review code --force                                       Bypass uncommitted-diff check
qode review security                                           Generate security review prompt to stdout (use in IDE via /qode-review-security)
qode review security --force                                   Bypass uncommitted-diff check

qode knowledge add <path>                                      Add file to knowledge base
qode knowledge add-context                                     Generate a lesson extraction prompt from the current context
qode knowledge list                                            List knowledge base files
qode knowledge search <query>                                  Search knowledge base

qode workflow                                                  Show full workflow diagram
qode workflow status                                           Show live completion status for the current context
```

## Project Setup

```markdown
my-project/
├── qode.yaml
├── src/
└── package.json
```

```bash
cd my-project && qode init
```

## Ticket Fetch via MCP

Ticket fetching uses IDE-native MCP servers — no API keys in qode itself. Configure the MCP server for your ticketing system (Jira, Linear, GitHub, Azure DevOps, Notion) and linked-resource services (Figma, Google Docs, Confluence, etc.) in your IDE, then use `/qode-ticket-fetch <url>` in Cursor, Claude Code, or Codex.

See [docs/how-to-use-ticket-fetch.md](docs/how-to-use-ticket-fetch.md) for full MCP setup instructions per service.

## Further Reading

- [docs/tutorial.md](docs/tutorial.md) — End-to-end walkthrough: one ticket, one branch, two subtasks, every workflow step
- [docs/versioning.md](docs/versioning.md) — Versioning strategy and release process
- [docs/how-to-use-ticket-fetch.md](docs/how-to-use-ticket-fetch.md) — Ticket system setup and token scopes
- [docs/qode-yaml-reference.md](docs/qode-yaml-reference.md) — Full `qode.yaml` configuration reference
- [docs/how-to-customise-prompts.md](docs/how-to-customise-prompts.md) — Override built-in prompt templates per project
- [CHANGELOG.md](CHANGELOG.md) — Release history

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to abide by its terms.
