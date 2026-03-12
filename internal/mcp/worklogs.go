package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageWorklogsArgs struct {
	Action     string `json:"action" jsonschema:"Action to perform: 'list', 'add'" jsonschema_enum:"list,add"`
	IssueKey   string `json:"issue_key" jsonschema:"Jira issue key (e.g., PROJ-123)"`
	TimeSpent  string `json:"time_spent,omitempty" jsonschema:"Time spent (for 'add'), e.g. '2h', '1d', '30m'"`
	Started    string `json:"started,omitempty" jsonschema:"Start datetime ISO 8601 (for 'add'), e.g. '2024-01-15T09:00:00.000+0000'. Defaults to now"`
	Comment    string `json:"comment,omitempty" jsonschema:"Worklog comment (for 'add')"`
	StartAt    int    `json:"start_at,omitempty" jsonschema:"Pagination start index (for 'list')"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"Maximum results to return (for 'list')"`
}

// ManageWorklogsHandler handles worklog operations on issues.
func ManageWorklogsHandler(c *jira.Client, perms map[string]bool) func(context.Context, *mcp.CallToolRequest, ManageWorklogsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageWorklogsArgs) (*mcp.CallToolResult, any, error) {
		if args.IssueKey == "" {
			return ToolResultError("issue_key is required"), nil, nil
		}

		switch args.Action {
		case "list":
			result, err := c.ListWorklogs(args.IssueKey, args.StartAt, args.MaxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list worklogs: %v", err)), nil, nil
			}
			type flatWorklog struct {
				ID        string `json:"id"`
				Author    string `json:"author"`
				TimeSpent string `json:"time_spent"`
				Started   string `json:"started"`
				Comment   string `json:"comment,omitempty"`
			}
			flat := struct {
				Total    int           `json:"total"`
				Worklogs []flatWorklog `json:"worklogs"`
			}{Total: result.Total}
			for _, w := range result.Worklogs {
				author := "unknown"
				if w.Author != nil {
					author = w.Author.DisplayName
				}
				flat.Worklogs = append(flat.Worklogs, flatWorklog{
					ID:        w.ID,
					Author:    author,
					TimeSpent: w.TimeSpent,
					Started:   w.Started,
					Comment:   jira.ADFToPlainText(w.Comment),
				})
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		case "add":
			if !perms["ADD_COMMENTS"] {
				return ToolResultError("token lacks ADD_COMMENTS permission (required for worklogs)"), nil, nil
			}
			if args.TimeSpent == "" {
				return ToolResultError("time_spent is required for 'add' action (e.g. '2h', '1d', '30m')"), nil, nil
			}
			worklogReq := jira.BuildAddWorklogRequest(args.TimeSpent, args.Started, args.Comment)
			result, err := c.AddWorklog(args.IssueKey, worklogReq)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to add worklog: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(map[string]string{
				"id":         result.ID,
				"time_spent": result.TimeSpent,
				"started":    result.Started,
				"status":     "worklog added",
			}, 30000)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, add", args.Action)), nil, nil
		}
	}
}
