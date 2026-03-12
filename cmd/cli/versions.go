package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var versionsCmd = &cobra.Command{
	Use:     "versions",
	Aliases: []string{"version", "v"},
	Short:   "Manage project versions (releases)",
}

var versionsListCmd = &cobra.Command{
	Use:   "list <project-key>",
	Short: "List all versions for a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		result, err := client.ListVersions(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result) == 0 {
				fmt.Printf("No versions found for %s.\n", args[0])
				return
			}
			t := NewTable()
			t.Header("ID", "Name", "Released", "Archived", "Release Date")
			for _, v := range result {
				t.Row(
					v.ID,
					v.Name,
					FormatBool(v.Released),
					FormatBool(v.Archived),
					v.ReleaseDate,
				)
			}
			t.Flush()
		})
	},
}

var versionsGetCmd = &cobra.Command{
	Use:   "get <version-id>",
	Short: "Get version details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		v, err := client.GetVersion(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, v, func() {
			fmt.Printf("Version: %s\n", v.Name)
			KV("ID", v.ID)
			if v.Description != "" {
				KV("Description", v.Description)
			}
			KV("Released", FormatBool(v.Released))
			KV("Archived", FormatBool(v.Archived))
			if v.StartDate != "" {
				KV("Start Date", v.StartDate)
			}
			if v.ReleaseDate != "" {
				KV("Release Date", v.ReleaseDate)
			}
		})
	},
}

func init() {
	RootCmd.AddCommand(versionsCmd)
	versionsCmd.AddCommand(versionsListCmd)
	versionsCmd.AddCommand(versionsGetCmd)
}
