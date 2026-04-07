# Code Review — qode / refactoring-and-code-cleanup

> **Incident pre-mortem:** The scaffold template migration shipped. A dev on a shared workstation runs `qode plan refine` on a feature branch. A teammate opens `.qode/branches/feat-login/.refine-prompt.md` and reads the confidential ticket spec for the unreleased feature. Nobody noticed: the file was always 0600 before, now it's 0644. The directory went from 0700 to 0755. The content was already there — just newly readable.
>
> Now read the diff. Did that ship?

It did. `writePromptToFile` now calls `iokit.AtomicWrite(path, []byte(content), 0644)`. The old code used `os.CreateTemp` (implicit 0600, no chmod) inside a 0700 directory. The diff file in `runReview` explicitly keeps 0600 — the inconsistency confirms this was unintentional.

---

## Files Reviewed

- `internal/branchcontext/context.go` (renamed from `internal/context/context.go`)
- `internal/cli/session.go` (new), `util.go` (new)
- `internal/cli/branch.go`, `branch_test.go`
- `internal/cli/help.go`, `init.go`, `init_test.go`
- `internal/cli/plan.go`, `plan_test.go`
- `internal/cli/review.go`, `review_test.go`
- `internal/cli/start.go`, `knowledge_cmd.go`
- `internal/cli/integration_test.go` (new), `root.go`
- `internal/config/validate.go` (new), `validate_test.go` (new), `config.go`, `defaults.go`
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

- `loadSession()` callers always need all five fields (root, config, branch, context, engine). No lazy loading — commands that only need config still pay the full session init cost.
- `flagStrict`, `flagRoot`, and `rootCmd` are never accessed concurrently during tests; the cleanup (`t.Cleanup`) restores them sequentially.
- `os.Stdout` in cobra `RunE` closures is evaluated at call time, not captured at function definition. `captureStdout` in integration tests relies on this.
- Prompt files (`.refine-prompt.md`, etc.) need no stronger access control than the project directory they live in.
- `.qode/prompts/scaffold/` is always deleted at the end of `runInitExisting`, so the local-override-first strategy in the prompt engine never picks them up on subsequent runs.

**Verified as safe:**

- `branchcontext.Load()` uses `iokit.ReadFileOrString` throughout — safe when the context dir or its files don't exist.
- `ParseIterationFromOutput` previously silenced write errors (`_ = os.WriteFile`); now they propagate. Correct.
- `runReview` explicitly calls `EnsureContextDir` after the diff is fetched to handle the no-`branch-create` path. The comment explains why.
- Integration test cleanup resets `flagRoot`, `flagStrict`, and `rootCmd.SetArgs(nil)`.
- `runReview` switch has a `default` case returning an explicit error for unknown kind.
- `stubName` (not `name`) is used in the stubs loop — no shadowing of the branch name parameter.
- All `plan_test.go` tests that were using `captureStdout` now use `bytes.Buffer` or `io.Discard`.
- `EnsureDir` wraps the `os.MkdirAll` error with the path: `fmt.Errorf("ensure dir %s: %w", path, err)`.
- `AtomicWrite` sequences correctly: write → close → chmod → rename; defer removes the temp on failure after a successful rename the Remove is a no-op (file moved), error silently swallowed, which is correct.
- Scaffold `.IDE` field is always set to `"claude"` or `"cursor"` by the callers before rendering.

---

## Issues

---

### Medium

#### 1. Prompt file permissions regressed from 0600 → 0644 and directory from 0700 → 0755

- **Severity:** Medium
- **File:** [internal/iokit/iokit.go:73-93](internal/iokit/iokit.go), [internal/cli/util.go:22-26](internal/cli/util.go)
- **Issue:** `writePromptToFile` now calls `iokit.AtomicWrite(path, []byte(content), 0644)`. `AtomicWrite` calls `EnsureDir` (0755) before `os.CreateTemp` + `os.Chmod(tmp.Name(), 0644)` + `Rename`. The old `writePromptToFile` created the directory at 0700 and relied on `os.CreateTemp`'s implicit 0600 for the file, so final files were 0600 in a 0700 directory. Prompt files contain rendered ticket content, specs, and code snippets. The diff file in `runReview` explicitly keeps `0600` (`iokit.WriteFile(diffPath, []byte(diff), 0600)`), confirming the author was aware of permissions — making the prompt file change appear unintentional.
- **Suggestion:** Either change `writePromptToFile` to use `0600`, or introduce a separate `AtomicWritePrivate` that applies 0600 and uses 0700 for the directory:

  ```go
  func writePromptToFile(path, content string) error {
      return iokit.AtomicWrite(path, []byte(content), 0600)
  }
  ```

  And in `AtomicWrite` (or a new variant), preserve the directory permission that existed before:

  ```go
  if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil { ... }
  ```

---

#### 2. `integration_test.go:runCommand` captures output via a fragile `os.Stdout` replacement

- **Severity:** Medium
- **File:** [internal/cli/integration_test.go:1062-1070](internal/cli/integration_test.go)
- **Issue:** `runCommand` calls `captureStdout(t, func() { rootCmd.Execute() })`. The cobra `RunE` handlers pass `os.Stdout` as the writer: `return runPlanRefine(os.Stdout, os.Stderr, ...)`. This only works because `os.Stdout` is the global `*os.File` variable evaluated at the time `RunE` executes inside `captureStdout`'s redirection scope. The comment says "CLI handlers write to os.Stdout via the io.Writer parameter, so captureStdout redirects correctly" — this misattributes the mechanism. The redirection works because `os.Stdout` is lazily evaluated at call time, not because of the `io.Writer` parameter. If any future handler stores `os.Stdout` in a local variable before `captureStdout` redirects it (e.g., middleware), the capture silently produces empty output. The test `if out == ""` guard would catch the breakage, but the root cause would be non-obvious.
- **Suggestion:** Fix the comment, and consider passing a writer through cobra's `SetOut`:

  ```go
  // captureStdout works here because os.Stdout is evaluated inside captureStdout's
  // redirection scope, not at cobra registration time. This is fragile.
  ```

  Long-term, cobra supports `cmd.SetOut(writer)` and `cmd.OutOrStdout()` — using that would make capture deterministic.

---

### Low

#### 3. No compile-time interface guard for `*Engine` implementing `Renderer`

- **Severity:** Low
- **File:** [internal/prompt/renderer.go](internal/prompt/renderer.go)
- **Issue:** `Renderer` is the central interface used across `plan`, `review`, `scaffold`, and `knowledge` packages. If `Engine` ever drops or renames `Render` or `ProjectName`, the compiler will not catch it until something passes `*Engine` where `Renderer` is expected. In a test-heavy codebase with mock implementations, interface drift is a real risk.
- **Suggestion:** Add to `renderer.go` or `engine.go`:

  ```go
  var _ Renderer = (*Engine)(nil)
  ```

---

#### 4. `internal/log` package has no tests

- **Severity:** Low
- **File:** [internal/log/log.go](internal/log/log.go)
- **Issue:** `Init()` parses `QODE_LOG_LEVEL` and sets the package-level `logger`. Invalid values (e.g., `"verbose"`, `"trace"`) silently fall through to the default `slog.LevelInfo`. There are no tests asserting: (a) valid log level strings produce the correct handler level; (b) invalid strings fall back to INFO without panicking; (c) uninitialized logger (before `Init()`) works safely. Functions calling `log.Warn` in tests (e.g., `buildBranchLessonData` when a branch context is missing) will write to the default `slog` handler, producing noisy test output.
- **Suggestion:** Add a `TestInit` in `log_test.go` that sets `QODE_LOG_LEVEL` and verifies the handler level. Consider accepting the level as a parameter to `Init` to make it testable without env var mutation.

---

#### 5. `TestRunBranchCreate_OutputMentionsBranchName` has a dead assignment

- **Severity:** Low
- **File:** [internal/cli/branch_test.go:532-533](internal/cli/branch_test.go)
- **Issue:** `root := setupBranchTestRoot(t)` followed immediately by `_ = root`. The return value is unused; the side effect (setting `flagRoot`) is why `setupBranchTestRoot` is called. This is a misleading pattern: a reader wonders why `root` was captured. It also means the test doesn't assert that the output contains a path relative to `root` — a stronger check that was available.
- **Suggestion:** Either remove the return value capture entirely if the side effect is all that's needed, or use `root` in a path assertion:

  ```go
  root := setupBranchTestRoot(t)
  // ... after runBranchCreate:
  if !strings.Contains(out, root) {
      t.Errorf("expected output to contain root path, got: %q", out)
  }
  ```

---

### Nit

#### 6. Integration test helper `withQodeYAML` bypasses `iokit.WriteFile`

- **Severity:** Nit
- **File:** [internal/cli/integration_test.go:998-1002](internal/cli/integration_test.go)
- **Issue:** `withQodeYAML` uses `os.WriteFile` directly while all production code and the other helpers (`withTicket`, `withRefinedAnalysis`) use `iokit.WriteFile`. The inconsistency is harmless here (parent dir is always a `t.TempDir()` that exists) but breaks the convention established by the refactoring.
- **Suggestion:**

  ```go
  if err := iokit.WriteFile(filepath.Join(root, "qode.yaml"), []byte(content), 0644); err != nil {
  ```

#### 7. `captureStdout` mechanism deserves a doc comment in the integration test

- **Severity:** Nit
- **File:** [internal/cli/integration_test.go:1058-1070](internal/cli/integration_test.go)
- **Issue:** The comment "CLI handlers write to os.Stdout via the io.Writer parameter, so captureStdout redirects correctly" is inaccurate (see Medium #2). Without a correct explanation, the next developer maintaining `runCommand` will either perpetuate the misunderstanding or break the capture.
- **Suggestion:** Replace the comment with a precise one:

  ```go
  // captureStdout replaces os.Stdout before Execute() is called. Each RunE
  // closure passes os.Stdout (evaluated at call time, after replacement) to
  // its run function, so output is captured correctly.
  ```

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 2 |
| Low | 3 |
| Nit | 2 |

**Overall assessment:** This is a disciplined refactoring. The `internal/context` → `internal/branchcontext` rename eliminates a real stdlib collision hazard. The `io.Writer` injection is applied consistently across all six command handlers and their tests. The `Session` struct removes identical five-line boilerplate from every command. `iokit` consolidates repeated OS primitives correctly — `AtomicWrite` handles the failure path and temp cleanup properly. `config.Validate()` surfaces bad YAML at load time with field-level error messages and tests covering all paths. `ParseIterationFromOutput` now propagates write errors that were previously silenced. The scaffold template extraction from Go string literals to `.md.tmpl` files eliminates duplicated content between Cursor and Claude Code generators.

The permissions regression (Medium #1) is the most important fix before merging — prompt files contain project-sensitive content and the intentional 0700/0600 posture from the old code was deliberately protective. The integration test stdout capture assumption (Medium #2) is correct today but worth a comment and a longer-term fix.

**Top 3 most important things to fix before merging:**

1. **Medium #1 — Restore 0600/0700 permissions for prompt files** — prompt content (tickets, specs, AI analysis) changed from owner-only to world-readable. The diff file in `runReview` explicitly uses 0600, confirming the intent was to keep these private.
2. **Medium #2 — Fix the misleading comment in `runCommand`** — the explanation of why `captureStdout` works is incorrect and will mislead future maintainers.
3. **Low #3 — Add compile-time interface guard for `*Engine`** — `Renderer` is the central interface; interface drift will only surface at use-site, not at the interface definition.

---

## Rating

| Dimension | Score | What you verified (not what you assumed) |
|-----------|-------|------------------------------------------|
| Correctness (0–3) | 2.8 | Traced `branchcontext.Load()` through all callers — safe on missing dirs via `ReadFileOrString`; confirmed `EnsureContextDir` is called in `runReview` for non-canonical setups; verified `AtomicWrite` sequences (write → close → chmod → rename, defer cleanup is a no-op after successful rename); confirmed `ParseIterationFromOutput` now propagates errors; confirmed `runReview` switch has default. Deduction: prompt file permissions regression (0600 → 0644) is a behavioral change not covered by any test or comment |
| CLI Contract (0–2) | 2.0 | Verified stdout → `out` writer, stderr → `errOut` writer across plan/review/start/knowledge/branch; `STOP.` messages go to `out`; `--strict` is a PersistentFlag applied after `loadSession()` (in-memory only, no re-validation); `--force` bypass preserved in plan spec, start, review; `EnsureContextDir` called before diff write in review |
| Go Idioms & Code Quality (0–2) | 1.8 | `Renderer` interface is idiomatic; `Session` struct eliminates boilerplate; `iokit` functions correctly sized; `EnsureDir` wraps errors with path; `branchcontext` naming avoids stdlib collision; `stubName` used correctly (no shadowing). Deductions: no compile-time `*Engine` implements `Renderer` guard; `_ = root` dead assignment in one test |
| Error Handling & UX (0–2) | 1.8 | `ParseIterationFromOutput` write errors now propagate; `config.Validate()` surfaces bad YAML at load with all violations collected; `EnsureDir` path-wrapped errors; `runReview` unknown kind returns explicit error; `runKnowledgeAddBranch` logs warning on missing branch context and continues. Deduction: permission change on prompt files has no test and no comment explaining the intentionality |
| Test Coverage (0–2) | 1.7 | `iokit` tests cover all cases including idempotency, permissions, and temp-file cleanup; `config.Validate` tests cover every validation path and multi-error accumulation; `branch_test.go` adds unit tests for extracted `runBranchCreate` / `runBranchRemove`; integration tests exercise full cobra command path with build tag. Gaps: `log` package entirely untested; `runWorkflow` extracted function untested; integration test's `captureStdout` relies on a fragile assumption |
| Template Safety (0–1) | 1.0 | `.IDE` is always `"claude"` or `"cursor"` set by callers before rendering; conditionals are simple equality checks; no user-controlled data reaches template execution; scaffold templates are well-formed Go templates; `{{if eq .IDE "cursor"}}` / `{{else}}` branches verified to produce correct output for both IDE values |

**Total Score: 11.1/12**
**Minimum passing score: 10/12** ✅

What makes this better than most shipped refactors: the `io.Writer` injection is applied completely and consistently — every function that emits output was updated, and the corresponding tests were updated to use `bytes.Buffer` rather than OS-level stdout capture. The `iokit.AtomicWrite` failure path is correct (the deferred `Remove` is a no-op after rename, not a resource leak). `config.Validate()` closes a gap where malformed rubric configs would silently produce 0/0 scores at review time. The `Renderer` interface decouples plan, review, and scaffold packages from the concrete engine, enabling isolated unit tests without embedded template setup.
