package cli

import (
	"fmt"

	"github.com/zach-snell/jtk/internal/jira"
)

// ResolveIssueKey resolves an issue key from the CLI args.
// If no arg is provided, it attempts to detect from the current git branch.
func ResolveIssueKey(args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}

	key, err := jira.DetectIssueKey()
	if err != nil {
		return "", fmt.Errorf("no issue key provided and could not detect from git branch: %v", err)
	}
	return key, nil
}
