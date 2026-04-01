# qode — Project Context

qode is an AI-assisted developer workflow CLI written in Go. It standardises how developers use AI coding assistants across projects with varied tech stacks.

## Tech Stack

- **default** (go): `.`

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run a single test
go test ./internal/<package>/... -run TestFunctionName

# Lint
golangci-lint run

# Install binary locally
go install ./cmd/qode/
```

## Architecture

**qode** is a Go CLI (`cmd/qode/main.go` → `internal/cli`) that generates structured AI prompts for a standardized feature development workflow. It does not run AI itself — it assembles context and renders prompt templates that developers paste into their AI IDE (Cursor, Claude Code, etc.).

### Core data flow

1. **Config** (`internal/config`) — loads `qode.yaml` from the project root. `Config.Layers()` normalizes both shorthand (`stack:`) and multi-layer (`layers:[]`) forms into `[]LayerConfig`.

2. **Branch context** (`internal/context`) — each feature branch gets a folder at `.qode/branches/<branch>/`. Files here (ticket.md, refined-analysis.md, spec.md, code-review.md, security-review.md) are the stateful inputs that get injected into prompts.

3. **Prompt engine** (`internal/prompt/engine.go`) — `Engine.Render(name, data)` resolves templates with a local-override-first strategy: checks `.qode/prompts/<name>.md.tmpl` before falling back to embedded templates (`//go:embed templates`). `TemplateData` is the single struct passed to every template.

4. **CLI commands** (`internal/cli`) — each command loads config, resolves branch context, populates `TemplateData`, and calls the engine. Rendered prompts are either printed to stdout or written to `.qode/branches/<branch>/.refine-prompt.md` via `writePromptToFile` (atomic rename).

### Two-pass scoring

Reviews use a worker/judge split to eliminate AI self-scoring bias:

- **Worker pass** (`/qode-review-code`, `/qode-review-security`): produces analysis without a score
- **Judge pass** (separate template): scores the analysis against a rubric independently

Scores are parsed from saved markdown files in the branch context folder.

### Key directories

- `internal/prompt/templates/` — embedded Go templates (`.md.tmpl`) organized by workflow step: `refine/`, `spec/`, `start/`, `review/`, `scoring/`, `knowledge/`

## Code standards

- Read existing code before writing new code
- Functions ≤ 50 lines, single responsibility
- Named constants — no magic numbers
- Explicit error handling — never swallow errors
- No TODO comments in committed code
- Follow existing patterns; do not introduce new ones

## Quality Standards

- Minimum refined analysis score: 25/25
- Minimum code review score: 10.0/12
- Minimum security review score: 10.0/12

## Additional instructions

- If asked by the the user to add something to `notes` or `notes.md` file, always append it to the `.qode/branches/$(git branch --show-current | sed 's|/|--|g')/context/notes.md` file
- Never change `CLAUDE.md` file
- If running `/qode-ticket-fetch` do what is described in the command, do not automatically call the MCP server.
