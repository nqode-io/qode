# Security Review — feat-support-apps-directory

## Changes Reviewed

Full diff scope (via `git diff main --stat`):
- `internal/workspace/workspace.go` — container dir scanning + monorepo signal files
- `internal/workspace/workspace_test.go` — 6 new test cases
- `internal/detect/detect.go` — container dir scanning in Composite()
- `internal/detect/detect_test.go` — 4 new test cases

## Vulnerability Analysis

### Injection
- **Not applicable.** No user input is passed to shell commands, SQL queries, or templates. All operations use `os.ReadDir()`, `os.Stat()`, and `filepath.Join()` on local filesystem paths derived from directory entries — not user-supplied strings.

### Authentication & Authorisation
- **Not applicable.** This is a local CLI tool performing filesystem scanning. No authentication or authorization is involved.

### Data Exposure
- **No issues found.** No sensitive data is logged, stored, or transmitted. The code only reads directory listings and file existence checks.

### Input Validation

#### 1.
- **Severity:** Informational
- **OWASP Category:** A01:2021 – Broken Access Control
- **File:** `internal/workspace/workspace.go:24-27`, `internal/detect/detect.go:12-15`
- **Vulnerability:** The `knownContainerDirs` map contains fixed well-known directory names. A malicious actor could create a directory named `apps/` containing a crafted project structure to influence `qode init` output. However, this is by design — `qode init` trusts the local filesystem, and a developer running the tool already has full control over the directory structure.
- **Exploit Scenario:** None meaningful. An attacker with write access to the repo could manipulate detection results, but they could also just edit `qode.yaml` directly.
- **Remediation:** No action needed. Trusting local filesystem state is the expected behavior for a developer CLI tool.

#### 2.
- **Severity:** Informational
- **OWASP Category:** A05:2021 – Security Misconfiguration
- **File:** `internal/workspace/workspace.go:80`, `internal/detect/detect.go:76`
- **Vulnerability:** `os.ReadDir()` errors in `countContainerProjects()` and `detectContainerChildren()` are silently swallowed (return 0 / nil). This means permission-denied errors on container directories are ignored rather than surfaced.
- **Exploit Scenario:** None. This is a local development tool. Permission errors would only affect the user's own detection results.
- **Remediation:** No action needed for security purposes. Error handling is consistent with existing codebase patterns where detection functions are best-effort.

### Path Traversal

- **No issues found.** All paths are constructed using `filepath.Join()` with directory entry names from `os.ReadDir()`. Directory entry names cannot contain path separators, so path traversal via `../` is not possible. Symlinks are not followed by `os.ReadDir()`.

### Cryptography
- **Not applicable.** No cryptographic operations.

### Dependency Issues
- **No new dependencies added.** All changes use standard library packages (`os`, `path/filepath`, `strings`, `sort`).

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| Informational | 2 |

**Overall security posture:** Excellent. This change adds local filesystem scanning using only standard library functions. No user input crosses trust boundaries. All paths are safely constructed with `filepath.Join()` using OS-provided directory entry names. No new dependencies, no network operations, no data persistence changes.

**Must-fix before merge:** None.

## Rating

| Dimension | Score (0-2) | Justification |
|-----------|-------------|---------------|
| Injection Prevention | 2.0 | No user input in any command execution or query; all paths from `os.ReadDir` entries |
| Auth & Access Control | 2.0 | N/A for local CLI tool; no auth mechanisms affected |
| Data Protection | 2.0 | No sensitive data handling; only filesystem metadata read |
| Input Validation | 2.0 | Directory names from OS, not user input; `filepath.Join` prevents traversal |
| Dependency Security | 2.0 | No new dependencies; all standard library |

**Total Score: 10.0/10**
