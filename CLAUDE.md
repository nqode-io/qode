# qode

Go CLI that generates structured AI prompts for a standardized developer workflow. It does **not** run AI — it assembles context and renders prompt templates for AI IDEs (Cursor, Claude Code, etc.).

## Commands

```bash
go build ./...                                          # Build
go test ./...                                           # All tests
go test ./internal/<package>/... -run TestFunctionName  # Single test
golangci-lint run                                       # Lint
go install ./cmd/qode/                                  # Install locally
```

## Architecture

### Data flow

Config (`config`) → Branch context (`branchcontext`) → Prompt engine (`prompt`) → Domain builders (`plan`, `review`) → CLI commands (`cli`) → Output

- **Config** loads `qode.yaml`; normalizes shorthand (`stack:`) and multi-layer (`layers:[]`) into `[]LayerConfig`
- **Branch context** reads per-branch state from `.qode/branches/<branch>/` (ticket, analysis, spec, reviews)
- **Prompt engine** resolves templates local-override-first: `.qode/prompts/` before `go:embed` fallback. `TemplateData` is the single struct for all templates
- **Workflow guards** (`workflow`) enforce step ordering — e.g. spec requires minimum refine score

### Two-pass scoring

Reviews use a worker/judge split to eliminate self-scoring bias:

- **Worker**: produces analysis without a score
- **Judge**: scores the analysis against a configurable rubric independently

### Dependency layering (MUST preserve)

Leaf packages (zero internal deps): `git`, `env`, `iokit`, `log`, `version`. Domain packages depend only on `config` and leaves. Only `cli` fans out to all packages. **Never introduce circular dependencies or upward imports.**

### Key design decisions

- Only one interface exists: `prompt.Renderer` — define interfaces only at consumption boundaries, not preemptively
- Use `iokit.AtomicWrite` for any file consumed by subsequent workflow steps
- Template override (local `.qode/prompts/` → embedded fallback) is the standard for user-extensible assets
- Dependencies are minimal (cobra, yaml.v3, godotenv) — prefer stdlib over new dependencies

## Code standards

- Functions ≤ 50 lines, single responsibility
- Named constants — no magic numbers
- Explicit error handling — never swallow errors; wrap with `%w` for context
- No TODO comments in committed code
- Follow existing patterns; do not introduce new ones
- Push domain logic into dedicated packages, not into `cli/` command files

## Quality standards

- Minimum refined analysis score: 25/25
- Minimum code review score: 10.0/12
- Minimum security review score: 10.0/12

## Gotchas

- IMPORTANT: Never change `CLAUDE.md` file
- If asked to add something to `notes` or `notes.md`, always append to `.qode/branches/$(git branch --show-current | sed 's|/|--|g')/context/notes.md`
- The `.qode/`, `.claude/`, `.cursor/`, `.cursorrules/` directories and `qode.yaml` are configuration — only read them when testing changes that affect these files, never modify directly (use `qode init` instead)
