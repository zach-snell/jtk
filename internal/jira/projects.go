package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ProjectsResponse represents the paginated project list response.
type ProjectsResponse struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	IsLast     bool      `json:"isLast"`
	Values     []Project `json:"values"`
}

// ListProjects returns all projects visible to the current user.
func (c *Client) ListProjects(startAt, maxResults int) (*ProjectsResponse, error) {
	params := url.Values{}
	if startAt > 0 {
		params.Set("startAt", fmt.Sprintf("%d", startAt))
	}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	path := "/project/search"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return GetJSON[ProjectsResponse](c, path)
}

// GetProject returns a single project by key or ID.
func (c *Client) GetProject(keyOrID string) (*Project, error) {
	path := fmt.Sprintf("/project/%s", url.PathEscape(keyOrID))
	return GetJSON[Project](c, path)
}

// CreateProjectRequest is the payload for creating a Jira project.
type CreateProjectRequest struct {
	Key                string `json:"key"`
	Name               string `json:"name"`
	ProjectTypeKey     string `json:"projectTypeKey"`
	ProjectTemplateKey string `json:"projectTemplateKey,omitempty"`
	Description        string `json:"description,omitempty"`
	LeadAccountID      string `json:"leadAccountId"`
	AssigneeType       string `json:"assigneeType,omitempty"`
}

// createProjectResponse is the raw API response from POST /project.
// The create endpoint returns id as a number, unlike other endpoints
// that return it as a string.
type createProjectResponse struct {
	ID  int    `json:"id"`
	Key string `json:"key"`
}

// CreateProject creates a new Jira project.
// Requires "Administer Jira" global permission.
func (c *Client) CreateProject(req CreateProjectRequest) (*Project, error) {
	data, err := c.Post("/project", req)
	if err != nil {
		return nil, err
	}

	var raw createProjectResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshaling created project: %w", err)
	}

	// Re-fetch the full project details so we get a consistent Project struct.
	return c.GetProject(raw.Key)
}

// GetProjectStatuses returns all statuses grouped by issue type for a project.
func (c *Client) GetProjectStatuses(keyOrID string) ([]IssueTypeWithStatuses, error) {
	path := fmt.Sprintf("/project/%s/statuses", url.PathEscape(keyOrID))

	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result []IssueTypeWithStatuses
	if err := unmarshalJSON(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
