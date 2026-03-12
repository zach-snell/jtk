package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts registers all MCP prompts on the server.
func registerPrompts(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "standup_report",
		Description: "Generate a standup report from recent Jira activity",
		Arguments: []*mcp.PromptArgument{
			{Name: "project_key", Description: "Jira project key", Required: true},
			{Name: "days", Description: "Number of days to look back (default: 1)"},
		},
	}, standupPromptHandler)

	s.AddPrompt(&mcp.Prompt{
		Name:        "sprint_status",
		Description: "Get current sprint status and progress",
		Arguments: []*mcp.PromptArgument{
			{Name: "board_id", Description: "Jira board ID", Required: true},
		},
	}, sprintStatusPromptHandler)

	s.AddPrompt(&mcp.Prompt{
		Name:        "release_notes",
		Description: "Generate release notes from a Jira version/release",
		Arguments: []*mcp.PromptArgument{
			{Name: "project_key", Description: "Jira project key", Required: true},
			{Name: "version_name", Description: "Version name to generate notes for", Required: true},
		},
	}, releaseNotesPromptHandler)

	s.AddPrompt(&mcp.Prompt{
		Name:        "dev_tree",
		Description: "Analyze development work (branches, PRs, commits) for an issue and its subtasks",
		Arguments: []*mcp.PromptArgument{
			{Name: "issue_key", Description: "Parent issue key", Required: true},
		},
	}, devTreePromptHandler)
}

func standupPromptHandler(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	projectKey := req.Params.Arguments["project_key"]
	if projectKey == "" {
		return nil, fmt.Errorf("project_key is required")
	}
	days := req.Params.Arguments["days"]
	if days == "" {
		days = "1"
	}

	return &mcp.GetPromptResult{
		Description: "Generate standup report for " + projectKey,
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Generate a standup report for project %s.

Use the manage_search tool with action 'jql' to find issues updated in the last %s day(s):
JQL: project = %s AND updated >= -%sd ORDER BY updated DESC

For each issue found, summarize:
1. What was done (based on status changes and comments)
2. What's in progress
3. Any blockers

Format as a clean standup report with sections: Done, In Progress, Blocked.`, projectKey, days, projectKey, days),
				},
			},
		},
	}, nil
}

func sprintStatusPromptHandler(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	boardID := req.Params.Arguments["board_id"]
	if boardID == "" {
		return nil, fmt.Errorf("board_id is required")
	}

	return &mcp.GetPromptResult{
		Description: "Sprint status for board " + boardID,
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Get the current sprint status for board %s.

Steps:
1. Use manage_boards with action 'get_active_sprint' and board_id %s to find the active sprint
2. Use manage_boards with action 'get_sprint_issues' with the sprint_id to get all issues
3. Categorize issues by status (To Do, In Progress, Done, etc.)
4. Calculate completion percentage (done issues / total issues)
5. List any blocked or stalled items

Present as a sprint progress dashboard:
- Sprint name, dates, and goal
- Progress bar (done/total)
- Issues by status category
- At-risk items (in progress but not updated recently)
- Remaining capacity summary`, boardID, boardID),
				},
			},
		},
	}, nil
}

func releaseNotesPromptHandler(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	projectKey := req.Params.Arguments["project_key"]
	versionName := req.Params.Arguments["version_name"]
	if projectKey == "" || versionName == "" {
		return nil, fmt.Errorf("project_key and version_name are required")
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Release notes for %s %s", projectKey, versionName),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Generate release notes for project %s, version "%s".

Steps:
1. Use manage_search with action 'jql' to find all issues in this release:
   JQL: project = %s AND fixVersion = "%s" ORDER BY issuetype ASC, priority DESC
2. Group issues by type (Features/Stories, Bug Fixes, Improvements, Tasks)
3. For each issue, include the key, summary, and any relevant details

Format the output as professional release notes:
- Version header with date
- Highlights section (top 3-5 most impactful changes)
- Detailed sections by category (New Features, Bug Fixes, Improvements, Other)
- Each entry: [ISSUE-KEY] Summary description
- Breaking changes section (if any issues mention breaking changes)`, projectKey, versionName, projectKey, versionName),
				},
			},
		},
	}, nil
}

func devTreePromptHandler(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	issueKey := req.Params.Arguments["issue_key"]
	if issueKey == "" {
		return nil, fmt.Errorf("issue_key is required")
	}

	return &mcp.GetPromptResult{
		Description: "Development tree for " + issueKey,
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Analyze the development work for issue %s and its subtasks.

Steps:
1. Use manage_issues with action 'get' for %s to get issue details
2. Use manage_search with action 'jql' to find subtasks:
   JQL: parent = %s ORDER BY key ASC
3. For the parent issue and each subtask, use manage_devinfo with action 'get_dev_info' to get branches, PRs, and commits

Present as a development tree:
- Parent issue: key, summary, status
  - Branches: name, last commit
  - PRs: title, status (open/merged/declined), reviewers
  - Commits: count, latest message
  - For each subtask:
    - Issue key, summary, status
    - Its branches, PRs, and commits (same format)

Highlight:
- PRs that need review (open with no reviewers)
- Stale branches (no recent commits)
- Issues without any development activity
- Overall progress summary`, issueKey, issueKey, issueKey),
				},
			},
		},
	}, nil
}
