# qode — Architecture Report

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

### Package map

| Package | Layer | Purpose |
|---|---|---|
| `cmd/qode` | Entry point | `main()` — delegates to `cli.Execute()` |
| `cli` | Orchestration | Cobra command tree; wires config → context → prompts → output |
| `config` | Core | Loads/validates/merges `qode.yaml` + `scoring.yaml` + user-level config |
| `branchcontext` | Core | Reads per-branch state from `.qode/branches/<branch>/` (ticket, analysis, spec, iterations, reviews) |
| `prompt` | Core | Template engine with local-override-first resolution (`go:embed` fallback); single `Renderer` interface; builder pattern for `TemplateData` |
| `scoring` | Core | Two-pass worker/judge scoring engine; rubric construction; YAML + regex score parsing |
| `workflow` | Core | Step-ordering enforcement (pure functions, no I/O) |
| `plan` | Domain | Builds refine, spec, start, and judge prompts; manages iteration files |
| `review` | Domain | Builds code-review and security-review prompts |
| `knowledge` | Domain | Manages project lessons-learned markdown files; search, list, save |
| `scaffold` | Domain | Generates IDE-specific configuration files for Cursor and Claude Code |
| `git` | Leaf | Thin wrappers around `git` CLI; branch ops, diffs, merge-base resolution |
| `iokit` | Leaf | File I/O utilities: atomic writes, directory helpers, safe reads |
| `env` | Leaf | `.env` file loading via `godotenv` |
| `log` | Leaf | Structured `slog`-based logging with `QODE_LOG_LEVEL` support |
| `version` | Leaf | Semver parsing and binary/config compatibility checks |

### Data flow

Config (`config`) → Branch context (`branchcontext`) → Prompt engine (`prompt`) → Domain builders (`plan`, `review`) → CLI commands (`cli`) → Output

- **Config** loads `qode.yaml`; merges project, scoring, and user-level configs
- **Branch context** reads per-branch state from `.qode/branches/<branch>/` (ticket, analysis, spec, reviews)
- **Prompt engine** resolves templates local-override-first: `.qode/prompts/` before `go:embed` fallback. `TemplateData` is the single struct for all templates
- **Workflow guards** (`workflow`) enforce step ordering — e.g. spec requires minimum refine score
- **Session** (`cli/session.go`) bootstraps the common state (root, config, branch, context, engine) used by most commands

### Dependency layering (MUST preserve)

Leaf packages (zero internal deps): `git`, `env`, `iokit`, `log`, `version`. Core packages depend only on leaves and each other where necessary. Domain packages depend on core + leaves. Only `cli` fans out to all packages. **Never introduce circular dependencies or upward imports.**

Current production dependency graph (non-test):
- `scoring` → `config`
- `prompt` → `scoring`
- `branchcontext` → `config`, `git`, `iokit`, `scoring`
- `workflow` → `config`, `branchcontext`, `scoring`
- `plan` → `branchcontext`, `config`, `git`, `iokit`, `prompt`, `scoring`
- `review` → `config`, `branchcontext`, `prompt`, `scoring`
- `knowledge` → `config`, `iokit`
- `scaffold` → `config`
- `cli` → all of the above

### Two-pass scoring

Reviews use a worker/judge split to eliminate self-scoring bias:

- **Worker**: produces analysis without a score
- **Judge**: scores the analysis against a configurable rubric independently

### Key design decisions

- Only one interface exists: `prompt.Renderer` — define interfaces only at consumption boundaries, not preemptively
- Use `iokit.AtomicWrite` for any file consumed by subsequent workflow steps
- Template override (local `.qode/prompts/` → embedded fallback) is the standard for user-extensible assets
- Dependencies are minimal (cobra, yaml.v3, godotenv) — prefer stdlib over new dependencies
- `Session` struct centralizes bootstrap; all commands use `loadSession()` to avoid repeated init logic

---

## Architectural Patterns

### Continue doing

1. **Strict dependency layering.** Leaf packages have zero internal deps; domain packages depend only on core and leaves; only `cli` fans out to everything. This keeps the compile graph shallow and prevents circular imports. *Example: `iokit` depends on nothing; `plan` depends on `prompt` and `scoring` but never on `cli`.*

2. **Single canonical interface at consumption boundary.** `prompt.Renderer` is the only interface, defined where consumers need it, not where implementors live. This avoids speculative abstraction and keeps the coupling explicit. *Example: `review.BuildCodePrompt` accepts `prompt.Renderer`, not `*prompt.Engine`.*

3. **Builder pattern for complex value objects.** `TemplateDataBuilder` provides fluent, method-chained construction of `TemplateData` — making it impossible to forget mandatory fields while keeping optional ones ergonomic. *Example: `prompt.NewTemplateData(name, branch).WithRubric(r).WithOutputPath(p).Build()`.*

4. **Workflow guards as pure functions.** `workflow.CheckStep` takes immutable state and returns a `CheckResult` — no side effects, fully testable, easy to reason about.

5. **Sentinel errors for common failure conditions.** `cli/errors.go` defines named errors (`ErrNotInitialised`, `ErrNoAnalysis`, `ErrNoSpec`, `ErrNoChanges`) enabling callers to match on specific conditions rather than parsing strings.

6. **Atomic file writes for pipeline artifacts.** Files consumed by subsequent workflow steps use `iokit.AtomicWrite` (temp + rename) to prevent partial writes from corrupting the pipeline.

7. **Template override ladder (local → embedded).** Users can customize prompts by dropping files in `.qode/prompts/` without modifying the binary. The engine checks local first, then falls back to `go:embed`.

8. **Minimal external dependencies.** Only three direct deps (cobra, yaml.v3, godotenv). Everything else is stdlib. This reduces supply-chain risk and keeps the binary small.

9. **Session centralizes bootstrap.** `loadSession()` wires root → config → branch → context → engine in one place, preventing duplication across commands and ensuring consistent initialization order.

10. **Good test coverage ratio.** ~175 test functions covering ~155 production functions. Tests exist for all core and domain packages.

### Stop doing

1. **Package-level mutable state in `cli`.** `root.go` uses `var rootCmd *cobra.Command` and `var flagRoot string` at package scope, mutated during `init()`. This makes the command tree untestable in isolation and ties the process to a single root command. *Example: `flagStrict` is a package-level bool read deep inside `runReview` — threading it through `Session` or method parameters would be cleaner.*

2. **Mixed output concerns in command functions.** Some commands write to `out` (stdout) for prompts and `errOut` (stderr) for status, but the decision of whether to write a file or stdout is handled by the same function with a `toFile` bool. This mixes I/O strategy into business logic. *Example: `runReview` both generates the prompt and decides how to deliver it.*

3. **Duplicated "resolve output path" logic across commands.** Each command (`plan.go`, `review.go`, `knowledge_cmd.go`) independently constructs `branchDir`, `promptPath`, and `outputPath` with similar `filepath.Join` patterns. This is not yet abstracted because each path differs slightly, but the pattern is repeated enough to warrant consolidation.

### Start doing

1. **Separate command wiring from business logic in `cli`.** Each `runXxx` function currently does both session bootstrap and domain orchestration. Extract the domain logic into the respective domain packages (e.g., `plan`, `review`) so that `cli` only handles flag parsing, session setup, and output routing. This would make the domain logic independently testable without cobra.

2. **Consolidate path construction for branch artifacts.** Branch artifact paths (e.g., `refined-analysis.md`, `code-review.md`, `.refine-prompt.md`) are computed by string concatenation across multiple packages. A single `branchcontext.Paths` type (or methods on `Context`) returning canonical paths would reduce duplication and prevent path drift.

3. **Make `Session` fields read-only after construction.** Currently `Session` fields are exported and mutable — any command handler can modify `sess.Config.Scoring.Strict = true`. Using a constructor that returns a frozen struct (or unexported fields with accessors) would make the data flow more predictable.

### Proposals for adoption (beneficial but optional)

1. **Functional options for Engine construction.** `NewEngine` currently takes only `root` and returns a fixed `funcMap`. If template functions need to grow (e.g., project-specific helpers), a functional options pattern (`WithFuncMap(...)`) would keep the constructor clean without a breaking API change.

2. **Structured error types for workflow violations.** `workflow.CheckResult` uses `Blocked bool` + `Message string`. A typed error (e.g., `StepBlockedError{Step, Reason, Remediation}`) would let callers programmatically distinguish violation kinds (missing prerequisite vs. insufficient score) rather than parsing messages.

3. **Table-driven command registration.** The `init()` block in `root.go` manually adds each subcommand. A registry pattern (slice of command descriptors) would make adding new commands more mechanical and reduce the chance of forgetting to wire one up.

---

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
