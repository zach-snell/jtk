package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// bulkValueOps require a positional value argument (e.g. the target epic key).
var bulkValueOps = map[string]bool{
	"set-parent": true, "transition": true, "add-label": true, "remove-label": true,
}

// bulkNoValueOps take no value.
var bulkNoValueOps = map[string]bool{
	"archive": true, "unarchive": true, "delete": true,
}

var bulkCmd = &cobra.Command{
	Use:   "bulk --jql <jql> <operation> [value]",
	Short: "Apply one operation to every issue matching a JQL query",
	Long: `Enumerate issues with JQL and apply a single operation to all of them.

Operations:
  set-parent <epic-key>    re-parent each issue (verified by readback)
  transition <name>        transition each issue to a status
  add-label <label>        add a label (does not clobber existing labels)
  remove-label <label>     remove a label
  archive                  archive the issues (bulk, reversible)
  unarchive                unarchive the issues
  delete                   delete the issues (destructive)

A preview is printed by default; pass --execute to actually apply the change.
The HTTP client paces itself and retries on 429, so large batches run unattended.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jql, _ := cmd.Flags().GetString("jql")
		execute, _ := cmd.Flags().GetBool("execute")
		maxN, _ := cmd.Flags().GetInt("max")

		if jql == "" {
			fmt.Fprintln(os.Stderr, "Error: --jql is required")
			os.Exit(1)
		}

		op := args[0]
		value := ""
		if len(args) > 1 {
			value = args[1]
		}
		switch {
		case bulkValueOps[op] && value == "":
			fmt.Fprintf(os.Stderr, "Error: operation %q requires a value (e.g. 'bulk --jql ... %s <value>')\n", op, op)
			os.Exit(1)
		case !bulkValueOps[op] && !bulkNoValueOps[op]:
			fmt.Fprintf(os.Stderr, "Error: unknown operation %q\n", op)
			os.Exit(1)
		}

		client := getClient()
		issues, truncated, err := client.SearchAllJQL(jql, maxN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		keys := make([]string, len(issues))
		for i, iss := range issues {
			keys[i] = iss.Key
		}

		// Preview.
		action := op
		if value != "" {
			action = op + " " + value
		}
		fmt.Printf("JQL matched %d issue(s). Operation: %s\n", len(keys), action)
		for _, iss := range issues {
			fmt.Printf("  %s  %s\n", iss.Key, truncate(iss.Fields.Summary, 60))
		}
		if len(keys) == 0 {
			return
		}
		// Never silently apply to a partial set: if more issues match than the
		// --max cap, refuse to execute so nothing is quietly left untouched.
		if truncated {
			fmt.Fprintf(os.Stderr, "\nWARNING: more than %d issues match — only the first %d were fetched.\n", maxN, maxN)
			if execute {
				fmt.Fprintln(os.Stderr, "Refusing to apply to a truncated set. Raise --max or narrow the JQL.")
				os.Exit(1)
			}
		}
		if !execute {
			fmt.Println("\nDry run. Re-run with --execute to apply.")
			return
		}

		// Bulk archive/unarchive go through one API call.
		switch op {
		case "archive":
			r, err := client.ArchiveIssues(keys)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("\nDone: archived %d issue(s)\n", r.NumberOfIssuesUpdated)
			return
		case "unarchive":
			r, err := client.UnarchiveIssues(keys)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("\nDone: unarchived %d issue(s)\n", r.NumberOfIssuesUpdated)
			return
		}

		// Per-issue operations with an ok/fail tally.
		ok, fail := 0, 0
		var failures []string
		for _, k := range keys {
			var e error
			switch op {
			case "set-parent":
				e = client.ReparentIssue(k, value)
			case "transition":
				e = client.TransitionIssue(k, value)
			case "add-label":
				e = client.ModifyLabels(k, []string{value}, nil)
			case "remove-label":
				e = client.ModifyLabels(k, nil, []string{value})
			case "delete":
				e = client.DeleteIssue(k)
			}
			if e != nil {
				fail++
				failures = append(failures, fmt.Sprintf("  %s: %v", k, e))
				fmt.Printf("  FAIL %s\n", k)
			} else {
				ok++
				fmt.Printf("  ok   %s\n", k)
			}
		}
		fmt.Printf("\nDone: %d ok, %d failed\n", ok, fail)
		for _, f := range failures {
			fmt.Fprintln(os.Stderr, f)
		}
		if fail > 0 {
			os.Exit(1)
		}
	},
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func init() {
	RootCmd.AddCommand(bulkCmd)
	bulkCmd.Flags().String("jql", "", "JQL query selecting the issues to operate on (required)")
	bulkCmd.Flags().Bool("execute", false, "Apply the change (default is a dry-run preview)")
	bulkCmd.Flags().Int("max", 1000, "Safety cap on issues to operate on; bulk refuses to run if more match")
}
