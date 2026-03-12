package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
)

// ListAttachments returns the attachments on an issue by fetching the issue's attachment field.
func (c *Client) ListAttachments(issueKey string) ([]Attachment, error) {
	path := fmt.Sprintf("/issue/%s?fields=attachment", url.PathEscape(issueKey))
	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Fields struct {
			Attachment []Attachment `json:"attachment"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshaling attachments: %w", err)
	}

	return raw.Fields.Attachment, nil
}

// GetAttachment returns metadata for a single attachment by ID.
func (c *Client) GetAttachment(attachmentID string) (*Attachment, error) {
	path := fmt.Sprintf("/attachment/%s", url.PathEscape(attachmentID))
	return GetJSON[Attachment](c, path)
}

// DownloadAttachment downloads an attachment's content from its absolute content URL.
// Returns the raw bytes, the filename, and any error.
func (c *Client) DownloadAttachment(contentURL string) (data []byte, contentType string, err error) {
	// First get the attachment metadata to know the filename
	resp, err := c.GetAbsolute(contentURL)
	if err != nil {
		return nil, "", fmt.Errorf("downloading attachment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("download failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading attachment body: %w", err)
	}

	// Extract filename from Content-Disposition header or URL
	filename := ""
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		// Try to parse filename from Content-Disposition
		for _, part := range bytes.Split([]byte(cd), []byte(";")) {
			trimmed := bytes.TrimSpace(part)
			if bytes.HasPrefix(trimmed, []byte("filename=")) {
				filename = string(bytes.Trim(trimmed[9:], "\" "))
			}
		}
	}

	return data, filename, nil
}

// DeleteAttachment deletes an attachment by its ID.
func (c *Client) DeleteAttachment(attachmentID string) error {
	path := fmt.Sprintf("/attachment/%s", url.PathEscape(attachmentID))
	return c.Delete(path)
}

// UploadAttachment uploads a file as an attachment to an issue.
// Uses multipart/form-data with the X-Atlassian-Token: no-check header.
func (c *Client) UploadAttachment(issueKey, filename string, content []byte) ([]Attachment, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}

	if _, err := part.Write(content); err != nil {
		return nil, fmt.Errorf("writing file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	path := fmt.Sprintf("/issue/%s/attachments", url.PathEscape(issueKey))
	respData, err := c.PostMultipart(path, buf.Bytes(), writer.FormDataContentType())
	if err != nil {
		return nil, err
	}

	var result []Attachment
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling attachment response: %w", err)
	}

	return result, nil
}
