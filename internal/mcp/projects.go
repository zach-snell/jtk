package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageProjectsArgs struct {
	Action     string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'list_statuses'" jsonschema_enum:"list,get,list_statuses"`
	ProjectKey string `json:"project_key,omitempty" jsonschema:"Project key or ID (for 'get', 'list_statuses')"`
	StartAt    int    `json:"start_at,omitempty" jsonschema:"Pagination start index (for 'list')"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"Maximum results to return (for 'list')"`
}

// ManageProjectsHandler handles project list and get operations.
func ManageProjectsHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageProjectsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageProjectsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListProjects(args.StartAt, args.MaxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list projects: %v", err)), nil, nil
			}
			type flatProject struct {
				Key         string `json:"key"`
				Name        string `json:"name"`
				ProjectType string `json:"project_type"`
				Lead        string `json:"lead,omitempty"`
			}
			flat := struct {
				Total    int           `json:"total"`
				Projects []flatProject `json:"projects"`
			}{Total: result.Total}
			for i := range result.Values {
				fp := flatProject{
					Key:         result.Values[i].Key,
					Name:        result.Values[i].Name,
					ProjectType: result.Values[i].ProjectType,
				}
				if result.Values[i].Lead != nil {
					fp.Lead = result.Values[i].Lead.DisplayName
				}
				flat.Projects = append(flat.Projects, fp)
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		case "get":
			if args.ProjectKey == "" {
				return ToolResultError("project_key is required for 'get' action"), nil, nil
			}
			result, err := c.GetProject(args.ProjectKey)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get project: %v", err)), nil, nil
			}
			type flatProject struct {
				Key         string   `json:"key"`
				Name        string   `json:"name"`
				Description string   `json:"description,omitempty"`
				ProjectType string   `json:"project_type"`
				Lead        string   `json:"lead,omitempty"`
				IssueTypes  []string `json:"issue_types,omitempty"`
			}
			fp := flatProject{
				Key:         result.Key,
				Name:        result.Name,
				Description: result.Description,
				ProjectType: result.ProjectType,
			}
			if result.Lead != nil {
				fp.Lead = result.Lead.DisplayName
			}
			for _, it := range result.IssueTypes {
				fp.IssueTypes = append(fp.IssueTypes, it.Name)
			}
			return ToolResultText(jira.SafeJSON(fp, 30000)), nil, nil

		case "list_statuses":
			if args.ProjectKey == "" {
				return ToolResultError("project_key is required for 'list_statuses' action"), nil, nil
			}
			result, err := c.GetProjectStatuses(args.ProjectKey)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get project statuses: %v", err)), nil, nil
			}
			type flatStatus struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Category string `json:"category,omitempty"`
			}
			type flatIssueTypeStatuses struct {
				IssueType string       `json:"issue_type"`
				Statuses  []flatStatus `json:"statuses"`
			}
			flat := struct {
				Total      int                     `json:"total"`
				IssueTypes []flatIssueTypeStatuses `json:"issue_types"`
			}{Total: len(result)}
			for _, its := range result {
				entry := flatIssueTypeStatuses{IssueType: its.Name}
				for _, s := range its.Statuses {
					fs := flatStatus{ID: s.ID, Name: s.Name}
					if s.StatusCategory.Name != "" {
						fs.Category = s.StatusCategory.Name
					}
					entry.Statuses = append(entry.Statuses, fs)
				}
				flat.IssueTypes = append(flat.IssueTypes, entry)
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, get, list_statuses", args.Action)), nil, nil
		}
	}
}
