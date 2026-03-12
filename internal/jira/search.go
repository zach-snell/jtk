package jira

import (
	"encoding/json"
	"fmt"
)

// SearchJQLRequest is the payload for POST /rest/api/3/search/jql.
type SearchJQLRequest struct {
	JQL           string   `json:"jql"`
	MaxResults    int      `json:"maxResults,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// defaultSearchFields are the fields requested in every JQL search.
// The POST /search/jql endpoint returns only "id" by default — we must be explicit.
var defaultSearchFields = []string{
	"summary", "status", "issuetype", "priority", "assignee", "reporter",
	"project", "created", "updated", "description", "labels", "components",
	"resolution", "fixVersions", "parent",
}

// SearchJQL performs a JQL search using the new POST /rest/api/3/search/jql endpoint.
// The old GET /rest/api/3/search?jql=... endpoint was deprecated and removed May 2025.
func (c *Client) SearchJQL(jql string, startAt, maxResults int) (*SearchResult, error) {
	reqBody := SearchJQLRequest{
		JQL:    jql,
		Fields: defaultSearchFields,
	}
	if maxResults > 0 {
		reqBody.MaxResults = maxResults
	}
	// Note: startAt is ignored for the new endpoint — use nextPageToken for pagination.
	// We keep the parameter for API compatibility but the new endpoint uses cursor-based paging.

	data, err := c.Post("/search/jql", reqBody)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling search response: %w", err)
	}

	return &result, nil
}

// SearchJQLPaginated performs a JQL search with cursor-based pagination using nextPageToken.
func (c *Client) SearchJQLPaginated(jql string, maxResults int, nextPageToken string) (*SearchResult, error) {
	reqBody := SearchJQLRequest{
		JQL:    jql,
		Fields: defaultSearchFields,
	}
	if maxResults > 0 {
		reqBody.MaxResults = maxResults
	}
	if nextPageToken != "" {
		reqBody.NextPageToken = nextPageToken
	}

	data, err := c.Post("/search/jql", reqBody)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling search response: %w", err)
	}

	return &result, nil
}

// QuickSearch performs a text-based search using JQL text matching.
func (c *Client) QuickSearch(text, projectKey string, maxResults int) (*SearchResult, error) {
	jql := fmt.Sprintf("text ~ %q", text)
	if projectKey != "" {
		jql = fmt.Sprintf("project = %q AND %s", projectKey, jql)
	}
	jql += " ORDER BY updated DESC"

	return c.SearchJQL(jql, 0, maxResults)
}

// unmarshalJSON is a helper for unmarshaling JSON data.
func unmarshalJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}
	return nil
}

// UnmarshalJSONPublic is the exported version of unmarshalJSON for use by other packages.
func UnmarshalJSONPublic(data []byte, v interface{}) error {
	return unmarshalJSON(data, v)
}
