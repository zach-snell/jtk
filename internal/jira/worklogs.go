package jira

import (
	"fmt"
	"net/url"
)

// ListWorklogs returns worklogs for an issue.
func (c *Client) ListWorklogs(issueKey string, startAt, maxResults int) (*WorklogPage, error) {
	params := url.Values{}
	if startAt > 0 {
		params.Set("startAt", fmt.Sprintf("%d", startAt))
	}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	path := fmt.Sprintf("/issue/%s/worklog", url.PathEscape(issueKey))
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return GetJSON[WorklogPage](c, path)
}

// AddWorklog adds a worklog entry to an issue.
func (c *Client) AddWorklog(issueKey string, req *AddWorklogRequest) (*Worklog, error) {
	path := fmt.Sprintf("/issue/%s/worklog", url.PathEscape(issueKey))

	data, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result Worklog
	if err := unmarshalJSON(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// BuildAddWorklogRequest constructs an AddWorklogRequest from simple parameters.
func BuildAddWorklogRequest(timeSpent, started, comment string) *AddWorklogRequest {
	req := &AddWorklogRequest{
		TimeSpent: timeSpent,
	}
	if started != "" {
		req.Started = started
	}
	if comment != "" {
		req.Comment = buildADFParagraph(comment)
	}
	return req
}
