# Code Review — add-gitignore-rules-during-init

Reviewed: 2026-04-09

---

## Pre-read Incident Report

**Scenario:** `qode init` shipped with the previous Medium-2 defect still present. A user who had manually added all five qode patterns to their `.gitignore` (without the `# qode temp files` header) ran `qode init`. The function passed the old `markerPresent && len(missing) == 0` guard — `markerPresent` was false, `len(missing)` was 0, so both conditions had to be true and weren't — fell through to the write path, assembled a block containing only the marker line, wrote it, and printed "Appended qode ignore rules to .gitignore". Their diff showed a spurious one-line change. CI flagged the unexpected gitignore mutation. On the second run the function found the marker and returned nil — the operation was not idempotent across states.

The fix in this revision collapses idempotency to a single axis: if every rule is present, no write occurs, regardless of whether the marker is present. This is the correct resolution.

---

## Reviewer Stance

**Assumptions in the implementation:**

- `strings.Contains` is sufficient for rule membership detection. True — no rule is a substring of another. Verified: `.qode/branches/*/.*.md` vs `.qode/branches/*/refined-analysis-*-score-*.md`, `.qode/branches/*/diff.md` — no overlap.
- `iokit.WriteFileCtx` with a pre-cancelled context will not create the destination file. The context-cancel test assumes atomic write (temp file + rename aborted before rename). A comment now documents this dependency in the test.
- `markerPresent` is computed before the early-return check even though it is only consumed in the write path. This is correct — it is not dead code — and the organization is cleaner than nesting marker detection inside the write block.

**Earliest silent failure point:** None — all errors are propagated. The previous orphaned-marker silent mutation is resolved.

---

## Issues

### Low-1 — Pre-existing tests in `init_test.go` lack `t.Parallel()`

**Severity:** Low
**File:** [internal/cli/init_test.go:17](internal/cli/init_test.go#L17) — `TestRunInitExisting_WritesQodeVersion` through `TestRunInitExisting_RerunPreservesScoringYaml` (8 tests)

The two new tests (`TestRunInitExisting_AppendsGitignoreRules`, `TestRunInitExisting_GitignoreIsIdempotent`) correctly call `t.Parallel()`. However, eight pre-existing tests in the same file do not, despite using `t.TempDir()` for isolation and reading only `rootCmd.Version` (no write to global state). Not introduced by this diff, but visible in the modified file and inconsistent with CLAUDE.md standards.

**Suggestion:**

```go
func TestRunInitExisting_WritesQodeVersion(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()
    // ...
}
// repeat for all 8 pre-existing tests
```

---

### Nit-1 — Missing test for "marker absent, partial rules present"

**Severity:** Nit
**File:** [internal/scaffold/gitignore_test.go](internal/scaffold/gitignore_test.go)

The test suite covers: no file, no marker (all missing), all present (marker + rules), marker present (2 rules missing), no trailing newline, all rules present (marker absent), context cancelled. Not covered: marker absent with some (not all) rules already present. This exercises the only path where both the marker and a subset of rules are written in a single operation. The code is clearly correct for this case (verified by tracing: `!markerPresent` → write marker; `missing` contains only absent rules → write them), but explicit coverage would lock the behaviour.

**Suggestion:**

```go
{
    name: "marker absent, two rules missing",
    setup: func(t *testing.T, dir string) {
        t.Helper()
        writeGitignore(t, dir, GitignoreRules[0]+"\n"+GitignoreRules[1]+"\n"+GitignoreRules[2]+"\n")
    },
    verify: func(t *testing.T, dir string) {
        t.Helper()
        content := readGitignore(t, dir)
        if !strings.Contains(content, GitignoreMarker) {
            t.Errorf("missing marker %q", GitignoreMarker)
        }
        for _, rule := range GitignoreRules {
            if !strings.Contains(content, rule) {
                t.Errorf("missing rule %q", rule)
            }
            if strings.Count(content, rule) != 1 {
                t.Errorf("rule %q duplicated (count=%d)", rule, strings.Count(content, rule))
            }
        }
    },
    wantOutput: "Appended qode ignore rules to .gitignore",
},
```

---

### Nit-2 — Error chain for write failure is slightly verbose

**Severity:** Nit
**File:** [internal/cli/init.go:97](internal/cli/init.go#L97), [internal/scaffold/gitignore.go:65](internal/scaffold/gitignore.go#L65)

A write failure surfaces as `"appending gitignore rules: updating .gitignore: permission denied"`. Both layers reference `.gitignore`. Not incorrect — the double reference comes from meaningful layering (CLI operation vs. filesystem operation) — but `"appending gitignore rules"` and `"updating .gitignore"` convey overlapping concepts. Not blocking; alternative would be to change the inner error to `"write: %w"` and keep the outer as-is.

---

## What was verified as safe

- **Orphaned-marker edge case resolved:** Early return changed to `if len(missing) == 0 { return nil }` — verified by tracing: all-rules-present-marker-absent returns nil without building or writing `sb`. New test case "all rules present but marker absent" confirms file is not modified and output is empty.
- **Partial-rules path with marker absent:** `markerPresent = false` → marker is written first; `missing` contains only absent rules → only those appended. Verified against the write block at `gitignore.go:56-63`.
- **No double error prefix:** CLI wrapper is `"appending gitignore rules: %w"`, inner is `"updating .gitignore: %w"` — confirmed distinct, no duplication. Read errors remain `"reading .gitignore: %w"` — meaningful layering preserved.
- **No trailing newline:** `strings.HasSuffix(existing, "\n")` guard at line 53 — verified: `"foo"` → prepends `"\n"` before marker/rules; `"foo\n"` → no extra newline inserted.
- **`errors.Is(err, os.ErrNotExist)` is correct:** Distinguishes missing file (empty `existing`, proceed as fresh) from read errors (return wrapped error) — verified at `gitignore.go:35`.
- **No injection risk:** `GitignoreRules` and `GitignoreMarker` are hardcoded; no user input reaches the `.gitignore` write path. `filepath.Join(root, ".gitignore")` — no traversal risk.
- **Idempotency on double-run:** `TestRunInitExisting_GitignoreIsIdempotent` verified: second run finds all rules present (`len(missing) == 0`), returns nil without writing. `strings.Count(content, rule) == 1` for all 5 rules.
- **Dependency layering preserved:** `scaffold` imports `iokit` (leaf) — no new edges. `cli` imports `scaffold` (domain) — existing edge. Verified against CLAUDE.md layer diagram.
- **`scaffold.Setup` error checked before `AppendGitignoreRules`:** No orphaned gitignore write on IDE config failure — verified at `init.go:92-98`.
- **`t.Parallel()` confirmed on all new tests:** Both CLI integration tests and all 7 scaffold unit test cases use `t.Parallel()`.
- **Context-cancel atomic behaviour documented:** Comment added to `gitignore_test.go:146-148` explaining that `iokit.WriteFileCtx` aborts before rename, leaving no destination file.

---

## Summary

| Severity | Count |
| -------- | ----- |
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 1 |
| Nit | 2 |

**Overall:** All Medium issues from the previous review are resolved correctly. The orphaned-marker fix is the right approach — rule-only idempotency is simpler and eliminates the two-run inconsistency. Error prefix disambiguation is clean. New test case for the fixed edge case is properly written. The implementation is correct, idiomatic, and matches all spec scenarios.

**Top 3 things to fix before merging:**

1. (Low-1) Add `t.Parallel()` to the 8 pre-existing CLI tests in `init_test.go` to conform to project standards — they use `t.TempDir()` for isolation and are safe to parallelize.
2. (Nit-1) Add a test case for "marker absent, partial rules present" to cover the one code path where both the marker and a subset of rules are written in a single operation.
3. (Nit-2) Optionally simplify the inner error prefix (`"write: %w"`) to reduce the terminological overlap in the write-failure error chain — low priority.

---

## Rating

| Dimension | Score | What you verified |
| --------- | ----- | ----------------- |
| Correctness (0–3) | 3.0 | Traced all 7 unit test scenarios. Confirmed orphaned-marker edge case (all rules present, marker absent) returns nil at line 48 without entering the write block. Confirmed partial-rules + marker-present path writes only missing rules without duplicating marker (line 56 guard). No-trailing-newline guard correct. All 5 spec test scenarios plus new edge case verified against code. |
| CLI Contract (0–2) | 2.0 | Error prefix chain traced: write failure → `"appending gitignore rules: updating .gitignore: permission denied"` — no double-prefix. `context.Background()` passed at `init.go:96`. Confirmation written to `out` (not stderr) via `fmt.Fprintln(out, ...)` at `gitignore.go:69`. `markerPresent` variable computed before early return but only consumed in write path — correct, not dead code. |
| Go Idioms & Code Quality (0–2) | 2.0 | `strings.Builder` used correctly; `errors.Is(err, os.ErrNotExist)` for ErrNotExist distinction; `filepath.Join` for path construction; function is 41 lines (within 50-line limit); variable names clear; `// Callers must not modify this slice.` doc comment added to exported mutable slice. |
| Error Handling & UX (0–2) | 2.0 | Read failure: `"reading .gitignore: %w"` at line 35 — confirmed. Write failure: `"updating .gitignore: %w"` at line 65 — confirmed. CLI wrapper: `"appending gitignore rules: %w"` at init.go:97 — no duplication. No silent failures anywhere; all error paths propagate. Confirmation message accurate: only printed after a successful write, never on no-op path. |
| Test Coverage (0–2) | 1.5 | 7 unit test cases, all parallel. CLI integration tests now parallel. Context-cancel test has dependency comment. Missing: test for marker-absent + partial-rules case (code correct but unverified by explicit test). Pre-existing 8 tests in modified file lack `t.Parallel()` — not introduced by this diff but inconsistent with CLAUDE.md. |
| Template Safety (0–1) | 1.0 | `GitignoreRules` and `GitignoreMarker` are hardcoded constants; no user input reaches file content. `filepath.Join(root, ".gitignore")` — no traversal risk. File mode `0644` — standard for source-controlled text files. |

**Total Score: 11.5/12**
**Minimum passing score: 10/12**

No Critical or High findings — cap constraints do not apply. Score of 11.5 reflects a clean implementation with all Medium issues from the prior review resolved. The remaining Low (pre-existing missing `t.Parallel()`) and Nits are not blocking.
