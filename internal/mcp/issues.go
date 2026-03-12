package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageIssuesArgs struct {
	Action      string `json:"action" jsonschema:"Action to perform: 'get', 'create', 'update', 'assign', 'transition', 'add_comment', 'list_comments', 'delete', 'link', 'get_links', 'get_history', 'list_types'" jsonschema_enum:"get,create,update,assign,transition,add_comment,list_comments,delete,link,get_links,get_history,list_types"`
	IssueKey    string `json:"issue_key,omitempty" jsonschema:"Jira issue key (e.g., PROJ-123). Required for most actions"`
	ProjectKey  string `json:"project_key,omitempty" jsonschema:"Project key (for 'create', 'list_types')"`
	ProjectID   string `json:"project_id,omitempty" jsonschema:"Project ID (for 'list_types' — use project_key or project_id)"`
	Summary     string `json:"summary,omitempty" jsonschema:"Issue summary/title (for 'create', 'update')"`
	Description string `json:"description,omitempty" jsonschema:"Issue description in markdown (for 'create', 'update'). Supports: # headings, **bold**, *italic*, ~~strikethrough~~, [links](url), - bullet lists, 1. numbered lists, > blockquotes, tables, and fenced code blocks. URLs are auto-linked."`
	IssueType   string `json:"issue_type,omitempty" jsonschema:"Issue type: Story, Bug, Task, Epic, Sub-task (for 'create')"`
	Priority    string `json:"priority,omitempty" jsonschema:"Priority: Highest, High, Medium, Low, Lowest (for 'create', 'update')"`
	AssigneeID  string `json:"assignee_id,omitempty" jsonschema:"Assignee account ID (for 'create', 'update', 'assign'). Use 'unassigned' to remove"`
	ParentKey   string `json:"parent_key,omitempty" jsonschema:"Parent issue key (for 'create')"`
	Labels      string `json:"labels,omitempty" jsonschema:"Comma-separated labels (for 'create', 'update')"`
	Transition  string `json:"transition,omitempty" jsonschema:"Target transition name (for 'transition'), e.g. 'In Progress', 'Done'"`
	Comment     string `json:"comment,omitempty" jsonschema:"Comment body in markdown (for 'add_comment', 'link'). Supports: **bold**, *italic*, ~~strikethrough~~, [links](url), - lists, > blockquotes, and fenced code blocks. URLs are auto-linked."`
	LinkType    string `json:"link_type,omitempty" jsonschema:"Link type name (for 'link'), e.g. 'Blocks', 'Duplicate', 'Relates'"`
	InwardKey   string `json:"inward_key,omitempty" jsonschema:"Inward issue key (for 'link') — the issue that IS affected"`
	OutwardKey  string `json:"outward_key,omitempty" jsonschema:"Outward issue key (for 'link') — the issue that CAUSES the effect"`
	StartAt     int    `json:"start_at,omitempty" jsonschema:"Pagination start (for 'list_comments', 'get_history')"`
	MaxResults  int    `json:"max_results,omitempty" jsonschema:"Max results to return"`
}

// ManageIssuesHandler handles the consolidated issue operations.
func ManageIssuesHandler(c *jira.Client, perms map[string]bool) func(context.Context, *mcp.CallToolRequest, ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "get":
			return handleGetIssue(c, args)
		case "create":
			return handleCreateIssue(c, perms, args)
		case "update":
			return handleUpdateIssue(c, perms, args)
		case "assign":
			return handleAssignIssue(c, perms, args)
		case "transition":
			return handleTransitionIssue(c, perms, args)
		case "add_comment":
			return handleAddComment(c, perms, args)
		case "list_comments":
			return handleListComments(c, args)
		case "delete":
			return handleDeleteIssue(c, perms, args)
		case "link":
			return handleLinkIssues(c, perms, args)
		case "get_links":
			return handleGetLinks(c, args)
		case "get_history":
			return handleGetHistory(c, args)
		case "list_types":
			return handleListTypes(c, args)
		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: get, create, update, assign, transition, add_comment, list_comments, delete, link, get_links, get_history, list_types", args.Action)), nil, nil
		}
	}
}

func handleGetIssue(c *jira.Client, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return ToolResultError("issue_key is required for 'get' action"), nil, nil
	}
	issue, err := c.GetIssue(args.IssueKey)
	if err != nil {
		return ToolResultError(fmt.Sprintf("failed to get issue: %v", err)), nil, nil
	}
	flat := jira.FlattenIssueFromTyped(issue)
	return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil
}

func handleCreateIssue(c *jira.Client, perms map[string]bool, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if !perms["CREATE_ISSUES"] {
		return ToolResultError("token lacks CREATE_ISSUES permission"), nil, nil
	}
	if args.ProjectKey == "" || args.Summary == "" {
		return ToolResultError("project_key and summary are required for 'create' action"), nil, nil
	}
	issueType := args.IssueType
	if issueType == "" {
		issueType = "Task"
	}
	labels := parseLabels(args.Labels)
	createReq := jira.BuildCreateIssueRequest(
		args.ProjectKey, args.Summary, issueType,
		args.Description, args.Priority, args.AssigneeID,
		args.ParentKey, labels,
	)
	result, err := c.CreateIssue(createReq)
	if err != nil {
		return ToolResultError(fmt.Sprintf("failed to create issue: %v", err)), nil, nil
	}
	return ToolResultText(jira.SafeJSON(result, 30000)), nil, nil
}

func handleUpdateIssue(c *jira.Client, perms map[string]bool, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if !perms["EDIT_ISSUES"] {
		return ToolResultError("token lacks EDIT_ISSUES permission"), nil, nil
	}
	if args.IssueKey == "" {
		return ToolResultError("issue_key is required for 'update' action"), nil, nil
	}
	labels := parseLabels(args.Labels)
	updateReq := jira.BuildUpdateIssueRequest(
		args.Summary, args.Description, args.Priority,
		args.AssigneeID, labels,
	)
	if err := c.UpdateIssue(args.IssueKey, updateReq); err != nil {
		return ToolResultError(fmt.Sprintf("failed to update issue: %v", err)), nil, nil
	}
	return ToolResultText(fmt.Sprintf("Issue %s updated successfully", args.IssueKey)), nil, nil
}

func handleAssignIssue(c *jira.Client, perms map[string]bool, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if !perms["ASSIGN_ISSUES"] {
		return ToolResultError("token lacks ASSIGN_ISSUES permission"), nil, nil
	}
	if args.IssueKey == "" {
		return ToolResultError("issue_key is required for 'assign' action"), nil, nil
	}
	accountID := args.AssigneeID
	if accountID == "unassigned" || accountID == "none" {
		accountID = ""
	}
	if err := c.AssignIssue(args.IssueKey, accountID); err != nil {
		return ToolResultError(fmt.Sprintf("failed to assign issue: %v", err)), nil, nil
	}
	if accountID == "" {
		return ToolResultText(fmt.Sprintf("Issue %s unassigned", args.IssueKey)), nil, nil
	}
	return ToolResultText(fmt.Sprintf("Issue %s assigned to %s", args.IssueKey, accountID)), nil, nil
}

func handleTransitionIssue(c *jira.Client, perms map[string]bool, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if !perms["TRANSITION_ISSUES"] {
		return ToolResultError("token lacks TRANSITION_ISSUES permission"), nil, nil
	}
	if args.IssueKey == "" || args.Transition == "" {
		return ToolResultError("issue_key and transition are required for 'transition' action"), nil, nil
	}
	if err := c.TransitionIssue(args.IssueKey, args.Transition); err != nil {
		return ToolResultError(fmt.Sprintf("failed to transition issue: %v", err)), nil, nil
	}
	return ToolResultText(fmt.Sprintf("Issue %s transitioned to %q", args.IssueKey, args.Transition)), nil, nil
}

func handleAddComment(c *jira.Client, perms map[string]bool, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if !perms["ADD_COMMENTS"] {
		return ToolResultError("token lacks ADD_COMMENTS permission"), nil, nil
	}
	if args.IssueKey == "" || args.Comment == "" {
		return ToolResultError("issue_key and comment are required for 'add_comment' action"), nil, nil
	}
	result, err := c.AddComment(args.IssueKey, args.Comment)
	if err != nil {
		return ToolResultError(fmt.Sprintf("failed to add comment: %v", err)), nil, nil
	}
	return ToolResultText(jira.SafeJSON(map[string]string{
		"id":      result.ID,
		"created": result.Created,
		"status":  "comment added",
	}, 30000)), nil, nil
}

func handleListComments(c *jira.Client, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return ToolResultError("issue_key is required for 'list_comments' action"), nil, nil
	}
	result, err := c.ListComments(args.IssueKey, args.StartAt, args.MaxResults)
	if err != nil {
		return ToolResultError(fmt.Sprintf("failed to list comments: %v", err)), nil, nil
	}
	// Flatten comments
	type flatComment struct {
		ID      string `json:"id"`
		Author  string `json:"author"`
		Body    string `json:"body"`
		Created string `json:"created"`
	}
	flat := struct {
		Total    int           `json:"total"`
		Comments []flatComment `json:"comments"`
	}{
		Total: result.Total,
	}
	for _, cm := range result.Comments {
		author := "unknown"
		if cm.Author != nil {
			author = cm.Author.DisplayName
		}
		flat.Comments = append(flat.Comments, flatComment{
			ID:      cm.ID,
			Author:  author,
			Body:    jira.ADFToPlainText(cm.Body),
			Created: cm.Created,
		})
	}
	return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil
}

func handleDeleteIssue(c *jira.Client, perms map[string]bool, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if !perms["DELETE_ISSUES"] {
		return ToolResultError("token lacks DELETE_ISSUES permission"), nil, nil
	}
	if args.IssueKey == "" {
		return ToolResultError("issue_key is required for 'delete' action"), nil, nil
	}
	if err := c.DeleteIssue(args.IssueKey); err != nil {
		return ToolResultError(fmt.Sprintf("failed to delete issue: %v", err)), nil, nil
	}
	return ToolResultText(fmt.Sprintf("Issue %s deleted", args.IssueKey)), nil, nil
}

func handleLinkIssues(c *jira.Client, perms map[string]bool, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if !perms["LINK_ISSUES"] {
		return ToolResultError("token lacks LINK_ISSUES permission"), nil, nil
	}
	if args.LinkType == "" || args.InwardKey == "" || args.OutwardKey == "" {
		return ToolResultError("link_type, inward_key, and outward_key are required for 'link' action"), nil, nil
	}
	linkReq := jira.BuildCreateIssueLinkRequest(args.LinkType, args.InwardKey, args.OutwardKey, args.Comment)
	if err := c.CreateIssueLink(linkReq); err != nil {
		return ToolResultError(fmt.Sprintf("failed to link issues: %v", err)), nil, nil
	}
	return ToolResultText(fmt.Sprintf("Linked %s -[%s]-> %s", args.OutwardKey, args.LinkType, args.InwardKey)), nil, nil
}

func handleGetLinks(c *jira.Client, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return ToolResultError("issue_key is required for 'get_links' action"), nil, nil
	}
	// Get full issue which includes issuelinks in fields
	data, err := c.Get(fmt.Sprintf("/issue/%s?fields=issuelinks", args.IssueKey))
	if err != nil {
		return ToolResultError(fmt.Sprintf("failed to get issue links: %v", err)), nil, nil
	}
	// Parse raw to extract issuelinks
	var raw map[string]interface{}
	if err := jira.UnmarshalJSONPublic(data, &raw); err != nil {
		return ToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil, nil
	}
	fields, _ := raw["fields"].(map[string]interface{})
	links, _ := fields["issuelinks"].([]interface{})

	type flatLink struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Relation string `json:"relation"`
		IssueKey string `json:"issue_key"`
		Summary  string `json:"summary"`
	}
	flat := struct {
		IssueKey string     `json:"issue_key"`
		Total    int        `json:"total"`
		Links    []flatLink `json:"links"`
	}{IssueKey: args.IssueKey, Total: len(links)}

	for _, l := range links {
		link, ok := l.(map[string]interface{})
		if !ok {
			continue
		}
		fl := flatLink{}
		if id, ok := link["id"].(string); ok {
			fl.ID = id
		}
		if lt, ok := link["type"].(map[string]interface{}); ok {
			fl.Type, _ = lt["name"].(string)
		}
		if inward, ok := link["inwardIssue"].(map[string]interface{}); ok {
			fl.IssueKey, _ = inward["key"].(string)
			if f, ok := inward["fields"].(map[string]interface{}); ok {
				fl.Summary, _ = f["summary"].(string)
			}
			fl.Relation = "inward"
		} else if outward, ok := link["outwardIssue"].(map[string]interface{}); ok {
			fl.IssueKey, _ = outward["key"].(string)
			if f, ok := outward["fields"].(map[string]interface{}); ok {
				fl.Summary, _ = f["summary"].(string)
			}
			fl.Relation = "outward"
		}
		flat.Links = append(flat.Links, fl)
	}
	return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil
}

func handleGetHistory(c *jira.Client, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	if args.IssueKey == "" {
		return ToolResultError("issue_key is required for 'get_history' action"), nil, nil
	}
	result, err := c.GetIssueChangelog(args.IssueKey, args.StartAt, args.MaxResults)
	if err != nil {
		return ToolResultError(fmt.Sprintf("failed to get issue history: %v", err)), nil, nil
	}
	type flatChange struct {
		Field string `json:"field"`
		From  string `json:"from"`
		To    string `json:"to"`
	}
	type flatEntry struct {
		ID      string       `json:"id"`
		Author  string       `json:"author"`
		Created string       `json:"created"`
		Changes []flatChange `json:"changes"`
	}
	flat := struct {
		Total   int         `json:"total"`
		History []flatEntry `json:"history"`
	}{Total: result.Total}
	for _, h := range result.Histories {
		entry := flatEntry{
			ID:      h.ID,
			Created: h.Created,
		}
		if h.Author != nil {
			entry.Author = h.Author.DisplayName
		}
		for _, item := range h.Items {
			entry.Changes = append(entry.Changes, flatChange{
				Field: item.Field,
				From:  item.FromString,
				To:    item.ToString,
			})
		}
		flat.History = append(flat.History, entry)
	}
	return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil
}

func handleListTypes(c *jira.Client, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	// Need project ID; if only key given, fetch project first
	projectID := args.ProjectID
	if projectID == "" && args.ProjectKey != "" {
		proj, err := c.GetProject(args.ProjectKey)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get project: %v", err)), nil, nil
		}
		projectID = proj.ID
	}
	if projectID == "" {
		return ToolResultError("project_key or project_id is required for 'list_types' action"), nil, nil
	}
	result, err := c.ListIssueTypes(projectID)
	if err != nil {
		return ToolResultError(fmt.Sprintf("failed to list issue types: %v", err)), nil, nil
	}
	type flatType struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Subtask     bool   `json:"subtask"`
	}
	flat := struct {
		Total int        `json:"total"`
		Types []flatType `json:"types"`
	}{Total: len(result)}
	for _, t := range result {
		flat.Types = append(flat.Types, flatType{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Subtask:     t.Subtask,
		})
	}
	return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil
}

// parseLabels splits a comma-separated labels string into a slice.
func parseLabels(labelsCSV string) []string {
	if labelsCSV == "" {
		return nil
	}
	var labels []string
	for _, l := range strings.Split(labelsCSV, ",") {
		labels = append(labels, strings.TrimSpace(l))
	}
	return labels
}
