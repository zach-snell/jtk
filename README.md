# jtk — Jira Toolkit

[![CI](https://github.com/zach-snell/jtk/actions/workflows/ci.yml/badge.svg)](https://github.com/zach-snell/jtk/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/zach-snell/jtk)](https://goreportcard.com/report/github.com/zach-snell/jtk)
[![Documentation](https://img.shields.io/badge/docs-reference-blue)](https://zach-snell.github.io/jtk/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

The most comprehensive dedicated Jira MCP server in the open-source ecosystem. A dual-mode Go binary that works as both a rich CLI tool and an MCP server for AI agents.

**11 MCP tools · 4 prompt templates · Full CLI · Single binary · Zero dependencies**

[![jtk MCP server](https://glama.ai/mcp/servers/zach-snell/jtk/badges/card.svg)](https://glama.ai/mcp/servers/zach-snell/jtk)

## Why jtk?

| | jtk | mcp-atlassian (Python) |
|---|---|---|
| **Language** | Go (single ~15MB binary) | Python (pip install + deps) |
| **Jira tools** | 11 dedicated tools | ~30 mixed Jira+Confluence |
| **Startup** | ~50ms | ~2s |
| **Permission introspection** | Dynamic at startup | None |
| **Dev status API** | Branches, PRs, commits | Not available |
| **Issue metrics** | Cycle time, lead time, time-in-status | Not available |
| **MCP prompts** | 4 built-in templates | None |
| **Auth** | Classic + scoped tokens | Classic only |

## Features

- **Dual Mode** — CLI for humans, MCP server for AI agents, same binary
- **Git Awareness** — Auto-detects Jira issue keys from branch names (e.g., `feature/PROJ-123-add-login` → `PROJ-123`)
- **Dynamic Permission Introspection** — Queries `/mypermissions` at MCP startup, only registers mutation tools your token allows
- **Dev Status API** — Surfaces branches, PRs, and commits linked to any issue via Jira's 3-step dev-status endpoint
- **Issue Metrics** — Cycle time, lead time, time-in-status breakdown with status transition history
- **MCP Prompts** — Standup summary, sprint status, release notes, dev dependency tree
- **Token-Efficient** — Consolidated action-based tools minimize schema overhead. ResponseFlattener strips bloated JSON. Full ADF↔Markdown conversion
- **Agile-First** — Boards, sprints, backlogs, sprint mutations, active sprint detection

## Installation

```bash
# From source
git clone https://github.com/zach-snell/jtk.git && cd jtk
./install.sh  # builds and copies to ~/.local/bin

# Or build manually
go build -o jtk ./cmd/jtk
```

Pre-built binaries available on the [Releases](https://github.com/zach-snell/jtk/releases) page.

## Quick Start

```bash
# Authenticate
jtk auth

# Get current issue from git branch
jtk issues get

# Search with JQL
jtk issues search --jql "project = PROJ AND status = 'In Progress'"

# Create an issue
jtk issues create --project PROJ --type Task --summary "Fix login bug"

# Sprint overview
jtk boards list
jtk boards active-sprint --board 1
```

## CLI Commands

```
jtk auth                    Authenticate with Jira Cloud
jtk issues                  Issue CRUD, search, comments, transitions, links
jtk boards                  Agile boards, sprints, backlogs
jtk projects                List, get, create projects
jtk users                   Search and get users
jtk versions                Project versions/releases
jtk worklogs                Time tracking
```

## MCP Server

### Stdio Transport (Claude Desktop, Cursor, OpenCode, etc.)

```json
{
  "mcpServers": {
    "jira": {
      "command": "/path/to/jtk",
      "args": ["mcp"],
      "env": {
        "JIRA_DOMAIN": "your-domain",
        "JIRA_EMAIL": "you@example.com",
        "JIRA_API_TOKEN": "your-api-token"
      }
    }
  }
}
```

### Streamable HTTP Transport

```bash
jtk mcp --port 8080
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `JIRA_DOMAIN` | Atlassian domain (e.g., `acme` for acme.atlassian.net) | Yes |
| `JIRA_EMAIL` | Email for the API token | Yes |
| `JIRA_API_TOKEN` | Atlassian API token | Yes |
| `JIRA_TOKEN_TYPE` | `classic` or `scoped` (auto-detected if omitted) | No |
| `JIRA_DISABLED_TOOLS` | Comma-separated tool names to hide | No |

## MCP Tools (11)

| Tool | Actions |
|------|---------|
| `manage_issues` | get, create, update, assign, transition, delete, add_comment, edit_comment, list_comments, list_types, get_links, get_history, link, list_link_types, get_watchers, add_watcher, remove_watcher |
| `manage_search` | jql, quick |
| `manage_boards` | list_boards, get_board, list_sprints, get_sprint_issues, get_backlog, get_active_sprint, search_sprints, create_sprint, update_sprint, move_to_sprint |
| `manage_projects` | list, get, list_statuses, create |
| `manage_devinfo` | get_dev_info |
| `manage_worklogs` | list, add |
| `manage_versions` | list, get, create |
| `manage_attachments` | list, download, upload, delete |
| `manage_users` | get_current, search, get |
| `manage_metrics` | get_dates, get_metrics |

### MCP Prompts (4)

| Prompt | Description |
|--------|-------------|
| `standup_summary` | Generate a standup report from recent activity |
| `sprint_status` | Analyze sprint health and progress |
| `release_notes` | Draft release notes from a version's issues |
| `dev_dependency_tree` | Map development dependencies across linked issues |

## Security

**Three-layer safety model:**

1. **Token scopes** — Atlassian scopes control which APIs the token can call (403 if missing)
2. **Permission introspection** — jtk queries `/mypermissions` at startup and dynamically hides mutation tools your account lacks
3. **Tool denial** — Explicitly hide tools: `JIRA_DISABLED_TOOLS="manage_boards,manage_worklogs"`

## Development

```bash
go test -race ./...          # Run tests
golangci-lint run ./...      # Lint
go build -o jtk ./cmd/jtk   # Build
```

## License

[Apache 2.0](LICENSE)