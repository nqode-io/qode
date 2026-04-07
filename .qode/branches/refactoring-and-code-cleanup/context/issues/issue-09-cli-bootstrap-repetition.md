# Issue #9: Extract the CLI Bootstrap Sequence into a Reusable Helper

## Summary

Every command that renders a prompt repeats the same 5-step bootstrap sequence: `resolveRoot() → config.Load() → git.CurrentBranch() → context.Load() → prompt.NewEngine()`. This pattern appears in 8 command implementations across 5 files, totalling 40+ lines of identical boilerplate and 5 independent error-handling chains. Extracting it into a `loadSession()` helper eliminates the duplication and gives one place to add logging, tracing, or new session fields.

## Affected Files

| File | Functions with full bootstrap | Lines |
|------|-------------------------------|-------|
| `internal/cli/plan.go` | `runPlanJudge`, `runPlanRefine`, `runPlanSpec` | 89–101, 139–151, 183–200 |
| `internal/cli/review.go` | `runReview` | 57–68 |
| `internal/cli/start.go` | `newStartCmd` RunE | 29–46 |
| `internal/cli/knowledge_cmd.go` | `runKnowledgeAddBranch` | 135–147 |
| `internal/cli/help.go` | `runWorkflowStatus` | 47–63 |

## Current State

All affected functions contain a near-identical block:

```go
root, err := resolveRoot()
if err != nil {
    return err
}
cfg, err := config.Load(root)
if err != nil {
    return err
}
branch, err := git.CurrentBranch(root)
if err != nil {
    return err
}
ctx, err := gocontext.Load(root, branch)
if err != nil {
    return err
}
engine, err := prompt.NewEngine(root)
if err != nil {
    return err
}
```

This block appears with minor variations in 8 places. Some commands (e.g., `runKnowledgeList`) need only a subset of the session — root + config — without a branch or engine.

## Proposed Fix

Create `internal/cli/session.go`:

```go
package cli

import (
    "github.com/nqode-io/qode/internal/config"
    gocontext "github.com/nqode-io/qode/internal/context"
    "github.com/nqode-io/qode/internal/git"
    "github.com/nqode-io/qode/internal/prompt"
)

// Session holds all state resolved during CLI command bootstrap.
type Session struct {
    Root    string
    Config  *config.Config
    Branch  string
    Context *gocontext.Context
    Engine  *prompt.Engine
}

// loadSession resolves the full session: root, config, current branch, context, and engine.
func loadSession() (*Session, error) {
    root, err := resolveRoot()
    if err != nil {
        return nil, err
    }
    cfg, err := config.Load(root)
    if err != nil {
        return nil, err
    }
    branch, err := git.CurrentBranch(root)
    if err != nil {
        return nil, err
    }
    ctx, err := gocontext.Load(root, branch)
    if err != nil {
        return nil, err
    }
    engine, err := prompt.NewEngine(root)
    if err != nil {
        return nil, err
    }
    return &Session{
        Root:    root,
        Config:  cfg,
        Branch:  branch,
        Context: ctx,
        Engine:  engine,
    }, nil
}
```

Each command reduces its bootstrap to one line:

```go
func runPlanRefine(ticketURL string, toFile bool) error {
    sess, err := loadSession()
    if err != nil {
        return err
    }
    // use sess.Root, sess.Config, sess.Branch, sess.Context, sess.Engine
}
```

For commands that don't need a branch (e.g., `runKnowledgeList`), a `loadSessionRoot()` variant can return just root + config.

## Impact

- **DRY**: 40+ lines of boilerplate consolidated into one function
- **Consistency**: all commands bootstrap identically — one place to add debug logging or future tracing
- **Extensibility**: adding a new session field (e.g., a logger) requires one change, not 8+
- **Testability**: `loadSession` logic can be tested in isolation
