# Security Review — bug-interactive-terminal-commands

**Score: 9.5/10**

---

## Summary

This change replaces buffered subprocess execution with a foreground interactive process handoff. The security surface is small and well-bounded: no new network calls, no new authentication paths, no user-controlled input reaches shell interpolation. The two additions with security relevance are the temp file and the `exec.CommandContext` invocation with a positional argument — both are handled correctly.

---

## Critical Vulnerabilities

None.

---

## High Vulnerabilities

None.

---

## Medium Findings

### M1 — Temp file path included in positional argument without sanitisation

**File:** `internal/dispatch/claude.go:116`
**OWASP:** A03:2021 — Injection (command argument injection)

```go
readInstruction := fmt.Sprintf("Read and execute the instructions in %s", f.Name())

cmd := exec.CommandContext(ctx, c.binaryPath,
    "--allowedTools", "Read,Write,Glob,Grep",
    "--model", "sonnet",
    readInstruction,
)
```

`f.Name()` is the path returned by `os.CreateTemp("", "qode-prompt-*.md")`. The OS chooses this path — typically `/tmp/qode-prompt-<random>.md` on macOS/Linux. The value is not user-controlled and is constructed by the Go runtime.

Because `exec.CommandContext` is used (not `exec.Command` via a shell), the argument is passed directly to `execve(2)` without shell interpretation. There is no injection vector here.

**However**, on some configurations `os.TempDir()` can return a path that includes spaces or special characters (e.g., a user home path via `TMPDIR=/Users/First Last/tmp`). In this case the positional argument `"Read and execute the instructions in /Users/First Last/tmp/qode-prompt-XXX.md"` is a single string passed to `claude` as one argument — this is still correct and safe because no shell is involved.

**Assessment:** No actual vulnerability. Noted for completeness. The use of `exec.CommandContext` (not shell) is the correct mitigation.

**No remediation required.**

---

## Low Findings

### L1 — Temp file default permissions are `0600` but not explicitly enforced

**File:** `internal/dispatch/claude.go:103`

```go
f, err := os.CreateTemp("", "qode-prompt-*.md")
```

`os.CreateTemp` creates the file with mode `0600` (user read/write only) on Unix. This is correct and sufficient — the prompt contents are also persisted to `.qode/branches/` prompt files which have `0644`. No sensitive credentials are written to the temp file.

**Assessment:** No vulnerability. Behaviour is correct by default. Worth noting that if `umask` is set permissively (e.g., `umask 000`), `os.CreateTemp` still creates the file as `0600` because it uses `O_EXCL` with explicit permissions. Safe.

**No remediation required.**

### L2 — Child process inherits full environment minus `CLAUDECODE`

**File:** `internal/dispatch/claude.go:126`

```go
cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
```

The `claude` subprocess receives the full process environment. This is intentional (tokens in `.env` loaded by `internal/env` must be available to `claude`), consistent with the existing `(*claudeCLI).Run()` behaviour, and explicitly documented in the spec. No change needed.

**Assessment:** Intended behaviour. Environment stripping of `CLAUDECODE` is preserved correctly.

**No remediation required.**

---

## Positive Security Observations

- **No shell injection.** `exec.CommandContext` is used throughout — no `sh -c`, no string concatenation into shell commands. The temp file path is passed as a Go string argument directly to `execve`.
- **`CLAUDECODE` stripping preserved.** Both `Run()` and `RunInteractive()` call `filterEnv(os.Environ(), "CLAUDECODE")`, preventing nested session detection bypass.
- **Temp file cleanup.** `defer os.Remove(f.Name())` executes on all exit paths including panics and early error returns after `CreateTemp`.
- **No new network surface.** All network calls are made by the `claude` subprocess. `qode` itself makes no HTTP calls in this change.
- **No new credentials handling.** No tokens, secrets, or API keys are introduced, stored, or logged.
- **No path traversal.** The `opts.WorkingDir` value originates from `resolveRoot()` which resolves the project root via `qode.yaml` walk-up — not from user CLI input.
- **Error messages do not leak secrets.** The `fmt.Errorf("create temp prompt file: %w", err)` and related errors expose only OS error strings and the temp file path — no credentials.
- **Context propagation is correct.** `exec.CommandContext` with `context.Background()` (no deadline) is appropriate for interactive sessions; SIGINT is forwarded to the child process by the OS, enabling clean user-initiated termination.

---

## OWASP Coverage Assessment

| Category | Relevant? | Finding |
|---|---|---|
| A01 Broken Access Control | No | No access control changes |
| A02 Cryptographic Failures | No | No crypto involved |
| A03 Injection | Yes | Reviewed — exec.CommandContext, no shell; safe |
| A04 Insecure Design | No | Design is reviewed in spec |
| A05 Security Misconfiguration | No | No config changes |
| A06 Vulnerable Components | No | No new dependencies |
| A07 Auth Failures | No | No auth changes |
| A08 Software/Data Integrity | No | Temp file via OS, not user input |
| A09 Logging/Monitoring | No | No log changes |
| A10 SSRF | No | No HTTP calls from qode |
