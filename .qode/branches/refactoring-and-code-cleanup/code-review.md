# Code Review — qode / refactoring-and-code-cleanup

> **Incident pre-mortem:** The scaffold templates shipped. On the next `qode init`, 1,500 teams regenerated their IDE slash commands — and every single one came out blank, because the template engine resolved `.IDE` to an empty string and both branches of every `{{if eq .IDE "cursor"}}` block produced nothing useful. The Cursor commands had no frontmatter; the Claude commands had no heading. Developers assumed qode was broken and filed issues. The fix was a one-liner in `SetupClaudeCode`/`SetupCursor`, but nobody caught it because the tests only checked that files were written, not that their content was IDE-correct.
>
> Now read the diff. Did that ship?

It did not — `data.IDE` is set to `"claude"` or `"cursor"` explicitly before the loop in each Setup function. The pre-mortem scenario is ruled out. The review documents what was actually found.

---

## Files Reviewed

- `internal/branchcontext/context.go` (renamed from `internal/context/context.go`)
- `internal/cli/session.go` (new), `util.go` (new)
- `internal/cli/branch.go`, `help.go`, `init.go`, `plan.go`, `review.go`, `start.go`, `knowledge_cmd.go`
- `internal/cli/integration_test.go` (new), `init_test.go`, `plan_test.go`, `review_test.go`
- `internal/config/validate.go` (new), `validate_test.go` (new)
- `internal/iokit/iokit.go` (new), `iokit_test.go` (new)
- `internal/log/log.go` (new)
- `internal/prompt/renderer.go` (new), `engine.go`
- `internal/scaffold/claudecode.go`, `cursor.go`, `scaffold.go`, `scaffold_test.go`
- `internal/plan/refine.go`, `refine_test.go`
- `internal/review/review.go`, `review_test.go`
- `internal/prompt/templates/scaffold/*.md.tmpl` (all new)
- `.claude/commands/*.md`, `.cursor/commands/*.mdc`, `cmd/qode/main.go`

---

## Assumptions, Unverified Invariants, and Silent Failure Points

**What the code assumes:**
- `flagRoot`, `flagStrict`, and `rootCmd` are never accessed concurrently during integration tests.
- `loadSession()` callers always need branch, context, and engine (eager loading).
- The `kind` parameter to `runReview` is always "code" or "security".
- `os.CreateTemp` + `Chmod` + `Rename` is atomic enough for prompt and diff files.
- After `qode init`, users will re-run `qode init` to pick up updated scaffold commands.

**Where it can fail silently:**
- `runReview`'s `switch kind` has no default: unknown kind writes an empty review file with no error.
- `EnsureDir` returns `os.MkdirAll` errors without explicit path wrapping (the error string from `os.MkdirAll` does include the path, so diagnostic value is retained, but callers that don't wrap may lose context).
- `runKnowledgeAddBranch` logs a `log.Warn` when a branch context fails to load, then continues — the resulting prompt could be incomplete with no indication to the caller.

---

## Issues

---

### Medium

**1. Integration tests mutate global state without complete cleanup**

- **Severity:** Medium
- **File:** [internal/cli/integration_test.go](internal/cli/integration_test.go#L895-L898)
- **Issue:** `flagRoot`, `flagStrict`, and `rootCmd.SetArgs()` are package-level state mutated per test. The `t.Cleanup` restores `flagRoot` and `flagStrict` but does **not** reset `rootCmd.Args`. After each `rootCmd.Execute()`, the cobra command retains the args from the previous `SetArgs(args)` call. If a test fails between `SetArgs` and `Execute`, the next test inherits stale args.
- **Suggestion:**
  ```go
  t.Cleanup(func() {
      flagRoot = ""
      flagStrict = false
      rootCmd.SetArgs(nil)  // add this
  })
  ```

---

**2. `plan_test.go` partially bypasses the `io.Writer` refactoring**

- **Severity:** Medium
- **File:** [internal/cli/plan_test.go](internal/cli/plan_test.go#L1549-L1587)
- **Issue:** Several `runPlanSpec` tests still pass `os.Stdout`/`os.Stderr` and use `captureStdout()` (OS-level fd redirection). The `init_test.go` tests were properly updated to pass `bytes.Buffer`. These are inconsistent. The `captureStdout` path is fragile: if the function writes to the injected `io.Writer` instead of `os.Stdout` (as it now does), the capture would miss output but the test would still pass. This was verified: `runPlanSpec(os.Stdout, ...)` + `captureStdout` works only because the OS stdout fd is redirected — it's fragile and defeats the purpose of the refactoring.
- **Suggestion:** Update plan tests to use `bytes.Buffer` directly:
  ```go
  var buf bytes.Buffer
  err := runPlanSpec(&buf, io.Discard, false, false)
  output := buf.String()
  ```

---

**3. `context.Load()` no longer creates the context subdirectory**

- **Severity:** Medium
- **File:** [internal/branchcontext/context.go](internal/branchcontext/context.go)
- **Issue:** The old `Load()` called `os.MkdirAll(ctxSubDir, 0755)` as a side effect, ensuring the context dir always existed. The new code removes this; `EnsureContextDir` is only called from `runReview`. Commands `plan refine`, `plan spec`, `plan judge`, `start`, and `knowledge add-branch` no longer guarantee the context subdirectory exists. File reads are safe (handled by `ReadFileOrString`). The impact is on non-canonical setups where the user manually initialises a branch directory without running `branch create` or `review`: the `context/` subdirectory and its stubs (`ticket.md`, `notes.md`) are never created.
- **What was verified:** `branch create` still calls `iokit.EnsureDir(contextDir)` and writes stubs. The canonical path is safe.
- **Suggestion:** Add a comment in `runReview` explaining the explicit call to `EnsureContextDir`, and document in `Load()` that it no longer creates directories.

---

**4. `EnsureDir` errors lack explicit path wrapping**

- **Severity:** Medium
- **File:** [internal/iokit/iokit.go](internal/iokit/iokit.go#L418-L420)
- **Issue:** `EnsureDir` returns `os.MkdirAll`'s error directly. `os.MkdirAll` does include the path in its error string (e.g., `mkdir /path: permission denied`), so diagnostic value is preserved in the happy-path error chain. However, `EnsureDir` is public, and callers that don't add their own wrapping will produce errors without layered context. Callers inside `WriteFile` add wrapping ("write %s: %w"), but the dir-creation failure inside `WriteFile` loses its path context since `EnsureDir`'s error is returned unwrapped.
- **Suggestion:**
  ```go
  func EnsureDir(path string) error {
      if err := os.MkdirAll(path, 0755); err != nil {
          return fmt.Errorf("ensure dir %s: %w", path, err)
      }
      return nil
  }
  ```

---

### Low

**5. `runReview` silently no-ops on unknown `kind`**

- **Severity:** Low
- **File:** [internal/cli/review.go](internal/cli/review.go#L1695-L1702)
- **Issue:** The `switch kind` has no `default` branch. If `kind` is neither "code" nor "security", `p` remains `""`, no error is returned, and an empty file is written. Both callers pass hardcoded strings so this cannot happen today. But the function accepts a string parameter and the implicit contract is invisible.
- **Suggestion:**
  ```go
  default:
      return fmt.Errorf("unknown review kind %q", kind)
  ```

---

**6. Variable shadowing: `name` in `runBranchCreate`**

- **Severity:** Low
- **File:** [internal/cli/branch.go](internal/cli/branch.go#L367)
- **Issue:** `for name, content := range stubs` shadows the function parameter `name` (branch name). No bug exists because `branchDir`/`contextDir` are resolved before the loop. But a future reader could be misled.
- **Suggestion:**
  ```go
  for stubName, content := range stubs {
      p := filepath.Join(contextDir, stubName)
  ```

---

**7. No unit tests for `runBranchCreate` / `runBranchRemove`**

- **Severity:** Low
- **File:** [internal/cli/branch.go](internal/cli/branch.go)
- **Issue:** Both functions were extracted from cobra closures to enable unit testing — the primary motivation for the `io.Writer` refactoring. Neither has a corresponding test. The `keepCtx` logic in `runBranchRemove` (merging flag with config) is particularly worth covering.
- **Suggestion:** Add tests using `io.Discard` and `t.TempDir()`:
  ```go
  func TestRunBranchRemove_KeepCtxFromConfig(t *testing.T) { ... }
  func TestRunBranchRemove_KeepCtxFromFlag(t *testing.T) { ... }
  ```

---

**8. `log.Logger` is a mutable exported var — no documented thread-safety**

- **Severity:** Low
- **File:** [internal/log/log.go](internal/log/log.go#L13-L14)
- **Issue:** `Logger` is exported and reassigned in `Init()`. In practice, `Init()` is called once at program start before goroutines, so there is no race. But the exported var can be accessed externally (e.g., from tests calling log functions before/after Init). No synchronisation or documentation exists.
- **Suggestion:** Make `Logger` unexported and expose only the package-level helper functions, or document the single-init constraint explicitly.

---

### Nits

**9. Scaffold templates are copied then immediately deleted — worth a comment**

- **Severity:** Nit
- **File:** [internal/cli/init.go](internal/cli/init.go#L651-L657)
- **Issue:** `copyEmbeddedTemplates` writes scaffold templates to `.qode/prompts/scaffold/`, `scaffold.Setup` renders them, then they are deleted via `os.RemoveAll`. The round-trip is necessary (local-override-first strategy) but non-obvious.
- **Suggestion:** Add an inline comment explaining why deletion is correct.

**10. `TestSetupClaudeCode_WritesTicketFetchCommand` checks for "Figma" — fragile sentinel**

- **Severity:** Nit
- **File:** [internal/scaffold/scaffold_test.go](internal/scaffold/scaffold_test.go#L3813-L3815)
- **Issue:** The test asserts `"Figma"` is in the rendered ticket-fetch command. This string lives in the embedded template, not in test code. If the template reference to Figma is updated, the test breaks with an opaque assertion failure rather than a meaningful test name.
- **Suggestion:** Either remove the "Figma" check (structural checks like `"MCP"` and `"$ARGUMENTS"` are sufficient) or add an inline comment explaining the purpose.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 4 |
| Low | 4 |
| Nit | 2 |

**Overall assessment:** The refactoring achieves its goals cleanly. The `internal/context` → `internal/branchcontext` rename eliminates stdlib shadowing. The `io.Writer` injection makes functions unit-testable. The `iokit` package consolidates repeated OS primitives correctly. The `Session` struct removes copy-paste boilerplate across six command handlers. The `config.Validate()` addition is well-tested and catches bad YAML at load time. The scaffold template extraction from Go string literals to `.md.tmpl` files eliminates duplicate content maintenance across two IDE generators.

The plan_test.go inconsistency (Medium #2) is the most important corrective action: the refactoring's testability goal was to avoid OS-level stdout capture, and the plan tests still use it. The integration test global state (Medium #1) is a latent brittleness worth addressing before the test suite grows.

**Top 3 most important things to fix before merging:**

1. **Medium #2 — Update `plan_test.go` to use `bytes.Buffer`** — the `io.Writer` injection was the primary stated goal; leaving these tests on `captureStdout` defeats the purpose.
2. **Medium #1 — Add `rootCmd.SetArgs(nil)` to integration test cleanup** — the global `rootCmd` state is not fully restored between tests.
3. **Low #7 — Add unit tests for `runBranchCreate` / `runBranchRemove`** — these were extracted specifically to enable testing; adding coverage completes the refactoring's intent.

---

## Rating

| Dimension | Score | What you verified (not what you assumed) |
|-----------|-------|------------------------------------------|
| Correctness (0–3) | 3 | Traced `context.Load()` dir-creation removal through all callers; confirmed `branch create` still creates context dir; confirmed `iokit.WriteFile` creates parent dirs via `EnsureDir`; verified `AtomicWrite` temp-file lifecycle (chmod-before-rename, cleanup-on-failure deferred); confirmed `kind` fallthrough cannot happen with current callers; verified `flagStrict` mutation only affects in-memory Config |
| CLI Contract (0–2) | 2 | Confirmed prompts go to the `out` writer (stdout), status messages to `errOut` (stderr); confirmed `--strict` is PersistentFlag applied correctly in plan/review/start; confirmed `--force` bypass paths are preserved in all three commands; confirmed `STOP.` output goes to `out` not `errOut` |
| Go Idioms & Code Quality (0–2) | 1.5 | `Renderer` interface is idiomatic; `Session` struct is clean; `branchcontext` naming is unambiguous; `iokit` functions are correctly sized; variable shadowing of `name` in branch loop is a real smell; `EnsureDir` lacks defensive error wrapping; `runReview` switch has no default |
| Error Handling & UX (0–2) | 1.5 | Previously swallowed errors in `ParseIterationFromOutput` now propagate correctly; `config.Validate()` surfaces bad YAML at load time with field-level messages; `EnsureDir` errors lack layered path context; `runReview` unknown kind silently writes empty file |
| Test Coverage (0–2) | 1.5 | `iokit` tests are thorough (7 cases including idempotency and permissions); `config.Validate` tests cover all error paths including multi-error accumulation; integration tests exercise full cobra command path; plan tests inconsistently bypass io.Writer injection; `runBranchCreate`/`runBranchRemove` have no unit tests |
| Template Safety (0–1) | 1 | Templates use `text/template` (correct for markdown); conditionals are simple equality checks on a controlled field; no user-controlled data reaches template execution; `.IDE` is always "claude" or "cursor" as set by callers |

**Total Score: 10.5/12**
**Minimum passing score: 10/12** ✅

What makes this better than most shipped refactors: the `io.Writer` injection is disciplined — every function that writes output has been updated. `iokit.AtomicWrite` is correct (chmod before rename, defer cleanup-on-failure). The scaffold template extraction eliminates duplicate content maintenance with zero complexity added to the render path. `config.Validate()` closes a gap where malformed rubric configs would silently produce zero scores at review time. The `Renderer` interface allows plan/review packages to be tested without a full engine setup.
