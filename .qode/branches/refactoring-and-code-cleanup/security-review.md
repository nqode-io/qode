# Security Review — qode

**Project:** qode  
**Branch:** refactoring-and-code-cleanup  
**Reviewer:** Claude Security Review  
**Date:** 2026-04-08

---

## Working Assumptions

**What this code trusts:**

| Input | Source | Enforcement |
|-------|--------|-------------|
| Branch `name` | CLI argument | Sanitized via `git.SanitizeBranchName`; path traversal blocked by `safeBranchDir`; leading-`-` **not** checked |
| Base branch `base` | CLI argument | Leading `-` explicitly rejected in `runBranchCreate` |
| Knowledge file path `src` | CLI argument | `filepath.Base(src)` constrains destination; source read is OS-controlled |
| `qode.yaml` / `scoring.yaml` | Filesystem (project root) | New `Validate()` enforces numeric bounds and rubric key whitelist |
| Template overrides | `.qode/prompts/` | Local filesystem only; embedded fallback; no remote fetch |

**Unverified trust:** Branch `name` argument reaches `git checkout -b <name>` without a leading-`-` guard. Since `os/exec` avoids shell interpretation this is not exploitable as code injection, but git would silently treat `-name` as a flag and produce confusing errors.

---

## Security Checklist

### Injection

**Command injection** — All git operations use `exec.CommandContext(ctx, "git", args...)` with arguments as a Go slice, not a shell string. No shell injection surface exists. **Not applicable.**

**Path traversal** — `safeBranchDir` (`internal/cli/branch.go:16–24`) computes `filepath.Rel(base, target)` and rejects any result starting with `..` or equal to `.`. This control has been present and is unchanged by this diff. **Protected.**

**Flag injection into git** — `base` argument validated at `internal/cli/branch.go:47` (`base[0] == '-'`). The `name` argument lacks this check but: (a) `os/exec` passes it as a literal string, not a shell token; (b) git rejects invalid branch names. Impact limited to confusing error messages. **Pre-existing gap, not introduced by this diff.**

**Template injection** — Go `text/template` treats data values as literals, not executable template code. A `{{.SomeField}}` in `ticket.md` would render as the literal string, not be executed. Template *definitions* come from embedded files (compile-time) or local `.qode/prompts/` (user's own filesystem). **Not applicable.**

### Authentication & Authorisation

Local CLI tool with no authentication surface. **Not applicable.**

### Data Exposure

**Sensitive data in logs** — Old code: `fmt.Fprintln(os.Stderr, "Warning: could not load .env file:", err)` — could expose `.env` load path. New code: `log.Warn("could not load .env file", "error", err)` — structured logging, no raw message interpolation. **Improved.**

**Diff file permissions** — `iokit.WriteFile(diffPath, []byte(diff), 0600)` (`internal/cli/review.go:389`). Code diffs may contain sensitive logic; 0600 restricts to owner only. **Correct and unchanged.**

**Secrets in output** — No API keys, credentials, or tokens anywhere in the diff. Templates produce AI prompts with no secret material. **Safe.**

### Input Validation

**Config validation** — New `config.Validate()` (`internal/config/validate.go`) enforces:
- `MinCodeScore >= 0`, `MinSecurityScore >= 0`
- `TargetScore >= 0`
- Rubric keys restricted to `{"refine", "review", "security"}`
- Dimension names non-empty, weights positive

Previously, a `min_code_score: -999` in `qode.yaml` could trivially satisfy any score threshold. This is now caught at load time. **Positive security addition.**

**Knowledge add path** — `dest := filepath.Join(kbDir, filepath.Base(src))` constrains the destination to a flat filename; directory traversal in the destination is impossible. **Safe.**

**Review kind** — New `default: return fmt.Errorf("unknown review kind %q", kind)` prevents silent undefined behavior for unrecognised review types. **Positive.**

### Cryptography

No cryptographic operations in this diff. **Not applicable.**

### Frontend-Specific

Not a web application. **Not applicable.**

### API Security

No HTTP server or API endpoints. **Not applicable.**

### Dependency Issues

No new dependencies introduced. Existing three dependencies (cobra, yaml.v3, godotenv) are unchanged. **Clean.**

---

## Adversary Simulation

1. **Attempt:** Path traversal via branch name `../../etc/passwd` to escape `.qode/branches/`  
   **Target:** `safeBranchDir` (`internal/cli/branch.go:16–24`)  
   **Result:** **BLOCKED** — `filepath.Rel` returns a path starting with `..`; function returns error before any filesystem access.

2. **Attempt:** Git flag injection via base branch `--force` or `-D` to alter git checkout behaviour  
   **Target:** `runBranchCreate` → `git.CreateBranch` (`internal/git/git.go`)  
   **Result:** **BLOCKED** — Leading-`-` check at `branch.go:47` rejects the input before reaching git.

3. **Attempt:** Config score bypass via `qode.yaml` with `min_code_score: -100` to make every review "pass"  
   **Target:** `config.Load` → `config.Validate()` (`internal/config/validate.go`)  
   **Result:** **BLOCKED** — `Validate()` rejects negative scores with an explicit error; `Load` returns the error before any command can proceed.

All three fail due to observed controls. Controls verified in source.

---

## Vulnerabilities

### Low Severity

---

**Severity:** Low  
**OWASP Category:** A03:2021 – Injection  
**File:** `internal/cli/branch.go:60`  
**Vulnerability:** Branch `name` argument passed to `git checkout -b <name>` without leading-`-` validation. If a user passes a name like `--detach`, git interprets it as a flag.  
**Exploit Scenario:** A developer runs `qode branch create -- --detach`. Git receives `git checkout -b --detach` and may behave unexpectedly (detaching HEAD rather than creating a named branch). No code execution is possible; `os/exec` does not use a shell.  
**Remediation:**
```go
// In runBranchCreate, after the base check:
if len(name) > 0 && name[0] == '-' {
    return fmt.Errorf("invalid branch name %q: must not start with '-'", name)
}
```
**Note:** This is a pre-existing condition not introduced by this diff. The refactoring preserved the base-branch check; the same guard should be added for `name`.

---

### Informational

---

**Severity:** Informational  
**OWASP Category:** A05:2021 – Security Misconfiguration  
**File:** `internal/iokit/iokit.go:18`  
**Vulnerability:** `ReadFileOrString` silently swallows all errors, including permission-denied. A file with overly restrictive permissions will fail silently and return the default value.  
**Exploit Scenario:** If `ticket.md` is accidentally set to mode `0000`, the ticket context is silently treated as absent rather than raising a diagnostic. No security vulnerability — a debugging gap only.  
**Remediation:** By design for graceful degradation. Consider adding `slog.Debug` when `os.IsPermission(err)` is true.

---

**Severity:** Informational  
**OWASP Category:** A04:2021 – Insecure Design  
**File:** `internal/cli/init.go:92`  
**Vulnerability:** `os.RemoveAll(scaffoldPromptsDir)` deletes `<root>/.qode/prompts/scaffold/` unconditionally on `qode init`. If `--root` is pointed at an unexpected path this removes a directory the user may not intend.  
**Exploit Scenario:** Developer with a misconfigured `--root` loses a directory under `.qode/prompts/scaffold/`. The blast radius is bounded to that subpath; `os.RemoveAll` on a non-existent path is a no-op.  
**Remediation:** No change required. The path is always `<root>/.qode/prompts/scaffold/`.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 1 |
| Informational | 2 |

**Overall security posture:** Strong. This is a pure refactoring branch. The existing path-traversal, flag-injection, and data-exposure controls are all preserved. Two **positive security additions** were made:

1. **`config.Validate()`** closes a pre-existing gap where negative score thresholds could trivially bypass score gates.
2. **Structured logging** removes `.env` path leakage from stderr.

**Must-fix before merge:** None (no Critical or High findings).

The one Low finding (branch `name` leading-`-`) is pre-existing and outside the scope of this refactoring PR.

---

## Rating

| Dimension | Score | Control or finding that determines this score |
|-----------|-------|------------------------------------------------|
| Command & Path Injection (0–3) | 2.9 | `safeBranchDir` at `branch.go:16–24` uses `filepath.Rel` check; all git calls via `exec.CommandContext` with explicit arg slices (no shell); base-branch `-` check at `branch.go:47`; minor: branch `name` lacks leading-`-` guard (Low, pre-existing) |
| Credential Safety (0–3) | 3.0 | `diff.md` written at 0600 (`review.go:389`); `.env` load error uses structured `log.Warn` (no raw message dump); no API keys or secrets anywhere in diff; `.gitignore` excludes sensitive paths |
| Template Injection (0–3) | 2.8 | Go `text/template` treats data values as literals, not executable code; template definitions from compile-time embedded files or user's own filesystem; scaffold prompt cleanup prevents stale user overrides from blocking future template updates |
| Input Validation & SSRF (0–2) | 1.9 | New `config.Validate()` at `config/validate.go` rejects negative scores and unknown rubric keys; `filepath.Base` constrains knowledge-add destination; no network calls (no SSRF surface); minor gap: branch name `-` prefix unvalidated |
| Dependency Safety (0–1) | 1.0 | No new dependencies introduced; existing three minimal deps (cobra, yaml.v3, godotenv) unchanged |

**Total Score: 11.6/12**  
**Minimum passing score: 10/12** ✅

> Score ≥ 9.6 — specific controls cited above with file:line references rather than absence-of-bugs reasoning.
