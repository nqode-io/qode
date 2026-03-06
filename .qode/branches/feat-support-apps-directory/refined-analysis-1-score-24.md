<!-- qode:iteration=1 score=24/25 -->

# Requirements Refinement — feat-support-apps-directory

## 1. Problem Understanding

### Restatement
Some monorepos organize their projects inside an `apps/` (or similar) subdirectory rather than placing them directly at the repository root. Currently, `qode init` only scans **one level deep** from the repo root when detecting topology and tech stacks. This means projects nested under `apps/frontend/`, `apps/backend/`, etc. are invisible to both `workspace.Detect()` and `detect.Composite()`. The feature must extend the scanning logic so that container directories like `apps/` are traversed to discover the actual project directories within them.

### User Need & Business Value
Developers working in monorepos with an `apps/` (or `packages/`, `services/`, `projects/`) directory structure cannot use `qode init` out of the box — the tool detects the wrong topology (typically `single` instead of `monorepo`) and misses all tech layers nested inside the container directory. This forces manual editing of `qode.yaml`, which defeats the purpose of automatic detection. Fixing this makes `qode init` work correctly for a common monorepo pattern (Turborepo, Nx, Lerna, Rush, custom setups).

### Ambiguities & Open Questions
1. **Which container directory names should be recognized?** Only `apps/`, or also `packages/`, `services/`, `projects/`, `libs/`? Common monorepo tools use different conventions:
   - Turborepo / Nx: `apps/`, `packages/`, `libs/`
   - Lerna: `packages/`
   - Rush: `apps/`, `libraries/`
   - Custom: `services/`, `projects/`
2. **Should the depth be configurable?** E.g., `apps/group/frontend/` (3 levels deep). The simplest approach is to go exactly one extra level (root → container → project).
3. **Should `qode init --new --scaffold` also support creating the `apps/` structure?** The ticket only mentions `qode init` (existing project detection), not scaffolding.
4. **What about workspace files?** Some monorepo tools have workspace config at root (e.g., `pnpm-workspace.yaml`, `turbo.json`, `nx.json`) that explicitly list the `apps/` and `packages/` globs. Should we parse these for more accurate detection?

## 2. Technical Analysis

### Affected Layers/Components

| Component | File | Change Required |
|-----------|------|-----------------|
| **Workspace detection** | `internal/workspace/workspace.go` | `Detect()` must recurse into container dirs |
| **Tech stack detection** | `internal/detect/detect.go` | `Composite()` must recurse into container dirs |
| **Init command** | `internal/cli/init.go` | `runInitExisting()` may need to pass detected container dirs through |
| **Config schema** | `internal/config/schema.go` | LayerConfig.Path must handle `./apps/frontend` style paths |

### Key Technical Decisions

1. **Container directory recognition strategy**: Define a set of well-known container directory names (e.g., `apps`, `packages`, `services`, `libs`, `libraries`, `projects`) that, when encountered at the first level, trigger a second-level scan of their children. This is the simplest approach and avoids unbounded recursion.

2. **Where to implement the logic**: The scanning happens in two places that must stay consistent:
   - `workspace.Detect()` (lines 24-61 in `workspace.go`) — topology classification. Currently iterates `os.ReadDir(root)` and checks immediate subdirs. Must also iterate children of recognized container dirs.
   - `detect.Composite()` (lines 14-54 in `detect.go`) — tech layer discovery. Currently iterates `os.ReadDir(root)` and calls `detectAt()` on immediate subdirs. Must also scan children of container dirs.

3. **Path representation**: Detected layers must use paths like `./apps/frontend` (not just `./frontend`). The existing `LayerConfig.Path` field already supports arbitrary relative paths, so no schema change needed.

4. **Topology impact**: A container dir with 2+ project children should contribute to `techDirCount` in `workspace.Detect()`, potentially flipping topology from `single` to `monorepo`.

### Patterns/Conventions to Follow

- The existing skip-list in `detect.Composite()` (`node_modules`, `vendor`, `dist`, `build`, `.git`, `__pycache__`, `bin`, `obj`) should also apply inside container dirs.
- The existing `looksLikeProjectDir()` check should be reused for children of container dirs.
- Confidence scoring and stack superseding logic in `detect.go` should work unchanged since `detectAt()` already accepts arbitrary paths.

### Dependencies
- No external dependencies. All changes are internal to `qode`.
- Tests for workspace detection and composite detection will need new fixtures.

## 3. Risk & Edge Cases

### What Could Go Wrong
- **False positive container dirs**: A directory named `apps` that is itself a project (has `package.json` at `apps/` level) could be treated as a container when it's actually a project. **Mitigation**: If a directory both looks like a project AND has project-like children, treat it as a project (not a container). Only recurse if the directory itself has no tech markers.
- **Performance**: Adding a second level of scanning doubles the I/O for large repos. **Mitigation**: Only scan children of recognized container names, not all subdirectories. The current skip-list already excludes heavy directories.

### Edge Cases
1. **`apps/` exists but is empty or has no project children** → Ignore it, behave as before.
2. **`apps/` itself has a `package.json`** (e.g., a workspace root package) → Treat it as a project, not a container. But also check children, since workspace roots often have a `package.json` with `"workspaces"` field but no real code.
3. **Nested containers**: `apps/group/subapp/` — only go one extra level. Do not recurse indefinitely.
4. **Symlinks inside `apps/`** → Follow the same convention as the rest of the codebase (currently no symlink handling; `os.ReadDir` doesn't follow symlinks by default — this is fine).
5. **Mixed layout**: Some projects at root level AND some inside `apps/` → Both should be detected. The deduplication logic in `Composite()` (line 44) handles this.
6. **Hidden directories inside container** (e.g., `apps/.internal/`) → Should be skipped, consistent with the existing `strings.HasPrefix(e.Name(), ".")` check in `workspace.go` line 30.

### Security Considerations
- None significant. This is local filesystem scanning with no user input beyond the directory path.

### Performance Implications
- Minimal. Adding one extra level of `os.ReadDir` for a handful of well-known directory names is negligible.

## 4. Completeness Check

### Acceptance Criteria
1. `qode init` on a repo with structure `apps/<project>/` correctly detects topology as `monorepo`.
2. `qode init` on a repo with structure `apps/<project>/` correctly detects all tech layers inside `apps/`.
3. Generated `qode.yaml` contains layers with paths like `./apps/frontend`, `./apps/backend`.
4. Existing single-project and flat-monorepo detection is not broken (no regression).
5. Container directory names recognized: at minimum `apps`, `packages`, `services`, `libs`.

### Implicit Requirements
- The skip-list for excluded directories must apply inside container dirs too.
- Layer deduplication must work correctly when projects exist both at root and inside containers.
- The `--new` and `--workspace` init modes are not affected (ticket only mentions existing project detection).

### Out of Scope
- Parsing workspace config files (`pnpm-workspace.yaml`, `turbo.json`, `nx.json`) for explicit glob patterns — nice to have but not required by the ticket.
- Scaffolding `apps/` structure via `qode init --new --scaffold`.
- Supporting arbitrary nesting depth beyond root → container → project (2 levels).
- Multi-repo workspace changes (`runInitWorkspace`).

## 5. Actionable Implementation Plan

### Prerequisites
- None. All changes are additive.

### Tasks (in order)

**Task 1: Define well-known container directory names**
- File: `internal/workspace/workspace.go`
- Add a package-level variable `knownContainerDirs` as a `map[string]bool` with entries: `apps`, `packages`, `services`, `libs`, `libraries`, `projects`.
- Add a helper function `isContainerDir(name string) bool` that checks the map.

**Task 2: Extend `workspace.Detect()` to recurse into container dirs**
- File: `internal/workspace/workspace.go`, function `Detect()` (lines 24-61)
- When iterating immediate subdirs, if a subdir name is in `knownContainerDirs` AND `!looksLikeProjectDir(subAbs)`, iterate its children with the same `isGitRepo` / `looksLikeProjectDir` checks.
- Increment `techDirCount` / `subRepoCount` for matching children.

**Task 3: Extend `detect.Composite()` to recurse into container dirs**
- File: `internal/detect/detect.go`, function `Composite()` (lines 14-54)
- When iterating immediate subdirs, if a subdir name is in `knownContainerDirs`, iterate its children and call `detectAt()` with path `"./" + containerName + "/" + childName`.
- Apply the same skip-list to children of container dirs.

**Task 4: Add unit tests for workspace detection with container dirs**
- File: `internal/workspace/workspace_test.go` (create or extend)
- Test cases:
  - Repo with `apps/frontend/package.json` + `apps/backend/go.mod` → `TopologyMonorepo`
  - Repo with `apps/` but no project children → `TopologySingle`
  - Repo with `apps/package.json` (project at apps level) → treated as project, not container

**Task 5: Add unit tests for composite detection with container dirs**
- File: `internal/detect/detect_test.go` (create or extend)
- Test cases:
  - `apps/frontend/package.json` with React deps → detected as `react` layer at `./apps/frontend`
  - `packages/shared/tsconfig.json` → detected as `typescript` layer at `./packages/shared`
  - Mixed: root `go.mod` + `apps/web/package.json` → both layers detected

**Task 6: Integration test with `qode init`**
- Create a temp directory with an `apps/` structure, run `qode init`, verify `qode.yaml` output.
- File: `internal/cli/init_test.go` (create or extend)
