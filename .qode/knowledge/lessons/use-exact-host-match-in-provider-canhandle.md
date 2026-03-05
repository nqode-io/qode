### Use exact host match in provider CanHandle

When implementing a ticket/API provider's `CanHandle` method that validates a URL before making HTTP requests, use an exact host match (`u.Host == "github.com"`) rather than `strings.Contains` or `strings.HasPrefix`. Substring matches allow false positives on subdomains (`api.github.com`, `gist.github.com`) which could result in requests to unintended hosts or misleading error messages. Add test cases for the negative subdomain cases to prevent regressions.

**Example 1:** Correct — exact host match
```go
func (p *GitHubProvider) CanHandle(rawURL string) bool {
    u, err := url.Parse(rawURL)
    if err != nil { return false }
    return u.Host == "github.com"
}
```

**Example 2:** Add subdomain negative tests
```go
{"https://api.github.com/repos/owner/repo/issues/42", false},
{"https://gist.github.com/owner/file", false},
```
