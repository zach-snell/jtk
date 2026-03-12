package jira_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	jira "github.com/zach-snell/jtk/internal/jira"
)

// update flag is declared in flattener_test.go (same package).

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustMarshalJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return string(b)
}

// docContent extracts the top-level "content" slice from an ADF doc.
func docContent(t *testing.T, doc map[string]interface{}) []interface{} {
	t.Helper()
	c, ok := doc["content"].([]interface{})
	if !ok {
		t.Fatalf("doc[\"content\"] is not []interface{}, got %T", doc["content"])
	}
	return c
}

// nodeType extracts the "type" string from an ADF node.
func nodeType(t *testing.T, node interface{}) string {
	t.Helper()
	m, ok := node.(map[string]interface{})
	if !ok {
		t.Fatalf("node is not map[string]interface{}, got %T", node)
	}
	tp, ok := m["type"].(string)
	if !ok {
		t.Fatalf("node[\"type\"] is not string, got %T", m["type"])
	}
	return tp
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_DocumentStructure — every result is a valid ADF doc
// ---------------------------------------------------------------------------

func TestMarkdownToADF_DocumentStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "plain text", input: "hello world"},
		{name: "heading", input: "# Title"},
		{name: "whitespace only", input: "   \n\n   "},
		{name: "literal backslash-n", input: `hello\nworld`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			if doc == nil {
				t.Fatal("MarkdownToADF returned nil")
			}

			docType, ok := doc["type"].(string)
			if !ok || docType != "doc" {
				t.Errorf("expected type=doc, got %v", doc["type"])
			}
			ver, ok := doc["version"].(int)
			if !ok || ver != 1 {
				t.Errorf("expected version=1, got %v", doc["version"])
			}
			content := docContent(t, doc)
			if len(content) == 0 {
				t.Error("expected at least one content node")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_EmptyInput
// ---------------------------------------------------------------------------

func TestMarkdownToADF_EmptyInput(t *testing.T) {
	t.Parallel()

	doc := jira.MarkdownToADF("")
	content := docContent(t, doc)
	if len(content) != 1 {
		t.Fatalf("expected 1 content node, got %d", len(content))
	}
	if nodeType(t, content[0]) != "paragraph" {
		t.Errorf("expected paragraph node, got %s", nodeType(t, content[0]))
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_Headings
// ---------------------------------------------------------------------------

func TestMarkdownToADF_Headings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantLevel int
		wantText  string
	}{
		{name: "h1", input: "# Hello", wantLevel: 1, wantText: "Hello"},
		{name: "h2", input: "## World", wantLevel: 2, wantText: "World"},
		{name: "h3", input: "### Foo", wantLevel: 3, wantText: "Foo"},
		{name: "h4", input: "#### Bar", wantLevel: 4, wantText: "Bar"},
		{name: "h5", input: "##### Baz", wantLevel: 5, wantText: "Baz"},
		{name: "h6", input: "###### Qux", wantLevel: 6, wantText: "Qux"},
		{name: "h1 with inline bold", input: "# **Bold Title**", wantLevel: 1, wantText: "Bold Title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			content := docContent(t, doc)
			if len(content) != 1 {
				t.Fatalf("expected 1 node, got %d", len(content))
			}

			node := content[0].(map[string]interface{})
			if node["type"] != "heading" {
				t.Fatalf("expected heading, got %s", node["type"])
			}
			attrs := node["attrs"].(map[string]interface{})
			if attrs["level"] != tt.wantLevel {
				t.Errorf("level = %v, want %d", attrs["level"], tt.wantLevel)
			}
			// Check that the heading contains text with the expected value.
			inlines := node["content"].([]interface{})
			if len(inlines) == 0 {
				t.Fatal("heading has no inline content")
			}
			firstInline := inlines[0].(map[string]interface{})
			if firstInline["text"] != tt.wantText {
				t.Errorf("text = %q, want %q", firstInline["text"], tt.wantText)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_InlineFormatting
// ---------------------------------------------------------------------------

func TestMarkdownToADF_InlineFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantMark string // expected mark type on first inline node
		wantText string
	}{
		{name: "bold", input: "**bold text**", wantMark: "strong", wantText: "bold text"},
		{name: "italic", input: "some *italic* here", wantMark: "em", wantText: "italic"},
		{name: "strikethrough", input: "~~struck~~", wantMark: "strike", wantText: "struck"},
		{name: "inline code", input: "`code here`", wantMark: "code", wantText: "code here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			content := docContent(t, doc)
			if len(content) != 1 {
				t.Fatalf("expected 1 content node, got %d", len(content))
			}
			para := content[0].(map[string]interface{})
			if para["type"] != "paragraph" {
				t.Fatalf("expected paragraph, got %s", para["type"])
			}

			inlines := para["content"].([]interface{})
			// Find the node with the expected mark.
			var found bool
			for _, inline := range inlines {
				n := inline.(map[string]interface{})
				marks, ok := n["marks"].([]interface{})
				if !ok || len(marks) == 0 {
					continue
				}
				mark := marks[0].(map[string]interface{})
				if mark["type"] == tt.wantMark && n["text"] == tt.wantText {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("did not find inline node with mark=%q text=%q in %s",
					tt.wantMark, tt.wantText, mustMarshalJSON(t, inlines))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_Links
// ---------------------------------------------------------------------------

func TestMarkdownToADF_Links(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantText string
		wantHref string
	}{
		{
			name:     "markdown link",
			input:    "[click me](https://example.com)",
			wantText: "click me",
			wantHref: "https://example.com",
		},
		{
			name:     "bare URL",
			input:    "visit https://example.com/path today",
			wantText: "https://example.com/path",
			wantHref: "https://example.com/path",
		},
		{
			name:     "bare http URL",
			input:    "http://insecure.example.com",
			wantText: "http://insecure.example.com",
			wantHref: "http://insecure.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			content := docContent(t, doc)
			para := content[0].(map[string]interface{})
			inlines := para["content"].([]interface{})

			var found bool
			for _, inline := range inlines {
				n := inline.(map[string]interface{})
				marks, ok := n["marks"].([]interface{})
				if !ok || len(marks) == 0 {
					continue
				}
				mark := marks[0].(map[string]interface{})
				if mark["type"] != "link" {
					continue
				}
				attrs, ok := mark["attrs"].(map[string]interface{})
				if !ok {
					continue
				}
				if n["text"] == tt.wantText && attrs["href"] == tt.wantHref {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("did not find link node text=%q href=%q in %s",
					tt.wantText, tt.wantHref, mustMarshalJSON(t, inlines))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_CodeBlock
// ---------------------------------------------------------------------------

func TestMarkdownToADF_CodeBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantLang string
		wantCode string
	}{
		{
			name:     "code block with language",
			input:    "```go\nfmt.Println(\"hi\")\n```",
			wantLang: "go",
			wantCode: "fmt.Println(\"hi\")",
		},
		{
			name:     "code block without language",
			input:    "```\nplain code\n```",
			wantLang: "",
			wantCode: "plain code",
		},
		{
			name:     "multiline code block",
			input:    "```python\nx = 1\ny = 2\nprint(x + y)\n```",
			wantLang: "python",
			wantCode: "x = 1\ny = 2\nprint(x + y)",
		},
		{
			name:     "unclosed code block",
			input:    "```go\nfmt.Println(\"oops\")",
			wantLang: "go",
			wantCode: "fmt.Println(\"oops\")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			content := docContent(t, doc)
			if len(content) < 1 {
				t.Fatal("expected at least 1 content node")
			}

			cb := content[0].(map[string]interface{})
			if cb["type"] != "codeBlock" {
				t.Fatalf("expected codeBlock, got %s", cb["type"])
			}

			attrs := cb["attrs"].(map[string]interface{})
			gotLang, _ := attrs["language"].(string)
			if gotLang != tt.wantLang {
				t.Errorf("language = %q, want %q", gotLang, tt.wantLang)
			}

			cbContent := cb["content"].([]interface{})
			if len(cbContent) != 1 {
				t.Fatalf("codeBlock content has %d nodes, want 1", len(cbContent))
			}
			textNode := cbContent[0].(map[string]interface{})
			if textNode["text"] != tt.wantCode {
				t.Errorf("code text = %q, want %q", textNode["text"], tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_Lists
// ---------------------------------------------------------------------------

func TestMarkdownToADF_Lists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantType string
		wantLen  int
	}{
		{
			name:     "unordered dash",
			input:    "- alpha\n- beta\n- gamma",
			wantType: "bulletList",
			wantLen:  3,
		},
		{
			name:     "unordered asterisk",
			input:    "* one\n* two",
			wantType: "bulletList",
			wantLen:  2,
		},
		{
			name:     "ordered",
			input:    "1. first\n2. second\n3. third",
			wantType: "orderedList",
			wantLen:  3,
		},
		{
			name:     "single item unordered",
			input:    "- alone",
			wantType: "bulletList",
			wantLen:  1,
		},
		{
			name:     "single item ordered",
			input:    "1. only",
			wantType: "orderedList",
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			content := docContent(t, doc)
			if len(content) < 1 {
				t.Fatal("expected at least 1 content node")
			}

			list := content[0].(map[string]interface{})
			if list["type"] != tt.wantType {
				t.Fatalf("type = %s, want %s", list["type"], tt.wantType)
			}
			items := list["content"].([]interface{})
			if len(items) != tt.wantLen {
				t.Errorf("list has %d items, want %d", len(items), tt.wantLen)
			}

			// Every item should be a listItem containing a paragraph.
			for i, item := range items {
				li := item.(map[string]interface{})
				if li["type"] != "listItem" {
					t.Errorf("items[%d] type = %s, want listItem", i, li["type"])
				}
				liContent := li["content"].([]interface{})
				if len(liContent) == 0 {
					t.Errorf("items[%d] has no content", i)
					continue
				}
				p := liContent[0].(map[string]interface{})
				if p["type"] != "paragraph" {
					t.Errorf("items[%d] inner type = %s, want paragraph", i, p["type"])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_Blockquote
// ---------------------------------------------------------------------------

func TestMarkdownToADF_Blockquote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantParas int
	}{
		{
			name:      "single line quote",
			input:     "> hello",
			wantParas: 1,
		},
		{
			name:      "multi line quote",
			input:     "> line one\n> line two\n> line three",
			wantParas: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			content := docContent(t, doc)
			if len(content) < 1 {
				t.Fatal("expected at least 1 content node")
			}

			bq := content[0].(map[string]interface{})
			if bq["type"] != "blockquote" {
				t.Fatalf("type = %s, want blockquote", bq["type"])
			}
			paras := bq["content"].([]interface{})
			if len(paras) != tt.wantParas {
				t.Errorf("blockquote has %d paragraphs, want %d", len(paras), tt.wantParas)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_Table
// ---------------------------------------------------------------------------

func TestMarkdownToADF_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantRows     int
		wantFirstRow string // "tableHeader" or "tableCell"
	}{
		{
			name:         "basic table",
			input:        "| a | b |\n|---|---|\n| 1 | 2 |",
			wantRows:     2,
			wantFirstRow: "tableHeader",
		},
		{
			name:         "table without separator",
			input:        "| x | y |\n| 1 | 2 |",
			wantRows:     2,
			wantFirstRow: "tableHeader",
		},
		{
			name:         "single row table",
			input:        "| col1 | col2 |",
			wantRows:     1,
			wantFirstRow: "tableHeader",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			content := docContent(t, doc)
			if len(content) < 1 {
				t.Fatal("expected at least 1 content node")
			}

			table := content[0].(map[string]interface{})
			if table["type"] != "table" {
				t.Fatalf("type = %s, want table", table["type"])
			}

			attrs := table["attrs"].(map[string]interface{})
			if attrs["layout"] != "default" {
				t.Errorf("layout = %v, want default", attrs["layout"])
			}
			if attrs["isNumberColumnEnabled"] != false {
				t.Errorf("isNumberColumnEnabled = %v, want false", attrs["isNumberColumnEnabled"])
			}

			rows := table["content"].([]interface{})
			if len(rows) != tt.wantRows {
				t.Fatalf("table has %d rows, want %d", len(rows), tt.wantRows)
			}

			// Check first row cell type.
			firstRow := rows[0].(map[string]interface{})
			cells := firstRow["content"].([]interface{})
			if len(cells) > 0 {
				firstCell := cells[0].(map[string]interface{})
				if firstCell["type"] != tt.wantFirstRow {
					t.Errorf("first row cell type = %s, want %s", firstCell["type"], tt.wantFirstRow)
				}
			}

			// Non-header rows should use tableCell.
			for i := 1; i < len(rows); i++ {
				row := rows[i].(map[string]interface{})
				rowCells := row["content"].([]interface{})
				for j, c := range rowCells {
					cell := c.(map[string]interface{})
					if cell["type"] != "tableCell" {
						t.Errorf("row[%d] cell[%d] type = %s, want tableCell", i, j, cell["type"])
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_HorizontalRule
// ---------------------------------------------------------------------------

func TestMarkdownToADF_HorizontalRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "dashes", input: "---"},
		{name: "long dashes", input: "-----"},
		{name: "asterisks", input: "***"},
		{name: "underscores", input: "___"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := jira.MarkdownToADF(tt.input)
			content := docContent(t, doc)
			if len(content) < 1 {
				t.Fatal("expected at least 1 content node")
			}

			rule := content[0].(map[string]interface{})
			if rule["type"] != "rule" {
				t.Errorf("type = %s, want rule", rule["type"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_LiteralBackslashN — \n in input (JSON payload style)
// ---------------------------------------------------------------------------

func TestMarkdownToADF_LiteralBackslashN(t *testing.T) {
	t.Parallel()

	// The converter replaces literal \n with newlines before processing.
	doc := jira.MarkdownToADF(`# Title\n\nSome text`)
	content := docContent(t, doc)

	if len(content) < 2 {
		t.Fatalf("expected at least 2 content nodes (heading + paragraph), got %d", len(content))
	}
	if nodeType(t, content[0]) != "heading" {
		t.Errorf("content[0] type = %s, want heading", nodeType(t, content[0]))
	}
	if nodeType(t, content[1]) != "paragraph" {
		t.Errorf("content[1] type = %s, want paragraph", nodeType(t, content[1]))
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_Paragraphs — consecutive plain text lines
// ---------------------------------------------------------------------------

func TestMarkdownToADF_Paragraphs(t *testing.T) {
	t.Parallel()

	doc := jira.MarkdownToADF("line one\nline two")
	content := docContent(t, doc)
	if len(content) != 2 {
		t.Fatalf("expected 2 paragraph nodes, got %d", len(content))
	}
	for i, c := range content {
		if nodeType(t, c) != "paragraph" {
			t.Errorf("content[%d] type = %s, want paragraph", i, nodeType(t, c))
		}
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_MixedContent — heading + paragraph + list + code
// ---------------------------------------------------------------------------

func TestMarkdownToADF_MixedContent(t *testing.T) {
	t.Parallel()

	input := "# Title\n\nSome text.\n\n- item 1\n- item 2\n\n```go\ncode()\n```"
	doc := jira.MarkdownToADF(input)
	content := docContent(t, doc)

	wantTypes := []string{"heading", "paragraph", "bulletList", "codeBlock"}
	if len(content) < len(wantTypes) {
		t.Fatalf("expected at least %d nodes, got %d: %s", len(wantTypes), len(content), mustMarshalJSON(t, content))
	}

	for i, want := range wantTypes {
		got := nodeType(t, content[i])
		if got != want {
			t.Errorf("content[%d] type = %s, want %s", i, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_SpecialCharacters
// ---------------------------------------------------------------------------

func TestMarkdownToADF_SpecialCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "angle brackets", input: "use <div> tags"},
		{name: "ampersand", input: "a & b"},
		{name: "unicode", input: "emoji: "},
		{name: "japanese", input: ""},
		{name: "backslashes", input: `path\to\file`},
		{name: "quotes", input: `she said "hello"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Primary assertion: no panic, returns valid doc.
			doc := jira.MarkdownToADF(tt.input)
			if doc == nil {
				t.Fatal("returned nil")
			}
			content := docContent(t, doc)
			if len(content) == 0 {
				t.Error("expected at least one content node")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestParseInlineFormatting — via export bridge
// ---------------------------------------------------------------------------

func TestParseInlineFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantLen  int // number of inline nodes returned
		checkIdx int // which node index to check
		checkFn  func(t *testing.T, node map[string]interface{})
	}{
		{
			name:    "empty string",
			input:   "",
			wantLen: 0,
		},
		{
			name:     "plain text",
			input:    "just text",
			wantLen:  1,
			checkIdx: 0,
			checkFn: func(t *testing.T, node map[string]interface{}) {
				t.Helper()
				if node["text"] != "just text" {
					t.Errorf("text = %q, want %q", node["text"], "just text")
				}
				if _, has := node["marks"]; has {
					t.Error("plain text should have no marks")
				}
			},
		},
		{
			name:     "bold",
			input:    "**hello**",
			wantLen:  1,
			checkIdx: 0,
			checkFn: func(t *testing.T, node map[string]interface{}) {
				t.Helper()
				marks := node["marks"].([]interface{})
				mark := marks[0].(map[string]interface{})
				if mark["type"] != "strong" {
					t.Errorf("mark type = %s, want strong", mark["type"])
				}
			},
		},
		{
			name:    "mixed bold and plain",
			input:   "pre **mid** post",
			wantLen: 3,
		},
		{
			name:    "bare URL in middle",
			input:   "see https://example.com here",
			wantLen: 3, // "see " + link + " here"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jira.ParseInlineFormatting(tt.input)
			if len(got) != tt.wantLen {
				t.Fatalf("ParseInlineFormatting(%q) returned %d nodes, want %d: %s",
					tt.input, len(got), tt.wantLen, mustMarshalJSON(t, got))
			}
			if tt.checkFn != nil && len(got) > tt.checkIdx {
				tt.checkFn(t, got[tt.checkIdx].(map[string]interface{}))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAutoLinkText — via export bridge
// ---------------------------------------------------------------------------

func TestAutoLinkText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{name: "no URLs", input: "plain text", wantLen: 1},
		{name: "empty", input: "", wantLen: 0},
		{name: "single URL", input: "https://example.com", wantLen: 1},
		{name: "URL in text", input: "see https://x.com here", wantLen: 3},
		{name: "multiple URLs", input: "https://a.com and https://b.com", wantLen: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jira.AutoLinkText(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("AutoLinkText(%q) returned %d nodes, want %d: %s",
					tt.input, len(got), tt.wantLen, mustMarshalJSON(t, got))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAdfHelpers — adfPlainText, adfMarkedText, adfLinkText via bridge
// ---------------------------------------------------------------------------

func TestAdfPlainText(t *testing.T) {
	t.Parallel()

	want := map[string]interface{}{
		"type": "text",
		"text": "hello",
	}
	got := jira.AdfPlainText("hello")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AdfPlainText mismatch (-want +got):\n%s", diff)
	}
}

func TestAdfMarkedText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		markType string
	}{
		{name: "strong", text: "bold", markType: "strong"},
		{name: "em", text: "italic", markType: "em"},
		{name: "code", text: "x", markType: "code"},
		{name: "strike", text: "old", markType: "strike"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := map[string]interface{}{
				"type": "text",
				"text": tt.text,
				"marks": []interface{}{
					map[string]interface{}{"type": tt.markType},
				},
			}
			got := jira.AdfMarkedText(tt.text, tt.markType)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("AdfMarkedText mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdfLinkText(t *testing.T) {
	t.Parallel()

	want := map[string]interface{}{
		"type": "text",
		"text": "click",
		"marks": []interface{}{
			map[string]interface{}{
				"type":  "link",
				"attrs": map[string]interface{}{"href": "https://example.com"},
			},
		},
	}
	got := jira.AdfLinkText("click", "https://example.com")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AdfLinkText mismatch (-want +got):\n%s", diff)
	}
}

// ---------------------------------------------------------------------------
// TestParseTable — via export bridge
// ---------------------------------------------------------------------------

func TestParseTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rows     []string
		wantNil  bool
		wantRows int
	}{
		{
			name:     "standard table",
			rows:     []string{"| a | b |", "|---|---|", "| 1 | 2 |"},
			wantRows: 2, // header + 1 data row (separator filtered out)
		},
		{
			name:    "only separator rows",
			rows:    []string{"|---|---|"},
			wantNil: true,
		},
		{
			name:     "no separator",
			rows:     []string{"| x | y |", "| 1 | 2 |"},
			wantRows: 2,
		},
		{
			name:     "three data rows",
			rows:     []string{"| h1 | h2 |", "|---|---|", "| a | b |", "| c | d |"},
			wantRows: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jira.ParseTable(tt.rows)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil table, got nil")
			}
			rows := got["content"].([]interface{})
			if len(rows) != tt.wantRows {
				t.Errorf("table has %d rows, want %d", len(rows), tt.wantRows)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMarkdownToADF_Golden — golden file tests for full ADF output
// ---------------------------------------------------------------------------

func TestMarkdownToADF_Golden(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/*.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test fixtures found in testdata/")
	}

	for _, inputFile := range files {
		name := strings.TrimSuffix(filepath.Base(inputFile), ".md")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(inputFile)
			if err != nil {
				t.Fatal(err)
			}

			got := jira.MarkdownToADF(string(input))
			gotJSON, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal result: %v", err)
			}

			goldenFile := strings.TrimSuffix(inputFile, ".md") + ".golden"
			if *update {
				if err := os.WriteFile(goldenFile, gotJSON, 0644); err != nil {
					t.Fatal(err)
				}
				t.Logf("updated golden file: %s", goldenFile)
				return
			}

			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("golden file not found (run with -update to create): %v", err)
			}
			if diff := cmp.Diff(string(want), string(gotJSON)); diff != "" {
				t.Errorf("golden mismatch for %s (-want +got):\n%s", name, diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FuzzMarkdownToADF — fuzz test: no input should cause a panic
// ---------------------------------------------------------------------------

func FuzzMarkdownToADF(f *testing.F) {
	// Seed corpus covering each code path.
	seeds := []string{
		"",
		"plain text",
		"# Heading 1",
		"## Heading 2",
		"### Heading 3",
		"#### Heading 4",
		"##### Heading 5",
		"###### Heading 6",
		"**bold** and *italic*",
		"`inline code`",
		"~~strikethrough~~",
		"[link](https://example.com)",
		"https://bare-url.com in text",
		"```go\nfmt.Println()\n```",
		"```\nno language\n```",
		"> blockquote",
		"> line 1\n> line 2",
		"- item 1\n- item 2",
		"* star 1\n* star 2",
		"1. first\n2. second",
		"| a | b |\n|---|---|\n| 1 | 2 |",
		"---",
		"***",
		"___",
		"mixed\ncontent\nwith\nnewlines",
		`literal\nnewlines`,
		"# **Bold heading**",
		"| only | header |",
		"> quote with **bold**",
		"- list with `code` item",
		"a & b < c > d",
		"emoji:  unicode: ",
		"```\n```",
		"**",
		"*",
		"~~",
		"[]",
		"[]()",
		"|",
		"||",
		"```",
		"> ",
		"- ",
		"1. ",
		"######",
		"# ",
		"",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Property 1: must never panic.
		result := jira.MarkdownToADF(input)

		// Property 2: always returns a non-nil doc.
		if result == nil {
			t.Error("MarkdownToADF returned nil")
			return
		}

		// Property 3: type is always "doc".
		docType, ok := result["type"].(string)
		if !ok || docType != "doc" {
			t.Errorf("expected type=doc, got %v", result["type"])
		}

		// Property 4: version is always 1.
		ver, ok := result["version"].(int)
		if !ok || ver != 1 {
			t.Errorf("expected version=1, got %v", result["version"])
		}

		// Property 5: content is always a non-empty slice.
		content, ok := result["content"].([]interface{})
		if !ok {
			t.Errorf("content is not []interface{}, got %T", result["content"])
			return
		}
		if len(content) == 0 {
			t.Error("content slice is empty")
		}

		// Property 6: every top-level node has a "type" field.
		for i, node := range content {
			m, ok := node.(map[string]interface{})
			if !ok {
				t.Errorf("content[%d] is not map[string]interface{}", i)
				continue
			}
			if _, hasType := m["type"]; !hasType {
				t.Errorf("content[%d] missing type field", i)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// BenchmarkMarkdownToADF — benchmarks for hot paths
// ---------------------------------------------------------------------------

func BenchmarkMarkdownToADF(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"plain_paragraph", "Just a simple paragraph of plain text."},
		{"heading", "# Hello World"},
		{"inline_formatting", "This has **bold**, *italic*, ~~strike~~, and `code`."},
		{"code_block", "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"},
		{"unordered_list", "- one\n- two\n- three\n- four\n- five"},
		{"ordered_list", "1. first\n2. second\n3. third"},
		{"table", "| a | b | c |\n|---|---|---|\n| 1 | 2 | 3 |\n| 4 | 5 | 6 |"},
		{"blockquote", "> This is a blockquote\n> with multiple lines"},
		{"bare_url", "Visit https://example.com/very/long/path?query=string&param=value for details."},
		{"complex_doc", "# Title\n\n**Bold** and *italic* with `code`.\n\n- item 1\n- item 2\n\n```go\ncode()\n```\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n---\n\n> quote\n\nhttps://example.com"},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				jira.MarkdownToADF(bc.input)
			}
		})
	}
}

func BenchmarkParseInlineFormatting(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"plain", "just some plain text nothing special"},
		{"bold_italic", "**bold** and *italic* mixed together"},
		{"link", "[click](https://example.com) and https://bare.url here"},
		{"code_strike", "`code` and ~~strikethrough~~ here"},
		{"all_marks", "**bold** *italic* ~~strike~~ `code` [link](https://x.com)"},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				jira.ParseInlineFormatting(bc.input)
			}
		})
	}
}
