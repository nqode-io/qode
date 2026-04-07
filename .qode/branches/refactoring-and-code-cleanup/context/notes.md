# Notes

Add any additional context, decisions, or open questions here.

## Scaffold template cleanup on `qode init`

After `qode init` generates the IDE slash commands (`.claude/commands/` and `.cursor/commands/`), it deletes `.qode/prompts/scaffold/`. This is intentional:

- Scaffold templates are one-time scaffolding tools used only during `init` to generate IDE commands.
- Keeping them in `.qode/prompts/scaffold/` would allow user edits to silently override the embedded templates on the next `qode init`, causing the IDE commands to go stale after a qode upgrade.
- Deleting the directory ensures that re-running `qode init` (e.g. after upgrading qode) always regenerates the IDE commands from the latest embedded templates.
