package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageVersionsArgs struct {
	Action      string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'create'" jsonschema_enum:"list,get,create"`
	ProjectKey  string `json:"project_key,omitempty" jsonschema:"Project key (for 'list', 'create')"`
	VersionID   string `json:"version_id,omitempty" jsonschema:"Version ID (for 'get')"`
	Name        string `json:"name,omitempty" jsonschema:"Version name (required for 'create')"`
	Description string `json:"description,omitempty" jsonschema:"Version description (for 'create')"`
	StartDate   string `json:"start_date,omitempty" jsonschema:"Start date YYYY-MM-DD (for 'create')"`
	ReleaseDate string `json:"release_date,omitempty" jsonschema:"Release date YYYY-MM-DD (for 'create')"`
}

// ManageVersionsHandler handles project version operations.
func ManageVersionsHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageVersionsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageVersionsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			if args.ProjectKey == "" {
				return ToolResultError("project_key is required for 'list' action"), nil, nil
			}
			result, err := c.ListVersions(args.ProjectKey)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list versions: %v", err)), nil, nil
			}
			type flatVersion struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description,omitempty"`
				Released    bool   `json:"released"`
				Archived    bool   `json:"archived"`
				ReleaseDate string `json:"release_date,omitempty"`
				StartDate   string `json:"start_date,omitempty"`
			}
			flat := struct {
				Total    int           `json:"total"`
				Versions []flatVersion `json:"versions"`
			}{Total: len(result)}
			for _, v := range result {
				flat.Versions = append(flat.Versions, flatVersion{
					ID:          v.ID,
					Name:        v.Name,
					Description: v.Description,
					Released:    v.Released,
					Archived:    v.Archived,
					ReleaseDate: v.ReleaseDate,
					StartDate:   v.StartDate,
				})
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		case "get":
			if args.VersionID == "" {
				return ToolResultError("version_id is required for 'get' action"), nil, nil
			}
			result, err := c.GetVersion(args.VersionID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get version: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(result, 30000)), nil, nil

		case "create":
			if args.ProjectKey == "" || args.Name == "" {
				return ToolResultError("project_key and name are required for 'create' action"), nil, nil
			}
			result, err := c.CreateVersion(&jira.CreateVersionRequest{
				Name:        args.Name,
				ProjectKey:  args.ProjectKey,
				Description: args.Description,
				StartDate:   args.StartDate,
				ReleaseDate: args.ReleaseDate,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create version: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(result, 30000)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, get, create", args.Action)), nil, nil
		}
	}
}
