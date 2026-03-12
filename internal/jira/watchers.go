package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetWatchers returns the watchers for an issue.
func (c *Client) GetWatchers(issueKey string) (*WatchersResponse, error) {
	path := fmt.Sprintf("/issue/%s/watchers", url.PathEscape(issueKey))
	return GetJSON[WatchersResponse](c, path)
}

// AddWatcher adds a user as a watcher on an issue.
// accountID is the Atlassian account ID of the user to add.
func (c *Client) AddWatcher(issueKey, accountID string) error {
	path := fmt.Sprintf("/issue/%s/watchers", url.PathEscape(issueKey))
	body, err := json.Marshal(accountID) // API expects a bare JSON string
	if err != nil {
		return fmt.Errorf("marshaling account ID: %w", err)
	}

	resp, err := c.do("POST", c.baseURL+path, body, "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to add watcher (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// RemoveWatcher removes a user from watching an issue.
func (c *Client) RemoveWatcher(issueKey, accountID string) error {
	path := fmt.Sprintf("/issue/%s/watchers?accountId=%s",
		url.PathEscape(issueKey), url.QueryEscape(accountID))
	return c.Delete(path)
}
