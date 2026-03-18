# Code Review — remove-prompt-only-flag

**Branch:** remove-prompt-only-flag
**Reviewer:** Claude (qode-review-code)
**Date:** 2026-03-18
**Score:** 9.0/10

---

## Summary

This PR removes the `--prompt-only` flag and interactive dispatch from all prompt-generating commands, replacing the default with stdout output and adding a `--to-file` debug flag. The `internal/dispatch/` package and VSCode IDE support are deleted entirely. The change is clean, well-scoped, and consistent across all affected commands.

All tests pass. Build is clean. `go vet` reports no issues.

---

## Critical Issues

None.

---

## High Issues

None.

---

## Medium Issues

### M1 — TODO comments in committed code violate CONTRIBUTING.md policy

**Location:** [internal/cli/ide.go](internal/cli/ide.go) — two TODO comments added above `ide.Setup()` calls

**Issue:** CONTRIBUTING.md explicitly states "No TODO comments in committed code." Two `// TODO: add --force flag before beta...` comments were added. While the intent is tracked, these will remain indefinitely unless enforced.

**Fix:** Either open a GitHub issue and reference it (`// TODO(#42): add --force flag...`) or remove the comments and track via the issue tracker. If left in, the issue number should be added to make intent discoverable.

---

## Low Issues

### L1 — `writePromptToFile` comment is misleading

**Location:** [internal/cli/root.go:91](internal/cli/root.go#L91)

**Issue:** The comment says "On template render error the caller should return before calling this." This describes a caller convention but is not relevant to what `writePromptToFile` does. The function is a generic atomic writer; the comment conflates concerns.

**Fix:** Remove or simplify the second sentence. The function doc can just be:
```go
// writePromptToFile atomically writes content to path, creating parent dirs as needed.
```

---

### L2 — `qode review all` removed without deprecation notice in output

**Location:** [internal/cli/review.go](internal/cli/review.go)

**Issue:** `newReviewAllCmd()` is deleted. Users who have scripts or muscle memory using `qode review all` will receive an "unknown command" error with no helpful hint. The README was updated, but there is no runtime guidance.

**Fix:** Consider adding `review all` back as a thin wrapper that prints a deprecation message to stderr and runs both reviews. Alternatively, the removal is acceptable since `review all` was just a shorthand — just document it clearly in the release notes.

---

### L3 — No unit test for `writePromptToFile`

**Location:** [internal/cli/root.go](internal/cli/root.go)

**Issue:** `writePromptToFile` is a non-trivial helper (atomic write via temp + rename) but has no test. The `internal/cli` package has no test files at all.

**Fix:** Add a test for `writePromptToFile` in `internal/cli/root_test.go` covering the success path, directory creation, and cleanup of temp file on write error. Not blocking, but the atomicity invariant is worth asserting.

---

## Positive Observations

- **Consistent pattern across all commands.** Every command follows the same stdout-first, `--to-file`-for-debug pattern. No exceptions or special cases.
- **Atomic file write is correct.** `defer os.Remove(tmpName)` safely handles cleanup on error; on success `os.Rename` moves the file and the deferred remove is a no-op (silent ENOENT).
- **Stderr/stdout discipline is solid.** All informational messages (banners, paths, warnings) go to stderr; all prompt content goes to stdout. This makes `qode review code 2>/dev/null | claude` work correctly.
- **Package deletion is clean.** `internal/dispatch/` is fully removed with no dangling imports or references.
- **Tests updated appropriately.** VSCode tests removed, `NoPromptOnly` tests added, `minimalConfig()` cleaned up.
- **`specPath` still correctly used as a template variable.** Passing `specPath` to `BuildSpecPromptWithOutput` so the AI knows where to save the spec is the right design — the function doesn't write there, just embeds the path in the prompt.
- **`branchDir` left in `runReview`.** `outputPath` is passed to the review prompt builder as a template variable (the AI is instructed to write there), so `branchDir` still has purpose even though `MkdirAll` moved into `writePromptToFile`.

---

## Verdict

The implementation is correct and complete. The only policy violation (M1 TODO comments) was explicitly requested in the spec; if that was intentional, consider adding issue references. L2 and L3 are minor quality improvements that can be addressed post-merge.
