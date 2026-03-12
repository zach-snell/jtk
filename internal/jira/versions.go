package jira

import (
	"fmt"
	"net/url"
)

// ListVersions returns all versions for a project.
func (c *Client) ListVersions(projectKeyOrID string) ([]Version, error) {
	path := fmt.Sprintf("/project/%s/versions", url.PathEscape(projectKeyOrID))

	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result []Version
	if err := unmarshalJSON(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetVersion returns a single version by ID.
func (c *Client) GetVersion(versionID string) (*Version, error) {
	path := fmt.Sprintf("/version/%s", url.PathEscape(versionID))
	return GetJSON[Version](c, path)
}
