<!-- qode:iteration=4 score=25/25 -->

# Requirements Refinement — reimplement-init-command

## 1. Problem Understanding

`qode init` currently auto-detects the tech stack and workspace topology, records them in `qode.yaml`, and requires the user to separately run `qode ide setup` to generate IDE configs. This is two commands where one suffices, the detection is unreliable, and the `.cursorrules/` files it generates duplicate what belongs in the user's `CLAUDE.md`.

**Goal**: a single `qode init` command that does everything to start using qode:
1. Write a minimal `qode.yaml` with defaults (`qode_version`; no project name, topology, layers, or test commands)
2. Create `.qode/branches/`, `.qode/knowledge/`, `.qode/prompts/` directories (idempotent)
3. Copy embedded prompt templates to `.qode/prompts/`
4. Generate IDE slash commands: `.cursor/commands/*.mdc` and `.claude/commands/*.md`

`qode ide` (setup + sync subcommands) is deleted entirely. There is no separate IDE subcommand. IDE configs are generated exclusively by `qode init`.

**User need**: one command, no detection noise, no `.cursorrules/` files competing with `CLAUDE.md`.

### All decisions (notes.md — authoritative, override ticket where they conflict)

| Decision | Detail |
|---|---|
| Remove `WorkspaceConfig` | `WorkspaceConfig`, `RepoRef`, `WorkspaceConfigFileName` deleted. |
| Keep `{{.Project.Name}}` in templates | Populated via `TemplateProject{Name string}` + `Engine.ProjectName()` = `filepath.Base(e.root)`. |
| Remove `{{range .Layers}}` from templates | Stack context comes from user's `CLAUDE.md`. |
| Merge init + IDE generation | `qode init` calls `ide.Setup()` as final step. |
| No `.cursorrules/` generation | Only `.cursor/commands/` is written by qode. `workflowRule()` and `cleanCodeRule()` deleted. |
| **Remove `qode ide` entirely** | Delete `internal/cli/ide.go`. Remove `newIDECmd()` from `root.go`. No `qode ide setup`, no `qode ide sync`. |
| Update all docs | README, `docs/qode-yaml-reference.md` updated to reflect single-command init. |

---

## 2. Technical Analysis

### Subsystems affected

| Subsystem | Location |
|---|---|
| Delete: detect package | `internal/detect/` — 10 files |
| Delete: workspace package | `internal/workspace/` — 2 files |
| **Delete: CLI ide command** | `internal/cli/ide.go` |
| Config schema | `internal/config/schema.go`, `config.go`, `defaults.go`, `config_test.go` |
| Prompt engine + TemplateData | `internal/prompt/engine.go` |
| Prompt templates (5) | `refine/base.md.tmpl`, `spec/base.md.tmpl`, `start/base.md.tmpl`, `review/code.md.tmpl`, `review/security.md.tmpl` |
| Plan builders | `internal/plan/refine.go` |
| Review builders | `internal/review/review.go` |
| Knowledge cmd | `internal/cli/knowledge_cmd.go` |
| IDE — cursor.go | `internal/ide/cursor.go` |
| IDE — claudecode.go | `internal/ide/claudecode.go` |
| IDE tests | `internal/ide/ide_test.go` |
| CLI root | `internal/cli/root.go` — remove `newIDECmd()` call (line 65) |
| CLI init | `internal/cli/init.go` |
| CLI plan tests (cleanup) | `internal/cli/plan_test.go` |
| Root config | `qode.yaml` |
| Docs | `docs/qode-yaml-reference.md`, `README.md` |

### Key technical decisions

**1. `TemplateProject` struct + `Engine.ProjectName()`**

Define `type TemplateProject struct { Name string }` in `internal/prompt/engine.go`. Change `TemplateData.Project` from `config.ProjectConfig` to `TemplateProject`. Add `func (e *Engine) ProjectName() string { return filepath.Base(e.root) }`. Every builder sets `data.Project = prompt.TemplateProject{Name: engine.ProjectName()}`. Templates keep `{{.Project.Name}}` verbatim — no template file changes for the name field.

**2. Removing `Layers []config.LayerConfig` from `TemplateData`**

Drop the field. Remove `{{range .Layers}}` blocks and `## Tech Stack:` headings from all five templates:
- `refine/base.md.tmpl:10–12`
- `spec/base.md.tmpl`: same pattern
- `start/base.md.tmpl:10–12` (first block) and `41–49` (second "Clean Code Requirements" block — replace with universal rules, no `{{range}}`)
- `review/code.md.tmpl:21–23`
- `review/security.md.tmpl:21–23`

**3. Config schema removals**

`internal/config/schema.go`: remove `ProjectConfig`, `LayerConfig`, `TestConfig`, `CoverageConfig`, `WorkspaceConfig`, `RepoRef`, `Topology` type + constants, `Config.Project`, `Config.Workspace`, `Config.Layers()`.
`internal/config/config.go:15`: remove `WorkspaceConfigFileName` constant.
`internal/config/defaults.go`: remove `StackDefaults` map (lines 123–176), `Project: ProjectConfig{Topology: TopologySingle}` from `DefaultConfig()`.
Add to `Config`: `QodeVersion string \`yaml:"qode_version,omitempty"\``.
Add to `DefaultConfig()`: `QodeVersion: "0.1"`.

**4. Removing `.cursorrules/` generation from `cursor.go`**

- Delete constant `cursorRulesDir = ".cursorrules"` (line 12)
- Delete `os.MkdirAll` for `cursorRulesDir` (lines 17–19) and the two `writeFile` calls for `.mdc` rule files (lines 25–31)
- Delete `workflowRule()` (lines 47–79) and `cleanCodeRule()` (lines 81–102)
- Delete dead helpers: `layerList`, `collectStacks`, `globsForStacks`, `stackCleanCodeRules`, `dedup`
- Change `slashCommands(cfg *config.Config)` → `slashCommands(name string, cfg *config.Config)`
- Update `SetupCursor`: compute `name := filepath.Base(root)`, pass to `slashCommands`; update `fmt.Printf` → `"  Cursor: .cursor/commands/ (%d commands)\n", len(cmds)`

**5. Project name in `claudecode.go`**

- Change `claudeSlashCommands(cfg *config.Config)` → `claudeSlashCommands(name string, cfg *config.Config)`
- Update `SetupClaudeCode`: compute `name := filepath.Base(root)`, pass to `claudeSlashCommands`

**6. Delete `internal/cli/ide.go`**

The file defines `newIDECmd()`, `newIDESetupCmd()`, and `newIDESyncCmd()` — all deleted. No replacement. The `ide` package (`internal/ide/`) is retained; `ide.Setup()` is called directly from `init.go`.

**7. Remove `newIDECmd()` from `internal/cli/root.go`**

Line 65: `newIDECmd(),` — removed. No other references to `newIDECmd` exist outside `ide.go`.

**8. `qode init` calls `ide.Setup()` as final step**

`internal/cli/init.go`: after `copyEmbeddedTemplates(root)`, add:
```go
if err := ide.Setup(root, cfg); err != nil {
    return fmt.Errorf("setting up IDE configs: %w", err)
}
```
Add import `"github.com/nqode/qode/internal/ide"`. Remove imports `detect` and `workspace`. `cfg.IDE.Cursor.Enabled` and `cfg.IDE.ClaudeCode.Enabled` both default to `true` in `DefaultConfig()`, so both sets of commands are always generated.

**9. `init.go` description and output updates**

- `Long`: replace current text ("Detect the project's tech stack…Run 'qode ide setup' afterwards") with "Writes a minimal qode.yaml, creates the .qode/ directory structure, copies prompt templates, and generates IDE slash commands."
- Next-steps: remove "Run 'qode ide setup' to generate IDE configs" (step 2). New steps: (1) Review qode.yaml to adjust scoring thresholds if needed; (2) Run 'qode branch create \<name\>' to start your first feature.

**10. Backward compatibility**

`gopkg.in/yaml.v3` ignores unknown fields — existing `qode.yaml` files with `project:` or `workspace:` sections load without error. No migration required for config files. Stale `.cursorrules/*.mdc` files left by previous qode runs are not touched (left in place).

---

## 3. Risk & Edge Cases

**R1: Template panic on removed `.Layers` field**
`text/template` panics on a missing field. Mitigation: update all five templates (Task 5) before removing `Layers` from `TemplateData` (Task 6). Enforced by compiler — Task 6 won't compile until templates no longer reference `.Layers`.

**R2: `.Project.Name` type change — no panic**
`TemplateProject{Name string}` exposes `.Name` exactly as `ProjectConfig` did. Templates access `{{.Project.Name}}` — unchanged behavior.

**R3: Existing `qode.yaml` with `project:` / `workspace:`**
yaml.v3 ignores unknown keys. Safe, no migration.

**R4: Local `.qode/prompts/` overrides referencing `{{range .Layers}}`**
User-customized local template overrides that use `{{range .Layers}}` will panic after `Layers` is removed. Mitigation: `copyEmbeddedTemplates` overwrites `.qode/prompts/`. Document as a breaking change in release notes.

**R5: `ide.go` deletion leaves a dangling reference in `root.go`**
`root.go:65` calls `newIDECmd()`. Removing `ide.go` without updating `root.go` causes a compile error. Task ordering: delete `ide.go` and update `root.go` in the same commit (Task 9).

**R6: `ide.Setup()` failure during `qode init` — partial state**
If `ide.Setup()` fails (e.g. filesystem permissions), `qode.yaml` and `.qode/` dirs are already written. The error is propagated and displayed. User can retry by re-running `qode init`. No silent partial success.

**R7: `qode init` overwrites existing `qode.yaml`**
Behavior unchanged: file is unconditionally overwritten. Custom rubrics or scoring thresholds are lost. Out of scope; document in next-steps output ("re-running qode init will reset qode.yaml to defaults").

**R8: Stale `.cursorrules/` files from prior runs**
Not touched by qode after the change. Left in place. Cursor still reads them, but the content remains valid (universal rules). Document in release notes.

**R9: `ide_test.go` calls `slashCommands(cfg)` and `claudeSlashCommands(cfg)` with old signatures**
After signature changes in Task 7, these calls won't compile. Fixed in Task 8 (update test call sites and `minimalConfig()`).

**R10: README example snippets show `qode ide setup`**
Three locations in README (lines 29, 144–145, 166/178/190) reference `qode ide setup` / `qode ide sync`. All must be updated in Task 14 to avoid user confusion.

**Security**: `filepath.Base(root)` used only in markdown descriptions, not shell commands. No injection risk.

---

## 4. Completeness Check

### Acceptance criteria (with named test locations)

1. `qode init` writes `qode.yaml` with `qode_version: "0.1"` and no `project:` key. — `internal/cli/init_test.go:TestRunInitExisting_WritesQodeVersion`
2. `qode init` creates `.qode/{branches,knowledge,prompts}/`. — `internal/cli/init_test.go:TestRunInitExisting_CreatesDirs`
3. `qode init` copies embedded templates to `.qode/prompts/`. — `internal/cli/init_test.go:TestRunInitExisting_CopiesTemplates`
4. `qode init` generates `.cursor/commands/*.mdc` and `.claude/commands/*.md`. — `internal/cli/init_test.go:TestRunInitExisting_CreatesIDEConfigs`
5. `qode init` does not create any file under `.cursorrules/`. — `internal/cli/init_test.go:TestRunInitExisting_NoCursorRules`
6. `qode init` stdout contains no "Detected", "topology", or "qode ide setup" text. — `internal/cli/init_test.go:TestRunInitExisting_NoDetectionOutput`
7. `qode ide` subcommand no longer exists (returns error or is absent from help). — `internal/cli/root.go` compile check; `internal/cli/init_test.go:TestRootCmd_NoIDESubcommand`
8. `SetupCursor` writes only `.cursor/commands/*.mdc`; no `.cursorrules/` dir created. — `internal/ide/ide_test.go:TestSetupCursor_NoCursorRulesDir`
9. `SetupCursor` derives project name from `filepath.Base(root)`. — `internal/ide/ide_test.go:TestSetupCursor_CommandsContainRootName`
10. `SetupClaudeCode` derives project name from `filepath.Base(root)`. — `internal/ide/ide_test.go:TestSetupClaudeCode_CommandsContainRootName`
11. `internal/detect/` entirely deleted. — Compiler: `go build ./...`
12. `internal/workspace/` entirely deleted. — Compiler
13. `internal/cli/ide.go` deleted; `newIDECmd` absent. — Compiler
14. `ProjectConfig`, `LayerConfig`, `TestConfig`, `CoverageConfig`, `WorkspaceConfig`, `RepoRef`, `Topology` absent. — Compiler
15. `Config.Layers()`, `WorkspaceConfigFileName`, `StackDefaults` absent. — Compiler
16. `DefaultConfig()` returns `QodeVersion == "0.1"`. — `internal/config/config_test.go:TestDefaultConfig` (updated)
17. `TemplateData` has `Project TemplateProject` and no `Layers` field. — Compiler
18. All five templates render without panic; `{{.Project.Name}}` renders to `filepath.Base(root)`. — `internal/plan/refine_test.go:TestBuildRefinePromptWithOutput_ContainsProjectName`; `internal/review/review_test.go:TestBuildCodePrompt_ContainsProjectName`, `TestBuildSecurityPrompt_ContainsProjectName`
19. `workflowRule`, `cleanCodeRule`, layer/stack helpers absent from `cursor.go`. — Compiler
20. `qode.yaml` (root) has `qode_version` and no `project:` section. — Manual + `go test ./...`
21. `docs/qode-yaml-reference.md`: no `project`/`topology`/`layers`/`test` sections; `qode_version` documented; no `qode ide` references. — Documentation review
22. `README.md`: single-command init; no `.cursorrules` in IDE table; no `qode ide setup`/`sync` entries. — Documentation review
23. `go build ./...` and `go test ./...` pass. — CI

### Implicit requirements

- `Engine.ProjectName() string` — new method on `Engine`
- `TemplateProject struct { Name string }` — new type in `internal/prompt/engine.go`, no `config` import
- `init.go` adds import `"github.com/nqode/qode/internal/ide"`, removes `detect` and `workspace` imports
- `plan_test.go` inline YAML cleanup (remove `project:` snippets; behavioral no-op)
- `qode ide setup` / `qode ide sync` must not appear in `qode --help` output after deletion

### Out of scope

- Version enforcement on `qode_version` — follow-up issue
- `qode init` `--force` / idempotency protection — not requested
- Deleting stale `.cursorrules/` files — user's responsibility
- Any new `qode ide`-style subcommand — explicitly removed, not replaced

---

## 5. Actionable Implementation Plan

Each task = one commit. Critical ordering: templates before `TemplateData` field removal; package deletions before schema cleanup; `ide.go` deletion and `root.go` update in the same commit.

### Task 1 — Delete `internal/detect/` and `internal/workspace/`

Delete all 10 files in `internal/detect/` and 2 files in `internal/workspace/`.

**Prerequisite for**: Tasks 2 and 9 (removes their imports).

### Task 2 — Update `internal/config/schema.go` and `config.go`

- Remove: `ProjectConfig`, `LayerConfig`, `TestConfig`, `CoverageConfig`, `WorkspaceConfig`, `RepoRef`, `Topology` + constants, `Config.Project`, `Config.Workspace`, `Config.Layers()`
- Add: `QodeVersion string \`yaml:"qode_version,omitempty"\`` to `Config`
- `config.go:15`: remove `WorkspaceConfigFileName`

### Task 3 — Update `internal/config/defaults.go`

- Delete `StackDefaults` map (lines 123–176)
- Remove `Project: ProjectConfig{Topology: TopologySingle}` from `DefaultConfig()`
- Add `QodeVersion: "0.1"` to `DefaultConfig()`

### Task 4 — Update `internal/config/config_test.go`

- Delete `TestConfigLayers_Shorthand` and `TestConfigLayers_Composite`
- Update `TestSave_Load`: remove `cfg.Project.Name` / `cfg.Project.Layers`; assert `Review.MinCodeScore` round-trips
- Update `TestDefaultConfig`: assert `cfg.QodeVersion == "0.1"`

### Task 5 — Update five prompt templates (must precede Task 6)

- `refine/base.md.tmpl`: remove `## Tech Stack:` heading + `{{range .Layers}}...{{end}}` (lines 10–12); keep `**Project:** {{.Project.Name}}` and `**Branch:** {{.Branch}}`
- `spec/base.md.tmpl`: same
- `start/base.md.tmpl`: remove lines 10–12 (first range); replace lines 41–49 (second range) with universal clean code list
- `review/code.md.tmpl`: remove `## Tech Stack:` heading + range block (lines 21–23)
- `review/security.md.tmpl`: same

### Task 6 — Add `TemplateProject`, `Engine.ProjectName()`, update `TemplateData` and all callers

- `internal/prompt/engine.go`: add `type TemplateProject struct { Name string }`; change `TemplateData.Project` to `TemplateProject`; remove `TemplateData.Layers`; add `func (e *Engine) ProjectName() string { return filepath.Base(e.root) }`
- `internal/plan/refine.go`: replace `Project: cfg.Project` with `Project: prompt.TemplateProject{Name: engine.ProjectName()}` and remove `Layers: cfg.Layers()` in `BuildRefinePromptWithOutput`, `BuildSpecPromptWithOutput`, `BuildStartPrompt`
- `internal/review/review.go`: same in `BuildCodePrompt`, `BuildSecurityPrompt`
- `internal/cli/knowledge_cmd.go`: same in `buildBranchLessonData`

### Task 7 — Update `internal/ide/cursor.go` and `claudecode.go`

**cursor.go:**
- Delete `cursorRulesDir` constant (line 12)
- Delete `os.MkdirAll` for `cursorRulesDir` + two rule file `writeFile` calls (lines 17–31)
- Delete `workflowRule()` (lines 47–79), `cleanCodeRule()` (lines 81–102)
- Delete `layerList`, `collectStacks`, `globsForStacks`, `stackCleanCodeRules`, `dedup`
- Change `slashCommands(cfg *config.Config)` → `slashCommands(name string, cfg *config.Config)`
- Update `SetupCursor`: `name := filepath.Base(root)`; pass to `slashCommands`; update `fmt.Printf` → `"  Cursor: .cursor/commands/ (%d commands)\n", len(cmds)`

**claudecode.go:**
- Change `claudeSlashCommands(cfg *config.Config)` → `claudeSlashCommands(name string, cfg *config.Config)`
- Update `SetupClaudeCode`: `name := filepath.Base(root)`; pass to `claudeSlashCommands`

### Task 8 — Update `internal/ide/ide_test.go`

- `minimalConfig()`: remove `cfg.Project.Name` and `cfg.Project.Topology` (compiler-forced after Task 2)
- Update `slashCommands(cfg)` → `slashCommands("testproject", cfg)` and `claudeSlashCommands(cfg)` → `claudeSlashCommands("testproject", cfg)` throughout
- Fix `TestCursorSlashCommands_ContainsTicketFetch` line 101: change `cfg.Project.Name` → `"testproject"`
- Add `TestSetupCursor_NoCursorRulesDir`: assert `.cursorrules/` dir does not exist after `SetupCursor`
- Add `TestSetupCursor_CommandsContainRootName`: temp dir named `myproject`; assert `.cursor/commands/qode-plan-refine.mdc` contains `"myproject"`
- Add `TestSetupClaudeCode_CommandsContainRootName`: same for `.claude/commands/qode-plan-refine.md`

### Task 9 — Delete `internal/cli/ide.go`, update `root.go`, update `init.go`

**Delete `internal/cli/ide.go`** (defines `newIDECmd`, `newIDESetupCmd`, `newIDESyncCmd`).

**`internal/cli/root.go`**: remove `newIDECmd(),` from line 65. (Must be same commit as deleting `ide.go` to avoid compile break.)

**`internal/cli/init.go`**:
- Remove imports: `detect`, `workspace`
- Add import: `"github.com/nqode/qode/internal/ide"`
- Remove from `runInitExisting`: `workspace.Detect` (line 39), `detect.Composite` (line 44), detection output `fmt.Printf` calls (lines 57–62), layer-building loop (lines 64–78)
- After `copyEmbeddedTemplates(root)` add: `if err := ide.Setup(root, cfg); err != nil { return fmt.Errorf("setting up IDE configs: %w", err) }`
- Update `Long`: remove "Scans the current directory, detects the tech stack…Run 'qode ide setup' afterwards"; new: "Writes a minimal qode.yaml, creates the .qode/ directory structure, copies prompt templates, and generates IDE slash commands."
- Update next-steps: remove "Run 'qode ide setup'"; add note that re-running `qode init` resets `qode.yaml` to defaults

### Task 10 — Update `internal/cli/plan_test.go` (cleanup)

Remove `project:` sections from inline YAML strings in `TestRunPlanSpec_*` tests. No behavior change.

### Task 11 — Add `internal/cli/init_test.go` (new file)

New file `package cli`:
- `TestRunInitExisting_WritesQodeVersion`: assert `qode_version: "0.1"` present, `project:` absent in written `qode.yaml`
- `TestRunInitExisting_CreatesDirs`: assert `.qode/branches/`, `.qode/knowledge/`, `.qode/prompts/` exist
- `TestRunInitExisting_CopiesTemplates`: assert ≥1 `.md.tmpl` file under `.qode/prompts/`
- `TestRunInitExisting_CreatesIDEConfigs`: assert `.claude/commands/qode-plan-refine.md` and `.cursor/commands/qode-plan-refine.mdc` exist
- `TestRunInitExisting_NoCursorRules`: assert `filepath.Join(root, ".cursorrules")` does not exist
- `TestRunInitExisting_NoDetectionOutput`: capture stdout, assert "Detected", "Scanning", "qode ide setup" absent
- `TestRootCmd_NoIDESubcommand`: execute root cobra command with args `["ide", "--help"]`, assert non-zero exit or error contains "unknown command"

### Task 12 — Add prompt render tests

- `internal/plan/refine_test.go`: add `TestBuildRefinePromptWithOutput_ContainsProjectName` — engine rooted at temp dir named `myproject`; assert rendered prompt contains `"myproject"`
- `internal/review/review_test.go`: add `TestBuildCodePrompt_ContainsProjectName`, `TestBuildSecurityPrompt_ContainsProjectName` — same pattern

### Task 13 — Update root `qode.yaml`

- Remove entire `project:` section
- Add `qode_version: "0.1"` at top

### Task 14 — Update docs and README

- `docs/qode-yaml-reference.md`: remove `project`/`topology`/`layers`/`test`/`coverage` sections; remove composite stack example; add `qode_version` entry; remove all `qode ide` references
- `README.md`:
  - Line 29: remove `qode ide setup` from quick-start
  - Line 110: remove `.cursorrules/*.mdc` from Cursor IDE table; keep `.cursor/commands/*.mdc`
  - Lines 144–145: remove `qode ide setup` and `qode ide sync` command reference rows
  - Lines 166, 178, 190: update monorepo/workspace examples that show separate `qode init` + `qode ide setup` steps

### Implementation order summary

```
Task 1  (delete detect + workspace packages)
→ Task 2  (config schema + config.go)
→ Task 3  (defaults.go)
→ Task 4  (config_test.go)
→ Task 5  (templates)          ← must precede Task 6
→ Task 6  (TemplateData + Engine.ProjectName + callers)
→ Task 7  (cursor.go + claudecode.go)
→ Task 8  (ide_test.go)
→ Task 9  (delete ide.go + root.go + init.go)  ← ide.go + root.go in same commit
→ Task 10 (plan_test.go cleanup)
→ Task 11 (init_test.go — new)
→ Task 12 (render tests)
→ Task 13 (qode.yaml)
→ Task 14 (docs + README)
```

`go build ./...` and `go test ./...` must pass after Task 6. Tasks 7–14 are independently verifiable.
