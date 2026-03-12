package jira

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FlattenedIssue is a token-efficient representation of a Jira issue.
type FlattenedIssue struct {
	Key         string   `json:"key"`
	Summary     string   `json:"summary"`
	Status      string   `json:"status"`
	Type        string   `json:"type"`
	Priority    string   `json:"priority"`
	Assignee    string   `json:"assignee"`
	Reporter    string   `json:"reporter"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	Components  []string `json:"components"`
	Sprint      string   `json:"sprint,omitempty"`
	StoryPoints float64  `json:"story_points,omitempty"`
	ParentKey   string   `json:"parent_key,omitempty"`
}

// FlattenIssue converts a raw Jira issue map into a FlattenedIssue.
func FlattenIssue(raw map[string]interface{}) *FlattenedIssue {
	fields, _ := raw["fields"].(map[string]interface{})
	if fields == nil {
		return &FlattenedIssue{
			Key: getStr(raw, "key"),
		}
	}

	fi := &FlattenedIssue{
		Key:         getStr(raw, "key"),
		Summary:     getStr(fields, "summary"),
		Status:      getNestedName(fields, "status"),
		Type:        getNestedName(fields, "issuetype"),
		Priority:    getNestedName(fields, "priority"),
		Assignee:    getNestedDisplayName(fields, "assignee"),
		Reporter:    getNestedDisplayName(fields, "reporter"),
		Created:     getStr(fields, "created"),
		Updated:     getStr(fields, "updated"),
		Description: ADFToPlainText(fields["description"]),
	}

	// Labels
	if labels, ok := fields["labels"].([]interface{}); ok {
		for _, l := range labels {
			if s, ok := l.(string); ok {
				fi.Labels = append(fi.Labels, s)
			}
		}
	}

	// Components
	if components, ok := fields["components"].([]interface{}); ok {
		for _, comp := range components {
			if m, ok := comp.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					fi.Components = append(fi.Components, name)
				}
			}
		}
	}

	// Sprint
	if sprint, ok := fields["sprint"].(map[string]interface{}); ok {
		fi.Sprint = getStr(sprint, "name")
	}

	// Story points - check common custom field
	if sp, ok := fields["story_points"].(float64); ok {
		fi.StoryPoints = sp
	} else if sp, ok := fields["customfield_10016"].(float64); ok {
		fi.StoryPoints = sp
	}

	// Parent
	if parent, ok := fields["parent"].(map[string]interface{}); ok {
		fi.ParentKey = getStr(parent, "key")
	}

	return fi
}

// FlattenIssueFromTyped converts a typed Issue to a FlattenedIssue.
func FlattenIssueFromTyped(issue *Issue) *FlattenedIssue {
	fi := &FlattenedIssue{
		Key:     issue.Key,
		Summary: issue.Fields.Summary,
		Created: issue.Fields.Created,
		Updated: issue.Fields.Updated,
	}

	if issue.Fields.Status != nil {
		fi.Status = issue.Fields.Status.Name
	}
	if issue.Fields.IssueType != nil {
		fi.Type = issue.Fields.IssueType.Name
	}
	if issue.Fields.Priority != nil {
		fi.Priority = issue.Fields.Priority.Name
	}
	if issue.Fields.Assignee != nil {
		fi.Assignee = issue.Fields.Assignee.DisplayName
	}
	if issue.Fields.Reporter != nil {
		fi.Reporter = issue.Fields.Reporter.DisplayName
	}
	if issue.Fields.Description != nil {
		fi.Description = ADFToPlainText(issue.Fields.Description)
	}

	fi.Labels = issue.Fields.Labels

	for _, comp := range issue.Fields.Components {
		fi.Components = append(fi.Components, comp.Name)
	}

	if issue.Fields.Sprint != nil {
		fi.Sprint = issue.Fields.Sprint.Name
	}
	if issue.Fields.StoryPoints != nil {
		fi.StoryPoints = *issue.Fields.StoryPoints
	}
	if issue.Fields.Parent != nil {
		fi.ParentKey = issue.Fields.Parent.Key
	}

	return fi
}

// StripJunkFields recursively removes token-heavy fields from JSON data.
func StripJunkFields(data map[string]interface{}) {
	junkKeys := []string{"self", "expand", "iconUrl", "avatarUrls", "_links", "icons"}
	for _, key := range junkKeys {
		delete(data, key)
	}

	for _, v := range data {
		switch val := v.(type) {
		case map[string]interface{}:
			StripJunkFields(val)
		case []interface{}:
			for _, item := range val {
				if m, ok := item.(map[string]interface{}); ok {
					StripJunkFields(m)
				}
			}
		}
	}
}

// ADFToPlainText converts an Atlassian Document Format (ADF) structure to markdown text.
// Handles text marks (bold, italic, code, strikethrough, underline), headings,
// bullet/ordered lists, code blocks, tables, mentions, emoji, inline cards,
// media, blockquotes, and horizontal rules.
func ADFToPlainText(adf interface{}) string {
	if adf == nil {
		return ""
	}

	switch v := adf.(type) {
	case string:
		return v
	case map[string]interface{}:
		var sb strings.Builder
		renderADFNode(v, &sb, 0, "")
		return strings.TrimSpace(sb.String())
	default:
		return ""
	}
}

// renderADFChildren renders all child nodes in a content array.
func renderADFChildren(content []interface{}, sb *strings.Builder, depth int, listPrefix string) {
	for _, child := range content {
		if childNode, ok := child.(map[string]interface{}); ok {
			renderADFNode(childNode, sb, depth, listPrefix)
		}
	}
}

// renderText renders a text node with optional marks (bold, italic, code, etc.).
func renderText(node map[string]interface{}, sb *strings.Builder) {
	text, _ := node["text"].(string)
	if marks, ok := node["marks"].([]interface{}); ok && len(marks) > 0 {
		text = applyMarks(text, marks)
	}
	sb.WriteString(text)
}

// applyMarks wraps text with markdown formatting based on ADF marks.
func applyMarks(text string, marks []interface{}) string {
	for _, mark := range marks {
		if m, ok := mark.(map[string]interface{}); ok {
			markType, _ := m["type"].(string)
			switch markType {
			case "strong":
				text = "**" + text + "**"
			case "em":
				text = "*" + text + "*"
			case "code":
				text = "`" + text + "`"
			case "strike":
				text = "~~" + text + "~~"
			case "underline":
				text = "__" + text + "__"
			}
		}
	}
	return text
}

// renderHeading renders a heading node with the appropriate markdown level.
func renderHeading(attrs map[string]interface{}, content []interface{}, sb *strings.Builder, depth int, listPrefix string) {
	level := 1
	if attrs != nil {
		if lvl, ok := attrs["level"].(float64); ok {
			level = int(lvl)
		}
	}
	sb.WriteString(strings.Repeat("#", level) + " ")
	renderADFChildren(content, sb, depth, listPrefix)
	sb.WriteString("\n\n")
}

// renderBulletList renders a bullet list node.
func renderBulletList(content []interface{}, sb *strings.Builder, depth int) {
	for _, child := range content {
		if childNode, ok := child.(map[string]interface{}); ok {
			renderADFNode(childNode, sb, depth, "- ")
		}
	}
}

// renderOrderedList renders an ordered list node.
func renderOrderedList(content []interface{}, sb *strings.Builder, depth int) {
	for i, child := range content {
		if childNode, ok := child.(map[string]interface{}); ok {
			prefix := fmt.Sprintf("%d. ", i+1)
			renderADFNode(childNode, sb, depth, prefix)
		}
	}
}

// renderListItem renders a list item node with proper indentation.
func renderListItem(content []interface{}, sb *strings.Builder, depth int, listPrefix string) {
	if listPrefix != "" {
		sb.WriteString(strings.Repeat("  ", depth))
		sb.WriteString(listPrefix)
	}
	for _, child := range content {
		if childNode, ok := child.(map[string]interface{}); ok {
			renderADFNode(childNode, sb, depth+1, "")
		}
	}
}

// renderCodeBlock renders a fenced code block with optional language.
func renderCodeBlock(attrs map[string]interface{}, content []interface{}, sb *strings.Builder, depth int, listPrefix string) {
	language := ""
	if attrs != nil {
		if lang, ok := attrs["language"].(string); ok {
			language = lang
		}
	}
	sb.WriteString("```" + language + "\n")
	renderADFChildren(content, sb, depth, listPrefix)
	sb.WriteString("```\n\n")
}

// renderBlockquote renders a blockquote by prefixing each line with "> ".
func renderBlockquote(content []interface{}, sb *strings.Builder, depth int, listPrefix string) {
	var innerSb strings.Builder
	renderADFChildren(content, &innerSb, depth, listPrefix)
	lines := strings.Split(strings.TrimSpace(innerSb.String()), "\n")
	for _, line := range lines {
		sb.WriteString("> " + line + "\n")
	}
	sb.WriteString("\n")
}

// renderTableRow renders a markdown table row.
func renderTableRow(content []interface{}, sb *strings.Builder, depth int, listPrefix string) {
	sb.WriteString("| ")
	for _, child := range content {
		if childNode, ok := child.(map[string]interface{}); ok {
			renderADFNode(childNode, sb, depth, listPrefix)
			sb.WriteString(" | ")
		}
	}
	sb.WriteString("\n")
}

// renderMedia renders a media node with descriptive bracketed text.
func renderMedia(attrs map[string]interface{}, sb *strings.Builder) {
	if attrs == nil {
		sb.WriteString("[Media/Image]")
		return
	}
	mediaID, _ := attrs["id"].(string)
	mediaType, _ := attrs["type"].(string)
	alt, _ := attrs["alt"].(string)

	switch {
	case alt != "":
		fmt.Fprintf(sb, "[Media: %s", alt)
	case mediaType != "":
		fmt.Fprintf(sb, "[Media: %s", mediaType)
	default:
		sb.WriteString("[Media")
	}

	if w, ok := attrs["width"].(float64); ok {
		if h, ok := attrs["height"].(float64); ok {
			fmt.Fprintf(sb, " (%dx%d)", int(w), int(h))
		}
	}

	if mediaID != "" {
		fmt.Fprintf(sb, " | id=%s", mediaID)
	}
	sb.WriteString("]")
}

// renderInlineNode renders mention, emoji, and inlineCard nodes.
func renderInlineNode(nodeType string, attrs map[string]interface{}, sb *strings.Builder) {
	if attrs == nil {
		return
	}
	switch nodeType {
	case "mention":
		if text, ok := attrs["text"].(string); ok {
			sb.WriteString("@" + text)
		}
	case "emoji":
		if shortName, ok := attrs["shortName"].(string); ok {
			sb.WriteString(shortName)
		}
	case "inlineCard":
		if url, ok := attrs["url"].(string); ok {
			sb.WriteString(url)
		}
	}
}

// renderADFNode recursively renders an ADF node to a string builder.
func renderADFNode(node map[string]interface{}, sb *strings.Builder, depth int, listPrefix string) {
	if node == nil {
		return
	}

	nodeType, _ := node["type"].(string)
	attrs, _ := node["attrs"].(map[string]interface{})
	content, _ := node["content"].([]interface{})

	switch nodeType {
	case "doc":
		renderADFChildren(content, sb, depth, listPrefix)
	case "paragraph":
		renderADFChildren(content, sb, depth, listPrefix)
		sb.WriteString("\n\n")
	case "text":
		renderText(node, sb)
	case "hardBreak":
		sb.WriteString("\n")
	case "heading":
		renderHeading(attrs, content, sb, depth, listPrefix)
	case "bulletList":
		renderBulletList(content, sb, depth)
	case "orderedList":
		renderOrderedList(content, sb, depth)
	case "listItem":
		renderListItem(content, sb, depth, listPrefix)
	case "codeBlock":
		renderCodeBlock(attrs, content, sb, depth, listPrefix)
	case "blockquote":
		renderBlockquote(content, sb, depth, listPrefix)
	case "rule":
		sb.WriteString("---\n\n")
	case "table":
		renderADFChildren(content, sb, depth, listPrefix)
		sb.WriteString("\n")
	case "tableRow":
		renderTableRow(content, sb, depth, listPrefix)
	case "tableHeader", "tableCell":
		renderADFChildren(content, sb, depth, listPrefix)
	case "mediaSingle", "mediaGroup":
		renderADFChildren(content, sb, depth, listPrefix)
	case "media":
		renderMedia(attrs, sb)
	case "mention", "emoji", "inlineCard":
		renderInlineNode(nodeType, attrs, sb)
	default:
		// Unknown node type — try to render children
		renderADFChildren(content, sb, depth, listPrefix)
	}
}

// SafeJSON marshals data to JSON, dumping to file if oversized.
// Truncates at a newline boundary and caps at maxChars (default 40,000).
// When truncated, appends guidance and saves full response to a temp file.
func SafeJSON(data interface{}, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 40000
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("error marshaling: %v", err)
	}

	if len(out) <= maxChars {
		return string(out)
	}

	// Always dump full response to file
	dir := filepath.Join(os.TempDir(), "jtk-logs")
	_ = os.MkdirAll(dir, 0o755)
	filename := fmt.Sprintf("response-%d.json", time.Now().UnixNano())
	path := filepath.Join(dir, filename)
	_ = os.WriteFile(path, out, 0o600)

	// Truncate at a newline boundary
	truncated := string(out[:maxChars])
	lastNewline := strings.LastIndex(truncated, "\n")
	if lastNewline > 0 {
		truncated = truncated[:lastNewline]
	}

	originalLen := len(out)
	truncatedLen := len(truncated)
	pct := (truncatedLen * 100) / originalLen

	guidance := fmt.Sprintf(`
---
## Response Truncated
This response was truncated to ~%dk chars (%d%% of original %dk chars).

**To get the full data:**
- Full response saved to: %s
- Try narrowing your JQL query or requesting fewer fields
- Use pagination with smaller maxResults`,
		truncatedLen/1000, pct, originalLen/1000, path)

	return truncated + guidance
}

// helpers

func getStr(m map[string]interface{}, key string) string {
	v, ok := m[key].(string)
	if !ok {
		return ""
	}
	return v
}

func getNestedName(m map[string]interface{}, key string) string {
	nested, ok := m[key].(map[string]interface{})
	if !ok {
		return ""
	}
	name, _ := nested["name"].(string)
	return name
}

func getNestedDisplayName(m map[string]interface{}, key string) string {
	nested, ok := m[key].(map[string]interface{})
	if !ok {
		return "unassigned"
	}
	name, _ := nested["displayName"].(string)
	if name == "" {
		return "unassigned"
	}
	return name
}
