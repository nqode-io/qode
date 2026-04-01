# Code Review — reimplement-init-command

**Reviewer:** Claude (worker pass)
**Branch:** reimplement-init-command
**Spec:** `.qode/branches/reimplement-init-command/spec.md`

---

## Pre-read incident report

The call comes in on a Monday: a senior engineer on a client project ran `qode init` to regenerate their IDE commands after editing `qode.yaml` (the README says to do exactly this). Their `.qode/prompts/review/code.md.tmpl` and `/security.md.tmpl` — two months of prompt iteration — are gone. So is their `qode.yaml`: custom rubric dimensions, `min_security_score: 12`, `target_score: 30`, all overwritten with the minimal default. No backup, no warning, no `--dry-run`. The output said "Generated: qode.yaml". It did.

---

## File-by-file interrogation

### `internal/cli/init.go`

**Assumptions made:** `root` is valid and writable; `DefaultConfig()` always represents the desired end-state; all calls to `ide.Setup` and `copyEmbeddedTemplates` succeed atomically enough for the user.

**What callers receive:** `runInitExisting` returns an error on failure and prints "Generated: qode.yaml" + next steps on success. There is no output indicating whether a prior `qode.yaml` or any `.qode/prompts/` files were overwritten.

**Earliest silent failure point:** Line 41 — `cfg := config.DefaultConfig()` followed immediately by `yaml.Marshal(&cfg)` and `os.WriteFile`. If `qode.yaml` already exists with custom content, it is silently replaced before any output is emitted.

**Items:**
1. `copyEmbeddedTemplates` comment says "Existing files are overwritten so projects stay in sync with the embedded defaults" — but no output line announces this. The user flow `qode init` → regenerate IDE configs destroys prompt customizations silently. `copyEmbeddedTemplates` gives no hint it touched anything. Verified: there is no `fmt.Printf` call in `copyEmbeddedTemplates` that names which templates were overwritten.
2. `ide.Setup(root, &cfg)` is called with a freshly-constructed `DefaultConfig()` pointer — not a config loaded from disk. If a user has `ide.cursor.enabled: false` in their existing `qode.yaml`, that preference is ignored because the config is rebuilt from defaults, not read. Verified: no `config.Load` call exists anywhere in `runInitExisting`.
3. `os.WriteFile(outPath, data, 0644)` — permissions are hardcoded to 0644. This is safe and matches other write sites in the codebase. Verified safe.

---

### `internal/ide/cursor.go` and `internal/ide/claudecode.go`

**Items:**
1. `slashCommands(name string, cfg *config.Config)` — `cfg` is accepted but never read. Same for `claudeSlashCommands(name string, cfg *config.Config)`. The `Enabled` check lives in `ide.Setup`, which is correct. But the dead parameter tells callers that the config influences command content; it does not. If a future maintainer passes a zero-value config (e.g. in a test) expecting it to suppress output, they'll be confused. Verified by reading both functions end-to-end: zero references to any `cfg.*` field.
2. `SetupCursor` and `SetupClaudeCode` both compute `name := filepath.Base(root)` independently. This is correct and safe — `filepath.Base` on an absolute path returns only the last component, no directory traversal possible. The name flows into markdown description strings, not shell commands. Verified safe.
3. `writeFile` in `cursor.go:153` calls `os.MkdirAll(filepath.Dir(path))` before writing — this means the commands directory is created even if the file write subsequently fails. The partial state (empty dir, no `.mdc` file) would be left behind. Low impact but worth noting: no rollback on partial failure.

---

### `internal/cli/init_test.go`

**Items:**
1. `TestRunInitExisting_NoDetectionOutput` calls `captureStdout` — a helper defined elsewhere in the `cli` test package. Verified this is a shared helper (grep confirms it's used in other test files). The test correctly captures and checks stdout for forbidden strings. IDE output from `SetupCursor`/`SetupClaudeCode` goes through `fmt.Printf` (stdout), so if any of those messages ever contained "Detected" or "Scanning", this test would catch it. Verified property is sound.
2. `TestRootCmd_NoIDESubcommand` uses `rootCmd.Find([]string{"ide"})` — Cobra returns `(rootCmd, nil)` for an unknown command (the unknown args are left unresolved). The condition `findErr == nil && ideCmd != rootCmd` is false when `ide` is not registered because `ideCmd == rootCmd`. This is correct Cobra behavior and the test logic is sound, though the condition reads confusingly (the positive "ide exists" case is the truthy branch that should fail the test). Nit only.
3. No test for the re-run scenario: `runInitExisting` called twice on the same directory. The spec lists this as an explicit E2E test case ("Re-running `qode init` in an already-initialised directory: exits 0, files overwritten cleanly, no duplicate dirs") but no unit test covers it. The "overwritten cleanly" assertion specifically should be tested — does the second run produce the same file count and content?

---

### `internal/config/defaults.go` and `internal/config/schema.go`

**Items:**
1. `DefaultConfig()` returns `QodeVersion: "0.1"`, but the project's own `qode.yaml` has `qode_version: "0.1.3-alpha"`. The field is described as "the qode configuration format version", so these should be different values — but this inconsistency means running `qode init` in the `qode` repo itself would downgrade the value from `"0.1.3-alpha"` to `"0.1"`. If `qode_version` is truly a format version (not a binary version), `"0.1.3-alpha"` in `qode.yaml` is wrong and should be `"0.1"`. One of these is incorrect. Verified: no version-enforcement logic exists today, so this is currently harmless but points to semantic drift.
2. `QodeVersion` uses `yaml:"qode_version,omitempty"` — the `omitempty` tag means if `QodeVersion` is an empty string, the field is omitted from the marshaled output. `DefaultConfig()` always sets it to `"0.1"`, so `omitempty` is benign in the init path. But a manually constructed `Config{}` with no version set would produce YAML without `qode_version`. Verified: the test `TestRunInitExisting_WritesQodeVersion` catches the init path correctly.
3. `BranchConfig{KeepBranchContext: false}` is explicitly set in `DefaultConfig()`, so it serializes as `branch: {keep_branch_context: false}`. Since `omitempty` is on the struct tag, the zero value wouldn't appear — but `false` is the zero value for `bool`, so this field is suppressed by `omitempty`. Verified: `yaml:"keep_branch_context,omitempty"` — false booleans are omitted, so this field never appears in the generated YAML even though it's set in the struct. No behavior change from this.

---

### Template changes (`internal/prompt/templates/`)

**Items:**
1. `start/base.md.tmpl` replacement of `{{range .Layers}}` with hardcoded "default (go):" clean code rules — this is a Go-only project (qode itself), so the hardcoded rules are accurate. Template execution no longer panics on a missing `Layers` field. Verified: `TemplateData` no longer has a `Layers` field; `text/template` would produce a runtime error on `{{range .Layers}}` with the new struct. The template update was required before the struct change (confirmed by task ordering in spec). No panic path remains.
2. The five template files no longer reference `.Layers` anywhere. `TemplateProject{Name string}` exposes `.Project.Name` identically to the old `config.ProjectConfig` — zero template syntax changes required for the name substitution. Verified by checking the `{{.Project.Name}}` pattern still works.

---

### `internal/plan/refine.go`, `internal/review/review.go`, `internal/cli/knowledge_cmd.go`

**Items:**
1. All callers correctly transitioned from `cfg.Project` + `cfg.Layers()` to `engine.ProjectName()`. Verified: no remaining references to `cfg.Project` or `.Layers` in these files.
2. `buildBranchLessonData` signature changed from `(root string, cfg *config.Config, branches []string)` to `(root string, engine *prompt.Engine, branches []string)` — the `config.Load` call in `runKnowledgeAddBranch` was removed. However, other branches of `runKnowledgeAddBranch` already loaded config independently (for `branch.KeepBranchContext`). The diff shows `cfg, err := config.Load(root)` was removed entirely. Verified: `cfg.Branch.KeepBranchContext` is not read in `runKnowledgeAddBranch` — the keep-context check happens in `runBranchRemove`, not here. Safe removal.
3. `engine.ProjectName()` returns `filepath.Base(e.root)` — if `root` is `/`, this returns `.`. This edge case exists across all callers. In practice `root` is always a non-root path (validated by `resolveRoot()`). Low risk but the behavior on `/` is "." appearing in prompts as the project name.

---

## Issue Summary

| # | Severity | File | Issue |
|---|---|---|---|
| 1 | **High** | `internal/cli/init.go:37-78` | `qode init` silently overwrites existing `qode.yaml` and all `.qode/prompts/` customizations with no warning; documented "re-run to regenerate IDE configs" flow destroys user data |
| 2 | **High** | `internal/cli/init.go:67` | `ide.Setup` called with freshly-constructed `DefaultConfig()`, ignoring user's existing `ide.cursor.enabled: false` or similar preference in their current `qode.yaml` |
| 3 | **Medium** | `internal/ide/cursor.go:32`, `internal/ide/claudecode.go:82` | Dead `cfg *config.Config` parameter in `slashCommands` and `claudeSlashCommands` — never read; misleads callers into thinking config affects command content |
| 4 | **Medium** | `internal/cli/init_test.go` | No test for re-run idempotency; spec lists this as explicit E2E case; the overwrite-cleanly assertion is untested |
| 5 | **Low** | `internal/config/defaults.go:6` | `QodeVersion: "0.1"` in `DefaultConfig()` vs `qode_version: "0.1.3-alpha"` in `qode.yaml` — semantic ambiguity about whether `qode_version` is a binary version or a config format version |
| 6 | **Low** | `internal/ide/cursor.go:153-158` | `writeFile` creates the parent directory before writing; on failure the directory is left behind without cleanup |
| 7 | **Nit** | `internal/cli/init_test.go:139` | `TestRootCmd_NoIDESubcommand` condition `findErr == nil && ideCmd != rootCmd` reads as the truthy-is-failure branch — confusing but correct |
| 8 | **Nit** | `internal/ide/claudecode.go:184` | Double blank line in `qode-knowledge-add-branch` template content after the heading |

**Total:** 2 High, 2 Medium, 2 Low, 2 Nit

---

## Top 3 to fix before merging

1. **Issue #1 — Silent overwrite of `qode.yaml` and `.qode/prompts/`**: Add a check for existing `qode.yaml` and print a warning ("Warning: overwriting existing qode.yaml — custom rubrics and scoring thresholds will be reset") before the write. For templates, log each overwritten file. Alternatively, add a `--reset` flag and skip the overwrite by default when the target exists.

2. **Issue #2 — Init ignores existing IDE preferences**: `runInitExisting` should attempt `config.Load(root)` first and fall back to `DefaultConfig()` only when no `qode.yaml` exists. The `ide.Setup` call then respects the loaded config. For a first-run scenario (no existing `qode.yaml`), behavior is unchanged.

3. **Issue #3 — Dead `cfg` parameter**: Remove `cfg *config.Config` from both `slashCommands` and `claudeSlashCommands`. Neither function reads it. The IDE `Enabled` check belongs in `ide.Setup` (which already has it), not in the command-generation functions.

---

## Rating

| Dimension | Score | What you verified |
|---|---|---|
| Correctness (0–3) | 2 | All 14 spec tasks implemented; behavioral gap: init ignores existing `ide.*` preferences in disk config; overwrite is silent |
| CLI Contract (0–2) | 2 | Prompts to stdout; errors via return value → cobra stderr; exit codes correct; `qode.yaml` keys are additive and backward-compatible (yaml.v3 ignores unknown fields confirmed) |
| Go Idioms & Code Quality (0–2) | 1 | Dead `cfg` parameter in two functions is deceptive API; all functions ≤50 lines; `fmt.Errorf("%w")` used throughout; no global mutable state introduced |
| Error Handling & UX (0–2) | 1 | Error messages are actionable (permission failures show paths); silent overwrite of user data is a UX defect on the re-run path |
| Test Coverage (0–2) | 2 | 7 integration tests in `init_test.go` using real temp dirs; 3 new project-name propagation tests across ide/plan/review; re-run idempotency not tested (low severity given spec coverage) |
| Template Safety (0–1) | 1 | `filepath.Base(root)` is not user input; flows into `text/template` markdown output only; no shell execution path; `TemplateProject.Name` substitution verified against the five modified templates |

**Total Score: 9.0/12**
**Minimum passing score: 10/12**

> A High finding caps the total at 9.0. The two High issues (silent overwrite of user data; ignoring disk config for IDE enabled state) are real data-loss and behavior-violation paths on the documented re-run workflow.
