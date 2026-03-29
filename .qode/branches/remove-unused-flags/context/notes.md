# Notes

Currently we are in an alpha stage, no external users for now, so removing functionalities without worrying about breaking changes and backward ccompatibility is okay - ONLY FOR NOW.

## Decision: qode init — always overwrite managed files (2026-03-29)

`qode init` and `qode ide setup` should treat all qode-managed files as owned by qode and always overwrite them on every run. Users who want custom variants should maintain them separately.

**Scope:**
- `.qode/prompts/*.md.tmpl` — always overwrite (currently skips if file exists)
- `.claude/commands/*.md` (Claude Code slash commands) — always overwrite (already the case)
- `.cursor/commands/*.mdc` (Cursor slash commands) — always overwrite (already the case)

**`CLAUDE.md` — never generate or touch**
qode should not generate or modify `CLAUDE.md`. This is a project-specific file the developer owns entirely. Generating it caused confusion because qode would silently not update it once created.
