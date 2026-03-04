# qode — Project Context

qode is an AI-assisted developer workflow CLI written in Go. It standardises how developers use AI coding assistants across projects with varied tech stacks.

## Tech Stack

- **default** (go): `.`

## Key Packages

| Package | Purpose |
|---|---|
| `internal/cli/` | Cobra command implementations |
| `internal/dispatch/` | Prompt dispatch — runs `claude` CLI interactively |
| `internal/prompt/` | Template engine — embedded templates with per-project `.qode/prompts/` overrides |
| `internal/config/` | `qode.yaml` loading and merging |
| `internal/ticket/` | Ticket provider integrations (Jira, Azure DevOps, Linear, GitHub) |
| `internal/runner/` | Quality gate execution (tests, lint, AI reviews) |

## Project Structure

Topology: single

- `./` — default (go)

## Development Workflow

**Terminal commands:**
1. `qode branch create <name>` — Create feature branch
2. `qode ticket fetch <url>` — Fetch ticket context

**IDE slash commands (Cursor / Claude Code):**
3. `/qode-plan-refine` — Iterate requirements (target 25/25)
4. `/qode-plan-spec` — Generate tech spec

**Either terminal or IDE:**
5. `qode start` / `/qode-start` — Run implementation prompt (launches interactive Claude session)
6. `/qode-review-code` + `/qode-review-security` — Reviews

**Terminal commands:**
7. `qode check` — All quality gates
8. `git commit && git push` — Ship
9. `qode branch remove <name>` — Cleanup

## Quality Standards

- Minimum code review score: 8.0/10
- Minimum security review score: 8.0/10
- Max function length: 50 lines

## Clean Code Rules

- Read existing code before writing new code
- Follow patterns in existing files — do not introduce new patterns
- Functions max 50 lines, single responsibility
- Handle all errors explicitly
- No magic numbers — use named constants
- No TODO comments in committed code
