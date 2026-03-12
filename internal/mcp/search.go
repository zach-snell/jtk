package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageSearchArgs struct {
	Action     string `json:"action" jsonschema:"Action to perform: 'jql', 'quick'" jsonschema_enum:"jql,quick"`
	JQL        string `json:"jql,omitempty" jsonschema:"JQL query string (for 'jql' action)"`
	Text       string `json:"text,omitempty" jsonschema:"Search text (for 'quick' action)"`
	ProjectKey string `json:"project_key,omitempty" jsonschema:"Optional project key to scope the search"`
	StartAt    int    `json:"start_at,omitempty" jsonschema:"Pagination start index"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"Maximum results to return (default 20)"`
}

// ManageSearchHandler handles JQL and quick search operations.
func ManageSearchHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageSearchArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageSearchArgs) (*mcp.CallToolResult, any, error) {
		maxResults := args.MaxResults
		if maxResults <= 0 {
			maxResults = 20
		}

		switch args.Action {
		case "jql":
			if args.JQL == "" {
				return ToolResultError("jql is required for 'jql' action"), nil, nil
			}
			result, err := c.SearchJQL(args.JQL, args.StartAt, maxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("JQL search failed: %v", err)), nil, nil
			}
			return ToolResultText(flattenSearchResult(result)), nil, nil

		case "quick":
			if args.Text == "" {
				return ToolResultError("text is required for 'quick' action"), nil, nil
			}
			result, err := c.QuickSearch(args.Text, args.ProjectKey, maxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("quick search failed: %v", err)), nil, nil
			}
			return ToolResultText(flattenSearchResult(result)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: jql, quick", args.Action)), nil, nil
		}
	}
}

func flattenSearchResult(result *jira.SearchResult) string {
	type flatIssue struct {
		Key      string `json:"key"`
		Summary  string `json:"summary"`
		Status   string `json:"status"`
		Type     string `json:"type"`
		Priority string `json:"priority"`
		Assignee string `json:"assignee"`
		Updated  string `json:"updated"`
	}

	flat := struct {
		Total         int         `json:"total"`
		StartAt       int         `json:"start_at"`
		MaxResults    int         `json:"max_results"`
		NextPageToken string      `json:"next_page_token,omitempty"`
		Issues        []flatIssue `json:"issues"`
	}{
		Total:         result.Total,
		StartAt:       result.StartAt,
		MaxResults:    result.MaxResults,
		NextPageToken: result.NextPageToken,
	}

	for _, issue := range result.Issues {
		fi := flatIssue{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Updated: issue.Fields.Updated,
		}
		if issue.Fields.Status != nil {
			fi.Status = issue.Fields.Status.Name
		}
		if issue.Fields.IssueType != nil {
			fi.Type = issue.Fields.IssueType.Name
		}
		if issue.Fields.Priority != nil {
			fi.Priority = issue.Fields.Priority.Name
		}
		if issue.Fields.Assignee != nil {
			fi.Assignee = issue.Fields.Assignee.DisplayName
		} else {
			fi.Assignee = "unassigned"
		}
		flat.Issues = append(flat.Issues, fi)
	}

	return jira.SafeJSON(flat, 40000)
}
