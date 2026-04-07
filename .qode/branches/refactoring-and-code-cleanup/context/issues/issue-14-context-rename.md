# Issue #14: Rename `internal/context` to `internal/branchcontext`

## Summary

The package name `context` clashes with the Go stdlib `context` package. Every CLI file that imports both must alias the internal package as `gocontext` to avoid the conflict. Renaming the directory and package to `branchcontext` eliminates the alias entirely — the package name now reflects its purpose (branch-scoped context) and requires no workaround.

## Affected Files

**11 files total need updated import paths.**

**5 CLI files that use the `gocontext` alias:**

- `internal/cli/plan.go:9` — `gocontext "github.com/nqode-io/qode/internal/context"`
- `internal/cli/review.go:10`
- `internal/cli/start.go:10`
- `internal/cli/knowledge_cmd.go:10`
- `internal/cli/help.go:7`

**6 non-CLI files that import without alias (no stdlib `context` conflict there):**

- `internal/plan/refine.go:10`
- `internal/review/review.go:5`
- `internal/workflow/guard.go:8`
- `internal/plan/refine_test.go:10`
- `internal/review/review_test.go:10`
- `internal/workflow/guard_test.go:8`

**Package source:**

- `internal/context/context.go` — package declaration needs updating
- `internal/context/context_test.go` — package declaration needs updating

## Current State

CLI files must alias the import to avoid the stdlib name clash:

```go
// internal/cli/start.go
import (
    "context"  // stdlib — if imported
    gocontext "github.com/nqode-io/qode/internal/context" // alias required
)

// usage
ctx, err := gocontext.Load(root, branch)
```

Non-CLI files happen to avoid the clash because they don't import stdlib `context`, so they can use the bare name — but this is fragile (adding any stdlib `context` usage would immediately require adding an alias).

## Proposed Fix

1. **Rename directory**: `internal/context/` → `internal/branchcontext/`
2. **Update package declarations** in both `.go` files: `package context` → `package branchcontext`
3. **Update all 11 import paths**: `internal/context` → `internal/branchcontext`
4. **Remove aliases** from the 5 CLI files; replace `gocontext.Load(...)` → `branchcontext.Load(...)`

After the rename, all files use a clean, unambiguous import:

```go
import (
    "github.com/nqode-io/qode/internal/branchcontext"
)

// usage — no alias needed
ctx, err := branchcontext.Load(root, branch)
```

The rename is a purely mechanical change — no logic, no behavior, no API surface changes. `gorename` or a simple find-and-replace across the 11 files handles it completely.

## Impact

- **Clarity**: `branchcontext` is self-describing; `gocontext` is a workaround with no semantic meaning
- **Robustness**: any file can now import both stdlib `context` and this package without a naming collision
- **Low risk**: internal package only; zero external API changes; fully covered by existing tests after the rename
