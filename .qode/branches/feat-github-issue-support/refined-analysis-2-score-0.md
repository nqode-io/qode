<!-- qode:iteration=3 -->

# Requirements Analysis: GitHub Issues Integration

**Branch:** feat-github-issue-support
**Date:** 2026-03-03

---

## 1. Problem Understanding

### Restated Problem

`qode ticket fetch <url>` supports three providers today: Jira, Azure DevOps, and Linear (registered in `internal/ticket/ticket.go:init()` lines 28–33). GitHub Issues is the most common issue tracker for open-source projects and small teams, yet there is no way to use a `github.com` URL with the command. This ticket adds GitHub Issues as a fourth provider, following the identical extension pattern used by the existing three.

### User Need & Business Value

- **User need:** `qode ticket fetch https://github.com/owner/repo/issues/42` saves the issue title and body to `.qode/branches/{branch}/context/ticket.md` in the same `# {Title}\n\n{Body}\n` format written by `internal/cli/ticket.go:59`: `fmt.Sprintf("# %s\n\n%s\n", t.Title, t.Body)`.
- **Business value:** Removes friction for GitHub-native teams and open-source contributors; makes qode usable without any paid third-party ticketing tool.

### Open Questions (assumptions documented)

1. **Private repos:** Assumed yes — a token with `repo` scope or fine-grained `Issues: Read` permission grants access. Without a token, public repos still work (GitHub REST API allows unauthenticated reads at 60 req/hr).
2. **Comments vs description:** Assumed description only (`body` field), matching the depth returned by Jira, Azure DevOps, and Linear.
3. **Pull Requests:** Out of scope — ticket says "issues". `CanHandle` must reject `/pull/` paths.
4. **GitHub Enterprise Server:** Out of scope — `CanHandle` only matches the `github.com` host.

---

## 2. Technical Analysis

### Affected Components

| File | Change | Notes |
|---|---|---|
| `internal/ticket/github.go` | **New** | GitHubProvider implementation |
| `internal/ticket/github_test.go` | **New** | Unit tests using `httptest.NewServer` |
| `internal/ticket/ticket.go` | **Modify** | Add `&GitHubProvider{}` to `providers` slice in `init()` line 29 |
| `internal/config/schema.go` | **Modify** | Line 63: add `github` to `// jira, azure-devops, linear, notion, manual` |
| `README.md` | **Modify** | Three locations: "Ticket System Setup" section (line 194), commands table (line 127), qode.yaml reference (line 243) |
| `docs/development-setup.md` | **New** | Comprehensive env-var setup guide for all four ticketing systems |

### Key Technical Decisions

#### A. GitHub REST API v3 (not GraphQL)

Endpoint: `GET https://api.github.com/repos/{owner}/{repo}/issues/{number}`

Linear uses GraphQL because Linear's REST API is incomplete. GitHub's REST API is first-class. Using REST keeps the provider consistent with `azuredevops.go` and `jira.go` and avoids a new dependency pattern. No new imports beyond stdlib.

Required headers per GitHub best practices:
- `Accept: application/vnd.github+json`
- `X-GitHub-Api-Version: 2022-11-28`
- `Authorization: Bearer {token}` (omitted when token is empty)

#### B. GITHUB_TOKEN is optional (unlike Jira/ADO)

Jira and Azure DevOps check the env var before making any HTTP request and return an error if missing (`jira.go:32–35`, `azuredevops.go:28–31`). GitHub cannot follow this pattern because public repos are accessible without authentication.

Correct approach: always attempt the request. Conditionally set the `Authorization` header if `GITHUB_TOKEN` is set. On a 401 or 403 response, return an actionable error that mentions `GITHUB_TOKEN`. On a 404, return an error that mentions the possibility of a private repo requiring the token.

Error format follows the existing convention from `jira.go:33` and `azuredevops.go:30`:
```
GITHUB_TOKEN environment variable not set
Set it with: export GITHUB_TOKEN=your-token
```

#### C. URL Parsing

GitHub issue URL structure: `https://github.com/{owner}/{repo}/issues/{number}`

- `hostContains(rawURL, "github.com")` — reuse the helper defined in `ticket.go:53–59`
- Parse path with `url.Parse`, use `path.Clean` to strip trailing slash, split by `/`, validate: `len(segments) >= 5`, `segments[3] == "issues"`, `segments[4]` is a valid decimal string
- `CanHandle` checks both host and path; returns `false` for `/pull/` paths and for `github.com/owner/repo` (no issue number)

#### D. Response Parsing

JSON response fields of interest:
```json
{ "number": 42, "title": "Fix the bug", "body": "Markdown text or null", "html_url": "..." }
```

`body` is already Markdown — no `stripHTML` call needed (contrast with `azuredevops.go:66` which calls `stripHTML(result.Fields.Description)`). `body` can be JSON `null` for issues with no description; decode into `Body *string` and default to empty string when nil.

`Ticket.ID` should be the issue number as a decimal string (e.g., `"42"`), consistent with `azuredevops.go:64`: `fmt.Sprintf("%d", result.ID)`. Use `strconv.Itoa(issue.Number)`.

`Ticket.URL` should be `rawURL` (the original input URL), consistent with all three existing providers which set `URL: rawURL`.

#### E. Testability — apiBase injection

Existing providers have no HTTP tests because they hardcode production URLs and provide no injection mechanism. This provider must be testable with `httptest.NewServer`.

`GitHubProvider` will have one unexported field:

```go
type GitHubProvider struct {
    apiBase string // empty → "https://api.github.com"
}
```

Production code (in `ticket.go` init): `&GitHubProvider{}` — zero value, uses default.
Tests: `&GitHubProvider{apiBase: server.URL}` — points at mock server.

This is the minimal deviation from the zero-value pattern used by other providers (`JiraProvider{}`, `AzureDevOpsProvider{}`, `LinearProvider{}` — all empty structs) while enabling reliable testing.

#### F. Function decomposition (50-line constraint)

- `(p *GitHubProvider) Name() string` — 1 line
- `(p *GitHubProvider) CanHandle(rawURL string) bool` — host + path check, ~10 lines
- `parseGitHubURL(rawURL string) (owner, repo, number string, err error)` — URL decomposition, ~20 lines
- `(p *GitHubProvider) fetchIssue(owner, repo, number string) (*githubIssue, error)` — HTTP + JSON, ~35 lines
- `(p *GitHubProvider) Fetch(rawURL string) (*Ticket, error)` — orchestrator, ~15 lines

All well within the 50-line limit.

#### G. Required imports for `github.go`

All stdlib — no new third-party dependencies:
- `encoding/json` — JSON decode
- `fmt` — error formatting, Sprintf for API URL
- `io` — `io.ReadAll` for non-200 error body (same as `jira.go:51`, `azuredevops.go:48`)
- `net/http` — HTTP client
- `net/url` — URL parsing in `parseGitHubURL`
- `os` — `os.Getenv("GITHUB_TOKEN")`
- `path` — `path.Clean` to normalize trailing slash
- `strconv` — `strconv.Atoi` in `parseGitHubURL`, `strconv.Itoa` in `Fetch`

### Patterns to Follow (verified against actual code)

- `hostContains()` from `ticket.go:53–59` — call in `CanHandle`.
- HTTP flow from `azuredevops.go:34–68`: `http.NewRequest` → set headers → `http.DefaultClient.Do` → check `resp.StatusCode != 200` → `io.ReadAll` for error body → `json.NewDecoder(resp.Body).Decode`.
- Error prefix for fetch errors from `jira.go:46`: `fmt.Errorf("fetching Jira issue: %w", err)` → use `"fetching GitHub issue: %w"`.
- Error prefix for decode errors from `jira.go:69`: `fmt.Errorf("decoding Jira response: %w", err)` → use `"decoding GitHub response: %w"`.
- Generic API error from `jira.go:52`: `fmt.Errorf("Jira API returned %d: %s", resp.StatusCode, string(body))` → use `"GitHub API returned %d: %s"`.
- Token env-var error from `jira.go:33`: `fmt.Errorf("JIRA_API_TOKEN environment variable not set\nSet it with: export JIRA_API_TOKEN=your-token")`.
- `Ticket.URL = rawURL` — all three existing providers set this to the input URL, not the API response URL.
- `Ticket.ID` — string representation of identifier: Jira uses key string, ADO uses `fmt.Sprintf("%d", result.ID)`, Linear uses identifier string.

### Dependencies on Other Features

None. All infrastructure exists: `Provider` interface, `DetectProvider`, `init()` registry, CLI `ticket fetch` command.

---

## 3. Risk & Edge Cases

| Risk | Handling |
|---|---|
| `body` is JSON `null` | Decode into `Body *string`; use `""` when nil in `Fetch` |
| 404 without token (private repo) | `"GitHub issue not found — if this is a private repository, set GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token"` |
| 401 with bad/expired token | `"GitHub API returned 401 — check GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token"` |
| 403 rate limit (no token, public repo) | Same pattern as 401 — `"GitHub API returned 403 — check GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token"` |
| Issue number not an integer | `strconv.Atoi` in `parseGitHubURL`; return `fmt.Errorf("invalid GitHub issue number: %q", segments[4])` |
| Trailing slash in URL | `path.Clean` before splitting — strips trailing slash |
| PR URL (`/pull/`) passed | `CanHandle` checks `segments[3] == "issues"`; returns `false` for PR URLs |
| `owner`/`repo` with special path characters | Both values come from `url.Parse` path segments; apply `url.PathEscape` before embedding in API URL |
| Empty `owner` or `repo` | Validate non-empty in `parseGitHubURL`; return `fmt.Errorf("could not parse GitHub URL: missing owner or repo: %s", rawURL)` |
| `http.NewRequest` error | Return it directly — identical to all three existing providers |
| `http.DefaultClient.Do` network error | Return `fmt.Errorf("fetching GitHub issue: %w", err)` — matches `jira.go:46` pattern |
| Non-200/non-404/non-401/non-403 status | `io.ReadAll(resp.Body)` → `fmt.Errorf("GitHub API returned %d: %s", ...)` — matches `jira.go:52` |

### Security Considerations

- **No token logging:** The `Authorization` header value must never appear in error messages or logs. All existing providers maintain this; match the pattern. The `fetchIssue` function reads `GITHUB_TOKEN` but only interpolates it into the request header, never into returned errors.
- **URL injection:** `owner`, `repo`, and `number` are interpolated into the API URL path. `url.PathEscape` each segment before interpolation to prevent path traversal. `number` is already validated as a decimal integer by `strconv.Atoi` in `parseGitHubURL`, so it is safe without escaping, but `owner` and `repo` must be escaped.
- **Response size:** No limit imposed by existing providers; maintain consistency. Note as future hardening.
- **Token scope:** Document minimum: classic PAT with `repo` scope (private repos) or no scope (public); fine-grained PAT with `Issues: Read`.

### Performance

Single HTTP GET per invocation. Identical to existing providers.

---

## 4. Completeness Check

### Acceptance Criteria

1. `qode ticket fetch https://github.com/owner/repo/issues/42` writes `.qode/branches/{branch}/context/ticket.md` with format `# {Title}\n\n{Body}\n`.
2. Works for public repos with no `GITHUB_TOKEN` set.
3. Works for private repos when `GITHUB_TOKEN` is set with `repo` scope or `Issues: Read`.
4. 404 response returns actionable error mentioning `GITHUB_TOKEN` for private repos.
5. 401/403 response returns actionable error referencing `GITHUB_TOKEN`.
6. `CanHandle` returns `false` for non-`github.com` URLs (e.g., `linear.app`).
7. `CanHandle` returns `false` for `github.com` PR URLs (path contains `/pull/`).
8. `CanHandle` returns `false` for `github.com` URLs without `/issues/{number}` path.
9. `CanHandle` returns `true` only for valid `github.com/owner/repo/issues/{number}` URLs.
10. Unit tests pass for all scenarios in Task 4.
11. `docs/development-setup.md` created covering all four ticketing systems' env var setup.
12. `README.md` updated at three locations (see Task 6).
13. `internal/ticket/ticket.go` `init()` registers `&GitHubProvider{}`.
14. `internal/config/schema.go` line 63 comment includes `github`.

### Implicit Requirements

- Output format: `# {Title}\n\n{Body}\n` — matches `internal/cli/ticket.go:59`: `fmt.Sprintf("# %s\n\n%s\n", t.Title, t.Body)`.
- No new third-party dependencies — stdlib only.
- All functions ≤ 50 lines (project rule from CLAUDE.md).
- Error messages follow the `"ENV_VAR ... not set\nSet it with: export ..."` convention from `jira.go` and `azuredevops.go`.
- `Ticket.URL = rawURL` — consistent with all three existing providers.
- `Ticket.ID = strconv.Itoa(issue.Number)` — numeric string, consistent with `azuredevops.go:64`.
- `Ticket.Body = ""` when `issue.Body == nil` — not `"<nil>"` or any other representation.

### Explicitly Out of Scope

- GitHub Enterprise Server / custom domain.
- GitHub Pull Request URLs.
- Issue comments.
- Creating or updating GitHub Issues.
- OAuth/OIDC — static token via env var only.
- GraphQL API.

---

## 5. Actionable Implementation Plan

### Task 1 — `internal/ticket/github.go` (commit: `feat: add GitHub Issues ticket provider`)

Create the file with package `ticket`. Full type and function signatures:

```go
package ticket

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "path"
    "strconv"
    "strings"
)

type GitHubProvider struct {
    apiBase string // empty → "https://api.github.com"
}

type githubIssue struct {
    Number  int     `json:"number"`
    Title   string  `json:"title"`
    Body    *string `json:"body"`
    HTMLURL string  `json:"html_url"`
}
```

Methods:

**`Name()`** — returns `"github"`.

**`CanHandle(rawURL string) bool`** — calls `hostContains(rawURL, "github.com")`, then parses path to confirm `segments[3] == "issues"` and `segments[4]` is non-empty and numeric. Returns false for `/pull/` paths and bare repo URLs.

**`parseGitHubURL(rawURL string) (owner, repo, number string, err error)`** — uses `url.Parse`, `path.Clean`, splits path segments, validates length ≥ 5, validates `segments[3] == "issues"`, calls `strconv.Atoi(segments[4])` to validate numeric, applies `url.PathEscape` to owner and repo before returning. Returns descriptive errors for each validation failure.

**`fetchIssue(owner, repo, number string) (*githubIssue, error)`**:
```go
base := p.apiBase
if base == "" {
    base = "https://api.github.com"
}
apiURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s", base, owner, repo, number)
req, err := http.NewRequest("GET", apiURL, nil)
if err != nil { return nil, err }
req.Header.Set("Accept", "application/vnd.github+json")
req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
if token := os.Getenv("GITHUB_TOKEN"); token != "" {
    req.Header.Set("Authorization", "Bearer "+token)
}
resp, err := http.DefaultClient.Do(req)
if err != nil { return nil, fmt.Errorf("fetching GitHub issue: %w", err) }
defer resp.Body.Close()
if resp.StatusCode == 401 || resp.StatusCode == 403 {
    return nil, fmt.Errorf("GitHub API returned %d — check GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token", resp.StatusCode)
}
if resp.StatusCode == 404 {
    return nil, fmt.Errorf("GitHub issue not found — if this is a private repository, set GITHUB_TOKEN\nSet it with: export GITHUB_TOKEN=your-token")
}
if resp.StatusCode != 200 {
    body, _ := io.ReadAll(resp.Body)
    return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
}
var issue githubIssue
if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
    return nil, fmt.Errorf("decoding GitHub response: %w", err)
}
return &issue, nil
```

**`Fetch(rawURL string) (*Ticket, error)`**:
```go
owner, repo, number, err := parseGitHubURL(rawURL)
if err != nil { return nil, err }
issue, err := p.fetchIssue(owner, repo, number)
if err != nil { return nil, err }
body := ""
if issue.Body != nil { body = *issue.Body }
return &Ticket{
    ID:    strconv.Itoa(issue.Number),
    Title: issue.Title,
    Body:  body,
    URL:   rawURL,
}, nil
```

Note: `CanHandle` calls `parseGitHubURL` internally (or duplicates the path check inline) to avoid false positives; a clean approach is to have `CanHandle` call `parseGitHubURL` and return `err == nil`. This keeps path validation in one place.

### Task 2 — Register provider in `internal/ticket/ticket.go`

In `init()` at line 29, add `&GitHubProvider{}` after `&LinearProvider{}`:

```go
providers = []Provider{
    &JiraProvider{},
    &AzureDevOpsProvider{},
    &LinearProvider{},
    &GitHubProvider{},
}
```

**Depends on Task 1.**

### Task 3 — Update `internal/config/schema.go`

Line 63: change comment from:
```go
Type string `yaml:"type,omitempty"` // jira, azure-devops, linear, notion, manual
```
to:
```go
Type string `yaml:"type,omitempty"` // jira, azure-devops, linear, github, notion, manual
```

Independent of Task 1.

### Task 4 — `internal/ticket/github_test.go`

Use `net/http/httptest.NewServer` to mock the API. Inject via `GitHubProvider{apiBase: server.URL}`.

| Test function | What it verifies |
|---|---|
| `TestGitHubProvider_Name` | Returns `"github"` |
| `TestGitHubProvider_CanHandle` | Table-driven: valid URL → `true`; `/pull/` URL → `false`; `linear.app` URL → `false`; `github.com/owner/repo` (no issue) → `false`; URL with trailing slash → `true` |
| `TestParseGitHubURL_Valid` | Correct owner, repo, number extraction; verifies PathEscape applied |
| `TestParseGitHubURL_InvalidPath` | Returns non-nil error for `https://github.com/owner` (missing segments) |
| `TestParseGitHubURL_NonNumericNumber` | Returns non-nil error for `https://github.com/owner/repo/issues/abc` |
| `TestGitHubProvider_Fetch_Success` | Mock returns 200 JSON with body; verifies `Ticket.Title`, `Ticket.Body`, `Ticket.URL`, `Ticket.ID == "42"` |
| `TestGitHubProvider_Fetch_NullBody` | Mock returns `"body": null`; verifies `Ticket.Body == ""` |
| `TestGitHubProvider_Fetch_NotFound` | Mock returns 404; verifies error contains `"not found"` and `"GITHUB_TOKEN"` |
| `TestGitHubProvider_Fetch_Unauthorized` | Mock returns 401; verifies error contains `"401"` and `"GITHUB_TOKEN"` |
| `TestGitHubProvider_Fetch_Forbidden` | Mock returns 403; verifies error contains `"403"` and `"GITHUB_TOKEN"` |

**Depends on Task 1.**

### Task 5 — `docs/development-setup.md`

New file at `docs/development-setup.md` with sections:

1. **Prerequisites** — Go version from `go.mod`, git
2. **Ticketing System API Keys**
   - Jira: `JIRA_EMAIL` + `JIRA_API_TOKEN` — how to generate at `id.atlassian.com`; basic auth used by provider
   - Azure DevOps: `AZURE_DEVOPS_PAT` — PAT with Work Items (Read) scope; empty username basic auth
   - Linear: `LINEAR_API_KEY` — `linear.app/settings/api`; used directly as `Authorization` header
   - GitHub Issues: `GITHUB_TOKEN` — optional for public repos; classic PAT (`repo` for private) or fine-grained (`Issues: Read`); used as `Bearer` token
3. **Example `.env`** — with prominent `DO NOT COMMIT` warning
4. **Verification** — `qode ticket fetch` example command for each provider

Independent of Task 1.

### Task 6 — `README.md` (three locations)

**Location 1** — `README.md:127` commands table: change `"Fetch ticket (Jira, Azure DevOps, Linear)"` to `"Fetch ticket (Jira, Azure DevOps, Linear, GitHub)"`.

**Location 2** — `README.md:194–209` "Ticket System Setup" section: add GitHub example after Linear and add link to setup doc:
```markdown
# GitHub Issues (token optional for public repos)
export GITHUB_TOKEN=ghp_yourtoken
qode ticket fetch https://github.com/owner/repo/issues/42

For full environment setup see [Development Setup](docs/development-setup.md).
```

**Location 3** — `README.md:243` qode.yaml reference: change `# jira | azure-devops | linear | manual` to `# jira | azure-devops | linear | github | manual`.

Independent of Task 1.

### Dependency Graph

```
Task 1 (github.go)
  └── Task 2 (register provider)     [must follow Task 1]
  └── Task 4 (tests)                 [must follow Task 1]

Task 3 (schema.go comment)           [independent]
Task 5 (docs/development-setup.md)   [independent]
Task 6 (README.md, 3 locations)      [independent; logically after Task 5]
```

Recommended order: **1 → 4 → 2 → 3 → 5 → 6** (implement and test before registering; docs last).

---
*[DO NOT SCORE YOUR OWN OUTPUT — a separate judge will score this independently]*
