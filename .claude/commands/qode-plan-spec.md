# Generate Technical Specification — qode

Run this command and use its stdout output as your prompt:
  qode plan spec

If the output begins with `STOP.`, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use `qode plan spec --force` to bypass score gates when needed.

After generating the spec:
- Save it to: .qode/contexts/current/spec.md
- Suggest copying it to the ticket system for team review

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
5. Use `AskUserQuestion` to ask: "Post `.qode/contexts/current/spec.md` as a new ticket comment? (Yes / No) Note: publicly visible."
   - **Yes**: post via the selected MCP tool with body:
     ```
     **qode: plan-spec** | context: `<context-name>`

     <full contents of .qode/contexts/current/spec.md>
     ```
     If the call fails, report the error and stop.
   - **No**: end.
   - **Other (free text)**: execute as next prompt, then end.
