package jira_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	jira "github.com/zach-snell/jtk/internal/jira"
)

// update controls golden file regeneration: go test -update
var update = flag.Bool("update", false, "update .golden files")

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustReadJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("failed to unmarshal %s: %v", path, err)
	}
	return out
}

func ptrFloat64(v float64) *float64 { return &v }

// ---------------------------------------------------------------------------
// TestFlattenIssue — raw map → FlattenedIssue
// ---------------------------------------------------------------------------

func TestFlattenIssue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  map[string]interface{}
		want *jira.FlattenedIssue
	}{
		{
			name: "nil fields returns key only",
			raw: map[string]interface{}{
				"key": "PROJ-1",
			},
			want: &jira.FlattenedIssue{
				Key: "PROJ-1",
			},
		},
		{
			name: "empty map returns empty issue",
			raw:  map[string]interface{}{},
			want: &jira.FlattenedIssue{},
		},
		{
			name: "fields is wrong type returns key only",
			raw: map[string]interface{}{
				"key":    "PROJ-2",
				"fields": "not-a-map",
			},
			want: &jira.FlattenedIssue{
				Key: "PROJ-2",
			},
		},
		{
			name: "full issue with all fields",
			raw: map[string]interface{}{
				"key": "PROJ-100",
				"fields": map[string]interface{}{
					"summary":   "Implement caching layer",
					"status":    map[string]interface{}{"name": "In Progress"},
					"issuetype": map[string]interface{}{"name": "Story"},
					"priority":  map[string]interface{}{"name": "High"},
					"assignee":  map[string]interface{}{"displayName": "Alice"},
					"reporter":  map[string]interface{}{"displayName": "Bob"},
					"created":   "2024-01-15T10:00:00.000+0000",
					"updated":   "2024-02-20T14:30:00.000+0000",
					"labels":    []interface{}{"backend", "performance"},
					"components": []interface{}{
						map[string]interface{}{"name": "API"},
						map[string]interface{}{"name": "Database"},
					},
					"sprint": map[string]interface{}{
						"name": "Sprint 42",
					},
					"story_points": 8.0,
					"parent":       map[string]interface{}{"key": "PROJ-50"},
					"description": map[string]interface{}{
						"type":    "doc",
						"version": 1,
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{
										"type": "text",
										"text": "Implement Redis caching.",
									},
								},
							},
						},
					},
				},
			},
			want: &jira.FlattenedIssue{
				Key:         "PROJ-100",
				Summary:     "Implement caching layer",
				Status:      "In Progress",
				Type:        "Story",
				Priority:    "High",
				Assignee:    "Alice",
				Reporter:    "Bob",
				Created:     "2024-01-15T10:00:00.000+0000",
				Updated:     "2024-02-20T14:30:00.000+0000",
				Labels:      []string{"backend", "performance"},
				Components:  []string{"API", "Database"},
				Sprint:      "Sprint 42",
				StoryPoints: 8.0,
				ParentKey:   "PROJ-50",
				Description: "Implement Redis caching.",
			},
		},
		{
			name: "no assignee or reporter defaults to unassigned",
			raw: map[string]interface{}{
				"key": "PROJ-3",
				"fields": map[string]interface{}{
					"summary": "Bug fix",
				},
			},
			want: &jira.FlattenedIssue{
				Key:      "PROJ-3",
				Summary:  "Bug fix",
				Assignee: "unassigned",
				Reporter: "unassigned",
			},
		},
		{
			name: "assignee with empty displayName defaults to unassigned",
			raw: map[string]interface{}{
				"key": "PROJ-4",
				"fields": map[string]interface{}{
					"summary":  "Task",
					"assignee": map[string]interface{}{"displayName": ""},
				},
			},
			want: &jira.FlattenedIssue{
				Key:      "PROJ-4",
				Summary:  "Task",
				Assignee: "unassigned",
				Reporter: "unassigned",
			},
		},
		{
			name: "story points via customfield_10016",
			raw: map[string]interface{}{
				"key": "PROJ-5",
				"fields": map[string]interface{}{
					"summary":           "Epic work",
					"customfield_10016": 5.0,
				},
			},
			want: &jira.FlattenedIssue{
				Key:         "PROJ-5",
				Summary:     "Epic work",
				Assignee:    "unassigned",
				Reporter:    "unassigned",
				StoryPoints: 5.0,
			},
		},
		{
			name: "labels with non-string entries are skipped",
			raw: map[string]interface{}{
				"key": "PROJ-6",
				"fields": map[string]interface{}{
					"summary": "Mixed labels",
					"labels":  []interface{}{"valid", 42, "also-valid", nil},
				},
			},
			want: &jira.FlattenedIssue{
				Key:      "PROJ-6",
				Summary:  "Mixed labels",
				Labels:   []string{"valid", "also-valid"},
				Assignee: "unassigned",
				Reporter: "unassigned",
			},
		},
		{
			name: "components with missing name field are skipped",
			raw: map[string]interface{}{
				"key": "PROJ-7",
				"fields": map[string]interface{}{
					"summary": "Partial components",
					"components": []interface{}{
						map[string]interface{}{"name": "Frontend"},
						map[string]interface{}{"id": "123"}, // no name
						"not-a-map",
					},
				},
			},
			want: &jira.FlattenedIssue{
				Key:        "PROJ-7",
				Summary:    "Partial components",
				Components: []string{"Frontend"},
				Assignee:   "unassigned",
				Reporter:   "unassigned",
			},
		},
		{
			name: "nil description produces empty string",
			raw: map[string]interface{}{
				"key": "PROJ-8",
				"fields": map[string]interface{}{
					"summary":     "No description",
					"description": nil,
				},
			},
			want: &jira.FlattenedIssue{
				Key:         "PROJ-8",
				Summary:     "No description",
				Description: "",
				Assignee:    "unassigned",
				Reporter:    "unassigned",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jira.FlattenIssue(tt.raw)
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FlattenIssue() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFlattenIssueFromTyped — typed Issue → FlattenedIssue
// ---------------------------------------------------------------------------

func TestFlattenIssueFromTyped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		issue *jira.Issue
		want  *jira.FlattenedIssue
	}{
		{
			name: "minimal issue with nil pointers",
			issue: &jira.Issue{
				Key: "MIN-1",
				Fields: jira.IssueFields{
					Summary: "Bare minimum",
					Created: "2024-01-01",
					Updated: "2024-01-02",
				},
			},
			want: &jira.FlattenedIssue{
				Key:     "MIN-1",
				Summary: "Bare minimum",
				Created: "2024-01-01",
				Updated: "2024-01-02",
			},
		},
		{
			name: "full issue with all optional fields",
			issue: &jira.Issue{
				Key: "FULL-1",
				Fields: jira.IssueFields{
					Summary:        "Full issue",
					Created:        "2024-03-01T09:00:00.000+0000",
					Updated:        "2024-03-15T17:00:00.000+0000",
					Status:         &jira.Status{Name: "Done"},
					IssueType:      &jira.IssueType{Name: "Bug"},
					Priority:       &jira.Priority{Name: "Critical"},
					Assignee:       &jira.User{DisplayName: "Charlie"},
					Reporter:       &jira.User{DisplayName: "Dana"},
					Labels:         []string{"urgent", "production"},
					DueDate:        "2024-04-01",
					ResolutionDate: "2024-03-14",
					Resolution:     &jira.Status{Name: "Fixed"},
					Components: []jira.Component{
						{Name: "Backend"},
						{Name: "API"},
					},
					FixVersions: []jira.Version{
						{Name: "v2.0"},
						{Name: "v2.1"},
					},
					Sprint:      &jira.Sprint{Name: "Sprint 10"},
					StoryPoints: ptrFloat64(13),
					Parent:      &jira.Issue{Key: "EPIC-5"},
					Description: map[string]interface{}{
						"type":    "doc",
						"version": 1,
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{
										"type": "text",
										"text": "Fix production crash.",
									},
								},
							},
						},
					},
				},
			},
			want: &jira.FlattenedIssue{
				Key:            "FULL-1",
				Summary:        "Full issue",
				Status:         "Done",
				Type:           "Bug",
				Priority:       "Critical",
				Assignee:       "Charlie",
				Reporter:       "Dana",
				Created:        "2024-03-01T09:00:00.000+0000",
				Updated:        "2024-03-15T17:00:00.000+0000",
				Labels:         []string{"urgent", "production"},
				DueDate:        "2024-04-01",
				Resolution:     "Fixed",
				ResolutionDate: "2024-03-14",
				Components:     []string{"Backend", "API"},
				FixVersions:    []string{"v2.0", "v2.1"},
				Sprint:         "Sprint 10",
				StoryPoints:    13,
				ParentKey:      "EPIC-5",
				Description:    "Fix production crash.",
			},
		},
		{
			name: "nil status and priority and assignee",
			issue: &jira.Issue{
				Key: "NIL-1",
				Fields: jira.IssueFields{
					Summary: "Nil optionals",
				},
			},
			want: &jira.FlattenedIssue{
				Key:     "NIL-1",
				Summary: "Nil optionals",
			},
		},
		{
			name: "empty labels and components stay nil",
			issue: &jira.Issue{
				Key: "EMPTY-1",
				Fields: jira.IssueFields{
					Summary:    "Empty slices",
					Labels:     []string{},
					Components: []jira.Component{},
				},
			},
			want: &jira.FlattenedIssue{
				Key:     "EMPTY-1",
				Summary: "Empty slices",
				Labels:  []string{},
			},
		},
		{
			name: "nil description produces empty string",
			issue: &jira.Issue{
				Key: "DESC-1",
				Fields: jira.IssueFields{
					Summary:     "No desc",
					Description: nil,
				},
			},
			want: &jira.FlattenedIssue{
				Key:     "DESC-1",
				Summary: "No desc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jira.FlattenIssueFromTyped(tt.issue)
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FlattenIssueFromTyped() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestADFToPlainText — table-driven unit tests
// ---------------------------------------------------------------------------

func TestADFToPlainText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{
			name:  "nil input",
			input: nil,
			want:  "",
		},
		{
			name:  "string passthrough",
			input: "plain string description",
			want:  "plain string description",
		},
		{
			name:  "integer input returns empty",
			input: 42,
			want:  "",
		},
		{
			name:  "bool input returns empty",
			input: true,
			want:  "",
		},
		{
			name:  "slice input returns empty",
			input: []interface{}{"a", "b"},
			want:  "",
		},
		{
			name: "empty doc",
			input: map[string]interface{}{
				"type":    "doc",
				"version": 1,
				"content": []interface{}{},
			},
			want: "",
		},
		{
			name: "simple paragraph",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": "Hello world",
							},
						},
					},
				},
			},
			want: "Hello world",
		},
		{
			name: "bold text",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type":  "text",
								"text":  "important",
								"marks": []interface{}{map[string]interface{}{"type": "strong"}},
							},
						},
					},
				},
			},
			want: "**important**",
		},
		{
			name: "italic text",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type":  "text",
								"text":  "emphasis",
								"marks": []interface{}{map[string]interface{}{"type": "em"}},
							},
						},
					},
				},
			},
			want: "*emphasis*",
		},
		{
			name: "inline code",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type":  "text",
								"text":  "var x",
								"marks": []interface{}{map[string]interface{}{"type": "code"}},
							},
						},
					},
				},
			},
			want: "`var x`",
		},
		{
			name: "strikethrough text",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type":  "text",
								"text":  "removed",
								"marks": []interface{}{map[string]interface{}{"type": "strike"}},
							},
						},
					},
				},
			},
			want: "~~removed~~",
		},
		{
			name: "underline text",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type":  "text",
								"text":  "underscored",
								"marks": []interface{}{map[string]interface{}{"type": "underline"}},
							},
						},
					},
				},
			},
			want: "__underscored__",
		},
		{
			name: "multiple marks stacked (bold+italic)",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": "both",
								"marks": []interface{}{
									map[string]interface{}{"type": "strong"},
									map[string]interface{}{"type": "em"},
								},
							},
						},
					},
				},
			},
			want: "*" + "**both**" + "*", // inner strong, outer em
		},
		{
			name: "heading levels 1 through 3",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type":  "heading",
						"attrs": map[string]interface{}{"level": float64(1)},
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "H1"},
						},
					},
					map[string]interface{}{
						"type":  "heading",
						"attrs": map[string]interface{}{"level": float64(2)},
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "H2"},
						},
					},
					map[string]interface{}{
						"type":  "heading",
						"attrs": map[string]interface{}{"level": float64(3)},
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "H3"},
						},
					},
				},
			},
			want: "# H1\n\n## H2\n\n### H3",
		},
		{
			name: "heading with no attrs defaults to level 1",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "heading",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "Default"},
						},
					},
				},
			},
			want: "# Default",
		},
		{
			name: "bullet list",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "bulletList",
						"content": []interface{}{
							map[string]interface{}{
								"type": "listItem",
								"content": []interface{}{
									map[string]interface{}{
										"type": "paragraph",
										"content": []interface{}{
											map[string]interface{}{"type": "text", "text": "A"},
										},
									},
								},
							},
							map[string]interface{}{
								"type": "listItem",
								"content": []interface{}{
									map[string]interface{}{
										"type": "paragraph",
										"content": []interface{}{
											map[string]interface{}{"type": "text", "text": "B"},
										},
									},
								},
							},
						},
					},
				},
			},
			want: "- A\n\n- B",
		},
		{
			name: "ordered list",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "orderedList",
						"content": []interface{}{
							map[string]interface{}{
								"type": "listItem",
								"content": []interface{}{
									map[string]interface{}{
										"type": "paragraph",
										"content": []interface{}{
											map[string]interface{}{"type": "text", "text": "First"},
										},
									},
								},
							},
							map[string]interface{}{
								"type": "listItem",
								"content": []interface{}{
									map[string]interface{}{
										"type": "paragraph",
										"content": []interface{}{
											map[string]interface{}{"type": "text", "text": "Second"},
										},
									},
								},
							},
						},
					},
				},
			},
			want: "1. First\n\n2. Second",
		},
		{
			name: "code block with language",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type":  "codeBlock",
						"attrs": map[string]interface{}{"language": "python"},
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "print('hi')"},
						},
					},
				},
			},
			want: "```python\nprint('hi')```",
		},
		{
			name: "code block without language",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "codeBlock",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "raw code"},
						},
					},
				},
			},
			want: "```\nraw code```",
		},
		{
			name: "blockquote",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "blockquote",
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{"type": "text", "text": "Wise words."},
								},
							},
						},
					},
				},
			},
			want: "> Wise words.",
		},
		{
			name: "horizontal rule",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "Above"},
						},
					},
					map[string]interface{}{"type": "rule"},
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "Below"},
						},
					},
				},
			},
			want: "Above\n\n---\n\nBelow",
		},
		{
			name: "mention node",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "Assigned to "},
							map[string]interface{}{
								"type":  "mention",
								"attrs": map[string]interface{}{"id": "u1", "text": "Alice"},
							},
						},
					},
				},
			},
			want: "Assigned to @Alice",
		},
		{
			name: "emoji node",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "Done "},
							map[string]interface{}{
								"type":  "emoji",
								"attrs": map[string]interface{}{"shortName": ":check_mark:"},
							},
						},
					},
				},
			},
			want: "Done :check_mark:",
		},
		{
			name: "inlineCard node",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "See "},
							map[string]interface{}{
								"type":  "inlineCard",
								"attrs": map[string]interface{}{"url": "https://example.com"},
							},
						},
					},
				},
			},
			want: "See https://example.com",
		},
		{
			name: "inline node with nil attrs is ignored",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "before"},
							map[string]interface{}{"type": "mention"},
							map[string]interface{}{"type": "text", "text": "after"},
						},
					},
				},
			},
			want: "beforeafter",
		},
		{
			name: "hardBreak",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "line one"},
							map[string]interface{}{"type": "hardBreak"},
							map[string]interface{}{"type": "text", "text": "line two"},
						},
					},
				},
			},
			want: "line one\nline two",
		},
		{
			name: "media with alt and dimensions",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "mediaSingle",
						"content": []interface{}{
							map[string]interface{}{
								"type": "media",
								"attrs": map[string]interface{}{
									"id":     "img-1",
									"type":   "file",
									"alt":    "Screenshot",
									"width":  float64(800),
									"height": float64(600),
								},
							},
						},
					},
				},
			},
			want: "[Media: Screenshot (800x600) | id=img-1]",
		},
		{
			name: "media with type but no alt",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "media",
						"attrs": map[string]interface{}{
							"id":   "vid-1",
							"type": "external",
						},
					},
				},
			},
			want: "[Media: external | id=vid-1]",
		},
		{
			name: "media with no attrs",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "media",
					},
				},
			},
			want: "[Media/Image]",
		},
		{
			name: "media with no alt no type",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type":  "media",
						"attrs": map[string]interface{}{"id": "m-1"},
					},
				},
			},
			want: "[Media | id=m-1]",
		},
		{
			name: "table rendering",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "table",
						"content": []interface{}{
							map[string]interface{}{
								"type": "tableRow",
								"content": []interface{}{
									map[string]interface{}{
										"type": "tableHeader",
										"content": []interface{}{
											map[string]interface{}{
												"type": "paragraph",
												"content": []interface{}{
													map[string]interface{}{"type": "text", "text": "Col1"},
												},
											},
										},
									},
									map[string]interface{}{
										"type": "tableHeader",
										"content": []interface{}{
											map[string]interface{}{
												"type": "paragraph",
												"content": []interface{}{
													map[string]interface{}{"type": "text", "text": "Col2"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			// Expected: "| Col1\n\n | Col2\n\n | \n\n"  trimmed
			want: "| Col1\n\n | Col2\n\n |",
		},
		{
			name: "unknown node type renders children",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "unknownWidget",
						"content": []interface{}{
							map[string]interface{}{
								"type": "paragraph",
								"content": []interface{}{
									map[string]interface{}{"type": "text", "text": "Fallback content"},
								},
							},
						},
					},
				},
			},
			want: "Fallback content",
		},
		{
			name: "doc with nil node in content array",
			input: map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					nil,
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "After nil"},
						},
					},
				},
			},
			want: "After nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jira.ADFToPlainText(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ADFToPlainText() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestADFToPlainText_Golden — golden file tests for complex ADF documents
// ---------------------------------------------------------------------------

func TestADFToPlainText_Golden(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/adf_*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no ADF test fixtures found in testdata/")
	}

	for _, inputFile := range files {
		name := strings.TrimSuffix(filepath.Base(inputFile), ".json")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			adf := mustReadJSON(t, inputFile)
			got := jira.ADFToPlainText(adf)

			goldenFile := strings.TrimSuffix(inputFile, ".json") + ".golden"
			if *update {
				if err := os.WriteFile(goldenFile, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("golden file not found (run with -update to create): %v", err)
			}
			if diff := cmp.Diff(string(want), got); diff != "" {
				t.Errorf("golden mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestStripJunkFields
// ---------------------------------------------------------------------------

func TestStripJunkFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "empty map",
			data: map[string]interface{}{},
			want: map[string]interface{}{},
		},
		{
			name: "strips top-level junk keys",
			data: map[string]interface{}{
				"key":        "PROJ-1",
				"self":       "https://jira.example.com/rest/api/3/issue/10001",
				"expand":     "renderedFields,names",
				"iconUrl":    "https://jira.example.com/icon.png",
				"avatarUrls": map[string]interface{}{"48x48": "url"},
				"_links":     map[string]interface{}{"web": "url"},
				"icons":      map[string]interface{}{"16x16": "url"},
			},
			want: map[string]interface{}{
				"key": "PROJ-1",
			},
		},
		{
			name: "strips nested junk recursively",
			data: map[string]interface{}{
				"key": "PROJ-2",
				"fields": map[string]interface{}{
					"summary": "Test issue",
					"self":    "https://jira.example.com/rest/api/3/issue/10002",
					"status": map[string]interface{}{
						"name":    "Open",
						"self":    "https://jira.example.com/rest/api/3/status/1",
						"iconUrl": "https://jira.example.com/status.png",
					},
				},
			},
			want: map[string]interface{}{
				"key": "PROJ-2",
				"fields": map[string]interface{}{
					"summary": "Test issue",
					"status": map[string]interface{}{
						"name": "Open",
					},
				},
			},
		},
		{
			name: "strips junk from arrays of objects",
			data: map[string]interface{}{
				"issues": []interface{}{
					map[string]interface{}{
						"key":  "A-1",
						"self": "https://jira.example.com/a1",
						"fields": map[string]interface{}{
							"summary": "Issue A",
							"expand":  "all",
						},
					},
					map[string]interface{}{
						"key":  "A-2",
						"self": "https://jira.example.com/a2",
					},
				},
			},
			want: map[string]interface{}{
				"issues": []interface{}{
					map[string]interface{}{
						"key": "A-1",
						"fields": map[string]interface{}{
							"summary": "Issue A",
						},
					},
					map[string]interface{}{
						"key": "A-2",
					},
				},
			},
		},
		{
			name: "preserves non-junk keys entirely",
			data: map[string]interface{}{
				"key": "PROJ-3",
				"fields": map[string]interface{}{
					"summary":   "Keep everything",
					"labels":    []interface{}{"a", "b"},
					"priority":  map[string]interface{}{"name": "High"},
					"customVal": 42.0,
				},
			},
			want: map[string]interface{}{
				"key": "PROJ-3",
				"fields": map[string]interface{}{
					"summary":   "Keep everything",
					"labels":    []interface{}{"a", "b"},
					"priority":  map[string]interface{}{"name": "High"},
					"customVal": 42.0,
				},
			},
		},
		{
			name: "array with mixed types only strips maps",
			data: map[string]interface{}{
				"items": []interface{}{
					"string-item",
					42,
					map[string]interface{}{
						"name": "keep",
						"self": "strip-me",
					},
				},
			},
			want: map[string]interface{}{
				"items": []interface{}{
					"string-item",
					42,
					map[string]interface{}{
						"name": "keep",
					},
				},
			},
		},
		{
			name: "deeply nested stripping (3 levels)",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"self": "strip",
					"level2": map[string]interface{}{
						"expand": "strip",
						"level3": map[string]interface{}{
							"iconUrl": "strip",
							"value":   "keep",
						},
					},
				},
			},
			want: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"value": "keep",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// StripJunkFields mutates in place, so work on the input directly.
			jira.StripJunkFields(tt.data)
			if diff := cmp.Diff(tt.want, tt.data); diff != "" {
				t.Errorf("StripJunkFields() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHelpers — exported unexported helper functions
// ---------------------------------------------------------------------------

func TestGetStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{name: "existing string key", m: map[string]interface{}{"a": "hello"}, key: "a", want: "hello"},
		{name: "missing key", m: map[string]interface{}{"a": "hello"}, key: "b", want: ""},
		{name: "non-string value", m: map[string]interface{}{"a": 42}, key: "a", want: ""},
		{name: "nil value", m: map[string]interface{}{"a": nil}, key: "a", want: ""},
		{name: "empty string", m: map[string]interface{}{"a": ""}, key: "a", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := jira.GetStr(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("GetStr(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}

func TestGetNestedName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "valid nested name",
			m:    map[string]interface{}{"status": map[string]interface{}{"name": "Open"}},
			key:  "status",
			want: "Open",
		},
		{
			name: "missing key",
			m:    map[string]interface{}{},
			key:  "status",
			want: "",
		},
		{
			name: "key is not a map",
			m:    map[string]interface{}{"status": "string"},
			key:  "status",
			want: "",
		},
		{
			name: "nested map has no name key",
			m:    map[string]interface{}{"status": map[string]interface{}{"id": "1"}},
			key:  "status",
			want: "",
		},
		{
			name: "nested name is not a string",
			m:    map[string]interface{}{"status": map[string]interface{}{"name": 99}},
			key:  "status",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := jira.GetNestedName(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("GetNestedName(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}

func TestGetNestedDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "valid display name",
			m:    map[string]interface{}{"assignee": map[string]interface{}{"displayName": "Alice"}},
			key:  "assignee",
			want: "Alice",
		},
		{
			name: "missing key returns unassigned",
			m:    map[string]interface{}{},
			key:  "assignee",
			want: "unassigned",
		},
		{
			name: "key is not a map returns unassigned",
			m:    map[string]interface{}{"assignee": "string"},
			key:  "assignee",
			want: "unassigned",
		},
		{
			name: "empty display name returns unassigned",
			m:    map[string]interface{}{"assignee": map[string]interface{}{"displayName": ""}},
			key:  "assignee",
			want: "unassigned",
		},
		{
			name: "nil value in map returns unassigned",
			m:    map[string]interface{}{"assignee": nil},
			key:  "assignee",
			want: "unassigned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := jira.GetNestedDisplayName(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("GetNestedDisplayName(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}

func TestApplyMarks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		marks []interface{}
		want  string
	}{
		{
			name:  "no marks returns text unchanged",
			text:  "hello",
			marks: []interface{}{},
			want:  "hello",
		},
		{
			name:  "strong mark",
			text:  "bold",
			marks: []interface{}{map[string]interface{}{"type": "strong"}},
			want:  "**bold**",
		},
		{
			name:  "em mark",
			text:  "italic",
			marks: []interface{}{map[string]interface{}{"type": "em"}},
			want:  "*italic*",
		},
		{
			name:  "code mark",
			text:  "code",
			marks: []interface{}{map[string]interface{}{"type": "code"}},
			want:  "`code`",
		},
		{
			name:  "strike mark",
			text:  "deleted",
			marks: []interface{}{map[string]interface{}{"type": "strike"}},
			want:  "~~deleted~~",
		},
		{
			name:  "underline mark",
			text:  "underlined",
			marks: []interface{}{map[string]interface{}{"type": "underline"}},
			want:  "__underlined__",
		},
		{
			name: "multiple marks applied in order",
			text: "text",
			marks: []interface{}{
				map[string]interface{}{"type": "strong"},
				map[string]interface{}{"type": "em"},
			},
			want: "*" + "**text**" + "*",
		},
		{
			name: "unknown mark type is ignored",
			text: "text",
			marks: []interface{}{
				map[string]interface{}{"type": "unknown_mark"},
			},
			want: "text",
		},
		{
			name: "non-map mark in array is ignored",
			text: "text",
			marks: []interface{}{
				"not-a-map",
				map[string]interface{}{"type": "strong"},
			},
			want: "**text**",
		},
		{
			name: "mark with no type key is ignored",
			text: "text",
			marks: []interface{}{
				map[string]interface{}{"color": "red"},
			},
			want: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := jira.ApplyMarks(tt.text, tt.marks)
			if got != tt.want {
				t.Errorf("ApplyMarks(%q, marks) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSafeJSON
// ---------------------------------------------------------------------------

func TestSafeJSON(t *testing.T) {
	t.Parallel()

	t.Run("small data returns full JSON", func(t *testing.T) {
		t.Parallel()
		data := map[string]string{"key": "value"}
		got := jira.SafeJSON(data, 10000)
		var parsed map[string]string
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("SafeJSON returned invalid JSON: %v", err)
		}
		if diff := cmp.Diff(data, parsed); diff != "" {
			t.Errorf("SafeJSON mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("large data is truncated with guidance", func(t *testing.T) {
		t.Parallel()
		// Build data larger than 100 bytes
		data := map[string]string{}
		for i := range 50 {
			data[strings.Repeat("k", 10)+string(rune('A'+i%26))] = strings.Repeat("v", 20)
		}
		got := jira.SafeJSON(data, 100)
		if !strings.Contains(got, "Response Truncated") {
			t.Error("expected truncation guidance, got none")
		}
		if !strings.Contains(got, "Full response saved to:") {
			t.Error("expected file path in guidance")
		}
	})

	t.Run("default maxChars when zero", func(t *testing.T) {
		t.Parallel()
		data := map[string]string{"a": "b"}
		got := jira.SafeJSON(data, 0) // should use 40000 default
		if strings.Contains(got, "Response Truncated") {
			t.Error("small data should not be truncated with default maxChars")
		}
	})

	t.Run("unmarshalable data returns error string", func(t *testing.T) {
		t.Parallel()
		// Channels can't be marshaled to JSON
		got := jira.SafeJSON(make(chan int), 100)
		if !strings.Contains(got, "error marshaling") {
			t.Errorf("expected error message, got: %s", got)
		}
	})
}

// ---------------------------------------------------------------------------
// BenchmarkFlattenIssue
// ---------------------------------------------------------------------------

func BenchmarkFlattenIssue(b *testing.B) {
	raw := map[string]interface{}{
		"key": "BENCH-1",
		"fields": map[string]interface{}{
			"summary":   "Benchmark issue",
			"status":    map[string]interface{}{"name": "Open"},
			"issuetype": map[string]interface{}{"name": "Task"},
			"priority":  map[string]interface{}{"name": "Medium"},
			"assignee":  map[string]interface{}{"displayName": "Tester"},
			"reporter":  map[string]interface{}{"displayName": "Reporter"},
			"created":   "2024-01-01",
			"updated":   "2024-01-02",
			"labels":    []interface{}{"a", "b", "c"},
			"components": []interface{}{
				map[string]interface{}{"name": "Core"},
			},
			"description": map[string]interface{}{
				"type": "doc",
				"content": []interface{}{
					map[string]interface{}{
						"type": "paragraph",
						"content": []interface{}{
							map[string]interface{}{"type": "text", "text": "Some description text."},
						},
					},
				},
			},
		},
	}

	for b.Loop() {
		jira.FlattenIssue(raw)
	}
}

func BenchmarkADFToPlainText(b *testing.B) {
	cases := []struct {
		name string
		file string
	}{
		{"simple_paragraph", "testdata/adf_simple_paragraph.json"},
		{"rich_document", "testdata/adf_rich_document.json"},
		{"deeply_nested", "testdata/adf_deeply_nested.json"},
	}

	for _, bc := range cases {
		data, err := os.ReadFile(bc.file)
		if err != nil {
			b.Fatalf("failed to read %s: %v", bc.file, err)
		}
		var adf map[string]interface{}
		if err := json.Unmarshal(data, &adf); err != nil {
			b.Fatalf("failed to unmarshal %s: %v", bc.file, err)
		}

		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				jira.ADFToPlainText(adf)
			}
		})
	}
}

func BenchmarkStripJunkFields(b *testing.B) {
	// Build template once, copy for each iteration
	template := map[string]interface{}{
		"key":    "BENCH-1",
		"self":   "https://jira.example.com/rest/api/3/issue/10001",
		"expand": "renderedFields,names",
		"fields": map[string]interface{}{
			"summary":    "Benchmark",
			"self":       "https://jira.example.com/self",
			"avatarUrls": map[string]interface{}{"48x48": "url"},
			"status": map[string]interface{}{
				"name":    "Open",
				"self":    "strip-me",
				"iconUrl": "strip-me",
			},
		},
	}

	for b.Loop() {
		// Deep copy via JSON round-trip for fair benchmark
		raw, _ := json.Marshal(template)
		var data map[string]interface{}
		_ = json.Unmarshal(raw, &data)
		jira.StripJunkFields(data)
	}
}
