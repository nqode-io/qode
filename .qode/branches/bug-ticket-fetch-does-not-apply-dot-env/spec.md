# Technical Specification — Load `.env` at CLI Startup

**Branch:** `bug-ticket-fetch-does-not-apply-dot-env`
**Date:** 2026-03-04
**Status:** Ready for implementation

---

## 1. Feature Overview

When `qode` CLI commands are invoked from an IDE extension (VS Code, Cursor, etc.), the process does not inherit the user's interactive shell environment. As a result, environment variables such as `GITHUB_TOKEN`, `JIRA_API_TOKEN`, `LINEAR_API_KEY`, and `AZURE_DEVOPS_PAT` are unavailable, causing commands like `qode ticket fetch` to fail with authentication errors.

This change adds transparent `.env` file loading at CLI startup. Before any command executes, `qode` looks for a `.env` file in the project root and loads it into the process environment, without overriding variables already present in the environment. This makes `qode` usable from IDEs without any additional shell configuration.

**Success criteria:**
- `qode ticket fetch <url>` succeeds when the required token is in `.env` but absent from the shell environment.
- No regressions when tokens are provided via shell exports or CI environment variables.
- Absence of `.env` is not an error.

---

## 2. Scope

### In Scope
- Load `.env` from the project root (directory containing `qode.yaml`) at program startup.
- Fall back to current working directory if no project root is found (e.g., `qode init`).
- Non-override semantics: existing env vars take precedence over `.env` values.
- New `internal/env` package with a single `Load` function and unit tests.
- Add `github.com/joho/godotenv` as a dependency.
- Confirm `.env` is present in `.gitignore` (already confirmed: line 28 of `.gitignore`).

### Out of Scope
- `.env.local`, `.env.production`, or other variant files.
- Per-user `~/.qode/.env` support.
- Changes to ticket providers — they continue using `os.Getenv()`.
- Changes to IDE slash command files.
- Secrets management integrations (Vault, AWS SSM, etc.).

### Assumptions
- The `.env` file uses standard `KEY=VALUE` format (with or without `export` prefix; both are supported by `godotenv`).
- The project root is the directory containing `qode.yaml`, as determined by `config.FindRoot`.
- `.env` files contain no sensitive side effects (no shell execution, no variable expansion beyond simple `$VAR` references).

---

## 3. Architecture & Design

### Component Overview

```
cmd/qode/main.go
  │
  ├─ config.FindRoot(".")          ← resolves project root via qode.yaml walk-up
  │    (internal/config/config.go:FindRoot)
  │
  ├─ env.Load(projectRoot)         ← NEW: loads .env non-destructively
  │    (internal/env/env.go)
  │         │
  │         └─ godotenv.Load(path) ← third-party: skips existing env vars
  │
  └─ cli.Execute()
       │
       └─ [all subcommands]
            └─ internal/ticket/*.go  ← unchanged; still uses os.Getenv()
```

### Affected Layers

| Layer | File | Change |
|-------|------|--------|
| Entry point | `cmd/qode/main.go` | Add root detection + `env.Load()` call before `cli.Execute()` |
| New package | `internal/env/env.go` | New file: `Load(projectRoot string) error` |
| New tests | `internal/env/env_test.go` | New file: unit tests for loader |
| Dependencies | `go.mod`, `go.sum` | Add `github.com/joho/godotenv` |

**No changes** to `internal/ticket/`, `internal/cli/`, or `internal/dispatch/`.

### Data Flow

```
IDE invokes: qode ticket fetch https://github.com/org/repo/issues/42
                │
                ▼
        main() starts
                │
                ▼
        config.FindRoot(".")  →  "/path/to/project"
                │
                ▼
        env.Load("/path/to/project")
          - reads /path/to/project/.env
          - for each KEY=VALUE: if os.Getenv(KEY) == "" → os.Setenv(KEY, VALUE)
                │
                ▼
        cli.Execute()  →  ticket fetch command
          - ticket/github.go calls os.Getenv("GITHUB_TOKEN")  ← now populated
```

---

## 4. API / Interface Contracts

### New Function: `env.Load`

**Package:** `internal/env`
**File:** `internal/env/env.go`

```go
// Load reads a .env file from projectRoot and sets any variables that are
// not already present in the environment. If projectRoot is empty, the
// current working directory is used. A missing .env file is not an error.
func Load(projectRoot string) error
```

**Behaviour:**
- If `projectRoot` is empty, substitutes `os.Getwd()`.
- Constructs path: `filepath.Join(projectRoot, ".env")`.
- If the file does not exist (`os.IsNotExist`), returns `nil`.
- Calls `godotenv.Read(path)` to parse the file, then for each key, sets it via `os.Setenv` only if `os.Getenv(key) == ""`.
- If the file exists but cannot be parsed, returns the wrapped error (caller logs and continues).

**Why `godotenv.Read` + manual `os.Setenv` instead of `godotenv.Load`:**
`godotenv.Load` does not override existing vars, which is the desired behaviour. However, using `godotenv.Read` + manual loop gives explicit control and makes the non-override semantics visible in code.

---

## 5. Data Model Changes

None. This change is purely runtime environment population. No files, databases, or config schemas are modified.

---

## 6. Implementation Tasks

- [ ] **Task 1** (deps): Add `github.com/joho/godotenv` dependency
  - Run: `go get github.com/joho/godotenv`
  - Files: `go.mod`, `go.sum`

- [ ] **Task 2** (internal/env): Create `internal/env/env.go` with `Load` function
  - Package `env`, single exported function matching the contract in §4
  - Files: `internal/env/env.go`

- [ ] **Task 3** (internal/env): Create `internal/env/env_test.go` with unit tests
  - See §7 for test cases
  - Files: `internal/env/env_test.go`

- [ ] **Task 4** (cmd): Wire `.env` loading into `main()` before `cli.Execute()`
  - Call `config.FindRoot(".")` — on error, fall back to `os.Getwd()`
  - Call `env.Load(projectRoot)` — on error, print warning to stderr and continue
  - Files: `cmd/qode/main.go`

---

## 7. Testing Strategy

### Unit Tests (`internal/env/env_test.go`)

| Test | Setup | Expected |
|------|-------|----------|
| Missing `.env` | No `.env` file in temp dir | Returns `nil`; no env vars changed |
| Present, var unset | `.env` with `FOO=bar`; `FOO` not in env | `os.Getenv("FOO") == "bar"` after `Load` |
| Present, var already set | `.env` with `FOO=bar`; `os.Setenv("FOO", "existing")` first | `os.Getenv("FOO") == "existing"` after `Load` (not overridden) |
| Malformed line | `.env` with `=NOKEY` | Returns non-nil error |
| Empty file | `.env` exists but is empty | Returns `nil`; no env vars changed |
| Multiple vars | `.env` with `A=1\nB=2` | Both `A` and `B` set correctly |

### Integration Tests

No new integration tests required. The existing `qode ticket fetch` integration tests (if any) cover the end-to-end path; the unit tests above are sufficient for this isolated change.

### E2E / Manual Verification

1. **Happy path:** Create `.env` with `GITHUB_TOKEN=<valid-token>`, unset `GITHUB_TOKEN` from shell, run `qode ticket fetch <github-url>` — ticket should be fetched successfully.
2. **Shell override wins:** Set `GITHUB_TOKEN=shell-token` in shell and `GITHUB_TOKEN=env-token` in `.env` — `os.Getenv("GITHUB_TOKEN")` should return `shell-token`.
3. **No `.env` present:** Remove `.env`, run any `qode` command — no error, normal behavior.
4. **IDE invocation:** Invoke from VS Code terminal (which may not have shell exports) — tokens in `.env` should work.

---

## 8. Security Considerations

### `.gitignore` Status
`.env` is already listed in `.gitignore` at line 28. No action needed.

### Token Exposure
- Tokens are loaded into the process environment. They are visible to child processes spawned by `qode` (e.g., `qode`'s Claude dispatcher at `internal/dispatch/claude.go` passes `os.Environ()` to the Claude subprocess). This is intentional and consistent with standard `.env` tooling behaviour.
- Tokens are not written to any log files, output files, or the `.qode/` state directory.

### Input Validation
- The `.env` file path is constructed via `filepath.Join` from a trusted source (project root resolved by `config.FindRoot`). No user-controlled string is used directly in file path construction.
- File parsing is delegated to `godotenv`, which is a well-maintained library. Variable values are treated as opaque strings; no shell execution or command substitution occurs.

### File Permissions
If the `.env` file has permissions more permissive than `0600` (e.g., world-readable), consider logging a warning. This is optional for this iteration and can be added as a follow-up.

---

## 9. Open Questions

None. All ambiguities from requirements analysis have been resolved:

| Question | Resolution |
|----------|-----------|
| Override semantics | Non-override: `.env` values do not replace existing env vars |
| File search location | Project root (`qode.yaml` directory), falling back to cwd |
| Scope of fix | Applied at startup — all subcommands benefit automatically |
| Other affected subcommands | Only ticket providers use `os.Getenv()` today; startup fix covers all future cases too |
| `.env` already in `.gitignore`? | Yes — line 28 of `.gitignore` |

---

*Spec generated by qode. Copy to GitHub issue / ticket for team review.*
