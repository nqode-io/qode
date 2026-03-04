# Security Review — bug-ticket-fetch-does-not-apply-dot-env

**Reviewer:** qode automated security review
**Date:** 2026-03-04
**Score:** 9.5 / 10

---

## Summary

No Critical or High vulnerabilities. The implementation is secure for its threat model: a local developer tool loading secrets from a developer-controlled file into a developer-owned process. Non-override semantics, path derivation from trusted sources, and delegation of parsing to `godotenv` (which does no shell execution) all reduce risk. Two Low findings are noted below.

---

## OWASP Assessment

| Category | Finding | Status |
|----------|---------|--------|
| A01 Broken Access Control | No auth gates changed | ✅ N/A |
| A02 Cryptographic Failures | No crypto involved | ✅ N/A |
| A03 Injection | No user input flows into commands or queries | ✅ Pass |
| A04 Insecure Design | Non-override semantics prevent CI token hijack | ✅ Pass |
| A05 Security Misconfiguration | `.env` in `.gitignore` (line 28, confirmed) | ✅ Pass |
| A06 Vulnerable Components | `godotenv v1.5.1` is current and well-maintained | ✅ Pass |
| A07 Authentication Failures | Tokens used correctly as Bearer / API key headers | ✅ Pass |
| A08 Software Integrity | `go.sum` pins exact module hash | ✅ Pass |
| A09 Security Logging Failures | Warning message includes path but no secret values | ✅ Pass |
| A10 SSRF | GitHub URL parsed with `url.PathEscape`; not affected by this diff | ✅ Pre-existing pass |

---

## Vulnerabilities

### CRITICAL
None.

### HIGH
None.

### MEDIUM
None.

### LOW

#### L1 — `godotenv.Read` expands `$VAR` references from the OS environment
**OWASP:** A03 (Injection — environment variable expansion)
**File:** `internal/env/env.go:27`
**Detail:** `godotenv.Read` performs `$VAR` / `${VAR}` substitution by default using `os.Getenv` at parse time. A `.env` file containing `ALIAS=$GITHUB_TOKEN` would cause `godotenv` to read the current value of `GITHUB_TOKEN` and set it as `ALIAS`. If `ALIAS` is not already in the environment, our code would then call `os.Setenv("ALIAS", <github_token_value>)`. This duplicates a secret under a different name.

**Exploitability:** Requires an attacker to have write access to `.env` in the project root. At that level of access, the attacker already controls the developer's working directory and can achieve far worse outcomes (modify source files, run arbitrary commands, etc.). Exploitability is therefore negligible.

**No action required** for this iteration. Noted for awareness.

#### L2 — World-readable `.env` silently loads without warning
**OWASP:** A05 (Security Misconfiguration)
**File:** `internal/env/env.go:27`
**Detail:** The code reads `.env` regardless of its file permission bits. If `.env` has permissions `0644` or `0755` (readable by all local users on a shared machine), tokens are still loaded without any indication to the user.

**Exploitability:** Requires a multi-user system with other users running `qode` in the same project directory — unlikely for a local developer tool. Low practical impact.

**Suggested remediation (optional):** After the `os.IsNotExist` check, add a permission check:
```go
info, err := os.Stat(path)
if err == nil && info.Mode().Perm()&0o077 != 0 {
    fmt.Fprintf(os.Stderr, "Warning: %s is readable by others (permissions %s)\n", path, info.Mode().Perm())
}
```
This is advisory only and does not block the load.

---

## Positive Security Properties

**Path construction is safe:**
`filepath.Join(root, ".env")` where `root` is derived from `config.FindRoot(".")` or `os.Getwd()` — both return absolute, canonicalized paths. The `--root` flag is *not* parsed at this point (cobra hasn't run yet), so no user-supplied path reaches this code. Path traversal is not possible.

**No shell execution:**
`godotenv.Read` parses the file as key-value pairs using a regex-based parser. It does not invoke `/bin/sh`, `eval`, or backtick substitution. A `.env` file cannot execute arbitrary commands via this code path.

**Non-override semantics protect CI tokens:**
`os.LookupEnv` correctly distinguishes unset from empty-string, ensuring that tokens injected by a CI system (even if set to `""`) cannot be silently overridden by a committed or misconfigured `.env` file.

**Secrets not written to disk:**
Tokens are placed only into the process environment (`os.Setenv`). They are not written to `.qode/` state files, log files, or any other persistent storage.

**Token propagation to subprocesses is intentional:**
`internal/dispatch/claude.go` passes `os.Environ()` to the Claude subprocess via `cmd.Env`. This means `.env` tokens are available to AI assistant invocations, which is the designed behaviour and not a leak.

**`.gitignore` coverage:**
`.env` is listed at line 28 of `.gitignore`. Secrets cannot be accidentally committed.

**Dependency integrity:**
`go.sum` contains the SHA-256 hash of `godotenv v1.5.1`. Any tampered module version would fail `go mod verify`.

---

**Verdict:** Ready to ship. Run `qode check` to verify all quality gates.
