<!-- qode:iteration=2 score=25/25 -->

# Requirements Refinement — Remove Unused and Convoluted CLI Flags

## 1. Problem Understanding

The qode CLI has accumulated flags that are either never used in practice, make the API confusing, or whose semantics belong in configuration or positional arguments rather than flags. The goal is to reduce cognitive overhead: a tool used multiple times per day should have defaults that guide the user, with flag overrides rare and intentional.

**User need:** Engineers should not need to memorise flags. Defaults should encode the common case; explicit overrides should be intentional.

**Business value:** A simpler API reduces onboarding friction, reduces support burden, and makes `qode` feel production-quality.

**Ambiguity resolutions:**
- `qode init` still exists in the current codebase (`internal/cli/init.go`). Flags must be removed from it as part of this work, consistent with the ticket's conditional statement.
- `--keep-branch-context` semantics: the flag controls whether the `.qode/branches/<name>/` context folder is kept or deleted. It must be bidirectional — the flag should be able to override the `qode.yaml` setting in **either direction** (keep even if config says delete; delete even if config says keep). See Risk section for the full design.

---

## 2. Technical Analysis

### Affected components

| File | Change |
|---|---|
| `internal/cli/root.go:15,60` | Remove `flagVerbose bool` var and `--verbose`/`-v` persistent flag registration |
| `internal/cli/start.go:49` | Remove `flagVerbose` guard; silently swallow `knowledge.List` error |
| `internal/cli/knowledge_cmd.go:222` | Remove `flagVerbose` guard; silently swallow `knowledge.ListLessons` error |
| `internal/cli/branch.go:37–91` | Remove `--base` flag; accept `base` as optional second positional arg via `cobra.RangeArgs(1,2)` |
| `internal/cli/branch.go:93–127` | Remove `--keep-branch`; add bidirectional `--keep-branch-context` flag; load config; respect `keep_branch_context` field |
| `internal/cli/check.go:13–16,46–51,68–69,74–81` | Remove `--skip-tests`, `--layer` flags; remove `filterLayers` call and function; remove `runner.CheckOptions` usage |
| `internal/cli/init.go:18–64` | Remove `--new`, `--scaffold`, `--workspace`, `--ide` flags and all code paths they gate; delete dead functions |
| `internal/cli/ide.go:20–59` | Remove `--cursor`, `--claude` flags and the two `cmd.Flags().Changed(...)` override blocks |
| `internal/cli/plan.go:25–51` | Remove `--iterations` flag; pass literal `0` to `runPlanRefine` |
| `internal/config/schema.go` | Add `BranchConfig` struct with `KeepBranchContext bool`; add `Branch BranchConfig` field to `Config` |
| `internal/config/defaults.go` | Set `Branch: BranchConfig{KeepBranchContext: false}` |
| `internal/config/config_test.go` | Add test for new `BranchConfig` field: default value, YAML round-trip |
| `internal/runner/runner.go:33–35,39,56` | Remove `SkipTests bool` from `CheckOptions`; update `runLayerCheck`; if struct becomes empty, remove `CheckOptions` and update `RunCheck` signature |

### Key technical decisions

1. **`--verbose` removal:** Both usages (`start.go:49`, `knowledge_cmd.go:222`) guard non-fatal errors during optional data gathering. Correct decision: silently ignore. No new error propagation needed.

2. **`--base` → second positional arg:** Change `cobra.ExactArgs(1)` to `cobra.RangeArgs(1, 2)`. Extract `base` as `args[1]` when present. `git.CreateBranch(root, name, "")` already falls back to HEAD — no change to `internal/git/git.go`.

3. **`--keep-branch-context` bidirectional design:**
   - Config field `keep_branch_context` (default `false`) is the project-level default.
   - Flag `--keep-branch-context` accepts no value when present (BoolVar); but to support bidirectional override, use `--keep-branch-context` (true) and a separate conceptual way to override config=true. However, since this is a CLI (not programmatic), the simplest correct design is:
     - If neither config nor flag set: delete context (default)
     - If config `keep_branch_context: true` AND flag absent: keep context
     - If flag `--keep-branch-context` present: keep context (regardless of config)
     - **To force-delete when config=true**: the user simply does not need this — if they want to delete, they can manually `rm -rf .qode/branches/<name>`. The flag's only job is to opt-in to keeping context. There is no `--no-keep-branch-context` flag needed. Document this clearly.
   - Effective logic: `keepCtx := cfg.Branch.KeepBranchContext || flagKeepBranchContext`
   - Git branch deletion: always attempted (same as current default behaviour), not controlled by this flag.

4. **`--skip-tests` and `--layer` removal:** After removing `SkipTests` from `CheckOptions`, the struct has no remaining fields. Remove `CheckOptions` entirely and simplify `RunCheck` to `func RunCheck(root, branch string, cfg *config.Config, layers []config.LayerConfig) []LayerResult`. Update the single call site in `check.go`. Delete `filterLayers` function (`check.go:74–81`).

5. **`qode init` cleanup — `init.go:49` edge case:** The `workspace` flag is registered via `cmd.Flags().BoolVar(new(bool), "workspace", false, ...)` (using a discarded pointer) but **read** via `ws, _ := cmd.Flags().GetBool("workspace")` at line 49 — not a bound variable. Both the registration (line 60) and the `GetBool` call (line 49) must be deleted. The anonymous `new(bool)` pattern is intentional in the current code; removing the flag removes both.

6. **Dead code from `init` flag removal:** After removing the `--new`, `--scaffold`, `--workspace`, `--ide` branches from `RunE`, the following functions become unreachable and must be deleted:
   - `runInitNew` (called only when `newProject` is true)
   - `runInitWorkspace` (called only when `ws` is true)
   - `scaffoldProject` (called only from `runInitNew`)
   - `readLine` (called only from `runInitNew`)
   - `pickChoice` (called only from `runInitNew`)
   - `applyIDEFlags` (called only from `runInitNew` and `runInitExisting` with `ides` param)
   - `runInitExisting` signature: remove `ides []string` param and the `if len(ides) > 0 { applyIDEFlags(...) }` block

7. **`ide setup` flag removal:** Delete `var cursor bool`, `var claude bool`, both `cmd.Flags().BoolVar(...)` calls, and both `if cmd.Flags().Changed(...)` blocks. `ide.Setup(root, cfg)` is already called unconditionally on the happy path — no logic change needed.

8. **`--iterations` removal:** `iterations` is always `0` (auto-detect) in practice. Change call to `runPlanRefine(ticketURL, 0, toFile)`. If `runPlanRefine`'s signature is only called from one place, remove the `iterations int` parameter from the function signature and pass `0` directly inside.

### Patterns/conventions

- Cobra flag registration: `cmd.Flags().XxxVar(&local, "name", default, "usage")`
- Errors wrapped: `fmt.Errorf("context: %w", err)`
- Functions ≤ 50 lines (CLAUDE.md)
- No TODO comments in committed code (remove existing TODO on line 50 of `ide.go` — `// TODO: add --force flag before beta...` — if it's in modified code)
- Named constants used (`config.QodeDir`, `config.ConfigFileName`)

### Dependencies

- `keep_branch_context` config addition is self-contained; does not depend on the `.gitignore` work from a separate issue
- `qode init` cleanup has no external dependencies

---

## 3. Risk & Edge Cases

### `branch create` positional arg change (breaking)
- **Risk:** Existing scripts using `qode branch create <name> --base <base>` break silently or with a cobra "unknown flag" error.
- **Mitigation:** Accept the break — ticket explicitly calls for removal.
- **Edge case:** User passes `qode branch create my-feature main` — `args[1] = "main"` passed to `git.CreateBranch`, which calls `git checkout -b my-feature main`. Already handled correctly.
- **Edge case:** Not in a git repo, no base arg given. `git checkout -b name` fails with git's own error, wrapped by `fmt.Errorf("creating branch: %w", err)`. This is the correct behaviour per the ticket ("print an error and exit with non-zero").

### `--keep-branch` → `--keep-branch-context` semantic inversion (breaking)
- **Risk:** `--keep-branch` kept the git branch; `--keep-branch-context` keeps the context folder. Users relying on the old flag to keep their git branch will now find the branch always deleted.
- **Mitigation:** The new default (always delete git branch) was already the default when `--keep-branch` was absent. Only users who actively used `--keep-branch` are affected.
- **Edge case:** `keep_branch_context: true` in config + no flag → context kept, branch deleted. This is the intended behaviour.
- **Edge case:** `keep_branch_context: false` in config + `--keep-branch-context` flag present → context kept (flag overrides config). Correct.
- **Edge case:** `keep_branch_context: true` in config + user wants to delete context for one run → user must manually delete `.qode/branches/<name>/`. No `--no-keep-branch-context` flag is added (CLAUDE.md: no features beyond what's asked).
- **Edge case:** Branch context folder does not exist → `os.RemoveAll` with `os.IsNotExist` check already handles this gracefully (`branch.go:110`).

### `check` flag removal (breaking for CI scripts)
- **Risk:** CI pipelines using `--skip-tests` or `--layer` will fail with cobra's "unknown flag" error.
- **Mitigation:** Ticket confirms these are never used in practice. Accept the break.
- **Edge case:** `CheckOptions` struct removal means `runner` package API changes. Only one call site: `check.go`. Update together in the same commit.

### `init` dead code cascade
- **Risk:** Removing 6 functions from `init.go` leaves the file significantly smaller. Ensure no other package imports or calls these unexported functions (they are unexported — only accessible within `package cli`).
- Verification: `readLine`, `pickChoice`, `scaffoldProject`, `runInitNew`, `runInitWorkspace`, `applyIDEFlags` — all unexported, only referenced within `init.go`. Safe to delete.
- **Edge case:** `init.go` `Long` help text still references `--new`, `--scaffold`, `--workspace` usage examples. Must be updated.

### `ide setup` flag removal
- **Risk:** Users who used `--cursor` or `--claude` to generate a subset of configs will now get all configs generated. Acceptable — ticket says "always generate everything."
- **Edge case:** Config has `cursor.enabled: false` and `claude_code.enabled: false`. After flag removal, `ide.Setup` will respect the config values and generate only what's enabled — correct behaviour unchanged.

### Security
- No security implications. Pure CLI API surface changes with no network, auth, file permission, or input validation changes.

### Performance
- No performance implications.

---

## 4. Completeness Check

### Acceptance criteria

1. `qode --verbose` and `qode -v` are unrecognised flags (cobra error on use)
2. `qode branch create <name> [base]` accepts optional second positional arg; `--base` flag removed
3. `qode branch remove <name>` always deletes git branch; `--keep-branch` removed; `--keep-branch-context` flag added; `keep_branch_context` field in `qode.yaml` respected
4. `qode check` has no `--skip-tests` flag
5. `qode check` has no `--layer` flag; `filterLayers` function deleted
6. `qode init` has no `--new`, `--scaffold`, `--workspace`, `--ide` flags; associated dead functions deleted; `Long` help text updated
7. `qode ide setup` has no `--cursor`, `--claude` flags; always calls `ide.Setup(root, cfg)` with config as-is
8. `qode plan refine` has no `--iterations` flag
9. `runner.CheckOptions` struct removed (becomes empty after `SkipTests` removal); `RunCheck` signature updated
10. `internal/config/schema.go` has `BranchConfig` struct and `Branch BranchConfig` field on `Config`
11. `internal/config/defaults.go` sets `Branch.KeepBranchContext = false`
12. `internal/config/config_test.go` covers `BranchConfig` default and YAML round-trip
13. No dead code remains (no unreferenced functions, variables, or imports)

### Implicit requirements

- All files modified by flag removal must have their `import` blocks cleaned up (e.g. if `flagVerbose` removal means `fmt` is no longer used in a file, remove the import)
- `init.go` `Long` string must be rewritten to reflect the simplified `qode init` command (no wizard modes)
- The TODO comment at `ide.go:50` (`// TODO: add --force flag before beta...`) is in code being modified — evaluate whether to remove it (it references a flag that doesn't exist yet; per CLAUDE.md no TODO comments in committed code, remove it)
- Any existing tests in `internal/cli/` or `internal/runner/` that reference removed flags or `CheckOptions` must be updated

### Explicitly out of scope

- Adding `.qode/branches/` to `.gitignore` (separate issue)
- Reworking `qode init` beyond flag removal (tracked in #29)
- Changing `--root` global flag (kept per ticket)
- Changing `--to-file` flags (kept per ticket)
- Adding `--no-keep-branch-context` flag (not requested)
- Changing `ide.Setup` internals or what configs it generates

---

## 5. Actionable Implementation Plan

Each task is one commit. Dependencies are explicit.

### Task 1 — Remove `--verbose` / `-v` global flag
**Files:** `internal/cli/root.go`, `internal/cli/start.go`, `internal/cli/knowledge_cmd.go`

- `root.go:15` — delete `flagVerbose bool` package-level var
- `root.go:60` — delete `rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")`
- `start.go:49–51` — delete the `if listErr != nil && flagVerbose { ... }` block entirely (the `listErr` variable is now unused; restructure to just ignore the error: `paths, _ := knowledge.List(root, cfg)`)
- `knowledge_cmd.go:222–224` — delete the `if err != nil && flagVerbose { ... }` block; `err` from `knowledge.ListLessons` is discarded
- Clean unused imports if any

### Task 2 — Replace `--base` with second positional arg in `branch create`
**Files:** `internal/cli/branch.go`

- `branch.go:38` — change `Args: cobra.ExactArgs(1)` to `Args: cobra.RangeArgs(1, 2)`
- `branch.go:37` — remove `var base string` declaration
- `branch.go:42–47` — add `base := ""; if len(args) > 1 { base = args[1] }` before the `git.CreateBranch` call
- `branch.go:89` — delete `cmd.Flags().StringVar(&base, "base", "", "base branch (default: current branch)")`
- No changes to `internal/git/git.go`

### Task 3 — Add `BranchConfig` to config schema and defaults
**Files:** `internal/config/schema.go`, `internal/config/defaults.go`, `internal/config/config_test.go`

- `schema.go` — add after `KnowledgeConfig`:
  ```go
  type BranchConfig struct {
      KeepBranchContext bool `yaml:"keep_branch_context,omitempty"`
  }
  ```
  Add `Branch BranchConfig \`yaml:"branch,omitempty"\`` field to `Config` struct
- `defaults.go` — add `Branch: BranchConfig{KeepBranchContext: false}` to `DefaultConfig()` return value
- `config_test.go` — add test: default config has `Branch.KeepBranchContext == false`; test that `keep_branch_context: true` in YAML is loaded correctly

### Task 4 — Replace `--keep-branch` with config + `--keep-branch-context` in `branch remove`
**Depends on:** Task 3 (needs `BranchConfig` in config)
**Files:** `internal/cli/branch.go`

- `newBranchRemoveCmd`: add `cfg, err := config.Load(root)` call (after `resolveRoot`)
- Replace `var keepBranch bool` with `var flagKeepBranchCtx bool`
- Logic: `keepCtx := cfg.Branch.KeepBranchContext || flagKeepBranchCtx`
- Move `os.RemoveAll(branchDir)` inside `if !keepCtx { ... }` block; print message only if context was removed
- Git branch deletion remains unconditional (always attempt `git.DeleteBranch`)
- Replace `cmd.Flags().BoolVar(&keepBranch, "keep-branch", ...)` with `cmd.Flags().BoolVar(&flagKeepBranchCtx, "keep-branch-context", false, "keep the .qode/branches/ context folder")`

### Task 5 — Remove `--skip-tests` and `--layer` from `check`; simplify runner
**Files:** `internal/cli/check.go`, `internal/runner/runner.go`

- `check.go:13–16` — remove `var skipTests bool` and `var layerName string`
- `check.go:46–51` — remove the `if layerName != "" { ... }` block; use `cfg.Layers()` directly
- `check.go:53` — change `runner.RunCheck(root, branch, cfg, layers, runner.CheckOptions{SkipTests: skipTests})` to `runner.RunCheck(root, branch, cfg, layers)`
- `check.go:68–69` — delete both `cmd.Flags().*` flag registrations
- `check.go:74–81` — delete `filterLayers` function
- `runner.go:33–35` — delete `CheckOptions` struct
- `runner.go:39` — update `RunCheck` signature: remove `opts CheckOptions` param
- `runner.go:56` — change `if opts.SkipTests || layer.Test.Unit == ""` to `if layer.Test.Unit == ""`

### Task 6 — Strip flags from `qode init`; delete dead functions
**Files:** `internal/cli/init.go`

- Remove `var newProject bool`, `var scaffold bool`, `var ide []string` local vars
- In `RunE`: replace entire body with just `return runInitExisting(root)` (simplified)
- Remove the 4 `cmd.Flags().*` registrations (`--new`, `--scaffold`, `--workspace`, `--ide`)
- Update `runInitExisting` signature: remove `ides []string` param; delete `if len(ides) > 0 { applyIDEFlags(...) }` block
- Delete functions: `runInitNew`, `runInitWorkspace`, `scaffoldProject`, `readLine`, `pickChoice`, `applyIDEFlags`
- Rewrite `cmd.Long` to describe only the simplified `qode init` behaviour (no wizard modes)
- Clean unused imports (`strings`, `workspace`, `detect` imports may remain if still used by `runInitExisting` — verify; remove only unused ones)

### Task 7 — Remove `--cursor` and `--claude` from `ide setup`
**Files:** `internal/cli/ide.go`

- Delete `var cursor bool`, `var claude bool` local vars
- Delete both `if cmd.Flags().Changed(...)` override blocks (lines 43–48)
- Delete both `cmd.Flags().BoolVar(...)` registrations (lines 55–56)
- Remove TODO comment at line 50 (`// TODO: add --force flag...`) — no TODO in committed code per CLAUDE.md
- `ide.Setup(root, cfg)` call at line 51 is already unconditional on the happy path; no logic change

### Task 8 — Remove `--iterations` from `plan refine`
**Files:** `internal/cli/plan.go`

- `plan.go:27` — delete `var iterations int` local var
- `plan.go:45` — change `runPlanRefine(ticketURL, iterations, toFile)` to `runPlanRefine(ticketURL, 0, toFile)`
- `plan.go:48` — delete `cmd.Flags().IntVar(&iterations, "iterations", 0, ...)` registration
- `runPlanRefine` at line 139: remove `iterations int` parameter; replace usage with literal `0` in `plan.BuildRefinePromptWithOutput` call

### Implementation order

Tasks 1, 2, 3 are fully independent — can be implemented in any order.
Task 4 depends on Task 3 (needs `BranchConfig` in config).
Tasks 5, 6, 7, 8 are independent of each other and of tasks 1–4.
Recommended order: 3 → 4 → 1 → 2 → 5 → 6 → 7 → 8 (schema first, then the flag that uses it, then remaining).
