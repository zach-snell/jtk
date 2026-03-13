package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zach-snell/jtk/internal/jira"
)

var issuesCmd = &cobra.Command{
	Use:     "issues",
	Aliases: []string{"issue", "i"},
	Short:   "Manage Jira issues",
}

var issuesGetCmd = &cobra.Command{
	Use:   "get [issue-key]",
	Short: "Get details for a specific issue (auto-detects from git branch if omitted)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey, err := ResolveIssueKey(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		issue, err := client.GetIssue(issueKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, issue, func() {
			fmt.Printf("%s: %s\n", issue.Key, issue.Fields.Summary)
			if issue.Fields.Status != nil {
				KV("Status", issue.Fields.Status.Name)
			}
			if issue.Fields.IssueType != nil {
				KV("Type", issue.Fields.IssueType.Name)
			}
			if issue.Fields.Priority != nil {
				KV("Priority", issue.Fields.Priority.Name)
			}
			if issue.Fields.Assignee != nil {
				KV("Assignee", issue.Fields.Assignee.DisplayName)
			} else {
				KV("Assignee", "unassigned")
			}
			if issue.Fields.Reporter != nil {
				KV("Reporter", issue.Fields.Reporter.DisplayName)
			}
			if issue.Fields.Project != nil {
				KV("Project", issue.Fields.Project.Key+" — "+issue.Fields.Project.Name)
			}
			if len(issue.Fields.Labels) > 0 {
				KV("Labels", strings.Join(issue.Fields.Labels, ", "))
			}
			if issue.Fields.Sprint != nil {
				KV("Sprint", issue.Fields.Sprint.Name)
			}
			if issue.Fields.Parent != nil {
				KV("Parent", issue.Fields.Parent.Key)
			}
			if issue.Fields.Description != nil {
				desc := jira.ADFToPlainText(issue.Fields.Description)
				if desc != "" {
					KV("Description", Truncate(desc, 120))
				}
			}
			KV("Created", FormatTime(issue.Fields.Created))
			KV("Updated", FormatTime(issue.Fields.Updated))
		})
	},
}

var issuesSearchCmd = &cobra.Command{
	Use:   "search <JQL query>",
	Short: "Search issues using JQL",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jql := strings.Join(args, " ")
		maxResults, _ := cmd.Flags().GetInt("max")

		client := getClient()
		result, err := client.SearchJQL(jql, 0, maxResults)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Issues) == 0 {
				fmt.Println("No issues found.")
				return
			}
			t := NewTable()
			t.Header("Key", "Summary", "Status", "Type", "Priority", "Assignee")
			for _, issue := range result.Issues {
				assignee := "unassigned"
				if issue.Fields.Assignee != nil {
					assignee = issue.Fields.Assignee.DisplayName
				}
				status := ""
				if issue.Fields.Status != nil {
					status = issue.Fields.Status.Name
				}
				issueType := ""
				if issue.Fields.IssueType != nil {
					issueType = issue.Fields.IssueType.Name
				}
				priority := ""
				if issue.Fields.Priority != nil {
					priority = issue.Fields.Priority.Name
				}
				t.Row(
					issue.Key,
					Truncate(issue.Fields.Summary, 50),
					status,
					issueType,
					priority,
					assignee,
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Total, result.StartAt, len(result.Issues), result.Total > result.StartAt+len(result.Issues))
		})
	},
}

var issuesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		summary, _ := cmd.Flags().GetString("summary")
		issueType, _ := cmd.Flags().GetString("type")
		description, _ := cmd.Flags().GetString("description")
		priority, _ := cmd.Flags().GetString("priority")
		assignee, _ := cmd.Flags().GetString("assignee")
		parent, _ := cmd.Flags().GetString("parent")
		labelsStr, _ := cmd.Flags().GetString("labels")

		if project == "" || summary == "" {
			fmt.Fprintln(os.Stderr, "Error: --project and --summary are required")
			os.Exit(1)
		}
		if issueType == "" {
			issueType = "Task"
		}

		var labels []string
		if labelsStr != "" {
			for _, l := range strings.Split(labelsStr, ",") {
				labels = append(labels, strings.TrimSpace(l))
			}
		}

		client := getClient()
		req := jira.BuildCreateIssueRequest(jira.CreateIssueParams{
			ProjectKey:  project,
			Summary:     summary,
			IssueType:   issueType,
			Description: description,
			Priority:    priority,
			AssigneeID:  assignee,
			ParentKey:   parent,
			Labels:      labels,
		})
		result, err := client.CreateIssue(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Created %s\n", result.Key)
			KV("ID", result.ID)
			KV("URL", result.Self)
		})
	},
}

var issuesTransitionCmd = &cobra.Command{
	Use:   "transition [issue-key] <transition-name>",
	Short: "Transition an issue to a new state",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var issueKey, transitionName string
		if len(args) == 2 {
			issueKey = args[0]
			transitionName = args[1]
		} else {
			// Single arg: try to detect issue key from git
			key, err := jira.DetectIssueKey()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: provide an issue key or run from a branch with a Jira key: %v\n", err)
				os.Exit(1)
			}
			issueKey = key
			transitionName = args[0]
		}

		client := getClient()
		if err := client.TransitionIssue(issueKey, transitionName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Transitioned %s to %q\n", issueKey, transitionName)
	},
}

var issuesCommentCmd = &cobra.Command{
	Use:   "comment [issue-key] <comment-text>",
	Short: "Add a comment to an issue",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var issueKey, commentText string
		if len(args) == 2 {
			issueKey = args[0]
			commentText = args[1]
		} else {
			key, err := jira.DetectIssueKey()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: provide an issue key or run from a branch with a Jira key: %v\n", err)
				os.Exit(1)
			}
			issueKey = key
			commentText = args[0]
		}

		client := getClient()
		result, err := client.AddComment(issueKey, commentText)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Comment added to %s (id: %s)\n", issueKey, result.ID)
	},
}

var issuesMoveCmd = &cobra.Command{
	Use:   "move [issue-key] --project <target-project-key>",
	Short: "Move an issue to a different project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey, err := ResolveIssueKey(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		targetProject, _ := cmd.Flags().GetString("project")
		if targetProject == "" {
			fmt.Fprintln(os.Stderr, "Error: --project is required")
			os.Exit(1)
		}
		targetType, _ := cmd.Flags().GetString("type")

		client := getClient()
		if err := client.MoveIssue(issueKey, targetProject, targetType); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Re-fetch to show updated issue
		issue, err := client.GetIssue(issueKey)
		if err != nil {
			fmt.Printf("Issue %s moved to project %s\n", issueKey, targetProject)
			return
		}

		PrintOrJSON(cmd, issue, func() {
			fmt.Printf("Moved %s to project %s\n", issueKey, targetProject)
			KV("Key", issue.Key)
			if issue.Fields.Project != nil {
				KV("Project", issue.Fields.Project.Key+" — "+issue.Fields.Project.Name)
			}
			if issue.Fields.IssueType != nil {
				KV("Type", issue.Fields.IssueType.Name)
			}
			if issue.Fields.Status != nil {
				KV("Status", issue.Fields.Status.Name)
			}
		})
	},
}

func init() {
	RootCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesGetCmd)
	issuesCmd.AddCommand(issuesSearchCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesTransitionCmd)
	issuesCmd.AddCommand(issuesCommentCmd)
	issuesCmd.AddCommand(issuesMoveCmd)

	issuesSearchCmd.Flags().Int("max", 20, "Maximum results to return")

	issuesCreateCmd.Flags().StringP("project", "P", "", "Project key (required)")
	issuesCreateCmd.Flags().StringP("summary", "s", "", "Issue summary/title (required)")
	issuesCreateCmd.Flags().StringP("type", "t", "Task", "Issue type: Story, Bug, Task, Epic, Sub-task")
	issuesCreateCmd.Flags().StringP("description", "d", "", "Issue description")
	issuesCreateCmd.Flags().String("priority", "", "Priority: Highest, High, Medium, Low, Lowest")
	issuesCreateCmd.Flags().String("assignee", "", "Assignee account ID")
	issuesCreateCmd.Flags().String("parent", "", "Parent issue key (for sub-tasks)")
	issuesCreateCmd.Flags().String("labels", "", "Comma-separated labels")

	issuesMoveCmd.Flags().StringP("project", "P", "", "Target project key (required)")
	issuesMoveCmd.Flags().StringP("type", "t", "", "Target issue type (optional, keeps current if omitted)")
}
