package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zach-snell/jtk/internal/jira"
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

var projectsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Jira project",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()

		key, _ := cmd.Flags().GetString("key")
		name, _ := cmd.Flags().GetString("name")
		leadID, _ := cmd.Flags().GetString("lead")
		projType, _ := cmd.Flags().GetString("type")
		template, _ := cmd.Flags().GetString("template")
		description, _ := cmd.Flags().GetString("description")

		if key == "" || name == "" {
			fmt.Fprintf(os.Stderr, "Error: --key and --name are required\n")
			os.Exit(1)
		}

		// If no lead specified, use current user
		if leadID == "" {
			me, err := client.GetCurrentUser()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: --lead is required (failed to auto-detect: %v)\n", err)
				os.Exit(1)
			}
			leadID = me.AccountID
		}

		if projType == "" {
			projType = "software"
		}

		proj, err := client.CreateProject(jira.CreateProjectRequest{
			Key:                key,
			Name:               name,
			ProjectTypeKey:     projType,
			ProjectTemplateKey: template,
			Description:        description,
			LeadAccountID:      leadID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, proj, func() {
			fmt.Printf("Created project %s: %s\n", proj.Key, proj.Name)
			KV("ID", proj.ID)
			KV("Type", projType)
		})
	},
}

func init() {
	RootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsGetCmd)
	projectsCmd.AddCommand(projectsCreateCmd)

	projectsCreateCmd.Flags().String("key", "", "Project key (e.g. PROJ)")
	projectsCreateCmd.Flags().String("name", "", "Project name")
	projectsCreateCmd.Flags().String("lead", "", "Lead account ID (defaults to current user)")
	projectsCreateCmd.Flags().String("type", "software", "Project type: software, business, service_desk")
	projectsCreateCmd.Flags().String("template", "", "Project template key (e.g. com.pyxis.greenhopper.jira:gh-simplified-agility-scrum)")
	projectsCreateCmd.Flags().String("description", "", "Project description")
}
