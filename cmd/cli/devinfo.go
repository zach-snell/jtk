package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/jtk/internal/jira"
)

var devinfoCmd = &cobra.Command{
	Use:   "devinfo [issue-key]",
	Short: "Show development info (branches, PRs, commits) for an issue",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey, err := ResolveIssueKey(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		noBranches, _ := cmd.Flags().GetBool("no-branches")
		noPRs, _ := cmd.Flags().GetBool("no-prs")
		noCommits, _ := cmd.Flags().GetBool("no-commits")
		noBuilds, _ := cmd.Flags().GetBool("no-builds")

		opts := &jira.DevInfoOptions{
			IncludeBranches: !noBranches,
			IncludePRs:      !noPRs,
			IncludeCommits:  !noCommits,
			IncludeBuilds:   !noBuilds,
		}

		client := getClient()
		result, err := client.GetDevelopmentInfo(issueKey, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Development info for %s\n\n", result.IssueKey)

			// Branches
			if opts.IncludeBranches {
				fmt.Printf("Branches (%d)\n", len(result.Branches))
				if len(result.Branches) == 0 {
					fmt.Println("  (none)")
				} else {
					t := NewTable()
					t.Header("Name", "Repository", "URL")
					for _, b := range result.Branches {
						repo := ""
						if b.Repository != nil {
							repo = b.Repository.Name
						}
						t.Row(b.Name, repo, b.URL)
					}
					t.Flush()
				}
				fmt.Println()
			}

			// Pull Requests
			if opts.IncludePRs {
				fmt.Printf("Pull Requests (%d)\n", len(result.PullRequests))
				if len(result.PullRequests) == 0 {
					fmt.Println("  (none)")
				} else {
					t := NewTable()
					t.Header("ID", "Name", "Status", "Author", "URL")
					for _, pr := range result.PullRequests {
						author := ""
						if pr.Author != nil {
							author = pr.Author.Name
						}
						t.Row(pr.ID, Truncate(pr.Name, 50), pr.Status, author, pr.URL)
					}
					t.Flush()
				}
				fmt.Println()
			}

			// Commits
			if opts.IncludeCommits {
				totalCommits := 0
				for _, repo := range result.Repositories {
					totalCommits += len(repo.Commits)
				}
				fmt.Printf("Commits (%d across %d repositories)\n", totalCommits, len(result.Repositories))
				if len(result.Repositories) == 0 {
					fmt.Println("  (none)")
				} else {
					for _, repo := range result.Repositories {
						fmt.Printf("  %s\n", repo.Name)
						t := NewTable()
						t.Header("ID", "Author", "Message")
						for _, c := range repo.Commits {
							author := ""
							if c.Author != nil {
								author = c.Author.Name
							}
							t.Row(c.DisplayID, author, Truncate(c.Message, 60))
						}
						t.Flush()
					}
				}
				fmt.Println()
			}

			// Builds
			if opts.IncludeBuilds {
				fmt.Printf("Builds (%d)\n", len(result.Builds))
				if len(result.Builds) == 0 {
					fmt.Println("  (none)")
				} else {
					t := NewTable()
					t.Header("Name", "State", "URL")
					for _, b := range result.Builds {
						t.Row(b.Name, b.State, b.URL)
					}
					t.Flush()
				}
			}
		})
	},
}

func init() {
	RootCmd.AddCommand(devinfoCmd)

	devinfoCmd.Flags().Bool("no-branches", false, "Exclude branches from output")
	devinfoCmd.Flags().Bool("no-prs", false, "Exclude pull requests from output")
	devinfoCmd.Flags().Bool("no-commits", false, "Exclude commits from output")
	devinfoCmd.Flags().Bool("no-builds", false, "Exclude builds from output")
}
