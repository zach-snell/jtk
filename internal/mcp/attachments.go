package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageAttachmentsArgs struct {
	Action       string `json:"action" jsonschema:"Action to perform: 'list', 'download'" jsonschema_enum:"list,download"`
	IssueKey     string `json:"issue_key,omitempty" jsonschema:"Jira issue key (for 'list')"`
	AttachmentID string `json:"attachment_id,omitempty" jsonschema:"Attachment ID (for 'download')"`
}

// ManageAttachmentsHandler handles attachment operations.
func ManageAttachmentsHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageAttachmentsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageAttachmentsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			if args.IssueKey == "" {
				return ToolResultError("issue_key is required for 'list' action"), nil, nil
			}
			attachments, err := c.ListAttachments(args.IssueKey)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list attachments: %v", err)), nil, nil
			}
			type flatAttachment struct {
				ID       string `json:"id"`
				Filename string `json:"filename"`
				Author   string `json:"author,omitempty"`
				Created  string `json:"created"`
				Size     int64  `json:"size"`
				MimeType string `json:"mime_type"`
			}
			flat := struct {
				IssueKey    string           `json:"issue_key"`
				Total       int              `json:"total"`
				Attachments []flatAttachment `json:"attachments"`
			}{IssueKey: args.IssueKey, Total: len(attachments)}
			for _, a := range attachments {
				author := ""
				if a.Author != nil {
					author = a.Author.DisplayName
				}
				flat.Attachments = append(flat.Attachments, flatAttachment{
					ID:       a.ID,
					Filename: a.Filename,
					Author:   author,
					Created:  a.Created,
					Size:     a.Size,
					MimeType: a.MimeType,
				})
			}
			return ToolResultText(jira.SafeJSON(flat, 30000)), nil, nil

		case "download":
			if args.AttachmentID == "" {
				return ToolResultError("attachment_id is required for 'download' action"), nil, nil
			}
			// Get attachment metadata to find the content URL and filename
			attachment, err := c.GetAttachment(args.AttachmentID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get attachment metadata: %v", err)), nil, nil
			}

			content, _, err := c.DownloadAttachment(attachment.Content)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to download attachment: %v", err)), nil, nil
			}

			// Save to /tmp/jtk-downloads/
			downloadDir := filepath.Join(os.TempDir(), "jtk-downloads")
			if err := os.MkdirAll(downloadDir, 0o755); err != nil {
				return ToolResultError(fmt.Sprintf("failed to create download directory: %v", err)), nil, nil
			}
			savePath := filepath.Join(downloadDir, attachment.Filename)
			if err := os.WriteFile(savePath, content, 0o644); err != nil {
				return ToolResultError(fmt.Sprintf("failed to save attachment: %v", err)), nil, nil
			}

			return ToolResultText(jira.SafeJSON(map[string]interface{}{
				"attachment_id": attachment.ID,
				"filename":      attachment.Filename,
				"size":          len(content),
				"mime_type":     attachment.MimeType,
				"saved_to":      savePath,
				"status":        "downloaded",
			}, 30000)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid actions: list, download", args.Action)), nil, nil
		}
	}
}
