package jira

import (
	"regexp"
	"strings"
)

// urlRe matches bare http/https URLs in text.
var urlRe = regexp.MustCompile(`https?://[^\s<>"{}|\\` + "`" + `]+`)

// combinedInlineRe is the master pattern used by parseInlineFormatting.
// Named groups keep the match logic readable.
var combinedInlineRe = regexp.MustCompile(
	"`(?P<code>[^`]+)`" +
		`|\*\*(?P<bold>.+?)\*\*` +
		`|~~(?P<strike>.+?)~~` +
		`|\[(?P<linkText>[^\]]+)\]\((?P<linkHref>[^)]+)\)` +
		`|(?:(?:^|[^*]))\*(?P<italic>[^*]+?)\*(?:[^*]|$)`,
)

// Block-level patterns.
var (
	headingRe    = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	unorderedRe  = regexp.MustCompile(`^[-*]\s+`)
	orderedRe    = regexp.MustCompile(`^\d+\.\s+`)
	hrRe         = regexp.MustCompile(`^\s*(?:---+|\*\*\*+|___+)\s*$`)
	tableSepRe   = regexp.MustCompile(`^:?-+:?$`)
	fencedCodeRe = regexp.MustCompile("^```")
)

// MarkdownToADF converts markdown text to a Jira ADF document.
// Returns a map suitable for JSON serialization as an ADF doc node.
func MarkdownToADF(markdown string) map[string]interface{} {
	doc := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []interface{}{},
	}

	if markdown == "" {
		doc["content"] = []interface{}{
			map[string]interface{}{"type": "paragraph", "content": []interface{}{}},
		}
		return doc
	}

	// Normalize literal \n sequences (common from JSON / MCP payloads)
	markdown = strings.ReplaceAll(markdown, `\n`, "\n")

	lines := strings.Split(markdown, "\n")
	content := make([]interface{}, 0, len(lines))
	i := 0

	for i < len(lines) {
		line := lines[i]

		// --- Fenced code block (```lang ... ```) ---
		if fencedCodeRe.MatchString(line) {
			lang := strings.TrimSpace(line[3:])
			var codeLines []string
			i++
			for i < len(lines) && !fencedCodeRe.MatchString(lines[i]) {
				codeLines = append(codeLines, lines[i])
				i++
			}
			if i < len(lines) {
				i++ // skip closing ```
			}
			cb := map[string]interface{}{
				"type":    "codeBlock",
				"content": []interface{}{map[string]interface{}{"type": "text", "text": strings.Join(codeLines, "\n")}},
			}
			if lang != "" {
				cb["attrs"] = map[string]interface{}{"language": lang}
			} else {
				cb["attrs"] = map[string]interface{}{}
			}
			content = append(content, cb)
			continue
		}

		stripped := strings.TrimSpace(line)

		// --- Horizontal rule (---, ***, ___) ---
		if len(stripped) >= 3 && !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") && hrRe.MatchString(line) {
			content = append(content, map[string]interface{}{"type": "rule"})
			i++
			continue
		}

		// --- Heading (# to ######) ---
		if m := headingRe.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			content = append(content, map[string]interface{}{
				"type":    "heading",
				"attrs":   map[string]interface{}{"level": level},
				"content": parseInlineFormatting(m[2]),
			})
			i++
			continue
		}

		// --- Blockquote (> text) ---
		if strings.HasPrefix(line, "> ") {
			var quoteParas []interface{}
			for i < len(lines) && strings.HasPrefix(lines[i], "> ") {
				quoteParas = append(quoteParas, makeParagraph(lines[i][2:]))
				i++
			}
			content = append(content, map[string]interface{}{
				"type":    "blockquote",
				"content": quoteParas,
			})
			continue
		}

		// --- Unordered list (- item or * item) ---
		if unorderedRe.MatchString(line) {
			var items []interface{}
			for i < len(lines) && unorderedRe.MatchString(lines[i]) {
				text := unorderedRe.ReplaceAllString(lines[i], "")
				items = append(items, makeListItem(text))
				i++
			}
			content = append(content, map[string]interface{}{
				"type":    "bulletList",
				"content": items,
			})
			continue
		}

		// --- Ordered list (1. item) ---
		if orderedRe.MatchString(line) {
			var items []interface{}
			for i < len(lines) && orderedRe.MatchString(lines[i]) {
				text := orderedRe.ReplaceAllString(lines[i], "")
				items = append(items, makeListItem(text))
				i++
			}
			content = append(content, map[string]interface{}{
				"type":    "orderedList",
				"content": items,
			})
			continue
		}

		// --- Table (| col | col |) ---
		if strings.HasPrefix(line, "|") && strings.Contains(line[1:], "|") {
			var tableLines []string
			for i < len(lines) && strings.HasPrefix(lines[i], "|") {
				tableLines = append(tableLines, lines[i])
				i++
			}
			tableNode := parseTable(tableLines)
			if tableNode != nil {
				content = append(content, tableNode)
			}
			continue
		}

		// --- Empty line: skip ---
		if stripped == "" {
			i++
			continue
		}

		// --- Default: paragraph with inline formatting ---
		content = append(content, makeParagraph(line))
		i++
	}

	// Ensure at least one content node.
	if len(content) == 0 {
		content = append(content, map[string]interface{}{
			"type":    "paragraph",
			"content": []interface{}{},
		})
	}

	doc["content"] = content
	return doc
}

// parseInlineFormatting converts inline markdown (bold, italic, code, links,
// strikethrough, bare URLs) into ADF inline nodes.
func parseInlineFormatting(text string) []interface{} {
	if text == "" {
		return []interface{}{}
	}

	// We process in two passes:
	//  1. Match explicit markdown syntax (bold, code, strike, links, italic).
	//  2. Within remaining plain-text segments, auto-link bare URLs.

	type segment struct {
		start, end int
		node       map[string]interface{}
	}

	var segs []segment

	// Pass 1: explicit markdown patterns.
	// We use FindAllStringSubmatchIndex for positional info.
	// Pattern groups (by index pair):
	//   [0,1]   = full match
	//   [2,3]   = code inner
	//   [4,5]   = bold inner
	//   [6,7]   = strike inner
	//   [8,9]   = link text
	//   [10,11] = link href
	//   [12,13] = italic inner
	matches := combinedInlineRe.FindAllStringSubmatchIndex(text, -1)
	for _, m := range matches {
		fullStart, fullEnd := m[0], m[1]

		switch {
		case m[2] >= 0: // code
			segs = append(segs, segment{fullStart, fullEnd, adfMarkedText(text[m[2]:m[3]], "code")})
		case m[4] >= 0: // bold
			segs = append(segs, segment{fullStart, fullEnd, adfMarkedText(text[m[4]:m[5]], "strong")})
		case m[6] >= 0: // strike
			segs = append(segs, segment{fullStart, fullEnd, adfMarkedText(text[m[6]:m[7]], "strike")})
		case m[8] >= 0: // link
			segs = append(segs, segment{fullStart, fullEnd, adfLinkText(text[m[8]:m[9]], text[m[10]:m[11]])})
		case m[12] >= 0: // italic
			// The italic regex may consume a leading/trailing non-* char.
			// We adjust the segment to cover only the *...* portion.
			innerStart := strings.Index(text[fullStart:fullEnd], "*")
			if innerStart < 0 {
				break
			}
			actualStart := fullStart + innerStart
			actualEnd := actualStart + 1 + len(text[m[12]:m[13]]) + 1 // *content*
			if actualEnd > fullEnd {
				actualEnd = fullEnd
			}
			segs = append(segs, segment{actualStart, actualEnd, adfMarkedText(text[m[12]:m[13]], "em")})
		}
	}

	// Build the result by filling gaps with plain text (which may contain bare URLs).
	var nodes []interface{}
	cursor := 0
	for _, s := range segs {
		if s.start > cursor {
			nodes = append(nodes, autoLinkText(text[cursor:s.start])...)
		}
		if s.start >= cursor {
			nodes = append(nodes, s.node)
			cursor = s.end
		}
	}
	// Trailing text after last match.
	if cursor < len(text) {
		nodes = append(nodes, autoLinkText(text[cursor:])...)
	}

	// If nothing matched at all, auto-link the entire text.
	if len(nodes) == 0 {
		nodes = autoLinkText(text)
	}

	return nodes
}

// autoLinkText splits plain text on bare URLs, returning ADF text/link nodes.
func autoLinkText(text string) []interface{} {
	locs := urlRe.FindAllStringIndex(text, -1)
	if len(locs) == 0 {
		if text != "" {
			return []interface{}{adfPlainText(text)}
		}
		return nil
	}

	var nodes []interface{}
	cursor := 0
	for _, loc := range locs {
		if loc[0] > cursor {
			nodes = append(nodes, adfPlainText(text[cursor:loc[0]]))
		}
		u := text[loc[0]:loc[1]]
		nodes = append(nodes, adfLinkText(u, u))
		cursor = loc[1]
	}
	if cursor < len(text) {
		nodes = append(nodes, adfPlainText(text[cursor:]))
	}
	return nodes
}

// makeParagraph creates an ADF paragraph node with parsed inline formatting.
func makeParagraph(text string) map[string]interface{} {
	inlines := parseInlineFormatting(text)
	if len(inlines) == 0 {
		inlines = []interface{}{adfPlainText("")}
	}
	return map[string]interface{}{
		"type":    "paragraph",
		"content": inlines,
	}
}

// makeListItem creates an ADF listItem wrapping a paragraph.
func makeListItem(text string) map[string]interface{} {
	return map[string]interface{}{
		"type":    "listItem",
		"content": []interface{}{makeParagraph(text)},
	}
}

// parseTable converts raw markdown table lines into an ADF table node.
// Returns nil if no data rows are found after filtering the separator row.
func parseTable(rows []string) map[string]interface{} {
	var dataRows [][]string

	for _, row := range rows {
		trimmed := strings.Trim(row, "|")
		cells := strings.Split(trimmed, "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		// Skip separator rows like |---|---|
		isSep := true
		for _, c := range cells {
			if c == "" {
				continue
			}
			if !tableSepRe.MatchString(c) {
				isSep = false
				break
			}
		}
		if isSep && len(dataRows) > 0 {
			// Only skip if it looks like a separator after a header row.
			continue
		}
		if !isSep {
			dataRows = append(dataRows, cells)
		}
	}

	if len(dataRows) == 0 {
		return nil
	}

	adfRows := make([]interface{}, 0, len(dataRows))
	for idx, cells := range dataRows {
		cellType := "tableCell"
		if idx == 0 {
			cellType = "tableHeader"
		}
		adfCells := make([]interface{}, 0, len(cells))
		for _, cellText := range cells {
			inlines := parseInlineFormatting(cellText)
			if len(inlines) == 0 {
				inlines = []interface{}{adfPlainText("")}
			}
			adfCells = append(adfCells, map[string]interface{}{
				"type": cellType,
				"content": []interface{}{
					map[string]interface{}{"type": "paragraph", "content": inlines},
				},
			})
		}
		adfRows = append(adfRows, map[string]interface{}{
			"type":    "tableRow",
			"content": adfCells,
		})
	}

	return map[string]interface{}{
		"type": "table",
		"attrs": map[string]interface{}{
			"isNumberColumnEnabled": false,
			"layout":                "default",
		},
		"content": adfRows,
	}
}

// --- ADF node helpers ---

// adfPlainText creates a plain ADF text node.
func adfPlainText(t string) map[string]interface{} {
	return map[string]interface{}{"type": "text", "text": t}
}

// adfMarkedText creates an ADF text node with a single mark (strong, em, code, strike).
func adfMarkedText(t, markType string) map[string]interface{} {
	return map[string]interface{}{
		"type": "text",
		"text": t,
		"marks": []interface{}{
			map[string]interface{}{"type": markType},
		},
	}
}

// adfLinkText creates an ADF text node with a link mark.
func adfLinkText(display, href string) map[string]interface{} {
	return map[string]interface{}{
		"type": "text",
		"text": display,
		"marks": []interface{}{
			map[string]interface{}{
				"type":  "link",
				"attrs": map[string]interface{}{"href": href},
			},
		},
	}
}
