# Code Review — Add Notion Ticket Support

## Issues Found

### Issue 1
- **Severity:** Medium
- **File:** `internal/ticket/notion.go:241`
- **Issue:** The `start_cursor` value is appended to the URL without URL-encoding. If the Notion API ever returns a cursor with special characters (`&`, `=`, `+`), the request URL would be malformed.
- **Suggestion:** Use `url.Values` to build query parameters:
```go
params := url.Values{}
params.Set("page_size", "100")
if cursor != "" {
    params.Set("start_cursor", cursor)
}
apiURL := fmt.Sprintf("%s/v1/blocks/%s/children?%s", base, blockID, params.Encode())
```

### Issue 2
- **Severity:** Low
- **File:** `internal/ticket/notion.go:28-38`
- **Issue:** `CanHandle` uses `strings.HasSuffix(host, ".notion.site")` which allows any subdomain (e.g., `evil.notion.site`). While this is intentional for workspace subdomains, the existing knowledge base lesson says to prefer exact host matches. However, Notion's design requires subdomain support (`{workspace}.notion.site`), so this is an acceptable deviation — just noting it.
- **Suggestion:** No change needed. The subdomain pattern is inherent to Notion's URL design.

### Issue 3
- **Severity:** Low
- **File:** `internal/ticket/notion.go:349-372`
- **Issue:** `blocksToMarkdown` is 23 lines, well within the 50-line limit. However, the empty paragraph handling (`if line == "" && b.Type != "divider"`) silently drops empty paragraphs. In Notion, empty paragraphs serve as visual spacers. Dropping them may make the output slightly different from the Notion page layout.
- **Suggestion:** Acceptable trade-off for a ticket-fetching context where spacing is less critical. No change needed.

### Issue 4
- **Severity:** Nit
- **File:** `internal/ticket/notion.go:108`
- **Issue:** The parameter name `hex` in `formatNotionUUID(hex string)` shadows the built-in `encoding/hex` package import (not currently imported, but could cause confusion if the import is added later).
- **Suggestion:** Rename to `hexStr` or `rawID` for clarity:
```go
func formatNotionUUID(rawID string) string {
```

### Issue 5
- **Severity:** Nit
- **File:** `internal/ticket/notion.go:278`
- **Issue:** `setHeaders` uses a value receiver `*NotionProvider` but doesn't use any fields from the receiver. It could be a standalone function for consistency, but the method form is also fine since it groups Notion-related logic.
- **Suggestion:** No change needed — method grouping is reasonable.

## Summary

| Severity | Count |
|---|---|
| Critical | 0 |
| High | 0 |
| Medium | 1 |
| Low | 2 |
| Nit | 2 |

### Overall Code Quality Assessment

The implementation is clean, well-structured, and follows existing codebase patterns closely. The `NotionProvider` matches the `GitHubProvider` pattern (injectable `apiBase` for testing, `httptest.NewServer` mocks, explicit error messages referencing env vars). The block-to-markdown conversion handles all common block types and gracefully skips unknown ones. Tests are comprehensive with 17 test functions covering positive paths, all error codes, pagination, nested blocks, rich text rendering, and edge cases.

### Top 3 Things to Fix Before Merging

1. **URL-encode the `start_cursor` query parameter** (Medium) — prevents potential URL malformation with unusual cursor values.
2. All other issues are Low/Nit severity and are acceptable to merge as-is.
3. N/A — the implementation is solid.

## Rating

| Dimension | Score (0-2) | Justification |
|-----------|-------------|---------------|
| Correctness | 2 | Implements spec correctly. URL parsing, API calls, pagination, block conversion, and error handling all work as specified. Edge cases are handled. |
| Code Quality | 2 | Functions are well-sized (all under 50 lines), names are clear, no duplication. Follows existing provider patterns exactly. |
| Architecture | 2 | Clean separation: URL parsing, API calls, content extraction, and markdown conversion are distinct functions. Provider interface is correctly implemented. Registration is a one-line change. |
| Error Handling | 2 | All HTTP status codes have specific, helpful error messages. Missing env var detected early. Response bodies are size-limited. Errors propagate correctly. |
| Testing | 2 | 17 tests cover Name, CanHandle (positive + negative), URL parsing (valid + invalid), Fetch (success, empty body, all error codes, auth header, pagination, nested blocks, custom title), block conversion, rich text rendering, and unknown block types. |

**Total Score: 10.0/10**
