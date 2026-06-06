package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Automation rules live on a separate Atlassian host
// (api.atlassian.com/automation/public/...), not the Jira REST gateway. The
// Automation Rule Management API is GA and reachable with an API token over
// Basic auth (it does NOT support OAuth2/Forge apps).

// automationBaseURL returns the automation API base, fetching the cloud ID if
// the client does not already have it (classic tokens don't carry one).
func (c *Client) automationBaseURL() (string, error) {
	if c.cloudID == "" {
		id, err := FetchCloudID(c.domain)
		if err != nil {
			return "", fmt.Errorf("automation API requires the cloud ID: %w", err)
		}
		c.cloudID = id
	}
	return fmt.Sprintf("https://api.atlassian.com/automation/public/jira/%s/rest/v1", c.cloudID), nil
}

// automationDo issues a request to the automation API and returns the raw body.
func (c *Client) automationDo(method, path string, body interface{}) ([]byte, error) {
	base, err := c.automationBaseURL()
	if err != nil {
		return nil, err
	}

	var bodyData []byte
	contentType := ""
	if body != nil {
		switch b := body.(type) {
		case []byte:
			bodyData = b
		default:
			marshaled, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshaling body: %w", err)
			}
			bodyData = marshaled
		}
		contentType = "application/json"
	}

	resp, err := c.do(method, base+path, bodyData, contentType)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, data)
	}
	return data, nil
}

// ListAutomationRules returns the raw rule-summary list JSON.
func (c *Client) ListAutomationRules() ([]byte, error) {
	return c.automationDo(http.MethodGet, "/rule/summary", nil)
}

// GetAutomationRule returns the raw rule JSON for a single rule.
func (c *Client) GetAutomationRule(ruleUUID string) ([]byte, error) {
	return c.automationDo(http.MethodGet, "/rule/"+ruleUUID, nil)
}

// SetAutomationRuleState enables or disables a rule.
func (c *Client) SetAutomationRuleState(ruleUUID string, enabled bool) error {
	value := "DISABLED"
	if enabled {
		value = "ENABLED"
	}
	_, err := c.automationDo(http.MethodPut, "/rule/"+ruleUUID+"/state", map[string]string{"value": value})
	return err
}

// DeleteAutomationRule deletes a rule (the rule must be disabled first).
func (c *Client) DeleteAutomationRule(ruleUUID string) error {
	_, err := c.automationDo(http.MethodDelete, "/rule/"+ruleUUID, nil)
	return err
}

// CreateAutomationRule posts a rule definition verbatim. The body must match
// the POST /rule schema: {"rule": {...}, "connections": [...]}. Build a rule in
// the UI and `automation get` it as a template, then tweak and create.
func (c *Client) CreateAutomationRule(ruleJSON []byte) ([]byte, error) {
	return c.automationDo(http.MethodPost, "/rule", ruleJSON)
}
