package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var boardsCmd = &cobra.Command{
	Use:     "boards",
	Aliases: []string{"board", "b"},
	Short:   "Manage Jira agile boards",
}

var boardsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agile boards",
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")

		client := getClient()
		result, err := client.ListBoards(project, 0, 50)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No boards found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Name", "Type", "Project")
			for _, board := range result.Values {
				projectKey := ""
				if board.Location != nil {
					projectKey = board.Location.ProjectKey
				}
				t.Row(
					fmt.Sprintf("%d", board.ID),
					board.Name,
					board.Type,
					projectKey,
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Total, result.StartAt, len(result.Values), !result.IsLast)
		})
	},
}

var boardsSprintsCmd = &cobra.Command{
	Use:   "sprints <board-id>",
	Short: "List sprints for a board",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		boardID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid board ID: %s\n", args[0])
			os.Exit(1)
		}

		state, _ := cmd.Flags().GetString("state")

		client := getClient()
		result, err := client.ListSprints(boardID, state, 0, 50)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No sprints found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Name", "State", "Start", "End", "Goal")
			for _, sprint := range result.Values {
				t.Row(
					fmt.Sprintf("%d", sprint.ID),
					sprint.Name,
					sprint.State,
					FormatTime(sprint.StartDate),
					FormatTime(sprint.EndDate),
					Truncate(sprint.Goal, 40),
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Total, result.StartAt, len(result.Values), !result.IsLast)
		})
	},
}

func init() {
	RootCmd.AddCommand(boardsCmd)
	boardsCmd.AddCommand(boardsListCmd)
	boardsCmd.AddCommand(boardsSprintsCmd)

	boardsListCmd.Flags().String("project", "", "Filter boards by project key")
	boardsSprintsCmd.Flags().String("state", "", "Filter by sprint state: active, future, closed")
}
