# BUG: The /qode-ticket-fetch slash command cannot read the GITHUB_TOKEN

When running the `/qode-ticket-fetch [url]` through claude code in Visual Studio extension the command tries running a bash command `qode ticket fetch [url]` but the GITHUB_TOKEN is not accessible.

Assume same thing happens with other ticketing systems.

Assume all IDE's are affected too.

One potential solution is to apply the `.env` file if it exists

The same solution for reading the env vars should be applied to all slash commands
