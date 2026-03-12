package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageVersionsArgs struct {
	Action     string `json:"action" jsonschema:"Action to perform: 'list', 'get'" jsonschema_enum:"list,get"`
	ProjectKey string `json:"project_key,omitempty" jsonschema:"Project key (for 'list')"`
	VersionID  string `json:"version_id,omitempty" jsonschema:"Version ID (for 'get')"`
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

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, get", args.Action)), nil, nil
		}
	}
}
