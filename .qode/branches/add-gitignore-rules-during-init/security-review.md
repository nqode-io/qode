# Security Review — add-gitignore-rules-during-init

**Project:** qode
**Branch:** add-gitignore-rules-during-init
**Reviewer:** Claude Sonnet 4.6
**Date:** 2026-04-09

---

## Working Assumptions

**What this code trusts:**

| Input | Source | Validated? |
|-------|--------|------------|
| `root` (directory path) | CLI caller (`runInitExisting`) | No explicit traversal check; trusted as project directory |
| `.gitignore` content | Local filesystem | Read as opaque bytes; not parsed or executed |
| Written content | Hardcoded `GitignoreRules` slice | Fully controlled — no user data flows in |

The only trust assumption that matters: `root` is expected to be the user's project directory. No external input reaches `AppendGitignoreRules` other than this path.

---

## Changes Reviewed

- `internal/scaffold/gitignore.go` — new file implementing `AppendGitignoreRules`
- `internal/cli/init.go` — wires `AppendGitignoreRules` into `runInitExisting`
- `internal/cli/init_test.go` — integration tests for the wiring
- `internal/scaffold/gitignore_test.go` — unit tests for `AppendGitignoreRules`
- `ROADMAP.md` — documentation only; not reviewed

---

## Vulnerabilities Found

### [1] Informational — Substring matching can silently skip a gitignore rule

- **Severity:** Informational
- **OWASP Category:** A05:2021 – Security Misconfiguration
- **File:** [internal/scaffold/gitignore.go:43](internal/scaffold/gitignore.go#L43)
- **Vulnerability:** Idempotency is checked with `strings.Contains(existing, rule)`. If a rule string appears inside a comment or a longer pattern (e.g., `# .qode/branches/*/diff.md`), the check considers the rule "present" and silently skips appending it. The rule then never takes effect in git, causing the file it was meant to ignore to be tracked.
- **Exploit Scenario:** A developer manually edits `.gitignore` and comments out a qode rule with `# .qode/branches/*/diff.md`. Re-running `qode init` does not restore the active rule — `diff.md` files get committed accidentally, leaking intermediate prompt diffs.
- **Remediation:** Match rules against non-comment lines only, or match by line equality rather than substring presence. A simple approach: strip comment lines before checking.

```go
var activeLines []string
for _, line := range strings.Split(existing, "\n") {
    if !strings.HasPrefix(strings.TrimSpace(line), "#") {
        activeLines = append(activeLines, line)
    }
}
activeExisting := strings.Join(activeLines, "\n")
// then check strings.Contains(activeExisting, rule)
```

---

### [2] Informational — Non-atomic write to `.gitignore`

- **Severity:** Informational
- **OWASP Category:** A05:2021 – Security Misconfiguration
- **File:** [internal/scaffold/gitignore.go:65](internal/scaffold/gitignore.go#L65)
- **Vulnerability:** `iokit.WriteFileCtx` is used instead of `iokit.AtomicWriteCtx`. Looking at the `iokit` implementation, `WriteFileCtx` calls `os.WriteFile` directly — there is no temp-file + rename. If the process is interrupted mid-write (SIGKILL, power loss), `.gitignore` is left truncated or partially written.
- **Exploit Scenario:** No active exploit, but a corrupted `.gitignore` could cause `git` to misinterpret patterns, potentially tracking files that should be ignored (e.g., scored iteration copies with sensitive prompt context).
- **Remediation:** Use `iokit.AtomicWriteCtx` for consistency with project conventions and robustness. The test comment at `gitignore_test.go:146` already *describes* atomic semantics ("aborts before the rename") — the implementation should match the documented intent.

---

### [3] Informational — Misleading test comment describes wrong mechanism

- **Severity:** Informational
- **OWASP Category:** N/A
- **File:** [internal/scaffold/gitignore_test.go:146](internal/scaffold/gitignore_test.go#L146)
- **Vulnerability:** The comment states "iokit.WriteFileCtx performs an atomic write; a pre-cancelled context aborts before the rename." There is no rename — `WriteFileCtx` performs a direct `os.WriteFile`. The test is correct (context checked before write = no file created), but the comment is wrong about the mechanism. If a future engineer changes the implementation expecting atomic semantics, they may break the test or rely on rename ordering.
- **Remediation:** Correct the comment to accurately describe the context check:

```go
// WriteFileCtx checks ctx.Err() before writing; a pre-cancelled context
// returns an error without creating the file.
```

---

## Adversary Simulation

**Attempt 1: Path traversal via crafted `root`**
| Field | Detail |
|-------|--------|
| Attempt | Pass `root = "../../../../tmp/evil"` to write `.gitignore` outside the project |
| Target | `AppendGitignoreRules` at `gitignore.go:32` via `filepath.Join(root, ".gitignore")` |
| Result | **Blocked by trust model.** `root` comes from `runInitExisting`, which is called by the `init` cobra command from the working directory. No shell input reaches `root` from an unauthenticated source. A local attacker who can control `root` already has full filesystem access. Content written is hardcoded gitignore patterns — not exploitable even if path is wrong. |

**Attempt 2: Gitignore rule suppression via comment manipulation**
| Field | Detail |
|-------|--------|
| Attempt | Edit `.gitignore` to include `# .qode/branches/*/diff.md` (commented), then run `qode init` again |
| Target | `strings.Contains` check at `gitignore.go:43` |
| Result | **Succeeds** (Informational finding #1). The rule is not re-added. `diff.md` files become tracked and may contain intermediate prompt context that the developer did not intend to commit. |

**Attempt 3: Symlink substitution before write**
| Field | Detail |
|-------|--------|
| Attempt | Create a symlink at `.gitignore → /tmp/target` before running `qode init` |
| Target | `os.ReadFile` then `os.WriteFile` in `gitignore.go:34` and `gitignore.go:65` |
| Result | **Blocked by content constraint + permissions.** Even if the symlink is followed, the only content written is hardcoded gitignore patterns — no sensitive data, no code execution, no credential leak. Writing to `/tmp/target` produces a harmless file. Overwriting a meaningful file requires that the attacker already control the symlink on the developer's local machine, which implies full compromise. |

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| Informational | 3 |

**Must-fix before merge:** None. All findings are Informational.

**Overall security posture:** The change has a minimal attack surface. All written content is hardcoded; no user input flows into file contents or path construction in a way that creates injection risk. The primary concern is a correctness gap (substring-match idempotency) that could cause `.gitignore` rules to be silently missed when commented out — this has a weak privacy implication (unintended commits of `diff.md` files) rather than a security exploit. Using `AtomicWriteCtx` instead of `WriteFileCtx` would align the implementation with the project's stated convention and the test's documented assumptions.

---

## Rating

| Dimension | Score | Control or finding that determines this score |
|-----------|-------|------------------------------------------------|
| Command & Path Injection (0–3) | 2.9 | `filepath.Join` at `gitignore.go:32` for path construction; all written content is hardcoded `GitignoreRules` at `gitignore.go:20-26`; zero shell execution in the diff |
| Credential Safety (0–3) | 3.0 | No credentials, tokens, or secrets anywhere in scope; error wrapping at `gitignore.go:36,65` exposes only file path, appropriate for a local CLI |
| Template Injection (0–3) | 3.0 | `strings.Builder` at `gitignore.go:52-63` concatenates only hardcoded constants; no user-supplied data reaches the written output |
| Input Validation & SSRF (0–2) | 1.8 | `root` accepted without traversal normalisation at `gitignore.go:32`; `strings.Contains` at `gitignore.go:43` produces false positives on commented rules (finding #1); no network surface |
| Dependency Safety (0–1) | 1.0 | No new external dependencies; only existing internal `iokit` package used |

**Total Score: 11.7/12**
**Minimum passing score: 10/12 — PASS**
