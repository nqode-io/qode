# Security Review — feat-lessons-learned

## Vulnerabilities Found

### 1. Silent rejection of traversal branch names with no user feedback

- **Severity:** Low
- **OWASP Category:** A04:2021 – Insecure Design
- **File:** `internal/cli/knowledge_cmd.go:250`
- **Vulnerability:** `parseBranchArgs` silently drops branch names containing `..`. While this correctly prevents directory traversal, it provides no feedback. A user who types a branch name that happens to contain `..` gets no branches processed and potentially confusing behavior.
- **Exploit Scenario:** Not exploitable — this is a defensive measure. The concern is usability, not security.
- **Remediation:** Add a stderr warning when rejecting:
```go
if strings.Contains(b, "..") {
    fmt.Fprintf(os.Stderr, "Warning: skipping branch %q (contains '..')\n", b)
    continue
}
```

### 2. Git diff may contain secrets passed to AI prompt

- **Severity:** Low
- **OWASP Category:** A02:2021 – Cryptographic Failures
- **File:** `internal/cli/knowledge_cmd.go:221-224`
- **Vulnerability:** `git.DiffFromBase()` captures the diff (truncated to 500 lines) and includes it in the prompt sent to the AI model. If the diff contains accidentally staged secrets, they are included.
- **Exploit Scenario:** Developer stages a `.env` change, runs `qode knowledge add-branch`. Secrets in the diff are sent to the AI.
- **Remediation:** Already mitigated by: (1) 500-line truncation, (2) diff is supplementary context, (3) prompt instructs AI not to include secrets in output. This is consistent with existing commands (code review, security review) that also include diffs. No additional mitigation needed.

### 3. Branch name used in filepath construction for reading review files

- **Severity:** Low
- **OWASP Category:** A01:2021 – Broken Access Control
- **File:** `internal/cli/knowledge_cmd.go:208`
- **Vulnerability:** User-provided branch names are used in `filepath.Join(root, config.QodeDir, "branches", b)` to read review files. Without the `..` check, this could read files outside `.qode/`.
- **Exploit Scenario:** Mitigated by `parseBranchArgs` rejecting `..`. Without that check: `qode knowledge add-branch "../../etc"` could attempt to read `/etc/code-review.md`.
- **Remediation:** Already fixed — `parseBranchArgs` rejects names containing `..`.

### 4. File permissions on lesson files are world-readable

- **Severity:** Informational
- **OWASP Category:** A01:2021 – Broken Access Control
- **File:** `internal/knowledge/knowledge.go:178`
- **Vulnerability:** `SaveLesson` creates files with `0644` (owner rw, others r). This matches the existing codebase pattern and is appropriate for project files that may be committed to git.
- **Exploit Scenario:** If a lesson inadvertently contained sensitive information, other users on a shared system could read it.
- **Remediation:** No change needed. `0644` is standard. Prompt explicitly instructs "Do NOT include credentials, API keys, or secrets."

### 5. `ToKebabCase` sanitizes filenames effectively

- **Severity:** Informational
- **OWASP Category:** A03:2021 – Injection
- **File:** `internal/knowledge/knowledge.go:182-195`
- **Vulnerability:** Not a vulnerability — noting positive security property. `ToKebabCase` only allows `[a-z0-9]`, replacing everything else with hyphens and collapsing them. A title like `../../etc/passwd` becomes `etc-passwd.md`. Null bytes, path separators, and special characters are all sanitized.
- **Exploit Scenario:** None — sanitization is thorough.
- **Remediation:** No change needed.

### 6. No size limit on lesson content

- **Severity:** Informational
- **OWASP Category:** A04:2021 – Insecure Design
- **File:** `internal/knowledge/knowledge.go:168-179`
- **Vulnerability:** `SaveLesson` writes arbitrary-length content to disk. Since this is called by AI output (not direct user CLI input), content size is bounded by AI token limits.
- **Exploit Scenario:** Theoretical only — bounded by AI output limits in practice.
- **Remediation:** No change needed for current usage.

## Checklist Results

| Area | Status | Notes |
|------|--------|-------|
| Injection (SQL/NoSQL/Command/Template) | N/A | No database queries. No shell command construction from user input. `dispatch.RunInteractive` passes content directly. |
| Authentication & Authorization | N/A | CLI tool, no auth system. |
| Data Exposure | Pass | No secrets in logs. Prompt instructs against secrets. Diff truncated. |
| Input Validation | Pass | Branch names reject `..`. `ToKebabCase` sanitizes filenames. Review file reads use hardcoded names. |
| Cryptography | N/A | No cryptographic operations. |
| Frontend-Specific | N/A | No frontend code. |
| API Security | N/A | No API endpoints. |
| Dependency Security | Pass | No new dependencies added. Standard library only for new code. |

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 3 |
| Informational | 3 |

**Overall security posture:** Good. The feature operates on local files within the project directory, introduces no network-facing attack surface, and properly sanitizes filenames and validates branch names. The main security considerations (directory traversal, secret leakage in diffs, filename injection) are all addressed.

**Must-fix before merge:** None — no Critical or High vulnerabilities found.

## Rating

| Dimension | Score (0-2) | Justification |
|-----------|-------------|---------------|
| Injection Prevention | 2.0 | No injection vectors. Filename sanitization is thorough. No shell command construction from user input. |
| Auth & Access Control | 2.0 | N/A for CLI tool. File permissions appropriate (0644). Directory traversal prevented via `..` rejection. |
| Data Protection | 1.5 | Diff may contain secrets, consistent with existing commands. Mitigated by truncation + prompt instructions. Silent branch rejection could use user feedback. |
| Input Validation | 2.0 | Branch names validated. Filenames sanitized. Review file reads use hardcoded names only. |
| Dependency Security | 2.0 | No new dependencies. Standard library only. |

**Total Score: 9.5/10**
