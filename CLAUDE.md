# qode

Go CLI that generates structured AI prompts for a standardized developer workflow. It does **not** run AI — it assembles context and renders prompt templates for AI IDEs (Cursor, Claude Code, etc.).

## Commands

```bash
go build ./...                                                           # Build
go test ./...                                                            # Unit tests (<2s)
go test -race ./...                                                      # Unit tests + race detection
go test -tags integration ./...                                          # Integration tests (separate)
go test ./internal/<pkg>/... -run TestName                               # Single test
go test -run TestGolden_Templates ./internal/prompt/ -args -update 2>&1  # Regenerate golden files
golangci-lint run                                                        # Lint
go install ./cmd/qode/                                                   # Install locally
```

CI enforces **minimum 70% coverage** with race detection.

## Architecture

Config (`config`) → Branch context (`branchcontext`) → Prompt engine (`prompt`) → Domain builders (`plan`, `review`) → CLI commands (`cli`) → Output

### Dependency layering — MUST preserve

```text
Leaf (zero internal deps): git, env, iokit, log, version
Mid-level:                 config → iokit; scoring → config; knowledge → config, iokit
Domain:                    branchcontext, prompt, workflow, plan, review, scaffold
Top-level (fan-out):       cli → ALL packages
```

**Only `cli` fans out. Never introduce circular deps or upward imports.** Every new package must declare its layer.

### Design decisions

- **One interface**: `prompt.Renderer` — define interfaces only at consumption boundaries, not preemptively
- **Atomic writes**: use `iokit.AtomicWrite` for any file consumed by subsequent workflow steps
- **Template override**: local `.qode/prompts/` → `go:embed` fallback for user-extensible assets
- **Fluent builder**: `TemplateDataBuilder` with `.WithXxx().Build()` for template data construction
- **Context threading**: every function performing I/O or calling a subprocess must accept `context.Context` as first parameter. New code uses context-accepting signatures directly; callers without a context pass `context.Background()`
- **Minimal deps**: only cobra, yaml.v3, godotenv — prefer stdlib; reject convenience-only deps
- **Two-pass scoring**: worker produces analysis (no score), judge scores independently against configurable rubric
- **Sentinel errors**: export sentinel errors (`ErrConfigNotFound`, etc.) for programmatic distinction; match with `errors.Is()`

## Code standards

- Functions ≤ 50 lines, single responsibility
- Named constants — no magic numbers
- Wrap errors with `%w` — never swallow errors
- No TODO comments in committed code
- Push domain logic into dedicated packages, not `cli/`
- Follow existing patterns; do not introduce new ones

## Test standards

Default shape: **table-driven** with `t.Run(tc.name, ...)` and `t.Parallel()` on parent and subtests, unless test mutates global state.

- `t.Helper()` on every helper, `t.Cleanup` for teardown, `t.TempDir()` for filesystem tests
- Never mock what you own — test real implementations; mock only at system boundaries (network, external processes)
- Golden files for template/structured output — always support `-update` flag
- Error paths must assert error type (`errors.Is`) or message content, not just `err != nil`
- Integration tests behind `//go:build integration` — create fresh command instances, never reset globals
- Sentinel assertions for prompt content — inject unique strings, assert presence/absence
- One assertion theme per test function — if a name needs "and", split into two tests

## Quality standards

- Minimum refined analysis score: 25/25
- Minimum code review score: 10.0/12
- Minimum security review score: 10.0/12

## Gotchas

- IMPORTANT: Never change `CLAUDE.md` file
- If asked to add something to `notes` or `notes.md`, always append to `.qode/contexts/current/notes.md`
- `.qode/`, `.claude/`, `.cursor/`, `.cursorrules/` directories and `qode.yaml` are configuration — only read when testing changes to these files, never modify directly (use `qode init` instead)
