# Issue #12: Structured Logging Instead of `fmt.Fprintf(os.Stderr, ...)`

## Summary

User-facing messages and diagnostic warnings are written via scattered `fmt.Fprintf(os.Stderr, ...)`, `fmt.Fprintln(os.Stderr, ...)`, and `fmt.Print()` calls with no consistent level semantics. A warning looks identical to an informational message; there is no way to filter by severity or produce machine-parseable output for CI pipelines. Adopting `log/slog` (Go stdlib, 1.21+) would add level-based filtering and structured output without any new dependencies.

The issue notes this should be adopted when the CLI grows beyond its current size or when machine-parseable output is needed — both conditions are approaching.

## Affected Files

### stderr writes (diagnostic/status output)

| File | Lines | Category |
|------|-------|----------|
| `cmd/qode/main.go` | 19, 37 | error, warning |
| `internal/cli/plan.go` | 108–109, 130, 175, 213–214, 236 | error, info |
| `internal/cli/start.go` | 59–60, 93 | error, info |
| `internal/cli/review.go` | 78, 117 | info |
| `internal/cli/knowledge_cmd.go` | 150, 168, 183 | info, warning |
| `internal/cli/branch.go` | 130 | warning |

### stdout writes (success/progress output)

| File | Lines | Category |
|------|-------|----------|
| `internal/cli/branch.go` | 132 | success |
| `internal/cli/init.go` | 56, 80, 93–97 | success, instructions |
| `internal/scaffold/scaffold.go` | 28, 32 | info, success |
| `internal/scaffold/cursor.go` | 26 | progress |

**Total:** 16 stderr writes, 14+ stdout writes across 11 files.

## Current State

Three distinct patterns are used interchangeably with no level signal:

```go
// Warning — indistinguishable from info at the call site
fmt.Fprintf(os.Stderr, "Warning: branch context %q: %v\n", b, err)  // knowledge_cmd.go:183

// Informational — same function, different semantic level
fmt.Fprintf(os.Stderr, "Iteration %d — worker prompt saved to:\n  %s\n", out.Iteration, workerPath)  // plan.go:175

// Error guidance — no structured context
fmt.Fprintln(os.Stderr, "No refined analysis found.")
fmt.Fprintf(os.Stderr, "Run 'qode plan refine' first...\n  %s/refined-analysis.md\n", ctx.ContextDir)  // plan.go:108–109
```

## Proposed Fix

Create `internal/log/log.go` with thin `slog` wrappers:

```go
package log

import (
    "log/slog"
    "os"
)

var Logger *slog.Logger

func Init(level slog.Level) {
    Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func Info(msg string, args ...any)  { Logger.Info(msg, args...) }
func Warn(msg string, args ...any)  { Logger.Warn(msg, args...) }
func Error(msg string, args ...any) { Logger.Error(msg, args...) }
func Debug(msg string, args ...any) { Logger.Debug(msg, args...) }
```

Call `log.Init(slog.LevelInfo)` in `main()`. Respect `QODE_LOG_LEVEL=debug` for debug output.

**Migration examples:**

```go
// Before
fmt.Fprintf(os.Stderr, "Warning: branch context %q: %v\n", b, err)
// After
log.Warn("could not load branch context", "branch", b, "error", err)

// Before
fmt.Fprintln(os.Stderr, "No refined analysis found.")
fmt.Fprintf(os.Stderr, "Run 'qode plan refine' first...\n  %s/refined-analysis.md\n", ctx.ContextDir)
// After
log.Error("no refined analysis found", "context_dir", ctx.ContextDir)

// Before
fmt.Fprintf(os.Stderr, "Iteration %d — worker prompt saved to:\n  %s\n", out.Iteration, workerPath)
// After
log.Info("prompt saved", "iteration", out.Iteration, "path", workerPath)
```

For CI/JSON output, swap to `slog.NewJSONHandler(os.Stderr, ...)` — no call site changes needed.

**Prompt/specification output** (the content rendered for the LLM) stays on stdout via `fmt.Print` — it is not logging.

## Impact

- **Level filtering**: `QODE_LOG_LEVEL=debug` reveals internal progress without code changes
- **CI integration**: switch to JSON handler for machine-parseable output in pipelines
- **Consistent format**: all diagnostic output follows one pattern
- **Testability**: log output can be captured via a custom handler in tests
- **No behavior change** for normal usage — informational messages remain visible at `INFO` level
- **No new dependencies** — `log/slog` is Go stdlib since 1.21

Migration can be incremental: `fmt` and `slog` calls can coexist during transition, prioritizing error/warning paths first.
