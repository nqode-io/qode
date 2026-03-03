# Technical Specification: GitHub Issues Ticket Provider

**Branch:** feat-github-issue-support
**Date:** 2026-03-03
**Status:** Ready for Implementation

---

## 1. Feature Overview

This feature adds GitHub Issues as a fourth ticket provider to `qode ticket fetch`, joining Jira, Azure DevOps, and Linear. When a user runs `qode ticket fetch https://github.com/owner/repo/issues/42`, qode fetches the issue title and body via the GitHub REST API v3 and writes them to `.qode/branches/{branch}/context/ticket.md` in the standard `# {Title}\n\n{Body}\n` format. The implementation follows the identical extension pattern established by the three existing providers — a new `Provider` implementation registered in `init()` — requiring no changes to the CLI layer or any shared infrastructure.

**Business value:** GitHub Issues is the dominant issue tracker for open-source projects and small teams. Without this provider, qode is unusable for GitHub-native workflows without a paid third-party ticketing tool. This change removes that friction.

**Success criteria:**
- `qode ticket fetch <github-issue-url>` writes the correct markdown file for public repos (no token) and private repos (with `GITHUB_TOKEN`).
- 401, 403, and 404 responses return actionable error messages referencing `GITHUB_TOKEN`.
- All unit tests pass. No new third-party dependencies introduced.

---

## 2. Scope

### In Scope

- New `GitHubProvider` struct implementing the `Provider` interface (`Name`, `CanHandle`, `Fetch`).
- GitHub REST API v3 endpoint: `GET https://api.github.com/repos/{owner}/{repo}/issues/{number}`.
- Optional `GITHUB_TOKEN` bearer token (public repos work unauthenticated; private repos require the token).
- Registration of `&GitHubProvider{}` in `internal/ticket/ticket.go` `init()`.
- Comment update in `internal/config/schema.go` to include `github` in the list of valid types.
- Unit tests using `httptest.NewServer` for all success and failure paths.
- New `docs/development-setup.md` covering env-var setup for all four ticketing systems.
- Three `README.md` location updates to document the new provider.

### Out of Scope

- GitHub Enterprise Server or custom GitHub domains.
- GitHub Pull Request URLs (`/pull/` paths).
- Issue comments — description body only.
- Creating, updating, or closing GitHub Issues.
- OAuth/OIDC flows — static token via env var only.
- GitHub GraphQL API.
- Response body size limits (consistent with existing providers; noted as future hardening).

### Assumptions

1. **Public repos:** accessible without authentication at GitHub's unauthenticated rate limit (60 req/hr).
2. **Private repos:** require `GITHUB_TOKEN` with classic PAT `repo` scope or fine-grained `Issues: Read` permission.
3. **Body field:** description only; `null` body decoded as empty string `""`.
4. **`github.com` host only:** no enterprise server support.
5. **`Ticket.URL`:** set to `rawURL` (the original input URL), consistent with all three existing providers.

---

## 3. Architecture & Design

### Component Overview

```
internal/ticket/
├── ticket.go          [MODIFY] — register &GitHubProvider{} in init()
├── jira.go            [unchanged]
├── azuredevops.go     [unchanged]
├── linear.go          [unchanged]
├── github.go          [NEW] — GitHubProvider implementation
└── github_test.go     [NEW] — unit tests via httptest

internal/config/
└── schema.go          [MODIFY] — add "github" to type comment (line 63)

docs/
└── development-setup.md  [NEW] — env-var setup guide for all providers

README.md              [MODIFY] — three locations
```

### Affected Layers

| Layer | Change |
|---|---|
| Ticket provider | New `GitHubProvider` struct in `internal/ticket/github.go` |
| Provider registry | One-line addition in `internal/ticket/ticket.go` `init()` |
| Config schema | Comment-only change in `internal/config/schema.go` |
| Documentation | New `docs/development-setup.md`; three README.md edits |
| CLI | No changes — `internal/cli/ticket.go` is untouched |

### Data Flow

```
User: qode ticket fetch https://github.com/owner/repo/issues/42
         │
         ▼
internal/cli/ticket.go
  DetectProvider(rawURL)
         │
         ▼
internal/ticket/ticket.go — iterates providers
  GitHubProvider.CanHandle(rawURL) → true
         │
         ▼
GitHubProvider.Fetch(rawURL)
  parseGitHubURL(rawURL) → owner, repo, number
  fetchIssue(owner, repo, number)
    GET https://api.github.com/repos/{owner}/{repo}/issues/{number}
    Headers: Accept, X-GitHub-Api-Version, Authorization (if GITHUB_TOKEN set)
    → 200: decode JSON → githubIssue{Number, Title, Body, HTMLURL}
    → 401/403: actionable error with GITHUB_TOKEN hint
    → 404: actionable error with private-repo hint
    → other: raw status + body
  return &Ticket{ID, Title, Body, URL}
         │
         ▼
internal/cli/ticket.go
  fmt.Sprintf("# %s\n\n%s\n", t.Title, t.Body)
  write to .qode/branches/{branch}/context/ticket.md
```

---

## 4. API / Interface Contracts

### Provider Interface (unchanged)

```go
type Provider interface {
    Name() string
    CanHandle(rawURL string) bool
    Fetch(rawURL string) (*Ticket, error)
}
```

### GitHub REST API

**Request:**
```
GET https://api.github.com/repos/{owner}/{repo}/issues/{number}
Accept: application/vnd.github+json
X-GitHub-Api-Version: 2022-11-28
Authorization: Bearer {GITHUB_TOKEN}   // omitted when token is empty
```

**Response (200):**
```json
{
  "number": 42,
  "title": "Fix the bug",
  "body": "Markdown description or null",
  "html_url": "https://github.com/owner/repo/issues/42"
}
```

### Error Responses

| Status | Returned error string |
|---|---|
| 401 | `"GitHub API returned 401 — check GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token"` |
| 403 | `"GitHub API returned 403 — check GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token"` |
| 404 | `"GitHub issue not found — if this is a private repository, set GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token"` |
| other | `"GitHub API returned {N}: {body}"` |
| network err | `"fetching GitHub issue: {err}"` |
| JSON decode err | `"decoding GitHub response: {err}"` |
| bad URL | `"could not parse GitHub URL: ..."` or `"invalid GitHub issue number: {segment}"` |

### `Ticket` Struct (unchanged)

```go
// Returned by GitHubProvider.Fetch:
&Ticket{
    ID:    strconv.Itoa(issue.Number),  // e.g. "42"
    Title: issue.Title,
    Body:  body,                         // "" when issue.Body == nil
    URL:   rawURL,                       // original input URL
}
```

---

## 5. Data Model Changes

No database, schema migrations, or persistent data model changes. The only structural addition is:

```go
// internal/ticket/github.go

type GitHubProvider struct {
    apiBase string // empty string → "https://api.github.com" (production default)
}

type githubIssue struct {
    Number  int     `json:"number"`
    Title   string  `json:"title"`
    Body    *string `json:"body"`   // pointer — JSON null → nil
    HTMLURL string  `json:"html_url"`
}
```

`apiBase` is an unexported field used only in tests to inject a mock server URL. Production code registers `&GitHubProvider{}` (zero value), which resolves to the live API. This is the minimal deviation from the zero-value pattern used by `JiraProvider{}`, `AzureDevOpsProvider{}`, and `LinearProvider{}` (all empty structs) while enabling reliable unit testing.

**Backward compatibility:** Additive only. Existing providers, config files, and CLI behavior are unchanged.

---

## 6. Implementation Tasks

- [ ] **Task 1 (ticket layer):** Create `internal/ticket/github.go` — `GitHubProvider` with `Name`, `CanHandle`, `parseGitHubURL`, `fetchIssue`, and `Fetch` methods. All functions ≤ 50 lines. No new third-party imports.
- [ ] **Task 2 (tests):** Create `internal/ticket/github_test.go` — table-driven and scenario tests using `httptest.NewServer`. Cover all status codes, null body, URL edge cases, and `CanHandle` variants.
- [ ] **Task 3 (registry):** Modify `internal/ticket/ticket.go` `init()` — append `&GitHubProvider{}` to the `providers` slice. (Depends on Task 1.)
- [ ] **Task 4 (config comment):** Modify `internal/config/schema.go` line 63 — add `github` to the inline `// jira, azure-devops, linear, notion, manual` comment.
- [ ] **Task 5 (docs):** Create `docs/development-setup.md` — env-var setup guide for Jira, Azure DevOps, Linear, and GitHub Issues, with example `.env` and verification commands.
- [ ] **Task 6 (README):** Modify `README.md` at three locations — commands table, Ticket System Setup section (add GitHub example + link to setup doc), qode.yaml type comment.

**Recommended order:** 1 → 2 → 3 → 4 → 5 → 6

---

## 7. Testing Strategy

### Unit Tests (`internal/ticket/github_test.go`)

All tests use `httptest.NewServer` and `GitHubProvider{apiBase: server.URL}`.

| Test | Scenario |
|---|---|
| `TestGitHubProvider_Name` | Returns `"github"` |
| `TestGitHubProvider_CanHandle` | Table-driven: valid URL → `true`; `/pull/` → `false`; `linear.app` → `false`; bare repo URL → `false`; trailing slash → `true` |
| `TestParseGitHubURL_Valid` | Correct owner, repo, number; PathEscape applied |
| `TestParseGitHubURL_InvalidPath` | Missing path segments → non-nil error |
| `TestParseGitHubURL_NonNumericNumber` | Non-integer issue segment → non-nil error |
| `TestGitHubProvider_Fetch_Success` | 200 JSON with body → correct `Ticket` fields including `ID == "42"`, `URL == rawURL` |
| `TestGitHubProvider_Fetch_NullBody` | `"body": null` → `Ticket.Body == ""` |
| `TestGitHubProvider_Fetch_NotFound` | 404 → error contains `"not found"` and `"GITHUB_TOKEN"` |
| `TestGitHubProvider_Fetch_Unauthorized` | 401 → error contains `"401"` and `"GITHUB_TOKEN"` |
| `TestGitHubProvider_Fetch_Forbidden` | 403 → error contains `"403"` and `"GITHUB_TOKEN"` |

### Integration Tests

None required — the provider is exercised end-to-end via `qode ticket fetch` using a live GitHub URL in manual verification. The `httptest` unit tests provide sufficient automated coverage.

### Edge Cases to Test Explicitly

- Trailing slash in URL: `https://github.com/owner/repo/issues/42/` → `CanHandle` returns `true`, `parseGitHubURL` succeeds after `path.Clean`.
- PR URL: `https://github.com/owner/repo/pull/42` → `CanHandle` returns `false`.
- Non-numeric issue number: `https://github.com/owner/repo/issues/abc` → `parseGitHubURL` returns error.
- `body` is JSON `null` → `Ticket.Body == ""` (not `"<nil>"` or `"null"`).
- Missing `GITHUB_TOKEN` env var on a 401 response → error message still actionable.

---

## 8. Security Considerations

### Token Handling

- `GITHUB_TOKEN` is read via `os.Getenv` and written only to the `Authorization` request header. It must **never** appear in error messages, log output, or returned error strings. All error paths in `fetchIssue` are constructed before the token is read or contain only the HTTP status code and response body — never the token value.
- Document minimum required token scope: classic PAT with `repo` scope for private repos (no scope needed for public); fine-grained PAT with `Issues: Read`.

### Input Validation

- `owner` and `repo` path segments are extracted from `url.Parse` and passed through `url.PathEscape` before embedding in the API URL to prevent path traversal.
- `number` is validated as a decimal integer via `strconv.Atoi` in `parseGitHubURL`. Since it is validated as pure digits, it is safe to embed in the URL without additional escaping, but `url.PathEscape` is harmless and may be applied for consistency.
- `CanHandle` validates path structure (length ≥ 5 segments, `segments[3] == "issues"`, numeric `segments[4]`) before `Fetch` is called, providing defense in depth.

### Authentication / Authorisation

- No auth state is persisted or cached. Each `Fetch` call reads `GITHUB_TOKEN` fresh from the environment.
- Unauthenticated requests to public repos are allowed by design. The provider does not fail-fast on a missing token (unlike Jira and Azure DevOps providers which error immediately if their required env vars are absent).

### Data Sensitivity

- Issue title and body are written to `.qode/branches/{branch}/context/ticket.md`, which is a local developer file. No data is transmitted beyond the single outbound GitHub API request.
- Future hardening: impose a response body size limit (e.g., 1 MB) consistent with GitHub's own issue body limit (65,536 characters). Currently deferred for consistency with existing providers.

---

## 9. Open Questions

All questions from the refined analysis have been resolved with documented assumptions:

1. **Private repo support** — Yes, supported via `GITHUB_TOKEN`. Resolved: optional token, conditional `Authorization` header.
2. **Comments vs description** — Description (`body` field) only. Resolved: consistent with depth returned by other providers.
3. **Pull Requests** — Out of scope. Resolved: `CanHandle` rejects `/pull/` paths.
4. **GitHub Enterprise Server** — Out of scope. Resolved: `CanHandle` matches `github.com` host only.

**No blocking open questions remain.** Implementation may begin.

---

*Spec generated by qode. Copy to GitHub issue or team wiki for review.*
