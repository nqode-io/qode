# Code Review — #27 Replace qode ticket fetch with MCP server integration

**Branch:** 27-implement-mcp-connections
**Reviewer:** qode judge pass

---

## Incident Report (pre-read)

Post-mortem: A developer runs `/qode-ticket-fetch` on a Jira ticket that has a linked Figma design. The AI calls the MCP server, fetches the ticket, but the output file `context/ticket.md` only has the title and body — the Figma design link is missing. Six sprints later the spec omits a critical layout constraint described only in that Figma frame. Root cause: the prompt didn't instruct the AI to follow linked resources, or the output file path was wrong.

Now reading the diff.

---

## File-by-file Review

### `internal/scaffold/claudecode.go` — `ticketFetchClaudeCommand`

**Verified safe:**
1. `$ARGUMENTS` is present on line 1 of the returned string — confirmed the AI receives the ticket URL from the user's command invocation.
2. The string does not start with `!` — verified; Claude Code will treat it as an AI prompt, not a shell command. `TestClaudeSlashCommands_TicketFetchNoExclamationPrefix` guards this.
3. All three output file paths are specified: `context/ticket.md`, `context/ticket-comments.md`, `context/ticket-links.md` — the AI has unambiguous write targets.

**Issues:**

- **Severity:** Medium
- **File:** [internal/scaffold/claudecode.go](internal/scaffold/claudecode.go) and [internal/scaffold/cursor.go](internal/scaffold/cursor.go)
- **Issue:** The 18-line MCP steps block (steps 1–5) is duplicated verbatim between `ticketFetchClaudeCommand` and `ticketFetchCursorCommand`. When output file paths change, or a new step is added (e.g. attachments), both functions must be edited in sync. There is no test that would catch a divergence between the two prompts.
- **Suggestion:** Extract the shared body to a package-level constant or helper:
  ```go
  const ticketFetchMCPSteps = `
  **Steps:**
  1. Use the MCP tool ...
  ...
  5. Report a one-line summary: "Fetched: <title> — <N> comments, <M> links."
  `

  func ticketFetchClaudeCommand(name string) string {
      return fmt.Sprintf("# Fetch Ticket via MCP — %s\n\nFetch the ticket...%s", name, ticketFetchMCPSteps)
  }
  ```

---

### `internal/scaffold/cursor.go` — `ticketFetchCursorCommand`

**Verified safe:**
1. `description:` YAML frontmatter is present — Cursor requires this for slash command registration.
2. Project `name` is interpolated into the description field — tested by `TestCursorSlashCommands_ContainsTicketFetch`.
3. Fallback instruction ("If no MCP server is available for a linked resource, record its URL and title only") is present — the AI degrades gracefully when, e.g., a Figma MCP is not configured.

---

### `internal/scaffold/scaffold_test.go`

**Verified safe:**
1. `TestClaudeSlashCommands_TicketFetchNoExclamationPrefix` is a correct regression test for the most critical invariant (no `!` prefix).
2. `TestClaudeSlashCommands_ContainsTicketFetch` asserts `["$ARGUMENTS", "context/ticket.md", "MCP", "Figma"]` — comprehensive.
3. `TestCursorSlashCommands_ContainsTicketFetch` asserts `["$ARGUMENTS", "context/ticket.md", "description:", "testproject", "Figma"]` — comprehensive.

**Issues:**

- **Severity:** Low
- **File:** [internal/scaffold/scaffold_test.go](internal/scaffold/scaffold_test.go)
- **Issue:** `TestSetupClaudeCode_WritesTicketFetchCommand` and `TestSetupCursor_WritesTicketFetchCommand` assert `["$ARGUMENTS", "context/ticket.md", "MCP"]` but do not assert that linked-resource instructions (`"Figma"`) are present in the written files. If `ticketFetchClaudeCommand` accidentally omits the linked-resource step, the in-memory map test (`ContainsTicketFetch`) would catch it but the file-write test would not. The file-write tests exercise a different code path (file I/O) and should assert the same invariants.
- **Suggestion:** Add `"Figma"` to the assertion slices in `TestSetupClaudeCode_WritesTicketFetchCommand` and `TestSetupCursor_WritesTicketFetchCommand`.

- **Severity:** Low
- **File:** [internal/scaffold/scaffold_test.go:29-33](internal/scaffold/scaffold_test.go#L29-L33)
- **Issue:** `func min(a, b int) int` is defined as a package-level function. Go 1.21+ (this project is on Go 1.24.5) introduced a built-in `min`. The local definition shadows the built-in, is dead weight, and will produce a `redeclared` warning from some linters. It is also only used once: `content[:min(len(content), 40)]`.
- **Suggestion:** Remove the helper and inline the guard:
  ```go
  n := 40
  if len(content) < n {
      n = len(content)
  }
  t.Errorf("qode-ticket-fetch must not start with '!', got: %q", content[:n])
  ```

---

### `internal/config/schema.go`

**Verified safe:**
1. `TicketSystemConfig` and `AuthConfig` are deleted — confirmed no remaining consumers via `go build ./...` passing.
2. `yaml.v3` silently ignores unknown YAML keys — existing `qode.yaml` files with `ticket_system:` will not error after upgrade.
3. `Config` struct fields compile cleanly.

**Issues:**

- **Severity:** Nit
- **File:** [internal/config/schema.go:4-11](internal/config/schema.go#L4-L11)
- **Issue:** After removing `TicketSystem`, `QodeVersion` and `Review` were realigned to a narrower width, but `Scoring`, `IDE`, `Knowledge`, `Branch` retained their old wider alignment. The struct now has two distinct alignment groups.
- **Suggestion:** Align all fields consistently (either use `gofmt`'s natural width or align all to the longest field name `QodeVersion`).

---

### `internal/cli/root.go`, `branch.go`, `init.go`, `help.go`

**Verified safe:**
1. `newTicketCmd()` is removed from root — `qode ticket fetch` no longer appears in `qode --help`.
2. `workflowList` constant in `help.go` now shows only `/qode-ticket-fetch <url>` for step 2 — the CLI-first line is removed.
3. Branch stub `ticket.md` now says "use /qode-ticket-fetch <url> in your IDE" — consistent with the new model.
4. `init.go` next-steps message updated — no stale CLI reference.

No issues in these files. All four user-facing references to `qode ticket fetch` have been updated consistently.

---

### `internal/ticket/` (deleted) and `internal/cli/ticket.go` (deleted)

**Verified:**
- Directory and file deleted. `go build ./...` passes — no orphaned imports.
- `~500 LOC of credential-handling Go code removed from the attack surface. Positive finding.

---

### `.claude/commands/qode-ticket-fetch.md` and `.cursor/commands/qode-ticket-fetch.mdc` (regenerated)

**Verified:**
1. Claude Code file starts with `# Fetch Ticket via MCP` (no `!`) — correct.
2. Cursor file starts with `---\ndescription:` frontmatter — correct.
3. Both files contain `$ARGUMENTS`, `context/ticket.md`, linked-resource instructions.

---

### `docs/how-to-use-ticket-fetch.md` (rewritten)

**Verified safe:**
1. All five ticketing systems (GitHub, Jira, Linear, Azure DevOps, Notion) have dedicated sections with install commands, auth requirements, and Claude Code + Cursor config snippets.
2. All five linked-resource services (Figma, Google Docs, SharePoint, Confluence, Miro) have sections. draw.io is omitted (no official MCP server — per spec).
3. Both Claude Code (`claude mcp add`) and Cursor (`mcp.json`) configurations are shown for each service — actionable for both IDE users.
4. Environment variable and token-scope requirements are documented per service.

---

## Summary

| Severity | Count |
|---|---|
| Critical | 0 |
| High | 0 |
| Medium | 1 |
| Low | 2 |
| Nit | 1 |

**Top 3 before merging:**

1. **M1 (prompt duplication):** Extract the 18-line MCP steps to a shared constant in `internal/scaffold/`. When the prompt evolves, a single-file edit is safer than keeping two copies in sync.
2. **L1 (file-write test gap):** Add `"Figma"` assertion to `TestSetupClaudeCode_WritesTicketFetchCommand` and `TestSetupCursor_WritesTicketFetchCommand` so the file-write path is as thoroughly verified as the map path.
3. **L2 (`min` helper):** Remove the redundant `min(a, b int)` function that shadows the Go 1.21+ built-in.

**Overall assessment:** The implementation is correct and complete — all spec tasks are done, build and tests pass, docs are thorough. The prompt duplication is the only architectural concern worth addressing before merge since it will make future prompt evolution error-prone. The test gaps are minor but worth closing given that this is the most user-facing part of the change.

---

## Rating

| Dimension | Score | What you verified |
|---|---|---|
| Correctness (0–3) | 3 | All spec tasks implemented: 8 files deleted, schema types removed, both scaffold helpers added with correct content (no `!` prefix, `$ARGUMENTS` present, all 3 output paths named, linked-resource fallback present), all user-facing CLI strings updated, docs rewritten |
| CLI Contract (0–2) | 2 | `qode ticket fetch` removed from root, help workflow list, branch stub, init next-steps. `workflowList` shows only `/qode-ticket-fetch`. Claude Code prompt invariants verified by test + manual inspection |
| Go Idioms & Code Quality (0–2) | 1.5 | Functions are < 50 lines, named helpers follow existing pattern. Deducted 0.5 for 18-line verbatim duplication between `ticketFetchClaudeCommand` and `ticketFetchCursorCommand` — divergence risk with no test guard |
| Error Handling & UX (0–2) | 2 | No errors to handle in string-returning functions. UX: all four CLI output locations updated consistently; fallback behaviour (URL-only for unconfigured MCP) documented in both prompts |
| Test Coverage (0–2) | 1.5 | New `TestClaudeSlashCommands_TicketFetchNoExclamationPrefix` correctly guards the critical `!` regression. In-memory map tests are thorough. Deducted 0.5 for file-write tests missing `"Figma"` assertion + redundant `min` helper shadowing built-in |
| Template Safety (0–1) | 1 | `name` from `filepath.Base(root)` cannot contain newlines on Linux/macOS/Windows — prompt injection via directory name is not feasible. `$ARGUMENTS` passed directly to AI at the correct boundary |

**Total Score: 11.0/12**
**Minimum passing score: 10/12** ✅
