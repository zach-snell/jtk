package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageMetricsArgs struct {
	Action   string `json:"action" jsonschema:"Action to perform: 'get_dates', 'get_metrics'" jsonschema_enum:"get_dates,get_metrics"`
	IssueKey string `json:"issue_key" jsonschema:"Jira issue key (e.g., PROJ-123)"`
}

// ManageMetricsHandler handles issue metrics operations.
// get_dates: returns raw date info with status transitions and time-in-status aggregation.
// get_metrics: returns computed cycle time, lead time, time in current status, and status breakdown.
func ManageMetricsHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageMetricsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageMetricsArgs) (*mcp.CallToolResult, any, error) {
		if args.IssueKey == "" {
			return ToolResultError("issue_key is required"), nil, nil
		}

		switch args.Action {
		case "get_dates":
			dates, err := c.GetIssueDates(args.IssueKey)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get issue dates: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(dates, 50000)), nil, nil

		case "get_metrics":
			metrics, err := c.GetIssueMetrics(args.IssueKey)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get issue metrics: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(metrics, 50000)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: get_dates, get_metrics", args.Action)), nil, nil
		}
	}
}
