<!-- qode:iteration=1 score=23/25 -->

# Requirements Refinement — BUG: Commands run in terminal are not interactive

## 1. Problem Understanding

**Restated Problem:**
When `qode` commands dispatch to the Claude CLI (e.g., `qode review code`, `qode start`, `qode plan refine`), they invoke `claude` with `--print` flag and capture its stdout into a buffer. This means Claude runs as a silent background process — the user cannot see its output in real time, cannot interact with it, and the terminal hangs with only a 5-second progress dot ticker. The expected behavior is that `claude` should take over the terminal as a foreground interactive process, with its stdin/stdout/stderr wired directly to the user's terminal.

**User Need:**
Developers using `qode` expect the same interactive experience as running `claude` directly. They want to see output as it streams, respond to prompts if needed, and get a proper exit when Claude finishes.

**Business Value:**
Without interactive mode, the tool feels broken and opaque. Users cannot observe what Claude is doing, cannot abort cleanly, and get no feedback until a timeout or completion. This directly undermines the core workflow.

**Open Questions:**
- None. The ticket is clear: replace background execution with foreground/interactive process handoff.

---

## 2. Technical Analysis

### Affected Components

**Primary: `internal/dispatch/claude.go`**
The `claudeCLI.Run()` method (lines 47–83) is the root cause:
```go
cmd := exec.CommandContext(ctx, c.binaryPath,
    "--print",          // ← Non-interactive batch mode
    "--allowedTools", "Read,Write,Glob,Grep",
    "--model", "sonnet",
    "--output-format", "text",
)
cmd.Stdin = strings.NewReader(prompt)
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout   // ← Capturing to buffer, not terminal
cmd.Stderr = &stderr
```

**Secondary: `internal/cli/review.go`**
- `reviewDispatch()` (lines 145–202): calls `d.Run()` then reads the score from the output file. With interactive mode, Claude writes the review file itself; this function just needs to wait for the process and then read the score.
- `newReviewAllCmd()` (lines ~58–71): currently calls `runReview("code", ...)` then `runReview("security", ...)` sequentially — this is already correct for sequential behavior, but both calls go through the non-interactive dispatcher.

**Tertiary: `internal/dispatch/dispatch.go`**
The `Dispatcher` interface returns `(string, error)`. For interactive mode, there is no meaningful string to return — Claude writes its output to files directly. A new interface method or a mode flag is needed.

### Key Technical Decisions

1. **New interactive run method vs. new dispatcher:** The cleanest approach is to add a new `RunInteractive(ctx, prompt, opts)` method to the `claudeCLI` struct (not on the `Dispatcher` interface, since `clipboardDispatcher` has no interactive analog). A package-level function `RunInteractive()` can wrap this.

2. **How to pass the prompt:** Currently passed via stdin (`cmd.Stdin = strings.NewReader(prompt)`). For interactive mode, stdin must be wired to `os.Stdin`. The prompt must instead be passed as a file or via the `--resume` / `--file` flag (claude supports `--file <path>`). The natural approach: write the prompt to the existing `.refine-prompt.md` or review prompt file (already done upstream), then pass it via `--file` or via a temporary file. Since the prompt files are already written before dispatch, we can use `--file <promptPath>` or pass it as a positional argument.

   **Preferred approach:** Pass the prompt file path via `--file` argument so stdin remains available for terminal interaction. If `claude` does not support `--file`, write to a temp file and use `--resume`. Investigation needed: check `claude --help` for supported flags.

   Actually, looking at current code — the prompt is passed via `cmd.Stdin = strings.NewReader(prompt)`. For interactive mode, we need stdin for the terminal. We can write the prompt to a temp file and use `claude --file /tmp/prompt.md` or use the `-p` flag only in non-interactive mode. For interactive mode: `claude` (without `--print`) reads the prompt from a file or we can pass it via `--file`.

3. **Removing `--print` and `--output-format`:** These flags activate non-interactive batch mode. Interactive mode should omit them. The `--allowedTools` and `--model` flags remain relevant.

4. **Wiring stdio:** Set `cmd.Stdin = os.Stdin`, `cmd.Stdout = os.Stdout`, `cmd.Stderr = os.Stderr`.

5. **Return value:** `RunInteractive` returns only `error` (no string output — Claude writes files directly).

6. **Context/timeout:** Interactive sessions should not have a hard timeout. Pass `context.Background()` without a deadline, and rely on `cmd.Wait()` for natural termination.

7. **`CLAUDECODE` env filter:** Keep `filterEnv(os.Environ(), "CLAUDECODE")` to prevent nested session detection.

### Patterns to Follow

- Follow existing `exec.CommandContext` usage in `internal/dispatch/claude.go`
- Keep the `Dispatcher` interface unchanged; add `RunInteractive` only to `claudeCLI`
- Expose a package-level `RunInteractive(ctx, prompt, opts)` function similar to `Resolve().Run(...)`
- Review commands should call `dispatch.RunInteractive(...)` instead of `d.Run(...)`

### Dependencies

- `claude` CLI binary must be available (already checked via `claudeCLI.Available()`)
- Prompt files are already written before dispatch, so `--file` can reference them
- No new external dependencies

---

## 3. Risk & Edge Cases

### What Could Go Wrong

1. **`claude` flag for file input:** If `claude` does not support `--file`, we need an alternative. Mitigation: verify via `claude --help`; fallback is using a temp file passed as positional argument or via heredoc via a shell wrapper.

2. **Non-TTY environments (CI, pipes):** If `os.Stdin` is not a TTY, interactive mode may behave unexpectedly. Mitigation: detect TTY (`term.IsTerminal(os.Stdin.Fd())`) and fall back to the current batch mode if not a TTY.

3. **Signal propagation:** Ctrl+C in interactive mode should terminate the `claude` process cleanly. `cmd.Wait()` with proper signal forwarding handles this. Using `exec.Command` (without a custom `SysProcAttr`) should propagate signals by default on macOS/Linux.

4. **`qode review all` — partial failure:** If code review interactive session exits with error, security review should not start. The existing sequential `runReview("code") → runReview("security")` pattern handles this correctly via error propagation.

5. **`qode review all` — user aborts code review:** If user Ctrl+C's the first review, the second should not run. Handled by checking exit error from `cmd.Wait()`.

6. **Score extraction after interactive run:** Currently `reviewDispatch()` reads score from the file Claude wrote. This still works — Claude writes the file, then we read it. No change needed for score extraction logic.

7. **Clipboard dispatcher:** The `clipboardDispatcher` cannot be made interactive. It should continue working as-is (manual paste workflow). `RunInteractive` must only be available when `claudeCLI` is the resolved dispatcher.

### Security Considerations

- No user input is interpolated into shell commands — `exec.Command` is used (not `exec.CommandContext` with shell), so no command injection risk.
- Prompt files are written to controlled paths under `.qode/branches/`, no user-controlled path traversal.
- `CLAUDECODE` env stripping remains in place.

### Performance

- No performance impact — interactive mode is a direct process handoff, lower overhead than capturing buffers.

---

## 4. Completeness Check

### Acceptance Criteria

1. Running `qode review code` takes over the terminal with an interactive Claude session.
2. Running `qode review security` takes over the terminal with an interactive Claude session.
3. Running `qode plan refine` takes over the terminal with an interactive Claude session.
4. Running `qode plan spec` takes over the terminal with an interactive Claude session.
5. Running `qode start` takes over the terminal with an interactive Claude session.
6. Running `qode review all` runs code review interactively, waits for completion, then runs security review interactively (two sequential interactive sessions).
7. If the first session in `qode review all` exits with an error, the second session does not start.
8. In non-TTY environments (CI), the tool falls back to batch mode (or documents this limitation).
9. Score extraction after interactive review still works (Claude writes the review file, qode reads it).
10. `qode check` is unaffected (it does not use the dispatcher).

### Implicit Requirements

- The interactive session must exit cleanly when Claude is done (no zombie processes).
- The user's terminal state must be restored after the session (Claude CLI is responsible for this, but we must not interfere).
- The `--allowedTools` and `--model` flags should be preserved in interactive mode to maintain consistent tool restrictions.

### Out of Scope

- Changing the clipboard dispatcher behavior.
- Adding progress tracking/logging during interactive sessions (Claude handles its own output).
- Changing the prompt generation logic.
- Making `qode check` interactive.

---

## 5. Actionable Implementation Plan

### Task 1 — Verify `claude` CLI flags for interactive mode
**File:** `internal/dispatch/claude.go`
- Run `claude --help` to confirm available flags.
- Determine how to pass the prompt without using stdin (e.g., `--file`, positional arg, or write to temp file).
- This informs Task 2.

### Task 2 — Add `RunInteractive` to `claudeCLI`
**File:** `internal/dispatch/claude.go`

```go
// RunInteractive runs the claude CLI as a foreground interactive process,
// wiring its stdin/stdout/stderr to the current terminal.
func (c *claudeCLI) RunInteractive(ctx context.Context, prompt string, opts Options) error {
    // Write prompt to temp file (since stdin is needed for terminal)
    f, err := os.CreateTemp("", "qode-prompt-*.md")
    if err != nil {
        return err
    }
    defer os.Remove(f.Name())
    if _, err := f.WriteString(prompt); err != nil {
        return err
    }
    f.Close()

    cmd := exec.CommandContext(ctx, c.binaryPath,
        "--allowedTools", "Read,Write,Glob,Grep",
        "--model", "sonnet",
        "--file", f.Name(),   // or positional arg if --file not supported
    )
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
    if opts.WorkingDir != "" {
        cmd.Dir = opts.WorkingDir
    }
    return cmd.Run()
}
```

Add a package-level function:
```go
// RunInteractive resolves the best available dispatcher and runs interactively.
// Falls back to Resolve().Run() if claudeCLI is not available.
func RunInteractive(ctx context.Context, prompt string, opts Options) error {
    if d := newClaudeCLI(); d.Available() {
        return d.RunInteractive(ctx, prompt, opts)
    }
    // Fallback: clipboard (manual workflow)
    d := &clipboardDispatcher{}
    _, err := d.Run(ctx, prompt, opts)
    return err
}
```

### Task 3 — Update `reviewDispatch` in `internal/cli/review.go`
**File:** `internal/cli/review.go`, function `reviewDispatch` (lines 145–202)

Replace:
```go
d := dispatch.Resolve()
output, err := d.Run(ctx, prompt, dispatch.Options{WorkingDir: root})
```

With:
```go
if err := dispatch.RunInteractive(ctx, prompt, dispatch.Options{WorkingDir: root}); err != nil {
    return err
}
// Score extraction continues unchanged (Claude wrote the file)
```

Remove the progress ticker (lines 154–166) — not needed for interactive mode.

### Task 4 — Update other command dispatchers
**Files:** `internal/cli/plan.go`, `internal/cli/start.go`

Apply the same pattern: replace `d.Run(...)` with `dispatch.RunInteractive(...)`.

### Task 5 — TTY detection guard (optional but recommended)
**File:** `internal/dispatch/claude.go`

```go
import "golang.org/x/term"

func isTTY() bool {
    return term.IsTerminal(int(os.Stdin.Fd()))
}
```

In `RunInteractive`: if `!isTTY()`, fall back to `c.Run(ctx, prompt, opts)` (batch mode) and log a warning.

### Task 6 — Tests
**File:** `internal/dispatch/claude_test.go` (if exists) or new test file

- Mock `exec.Command` to verify `RunInteractive` wires stdio correctly.
- Test that `qode review all` sequential flow works: first interactive call, wait, second interactive call.

### Task 7 — Verify and fix `qode review all`
**File:** `internal/cli/review.go`, `newReviewAllCmd`

Review the sequential call chain and confirm error propagation is correct with `RunInteractive`. Likely no changes needed beyond Task 3.

### Implementation Order

1. Task 1 (flag verification — quick, informs everything)
2. Task 2 (core change)
3. Task 3 (review commands)
4. Task 4 (other commands)
5. Task 5 (TTY guard)
6. Task 6 (tests)
7. Task 7 (verify review all)
