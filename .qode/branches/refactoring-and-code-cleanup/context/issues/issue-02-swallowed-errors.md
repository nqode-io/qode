# Issue #2: Swallowing Errors in Non-Cleanup Paths

## Summary

`ParseIterationFromOutput()` in `internal/plan/refine.go` silently ignores I/O errors when writing two essential files:
1. The numbered iteration file: `refined-analysis-<N>-score-<S>.md`
2. The canonical file: `refined-analysis.md`

Both writes use `_ = os.WriteFile(...)`, discarding any error. On disk-full, permission denied, or missing directory, the function returns a successful result while the files were never actually written. Downstream operations (`BuildJudgePrompt`, context loading, `LatestScore`) then operate on stale or absent data.

## Affected Files

**Source of the problem:**
- `internal/plan/refine.go` — `ParseIterationFromOutput()` lines 138–157
  - Line 149: `_ = os.WriteFile(iterFile, []byte(analysisText), 0644)`
  - Line 154: `_ = os.WriteFile(latestFile, []byte(header+analysisText), 0644)`
  - No `os.MkdirAll` call before either write (unlike sibling functions)

**Correct reference patterns:**
- `internal/plan/refine.go` — `SaveIterationResult()` lines 121–135: propagates all errors correctly
- `internal/plan/refine.go` — `SaveIterationFiles()` lines 87–100: propagates all errors correctly

**Tests needing updates:**
- `internal/plan/refine_test.go` lines 223, 245: both test callers of `ParseIterationFromOutput`

## Current State

```go
// ParseIterationFromOutput (lines 138–157)
func ParseIterationFromOutput(analysisText, branchDir string, rubric scoring.Rubric) (scoring.Result, error) {
    result := scoring.ParseScore(analysisText, rubric)
    // ...
    iterFile := filepath.Join(branchDir, fmt.Sprintf("refined-analysis-%d-score-%d.md", n, score))
    latestFile := filepath.Join(branchDir, "refined-analysis.md")

    _ = os.WriteFile(iterFile, []byte(analysisText), 0644)          // line 149: error swallowed
    _ = os.WriteFile(latestFile, []byte(header+analysisText), 0644) // line 154: error swallowed

    return result, nil
}
```

Compare with the correct sibling pattern in `SaveIterationResult()`:

```go
func SaveIterationResult(...) error {
    if err := os.MkdirAll(branchDir, 0755); err != nil {
        return err
    }
    if err := os.WriteFile(iterFile, ...); err != nil {
        return err
    }
    return os.WriteFile(latestFile, ...)
}
```

`ParseIterationFromOutput` also lacks the `os.MkdirAll` call, so if the branch directory doesn't exist both writes fail silently.

## Proposed Fix

Three concrete changes inside `ParseIterationFromOutput`:

```go
// 1. Add directory creation
if err := os.MkdirAll(branchDir, 0755); err != nil {
    return result, fmt.Errorf("create branch directory %q: %w", branchDir, err)
}

// 2. Replace line 149
if err := os.WriteFile(iterFile, []byte(analysisText), 0644); err != nil {
    return result, fmt.Errorf("write iteration file %q: %w", iterFile, err)
}

// 3. Replace line 154
if err := os.WriteFile(latestFile, []byte(header+analysisText), 0644); err != nil {
    return result, fmt.Errorf("write canonical analysis file %q: %w", latestFile, err)
}
```

No caller signature changes needed — the function already returns `(scoring.Result, error)`.

Update the two existing tests in `refine_test.go` (lines 223, 245) to assert that both files exist on the filesystem after the call.

## Impact

| Failure mode | Current behavior | After fix |
|---|---|---|
| Disk full | Silent: function returns success, files missing | Error returned to caller |
| Permission denied | Silent: files not written | Error returned to caller |
| Missing branch directory | Silent: both writes fail | Error returned, directory created |
| Stale `refined-analysis.md` | Downstream judge prompt uses old data | Write failure surfaces before downstream use |

**Severity:** Medium — data loss is silent. The CI pipeline can evaluate an old analysis, silently pass gating, and advance to the wrong workflow step.
