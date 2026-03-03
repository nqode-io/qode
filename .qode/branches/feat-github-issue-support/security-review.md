# Security Review — qode (feat-github-issue-support)

**Branch:** feat-github-issue-support
**Tech Stack:** default (go) at `.`
**Reviewer:** Claude Security Review
**Date:** 2026-03-03 (iteration 3 — post HIGH-1 fix)

---

## Vulnerabilities Found

No Critical or High vulnerabilities remain.

---

### [LOW-1] Success-Path Response Body Not Size-Limited

- **Severity:** Low
- **OWASP Category:** A05:2021 – Security Misconfiguration
- **Files:** `internal/ticket/github.go:112`, `internal/ticket/jira.go:82`, `internal/ticket/azuredevops.go:61`, `internal/ticket/linear.go:61`
- **Vulnerability:** All four providers apply `io.LimitReader(resp.Body, 512)` on error paths (non-200 responses). The success path (200) pipes the body directly to `json.NewDecoder(resp.Body).Decode(...)` without a size cap. A slow/adversarial server (most realistic: a self-hosted Jira instance or compromised Linear account) returning a multi-megabyte body could cause high heap usage.
- **Exploitation context:** For GitHub, `apiBase` is always `https://api.github.com` in production — no user-controlled server. For Jira, the server is constrained to `isAllowedJiraHost` but self-hosted `jira.*` servers are user-controlled. For Azure DevOps and Linear, the API host is hardcoded. Practical risk is low but not zero.
- **Remediation:** Add a named constant to `ticket.go` and wrap the success body in all four providers:
```go
const successBodyMaxBytes int64 = 1 << 20 // 1 MB
// In each provider's decode call:
if err := json.NewDecoder(io.LimitReader(resp.Body, successBodyMaxBytes)).Decode(&result); err != nil {
```

---

### [LOW-2] `dangerTagRe` Does Not Handle Unclosed Dangerous Tags

- **Severity:** Low
- **OWASP Category:** A03:2021 – Injection
- **File:** `internal/ticket/azuredevops.go:93`
- **Vulnerability:** `dangerTagRe` (`(?is)<(script|...)>.*?</(script|...)>`) requires a matching closing tag. A ticket description containing `<script>malicious content` (no `</script>`) passes `dangerTagRe` unchanged; `htmlTagRe` then strips only the `<script>` opener, leaving `malicious content` verbatim in the output written to `ticket.md`. Since `ticket.md` is a Markdown file read by the AI and potentially rendered in a developer IDE, the risk of content injection is low but non-zero.
- **Remediation:** Add a second regex pass for unclosed dangerous tags:
```go
var (
    dangerTagRe      = regexp.MustCompile(`(?is)<(script|style|iframe|object|embed)[^>]*>.*?</(script|style|iframe|object|embed)>`)
    unclosedDangerRe = regexp.MustCompile(`(?is)<(script|style|iframe|object|embed)[^>]*>.*`)
    htmlTagRe        = regexp.MustCompile(`<[^>]+>`)
)

func stripHTML(s string) string {
    s = dangerTagRe.ReplaceAllString(s, "")
    s = unclosedDangerRe.ReplaceAllString(s, "")
    return htmlTagRe.ReplaceAllString(s, "")
}
```

---

### [LOW-3] Clipboard Retains Sensitive Prompt Data Indefinitely

- **Severity:** Low
- **OWASP Category:** A02:2021 – Cryptographic Failures (data exposure)
- **Files:** `internal/cli/start.go:68`, `internal/cli/plan.go:127`, `internal/cli/plan.go:244`
- **Vulnerability:** The `--no-clipboard` opt-out flag was added (good), but clipboard copy remains the default behaviour. Prompts include ticket body, refined analysis, knowledge base content, and tech specs — potentially including PII, internal system architecture, or business logic. On systems with clipboard history managers (Raycast, Alfred, system clipboard history), this content persists indefinitely.
- **Remediation:** The `--no-clipboard` flag is an adequate mitigation for informed users. Documenting this data sensitivity in the help text for the `start` and `plan` commands would improve discoverability. A clipboard-clear timeout (e.g., 30 seconds via a background goroutine clearing the clipboard) would further reduce exposure.

---

## Informational Findings

### [INFO-1] `jira.` Prefix Allowlist Is Intentionally Permissive

- **Severity:** Informational
- **File:** `internal/ticket/jira.go:27-31`
- `isAllowedJiraHost` allows any host with `strings.HasPrefix(host, "jira.")`, e.g., `jira.evil.com`. This is the intended design for self-hosted Jira instances. Users who configure `jira.company.com` explicitly supply that URL. Jira credentials (`JIRA_API_TOKEN`, `JIRA_EMAIL`) would be forwarded to any `jira.*` domain provided. This is a documented trust boundary, not a vulnerability, but worth noting in developer guidance.

### [INFO-2] GitHub Provider Has No Exploitable SSRF Surface

- **Severity:** Informational
- **File:** `internal/ticket/github.go`
- `apiBase` is unexported and only set in tests via struct literal. Production code uses `&GitHubProvider{}` → zero value → `githubAPIBase = "https://api.github.com"`. `CanHandle` uses exact host match (`u.Host != "github.com"`). No user-controlled URL reaches the HTTP layer. Verified clean.

### [INFO-3] Linear / Azure DevOps `CanHandle` Uses Substring Match

- **Severity:** Informational
- **Files:** `internal/ticket/ticket.go:58-64` (`hostContains`), `internal/ticket/azuredevops.go:19`
- `hostContains` uses `strings.Contains(u.Host, sub)`. A URL with host `notlinear.app` passes the Linear check; a URL with host `evildev.azure.com` passes the Azure check. However, both providers construct their API requests against hardcoded hosts (`api.linear.app` and `dev.azure.com` respectively). No credentials can be forwarded to the attacker-controlled `CanHandle`-matching host. The only effect is an improved (or misleading) error message, not a security vulnerability. No action required.

### [INFO-4] `GITHUB_TOKEN` Cannot Appear in Error Strings

- **Severity:** Informational
- **File:** `internal/ticket/github.go:101-113`
- Token value is written only to the `Authorization` request header. All error messages use literal strings and HTTP status codes only. Token cannot leak through logs or error output. Verified correct.

---

## What Was Fixed Across All Iterations

| Issue | Severity | Status |
|---|---|---|
| SSRF + credential forwarding — Jira `CanHandle` substring match | HIGH | **Fixed** — `CanHandle` now uses `url.Parse` + `isAllowedJiraHost` |
| SSRF + credential forwarding — Jira `extractJiraBase` substring match | HIGH | **Fixed** — `extractJiraBase` now uses `isAllowedJiraHost` (exact suffix check) |
| Path traversal → `os.RemoveAll` in `branch.go` | HIGH | **Fixed** — `safeBranchDir` with `filepath.Rel` guard applied to all three branch subcommands |
| Azure DevOps URL path injection | MEDIUM | **Fixed** — `url.PathEscape` on `org`, `project`, `id` |
| No HTTP client timeout | MEDIUM | **Fixed** — `httpClient` with 30s timeout, shared across all four providers |
| Error-path response body unbounded | LOW | **Fixed** — `io.LimitReader(resp.Body, 512)` on all four providers' error paths |
| Sensitive data copied to clipboard | LOW | **Fixed** — `--no-clipboard` persistent flag added; all three copy call sites guarded |
| Closed dangerous HTML tags stripped | LOW | **Fixed** — `dangerTagRe` removes `<script>...</script>` and similar |
| Unclosed dangerous HTML tags pass through | LOW | **Not yet fixed** — LOW-2 above |

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 3 |
| Informational | 4 |

**Must-fix before merge:**
None — no Critical or High vulnerabilities.

**Should-fix before merge:**
1. **[LOW-1]** Add `io.LimitReader` on success-path JSON decode in all four providers.
2. **[LOW-2]** Add `unclosedDangerRe` pass in `stripHTML` for unclosed dangerous tags.
3. **[LOW-3]** No action required on the `--no-clipboard` implementation; consider adding help text noting clipboard sensitivity.

**Overall Security Posture:**

All previously identified Critical and High vulnerabilities are resolved. The Jira SSRF fix is now complete: both `CanHandle` and `extractJiraBase` use `isAllowedJiraHost`, which uses suffix matching (`HasSuffix(".atlassian.net")`) to prevent subdomain-spoofing attacks. The path-traversal fix in `branch.go` is correct and applied to all three subcommands. The new GitHub provider has no exploitable attack surface — exact host match, no user-controlled API base, integer-validated issue numbers, and URL-escaped path parameters. Remaining issues are all Low severity with low exploit probability.

---

## Rating

| Dimension | Score (0-2) | Justification |
|-----------|-------------|---------------|
| Injection Prevention | 1.5 | No command/SQL/GraphQL injection. `url.PathEscape` applied. `dangerTagRe` for closed tags; unclosed tags not handled (LOW-2). |
| Auth & Access Control | 2.0 | All SSRF + credential-forwarding paths closed. Path traversal fully fixed. `isAllowedJiraHost` uses suffix matching. |
| Data Protection | 1.5 | Tokens never in error messages. `LimitReader` on error paths. `--no-clipboard` flag added. Success-path body unlimited (LOW-1). |
| Input Validation | 1.5 | Branch path traversal fixed. GitHub exact host + integer validation. `jira.` prefix intentionally permissive. Azure/Linear hardcoded API hosts remove the risk of permissive CanHandle. |
| Dependency Security | 2.0 | No new third-party dependencies introduced. |

**Total Score: 8.5/10**
