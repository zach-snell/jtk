package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetCurrentUser returns the currently authenticated user.
func (c *Client) GetCurrentUser() (*UserDetail, error) {
	return GetJSON[UserDetail](c, "/myself")
}

// SearchUsers searches for users by display name, email, or username.
func (c *Client) SearchUsers(query string, maxResults int) ([]UserDetail, error) {
	params := url.Values{}
	params.Set("query", query)
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	path := "/user/search?" + params.Encode()
	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result []UserDetail
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling user search: %w", err)
	}
	return result, nil
}

// GetUser returns a user by account ID.
func (c *Client) GetUser(accountID string) (*UserDetail, error) {
	params := url.Values{}
	params.Set("accountId", accountID)
	path := "/user?" + params.Encode()
	return GetJSON[UserDetail](c, path)
}
