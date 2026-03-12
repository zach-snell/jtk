package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// outputJSON returns true if the user passed --json.
func outputJSON(cmd *cobra.Command) bool {
	j, _ := cmd.Root().PersistentFlags().GetBool("json")
	return j
}

// PrintJSON formats any Go struct as pretty JSON and prints to stdout.
func PrintJSON(data any) {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

// PrintOrJSON prints formatted output or JSON depending on the --json flag.
// The formatter func should print the human-readable output.
func PrintOrJSON(cmd *cobra.Command, data any, formatter func()) {
	if outputJSON(cmd) {
		PrintJSON(data)
		return
	}
	formatter()
}

// --- Table helpers ---

// Table is a simple tabwriter-based table printer.
type Table struct {
	w *tabwriter.Writer
}

// NewTable creates a new table with tabwriter defaults.
func NewTable() *Table {
	return &Table{
		w: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

// Header writes a header row (uppercased automatically).
func (t *Table) Header(cols ...string) {
	upper := make([]string, len(cols))
	for i, c := range cols {
		upper[i] = strings.ToUpper(c)
	}
	fmt.Fprintln(t.w, strings.Join(upper, "\t"))
}

// Row writes a data row.
func (t *Table) Row(vals ...string) {
	fmt.Fprintln(t.w, strings.Join(vals, "\t"))
}

// Flush flushes the underlying tabwriter.
func (t *Table) Flush() {
	t.w.Flush()
}

// --- Key-value helpers (single-item display like `status`) ---

// KV prints a labeled key-value pair with consistent padding.
func KV(label, value string) {
	fmt.Printf("  %-16s%s\n", label+":", value)
}

// KVf prints a formatted key-value pair.
func KVf(label, format string, args ...any) {
	KV(label, fmt.Sprintf(format, args...))
}

// --- Common formatters ---

// FormatTime formats a time string as a short human-readable string.
func FormatTime(t string) string {
	if t == "" {
		return "-"
	}
	parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", t)
	if err != nil {
		// Try alternate format
		parsed, err = time.Parse(time.RFC3339, t)
		if err != nil {
			return t // return raw if unparseable
		}
	}
	return parsed.Format("2006-01-02 15:04")
}

// FormatBool returns a readable yes/no.
func FormatBool(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// Truncate truncates a string to maxLen and adds "..." if needed.
func Truncate(s string, maxLen int) string {
	// Replace newlines with spaces for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// PrintPaginationFooter prints a summary line showing current page info.
func PrintPaginationFooter(total, startAt, count int, hasMore bool) {
	if total > 0 {
		fmt.Printf("\nShowing %d items (starting at %d, %d total)", count, startAt, total)
		if hasMore {
			fmt.Print(" — more results available")
		}
		fmt.Println()
	}
}
