# Code Review — qode (feat-github-issue-support)

You are a senior software engineer performing a code review.
Review the diff below objectively.

## Project Context

**Project:** qode
**Branch:** feat-github-issue-support
**Tech Stack:**
- **default** (go) at `.`

---

## Issues Found

### 1. Medium — False "copied to clipboard" message on silent clipboard failure

- **Severity:** Medium
- **File:** `internal/cli/plan.go:126-133`
- **Issue:** The `else` branch of `if err := copyToClipboard(...); err != nil && flagVerbose` is entered in two cases: (a) copy succeeded (`err == nil`) and (b) copy failed but verbose mode is off (`err != nil && !flagVerbose`). In case (b), the message `"Worker prompt copied to clipboard."` is printed to stdout even though the copy failed. A user on a system without a clipboard tool (e.g., a headless CI server running `--prompt-only`) would see a false confirmation.

  Current code:
  ```go
  if err := copyToClipboard(out.WorkerPrompt); err != nil && flagVerbose {
      fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
  } else {
      fmt.Println("Worker prompt copied to clipboard.")
  }
  ```

- **Suggestion:** Split the error check and the confirmation message:
  ```go
  if err := copyToClipboard(out.WorkerPrompt); err != nil {
      if flagVerbose {
          fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
      }
  } else {
      fmt.Println("Worker prompt copied to clipboard.")
  }
  ```

---

### 2. Low — Magic numbers `30` and `512` violate project clean code rules

- **Severity:** Low
- **Files:** `internal/ticket/ticket.go:13`, all four provider files
- **Issue:** The project's clean code rules state "No magic numbers — use named constants." The HTTP client timeout (`30 * time.Second`) and the error body size limit (`512`) are used as literals across multiple files without named constants. If these values ever need to change, all call sites must be updated manually.

- **Suggestion:** Add named constants to `ticket.go`:
  ```go
  const (
      httpTimeout       = 30 * time.Second
      errBodyMaxBytes   = 512
  )

  var httpClient = &http.Client{Timeout: httpTimeout}
  ```
  Then replace `512` with `errBodyMaxBytes` in all four `io.LimitReader` calls.

---

### 3. Low — No `CanHandle` test case for GitHub subdomains

- **Severity:** Low
- **File:** `internal/ticket/github_test.go:19-39`
- **Issue:** The Medium fix in the previous review tightened `CanHandle` to require `u.Host == "github.com"` (exact match) specifically to prevent false positives on subdomains like `api.github.com` and `gist.github.com`. The `TestGitHubProvider_CanHandle` table does not include a test case for `api.github.com`, so a regression would not be caught.

- **Suggestion:** Add two cases to the table:
  ```go
  {"https://api.github.com/repos/owner/repo/issues/42", false},
  {"https://gist.github.com/owner/repo/issues/42", false},
  ```

---

### 4. Nit — `for name, content := range stubs` shadows outer `name` (pre-existing)

- **Severity:** Nit
- **File:** `internal/cli/branch.go:71`
- **Issue:** The range variable `name` in `for name, content := range stubs` shadows the outer `name := args[0]`. This was noted in the previous review. The outer `name` is correctly restored after the loop (Go scope rules), so there is no functional bug, but the code is a readability hazard.

- **Suggestion:** Rename the loop variable: `for filename, content := range stubs`.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 1 |
| Low | 2 |
| Nit | 1 |

**Overall assessment:** The security and correctness fixes in this iteration are well-implemented. The Jira SSRF fix (`extractJiraBase` now returns an error and validates scheme + allowed hosts) is correct and consistent with `CanHandle`. The path-traversal fix in `branch.go` correctly uses `filepath.Rel` and is applied to all three subcommands, with the most critical ordering — `safeBranchDir` before `os.RemoveAll` in the remove command — done correctly. The GitHub provider's `CanHandle` host exact match, shared `httpClient` with timeout, `io.LimitReader` on error paths, and `url.PathEscape` for Azure DevOps are all clean, minimal, and correctly placed. The main gap is the clipboard feedback logic bug in `plan.go`, which is a behavioral correctness issue (false confirmation to user) that should be fixed before merge.

**Top 3 things to fix before merging:**
1. **(Medium)** Fix the `else` branch logic in `refinePromptOnly` so the clipboard confirmation is not printed when the copy failed silently.
2. **(Low)** Extract `30` and `512` as named constants (`httpTimeout`, `errBodyMaxBytes`) to satisfy the project's clean code rules.
3. **(Low)** Add `api.github.com` and `gist.github.com` cases to `TestGitHubProvider_CanHandle` to regression-test the Medium fix.

---

## Rating

| Dimension | Score (0-2) | Justification |
|-----------|-------------|---------------|
| Correctness | 1.5 | Core logic correct; behavioral bug in `refinePromptOnly` (false clipboard confirmation on silent failure) |
| Code Quality | 1.5 | Clean, idiomatic Go, follows patterns; `30` and `512` magic literals violate project rules |
| Architecture | 2 | `safeBranchDir` and `httpClient` placed correctly; all changes additive; no layer violations |
| Error Handling | 2 | SSRF, path traversal, LimitReader, and timeout all implemented correctly; `extractJiraBase` error propagation clean |
| Testing | 1.5 | 13 tests covering all specified scenarios; missing subdomain cases for the host-exact-match fix |

**Total Score: 8.5/10**
