# Technical Specification — Remove Unused CLI Flags

**Branch:** remove-unused-flags
**Issue:** nqode-io/qode#44
**Date:** 2026-03-29

---

## 1. Feature Overview

Several `qode` CLI flags have accumulated that are either never used in practice, make the API unnecessarily complex, or encode logic that belongs in configuration or positional arguments. This change removes eight flags across six commands, replaces two of them with cleaner alternatives (a positional argument and a config-backed flag), and deletes all resulting dead code.

The result is a CLI where the common path requires no flags at all, overrides are intentional and self-documenting, and the surface area engineers must memorise is reduced to the minimum.

**Success criteria:**
- All eight listed flags are unrecognisable by cobra (attempts to use them produce a "unknown flag" error)
- `qode branch create <name> [base]` accepts the base as an optional positional argument
- `qode branch remove` respects a `keep_branch_context` config field and a `--keep-branch-context` flag
- No dead code (functions, variables, imports) remains after the change
- All tests pass

---

## 2. Scope

### In scope
- Remove `--verbose` / `-v` global flag; decide per call-site whether to show or swallow the error
- Replace `--base` flag on `branch create` with an optional second positional argument
- Remove `--keep-branch` from `branch remove`; add `keep_branch_context` field to `qode.yaml` and `--keep-branch-context` flag
- Remove `--skip-tests` and `--layer` from `check`; remove `filterLayers` helper and `CheckOptions` struct
- Remove `--new`, `--scaffold`, `--workspace`, `--ide` from `init`; delete the wizard code paths and all dead functions
- Remove `--cursor` and `--claude` from `ide setup`; remove the per-flag override blocks; remove the existing TODO comment
- Remove `--iterations` from `plan refine`; always pass `0` (auto-detect) internally
- Add `BranchConfig` struct to `internal/config/schema.go` with `KeepBranchContext bool`
- Add default and config_test.go coverage for the new field

### Out of scope
- Adding `.qode/branches/` to `.gitignore` (separate issue)
- Reworking `qode init` beyond flag removal (tracked in #29)
- Changing or removing `--root` global flag
- Changing or removing any `--to-file` flag
- Adding a `--no-keep-branch-context` flag
- Changing the internals of `ide.Setup` or what configs it generates
- Any changes to `internal/git/git.go`

### Assumptions
- The `--base`, `--keep-branch`, `--skip-tests`, `--layer` flags are not used in any known CI pipeline (confirmed by ticket: "never used in practice")
- `qode init` still exists at the time this PR lands; if #29 lands first and removes `init`, this PR's `init` changes can be dropped

---

## 3. Architecture & Design

### Component map (affected only)

```
cmd/qode/main.go
└── internal/cli/
    ├── root.go              ← remove flagVerbose var + --verbose flag
    ├── branch.go            ← create: --base→positional; remove: --keep-branch→config+flag
    ├── check.go             ← remove --skip-tests, --layer, filterLayers
    ├── init.go              ← remove 4 flags + dead wizard functions
    ├── ide.go               ← remove --cursor, --claude, flag-override blocks, TODO
    ├── plan.go              ← remove --iterations
    ├── start.go             ← swallow knowledge.List error silently
    └── knowledge_cmd.go     ← swallow knowledge.ListLessons error silently

internal/config/
    ├── schema.go            ← add BranchConfig + Branch field on Config
    ├── defaults.go          ← add Branch default
    └── config_test.go       ← add BranchConfig tests

internal/runner/
    └── runner.go            ← remove CheckOptions struct, simplify RunCheck signature
```

### Layers affected

Only the **default (go)** layer at `.` is affected. No other repos, services, or external systems are touched.

### New vs modified

| Component | Change type |
|---|---|
| `BranchConfig` struct | New |
| `Config.Branch` field | New |
| `flagVerbose` package var | Deleted |
| `CheckOptions` struct | Deleted |
| `filterLayers` function | Deleted |
| `runInitNew`, `runInitWorkspace`, `scaffoldProject`, `readLine`, `pickChoice`, `applyIDEFlags` | Deleted |
| All other touched items | Modified (flag removal only) |

### Data flow — `branch remove` (changed path)

```
qode branch remove <name> [--keep-branch-context]
  │
  ├── resolveRoot()
  ├── config.Load(root)              ← NEW: load config for keep_branch_context default
  │     └── cfg.Branch.KeepBranchContext (bool, default false)
  │
  ├── keepCtx = cfg.Branch.KeepBranchContext || flagKeepBranchCtx
  │
  ├── if !keepCtx → os.RemoveAll(.qode/branches/<name>/)
  │
  └── git.DeleteBranch(root, name)  ← always attempted (unchanged)
```

---

## 4. API / Interface Contracts

### CLI interface changes

#### `qode branch create` — signature change
```
# Before
qode branch create <name> [--base <base>]

# After
qode branch create <name> [base]
```
- `name` — required positional arg (unchanged)
- `base` — optional second positional arg; defaults to current HEAD when absent
- Error if git operation fails: printed to stderr, non-zero exit

#### `qode branch remove` — flag replacement
```
# Before
qode branch remove <name> [--keep-branch]

# After
qode branch remove <name> [--keep-branch-context]
```
- `--keep-branch-context` — boolean flag; when present, `.qode/branches/<name>/` is NOT deleted
- Config default: `keep_branch_context: false` in `qode.yaml`
- Effective behaviour: `keepCtx = cfg.Branch.KeepBranchContext || flagPresent`
- Git branch is **always** deleted (regardless of flag)

#### Removed flag signatures (all produce cobra "unknown flag" error after this change)
| Command | Removed flag |
|---|---|
| `qode` | `--verbose`, `-v` |
| `qode branch create` | `--base` |
| `qode branch remove` | `--keep-branch` |
| `qode check` | `--skip-tests` |
| `qode check` | `--layer` |
| `qode init` | `--new`, `--scaffold`, `--workspace`, `--ide` |
| `qode ide setup` | `--cursor`, `--claude` |
| `qode plan refine` | `--iterations` |

### Internal Go API changes

#### `runner.RunCheck` — signature simplification
```go
// Before
func RunCheck(root, branch string, cfg *config.Config, layers []config.LayerConfig, opts CheckOptions) []LayerResult

// After
func RunCheck(root, branch string, cfg *config.Config, layers []config.LayerConfig) []LayerResult
```

#### `runInitExisting` — signature simplification (unexported)
```go
// Before
func runInitExisting(root string, ides []string) error

// After
func runInitExisting(root string) error
```

#### `runPlanRefine` — signature simplification (unexported)
```go
// Before
func runPlanRefine(ticketURL string, iterations int, toFile bool) error

// After
func runPlanRefine(ticketURL string, toFile bool) error
```

---

## 5. Data Model Changes

### New config field: `BranchConfig`

**`internal/config/schema.go`** — add struct and field:
```go
// BranchConfig controls branch lifecycle behaviour.
type BranchConfig struct {
    KeepBranchContext bool `yaml:"keep_branch_context,omitempty"`
}
```

Add to `Config`:
```go
Branch BranchConfig `yaml:"branch,omitempty"`
```

**`internal/config/defaults.go`** — add to `DefaultConfig()`:
```go
Branch: BranchConfig{KeepBranchContext: false},
```

### Migration strategy
- Existing `qode.yaml` files that do not have a `branch:` section continue to work — `omitempty` on the field means it is absent from the YAML until explicitly set.
- Default is `false` (delete context), which matches the current implicit behaviour.
- No migration script needed.

### Backward compatibility
- YAML files: fully backward compatible (new field is optional, defaulted)
- CLI: **breaking** for users of the removed flags (by design)
- Go API: `runner.CheckOptions` removal is a breaking change to the `runner` package API; only one call site (`check.go`) so the impact is contained within this repo

---

## 6. Implementation Tasks

In recommended order (schema-first, then flag that uses it, then remaining in any order):

- [ ] **Task 1 — (config) Add `BranchConfig` to schema and defaults**
  - `internal/config/schema.go`: add `BranchConfig` struct; add `Branch BranchConfig` to `Config`
  - `internal/config/defaults.go`: add `Branch: BranchConfig{KeepBranchContext: false}`
  - `internal/config/config_test.go`: add test for default value and YAML round-trip

- [ ] **Task 2 — (cli) Replace `--keep-branch` with config + `--keep-branch-context` in `branch remove`**
  - Depends on Task 1
  - `internal/cli/branch.go`: load config in `newBranchRemoveCmd`; replace flag; add `keepCtx` logic; git deletion unconditional

- [ ] **Task 3 — (cli) Remove `--verbose` / `-v` global flag**
  - `internal/cli/root.go`: delete `flagVerbose` var and flag registration
  - `internal/cli/start.go:49`: remove `flagVerbose` guard; restructure to `paths, _ := knowledge.List(root, cfg)`
  - `internal/cli/knowledge_cmd.go:222`: remove `flagVerbose` guard; discard `ListLessons` error
  - Clean unused imports

- [ ] **Task 4 — (cli) Replace `--base` with second positional arg in `branch create`**
  - `internal/cli/branch.go`: `RangeArgs(1,2)`; inline `base` from `args[1]`; remove flag

- [ ] **Task 5 — (cli/runner) Remove `--skip-tests` and `--layer` from `check`; simplify runner**
  - `internal/cli/check.go`: remove flag vars, registrations, `filterLayers` call and function; update `RunCheck` call
  - `internal/runner/runner.go`: remove `CheckOptions` struct; update `RunCheck` signature; simplify `if` condition in `runLayerCheck`

- [ ] **Task 6 — (cli) Strip flags from `qode init`; delete dead functions**
  - `internal/cli/init.go`: remove flag vars and registrations; simplify `RunE` to call `runInitExisting(root)` directly; remove 6 dead functions; update `runInitExisting` signature; rewrite `Long` help text; clean imports

- [ ] **Task 7 — (cli) Remove `--cursor` and `--claude` from `ide setup`**
  - `internal/cli/ide.go`: remove flag vars, registrations, `Changed` override blocks; remove TODO comment

- [ ] **Task 8 — (cli) Remove `--iterations` from `plan refine`**
  - `internal/cli/plan.go`: remove flag var and registration; update call and function signature

---

## 7. Testing Strategy

### Unit tests

**`internal/config/config_test.go`** (Task 1):
- `TestDefaultConfig_BranchKeepBranchContext`: assert `DefaultConfig().Branch.KeepBranchContext == false`
- `TestConfig_BranchYAMLRoundTrip`: marshal a `Config` with `Branch.KeepBranchContext = true`, unmarshal, assert value preserved
- `TestConfig_BranchOmitempty`: marshal a config with default (false) `Branch`; assert no `branch:` key appears in the YAML output

**`internal/runner/` (Task 5)**:
- If any existing tests reference `CheckOptions`, update them to remove the struct usage
- Verify `RunCheck` tests still pass with the simplified signature

### Integration / CLI tests

If `internal/cli/` has any test files, update any that reference:
- `flagVerbose`, `--verbose`, `-v`
- `--base` flag
- `--keep-branch` flag
- `--skip-tests`, `--layer`
- `--new`, `--scaffold`, `--workspace`, `--ide`
- `--cursor`, `--claude`
- `--iterations`
- `runner.CheckOptions`

### Manual verification checklist

- [ ] `qode branch create my-feature` — succeeds, creates branch from HEAD
- [ ] `qode branch create my-feature main` — succeeds, creates branch from `main`
- [ ] `qode branch create my-feature --base main` — cobra error: "unknown flag: --base"
- [ ] `qode branch remove my-feature` — deletes context folder AND git branch
- [ ] `qode branch remove my-feature --keep-branch-context` — deletes git branch, keeps `.qode/branches/my-feature/`
- [ ] `qode.yaml` with `branch: { keep_branch_context: true }` + `qode branch remove my-feature` — keeps context
- [ ] `qode check` (no flags) — runs all gates for all layers
- [ ] `qode check --skip-tests` — cobra error: "unknown flag: --skip-tests"
- [ ] `qode check --layer default` — cobra error: "unknown flag: --layer"
- [ ] `qode init` — runs auto-detect path; no wizard prompt appears
- [ ] `qode ide setup` — generates all enabled IDE configs; no filter flags accepted
- [ ] `qode plan refine` — generates refinement prompt; `--iterations` not accepted
- [ ] `qode --verbose check` — cobra error: "unknown flag: --verbose"

### Edge cases to test explicitly

- `qode branch create` with three positional args — cobra `RangeArgs(1,2)` error
- `qode branch create` not in a git repo — git error propagated, non-zero exit
- `qode branch remove` when `.qode/branches/<name>/` does not exist — no error (graceful via `os.IsNotExist`)
- `qode.yaml` missing `branch:` section — `Branch.KeepBranchContext` defaults to `false`
- `keep_branch_context: true` in config + `--keep-branch-context` flag also present — context kept (idempotent)

---

## 8. Security Considerations

- **Authentication / authorisation:** No changes. None of the removed flags touch auth or permissions.
- **Input validation:** The new `base` positional argument in `branch create` is passed directly to `git checkout -b <name> <base>`. Git validates the ref itself; the existing `safeBranchDir` path-traversal check on `name` is unaffected. No new validation surface is introduced.
- **Data sensitivity:** No sensitive data involved. Config field `keep_branch_context` is a boolean preference with no security implications.
- **Dead code removal:** Removing `runInitWorkspace`, `runInitNew` and related interactive wizard code reduces the overall attack surface of the binary by eliminating stdin-reading code paths.

---

## 9. Open Questions

None. All ambiguities from the requirements analysis were resolved:

1. **Is `qode init` still present?** — Yes, confirmed by reading `internal/cli/init.go`. Flags are removed as part of this PR.
2. **`--keep-branch-context` bidirectional semantics** — Resolved: the flag is unidirectional (opt-in to keeping context). Force-deleting when `keep_branch_context: true` is done manually. No `--no-keep-branch-context` flag added (out of scope per CLAUDE.md).
3. **`CheckOptions` struct with no fields** — Remove the struct entirely; update the one call site in `check.go`.

---

*Spec generated by qode. Copy to nqode-io/qode#44 for team review.*
