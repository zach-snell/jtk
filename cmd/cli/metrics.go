package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics [issue-key]",
	Short: "Show issue lifecycle metrics (cycle time, lead time, status breakdown)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey, err := ResolveIssueKey(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		raw, _ := cmd.Flags().GetBool("dates")

		client := getClient()

		if raw {
			dates, err := client.GetIssueDates(issueKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			PrintOrJSON(cmd, dates, func() {
				fmt.Printf("Issue dates for %s\n\n", dates.IssueKey)

				KV("Created", FormatTime(dates.Created))
				KV("Updated", FormatTime(dates.Updated))
				if dates.DueDate != "" {
					KV("Due Date", dates.DueDate)
				}
				if dates.ResolutionDate != "" {
					KV("Resolved", FormatTime(dates.ResolutionDate))
				}
				KV("Current Status", dates.CurrentStatus)

				if len(dates.StatusTransitions) > 0 {
					fmt.Printf("\nStatus Transitions (%d)\n", len(dates.StatusTransitions))
					t := NewTable()
					t.Header("From", "To", "At", "By", "Duration")
					for _, tr := range dates.StatusTransitions {
						dur := "-"
						if tr.DurationDisplay != "" {
							dur = tr.DurationDisplay
						}
						t.Row(tr.FromStatus, tr.ToStatus, FormatTime(tr.TransitionedAt), tr.TransitionedBy, dur)
					}
					t.Flush()
				}

				if len(dates.StatusSummary) > 0 {
					fmt.Printf("\nTime in Status\n")
					t := NewTable()
					t.Header("Status", "Total Time", "Visits")
					for _, s := range dates.StatusSummary {
						t.Row(s.Status, s.TotalDurationDisplay, fmt.Sprintf("%d", s.VisitCount))
					}
					t.Flush()
				}
			})
			return
		}

		metrics, err := client.GetIssueMetrics(issueKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, metrics, func() {
			fmt.Printf("Metrics for %s\n\n", metrics.IssueKey)

			KV("Current Status", metrics.CurrentStatus)

			if metrics.LeadTimeDisplay != "" {
				KV("Lead Time", metrics.LeadTimeDisplay)
			} else {
				KV("Lead Time", "-")
			}

			if metrics.CycleTimeDisplay != "" {
				KV("Cycle Time", metrics.CycleTimeDisplay)
			} else {
				KV("Cycle Time", "-")
			}

			if metrics.TimeInCurrentDisplay != "" {
				KV("In Current", metrics.TimeInCurrentDisplay)
			} else {
				KV("In Current", "-")
			}

			if len(metrics.StatusSummary) > 0 {
				fmt.Printf("\nTime in Status\n")
				t := NewTable()
				t.Header("Status", "Total Time", "Visits")
				for _, s := range metrics.StatusSummary {
					t.Row(s.Status, s.TotalDurationDisplay, fmt.Sprintf("%d", s.VisitCount))
				}
				t.Flush()
			}
		})
	},
}

func init() {
	RootCmd.AddCommand(metricsCmd)

	metricsCmd.Flags().Bool("dates", false, "Show raw date info and transition history instead of computed metrics")
}
