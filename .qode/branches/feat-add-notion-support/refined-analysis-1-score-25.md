<!-- qode:iteration=1 score=25/25 -->

# Requirements Refinement — Add Notion Ticket Support

## 1. Problem Understanding

### Restated Problem
Some teams manage their work in Notion databases (used as kanban boards with "tickets" stored as Notion pages). Currently, qode supports four ticket providers (Jira, Azure DevOps, Linear, GitHub Issues), but not Notion. Users on Notion-based teams must manually copy ticket content into `.qode/branches/{branch}/context/ticket.md`, which is slow, error-prone, and defeats the purpose of the automated `qode ticket fetch` workflow.

### User Need & Business Value
- **User need:** Run `qode ticket fetch <notion-url>` and have the ticket content automatically extracted and written to `context/ticket.md`, just like every other supported provider.
- **Business value:** Expands qode's reach to teams using Notion for project management — a popular choice especially in startups and smaller teams. Removes a friction point that could cause teams to abandon the qode workflow entirely.

### Ambiguities & Open Questions
1. **Notion URL formats:** Notion pages can appear as:
   - `https://www.notion.so/{workspace}/{page-title}-{page-id}` (32-hex-char ID at the end)
   - `https://www.notion.so/{page-id}` (bare ID)
   - `https://{workspace}.notion.site/{page-title}-{page-id}` (custom domain variant)
   - **Question:** Should we support all three URL formats? **Recommendation:** Yes, all three.
2. **Database item vs. standalone page:** A Notion "ticket" in a database is really a page within a database. The API treats both as pages. The same `GET /v1/pages/{page_id}` + `GET /v1/blocks/{page_id}/children` approach works for both. No distinction needed.
3. **What constitutes the ticket "title"?** For database items, the title is the `Title` property. For standalone pages, it may be in `properties` or as the first heading block. **Recommendation:** Use the `Title`-type property from the page properties (the Notion API marks exactly one property as `type: "title"`).
4. **What constitutes the ticket "body"?** The page body is stored as blocks (paragraphs, headings, lists, code blocks, etc.), not in a single field. **Recommendation:** Fetch block children via `GET /v1/blocks/{page_id}/children` and convert to plain text/markdown.
5. **Nested blocks:** Some Notion blocks (toggles, nested lists) have `has_children: true` and require recursive fetching. **Recommendation:** Implement one level of recursion (children of children) for the initial version; deeply nested content is uncommon in tickets.
6. **Rich text formatting:** Notion stores text with annotations (bold, italic, code, links). **Recommendation:** Convert to basic markdown (bold → `**text**`, etc.) to preserve readability in `ticket.md`.

## 2. Technical Analysis

### Affected Layers/Components

| Layer | File(s) | Changes |
|---|---|---|
| `internal/ticket/` | New `notion.go`, new `notion_test.go` | New `NotionProvider` implementing `Provider` interface |
| `internal/ticket/` | `ticket.go` | Register `&NotionProvider{}` in `init()` providers slice |
| `internal/config/` | `schema.go` | Already lists `notion` in the type comment — no code change needed |
| `internal/cli/` | No changes | `ticket.go` and `help.go` are generic; they work with any registered provider |
| `internal/ide/` | `claudecode.go` | No changes needed — slash command is a generic passthrough |
| `docs/` | `how-to-use-ticket-fetch.md` | Add Notion section with env var, example, token generation link |
| `docs/` | `qode-yaml-reference.md` | Add `notion` row to `ticket_system.type` table |
| `README.md` | Top-level README | Add Notion to supported providers list |
| `CLAUDE.md` | Project instructions | Add Notion to ticket provider references if present |

### Key Technical Decisions

1. **API Approach:** Two API calls per fetch:
   - `GET https://api.notion.com/v1/pages/{page_id}` — to get the page title from properties.
   - `GET https://api.notion.com/v1/blocks/{page_id}/children` — to get page body content.
   - Pagination via `start_cursor` / `has_more` for block children if the page has many blocks.

2. **Authentication:** Notion uses integration tokens (Bearer auth).
   - Env var: `NOTION_API_KEY` (consistent naming with `LINEAR_API_KEY`).
   - Required header: `Authorization: Bearer {token}`.
   - Required header: `Notion-Version: 2022-06-28` (stable API version).

3. **URL Parsing:** Extract the 32-character hex page ID from the URL. The ID is always the last 32 hex characters (possibly hyphenated as a UUID). Strategy:
   - Strip hyphens from the last path segment.
   - Extract the trailing 32 hex characters.
   - Format as UUID (8-4-4-4-12) for the API call.

4. **Block-to-Markdown Conversion:** Convert Notion block types to markdown:
   - `paragraph` → plain text with newline
   - `heading_1/2/3` → `#/##/###`
   - `bulleted_list_item` → `- `
   - `numbered_list_item` → `1. ` (sequential numbering)
   - `to_do` → `- [ ]` / `- [x]`
   - `code` → fenced code block with language
   - `quote` → `> `
   - `divider` → `---`
   - `toggle` → treat as paragraph (fetch children if `has_children`)
   - Other types → skip silently (images, embeds, etc. are not useful in text-only ticket.md)

5. **Rich text rendering:** Each block contains `rich_text` array entries with `plain_text`, `href`, and `annotations` (bold, italic, strikethrough, code). Convert to markdown inline formatting.

### Patterns to Follow

- **File structure:** One file per provider (`notion.go`) plus test file (`notion_test.go`), matching `github.go` / `github_test.go`.
- **Struct pattern:** `type NotionProvider struct { apiBase string }` — injectable base URL for testing, matching `GitHubProvider`.
- **Test pattern:** Use `httptest.NewServer` for mock API responses, matching `github_test.go`.
- **Error messages:** Reference `NOTION_API_KEY` in auth errors, matching the pattern in `github.go` (lines 101-105).
- **Use shared `httpClient`:** Reuse the package-level `httpClient` from `ticket.go:13`.
- **`CanHandle` simplicity:** URL host check via `hostContains(rawURL, "notion.so") || hostContains(rawURL, "notion.site")`.

### Dependencies

- No external Go dependencies needed — only `net/http`, `encoding/json`, `net/url`, `regexp`, `strings`, `fmt`.
- Notion API is a REST API with JSON responses — no special SDK required.
- User must create a Notion integration at https://www.notion.so/my-integrations and share the target page/database with the integration.

## 3. Risk & Edge Cases

### What Could Go Wrong

1. **Integration not connected to page:** Notion requires the integration to be explicitly shared on the page/database. If not, the API returns 404. Must provide a clear error message explaining this.
2. **Rate limiting:** Notion API has a rate limit of 3 requests/second. For a single ticket fetch (2 requests), this is unlikely to be hit. No special handling needed.
3. **Large pages:** A Notion page could have hundreds of blocks. The block children endpoint paginates (default 100 per page). Must handle pagination to avoid truncated output.
4. **Unsupported block types:** New block types may be added to the Notion API over time. Must handle unknown types gracefully (skip rather than error).

### Edge Cases

1. **Page with no body content:** Return empty body string (same as GitHub's null body handling).
2. **Page with only a title property and no other content:** Valid — return title with empty body.
3. **URL with query parameters or fragments:** Strip query params and fragments before extracting page ID.
4. **URL with trailing slash:** Handle via path cleaning (same as GitHub provider).
5. **Page ID in UUID format (with hyphens) vs. raw 32 hex chars:** Support both by stripping hyphens before validation.
6. **Workspace subdomains:** URLs like `https://myteam.notion.site/Page-Title-abc123...` must be supported alongside `notion.so`.
7. **Nested blocks (children of children):** Fetch one level deep for toggle blocks and callouts that commonly contain ticket details.

### Security Considerations

- **Token handling:** `NOTION_API_KEY` is read from environment (loaded via `.env`), never logged or included in error messages beyond "check NOTION_API_KEY".
- **URL validation:** Validate that the host is `notion.so` or `notion.site` before making API calls to prevent SSRF.
- **Response size:** Use `io.LimitReader` on API responses (matching patterns in other providers) to prevent memory exhaustion from malicious responses.

### Performance Implications

- Two HTTP requests per fetch (page metadata + block children). Comparable to or faster than Jira (1 request) and Linear (1 GraphQL request).
- Pagination adds latency for very large pages, but ticket pages are typically small.
- No caching needed — this is a one-shot fetch operation.

## 4. Completeness Check

### Acceptance Criteria

1. `qode ticket fetch https://www.notion.so/workspace/My-Ticket-{page_id}` fetches the page title and body and writes `context/ticket.md`.
2. `qode ticket fetch https://notion.so/{page_id}` works with bare ID URLs.
3. `qode ticket fetch https://myteam.notion.site/Page-{page_id}` works with custom domain URLs.
4. Authentication via `NOTION_API_KEY` environment variable (loaded from `.env`).
5. Clear error message when `NOTION_API_KEY` is not set.
6. Clear error message when the integration doesn't have access to the page (404).
7. Block content is converted to readable markdown in `ticket.md`.
8. Pagination is handled for pages with many blocks.
9. `qode.yaml` `ticket_system.type: notion` is documented.
10. `docs/how-to-use-ticket-fetch.md` includes Notion section.
11. `docs/qode-yaml-reference.md` includes `notion` in the type table.
12. `README.md` lists Notion as a supported provider.
13. Unit tests cover: `Name()`, `CanHandle()` (positive + negative), `Fetch()` (success, null body, auth errors, not found, server error), URL parsing, block-to-markdown conversion.

### Implicit Requirements

- The `/qode-ticket-fetch` slash command (in all IDEs) must work with Notion URLs without any changes — this is already the case since it's a generic passthrough to `qode ticket fetch $ARGUMENTS`.
- The `CLAUDE.md` ticket system section (generated by `claudecode.go:88-93`) will automatically include `notion` when `ticket_system.type: notion` is configured — no code change needed.
- The workflow diagram in `help.go` is provider-agnostic — no change needed.

### Explicitly Out of Scope

- Fetching comments or activity from Notion pages.
- Writing back to Notion (e.g., updating ticket status).
- Supporting Notion databases as a whole (listing all tickets) — only individual page fetch.
- OAuth-based authentication (Notion supports it, but env-var-based internal integration tokens are simpler and consistent with other providers).
- Rendering images, files, or embedded content from Notion blocks.

## 5. Actionable Implementation Plan

### Task Breakdown (in order)

**Task 1: Implement `NotionProvider` core** (`internal/ticket/notion.go`)
- Define `NotionProvider` struct with `apiBase string` field.
- Implement `Name() string` → `"notion"`.
- Implement `CanHandle(rawURL string) bool` — match `notion.so` and `notion.site` hosts, validate page ID extraction.
- Implement `parseNotionURL(rawURL string) (pageID string, err error)` — extract and validate 32-hex-char page ID from URL path.
- Implement `Fetch(rawURL string) (*Ticket, error)`:
  - Read `NOTION_API_KEY` from env.
  - Call `GET /v1/pages/{page_id}` to get title.
  - Call `GET /v1/blocks/{page_id}/children` (with pagination) to get body blocks.
  - Convert blocks to markdown string.
  - Return `Ticket{ID: pageID, Title: title, Body: markdown, URL: rawURL}`.
- Implement block-to-markdown conversion helpers.

**Task 2: Register provider** (`internal/ticket/ticket.go`)
- Add `&NotionProvider{}` to the `providers` slice in `init()`.

**Task 3: Write unit tests** (`internal/ticket/notion_test.go`)
- Test `Name()`.
- Test `CanHandle()` with valid/invalid URLs (all URL formats).
- Test `parseNotionURL()` with various formats.
- Test `Fetch()` success path with mock HTTP server.
- Test `Fetch()` error paths: missing token, 404, 401, 403, 500.
- Test block-to-markdown conversion for each supported block type.
- Test pagination handling.

**Task 4: Update documentation**
- `docs/how-to-use-ticket-fetch.md` — Add "## Notion" section with env var, example URLs, integration setup link.
- `docs/qode-yaml-reference.md` — Add `notion` row: `| notion | NOTION_API_KEY |`.
- `README.md` — Add Notion to supported providers list.

**Task 5: Run quality gates**
- `go test ./internal/ticket/...` — ensure all tests pass.
- `go vet ./...` and `golangci-lint run` — no lint issues.
- `go build ./...` — clean build.

### Prerequisites
- None — the provider interface and registration mechanism already exist. This is a purely additive change.
