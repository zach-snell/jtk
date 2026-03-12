package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/jtk/internal/jira"
)

var worklogsCmd = &cobra.Command{
	Use:     "worklogs",
	Aliases: []string{"worklog", "wl"},
	Short:   "Manage issue worklogs (time tracking)",
}

var worklogsListCmd = &cobra.Command{
	Use:   "list [issue-key]",
	Short: "List worklogs for an issue (auto-detects from git branch if omitted)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey, err := ResolveIssueKey(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		result, err := client.ListWorklogs(issueKey, 0, 50)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Worklogs) == 0 {
				fmt.Printf("No worklogs for %s.\n", issueKey)
				return
			}
			t := NewTable()
			t.Header("ID", "Author", "Time Spent", "Started")
			for _, w := range result.Worklogs {
				author := "unknown"
				if w.Author != nil {
					author = w.Author.DisplayName
				}
				t.Row(w.ID, author, w.TimeSpent, FormatTime(w.Started))
			}
			t.Flush()
			PrintPaginationFooter(result.Total, result.StartAt, len(result.Worklogs), result.Total > result.StartAt+len(result.Worklogs))
		})
	},
}

var worklogsAddCmd = &cobra.Command{
	Use:   "add [issue-key]",
	Short: "Add a worklog entry to an issue",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey, err := ResolveIssueKey(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		timeSpent, _ := cmd.Flags().GetString("time")
		started, _ := cmd.Flags().GetString("started")
		comment, _ := cmd.Flags().GetString("comment")

		if timeSpent == "" {
			fmt.Fprintln(os.Stderr, "Error: --time is required (e.g. '2h', '1d', '30m')")
			os.Exit(1)
		}

		client := getClient()
		req := jira.BuildAddWorklogRequest(timeSpent, started, comment)
		result, err := client.AddWorklog(issueKey, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Worklog added to %s\n", issueKey)
			KV("ID", result.ID)
			KV("Time Spent", result.TimeSpent)
			KV("Started", FormatTime(result.Started))
		})
	},
}

func init() {
	RootCmd.AddCommand(worklogsCmd)
	worklogsCmd.AddCommand(worklogsListCmd)
	worklogsCmd.AddCommand(worklogsAddCmd)

	worklogsAddCmd.Flags().StringP("time", "t", "", "Time spent (required), e.g. '2h', '1d', '30m'")
	worklogsAddCmd.Flags().String("started", "", "Start datetime (ISO 8601), defaults to now")
	worklogsAddCmd.Flags().StringP("comment", "c", "", "Worklog comment")
}
