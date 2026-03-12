package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageSearchArgs struct {
	Action     string `json:"action" jsonschema:"Action to perform: 'jql', 'quick'" jsonschema_enum:"jql,quick"`
	JQL        string `json:"jql,omitempty" jsonschema:"JQL query string (required for 'jql' action). Example: type=page AND space=DEV AND title~'architecture'. Common JQL patterns: 'project = PROJ AND status = \"In Progress\"', 'assignee = currentUser() ORDER BY updated DESC', 'labels = bug AND priority in (High, Highest)', 'sprint in openSprints()', 'created >= -7d AND type = Bug', 'text ~ \"search term\"', 'status changed TO Done AFTER -30d'"`
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

	for i := range result.Issues {
		fi := flatIssue{
			Key:     result.Issues[i].Key,
			Summary: result.Issues[i].Fields.Summary,
			Updated: result.Issues[i].Fields.Updated,
		}
		if result.Issues[i].Fields.Status != nil {
			fi.Status = result.Issues[i].Fields.Status.Name
		}
		if result.Issues[i].Fields.IssueType != nil {
			fi.Type = result.Issues[i].Fields.IssueType.Name
		}
		if result.Issues[i].Fields.Priority != nil {
			fi.Priority = result.Issues[i].Fields.Priority.Name
		}
		if result.Issues[i].Fields.Assignee != nil {
			fi.Assignee = result.Issues[i].Fields.Assignee.DisplayName
		} else {
			fi.Assignee = "unassigned"
		}
		flat.Issues = append(flat.Issues, fi)
	}

	return jira.SafeJSON(flat, 40000)
}
