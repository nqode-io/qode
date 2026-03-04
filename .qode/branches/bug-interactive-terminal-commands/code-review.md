# Code Review — bug-interactive-terminal-commands

**Score: 8.5/10**

---

## Summary

The implementation correctly solves the problem. `isTTY()`, `(*claudeCLI).RunInteractive()`, and `RunInteractive()` are clean additions. All four call sites are updated consistently. No regressions in the clipboard fallback or error propagation paths. The main issues are: one silent error swallow in `refineDispatch`, two test gaps, and a potential test flakiness issue.

---

## Critical Issues

None.

---

## High Issues

None.

---

## Medium Issues

### M1 — `refineDispatch`: silent error on `os.ReadFile` after worker session

**File:** `internal/cli/plan.go:152`

```go
savedAnalysis, _ := os.ReadFile(analysisPath)
```

If the interactive worker session exits cleanly but Claude failed to write `analysisPath` (e.g., the AI was aborted mid-write, permission error, or prompt misunderstood), `savedAnalysis` is `[]byte(nil)`. This empty content is then passed to `BuildJudgePrompt` and `SaveIterationResult` silently — the judge runs against an empty analysis, the saved result is meaningless, and the user gets no signal that anything went wrong.

**Fix:** Check the error and return early if the file is missing:

```go
savedAnalysis, err := os.ReadFile(analysisPath)
if err != nil {
    return fmt.Errorf("refine: analysis file not found after worker session — did Claude write %s? (%w)", analysisPath, err)
}
```

### M2 — `TestRunInteractive_TempFileCleanedUp` does not test `RunInteractive`

**File:** `internal/dispatch/claude_test.go:41`

The test creates a temp file manually, removes it manually, and asserts the removal works. It never calls `RunInteractive`. This tests `os.Remove` — not the deferred cleanup in the function under test.

**Fix:** Either delete this test (it adds no value), or replace it with a test that uses a mock binary to verify the temp file is created and removed:

```go
func TestRunInteractive_TempFileCleanedUp(t *testing.T) {
    if isTTY() {
        t.Skip("test requires non-TTY stdin")
    }
    // In non-TTY mode, RunInteractive delegates to c.Run (no temp file).
    // The temp file path is only observable in TTY mode, which requires manual testing.
    // This test documents that limitation.
    t.Log("temp file cleanup is verified by manual TTY testing")
}
```

### M3 — `TestRunInteractive_PackageLevel_ClipboardFallback` may not reach clipboard path

**File:** `internal/dispatch/claude_test.go:63`

```go
t.Setenv("PATH", "")
```

`newClaudeCLI()` falls back to checking `knownClaudePaths` (`~/.local/bin/claude`, `/usr/local/bin/claude`) directly via `os.Stat`, bypassing PATH. If `claude` is installed at either of these locations (which it likely is on a developer machine), the test will call `d.RunInteractive()` instead of clipboard fallback — the test passes for the wrong reason or fails unpredictably.

**Fix:** Also shadow the home dir or explicitly construct a `claudeCLI` with an empty `binaryPath`:

```go
func TestRunInteractive_PackageLevel_ClipboardFallback(t *testing.T) {
    if isTTY() {
        t.Skip("clipboard fallback test requires non-TTY stdin")
    }
    // Directly test the clipboard path without relying on PATH manipulation.
    d := &clipboardDispatcher{}
    _, err := d.Run(context.Background(), "test", Options{})
    if !errors.Is(err, ErrManualDispatch) {
        t.Errorf("expected ErrManualDispatch, got %v", err)
    }
}
```

---

## Low Issues

### L1 — `reviewDispatch` and `specDispatch`: no feedback before interactive session starts

**Files:** `internal/cli/review.go:145`, `internal/cli/plan.go:249`

The original `reviewDispatch` printed `"Running %s review via %s"` before dispatch. The new version has no print statement. The user starts the command and the terminal goes quiet until `claude` initialises (1–2 seconds). The original feedback was minimal but better than nothing.

**Fix:** Add a brief line before the `RunInteractive` call:

```go
// reviewDispatch
fmt.Printf("Starting %s review...\n", kind)

// specDispatch
fmt.Println("Generating spec...")
```

### L2 — `specDispatch` and `reviewDispatch`: no check that output file was written

**Files:** `internal/cli/plan.go:249`, `internal/cli/review.go:145`

The original code had a stdout fallback: if Claude didn't write the file, it saved stdout as a fallback. That fallback is gone. If Claude exits 0 but didn't write the output file, `specDispatch` prints "Spec saved to: ..." pointing to a non-existent file. `reviewDispatch` prints "Score: not found" which is correct but slightly misleading.

This is acceptable for interactive mode (Claude almost always writes the file when instructed), but worth noting as a regression in error clarity.

**Suggested check** (optional for `specDispatch`):
```go
if _, err := os.Stat(specPath); os.IsNotExist(err) {
    return fmt.Errorf("plan spec: claude did not write the spec file at %s", specPath)
}
```

### L3 — `refineDispatch`: `SaveIterationResult` called with score 0 after interactive judge

**File:** `internal/cli/plan.go:177`

```go
result := scoring.ParseScore("", scoring.RefineRubric)
result.TargetScore = 25
plan.SaveIterationResult(root, branch, out.Iteration, string(savedAnalysis), result)
```

This saves iteration N with score 0, which means `qode plan status` will show `0/25` for this iteration even though the user saw the actual score in the terminal during the judge session. This is a UX regression from the original which showed and saved the real score.

The spec acknowledges this trade-off, but it's worth tracking as a follow-up. The workaround is to re-run `qode plan refine --prompt-only` to get a scoreable iteration, or to run the judge pass manually as in the `/qode-plan-refine` IDE workflow.

---

## Positive Observations

- `isTTY()` using stdlib `os.Stdin.Stat()` is the right approach — no new dependency, explicit, testable.
- Temp file approach correctly solves the ARG_MAX/stdin conflict. `defer os.Remove` is placed correctly (before any early returns that follow `CreateTemp`).
- `filterEnv(os.Environ(), "CLAUDECODE")` preserved — no nested session detection issues.
- `exec.CommandContext` used consistently with the existing pattern.
- Progress ticker goroutine cleanly removed — no goroutine leak risk.
- `ErrManualDispatch` fallback paths preserved in all four call sites.
- `time` import cleanly removed from `review.go`.

---

## Required Fixes Before Merge

1. **M1** — Add error check for `os.ReadFile(analysisPath)` in `refineDispatch`.
2. **M2** — Replace `TestRunInteractive_TempFileCleanedUp` with a meaningful test or remove it.
3. **M3** — Fix `TestRunInteractive_PackageLevel_ClipboardFallback` to not rely on PATH manipulation.
