package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ListBoards returns all agile boards.
func (c *Client) ListBoards(projectKeyOrID string, startAt, maxResults int) (*BoardsResponse, error) {
	params := url.Values{}
	if projectKeyOrID != "" {
		params.Set("projectKeyOrId", projectKeyOrID)
	}
	if startAt > 0 {
		params.Set("startAt", fmt.Sprintf("%d", startAt))
	}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	path := "/board"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return GetAgileJSON[BoardsResponse](c, path)
}

// GetBoard returns a single board by ID.
func (c *Client) GetBoard(boardID int) (*Board, error) {
	path := fmt.Sprintf("/board/%d", boardID)
	return GetAgileJSON[Board](c, path)
}

// ListSprints returns sprints for a board.
func (c *Client) ListSprints(boardID int, state string, startAt, maxResults int) (*SprintsResponse, error) {
	params := url.Values{}
	if state != "" {
		params.Set("state", state) // active, future, closed
	}
	if startAt > 0 {
		params.Set("startAt", fmt.Sprintf("%d", startAt))
	}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	path := fmt.Sprintf("/board/%d/sprint", boardID)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return GetAgileJSON[SprintsResponse](c, path)
}

// GetSprintIssues returns issues in a sprint.
func (c *Client) GetSprintIssues(sprintID int, startAt, maxResults int) (*SearchResult, error) {
	params := url.Values{}
	if startAt > 0 {
		params.Set("startAt", fmt.Sprintf("%d", startAt))
	}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	path := fmt.Sprintf("/sprint/%d/issue", sprintID)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return GetAgileJSON[SearchResult](c, path)
}

// GetBoardBacklog returns issues in the backlog of a board.
func (c *Client) GetBoardBacklog(boardID int, startAt, maxResults int) (*SearchResult, error) {
	params := url.Values{}
	if startAt > 0 {
		params.Set("startAt", fmt.Sprintf("%d", startAt))
	}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	path := fmt.Sprintf("/board/%d/backlog", boardID)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return GetAgileJSON[SearchResult](c, path)
}

// GetActiveSprint returns the active sprint for a board.
// Returns nil if no active sprint is found.
func (c *Client) GetActiveSprint(boardID int) (*Sprint, error) {
	result, err := c.ListSprints(boardID, "active", 0, 1)
	if err != nil {
		return nil, err
	}
	if len(result.Values) == 0 {
		return nil, nil
	}
	return &result.Values[0], nil
}

// SearchSprintByName searches sprints for a board by name (case-insensitive contains).
func (c *Client) SearchSprintByName(boardID int, name string) ([]Sprint, error) {
	// Fetch all sprints (paginating if necessary)
	var allSprints []Sprint
	startAt := 0
	for {
		result, err := c.ListSprints(boardID, "", startAt, 50)
		if err != nil {
			return nil, err
		}
		allSprints = append(allSprints, result.Values...)
		if result.IsLast || len(result.Values) == 0 {
			break
		}
		startAt += len(result.Values)
	}

	// Filter by name
	nameLower := strings.ToLower(name)
	var matched []Sprint
	for _, sp := range allSprints {
		if strings.Contains(strings.ToLower(sp.Name), nameLower) {
			matched = append(matched, sp)
		}
	}
	return matched, nil
}

// CreateSprint creates a new sprint for a board.
func (c *Client) CreateSprint(boardID int, name, startDate, endDate, goal string) (*Sprint, error) {
	req := &CreateSprintRequest{
		Name:          name,
		OriginBoardID: boardID,
	}
	if startDate != "" {
		req.StartDate = startDate
	}
	if endDate != "" {
		req.EndDate = endDate
	}
	if goal != "" {
		req.Goal = goal
	}

	data, err := c.PostAgile("/sprint", req)
	if err != nil {
		return nil, err
	}

	var result Sprint
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling sprint: %w", err)
	}
	return &result, nil
}

// MoveIssuesToSprint moves one or more issues to a sprint.
func (c *Client) MoveIssuesToSprint(sprintID int, issueKeys []string) error {
	path := fmt.Sprintf("/sprint/%d/issue", sprintID)
	req := &MoveIssuesToSprintRequest{
		Issues: issueKeys,
	}
	_, err := c.PostAgile(path, req)
	return err
}
