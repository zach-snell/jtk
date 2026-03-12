package jira

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// GetIssue retrieves a single issue by key.
func (c *Client) GetIssue(issueKey string) (*Issue, error) {
	path := fmt.Sprintf("/issue/%s", url.PathEscape(issueKey))
	return GetJSON[Issue](c, path)
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(req *CreateIssueRequest) (*CreatedIssue, error) {
	data, err := c.Post("/issue", req)
	if err != nil {
		return nil, err
	}

	var result CreatedIssue
	if err := unmarshalJSON(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateIssue updates an existing issue's fields.
func (c *Client) UpdateIssue(issueKey string, req *UpdateIssueRequest) error {
	path := fmt.Sprintf("/issue/%s", url.PathEscape(issueKey))
	_, err := c.Put(path, req)
	return err
}

// AssignIssue assigns an issue to a user by account ID.
// Pass empty string to unassign.
func (c *Client) AssignIssue(issueKey, accountID string) error {
	path := fmt.Sprintf("/issue/%s/assignee", url.PathEscape(issueKey))
	body := map[string]interface{}{}
	if accountID != "" {
		body["accountId"] = accountID
	} else {
		body["accountId"] = nil
	}
	_, err := c.Put(path, body)
	return err
}

// BuildCreateIssueRequest constructs a CreateIssueRequest from simple parameters.
func BuildCreateIssueRequest(projectKey, summary, issueType, description, priority, assigneeID, parentKey string, labels []string) *CreateIssueRequest {
	req := &CreateIssueRequest{
		Fields: CreateIssueFields{
			Project:   ProjectRef{Key: projectKey},
			Summary:   summary,
			IssueType: IssueTypeRef{Name: issueType},
		},
	}

	if description != "" {
		req.Fields.Description = buildADFParagraph(description)
	}
	if priority != "" {
		req.Fields.Priority = &PriorityRef{Name: priority}
	}
	if assigneeID != "" {
		req.Fields.Assignee = &UserRef{AccountID: assigneeID}
	}
	if parentKey != "" {
		req.Fields.Parent = &IssueRef{Key: parentKey}
	}
	if len(labels) > 0 {
		req.Fields.Labels = labels
	}

	return req
}

// BuildUpdateIssueRequest constructs an UpdateIssueRequest from optional fields.
func BuildUpdateIssueRequest(summary, description, priority, assigneeID string, labels []string) *UpdateIssueRequest {
	fields := make(map[string]interface{})

	if summary != "" {
		fields["summary"] = summary
	}
	if description != "" {
		fields["description"] = buildADFParagraph(description)
	}
	if priority != "" {
		fields["priority"] = map[string]string{"name": priority}
	}
	if assigneeID != "" {
		if assigneeID == "unassigned" || assigneeID == "none" {
			fields["assignee"] = nil
		} else {
			fields["assignee"] = map[string]string{"accountId": assigneeID}
		}
	}
	if labels != nil {
		fields["labels"] = labels
	}

	return &UpdateIssueRequest{Fields: fields}
}

// urlPattern matches http/https URLs in text.
var urlPattern = regexp.MustCompile(`https?://[^\s<>"{}|\\^\x60]+`)

// buildADFParagraph wraps plain text in ADF document format.
// Handles literal \n sequences, actual newlines, and auto-links URLs.
func buildADFParagraph(text string) map[string]interface{} {
	// Normalize literal \n sequences to real newlines
	text = strings.ReplaceAll(text, `\n`, "\n")

	paragraphs := strings.Split(text, "\n")
	content := make([]interface{}, 0, len(paragraphs))

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		content = append(content, map[string]interface{}{
			"type":    "paragraph",
			"content": buildADFInlineContent(p),
		})
	}

	return map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

// buildADFInlineContent splits a line into text and link nodes.
func buildADFInlineContent(line string) []interface{} {
	matches := urlPattern.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return []interface{}{adfText(line)}
	}

	var nodes []interface{}
	cursor := 0
	for _, m := range matches {
		// Text before the URL
		if m[0] > cursor {
			nodes = append(nodes, adfText(line[cursor:m[0]]))
		}
		// The URL as a clickable link
		u := line[m[0]:m[1]]
		nodes = append(nodes, adfLink(u))
		cursor = m[1]
	}
	// Trailing text after last URL
	if cursor < len(line) {
		nodes = append(nodes, adfText(line[cursor:]))
	}
	return nodes
}

func adfText(t string) map[string]interface{} {
	return map[string]interface{}{"type": "text", "text": t}
}

func adfLink(u string) map[string]interface{} {
	return map[string]interface{}{
		"type": "text",
		"text": u,
		"marks": []interface{}{
			map[string]interface{}{
				"type":  "link",
				"attrs": map[string]interface{}{"href": u},
			},
		},
	}
}
