package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageUsersArgs struct {
	Action     string `json:"action" jsonschema:"Action to perform: 'get_current', 'search', 'get'" jsonschema_enum:"get_current,search,get"`
	Query      string `json:"query,omitempty" jsonschema:"Search query — display name, email, etc. (for 'search')"`
	AccountID  string `json:"account_id,omitempty" jsonschema:"User account ID (for 'get')"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"Maximum results to return (for 'search', default 20)"`
}

// ManageUsersHandler handles user lookup operations.
func ManageUsersHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageUsersArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageUsersArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "get_current":
			user, err := c.GetCurrentUser()
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get current user: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(flattenUserDetail(user), 30000)), nil, nil

		case "search":
			if args.Query == "" {
				return ToolResultError("query is required for 'search' action"), nil, nil
			}
			maxResults := args.MaxResults
			if maxResults <= 0 {
				maxResults = 20
			}
			users, err := c.SearchUsers(args.Query, maxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to search users: %v", err)), nil, nil
			}
			type flatUser struct {
				AccountID   string `json:"account_id"`
				DisplayName string `json:"display_name"`
				Email       string `json:"email,omitempty"`
				Active      bool   `json:"active"`
			}
			flat := struct {
				Total int        `json:"total"`
				Users []flatUser `json:"users"`
			}{Total: len(users)}
			for _, u := range users {
				flat.Users = append(flat.Users, flatUser{
					AccountID:   u.AccountID,
					DisplayName: u.DisplayName,
					Email:       u.EmailAddress,
					Active:      u.Active,
				})
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		case "get":
			if args.AccountID == "" {
				return ToolResultError("account_id is required for 'get' action"), nil, nil
			}
			user, err := c.GetUser(args.AccountID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get user: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(flattenUserDetail(user), 30000)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: get_current, search, get", args.Action)), nil, nil
		}
	}
}

func flattenUserDetail(u *jira.UserDetail) map[string]interface{} {
	result := map[string]interface{}{
		"account_id":   u.AccountID,
		"display_name": u.DisplayName,
		"active":       u.Active,
	}
	if u.EmailAddress != "" {
		result["email"] = u.EmailAddress
	}
	if u.AccountType != "" {
		result["account_type"] = u.AccountType
	}
	if u.TimeZone != "" {
		result["time_zone"] = u.TimeZone
	}
	if u.Locale != "" {
		result["locale"] = u.Locale
	}
	return result
}
