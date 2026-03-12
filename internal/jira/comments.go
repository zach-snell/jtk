package jira

import (
	"fmt"
	"net/url"
)

// ListComments returns comments on an issue.
func (c *Client) ListComments(issueKey string, startAt, maxResults int) (*CommentPage, error) {
	params := url.Values{}
	if startAt > 0 {
		params.Set("startAt", fmt.Sprintf("%d", startAt))
	}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	path := fmt.Sprintf("/issue/%s/comment", url.PathEscape(issueKey))
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return GetJSON[CommentPage](c, path)
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(issueKey, body string) (*Comment, error) {
	path := fmt.Sprintf("/issue/%s/comment", url.PathEscape(issueKey))
	req := &AddCommentRequest{
		Body: buildADFParagraph(body),
	}

	data, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result Comment
	if err := unmarshalJSON(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// EditComment updates an existing comment on an issue.
func (c *Client) EditComment(issueKey, commentID, body string) (*Comment, error) {
	path := fmt.Sprintf("/issue/%s/comment/%s",
		url.PathEscape(issueKey), url.PathEscape(commentID))
	req := &AddCommentRequest{
		Body: buildADFParagraph(body),
	}

	data, err := c.Put(path, req)
	if err != nil {
		return nil, err
	}

	var result Comment
	if err := unmarshalJSON(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
