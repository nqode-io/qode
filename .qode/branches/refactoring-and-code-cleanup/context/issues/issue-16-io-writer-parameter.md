# Issue #16: `io.Writer` Parameter in CLI `run*` Functions

## Summary

All `run*` functions write directly to `os.Stdout` and `os.Stderr` via `fmt.Print*`, making output testing require OS-level pipe gymnastics (`os.Pipe()` swapping `os.Stdout`). Passing `out io.Writer` and `errOut io.Writer` as parameters lets CLI commands wire `os.Stdout`/`os.Stderr` while tests wire a `bytes.Buffer` — consistent with the pure-function style of `workflow.CheckStep`.

## Affected Files

| File | Function | Direct output calls |
|------|----------|---------------------|
| `internal/cli/init.go:38` | `runInitExisting` | 7 stdout writes (lines 56, 80, 93–97) |
| `internal/cli/help.go:47` | `runWorkflowStatus` | 3 stdout writes (lines 69, 72, 73) |
| `internal/cli/knowledge_cmd.go:135` | `runKnowledgeAddBranch` | 3 stderr + 1 stdout (lines 150, 168, 172, 183) |
| `internal/cli/plan.go:88` | `runPlanJudge` | 2 stderr + 1 stdout (lines 108–109, 130, 134) |
| `internal/cli/plan.go:138` | `runPlanRefine` | 1 stderr + 1 stdout (lines 175, 179) |
| `internal/cli/plan.go:183` | `runPlanSpec` | 2 stderr + 2 stdout (lines 207, 214, 236, 240) |
| `internal/cli/review.go:56` | `runReview` | 2 stderr + 1 stdout (lines 78, 117, 121) |

**Total:** ~27 direct `fmt.Print*` / `fmt.Fprintf(os.Stderr)` calls across 7 functions.

## Current State

Direct writes make output un-capturable without OS-level tricks:

```go
// plan.go:207 — stdout, no way to intercept in-process
fmt.Printf("STOP. Do not continue with this prompt.\n\n%s\n...", result.Message)

// plan.go:175 — stderr, same problem
fmt.Fprintf(os.Stderr, "Iteration %d — worker prompt saved to:\n  %s\n", out.Iteration, workerPath)
```

Current test workaround in `plan_test.go:54–78`:

```go
func captureStdout(t *testing.T, fn func()) string {
    r, w, _ := os.Pipe()
    old := os.Stdout
    os.Stdout = w       // mutates global state
    fn()
    w.Close()
    os.Stdout = old     // restore — fragile if fn panics
    var buf bytes.Buffer
    io.Copy(&buf, r)
    return buf.String()
}
```

This cannot separately capture stdout and stderr, mutates process-global state, and is fragile on panic.

**Reference — pure-function style** (`internal/workflow/guard.go:24`):

```go
// CheckStep is pure — no side effects or I/O. Returns a structured result.
func CheckStep(step string, ctx *context.Context, cfg *config.Config) CheckResult { ... }
```

## Proposed Fix

Add `out io.Writer` and `errOut io.Writer` to each `run*` function signature. CLI commands wire `os.Stdout`/`os.Stderr`; tests wire `bytes.Buffer`.

```go
// Before
func runPlanSpec(toFile, force bool) error {
    // ...
    fmt.Printf("STOP. Do not continue...\n")
    fmt.Fprintf(os.Stderr, "Spec prompt saved to:\n  %s\n", promptPath)
}

// After
func runPlanSpec(out, errOut io.Writer, toFile, force bool) error {
    // ...
    fmt.Fprintf(out, "STOP. Do not continue...\n")
    fmt.Fprintf(errOut, "Spec prompt saved to:\n  %s\n", promptPath)
}
```

CLI command `RunE` wires real streams:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    return runPlanSpec(os.Stdout, os.Stderr, toFile, force)
},
```

Test captures output in-process:

```go
func TestRunPlanSpec_GuardBlocked(t *testing.T) {
    var out, errOut bytes.Buffer
    err := runPlanSpec(&out, &errOut, false, false)
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(out.String(), "STOP.") {
        t.Errorf("expected STOP guard message, got: %s", out.String())
    }
}
```

The `captureStdout` helper in `plan_test.go` can then be deleted.

## Impact

- **Testability**: in-process output capture replaces `os.Pipe()` global mutation
- **Separate capture**: tests can independently assert stdout (prompt content) vs stderr (status messages)
- **Consistency**: all `run*` functions follow the same I/O pattern
- **Low risk**: internal functions only; CLI behavior unchanged (same streams wired in production)
