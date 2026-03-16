package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
	"github.com/zach-snell/jtk/internal/version"
)

// New creates and configures the Jira MCP server with a classic-auth client.
// Prefer NewFromCredentials for automatic token type detection.
func New(domain, email, token string) *mcp.Server {
	client := jira.NewClient(domain, email, token)
	return newServer(client)
}

// NewFromCredentials creates the MCP server from stored credentials.
// Automatically detects token type and configures auth accordingly.
func NewFromCredentials(creds *jira.Credentials) (*mcp.Server, error) {
	client, err := jira.NewClientFromCredentials(creds)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return newServer(client), nil
}

func newServer(client *jira.Client) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "jtk",
			Version: version.Version,
		},
		nil,
	)

	registerTools(s, client)
	registerPrompts(s)
	return s
}

// NewUnauthenticated creates an MCP server with no credentials.
// All tools are registered but return an auth-required message when called.
// Prompts are registered normally (they don't require auth).
func NewUnauthenticated() *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "jtk",
			Version: version.Version,
		},
		nil,
	)

	registerToolsUnauthenticated(s)
	registerPrompts(s)
	return s
}

const authRequiredMessage = `Authentication required. Configure credentials via:
  1. Run: jtk auth
  2. Set environment variables: JIRA_DOMAIN + JIRA_EMAIL + JIRA_API_TOKEN
  3. Config file: ~/.config/jtk/credentials.json`

// addUnauthenticatedTool registers a tool with a stub handler that returns an auth-required message.
func addUnauthenticatedTool[In any](s *mcp.Server, tool mcp.Tool) {
	mcp.AddTool(s, &tool, func(_ context.Context, _ *mcp.CallToolRequest, _ In) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: authRequiredMessage},
			},
		}, nil, nil
	})
}

// registerToolsUnauthenticated registers all tools with stub handlers that return auth-required errors.
// Tool names, descriptions, and input schemas match the real tools.
func registerToolsUnauthenticated(s *mcp.Server) {
	addUnauthenticatedTool[ManageIssuesArgs](s, mcp.Tool{
		Name:        "manage_issues",
		Description: "Unified tool for Jira issue operations. Actions: 'get', 'list_types', 'get_links', 'get_history', 'create', 'update', 'assign', 'transition', 'add_comment', 'edit_comment', 'list_comments', 'delete', 'link', 'list_link_types', 'get_watchers', 'add_watcher', 'remove_watcher', 'move'",
	})

	addUnauthenticatedTool[ManageSearchArgs](s, mcp.Tool{
		Name:        "manage_search",
		Description: "Search Jira issues using JQL or quick text search. Actions: 'jql', 'quick'",
	})

	addUnauthenticatedTool[ManageBoardsArgs](s, mcp.Tool{
		Name:        "manage_boards",
		Description: "Manage Jira agile boards and sprints. Actions: 'list_boards', 'get_board', 'list_sprints', 'get_sprint_issues', 'get_backlog', 'get_active_sprint', 'search_sprints', 'create_sprint', 'move_to_sprint'",
	})

	addUnauthenticatedTool[ManageProjectsArgs](s, mcp.Tool{
		Name:        "manage_projects",
		Description: "List and get Jira project details and statuses. Actions: 'list', 'get', 'list_statuses', 'create'",
	})

	addUnauthenticatedTool[ManageDevInfoArgs](s, mcp.Tool{
		Name:        "manage_devinfo",
		Description: "Get development information (branches, PRs, commits) linked to a Jira issue. Actions: 'get_dev_info'",
	})

	addUnauthenticatedTool[ManageWorklogsArgs](s, mcp.Tool{
		Name:        "manage_worklogs",
		Description: "Manage time tracking worklogs on Jira issues. Actions: 'list', 'add'",
	})

	addUnauthenticatedTool[ManageVersionsArgs](s, mcp.Tool{
		Name:        "manage_versions",
		Description: "List and get project versions (releases/fixVersions). Actions: 'list', 'get'",
	})

	addUnauthenticatedTool[ManageAttachmentsArgs](s, mcp.Tool{
		Name:        "manage_attachments",
		Description: "Manage Jira issue attachments (list, download, upload, delete). Actions: 'list', 'download', 'upload', 'delete'",
	})

	addUnauthenticatedTool[ManageUsersArgs](s, mcp.Tool{
		Name:        "manage_users",
		Description: "Search and get Jira users. Actions: 'get_current', 'search', 'get'",
	})

	addUnauthenticatedTool[ManageMetricsArgs](s, mcp.Tool{
		Name:        "manage_metrics",
		Description: "Get issue lifecycle metrics for dashboards and visualizations. Actions: 'get_dates' (raw date info, status transitions, time-in-status), 'get_metrics' (computed cycle time, lead time, time in current status, status breakdown)",
	})
}

// getPermissions checks what the current token can do via the mypermissions API.
func getPermissions(c *jira.Client) map[string]bool {
	perms := make(map[string]bool)

	resp, err := c.GetMyPermissions([]string{
		"BROWSE_PROJECTS",
		"CREATE_ISSUES",
		"TRANSITION_ISSUES",
		"ADD_COMMENTS",
		"EDIT_ISSUES",
		"ASSIGN_ISSUES",
		"DELETE_ISSUES",
		"LINK_ISSUES",
		"CREATE_ATTACHMENTS",
		"DELETE_ALL_ATTACHMENTS",
		"MOVE_ISSUES",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch permissions for introspection: %v\n", err)
		// Assume all permissions if we can't check — let the API reject if needed
		perms["BROWSE_PROJECTS"] = true
		perms["CREATE_ISSUES"] = true
		perms["TRANSITION_ISSUES"] = true
		perms["ADD_COMMENTS"] = true
		perms["EDIT_ISSUES"] = true
		perms["ASSIGN_ISSUES"] = true
		perms["DELETE_ISSUES"] = true
		perms["LINK_ISSUES"] = true
		perms["CREATE_ATTACHMENTS"] = true
		perms["DELETE_ALL_ATTACHMENTS"] = true
		perms["MOVE_ISSUES"] = true
		return perms
	}

	for key, perm := range resp.Permissions {
		perms[key] = perm.HavePermission
	}

	return perms
}

// addTool is a helper function to conditionally register a tool handler.
func addTool[In any](s *mcp.Server, disabled map[string]bool, tool mcp.Tool, handler func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, any, error)) {
	if disabled[tool.Name] {
		return
	}
	mcp.AddTool(s, &tool, handler)
}

func registerTools(s *mcp.Server, c *jira.Client) {
	disabledToolsEnv := os.Getenv("JIRA_DISABLED_TOOLS")
	disabled := make(map[string]bool)
	if disabledToolsEnv != "" {
		for _, t := range strings.Split(disabledToolsEnv, ",") {
			disabled[strings.TrimSpace(t)] = true
		}
	}

	perms := getPermissions(c)

	// Build write permission description suffix
	writeNote := ""
	if !perms["CREATE_ISSUES"] {
		writeNote = " (read-only: token lacks CREATE_ISSUES permission)"
	}

	// ─── Issues ──────────────────────────────────────────────────────
	issueActions := "'get', 'list_types', 'get_links', 'get_history'"
	if perms["CREATE_ISSUES"] {
		issueActions += ", 'create'"
	}
	if perms["EDIT_ISSUES"] {
		issueActions += ", 'update'"
	}
	if perms["ASSIGN_ISSUES"] {
		issueActions += ", 'assign'"
	}
	if perms["TRANSITION_ISSUES"] {
		issueActions += ", 'transition'"
	}
	if perms["ADD_COMMENTS"] {
		issueActions += ", 'add_comment', 'list_comments'"
	} else {
		issueActions += ", 'list_comments'"
	}
	if perms["DELETE_ISSUES"] {
		issueActions += ", 'delete'"
	}
	if perms["LINK_ISSUES"] {
		issueActions += ", 'link'"
	}
	if perms["MOVE_ISSUES"] {
		issueActions += ", 'move'"
	}

	addTool(s, disabled, mcp.Tool{
		Name:        "manage_issues",
		Description: fmt.Sprintf("Unified tool for Jira issue operations. Actions: %s%s", issueActions, writeNote),
	}, ManageIssuesHandler(c, perms))

	// ─── Search ──────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_search",
		Description: "Search Jira issues using JQL or quick text search. Actions: 'jql', 'quick'",
	}, ManageSearchHandler(c))

	// ─── Boards ──────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_boards",
		Description: "Manage Jira agile boards and sprints. Actions: 'list_boards', 'get_board', 'list_sprints', 'get_sprint_issues', 'get_backlog', 'get_active_sprint', 'search_sprints', 'create_sprint', 'move_to_sprint'",
	}, ManageBoardsHandler(c))

	// ─── Projects ────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_projects",
		Description: "List and get Jira project details and statuses. Actions: 'list', 'get', 'list_statuses', 'create'",
	}, ManageProjectsHandler(c))

	// ─── Dev Info ────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_devinfo",
		Description: "Get development information (branches, PRs, commits) linked to a Jira issue. Actions: 'get_dev_info'",
	}, ManageDevInfoHandler(c))

	// ─── Worklogs ───────────────────────────────────────────────────
	worklogActions := "'list'"
	if perms["ADD_COMMENTS"] {
		worklogActions += ", 'add'"
	}
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_worklogs",
		Description: fmt.Sprintf("Manage time tracking worklogs on Jira issues. Actions: %s", worklogActions),
	}, ManageWorklogsHandler(c, perms))

	// ─── Versions ───────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_versions",
		Description: "List and get project versions (releases/fixVersions). Actions: 'list', 'get'",
	}, ManageVersionsHandler(c))

	// ─── Attachments ────────────────────────────────────────────────
	attachActions := "'list', 'download'"
	if perms["CREATE_ATTACHMENTS"] {
		attachActions += ", 'upload'"
	}
	if perms["DELETE_ALL_ATTACHMENTS"] {
		attachActions += ", 'delete'"
	}
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_attachments",
		Description: fmt.Sprintf("Manage Jira issue attachments (list, download, upload, delete). Actions: %s", attachActions),
	}, ManageAttachmentsHandler(c, perms))

	// ─── Users ──────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_users",
		Description: "Search and get Jira users. Actions: 'get_current', 'search', 'get'",
	}, ManageUsersHandler(c))

	// ─── Metrics ────────────────────────────────────────────────────
	addTool(s, disabled, mcp.Tool{
		Name:        "manage_metrics",
		Description: "Get issue lifecycle metrics for dashboards and visualizations. Actions: 'get_dates' (raw date info, status transitions, time-in-status), 'get_metrics' (computed cycle time, lead time, time in current status, status breakdown)",
	}, ManageMetricsHandler(c))
}
