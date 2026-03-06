# Security Review — Add Notion Ticket Support

## Files Reviewed

- `internal/ticket/notion.go` (new — 428 lines)
- `internal/ticket/notion_test.go` (new — 410 lines)
- `internal/ticket/ticket.go` (modified — 1 line)
- `docs/how-to-use-ticket-fetch.md` (modified)
- `docs/qode-yaml-reference.md` (modified)
- `README.md` (modified)

## Vulnerabilities Found

### Vulnerability 1
- **Severity:** Informational
- **OWASP Category:** A07:2021 – Identification and Authentication Failures
- **File:** `internal/ticket/notion.go:46-48`
- **Vulnerability:** The `NOTION_API_KEY` is read from environment variables, which is the standard approach used by all other providers in the codebase. The error message correctly avoids including the token value — only references the variable name.
- **Exploit Scenario:** No exploit — this is a positive observation. An attacker with access to the process environment could read the token, but this is inherent to all env-var-based auth and is mitigated by `.env` files being in `.gitignore`.
- **Remediation:** None needed. Consistent with existing providers.

### Vulnerability 2
- **Severity:** Informational
- **OWASP Category:** A10:2021 – Server-Side Request Forgery (SSRF)
- **File:** `internal/ticket/notion.go:28-38`
- **Vulnerability:** `CanHandle()` validates the URL host against `notion.so` and `*.notion.site` before any HTTP requests are made. The `pageID` is validated as exactly 32 hex characters before being used in API URLs. This prevents SSRF attacks where a malicious URL could trick the provider into making requests to arbitrary hosts.
- **Exploit Scenario:** An attacker provides `https://evil.com/page-id` — `CanHandle()` returns false, and the provider is never invoked. If `https://www.notion.so/../../evil.com` is provided, the `url.Parse` + `path.Clean` handling normalises the path safely.
- **Remediation:** None needed. SSRF is properly mitigated.

### Vulnerability 3
- **Severity:** Informational
- **OWASP Category:** A05:2021 – Security Misconfiguration
- **File:** `internal/ticket/notion.go:202, 261`
- **Vulnerability:** API response bodies are read through `io.LimitReader(resp.Body, notionMaxResponseBytes)` where `notionMaxResponseBytes = 1 << 20` (1 MB). This prevents memory exhaustion from malicious or unexpectedly large responses.
- **Exploit Scenario:** A compromised or malicious Notion API returns a multi-gigabyte response to exhaust server memory — the `LimitReader` caps it at 1 MB.
- **Remediation:** None needed. Properly mitigated.

### Vulnerability 4
- **Severity:** Informational
- **OWASP Category:** A08:2021 – Software and Data Integrity Failures
- **File:** `internal/ticket/notion.go:279`
- **Vulnerability:** The `Authorization: Bearer {token}` header is only sent to the configured `apiBase` (defaulting to `api.notion.com`). In tests, it's sent to `httptest.NewServer` localhost URLs. The token is never logged, printed to stdout, or included in error messages.
- **Exploit Scenario:** None — the token is handled securely.
- **Remediation:** None needed.

### Vulnerability 5
- **Severity:** Informational
- **OWASP Category:** A06:2021 – Vulnerable and Outdated Components
- **File:** `internal/ticket/notion.go`
- **Vulnerability:** No new external dependencies are added. The implementation uses only Go standard library packages (`net/http`, `encoding/json`, `net/url`, `io`, `os`, `path`, `strings`, `fmt`). No third-party libraries to audit.
- **Exploit Scenario:** N/A
- **Remediation:** None needed.

## Summary

| Severity | Count |
|---|---|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| Informational | 5 |

### Overall Security Posture Assessment

The Notion provider implementation demonstrates strong security practices:

1. **SSRF Prevention:** URL host validation in `CanHandle()` restricts requests to `notion.so` and `*.notion.site` domains only. Page IDs are validated as 32-character hex strings.
2. **Credential Handling:** API key is read from environment, never logged or exposed in error messages. Bearer token is only sent to the validated Notion API endpoint.
3. **Input Validation:** Page ID is validated as exactly 32 hex characters. URLs are parsed with `url.Parse` and paths cleaned with `path.Clean`.
4. **Resource Exhaustion Protection:** Response bodies are limited to 1 MB via `io.LimitReader`. Pagination prevents unbounded memory growth by fetching blocks in pages of 100.
5. **No New Dependencies:** Zero new third-party packages, eliminating supply chain risk.

### Must-Fix Before Merge

None — no Critical or High vulnerabilities found.

## Rating

| Dimension | Score (0-2) | Justification |
|-----------|-------------|---------------|
| Injection Prevention | 2 | No user input reaches shell commands, SQL, or templates. URL components are properly parsed and validated. Page ID is hex-validated before use in API URLs. |
| Auth & Access Control | 2 | Token read from env var, sent via Bearer header only to validated Notion API host. Error messages reference env var name without exposing token value. Consistent with all existing providers. |
| Data Protection | 2 | No secrets in logs or error messages. API responses size-limited. Fetched content written only to local `context/ticket.md`. `.env` documented as requiring `.gitignore`. |
| Input Validation | 2 | URL host validated against allowlist. Page ID validated as 32 hex chars. Query parameters properly URL-encoded via `url.Values`. Unknown block types silently skipped (no panic). |
| Dependency Security | 2 | Zero new dependencies. Uses only Go standard library. No supply chain risk introduced. |

**Total Score: 10.0/10**
