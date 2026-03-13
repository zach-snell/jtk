package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var attachmentsCmd = &cobra.Command{
	Use:     "attachments",
	Aliases: []string{"attachment", "att"},
	Short:   "Manage issue attachments",
}

var attachmentsListCmd = &cobra.Command{
	Use:   "list [issue-key]",
	Short: "List attachments on an issue (auto-detects from git branch if omitted)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueKey, err := ResolveIssueKey(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		attachments, err := client.ListAttachments(issueKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, attachments, func() {
			if len(attachments) == 0 {
				fmt.Printf("No attachments on %s.\n", issueKey)
				return
			}
			t := NewTable()
			t.Header("ID", "Filename", "Size", "MIME Type", "Author", "Created")
			for _, a := range attachments {
				author := ""
				if a.Author != nil {
					author = a.Author.DisplayName
				}
				t.Row(
					a.ID,
					a.Filename,
					formatFileSize(a.Size),
					a.MimeType,
					author,
					FormatTime(a.Created),
				)
			}
			t.Flush()
		})
	},
}

var attachmentsDownloadCmd = &cobra.Command{
	Use:   "download <attachment-id>",
	Short: "Download an attachment by ID",
	Long: `Download an attachment by its numeric ID.
Use 'jtk attachments list <issue-key>' to find attachment IDs.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		attachmentID := args[0]
		outPath, _ := cmd.Flags().GetString("output")

		client := getClient()

		// First get metadata to find the content URL
		att, err := client.GetAttachment(attachmentID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting attachment metadata: %v\n", err)
			os.Exit(1)
		}

		data, filename, err := client.DownloadAttachment(att.Content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading attachment: %v\n", err)
			os.Exit(1)
		}

		// Determine output filename
		dest := outPath
		if dest == "" {
			if filename != "" {
				dest = filename
			} else {
				dest = att.Filename
			}
		}

		if err := os.WriteFile(dest, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Downloaded %s (%s) to %s\n", att.Filename, formatFileSize(att.Size), dest)
	},
}

// formatFileSize returns a human-readable file size string.
func formatFileSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func init() {
	RootCmd.AddCommand(attachmentsCmd)
	attachmentsCmd.AddCommand(attachmentsListCmd)
	attachmentsCmd.AddCommand(attachmentsDownloadCmd)

	attachmentsDownloadCmd.Flags().StringP("output", "o", "", "Output file path (defaults to original filename)")
}
