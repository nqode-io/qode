### Allow subdomain matching in CanHandle for multi-tenant hosts

The "use exact host match" lesson applies to single-domain providers (e.g., `github.com`), but some services use workspace subdomains (e.g., Notion's `{workspace}.notion.site`). For these, use `strings.HasSuffix(host, ".notion.site")` or equivalent after stripping `www.` prefix. Always add negative test cases for lookalike domains (`fake-notion.so`, `notion.so.evil.com`) to ensure the suffix match doesn't create false positives.

**Example 1:** Subdomain-aware host matching
```go
host := strings.TrimPrefix(u.Host, "www.")
if host == "notion.so" {
    return true
}
return host == "notion.site" || strings.HasSuffix(host, ".notion.site")
```

**Example 2:** Negative tests for lookalike domains
```go
{"https://fake-notion.so/page", false},
{"https://notion.so.evil.com/page", false},
```
