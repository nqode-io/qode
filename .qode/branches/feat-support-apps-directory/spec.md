# Technical Specification — feat-support-apps-directory

## 1. Feature Overview

Extend `qode init` to detect projects nested inside well-known container directories (`apps/`, `packages/`, `libs/`, `libraries/`, `services/`, `projects/`) commonly used by monorepo toolchains (Turborepo, Nx, Lerna, Rush). Currently, both topology detection (`workspace.Detect()`) and tech stack discovery (`detect.Composite()`) only scan one level deep from the repo root, causing monorepos with an `apps/` structure to be misclassified as `single` with missing layers. This feature adds a second-level scan for recognized container directories so that `qode init` produces correct `qode.yaml` output without manual editing.

**Business value:** Developers using common monorepo patterns can onboard with `qode init` out of the box, eliminating manual configuration and reducing friction for the most popular monorepo layouts.

**Success criteria:**
- `qode init` on a repo with `apps/frontend/ + apps/backend/` detects topology `monorepo` and discovers all tech layers.
- Zero regressions on existing single-project and flat-monorepo detection.

## 2. Scope

### In scope
- Extend `workspace.Detect()` to recurse into known container directories for topology classification.
- Extend `detect.Composite()` to recurse into known container directories for tech stack discovery.
- Recognize container directory names: `apps`, `packages`, `libs`, `libraries`, `services`, `projects`.
- Add monorepo signal file detection (`turbo.json`, `nx.json`, `pnpm-workspace.yaml`, `lerna.json`) as a topology tiebreaker.
- Handle edge case where container dir itself has tech markers (workspace root `package.json`).
- Unit and integration tests for new behavior.

### Out of scope
- Parsing workspace config files (`turbo.json`, `nx.json`, `pnpm-workspace.yaml`) for glob patterns.
- Scaffolding `apps/` structure via `qode init --new --scaffold`.
- Arbitrary nesting depth beyond root → container → project (2 levels).
- Changes to `runInitWorkspace()` (multi-repo mode).
- Making the container directory list configurable via `qode.yaml`.

### Assumptions
- Container directory names are stable and well-known across the ecosystem.
- One extra level of scanning is sufficient for all standard monorepo tools.
- The existing `LayerConfig.Path` field already supports multi-segment relative paths (e.g., `./apps/frontend`).

## 3. Architecture & Design

### Component diagram

```
qode init (runInitExisting)
    │
    ├── workspace.Detect(root)          ← MODIFIED
    │     ├── scan immediate subdirs         (existing)
    │     ├── scan children of container dirs (NEW)
    │     └── check monorepo signal files     (NEW)
    │
    ├── detect.Composite(root)          ← MODIFIED
    │     ├── detectAt(root, ".", 0)         (existing)
    │     ├── detectAt(subdir, "./X", 1)     (existing)
    │     └── detectAt(child, "./X/Y", 2)    (NEW)
    │
    └── config.Save(root, cfg)          ← UNCHANGED
```

### Affected components

| Component | File | Change |
|-----------|------|--------|
| `workspace.Detect()` | `internal/workspace/workspace.go` | Add container dir recursion + monorepo signal files |
| `detect.Composite()` | `internal/detect/detect.go` | Add container dir recursion in subdir loop |
| `internal/cli/init.go` | — | No change (calls `Detect` + `Composite` which now return correct results) |
| `internal/config/schema.go` | — | No change (`LayerConfig.Path` already handles `./apps/frontend`) |

### Data flow

1. `runInitExisting()` calls `workspace.Detect(root)`.
2. `Detect()` iterates root entries. For entries matching `knownContainerDirs`, it calls `hasProjectChildren()` to scan one level deeper. It also checks for monorepo signal files at root.
3. Topology is classified: `monorepo` if `techDirCount >= 2` OR (`techDirCount >= 1` AND `hasMonorepoSignal`).
4. `runInitExisting()` calls `detect.Composite(root)`.
5. `Composite()` iterates root entries. For entries matching `knownContainerDirs`, it iterates their children and calls `detectAt()` with path `./container/child` and depth 2.
6. Dedup and sort run as before. Layers with paths like `./apps/frontend` are included in the result.
7. Config is saved to `qode.yaml` with the correct topology and layers.

## 4. API / Interface Contracts

No public APIs, CLI flags, or config schema changes. All modifications are internal to the detection logic.

### Modified internal function behavior

**`workspace.Detect(root string) (Topology, error)`**
- Before: Scans immediate subdirs only.
- After: Also scans children of known container dirs. Uses monorepo signal files as tiebreaker.
- Return values unchanged.

**`detect.Composite(root string) ([]DetectedLayer, error)`**
- Before: Calls `detectAt()` on root + immediate subdirs.
- After: Also calls `detectAt()` on children of known container dirs with depth=2.
- Return type unchanged. Paths now may include two-segment relative paths (e.g., `./apps/frontend`).

### New internal helpers

**`workspace` package:**
```go
// knownContainerDirs — well-known monorepo container directory names.
var knownContainerDirs = map[string]bool{
    "apps": true, "packages": true, "libs": true,
    "libraries": true, "services": true, "projects": true,
}

// monorepoSignalFiles — files at repo root that indicate a monorepo.
var monorepoSignalFiles = []string{
    "turbo.json", "nx.json", "pnpm-workspace.yaml", "lerna.json",
}

// hasProjectChildren returns the number of subdirectories in dir
// that look like project directories, skipping hidden and excluded dirs.
func hasProjectChildren(dir string) int
```

**`detect` package:**
```go
// knownContainerDirs — same list, duplicated to avoid cross-package dependency.
var knownContainerDirs = map[string]bool{
    "apps": true, "packages": true, "libs": true,
    "libraries": true, "services": true, "projects": true,
}

// skipDirs — existing skip-list, extracted to a package-level set for reuse.
var skipDirs = map[string]bool{
    "node_modules": true, "vendor": true, "dist": true, "build": true,
    ".git": true, "__pycache__": true, "bin": true, "obj": true,
}
```

## 5. Data Model Changes

None. The `qode.yaml` schema (`LayerConfig`) already supports arbitrary relative paths in the `Path` field. No migrations required. Full backward compatibility — existing `qode.yaml` files are unaffected.

## 6. Implementation Tasks

- [ ] **Task 1:** (workspace) Add `knownContainerDirs`, `monorepoSignalFiles`, and `hasProjectChildren()` helper to `internal/workspace/workspace.go`
- [ ] **Task 2:** (workspace) Modify `Detect()` to recurse into container dirs and check monorepo signal files
- [ ] **Task 3:** (detect) Add `knownContainerDirs` and `skipDirs` set to `internal/detect/detect.go`; extract existing skip-list `switch` to use the set
- [ ] **Task 4:** (detect) Modify `Composite()` to recurse into container dirs, calling `detectAt()` with depth=2
- [ ] **Task 5:** (test) Add unit tests for `workspace.Detect()` — 6 test cases covering container dirs, empty containers, signal file tiebreaker, and regression
- [ ] **Task 6:** (test) Add unit tests for `detect.Composite()` — 5 test cases covering nested detection, dedup, mixed layouts, and layer naming
- [ ] **Task 7:** (test) Add integration test for `qode init` with `apps/` structure — verify `qode.yaml` output

## 7. Testing Strategy

### Unit tests

**`internal/workspace/workspace_test.go`** — using `t.TempDir()` fixtures:

| # | Fixture | Expected |
|---|---------|----------|
| 1 | `apps/frontend/package.json` + `apps/backend/go.mod` | `TopologyMonorepo` |
| 2 | `apps/` exists but empty | `TopologySingle` |
| 3 | `apps/package.json` only (no children) | `TopologySingle` (1 tech dir) |
| 4 | `apps/package.json` + `apps/web/package.json` + `apps/api/go.mod` | `TopologyMonorepo` (children win) |
| 5 | `turbo.json` at root + `apps/web/package.json` (single child) | `TopologyMonorepo` (signal tiebreaker) |
| 6 | `frontend/package.json` + `backend/go.mod` (flat) | `TopologyMonorepo` (regression) |

**`internal/detect/detect_test.go`** — using `t.TempDir()` fixtures:

| # | Fixture | Expected |
|---|---------|----------|
| 1 | `apps/frontend/package.json` with React deps | `react` layer at `./apps/frontend` |
| 2 | `packages/shared/tsconfig.json` | `typescript` layer at `./packages/shared` |
| 3 | Root `go.mod` + `apps/web/package.json` | Both layers detected |
| 4 | `apps/package.json` + `apps/web/package.json` | Child detected, container-level deduped |
| 5 | Layer at `./apps/frontend` | Name is `frontend` (not `apps`) |

### Integration tests

**`internal/cli/init_test.go`:**
- Create temp dir with `apps/web/package.json` + `apps/api/go.mod`.
- Run init logic, parse generated `qode.yaml`.
- Assert: topology=`monorepo`, layers include `./apps/web` and `./apps/api`, names are `web` and `api`.

### Edge cases to test explicitly
- Empty container dir → no contribution to topology or layers.
- Container dir with tech markers but no project children → treated as regular project.
- Container dir with tech markers AND project children → children take precedence.
- Monorepo signal file + single container child → topology is `monorepo`.
- Hidden dirs inside container → skipped.
- Skip-list dirs inside container → skipped.

## 8. Security Considerations

- No authentication, authorization, or input validation changes.
- All operations are local filesystem reads (`os.ReadDir`, `os.Stat`).
- No user-supplied input beyond the working directory path (already trusted).
- No new attack surface introduced.

## 9. Open Questions

None. All questions from the requirements analysis have been resolved:

1. **Container dir names** → `apps`, `packages`, `libs`, `libraries`, `services`, `projects` (confirmed by notes).
2. **Depth** → Fixed at one extra level.
3. **Shared constant** → Duplicated in both packages (6 stable strings).
4. **Workspace config files** → Used as monorepo signal (file-exists only, no parsing).

---
*Spec generated by qode. Copy to Jira/Azure DevOps ticket for team review.*
