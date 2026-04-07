# Issue #10: Define an Explicit Interface for the Prompt Engine

## Summary

CLI command files and domain packages (`plan`, `review`) all accept `*prompt.Engine` as a concrete type. Only two methods are ever called on it: `Render()` and `ProjectName()`. Defining a two-method `Renderer` interface would decouple these packages from the engine's constructor and filesystem, and allow unit tests to inject a stub renderer without any filesystem setup.

## Affected Files

**Domain packages accepting `*prompt.Engine`:**

- `internal/plan/refine.go` lines 23, 29, 58, 64, 75, 104 — 6 functions
- `internal/review/review.go` lines 12, 26 — 2 functions
- `internal/cli/knowledge_cmd.go:176` — `buildBranchLessonData`

**CLI commands instantiating `*prompt.Engine`:**

- `internal/cli/plan.go` lines 113, 157, 218
- `internal/cli/review.go:94`
- `internal/cli/start.go:78`
- `internal/cli/knowledge_cmd.go:144`

**Tests that must construct a real filesystem for `NewEngine`:**

- `internal/plan/refine_test.go` lines 90, 116, 142, 167, 256, 288
- `internal/review/review_test.go` lines 16, 45, 87, 114, 134

## Current State

```go
// internal/plan/refine.go
func BuildRefinePrompt(engine *prompt.Engine, cfg *config.Config, ctx *context.Context, ...) (*RefineOutput, error) {
    data := prompt.TemplateData{
        Project: prompt.TemplateProject{Name: engine.ProjectName()},
        // ...
    }
    workerPrompt, err := engine.Render("refine/base", data)
    // ...
}
```

Tests must pay the full `NewEngine` cost (temp dir, embedded template parsing) even when the test only cares about the data structure being assembled:

```go
func TestBuildSpecPromptWithOutput_OmitsAnalysis(t *testing.T) {
    root := t.TempDir()                    // filesystem required
    engine, err := prompt.NewEngine(root) // full engine, not just Render+ProjectName
    // ...
}
```

## Proposed Fix

### Step 1: Define the interface

New file `internal/prompt/renderer.go`:

```go
package prompt

// Renderer is the minimal interface required by packages that render prompt templates.
type Renderer interface {
    Render(name string, data TemplateData) (string, error)
    ProjectName() string
}
```

`*Engine` already satisfies this interface — no changes to `engine.go`.

### Step 2: Update domain package function signatures

Replace `engine *prompt.Engine` → `renderer Renderer` in all 9 domain functions:

```go
// internal/plan/refine.go — all 6 functions
func BuildRefinePrompt(renderer Renderer, cfg *config.Config, ...) (*RefineOutput, error)
func BuildRefinePromptWithOutput(renderer Renderer, ...) (*RefineOutput, error)
func BuildSpecPrompt(renderer Renderer, ...) (string, error)
func BuildSpecPromptWithOutput(renderer Renderer, ...) (string, error)
func BuildStartPrompt(renderer Renderer, ...) (string, error)
func BuildJudgePrompt(renderer Renderer, ...) (string, error)

// internal/review/review.go — 2 functions
func BuildCodePrompt(renderer Renderer, ...) (string, error)
func BuildSecurityPrompt(renderer Renderer, ...) (string, error)

// internal/cli/knowledge_cmd.go
func buildBranchLessonData(root string, renderer Renderer, branches []string) (prompt.TemplateData, error)
```

CLI commands require **no changes** — they already pass `*Engine`, which satisfies `Renderer` implicitly.

### Step 3: Add a test stub

New file `internal/prompt/stub.go`:

```go
package prompt

import "fmt"

// StubRenderer is a test double for Renderer.
type StubRenderer struct {
    RenderFunc      func(name string, data TemplateData) (string, error)
    ProjectNameFunc func() string
}

var _ Renderer = (*StubRenderer)(nil)

func (s *StubRenderer) Render(name string, data TemplateData) (string, error) {
    if s.RenderFunc != nil {
        return s.RenderFunc(name, data)
    }
    return fmt.Sprintf("rendered:%s", name), nil
}

func (s *StubRenderer) ProjectName() string {
    if s.ProjectNameFunc != nil {
        return s.ProjectNameFunc()
    }
    return "test-project"
}
```

Tests that focus on data assembly can now use the stub without any filesystem:

```go
func TestBuildJudgePrompt_ReferencesAnalysis(t *testing.T) {
    stub := &prompt.StubRenderer{}
    got, err := plan.BuildJudgePrompt(stub, &config.Config{}, ctx)
    // assert on got without needing a real temp dir or template files
}
```

## Impact

- **Testability**: domain-package tests can run without a filesystem; `t.TempDir()` + `NewEngine` only needed for full integration tests
- **Decoupling**: `internal/plan` and `internal/review` no longer depend on the engine constructor
- **Documentation**: function signatures communicate "you only need Render + ProjectName"
- **CLI commands**: zero changes — `*Engine` satisfies `Renderer` automatically
- **Risk**: low — all changes are inside `internal/`, no public API affected
