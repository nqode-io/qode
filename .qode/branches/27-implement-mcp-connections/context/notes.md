# Notes

Add any additional context, decisions, or open questions here.

## Requirements update (2026-04-02)

- **Always use MCP** for ticket fetch. No `mode` field, no API fallback, no two-phase approach.
- This is alpha stage — breaking changes are expected. No backward compatibility concerns.
- `qode ticket fetch` CLI command is deleted entirely. Users must use `/qode-ticket-fetch` in their IDE via MCP.
- The slash commands always emit MCP prompts (no conditional logic based on a mode config field).
- **Documentation**: instruct users to always use official MCP servers where available. Add setup instructions for each supported ticket system's official MCP server into the docs.
- Single-phase implementation — no Phase 2 deferred work.

## Additional instructions (2026-04-02)

- `qode ide sync` is deprecated — use `qode init` to regenerate IDE configs (e.g. after updating scaffold templates).
- When the slash command reads a ticket and finds links to other services that have MCP servers configured, the AI should use those MCP tools to fetch the linked content as well. Examples: Figma designs, draw.io diagrams, Google Docs, SharePoint documents, Confluence pages, Miro boards, etc. The prompt must instruct the AI to follow linked resources using whatever MCP tools are available for those services, not just the ticketing system MCP server.
