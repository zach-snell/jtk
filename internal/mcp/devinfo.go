package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageDevInfoArgs struct {
	Action              string `json:"action" jsonschema:"Action to perform: 'get_dev_info'" jsonschema_enum:"get_dev_info"`
	IssueKey            string `json:"issue_key" jsonschema:"Jira issue key (e.g., PROJ-123)"`
	IncludeBranches     *bool  `json:"include_branches,omitempty" jsonschema:"Include branches (default: true)"`
	IncludePullRequests *bool  `json:"include_pull_requests,omitempty" jsonschema:"Include pull requests (default: true)"`
	IncludeCommits      *bool  `json:"include_commits,omitempty" jsonschema:"Include commits (default: true)"`
	IncludeBuilds       *bool  `json:"include_builds,omitempty" jsonschema:"Include builds (default: true)"`
}

// ManageDevInfoHandler handles development info queries for an issue.
func ManageDevInfoHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageDevInfoArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageDevInfoArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "get_dev_info":
			if args.IssueKey == "" {
				return ToolResultError("issue_key is required for 'get_dev_info' action"), nil, nil
			}

			opts := &jira.DevInfoOptions{}
			// If any flag is explicitly set, use those; otherwise GetDevelopmentInfo defaults to all
			if args.IncludeBranches != nil || args.IncludePullRequests != nil || args.IncludeCommits != nil || args.IncludeBuilds != nil {
				if args.IncludeBranches != nil {
					opts.IncludeBranches = *args.IncludeBranches
				}
				if args.IncludePullRequests != nil {
					opts.IncludePRs = *args.IncludePullRequests
				}
				if args.IncludeCommits != nil {
					opts.IncludeCommits = *args.IncludeCommits
				}
				if args.IncludeBuilds != nil {
					opts.IncludeBuilds = *args.IncludeBuilds
				}
			} else {
				opts = nil // let GetDevelopmentInfo default to all
			}

			result, err := c.GetDevelopmentInfo(args.IssueKey, opts)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get dev info for %s: %v", args.IssueKey, err)), nil, nil
			}

			return ToolResultText(jira.SafeJSON(result, 40000)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: get_dev_info", args.Action)), nil, nil
		}
	}
}
