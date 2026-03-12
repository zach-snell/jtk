# jtk (Jira CLI & MCP Server)

[![Documentation](https://img.shields.io/badge/docs-reference-blue)](https://zach-snell.github.io/jtk/)
[![Go Report Card](https://goreportcard.com/badge/github.com/zach-snell/jtk)](https://goreportcard.com/report/github.com/zach-snell/jtk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A complete command-line interface and Model Context Protocol (MCP) server written in Go that provides programmatic integration with Jira Cloud.

<p align="center">
  <img src="demo.gif" alt="jtk CLI demo" width="700" />
</p>

## Features

- **Dual Mode**: Run as a rich, interactive CLI tool for daily developer tasks, or as an MCP server for AI agents.
- **Git Awareness**: Automatically detects Jira issue keys from your current branch name (e.g., `feature/PROJ-123-add-login` → `PROJ-123`).
- **Dynamic Permission Introspection**: The MCP server calls `/mypermissions` at startup and only registers mutation tools your token is authorized for.
- **Dev Status API**: Surfaces branches, PRs, and commits linked to any issue via Jira's undocumented 3-step dev-status endpoint.
- **MCP Prompts**: Pre-built prompt templates for standups, sprint status, release notes, and dev dependency trees.
- **Token-Efficient**: Consolidated action-based tools minimize schema injection overhead. ResponseFlattener strips bloated Jira JSON. Full ADF→Markdown renderer.
- **Agile-First**: Full sprint lifecycle — boards, sprints, backlogs, sprint mutations, and active sprint detection.

## Installation

### From Source
```bash
# Clone the repository
git clone https://github.com/zach-snell/jtk.git
cd jtk

# Run the install script (builds and moves to ~/.local/bin)
./install.sh
```

Ensure `~/.local/bin` is added to your system `$PATH` for the executable to be universally available.

### From GitHub Releases
Download the appropriate binary for your system (Linux, macOS, Windows) from the [Releases](https://github.com/zach-snell/jtk/releases) page.

## CLI Usage

`jtk` provides a robust command-line interface with the following core modules:

```bash
# Authenticate (stores credentials in ~/.config/jtk/)
jtk auth

# Manage issues (auto-detects issue key from git branch)
jtk issues [get, create, update, transition, assign, delete, comments, links]

# Search with JQL or quick text
jtk issues search --jql "project = PROJ AND status = 'In Progress'"

# Agile boards and sprints
jtk boards [list, get, sprints, backlog, active-sprint]

# Projects and statuses
jtk projects [list, get, statuses]

# Users
jtk users [search, get, me]

# Time tracking
jtk worklogs [list, add]

# Release management
jtk versions [list, get]
```

## MCP Usage

The tool also serves as an MCP server. It supports two protocols: Stdio (default via `jtk mcp`) and the official Streamable Transport API over HTTP.

### Stdio Transport (Default)
If you intend to use this with an MCP client (such as Claude Desktop or Cursor), add it to your client's configuration file as a local command:

```json
{
  "mcpServers": {
    "jira": {
      "command": "/absolute/path/to/jtk",
      "args": ["mcp"],
      "env": {
        "JIRA_DOMAIN": "your-domain.atlassian.net",
        "JIRA_EMAIL": "you@example.com",
        "JIRA_API_TOKEN": "your-api-token"
      }
    }
  }
}
```

### Streamable Transport (HTTP)
You can run the server as a long-lived HTTP process serving the Streamable Transport API (which uses Server-Sent Events underneath). This is useful for remote network clients.

```bash
jtk mcp --port 8080
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `JIRA_DOMAIN` | Your Atlassian domain (e.g., `acme.atlassian.net`) | Yes |
| `JIRA_EMAIL` | Email associated with the API token | Yes |
| `JIRA_API_TOKEN` | An Atlassian API Token | Yes |
| `JIRA_DISABLED_TOOLS` | Comma-separated tool names to hide from AI agents | No |

### API Token Scopes

Create a **Jira** app token at [id.atlassian.com](https://id.atlassian.com/manage-profile/security/api-tokens) with granular scopes. Run `jtk auth` to see the full recommended scope list, or use these:

**Read-only (18 scopes):**
```
read:me, read:jql:jira, read:issue-details:jira, read:issue-type:jira,
read:issue-link:jira, read:issue-worklog:jira, read:issue.changelog:jira,
read:issue.transition:jira, read:comment:jira, read:attachment:jira,
read:project:jira, read:project-version:jira, read:status:jira,
read:user:jira, read:permission:jira, read:board-scope:jira-software,
read:sprint:jira-software, read:dev-info:jira
```

**Full access (add these 7):**
```
write:issue:jira, write:comment:jira, write:issue-worklog:jira,
write:issue-link:jira, write:attachment:jira, write:sprint:jira-software,
delete:issue:jira
```

### Permission Introspection & Security

**Two-layer safety model:**

1. **Token scopes** — granular Atlassian scopes control which APIs the token can call at all (403 if missing)
2. **Jira permissions** — jtk queries `/mypermissions` at MCP startup and dynamically hides mutation tools your account lacks (e.g., no `CREATE_ISSUES` → no create action)

**Explicit Tool Denial:** Even with full scopes and permissions, you can explicitly deny the AI agent access to any tool:

```bash
export JIRA_DISABLED_TOOLS="manage_boards,manage_worklogs"
```

## Tools Provided

| Tool | Description |
|------|-------------|
| `manage_issues` | Issue operations — get, create, update, assign, transition, delete, add comment, list comments, list types, get links, get history, link |
| `manage_search` | Search via JQL or quick text with cursor-based pagination |
| `manage_boards` | Agile boards — list, get, sprints, backlog, active sprint, search sprints, create sprint, move to sprint |
| `manage_projects` | List and get project details and statuses |
| `manage_devinfo` | Dev-status API — branches, PRs, commits linked to an issue |
| `manage_worklogs` | Time tracking — list and add worklogs |
| `manage_versions` | Project versions/releases — list and get |
| `manage_attachments` | Issue attachments — list and download |
| `manage_users` | User operations — get current, search, get by ID |

### MCP Prompts

| Prompt | Description |
|--------|-------------|
| `standup_summary` | Generate a standup report from recent activity |
| `sprint_status` | Analyze sprint health and progress |
| `release_notes` | Draft release notes from a version's issues |
| `dev_dependency_tree` | Map development dependencies across linked issues |

## Development

Requirements:
- Go 1.26+

```bash
# Run tests
go test ./...

# Run the linter
golangci-lint run ./...
```

## License

This project is licensed under the [Apache 2.0 License](LICENSE).
