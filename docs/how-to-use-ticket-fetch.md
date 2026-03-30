# How to use ticket fetch

Each ticketing system requires specific environment variables. qode auto-loads a `.env` file from the project root, so you don't need to `source` it manually.

## Jira

Requires a Jira account with API token access.

```bash
export JIRA_EMAIL=you@company.com
export JIRA_API_TOKEN=your-token

qode ticket fetch https://company.atlassian.net/browse/ENG-123
```

Generate a token at: **Atlassian Account → Security → API tokens**

## Azure DevOps

Requires a Personal Access Token (PAT) with Work Items read scope.

```bash
export AZURE_DEVOPS_PAT=your-pat

qode ticket fetch https://dev.azure.com/org/project/_workitems/edit/456
```

Generate a PAT at: **Azure DevOps → User Settings → Personal access tokens**

Required scope: **Work Items → Read**

## Linear

Requires a Linear API key.

```bash
export LINEAR_API_KEY=your-key

qode ticket fetch https://linear.app/team/ENG-123
```

Generate a key at: **Linear → Settings → API → Personal API keys**

## GitHub Issues

Public repositories work without authentication (subject to GitHub's unauthenticated rate limit of 60 requests/hour). Private repositories require a `GITHUB_TOKEN`.

```bash
# Public repos — no token required
qode ticket fetch https://github.com/owner/repo/issues/42

# Private repos — token required
export GITHUB_TOKEN=your-token
qode ticket fetch https://github.com/owner/private-repo/issues/42
```

**Classic PAT** — generate at: **GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)**

Required scope: `repo` (for private repositories; no scope needed for public)

**Fine-grained PAT** — generate at: **GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens**

Required permission: `Issues: Read` (for private repositories)

## Notion

Requires a Notion internal integration token. The integration must be shared with the target page or database.

```bash
export NOTION_API_KEY=your-token

qode ticket fetch https://www.notion.so/workspace/My-Ticket-abc123de1234567890abcdef12345678
```

**Setup:**
1. Create an integration at: **[notion.so/my-integrations](https://www.notion.so/my-integrations)**
2. Copy the **Internal Integration Secret**
3. Share the target page or database with the integration (click `...` → `Connect to` → select your integration)

## Using a .env File

Store credentials in a `.env` file at the project root (never commit this file). qode loads it automatically before fetching tickets.

```bash
# .env — add to .gitignore
JIRA_EMAIL=you@company.com
JIRA_API_TOKEN=your-jira-token
AZURE_DEVOPS_PAT=your-ado-pat
LINEAR_API_KEY=your-linear-key
GITHUB_TOKEN=your-github-token
NOTION_API_KEY=your-notion-token
```

## IDE Slash Command

In Cursor or Claude Code, you can use the slash command instead of the terminal:

```
/qode-ticket-fetch https://company.atlassian.net/browse/ENG-123
```

This runs the same fetch and writes the result to `context/ticket.md`.

## Verification

```bash
# Verify each integration by fetching a known ticket
qode ticket fetch https://company.atlassian.net/browse/ENG-1
qode ticket fetch https://dev.azure.com/org/project/_workitems/edit/1
qode ticket fetch https://linear.app/team/ENG-1
qode ticket fetch https://github.com/owner/repo/issues/1
qode ticket fetch https://www.notion.so/workspace/My-Ticket-abc123de1234567890abcdef12345678

# Output is written to .qode/branches/{branch}/context/ticket.md
cat .qode/branches/$(git branch --show-current | sed 's|/|--|g')/context/ticket.md
```
