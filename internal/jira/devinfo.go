package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// DevInfoResult is the aggregated development information for an issue.
type DevInfoResult struct {
	IssueKey     string          `json:"issue_key"`
	Branches     []DevBranch     `json:"branches"`
	PullRequests []DevPR         `json:"pull_requests"`
	Repositories []DevRepository `json:"repositories"`
	Builds       []DevBuild      `json:"builds"`
}

// DevInfoOptions controls which data types to include.
type DevInfoOptions struct {
	IncludeBranches bool
	IncludePRs      bool
	IncludeCommits  bool
	IncludeBuilds   bool
}

// endpointPair represents a (appType, dataType) pair discovered from the summary.
type endpointPair struct {
	appType  string
	dataType string
}

// GetDevelopmentInfo retrieves development information for an issue using the 3-step
// undocumented dev-status API:
//  1. Convert issue key to numeric ID via /rest/api/3/issue/{key}?fields=id
//  2. Discover VCS application types via /rest/dev-status/latest/issue/summary
//  3. Fetch details per (appType, dataType) via /rest/dev-status/latest/issue/detail
func (c *Client) GetDevelopmentInfo(issueKey string, opts *DevInfoOptions) (*DevInfoResult, error) {
	// Default: include all if nothing specified
	if opts == nil {
		opts = &DevInfoOptions{
			IncludeBranches: true,
			IncludePRs:      true,
			IncludeCommits:  true,
			IncludeBuilds:   true,
		}
	}
	if !opts.IncludeBranches && !opts.IncludePRs && !opts.IncludeCommits && !opts.IncludeBuilds {
		opts.IncludeBranches = true
		opts.IncludePRs = true
		opts.IncludeCommits = true
		opts.IncludeBuilds = true
	}

	// Step 1: Convert issue key to numeric ID
	issue, err := c.GetIssue(issueKey)
	if err != nil {
		return nil, fmt.Errorf("getting issue %s: %w", issueKey, err)
	}
	numericID := issue.ID

	// Step 2: Discover which (appType, dataType) pairs have data
	summaryPath := fmt.Sprintf("/rest/dev-status/latest/issue/summary?issueId=%s", url.QueryEscape(numericID))
	summaryData, err := c.GetDevStatus(summaryPath)
	if err != nil {
		return nil, fmt.Errorf("fetching dev-status summary: %w", err)
	}

	endpoints, err := parseSummaryEndpoints(summaryData)
	if err != nil {
		return nil, fmt.Errorf("parsing dev-status summary: %w", err)
	}

	result := &DevInfoResult{
		IssueKey:     issueKey,
		Branches:     []DevBranch{},
		PullRequests: []DevPR{},
		Repositories: []DevRepository{},
		Builds:       []DevBuild{},
	}

	if len(endpoints) == 0 {
		return result, nil
	}

	// Step 3: Fetch detail for each discovered (appType, dataType) pair
	for _, ep := range endpoints {
		detailPath := fmt.Sprintf(
			"/rest/dev-status/latest/issue/detail?issueId=%s&applicationType=%s&dataType=%s",
			url.QueryEscape(numericID),
			url.QueryEscape(ep.appType),
			url.QueryEscape(ep.dataType),
		)

		detailData, err := c.GetDevStatus(detailPath)
		if err != nil {
			continue // silently skip individual failures
		}

		var detail DevStatusDetail
		if err := json.Unmarshal(detailData, &detail); err != nil {
			continue
		}

		if len(detail.Errors) > 0 {
			continue
		}

		// Aggregate data from all detail items
		for _, item := range detail.Detail {
			if opts.IncludeBranches {
				result.Branches = append(result.Branches, item.Branches...)
			}
			if opts.IncludePRs {
				result.PullRequests = append(result.PullRequests, item.PullRequests...)
			}
			if opts.IncludeCommits {
				result.Repositories = append(result.Repositories, item.Repositories...)
			}
			if opts.IncludeBuilds {
				result.Builds = append(result.Builds, item.Builds...)
			}
		}
	}

	return result, nil
}

// parseSummaryEndpoints parses the dev-status summary response and extracts
// (appType, dataType) pairs that have data.
// The summary structure is: {"summary": {"<dataType>": {"byInstanceType": {"<appType>": ...}}}}
func parseSummaryEndpoints(data []byte) ([]endpointPair, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	summaryRaw, ok := raw["summary"]
	if !ok {
		return nil, nil
	}

	var summary map[string]json.RawMessage
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		return nil, err
	}

	dataTypes := []string{"repository", "branch", "pullrequest", "build"}
	var pairs []endpointPair

	for _, dt := range dataTypes {
		dtRaw, ok := summary[dt]
		if !ok {
			continue
		}

		var dtObj map[string]json.RawMessage
		if err := json.Unmarshal(dtRaw, &dtObj); err != nil {
			continue
		}

		byInstanceRaw, ok := dtObj["byInstanceType"]
		if !ok {
			continue
		}

		var byInstance map[string]json.RawMessage
		if err := json.Unmarshal(byInstanceRaw, &byInstance); err != nil {
			continue
		}

		for appType := range byInstance {
			pairs = append(pairs, endpointPair{
				appType:  appType,
				dataType: dt,
			})
		}
	}

	return pairs, nil
}

// GetMyPermissions checks what permissions the current token has.
func (c *Client) GetMyPermissions(permissions []string) (*PermissionsResponse, error) {
	params := url.Values{}
	for _, p := range permissions {
		params.Add("permissions", p)
	}

	path := "/mypermissions?" + params.Encode()
	return GetJSON[PermissionsResponse](c, path)
}
