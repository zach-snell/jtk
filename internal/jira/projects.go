package jira

import (
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
