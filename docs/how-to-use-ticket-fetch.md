# How to use ticket fetch

qode uses IDE-native MCP (Model Context Protocol) servers to fetch ticket content. This gives the AI access to the full ticket — title, description, all comments with authors and timestamps, attachments, and linked resources (Figma designs, Google Docs, Confluence pages, etc.).

Use the `/qode-ticket-fetch <url>` slash command in Cursor or Claude Code. The AI will call the appropriate MCP server and write structured output to your branch context folder.

## Quick reference

| Category | Service | MCP server / docs |
|---|---|---|
| **Git provider** | GitHub | [github.com/modelcontextprotocol/servers](https://github.com/modelcontextprotocol/servers/tree/main/src/github) |
| **Ticketing** | Jira | [developer.atlassian.com/cloud/jira/platform/mcp](https://developer.atlassian.com/cloud/jira/platform/mcp) |
| **Ticketing** | Linear | [linear.app → Settings → API → MCP Server](https://linear.app/settings/api) |
| **Ticketing** | Azure DevOps | [github.com/microsoft/azure-devops-mcp](https://github.com/microsoft/azure-devops-mcp) |
| **Ticketing** | Notion | [github.com/notionhq/notion-mcp-server](https://github.com/notionhq/notion-mcp-server) |
| **Designs** | Figma | [figma.com → Settings → MCP Server](https://www.figma.com/developers/mcp) |
| **Docs** | Google Docs / Drive | [github.com/modelcontextprotocol/servers](https://github.com/modelcontextprotocol/servers/tree/main/src/gdrive) |
| **Docs** | SharePoint / OneDrive | [github.com/microsoft/sharepoint-mcp](https://github.com/microsoft/sharepoint-mcp) |
| **Docs** | Confluence | [developer.atlassian.com/cloud/confluence/mcp](https://developer.atlassian.com/cloud/confluence/mcp) |
| **Whiteboard** | Miro | [miro.com → Apps → Developers → MCP](https://developers.miro.com/docs/mcp) |

Jump to the relevant section below for install commands and auth setup.

---

## Ticketing systems

### GitHub Issues

**Official MCP server:** `@modelcontextprotocol/server-github`

```bash
# Install (Claude Code)
claude mcp add github -e GITHUB_PERSONAL_ACCESS_TOKEN=your-token -- npx -y @modelcontextprotocol/server-github
```

For Cursor, add to `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project):

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

Generate a token at: **GitHub → Settings → Developer settings → Personal access tokens**
Required scope: `repo` (private repos); no scope needed for public repos.

---

### Jira

**Official MCP server:** Atlassian Remote MCP Server

```bash
# Install (Claude Code)
claude mcp add jira --transport http --url https://mcp.atlassian.com/v1/sse
```

For Cursor, add to `mcp.json`:

```json
{
  "mcpServers": {
    "jira": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://mcp.atlassian.com/v1/sse"]
    }
  }
}
```

Authentication is handled via OAuth through the Atlassian developer portal. See: [developer.atlassian.com/cloud/jira/platform/mcp](https://developer.atlassian.com/cloud/jira/platform/mcp)

---

### Linear

**Official MCP server:** Linear MCP (built into Linear)

Enable at: **Linear → Settings → API → MCP Server**

Copy the generated MCP URL and add to Claude Code:

```bash
claude mcp add linear --transport http --url <your-linear-mcp-url>
```

For Cursor, add to `mcp.json`:

```json
{
  "mcpServers": {
    "linear": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "<your-linear-mcp-url>"]
    }
  }
}
```

---

### Azure DevOps

**Official MCP server:** `@microsoft/azure-devops-mcp`

```bash
# Install (Claude Code)
claude mcp add azure-devops -e AZURE_DEVOPS_ORG_URL=https://dev.azure.com/yourorg -e AZURE_DEVOPS_AUTH_TOKEN=your-pat -- npx -y @microsoft/azure-devops-mcp
```

For Cursor, add to `mcp.json`:

```json
{
  "mcpServers": {
    "azure-devops": {
      "command": "npx",
      "args": ["-y", "@microsoft/azure-devops-mcp"],
      "env": {
        "AZURE_DEVOPS_ORG_URL": "https://dev.azure.com/yourorg",
        "AZURE_DEVOPS_AUTH_TOKEN": "your-pat"
      }
    }
  }
}
```

Generate a PAT at: **Azure DevOps → User Settings → Personal access tokens**
Required scope: **Work Items → Read**

---

### Notion

**Official MCP server:** `@notionhq/notion-mcp-server`

```bash
# Install (Claude Code)
claude mcp add notion -e OPENAPI_MCP_HEADERS='{"Authorization":"Bearer your-token","Notion-Version":"2022-06-28"}' -- npx -y @notionhq/notion-mcp-server
```

For Cursor, add to `mcp.json`:

```json
{
  "mcpServers": {
    "notion": {
      "command": "npx",
      "args": ["-y", "@notionhq/notion-mcp-server"],
      "env": {
        "OPENAPI_MCP_HEADERS": "{\"Authorization\":\"Bearer your-token\",\"Notion-Version\":\"2022-06-28\"}"
      }
    }
  }
}
```

**Setup:**
1. Create an integration at: [notion.so/my-integrations](https://www.notion.so/my-integrations)
2. Copy the **Internal Integration Secret**
3. Share the target page or database with the integration (click `...` → `Connect to` → select your integration)

---

## Linked resource services

When your ticket contains links to other services, the AI will also fetch their content if an MCP server is configured.

### Figma

**Official MCP server:** Figma MCP (built into Figma)

Enable at: **Figma → Settings → MCP Server**

```bash
# Install (Claude Code)
claude mcp add figma -e FIGMA_API_KEY=your-key -- npx -y figma-mcp
```

For Cursor, add to `mcp.json`:

```json
{
  "mcpServers": {
    "figma": {
      "command": "npx",
      "args": ["-y", "figma-mcp"],
      "env": {
        "FIGMA_API_KEY": "your-key"
      }
    }
  }
}
```

Generate a key at: **Figma → Settings → Account → Personal access tokens**

---

### Google Docs / Drive

**Official MCP server:** `@modelcontextprotocol/server-gdrive`

```bash
# Install (Claude Code)
claude mcp add gdrive -- npx -y @modelcontextprotocol/server-gdrive
```

For Cursor, add to `mcp.json`:

```json
{
  "mcpServers": {
    "gdrive": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-gdrive"]
    }
  }
}
```

Authentication uses OAuth 2.0. Follow the setup guide at the MCP server's repository to configure Google Cloud credentials.

---

### SharePoint / OneDrive

**Official MCP server:** `@microsoft/sharepoint-mcp`

```bash
# Install (Claude Code)
claude mcp add sharepoint -e TENANT_ID=your-tenant -e CLIENT_ID=your-client-id -e CLIENT_SECRET=your-secret -- npx -y @microsoft/sharepoint-mcp
```

For Cursor, add to `mcp.json`:

```json
{
  "mcpServers": {
    "sharepoint": {
      "command": "npx",
      "args": ["-y", "@microsoft/sharepoint-mcp"],
      "env": {
        "TENANT_ID": "your-tenant",
        "CLIENT_ID": "your-client-id",
        "CLIENT_SECRET": "your-secret"
      }
    }
  }
}
```

Requires an Azure App Registration with Sites.Read.All permission.

---

### Confluence

**Official MCP server:** Atlassian Remote MCP Server (same as Jira)

The Atlassian MCP server covers both Jira and Confluence. If you have already configured the Jira MCP server, Confluence links are fetched automatically.

---

### Miro

**Official MCP server:** Miro MCP

Enable at: **Miro → Profile → Apps → Miro for Developers → MCP**

```bash
# Install (Claude Code)
claude mcp add miro -e MIRO_ACCESS_TOKEN=your-token -- npx -y @mirohq/miro-mcp
```

For Cursor, add to `mcp.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "npx",
      "args": ["-y", "@mirohq/miro-mcp"],
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

---

## Usage

Once your MCP servers are configured, run the slash command in Cursor or Claude Code:

```
/qode-ticket-fetch https://github.com/owner/repo/issues/42
```

The AI fetches the ticket via MCP and writes to your branch context:

```
.qode/branches/<branch>/context/ticket.md          # title, description, metadata
.qode/branches/<branch>/context/ticket-comments.md  # comments with authors and timestamps
.qode/branches/<branch>/context/ticket-links.md     # linked resources with summaries
```

If an MCP server is not available for a service, the AI records the URL and title of linked resources without fetching their content.
