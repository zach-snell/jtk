package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"project", "p"},
	Short:   "Manage Jira projects",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListProjects(0, 50)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No projects found.")
				return
			}
			t := NewTable()
			t.Header("Key", "Name", "Type", "Lead")
			for _, proj := range result.Values {
				lead := "-"
				if proj.Lead != nil {
					lead = proj.Lead.DisplayName
				}
				t.Row(
					proj.Key,
					proj.Name,
					proj.ProjectType,
					lead,
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Total, result.StartAt, len(result.Values), !result.IsLast)
		})
	},
}

var projectsGetCmd = &cobra.Command{
	Use:   "get <project-key>",
	Short: "Get project details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		proj, err := client.GetProject(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, proj, func() {
			fmt.Printf("%s: %s\n", proj.Key, proj.Name)
			KV("Type", proj.ProjectType)
			if proj.Lead != nil {
				KV("Lead", proj.Lead.DisplayName)
			}
			if proj.Description != "" {
				KV("Description", Truncate(proj.Description, 80))
			}
			if len(proj.IssueTypes) > 0 {
				var types []string
				for _, it := range proj.IssueTypes {
					types = append(types, it.Name)
				}
				KV("Issue Types", strings.Join(types, ", "))
			}
		})
	},
}

func init() {
	RootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsGetCmd)
}
