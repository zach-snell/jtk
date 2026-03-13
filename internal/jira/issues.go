package jira

import (
	"fmt"
	"net/url"
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

// MoveIssue moves an issue to a different project by updating its project field.
// Optionally changes the issue type if the target project doesn't have the same type.
// Note: This only works with company-managed (classic) projects. Team-managed (next-gen)
// projects silently ignore project field changes via the REST API.
func (c *Client) MoveIssue(issueKey, targetProjectKey, issueType string) error {
	fields := map[string]interface{}{
		"project": map[string]string{"key": targetProjectKey},
	}
	if issueType != "" {
		fields["issuetype"] = map[string]string{"name": issueType}
	}

	req := &UpdateIssueRequest{Fields: fields}
	if err := c.UpdateIssue(issueKey, req); err != nil {
		return err
	}

	// Verify the move actually happened — team-managed projects silently ignore this
	issue, err := c.GetIssue(issueKey)
	if err != nil {
		return nil // Update succeeded, can't verify but assume success
	}
	if issue.Fields.Project != nil && issue.Fields.Project.Key != targetProjectKey {
		return fmt.Errorf("move failed: issue %s is still in project %s (team-managed/next-gen projects do not support cross-project moves via REST API)", issueKey, issue.Fields.Project.Key)
	}
	return nil
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

// CreateIssueParams bundles all parameters for creating an issue.
type CreateIssueParams struct {
	ProjectKey  string
	Summary     string
	IssueType   string
	Description string
	Priority    string
	AssigneeID  string
	ParentKey   string
	Labels      []string
	Components  []string
	FixVersions []string
	DueDate     string
}

// BuildCreateIssueRequest constructs a CreateIssueRequest from parameters.
func BuildCreateIssueRequest(p CreateIssueParams) *CreateIssueRequest {
	req := &CreateIssueRequest{
		Fields: CreateIssueFields{
			Project:   ProjectRef{Key: p.ProjectKey},
			Summary:   p.Summary,
			IssueType: IssueTypeRef{Name: p.IssueType},
		},
	}

	if p.Description != "" {
		req.Fields.Description = buildADFParagraph(p.Description)
	}
	if p.Priority != "" {
		req.Fields.Priority = &PriorityRef{Name: p.Priority}
	}
	if p.AssigneeID != "" {
		req.Fields.Assignee = &UserRef{AccountID: p.AssigneeID}
	}
	if p.ParentKey != "" {
		req.Fields.Parent = &IssueRef{Key: p.ParentKey}
	}
	if len(p.Labels) > 0 {
		req.Fields.Labels = p.Labels
	}
	if len(p.Components) > 0 {
		refs := make([]ComponentRef, len(p.Components))
		for i, name := range p.Components {
			refs[i] = ComponentRef{Name: name}
		}
		req.Fields.Components = refs
	}
	if len(p.FixVersions) > 0 {
		refs := make([]VersionRef, len(p.FixVersions))
		for i, name := range p.FixVersions {
			refs[i] = VersionRef{Name: name}
		}
		req.Fields.FixVersions = refs
	}
	if p.DueDate != "" {
		req.Fields.DueDate = p.DueDate
	}

	return req
}

// UpdateIssueParams bundles all parameters for updating an issue.
type UpdateIssueParams struct {
	Summary     string
	Description string
	Priority    string
	AssigneeID  string
	Labels      []string
	Components  []string
	FixVersions []string
	DueDate     string
}

// BuildUpdateIssueRequest constructs an UpdateIssueRequest from parameters.
func BuildUpdateIssueRequest(p UpdateIssueParams) *UpdateIssueRequest {
	fields := make(map[string]interface{})

	if p.Summary != "" {
		fields["summary"] = p.Summary
	}
	if p.Description != "" {
		fields["description"] = buildADFParagraph(p.Description)
	}
	if p.Priority != "" {
		fields["priority"] = map[string]string{"name": p.Priority}
	}
	if p.AssigneeID != "" {
		if p.AssigneeID == "unassigned" || p.AssigneeID == "none" {
			fields["assignee"] = nil
		} else {
			fields["assignee"] = map[string]string{"accountId": p.AssigneeID}
		}
	}
	if p.Labels != nil {
		fields["labels"] = p.Labels
	}
	if p.Components != nil {
		refs := make([]map[string]string, len(p.Components))
		for i, name := range p.Components {
			refs[i] = map[string]string{"name": name}
		}
		fields["components"] = refs
	}
	if p.FixVersions != nil {
		refs := make([]map[string]string, len(p.FixVersions))
		for i, name := range p.FixVersions {
			refs[i] = map[string]string{"name": name}
		}
		fields["fixVersions"] = refs
	}
	if p.DueDate != "" {
		fields["duedate"] = p.DueDate
	}

	return &UpdateIssueRequest{Fields: fields}
}

// buildADFParagraph converts text (plain or markdown) to an ADF document.
// Delegates to the full markdown-to-ADF converter in markdown.go.
func buildADFParagraph(text string) map[string]interface{} {
	return MarkdownToADF(text)
}
