# Security Review — qode

Run this command and use its stdout output as your prompt:
  qode review security

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use `qode review security --force` to bypass the uncommitted-diff check.

After completing the review:
- Save to: .qode/contexts/current/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue

## Post Step to Ticket (Optional)

1. Read `.qode/contexts/current/ticket.md`. If absent or no line matching `**URL:** <url>` is found, notify the user "No ticket URL found — skipping ticket comment." and skip the remaining steps.
2. Extract the URL and select the MCP comment tool:
   - `https://github.com/*/issues/*` → use tool `mcp__github__add_issue_comment`
   - `https://*.atlassian.net/browse/*` → use tool `addCommentToJiraIssue`
   - `https://linear.app/*/issue/*` → use tool `create_comment`
   - `https://dev.azure.com/*/_workitems/*` → use tool `mcp_ado_wit_add_work_item_comment`
   - `https://www.notion.so/*` → use tool `create-a-comment`
   - Unrecognised URL → skip silently
3. If the required MCP tool is not available in your tool list, skip silently.
4. Read `.qode/contexts/current/.ctx-name.md` for the context name.
5. Ask: "Post `.qode/contexts/current/security-review.md` as a new ticket comment? Yes or No. (NOTE: THIS MIGHT BE PUBLICLY VISIBLE)"
   - **Yes**: post via the selected MCP tool with body:
     ```
     **qode: review-security** | context: `<context-name>`

     <full contents of .qode/contexts/current/security-review.md>
     ```
     If the call fails, report the error and stop.
   - **No**: end.
   - **Other (free text)**: execute as next prompt, then end.
