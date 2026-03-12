package jira

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// jiraKeyRegex matches Jira issue keys like PROJ-123, ABC-1, etc.
var jiraKeyRegex = regexp.MustCompile(`([A-Z][A-Z0-9]+-\d+)`)

// DetectIssueKey runs `git rev-parse --abbrev-ref HEAD` and regex-matches
// a Jira key pattern from the branch name.
func DetectIssueKey() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository or git not available: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("could not determine branch name (detached HEAD?)")
	}

	matches := jiraKeyRegex.FindStringSubmatch(branch)
	if len(matches) < 2 {
		return "", fmt.Errorf("no Jira issue key found in branch name: %s", branch)
	}

	return matches[1], nil
}

// DetectIssueKeyOrArg returns the issue key from the argument if provided,
// otherwise attempts to detect it from the current git branch.
func DetectIssueKeyOrArg(arg string) (string, error) {
	if arg != "" {
		return arg, nil
	}
	return DetectIssueKey()
}
