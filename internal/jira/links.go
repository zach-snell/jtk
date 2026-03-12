package jira

import (
	"fmt"
	"net/url"
)

// GetIssueLinkTypes returns all available issue link types.
func (c *Client) GetIssueLinkTypes() (*IssueLinkTypesResponse, error) {
	return GetJSON[IssueLinkTypesResponse](c, "/issueLinkType")
}

// GetIssueLinks returns the links on an issue (from the issue's fields).
func (c *Client) GetIssueLinks(issueKey string) ([]IssueLink, error) {
	path := fmt.Sprintf("/issue/%s?fields=issuelinks", url.PathEscape(issueKey))
	type resp struct {
		Fields struct {
			IssueLinks []IssueLink `json:"issuelinks"`
		} `json:"fields"`
	}
	r, err := GetJSON[resp](c, path)
	if err != nil {
		return nil, err
	}
	return r.Fields.IssueLinks, nil
}

// CreateIssueLink links two issues together.
func (c *Client) CreateIssueLink(req *CreateIssueLinkRequest) error {
	_, err := c.Post("/issueLink", req)
	return err
}

// BuildCreateIssueLinkRequest constructs a CreateIssueLinkRequest from simple parameters.
func BuildCreateIssueLinkRequest(linkType, inwardKey, outwardKey, comment string) *CreateIssueLinkRequest {
	req := &CreateIssueLinkRequest{
		Type:         IssueLinkTypeRef{Name: linkType},
		InwardIssue:  IssueRef{Key: inwardKey},
		OutwardIssue: IssueRef{Key: outwardKey},
	}
	if comment != "" {
		req.Comment = map[string]interface{}{
			"body": buildADFParagraph(comment),
		}
	}
	return req
}

// GetIssueChangelog retrieves the changelog (history) for an issue.
func (c *Client) GetIssueChangelog(issueKey string, startAt, maxResults int) (*ChangelogPage, error) {
	path := fmt.Sprintf("/issue/%s/changelog", issueKey)
	params := ""
	if startAt > 0 || maxResults > 0 {
		path += "?"
		if startAt > 0 {
			params += fmt.Sprintf("startAt=%d", startAt)
		}
		if maxResults > 0 {
			if params != "" {
				params += "&"
			}
			params += fmt.Sprintf("maxResults=%d", maxResults)
		}
		path += params
	}
	return GetJSON[ChangelogPage](c, path)
}

// DeleteIssue deletes an issue by key.
func (c *Client) DeleteIssue(issueKey string) error {
	path := fmt.Sprintf("/issue/%s", issueKey)
	return c.Delete(path)
}

// ListIssueTypes returns available issue types for a project by project ID.
func (c *Client) ListIssueTypes(projectID string) ([]IssueType, error) {
	path := fmt.Sprintf("/issuetype/project?projectId=%s", projectID)

	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result []IssueType
	if err := unmarshalJSON(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
