package jira

import (
	"fmt"
	"net/url"
	"strings"
)

// GetTransitions returns the available transitions for an issue.
func (c *Client) GetTransitions(issueKey string) (*TransitionsResponse, error) {
	path := fmt.Sprintf("/issue/%s/transitions", url.PathEscape(issueKey))
	return GetJSON[TransitionsResponse](c, path)
}

// TransitionIssue transitions an issue to a new state.
// The transitionName is matched case-insensitively against:
//  1. The transition name (e.g., "Start Progress")
//  2. The target status name (e.g., "In Progress")
func (c *Client) TransitionIssue(issueKey, transitionName string) error {
	transitions, err := c.GetTransitions(issueKey)
	if err != nil {
		return fmt.Errorf("getting transitions: %w", err)
	}

	var transitionID string
	var available []string

	// First pass: match against transition name
	for _, t := range transitions.Transitions {
		available = append(available, t.Name)
		if t.To != nil && t.To.Name != "" {
			available = append(available, fmt.Sprintf("%s (→ %s)", t.Name, t.To.Name))
		}
		if strings.EqualFold(t.Name, transitionName) {
			transitionID = t.ID
			break
		}
	}

	// Second pass: match against target status name (transition.to.name)
	if transitionID == "" {
		for _, t := range transitions.Transitions {
			if t.To != nil && strings.EqualFold(t.To.Name, transitionName) {
				transitionID = t.ID
				break
			}
		}
	}

	if transitionID == "" {
		return fmt.Errorf("transition %q not found for %s. Available: %s",
			transitionName, issueKey, strings.Join(available, ", "))
	}

	path := fmt.Sprintf("/issue/%s/transitions", url.PathEscape(issueKey))
	req := &TransitionRequest{
		Transition: TransitionRef{ID: transitionID},
	}
	_, err = c.Post(path, req)
	return err
}
