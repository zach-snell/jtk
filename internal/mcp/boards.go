package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageBoardsArgs struct {
	Action     string `json:"action" jsonschema:"Action to perform: 'list_boards', 'get_board', 'list_sprints', 'get_sprint_issues', 'get_backlog', 'get_active_sprint', 'search_sprints', 'create_sprint', 'move_to_sprint'" jsonschema_enum:"list_boards,get_board,list_sprints,get_sprint_issues,get_backlog,get_active_sprint,search_sprints,create_sprint,move_to_sprint"`
	ProjectKey string `json:"project_key,omitempty" jsonschema:"Filter boards by project key (for 'list_boards')"`
	BoardID    int    `json:"board_id,omitempty" jsonschema:"Board ID (for 'get_board', 'list_sprints', 'get_backlog', 'get_active_sprint', 'search_sprints', 'create_sprint')"`
	SprintID   int    `json:"sprint_id,omitempty" jsonschema:"Sprint ID (for 'get_sprint_issues', 'move_to_sprint')"`
	State      string `json:"state,omitempty" jsonschema:"Sprint state filter: active, future, closed (for 'list_sprints')"`
	Query      string `json:"query,omitempty" jsonschema:"Sprint name search query (for 'search_sprints')"`
	Name       string `json:"name,omitempty" jsonschema:"Sprint name (for 'create_sprint')"`
	StartDate  string `json:"start_date,omitempty" jsonschema:"Sprint start date ISO 8601 (for 'create_sprint')"`
	EndDate    string `json:"end_date,omitempty" jsonschema:"Sprint end date ISO 8601 (for 'create_sprint')"`
	Goal       string `json:"goal,omitempty" jsonschema:"Sprint goal (for 'create_sprint')"`
	IssueKeys  string `json:"issue_keys,omitempty" jsonschema:"Comma-separated issue keys (for 'move_to_sprint')"`
	StartAt    int    `json:"start_at,omitempty" jsonschema:"Pagination start index"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"Maximum results to return"`
}

// ManageBoardsHandler handles agile board and sprint operations.
func ManageBoardsHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageBoardsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageBoardsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list_boards":
			result, err := c.ListBoards(args.ProjectKey, args.StartAt, args.MaxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list boards: %v", err)), nil, nil
			}
			type flatBoard struct {
				ID         int    `json:"id"`
				Name       string `json:"name"`
				Type       string `json:"type"`
				ProjectKey string `json:"project_key,omitempty"`
			}
			flat := struct {
				Total  int         `json:"total"`
				Boards []flatBoard `json:"boards"`
			}{Total: result.Total}
			for _, b := range result.Values {
				fb := flatBoard{ID: b.ID, Name: b.Name, Type: b.Type}
				if b.Location != nil {
					fb.ProjectKey = b.Location.ProjectKey
				}
				flat.Boards = append(flat.Boards, fb)
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		case "get_board":
			if args.BoardID == 0 {
				return ToolResultError("board_id is required for 'get_board' action"), nil, nil
			}
			result, err := c.GetBoard(args.BoardID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get board: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(result, 30000)), nil, nil

		case "list_sprints":
			if args.BoardID == 0 {
				return ToolResultError("board_id is required for 'list_sprints' action"), nil, nil
			}
			result, err := c.ListSprints(args.BoardID, args.State, args.StartAt, args.MaxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list sprints: %v", err)), nil, nil
			}
			type flatSprint struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				State     string `json:"state"`
				StartDate string `json:"start_date,omitempty"`
				EndDate   string `json:"end_date,omitempty"`
				Goal      string `json:"goal,omitempty"`
			}
			flat := struct {
				Total   int          `json:"total"`
				Sprints []flatSprint `json:"sprints"`
			}{Total: result.Total}
			for _, sp := range result.Values {
				flat.Sprints = append(flat.Sprints, flatSprint{
					ID:        sp.ID,
					Name:      sp.Name,
					State:     sp.State,
					StartDate: sp.StartDate,
					EndDate:   sp.EndDate,
					Goal:      sp.Goal,
				})
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		case "get_sprint_issues":
			if args.SprintID == 0 {
				return ToolResultError("sprint_id is required for 'get_sprint_issues' action"), nil, nil
			}
			result, err := c.GetSprintIssues(args.SprintID, args.StartAt, args.MaxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get sprint issues: %v", err)), nil, nil
			}
			return ToolResultText(flattenSearchResult(result)), nil, nil

		case "get_backlog":
			if args.BoardID == 0 {
				return ToolResultError("board_id is required for 'get_backlog' action"), nil, nil
			}
			result, err := c.GetBoardBacklog(args.BoardID, args.StartAt, args.MaxResults)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get board backlog: %v", err)), nil, nil
			}
			return ToolResultText(flattenSearchResult(result)), nil, nil

		case "get_active_sprint":
			if args.BoardID == 0 {
				return ToolResultError("board_id is required for 'get_active_sprint' action"), nil, nil
			}
			sprint, err := c.GetActiveSprint(args.BoardID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get active sprint: %v", err)), nil, nil
			}
			if sprint == nil {
				return ToolResultText(jira.SafeJSON(map[string]interface{}{
					"board_id": args.BoardID,
					"message":  "no active sprint found",
				}, 30000)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(map[string]interface{}{
				"id":         sprint.ID,
				"name":       sprint.Name,
				"state":      sprint.State,
				"start_date": sprint.StartDate,
				"end_date":   sprint.EndDate,
				"goal":       sprint.Goal,
			}, 30000)), nil, nil

		case "search_sprints":
			if args.BoardID == 0 || args.Query == "" {
				return ToolResultError("board_id and query are required for 'search_sprints' action"), nil, nil
			}
			sprints, err := c.SearchSprintByName(args.BoardID, args.Query)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to search sprints: %v", err)), nil, nil
			}
			type flatSprint struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				State     string `json:"state"`
				StartDate string `json:"start_date,omitempty"`
				EndDate   string `json:"end_date,omitempty"`
				Goal      string `json:"goal,omitempty"`
			}
			flat := struct {
				Query   string       `json:"query"`
				Total   int          `json:"total"`
				Sprints []flatSprint `json:"sprints"`
			}{Query: args.Query, Total: len(sprints)}
			for _, sp := range sprints {
				flat.Sprints = append(flat.Sprints, flatSprint{
					ID:        sp.ID,
					Name:      sp.Name,
					State:     sp.State,
					StartDate: sp.StartDate,
					EndDate:   sp.EndDate,
					Goal:      sp.Goal,
				})
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		case "create_sprint":
			if args.BoardID == 0 || args.Name == "" {
				return ToolResultError("board_id and name are required for 'create_sprint' action"), nil, nil
			}
			sprint, err := c.CreateSprint(args.BoardID, args.Name, args.StartDate, args.EndDate, args.Goal)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create sprint: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(map[string]interface{}{
				"id":         sprint.ID,
				"name":       sprint.Name,
				"state":      sprint.State,
				"start_date": sprint.StartDate,
				"end_date":   sprint.EndDate,
				"goal":       sprint.Goal,
				"status":     "sprint created",
			}, 30000)), nil, nil

		case "move_to_sprint":
			if args.SprintID == 0 || args.IssueKeys == "" {
				return ToolResultError("sprint_id and issue_keys are required for 'move_to_sprint' action"), nil, nil
			}
			var keys []string
			for _, k := range strings.Split(args.IssueKeys, ",") {
				trimmed := strings.TrimSpace(k)
				if trimmed != "" {
					keys = append(keys, trimmed)
				}
			}
			if len(keys) == 0 {
				return ToolResultError("issue_keys must contain at least one issue key"), nil, nil
			}
			if err := c.MoveIssuesToSprint(args.SprintID, keys); err != nil {
				return ToolResultError(fmt.Sprintf("failed to move issues to sprint: %v", err)), nil, nil
			}
			return ToolResultText(jira.SafeJSON(map[string]interface{}{
				"sprint_id": args.SprintID,
				"issues":    keys,
				"status":    "issues moved to sprint",
			}, 30000)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list_boards, get_board, list_sprints, get_sprint_issues, get_backlog, get_active_sprint, search_sprints, create_sprint, move_to_sprint", args.Action)), nil, nil
		}
	}
}
