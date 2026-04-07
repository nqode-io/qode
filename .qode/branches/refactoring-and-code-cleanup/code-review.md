# Code Review — qode / refactoring-and-code-cleanup

## Incident Report (written before reading the diff)

The production incident: a user runs `qode knowledge add-branch feature/payments`. The command silently builds a lesson-extraction prompt using the *current* branch as both the template `Branch` field and the directory to read from — not the named branch. The AI extracts lessons from the wrong context, the user saves them to the knowledge base, and future sessions are seeded with incorrect artifacts. The root cause was a refactoring that introduced a `currentBranch string` parameter to `buildBranchLessonData`, replacing the correct `branches[0]` reference. Silent wrong output with no error.

Confirmed in the diff: the original commit introduced `currentBranch` and the subsequent fix commit restored `branches[0]`.

---

## File-by-File Analysis

### `internal/iokit/iokit.go`

- **Verified safe**: `AtomicWrite` correctly calls `os.Chmod(tmp.Name(), perm)` before `os.Rename`. The `perm` parameter is now honoured. Confirmed by `TestAtomicWrite_VerifyPermissions`.
- **Verified safe**: `WriteFile` creates parent dirs before writing. `EnsureDir` is idempotent.
- **Concern (flagged as Nit)**: `ReadFileOrString` silently ignores ALL read errors, not just `os.IsNotExist`. A permission-denied error on an existing file returns `defaultVal` with no signal to the caller. Acceptable for optional context files but the contract should be documented.

### `internal/branchcontext/context.go`

- **Verified safe**: `ContextDir` is set to the branch dir (not the `context/` subdir). All review outputs (diff.md, code-review.md) correctly land in `branchDir = ContextDir`.
- **Verified safe**: Removal of the silent `_ = os.MkdirAll(ctxSubDir, 0755)` from `Load` is correct. Reads from a missing dir return empty strings gracefully. The `EnsureContextDir` export provides creation behaviour when callers need it.
- **Concern (flagged as Medium — see issue #1)**: `runReview` calls `EnsureContextDir` but the files it writes are in the parent dir, not the `context/` subdir.

### `internal/cli/session.go`

- **Verified safe**: `Session.Engine` is typed as `prompt.Renderer` (interface). `*prompt.Engine` satisfies it. Verified in current file.
- **Verified safe**: `loadSession` correctly chains resolveRoot → config.Load → git.CurrentBranch → branchcontext.Load → prompt.NewEngine.

### `internal/config/validate.go`

- **Verified safe**: Validation is called inside `config.Load`. Empty `Scoring.Rubrics` passes (loop does not execute). Zero `MinCodeScore` and `MinSecurityScore` pass (check is `< 0`). Minimal integration-test configs are not broken.
- **Concern (flagged as Low — see issue #3)**: `validRubricKeys` is a static list baked into the binary. An older `qode` binary running against a newer `qode.yaml` that uses a rubric key added in a later version will now fail at startup with `invalid qode config: unknown rubric key "..."` instead of silently ignoring it. This is a forward-compatibility risk.

### `internal/log/log.go`

- **Verified safe**: `var Logger = slog.Default()` ensures no nil-pointer panic if `Init()` is never called (e.g., in test helpers that trigger warning paths). Confirmed in current file.

### `internal/cli/review.go`

- **Verified safe**: `flagStrict` is applied after `loadSession()`, consistent with `runPlanSpec` and `runStart`.
- **Concern (flagged as Medium — see issue #2)**: `EnsureContextDir` call at line 79 creates `.qode/branches/<branch>/context/` but `iokit.WriteFile(diffPath, ...)` at line 84 creates its own parent `.qode/branches/<branch>/` independently. The `EnsureContextDir` call has no protective effect on the write operations that follow it.

### `internal/cli/plan.go`

- **Verified safe**: `runPlanJudge` now applies `flagStrict` — previously missing, now present.
- **Verified safe**: `runPlanRefine` local variable renamed from `out` to `refOut` to avoid collision with the `out io.Writer` parameter. No shadow bug.

### `internal/cli/knowledge_cmd.go`

- **Verified safe**: `buildBranchLessonData` no longer accepts `currentBranch` parameter. Both `branchDir` and `Branch` in `TemplateData` use `branches[0]`. Confirmed in current file (lines 225, 229).

### `internal/cli/integration_test.go`

- **Verified safe**: `t.Cleanup` resets both `flagRoot` and `flagStrict` — confirmed at lines 94–96.
- **Concern (flagged as Medium — see issue #4)**: `assertGolden` helper is defined but never called by any test.
- **Nit**: `runCommand` comment at line 108 says "CLI handlers use fmt.Print" — inaccurate after the refactoring.

### `internal/cli/init_test.go`

- **Verified safe**: `TestRunInitExisting_NoDetectionOutput` captures output via a `bytes.Buffer`, checks forbidden terms are absent, verifies expected content is present, and asserts nothing went directly to `os.Stdout`.

### `internal/scaffold/`

- **Verified safe**: `SetupClaudeCode`, `SetupCursor`, and `Setup` all accept `io.Writer`. Tests updated with `io.Discard`. No `fmt.Printf`/`Println` remaining in production paths.

---

## Issues

### Issue #1 — Medium

**File:** [internal/cli/review.go:79](internal/cli/review.go#L79)  
**Issue:** `branchcontext.EnsureContextDir` is called to create `.qode/branches/<branch>/context/`, but none of the file writes that follow (`diff.md`, `code-review.md`, `.code-review-prompt.md`) use that directory — they all go to `branchDir = sess.Context.ContextDir = .qode/branches/<branch>/`. `iokit.WriteFile` already creates its own parent dirs. The call is not harmful but is semantically misleading: it creates the input directory (for ticket.md etc.), not the output directory. A future reader may incorrectly assume the `EnsureContextDir` call is protecting the writes that follow.  
**Suggestion:** Add a comment explaining the purpose:
```go
// Ensure context/ exists so the user can populate ticket.md etc.
// (not needed for the writes below, which create their own parent dirs)
if err := branchcontext.EnsureContextDir(sess.Root, sess.Branch); err != nil {
```

### Issue #2 — Medium

**File:** [internal/cli/integration_test.go:120](internal/cli/integration_test.go#L120)  
**Issue:** `assertGolden` is defined with a full implementation (reads/writes golden files, compares output) but is never called by any test in the file. All assertions are inline. Dead infrastructure adds maintenance burden — future contributors may not notice it needs updating when outputs change, or may wonder why it exists.  
**Suggestion:** Either add at least one golden test that exercises it (e.g., verify the full rendered output of `plan refine` against a golden file), or delete the function.

### Issue #3 — Low

**File:** [internal/config/validate.go:10](internal/config/validate.go#L10)  
**Issue:** `validRubricKeys` is a static compile-time list. If a newer version of qode adds a rubric key and an older binary is pointed at that project's `qode.yaml`, `config.Load` returns `invalid qode config: unknown rubric key "..."` at startup — blocking all commands. This is a forward-compatibility break in a tool that operates on versioned config files.  
**Suggestion:** Either (a) make unknown keys a `log.Warn` instead of an error, or (b) improve the error message: `(run with a newer qode binary if this key was added in a later version)`.

### Issue #4 — Low

**File:** [internal/cli/integration_test.go:713](internal/cli/integration_test.go#L713)  
**Issue:** `withTicket` and `withRefinedAnalysis` use `os.WriteFile` directly. They work only because `setupProject` pre-creates `ctxDir` via `os.MkdirAll`. If `setupProject` is ever refactored to create dirs lazily, these helpers will break. Minor consistency issue — all production code uses `iokit.WriteFile`.  
**Suggestion:** Replace `os.WriteFile(...)` with `iokit.WriteFile(...)` in both helpers for defensive consistency. Low priority.

### Issue #5 — Nit

**File:** [internal/cli/integration_test.go:108](internal/cli/integration_test.go#L108)  
**Issue:** Comment reads "CLI handlers use fmt.Print which goes to os.Stdout". After the refactoring, handlers use `fmt.Fprint(out, ...)` where `out = os.Stdout`. The behaviour is identical but the comment is inaccurate.  
**Suggestion:** Update to: "CLI handlers write to os.Stdout via the io.Writer parameter, so captureStdout redirects correctly."

### Issue #6 — Nit

**File:** [internal/iokit/iokit.go:10](internal/iokit/iokit.go#L10)  
**Issue:** `ReadFileOrString` returns `defaultVal` for any error, not just file-not-found. The function name implies a "missing file" fallback contract, but a permission-denied error on an existing file returns the same result silently.  
**Suggestion:** Add a doc comment: `// Any error (including permission denied) returns defaultVal.`

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 0 |
| Medium   | 2 |
| Low      | 2 |
| Nit      | 2 |

**Overall assessment:** The refactoring achieves its goals cleanly. The `io.Writer` injection pattern is complete and consistent across all `run*` functions. The `Session` struct eliminates meaningful boilerplate. The `iokit` package is well-tested and correctly used. The 10 issues from the prior review are all fixed and verified in the current source files.

The two Medium issues are not blockers: one is a documentation gap in `EnsureContextDir` placement, the other is dead test infrastructure. Neither affects correctness or the CLI contract.

**Top 3 things to fix before merging:**

1. Add a comment to the `EnsureContextDir` call in `runReview` explaining it creates the input dir (context/) not the output dirs
2. Either add a golden test using `assertGolden` or delete the function
3. Improve the `validRubricKeys` error message to mention the forward-compatibility implication

---

## Rating

| Dimension | Score | What you verified |
|-----------|-------|-------------------|
| Correctness (0–3) | 3 | `buildBranchLessonData` uses `branches[0]` (current file lines 225, 229); `AtomicWrite` calls `os.Chmod` before rename (iokit.go:46); `log.Logger = slog.Default()` (log.go:11); `Session.Engine` is `prompt.Renderer` (session.go:16); `runPlanJudge` applies `flagStrict` |
| CLI Contract (0–2) | 2 | All 6 `run*` functions accept `out, errOut io.Writer`; `flagStrict` applied in `runPlanSpec`, `runPlanJudge`, `runStart`, `runReview`; scaffold functions propagate `io.Writer`; `writePromptToFile` delegates to `iokit.AtomicWrite` |
| Go Idioms & Code Quality (0–2) | 1.5 | `assertGolden` dead code; `EnsureContextDir` call semantically misplaced before unrelated writes; otherwise idiomatic Go — named functions, explicit errors, no global output writes |
| Error Handling & UX (0–2) | 1.5 | `config.Validate` error messages are specific and actionable; `ReadFileOrString` silently swallows all errors (Nit); forward-compat risk in strict rubric-key validation (Low) |
| Test Coverage (0–2) | 1.5 | New iokit tests cover permissions and atomicity; validate tests cover 8 cases; integration tests cover guard-blocked and guard-passed paths; `assertGolden` infra is unused |
| Template Safety (0–1) | 1 | Scaffold templates verified: `{{.Project.Name}}` is the only interpolation point; rendered at `qode init` time from qode.yaml string field; no user-controlled runtime input reaches template engine |

**Total Score: 10.5/12**  
**Minimum passing score: 10/12** ✓
