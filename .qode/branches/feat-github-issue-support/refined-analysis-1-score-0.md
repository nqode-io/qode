# Requirements Analysis: GitHub Issues Integration

**Branch:** feat-github-issue-support
**Date:** 2026-03-03

---

## 1. Problem Understanding

### Restated Problem

Users of `qode` can already pull ticket context from Jira, Azure DevOps, and Linear via `qode ticket fetch <url>`. A significant portion of open-source and small-team development happens on GitHub, which has its own native issue tracker. Currently there is no way to use a GitHub Issues URL with `qode ticket fetch`, leaving GitHub-native teams without first-class support.

### User Need & Business Value

- **User need:** Run `qode ticket fetch https://github.com/owner/repo/issues/42` and have the issue title and body saved to `.qode/branches/{branch}/context/ticket.md`, exactly as with other providers.
- **Business value:** Expands the addressable user base to GitHub-first teams; removes friction for open-source contributors who naturally work in GitHub Issues.

### Open Questions

1. Should private-repo issues be supported? (Assumed yes — a token with `repo` scope grants access; a public-only mode would require no token but is a subset.)
2. Should GitHub issue comments be included in the fetched body, or only the issue description? (Assumed description only, matching the level of detail returned by other providers.)
3. Should GitHub pull requests (same URL path pattern with `/pull/` instead of `/issues/`) be out of scope? (Assumed yes — the ticket explicitly says "issues".)
4. Should `github.com/login` (GitHub Enterprise Server at custom domains) be supported? (Assumed out of scope for this ticket; note it as a future extension.)

---

## 2. Technical Analysis

### Affected Components

| File | Change Type | Reason |
|---|---|---|
| `internal/ticket/github.go` | **New** | GitHub Issues provider implementation |
| `internal/ticket/github_test.go` | **New** | Tests for the provider |
| `internal/ticket/ticket.go` | **Modify** | Register `&GitHubProvider{}` in `providers` slice (init, lines 26-34) |
| `internal/config/schema.go` | **Modify** | Add `"github"` to the comment on line 63 listing valid `type` values |
| `README.md` | **Modify** | Add GitHub setup snippet to the existing "Ticket System Setup" section (lines 194-209) |
| `docs/development-setup.md` | **New** | Comprehensive dev-environment setup guide covering all ticketing system API keys |

### Key Technical Decisions

#### A. API Choice: REST v3, not GraphQL

GitHub offers both REST (`api.github.com/repos/{owner}/{repo}/issues/{number}`) and GraphQL (`api.github.com/graphql`). The Linear provider uses GraphQL because Linear's REST API is limited. GitHub's REST API is first-class and straightforward; using it keeps the provider consistent with Jira and Azure DevOps and avoids a new dependency pattern.

Endpoint: `GET https://api.github.com/repos/{owner}/{repo}/issues/{number}`

#### B. Authentication: `GITHUB_TOKEN` environment variable

`GITHUB_TOKEN` is the conventional name used by GitHub Actions and the `gh` CLI. It supports both classic PATs and fine-grained PATs. The token is sent as `Authorization: Bearer {token}`.

Unauthenticated requests are allowed for public repos but are rate-limited to 60 requests/hour. Providing the token raises this to 5,000/hour. The provider should work without a token for public repos, but warn if the token is absent.

#### C. URL Parsing

GitHub issue URLs follow a fixed scheme:
```
https://github.com/{owner}/{repo}/issues/{number}
```
Parsing rules:
- Host must be exactly `github.com` (use `hostContains("github.com")` helper already in `ticket.go`).
- Path segments: `["", owner, repo, "issues", number]` — split by `/`, validate length ≥ 5 and segment `[3]` == `"issues"`.
- Issue number: extract segment `[4]`, convert to integer for validation, use raw string in URL.

#### D. Response Parsing

The REST v3 response for `GET /repos/{owner}/{repo}/issues/{number}` returns JSON. Relevant fields:

```json
{
  "number": 42,
  "title": "Fix the bug",
  "body": "Description in Markdown",
  "html_url": "https://github.com/owner/repo/issues/42"
}
```

`body` is already Markdown — no HTML stripping required (unlike Azure DevOps). The field can be `null` for issues with no description; handle with a nil-safe check.

#### E. Struct and Function Size Constraints

All functions must be ≤ 50 lines. The provider should be decomposed as follows:
- `Name() string` — trivial
- `CanHandle(rawURL string) bool` — URL host + path check
- `Fetch(rawURL string) (*Ticket, error)` — orchestrator calling helpers
- `parseGitHubURL(rawURL string) (owner, repo, number string, err error)` — URL decomposition
- `fetchIssue(owner, repo, number string) (*githubIssue, error)` — HTTP call and JSON decode

### Patterns to Follow

- Provider struct with unexported fields: see `LinearProvider` (no fields needed; `GitHubProvider{}` is a zero-value struct like all existing providers).
- `hostContains()` helper is in `ticket.go`; call it from `CanHandle`.
- HTTP pattern from `azuredevops.go`: `http.NewRequest` → set headers → `http.DefaultClient.Do` → check status → `json.NewDecoder`.
- Error wrapping: `fmt.Errorf("github: parse URL %q: %w", rawURL, err)`.
- Missing env var: return `fmt.Errorf("github: GITHUB_TOKEN not set; ...")` only when token is required (i.e., API returns 404 or 401).

### Dependencies on Other Features

None. This is a self-contained new provider. All infrastructure (Provider interface, DetectProvider, CLI `ticket fetch` command) already exists.

---

## 3. Risk & Edge Cases

### What Could Go Wrong

| Risk | Mitigation |
|---|---|
| API returns 404 for a private repo when no token is set | Return clear error: "issue not found; if this is a private repository, set GITHUB_TOKEN" |
| API returns 401 (bad token) | Return error with token env var name |
| API rate limit hit (no token, public repo) | HTTP 403 with `X-RateLimit-Remaining: 0`; return error explaining rate limit and how to set token |
| `body` field is JSON `null` | Nil-pointer dereference; use a pointer field `Body *string` in the decode struct or check for nil and default to empty string |
| Issue number is not a valid integer in the URL | Validate during parsing, return descriptive parse error |
| URL path has trailing slash or query params | `url.Parse` handles this; extract from cleaned path segments |
| GitHub Enterprise Server (custom domain) | Out of scope; `CanHandle` only matches `github.com` |
| PR URL passed instead of issue URL (`/pull/`) | `CanHandle` must require path segment `[3] == "issues"` to reject PR URLs |

### Security Considerations

- **Token leakage in logs:** Never log the `Authorization` header or the token value. All existing providers follow this; maintain the pattern.
- **URL injection:** The owner, repo, and number are embedded in the API URL path. Validate that `number` matches `^\d+$` and that `owner`/`repo` do not contain path-traversal characters. `url.PathEscape` can be applied but these segments should only contain alphanumeric, `-`, and `.` characters for valid GitHub slugs.
- **Response body size:** GitHub issue bodies can theoretically be very large. The existing providers do not impose a limit; maintain consistency but note this as a future hardening opportunity.
- **Minimum required token scope:** Document that a fine-grained PAT with `Issues: Read` permission is sufficient. A classic PAT needs only `repo` scope for private repos (no scope for public repos).

### Performance Implications

Single HTTP request per invocation. No pagination needed (one issue). No caching layer. Acceptable — identical to other providers.

---

## 4. Completeness Check

### Acceptance Criteria

1. `qode ticket fetch https://github.com/owner/repo/issues/42` fetches the issue and saves title + body to `.qode/branches/{branch}/context/ticket.md`.
2. Works for public repos without a token.
3. Works for private repos when `GITHUB_TOKEN` is set to a valid PAT with `repo` scope or Issues read permission.
4. Returns a clear, actionable error when the issue is not found (404).
5. Returns a clear, actionable error when the token is invalid or missing and the repo is private (401/404).
6. `CanHandle` returns `false` for non-GitHub URLs.
7. `CanHandle` returns `false` for GitHub PR URLs (`/pull/`).
8. `CanHandle` returns `true` only for `github.com` host with `/issues/` path.
9. Unit tests cover: `Name()`, `CanHandle()` (multiple cases), `Fetch()` (success), `Fetch()` (404), `Fetch()` (missing token message), `parseGitHubURL()` (valid and invalid inputs).
10. `docs/development-setup.md` is created covering environment setup for all four ticketing systems (Jira, Azure DevOps, Linear, GitHub Issues).
11. `README.md` references `docs/development-setup.md` and includes a GitHub Issues snippet in the Ticket System Setup section.
12. `internal/ticket/ticket.go` registers `&GitHubProvider{}` in the `providers` slice.
13. `internal/config/schema.go` comment on `Type` field updated to include `"github"`.

### Implicit Requirements

- The `Fetch` output format must match other providers: `# {Title}\n\n{Body}` (see `internal/cli/ticket.go` lines where the file is written).
- The provider file must not exceed 50 lines per function.
- No new third-party dependencies may be introduced (the GitHub REST API can be consumed with `net/http` + `encoding/json`, both already used by existing providers).

### Explicitly Out of Scope

- GitHub Enterprise Server (custom domain) support.
- Fetching GitHub Pull Requests.
- Fetching issue comments.
- Creating or updating GitHub Issues.
- OAuth flow — only static token via env var.
- GraphQL API usage.

---

## 5. Actionable Implementation Plan

### Task Order

#### Task 1 — Implement `GitHubProvider` (commit: `feat: add GitHub Issues ticket provider`)

**File:** `internal/ticket/github.go`

```
Functions:
  GitHubProvider struct                      (zero-value, no fields)
  Name() string                              → "github"
  CanHandle(rawURL string) bool              → host is github.com AND path[3] == "issues"
  parseGitHubURL(rawURL string) → (owner, repo, number string, err error)
  fetchIssue(owner, repo, number string) → (*githubIssue, error)   [HTTP + JSON]
  Fetch(rawURL string) → (*Ticket, error)    [orchestrator]

Internal types:
  githubIssue struct { Number int; Title string; Body *string; HTMLURL string }
```

Constraints: each function ≤ 50 lines; use `hostContains`; token from `os.Getenv("GITHUB_TOKEN")`; set `Accept: application/vnd.github+json` and `X-GitHub-Api-Version: 2022-11-28` headers per GitHub best practices.

#### Task 2 — Register provider (commit: `feat: register GitHubProvider in ticket registry`)

**File:** `internal/ticket/ticket.go`

Add `&GitHubProvider{}` to the `providers` slice in `init()`. No other changes.

#### Task 3 — Update config schema comment (commit: `chore: add github to ticket system type comment`)

**File:** `internal/config/schema.go`

Update the comment on the `Type` field (line 63) from `// jira, azure-devops, linear, notion, manual` to include `github`.

#### Task 4 — Write unit tests (commit: `test: add unit tests for GitHubProvider`)

**File:** `internal/ticket/github_test.go`

Tests (use `httptest.NewServer` to mock the GitHub API — this is the correct approach; existing providers don't test HTTP because they lack mocks, but new code should):

| Test | What it verifies |
|---|---|
| `TestGitHubProvider_Name` | Returns `"github"` |
| `TestGitHubProvider_CanHandle_ValidURL` | Returns `true` for `https://github.com/owner/repo/issues/42` |
| `TestGitHubProvider_CanHandle_PRUrl` | Returns `false` for `/pull/` URL |
| `TestGitHubProvider_CanHandle_NonGitHub` | Returns `false` for `linear.app` URL |
| `TestGitHubProvider_CanHandle_NoIssuesPath` | Returns `false` for `github.com/owner/repo` |
| `TestParseGitHubURL_Valid` | Correctly extracts owner, repo, number |
| `TestParseGitHubURL_InvalidPath` | Returns error for malformed URL |
| `TestGitHubProvider_Fetch_Success` | Mock server returns 200, verifies Ticket fields |
| `TestGitHubProvider_Fetch_NullBody` | Mock server returns `"body": null`, verifies empty string body |
| `TestGitHubProvider_Fetch_NotFound` | Mock server returns 404, verifies error message |

#### Task 5 — Create `docs/development-setup.md` (commit: `docs: add development environment setup guide`)

**File:** `docs/development-setup.md`

Content sections:
1. Prerequisites (Go version, git)
2. Ticketing System API Keys
   - **Jira:** `JIRA_EMAIL`, `JIRA_API_TOKEN` — how to generate at `id.atlassian.com/manage-profile/security/api-tokens`
   - **Azure DevOps:** `AZURE_DEVOPS_PAT` — how to generate a PAT with Work Items read scope
   - **Linear:** `LINEAR_API_KEY` — how to generate at `linear.app/settings/api`
   - **GitHub Issues:** `GITHUB_TOKEN` — classic PAT (no scopes for public, `repo` for private) or fine-grained PAT (`Issues: Read`)
3. Example `.env` file snippet (do NOT commit)
4. Verification commands using `qode ticket fetch` for each provider

#### Task 6 — Update `README.md` (commit: `docs: add GitHub Issues to README ticket system setup`)

**File:** `README.md`

- Add GitHub Issues example to the existing "Ticket System Setup" section (after line 209).
- Add a link to `docs/development-setup.md` at the top of the Ticket System Setup section.

```markdown
For full setup instructions see [Development Setup](docs/development-setup.md).

# GitHub Issues
export GITHUB_TOKEN=ghp_yourtoken   # optional for public repos
qode ticket fetch https://github.com/owner/repo/issues/42
```

### Prerequisite Work

None. All necessary infrastructure exists. Tasks 1–6 are independent except:
- Task 2 depends on Task 1 (provider must exist before it can be registered).
- Tasks 5 and 6 can be done in any order relative to Tasks 1–4.
