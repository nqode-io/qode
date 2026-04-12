# qode Architecture Report

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

### Directory layout

```
cmd/qode/          Entry point (main.go)
internal/
  cli/             Cobra command definitions — the only fan-out package
  config/          Loads, validates, and normalizes qode.yaml
  qodecontext/     Per-context state from .qode/contexts/<name>/
  prompt/          Template engine with go:embed + local-override resolution
  plan/            Builds refine, spec, and start prompts
  review/          Builds code-review and security-review prompts
  scoring/         Rubrics, two-pass worker/judge scoring
  workflow/        Step-ordering guards (pure, no I/O)
  knowledge/       Project lessons-learned markdown loader
  scaffold/        IDE configuration generator (qode init)
  env/             .env file loader
  iokit/           File I/O utilities, atomic writes
  log/             Structured logging (slog)
  version/         Semver parsing, compatibility checks
docs/              User-facing documentation
tools/             Release scripts
```

### Data flow

Config (`config`) -> Context (`qodecontext`) -> Prompt engine (`prompt`) -> Domain builders (`plan`, `review`) -> CLI commands (`cli`) -> Output

- **Config** loads `qode.yaml`; provides scoring rubrics, review thresholds, diff command, and IDE settings
- **Context** reads per-context state from `.qode/contexts/<name>/` via the `current` symlink (ticket, analysis, spec, reviews)
- **Prompt engine** resolves templates local-override-first: `.qode/prompts/` before `go:embed` fallback. `TemplateData` is the single struct for all templates
- **Workflow guards** (`workflow`) enforce step ordering — e.g. spec requires minimum refine score

### Two-pass scoring

Reviews use a worker/judge split to eliminate self-scoring bias:

- **Worker**: produces analysis without a score
- **Judge**: scores the analysis against a configurable rubric independently

### Dependency graph (production code only)

```
Leaf packages (zero internal deps):
  env, iokit, log, version

Mid-level:
  config      -> iokit
  scoring     -> config
  knowledge   -> config, iokit

Domain:
  qodecontext -> config, iokit
  prompt      -> config, scoring
  workflow    -> qodecontext, config, scoring
  plan        -> qodecontext, config, iokit, prompt, scoring
  review      -> qodecontext, config, prompt, scoring
  scaffold    -> config, iokit, prompt

Top-level (fan-out):
  cli         -> ALL packages
```

**Rule: Only `cli` fans out to all packages. Never introduce circular deps or upward imports.**

### Key design decisions

- Only one interface exists: `prompt.Renderer` — define interfaces only at consumption boundaries, not preemptively
- Use `iokit.AtomicWrite` for any file consumed by subsequent workflow steps
- Template override (local `.qode/prompts/` -> embedded fallback) is the standard for user-extensible assets
- `TemplateDataBuilder` uses a fluent builder pattern (`.WithXxx().Build()`) for constructing template data
- Dependencies are minimal (cobra, yaml.v3, godotenv) — prefer stdlib over new dependencies
- `context.Context` is threaded through CLI commands and I/O-performing functions

### External dependencies

Only three direct dependencies: `cobra`, `yaml.v3`, `godotenv`. Stdlib is strongly preferred.

## Code standards

- Functions <= 50 lines, single responsibility
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
- If asked to add something to `notes` or `notes.md`, always append to `.qode/contexts/current/notes.md`
- The `.qode/`, `.claude/`, `.cursor/`, `.cursorrules/` directories and `qode.yaml` are configuration — only read them when testing changes that affect these files, never modify directly (use `qode init` instead)

---

## Architectural Patterns Assessment

### Continue doing

1. **Strict dependency layering** — Leaf packages have zero internal imports; domain packages depend only on config and leaves; only `cli` fans out. This prevents circular dependencies and keeps compilation fast. _Apply: every new package must declare its layer; reject PRs that add upward imports._

2. **Single interface at consumption boundary** — `prompt.Renderer` is the only interface, defined where it is consumed, not where it is implemented. This avoids premature abstraction and keeps the codebase concrete. _Apply: introduce a new interface only when a second implementation or a test double at a package boundary demands it._

3. **Fluent builder for template data** — `TemplateDataBuilder` with `WithXxx()` chains keeps template construction readable and extensible without constructor bloat. _Apply: prefer builders over large parameter lists or option structs when the number of optional fields exceeds 3-4._

4. **Atomic file writes** — `iokit.AtomicWrite` (temp + rename) prevents partial writes that would corrupt state consumed by later workflow steps. _Apply: always use `AtomicWrite` for any file that is an input to a subsequent step._

5. **Template override chain** — Local `.qode/prompts/` first, then `go:embed` fallback. Users can customize without forking. _Apply: any user-extensible asset should follow the same local-then-embedded resolution pattern._

6. **Minimal external dependencies** — Three direct deps (cobra, yaml.v3, godotenv). Stdlib preferred. _Apply: any new dependency must justify itself against a stdlib alternative; reject convenience-only deps._

7. **Parallel-safe tests** — Widespread use of `t.Parallel()` keeps the test suite fast and flushes out shared-state bugs early. _Apply: every new test function must call `t.Parallel()` unless it mutates truly global state._

8. **Sentinel errors at package boundaries** — `ErrConfigNotFound`, `ErrNoBaseBranch`, `ErrEmptyJudgment` are exported, enabling callers to use `errors.Is()`. _Apply: define sentinel errors for conditions that callers must programmatically distinguish._

9. **Context threading** — `context.Context` is passed through CLI -> domain -> I/O layers, enabling cancellation and timeout propagation. _Apply: every function that performs I/O or calls a subprocess must accept a context as its first parameter._

### Stop doing

1. **Oversized CLI functions** — Several `cli/` functions exceed the 50-line guideline (e.g. `runInitExisting` at 71, `runReview` at 62). This pushes domain logic into the command layer. _Fix: extract the domain logic into the relevant domain package, leaving `cli/` as a thin orchestration shell._

2. **Self-referential internal imports** — `iokit` and `version` import themselves (likely via sub-files referencing sibling symbols through the full module path). While technically harmless, it obscures the dependency graph. _Fix: use package-local references instead of fully-qualified self-imports._

### Start doing

1. **Consistent `Ctx` suffix convention** — Some functions have both `Foo` and `FooCtx` variants (e.g. `AtomicWrite` / `AtomicWriteCtx`). New code should accept `context.Context` as the primary signature and drop the non-context variant over time. _Apply: default to context-accepting signatures; callers that lack a context should pass `context.Background()` at the call site, not force a wrapper function._

2. **Structured error types for domain failures** — Currently only sentinel `var` errors exist. When errors carry structured data (iteration number, file path, threshold vs actual score), a typed error struct would enable richer handling without string parsing. _Apply: when an error must carry context beyond a message, define a struct type implementing `error`._

3. **Package-level doc comments** — Most packages have a doc comment (`// Package foo ...`) but not all. _Apply: every package must have a `doc.go` or a comment on the `package` line describing its responsibility in one sentence._

### Proposals for adoption (beneficial but not mandatory)

1. **Table-driven CLI integration tests** — The CLI layer is the largest package (~1363 LOC) but testing individual command flows end-to-end via table-driven tests against a temp directory would catch regressions cheaply without needing mocks. _Benefit: high coverage of the orchestration layer with low maintenance cost._

2. **Functional options for complex constructors** — Where the builder pattern becomes cumbersome (many optional fields with validation), consider functional options (`func WithFoo(v T) Option`) which allow validation at construction time. _Benefit: compile-time safety and self-documenting option names._

3. **Internal `testutil` package** — Shared test fixtures (temp dirs, sample configs, branch context builders) are likely duplicated across test files. A small `internal/testutil` package would reduce boilerplate. _Benefit: DRY test setup without leaking test helpers into production code._

4. **Makefile or Taskfile** — The project relies on raw `go` commands. A thin `Makefile` (or `Taskfile.yml`) wrapping build, test, lint, and release would standardize CI and developer workflows. _Benefit: single entry point for all common operations._
