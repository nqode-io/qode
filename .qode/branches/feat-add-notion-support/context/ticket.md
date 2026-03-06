# Add support for notion tickets

## Problem / motivation
Some of the projects don't use any of the already implemented. Instead they use Notion. Notion can be set up as a board with tickets.

## Proposed solution

Implement support for fetching tickets from Notion via `qode ticket fetch <url>` or equivalent slash commands in all supported IDEs.

## Alternatives considered

There are no alternatives apart from copying the content of the tickets from Notion to the `ticket.md` file by hand which is not acceptable.

## Additional context

The Notion API can be accessed [here](https://developers.notion.com/guides/get-started/getting-started)

Also, documentation should be updated to reflect the changes, namely the README.md, CLAUDE.md,  and how-to-use-ticket-fetch.md files as well as the inline help in the help.go but also other places that deal with slash commands and equivalents in all IDEs such as inline templates, configuration files, and prompt templates.

