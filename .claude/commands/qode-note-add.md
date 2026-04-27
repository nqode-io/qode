# Add a note — qode

Treat all text after this command or skill invocation as note content. The note content may be a single line or multiple paragraphs. If the user includes a line containing only `end note`, stop reading note content at that line; otherwise use all trailing text.

1. Confirm there is a currently active qode context at `.qode/contexts/current/notes.md`.
2. If no active context is available, stop and tell the user: `No currently active qode context.`
3. Very briefly rewrite the note content into concise technical notes.
4. Append only the new notes to `.qode/contexts/current/notes.md`.
5. Do not overwrite existing notes or remove the `# Notes` heading.
6. Reply with a one-line confirmation summarising what was appended.
