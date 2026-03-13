package jira

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// intPtr is a test helper that returns a pointer to an int.
func intPtr(v int) *int {
	return &v
}

// ts is a test helper that builds a Jira-style timestamp string from a time.Time.
func ts(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.000-0700")
}

func TestParseJiraTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "standard Jira format with millis and numeric tz",
			input: "2024-01-15T09:30:00.000+0000",
			want:  time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
		},
		{
			name:  "Jira format with negative timezone offset",
			input: "2024-07-04T14:00:00.000-0500",
			want:  time.Date(2024, 7, 4, 14, 0, 0, 0, time.FixedZone("", -5*3600)),
		},
		{
			name:  "Jira format with colon in tz offset",
			input: "2024-01-15T09:30:00.000+00:00",
			want:  time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
		},
		{
			name:  "Z suffix (UTC)",
			input: "2024-01-15T09:30:00.000Z",
			want:  time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
		},
		{
			name:  "no milliseconds with numeric tz",
			input: "2024-01-15T09:30:00+0000",
			want:  time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
		},
		{
			name:  "no milliseconds with Z",
			input: "2024-01-15T09:30:00Z",
			want:  time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
		},
		{
			name:  "RFC3339 format",
			input: "2024-01-15T09:30:00+00:00",
			want:  time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "garbage input",
			input:   "not-a-timestamp",
			wantErr: true,
		},
		{
			name:    "date only (no time component)",
			input:   "2024-01-15",
			wantErr: true,
		},
		{
			name:    "unix epoch number",
			input:   "1705312200",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseJiraTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseJiraTimestamp(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseJiraTimestamp(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		minutes int
		want    string
	}{
		{name: "zero", minutes: 0, want: "0m"},
		{name: "negative", minutes: -5, want: "0m"},
		{name: "one minute", minutes: 1, want: "1m"},
		{name: "59 minutes", minutes: 59, want: "59m"},
		{name: "exactly one hour", minutes: 60, want: "1h 0m"},
		{name: "one hour one minute", minutes: 61, want: "1h 1m"},
		{name: "90 minutes", minutes: 90, want: "1h 30m"},
		{name: "exactly one day", minutes: 24 * 60, want: "1d 0h 0m"},
		{name: "one day one minute", minutes: 24*60 + 1, want: "1d 0h 1m"},
		{name: "one day one hour one minute", minutes: 24*60 + 61, want: "1d 1h 1m"},
		{name: "two days 12 hours", minutes: 2*24*60 + 12*60, want: "2d 12h 0m"},
		{name: "large value 30 days", minutes: 30 * 24 * 60, want: "30d 0h 0m"},
		{name: "23 hours 59 minutes", minutes: 23*60 + 59, want: "23h 59m"},
		{name: "exactly 2 hours", minutes: 120, want: "2h 0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatDuration(tt.minutes)
			if got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.minutes, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "already lowercase", input: "in progress", want: "in progress"},
		{name: "all uppercase", input: "IN PROGRESS", want: "in progress"},
		{name: "mixed case", input: "In Progress", want: "in progress"},
		{name: "numbers and symbols", input: "Status-123!", want: "status-123!"},
		{name: "single char upper", input: "A", want: "a"},
		{name: "single char lower", input: "z", want: "z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := toLower(tt.input)
			if got != tt.want {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseStatusTransitions(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	createdDate := ts(baseTime)

	tests := []struct {
		name        string
		changelogs  []Changelog
		createdDate string
		want        []StatusTransition
	}{
		{
			name:        "empty changelog",
			changelogs:  nil,
			createdDate: createdDate,
			want:        nil,
		},
		{
			name:        "empty changelog with empty created date",
			changelogs:  []Changelog{},
			createdDate: "",
			want:        nil,
		},
		{
			name: "single status transition with created date",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(30 * time.Minute)),
					Author:  &User{DisplayName: "Alice"},
					Items: []ChangelogItem{
						{Field: "status", FromString: "To Do", ToString: "In Progress"},
					},
				},
			},
			createdDate: createdDate,
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "In Progress",
					TransitionedAt:  ts(baseTime.Add(30 * time.Minute)),
					TransitionedBy:  "Alice",
					DurationMinutes: intPtr(30),
					DurationDisplay: "30m",
				},
			},
		},
		{
			name: "single transition without created date",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(30 * time.Minute)),
					Items: []ChangelogItem{
						{Field: "status", FromString: "To Do", ToString: "In Progress"},
					},
				},
			},
			createdDate: "",
			want: []StatusTransition{
				{
					FromStatus:     "To Do",
					ToStatus:       "In Progress",
					TransitionedAt: ts(baseTime.Add(30 * time.Minute)),
				},
			},
		},
		{
			name: "multiple transitions with duration calculations",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(60 * time.Minute)),
					Author:  &User{DisplayName: "Alice"},
					Items: []ChangelogItem{
						{Field: "status", FromString: "To Do", ToString: "In Progress"},
					},
				},
				{
					ID:      "2",
					Created: ts(baseTime.Add(180 * time.Minute)), // 2 hours later
					Author:  &User{DisplayName: "Bob"},
					Items: []ChangelogItem{
						{Field: "status", FromString: "In Progress", ToString: "Done"},
					},
				},
			},
			createdDate: createdDate,
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "In Progress",
					TransitionedAt:  ts(baseTime.Add(60 * time.Minute)),
					TransitionedBy:  "Alice",
					DurationMinutes: intPtr(60),
					DurationDisplay: "1h 0m",
				},
				{
					FromStatus:      "In Progress",
					ToStatus:        "Done",
					TransitionedAt:  ts(baseTime.Add(180 * time.Minute)),
					TransitionedBy:  "Bob",
					DurationMinutes: intPtr(120),
					DurationDisplay: "2h 0m",
				},
			},
		},
		{
			name: "non-status changelog items are ignored",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(30 * time.Minute)),
					Items: []ChangelogItem{
						{Field: "assignee", FromString: "Alice", ToString: "Bob"},
						{Field: "priority", FromString: "Low", ToString: "High"},
					},
				},
			},
			createdDate: createdDate,
			want:        nil,
		},
		{
			name: "mixed status and non-status items in same changelog",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(30 * time.Minute)),
					Author:  &User{DisplayName: "Alice"},
					Items: []ChangelogItem{
						{Field: "assignee", FromString: "Alice", ToString: "Bob"},
						{Field: "status", FromString: "To Do", ToString: "In Progress"},
						{Field: "priority", FromString: "Low", ToString: "High"},
					},
				},
			},
			createdDate: createdDate,
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "In Progress",
					TransitionedAt:  ts(baseTime.Add(30 * time.Minute)),
					TransitionedBy:  "Alice",
					DurationMinutes: intPtr(30),
					DurationDisplay: "30m",
				},
			},
		},
		{
			name: "changelogs arrive out of order — sorted by time",
			changelogs: []Changelog{
				{
					ID:      "2",
					Created: ts(baseTime.Add(120 * time.Minute)),
					Author:  &User{DisplayName: "Bob"},
					Items:   []ChangelogItem{{Field: "status", FromString: "In Progress", ToString: "Done"}},
				},
				{
					ID:      "1",
					Created: ts(baseTime.Add(60 * time.Minute)),
					Author:  &User{DisplayName: "Alice"},
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "In Progress"}},
				},
			},
			createdDate: createdDate,
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "In Progress",
					TransitionedAt:  ts(baseTime.Add(60 * time.Minute)),
					TransitionedBy:  "Alice",
					DurationMinutes: intPtr(60),
					DurationDisplay: "1h 0m",
				},
				{
					FromStatus:      "In Progress",
					ToStatus:        "Done",
					TransitionedAt:  ts(baseTime.Add(120 * time.Minute)),
					TransitionedBy:  "Bob",
					DurationMinutes: intPtr(60),
					DurationDisplay: "1h 0m",
				},
			},
		},
		{
			name: "zero-duration transition (instant transition)",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime), // same as created
					Author:  &User{DisplayName: "Alice"},
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "In Progress"}},
				},
			},
			createdDate: createdDate,
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "In Progress",
					TransitionedAt:  ts(baseTime),
					TransitionedBy:  "Alice",
					DurationMinutes: intPtr(0),
					DurationDisplay: "0m",
				},
			},
		},
		{
			name: "rapid transitions within same minute",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(10 * time.Second)),
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "In Progress"}},
				},
				{
					ID:      "2",
					Created: ts(baseTime.Add(20 * time.Second)),
					Items:   []ChangelogItem{{Field: "status", FromString: "In Progress", ToString: "In Review"}},
				},
				{
					ID:      "3",
					Created: ts(baseTime.Add(30 * time.Second)),
					Items:   []ChangelogItem{{Field: "status", FromString: "In Review", ToString: "Done"}},
				},
			},
			createdDate: createdDate,
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "In Progress",
					TransitionedAt:  ts(baseTime.Add(10 * time.Second)),
					DurationMinutes: intPtr(0), // 10 seconds < 1 minute
					DurationDisplay: "0m",
				},
				{
					FromStatus:      "In Progress",
					ToStatus:        "In Review",
					TransitionedAt:  ts(baseTime.Add(20 * time.Second)),
					DurationMinutes: intPtr(0),
					DurationDisplay: "0m",
				},
				{
					FromStatus:      "In Review",
					ToStatus:        "Done",
					TransitionedAt:  ts(baseTime.Add(30 * time.Second)),
					DurationMinutes: intPtr(0),
					DurationDisplay: "0m",
				},
			},
		},
		{
			name: "transition spanning midnight",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(time.Date(2024, 1, 15, 23, 30, 0, 0, time.UTC)),
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "In Progress"}},
				},
				{
					ID:      "2",
					Created: ts(time.Date(2024, 1, 16, 0, 30, 0, 0, time.UTC)),
					Items:   []ChangelogItem{{Field: "status", FromString: "In Progress", ToString: "Done"}},
				},
			},
			createdDate: ts(time.Date(2024, 1, 15, 22, 0, 0, 0, time.UTC)),
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "In Progress",
					TransitionedAt:  ts(time.Date(2024, 1, 15, 23, 30, 0, 0, time.UTC)),
					DurationMinutes: intPtr(90), // 22:00 → 23:30 = 90 min
					DurationDisplay: "1h 30m",
				},
				{
					FromStatus:      "In Progress",
					ToStatus:        "Done",
					TransitionedAt:  ts(time.Date(2024, 1, 16, 0, 30, 0, 0, time.UTC)),
					DurationMinutes: intPtr(60), // 23:30 → 00:30 = 60 min
					DurationDisplay: "1h 0m",
				},
			},
		},
		{
			name: "multi-day transition",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(3 * 24 * time.Hour)), // 3 days later
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "Done"}},
				},
			},
			createdDate: createdDate,
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "Done",
					TransitionedAt:  ts(baseTime.Add(3 * 24 * time.Hour)),
					DurationMinutes: intPtr(3 * 24 * 60),
					DurationDisplay: "3d 0h 0m",
				},
			},
		},
		{
			name: "changelog with nil author",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(60 * time.Minute)),
					Author:  nil,
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "In Progress"}},
				},
			},
			createdDate: createdDate,
			want: []StatusTransition{
				{
					FromStatus:      "To Do",
					ToStatus:        "In Progress",
					TransitionedAt:  ts(baseTime.Add(60 * time.Minute)),
					TransitionedBy:  "",
					DurationMinutes: intPtr(60),
					DurationDisplay: "1h 0m",
				},
			},
		},
		{
			name: "changelog with unparseable timestamp is skipped",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: "not-a-timestamp",
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "In Progress"}},
				},
			},
			createdDate: createdDate,
			want:        nil,
		},
		{
			name: "unparseable created date — first transition is silently dropped",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(60 * time.Minute)),
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "In Progress"}},
				},
			},
			createdDate: "bad-date",
			// When createdDate is non-empty but unparseable, the first transition
			// is skipped entirely (the else-if branch only runs when createdDate == "").
			want: nil,
		},
		{
			name: "unparseable created date with two transitions — second survives",
			changelogs: []Changelog{
				{
					ID:      "1",
					Created: ts(baseTime.Add(60 * time.Minute)),
					Items:   []ChangelogItem{{Field: "status", FromString: "To Do", ToString: "In Progress"}},
				},
				{
					ID:      "2",
					Created: ts(baseTime.Add(120 * time.Minute)),
					Items:   []ChangelogItem{{Field: "status", FromString: "In Progress", ToString: "Done"}},
				},
			},
			createdDate: "bad-date",
			// First transition is dropped, but the loop starting at i=1 still runs.
			want: []StatusTransition{
				{
					FromStatus:      "In Progress",
					ToStatus:        "Done",
					TransitionedAt:  ts(baseTime.Add(120 * time.Minute)),
					DurationMinutes: intPtr(60),
					DurationDisplay: "1h 0m",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseStatusTransitions(tt.changelogs, tt.createdDate)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("parseStatusTransitions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAggregateStatusTimes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		transitions []StatusTransition
		want        []StatusTimeSummary
	}{
		{
			name:        "empty transitions",
			transitions: nil,
			want:        nil,
		},
		{
			name: "single transition",
			transitions: []StatusTransition{
				{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: intPtr(60)},
			},
			want: []StatusTimeSummary{
				{Status: "To Do", TotalDurationMinutes: 60, TotalDurationDisplay: "1h 0m", VisitCount: 1},
			},
		},
		{
			name: "transition without duration is not counted",
			transitions: []StatusTransition{
				{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: nil},
			},
			want: nil,
		},
		{
			name: "multiple statuses — sorted by total duration descending",
			transitions: []StatusTransition{
				{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: intPtr(30)},
				{FromStatus: "In Progress", ToStatus: "In Review", DurationMinutes: intPtr(120)},
				{FromStatus: "In Review", ToStatus: "Done", DurationMinutes: intPtr(45)},
			},
			want: []StatusTimeSummary{
				{Status: "In Progress", TotalDurationMinutes: 120, TotalDurationDisplay: "2h 0m", VisitCount: 1},
				{Status: "In Review", TotalDurationMinutes: 45, TotalDurationDisplay: "45m", VisitCount: 1},
				{Status: "To Do", TotalDurationMinutes: 30, TotalDurationDisplay: "30m", VisitCount: 1},
			},
		},
		{
			name: "repeated visits to same status accumulate",
			transitions: []StatusTransition{
				{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: intPtr(30)},
				{FromStatus: "In Progress", ToStatus: "To Do", DurationMinutes: intPtr(60)},
				{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: intPtr(15)},
				{FromStatus: "In Progress", ToStatus: "Done", DurationMinutes: intPtr(90)},
			},
			want: []StatusTimeSummary{
				{Status: "In Progress", TotalDurationMinutes: 150, TotalDurationDisplay: "2h 30m", VisitCount: 2},
				{Status: "To Do", TotalDurationMinutes: 45, TotalDurationDisplay: "45m", VisitCount: 2},
			},
		},
		{
			name: "zero-duration transitions still count as visits",
			transitions: []StatusTransition{
				{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: intPtr(0)},
				{FromStatus: "In Progress", ToStatus: "Done", DurationMinutes: intPtr(0)},
			},
			want: []StatusTimeSummary{
				{Status: "In Progress", TotalDurationMinutes: 0, TotalDurationDisplay: "0m", VisitCount: 1},
				{Status: "To Do", TotalDurationMinutes: 0, TotalDurationDisplay: "0m", VisitCount: 1},
			},
		},
		{
			name: "mix of nil and non-nil durations",
			transitions: []StatusTransition{
				{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: nil},
				{FromStatus: "In Progress", ToStatus: "Done", DurationMinutes: intPtr(120)},
			},
			want: []StatusTimeSummary{
				{Status: "In Progress", TotalDurationMinutes: 120, TotalDurationDisplay: "2h 0m", VisitCount: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := aggregateStatusTimes(tt.transitions)
			// Use SortSlices for deterministic comparison when durations are equal
			if diff := cmp.Diff(tt.want, got,
				cmpopts.SortSlices(func(a, b StatusTimeSummary) bool {
					if a.TotalDurationMinutes != b.TotalDurationMinutes {
						return a.TotalDurationMinutes > b.TotalDurationMinutes
					}
					return a.Status < b.Status
				}),
				cmpopts.EquateEmpty(),
			); diff != "" {
				t.Errorf("aggregateStatusTimes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindFirstInProgressTime(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		transitions []StatusTransition
		wantZero    bool
		want        time.Time
	}{
		{
			name:        "empty transitions",
			transitions: nil,
			wantZero:    true,
		},
		{
			name: "no in-progress status in transitions",
			transitions: []StatusTransition{
				{ToStatus: "To Do", TransitionedAt: ts(baseTime)},
				{ToStatus: "Done", TransitionedAt: ts(baseTime.Add(time.Hour))},
			},
			wantZero: true,
		},
		{
			name: "exact match: In Progress",
			transitions: []StatusTransition{
				{ToStatus: "In Progress", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "case insensitive: IN PROGRESS",
			transitions: []StatusTransition{
				{ToStatus: "IN PROGRESS", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "mixed case: In progress",
			transitions: []StatusTransition{
				{ToStatus: "In progress", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "alternative in-progress statuses",
			transitions: []StatusTransition{
				{ToStatus: "In Review", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "In Testing status",
			transitions: []StatusTransition{
				{ToStatus: "In Testing", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "In QA status",
			transitions: []StatusTransition{
				{ToStatus: "In QA", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "Development status",
			transitions: []StatusTransition{
				{ToStatus: "Development", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "Coding status",
			transitions: []StatusTransition{
				{ToStatus: "Coding", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "Implementing status",
			transitions: []StatusTransition{
				{ToStatus: "Implementing", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "In Dev status",
			transitions: []StatusTransition{
				{ToStatus: "In Dev", TransitionedAt: ts(baseTime)},
			},
			want: baseTime,
		},
		{
			name: "returns first match — earlier in-progress wins",
			transitions: []StatusTransition{
				{ToStatus: "To Do", TransitionedAt: ts(baseTime)},
				{ToStatus: "In Progress", TransitionedAt: ts(baseTime.Add(time.Hour))},
				{ToStatus: "To Do", TransitionedAt: ts(baseTime.Add(2 * time.Hour))},
				{ToStatus: "In Progress", TransitionedAt: ts(baseTime.Add(3 * time.Hour))},
			},
			want: baseTime.Add(time.Hour),
		},
		{
			name: "in-progress status with unparseable timestamp returns zero",
			transitions: []StatusTransition{
				{ToStatus: "In Progress", TransitionedAt: "bad-timestamp"},
			},
			wantZero: true,
		},
		{
			name: "unrecognized status is not treated as in-progress",
			transitions: []StatusTransition{
				{ToStatus: "Blocked", TransitionedAt: ts(baseTime)},
				{ToStatus: "Waiting", TransitionedAt: ts(baseTime.Add(time.Hour))},
			},
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := findFirstInProgressTime(tt.transitions)
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("findFirstInProgressTime() = %v, want zero time", got)
				}
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("findFirstInProgressTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindCurrentStatusEntryTime(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		transitions   []StatusTransition
		currentStatus string
		createdDate   string
		wantZero      bool
		want          time.Time
	}{
		{
			name:          "empty current status returns zero",
			transitions:   nil,
			currentStatus: "",
			createdDate:   ts(baseTime),
			wantZero:      true,
		},
		{
			name:          "no transitions — falls back to created date",
			transitions:   nil,
			currentStatus: "To Do",
			createdDate:   ts(baseTime),
			want:          baseTime,
		},
		{
			name:          "no transitions no created date — returns zero",
			transitions:   nil,
			currentStatus: "To Do",
			createdDate:   "",
			wantZero:      true,
		},
		{
			name: "finds last transition to current status",
			transitions: []StatusTransition{
				{ToStatus: "In Progress", TransitionedAt: ts(baseTime.Add(time.Hour))},
				{ToStatus: "To Do", TransitionedAt: ts(baseTime.Add(2 * time.Hour))},
				{ToStatus: "In Progress", TransitionedAt: ts(baseTime.Add(3 * time.Hour))},
			},
			currentStatus: "In Progress",
			createdDate:   ts(baseTime),
			want:          baseTime.Add(3 * time.Hour),
		},
		{
			name: "current status matches only the first transition",
			transitions: []StatusTransition{
				{ToStatus: "In Progress", TransitionedAt: ts(baseTime.Add(time.Hour))},
				{ToStatus: "Done", TransitionedAt: ts(baseTime.Add(2 * time.Hour))},
			},
			currentStatus: "In Progress",
			createdDate:   ts(baseTime),
			want:          baseTime.Add(time.Hour),
		},
		{
			name: "current status not found in transitions — falls back to created date",
			transitions: []StatusTransition{
				{ToStatus: "In Progress", TransitionedAt: ts(baseTime.Add(time.Hour))},
				{ToStatus: "Done", TransitionedAt: ts(baseTime.Add(2 * time.Hour))},
			},
			currentStatus: "To Do",
			createdDate:   ts(baseTime),
			want:          baseTime,
		},
		{
			name: "current status transition has unparseable timestamp — falls back to created",
			transitions: []StatusTransition{
				{ToStatus: "Done", TransitionedAt: "bad-timestamp"},
			},
			currentStatus: "Done",
			createdDate:   ts(baseTime),
			want:          baseTime,
		},
		{
			name: "current status transition unparseable + bad created date — returns zero",
			transitions: []StatusTransition{
				{ToStatus: "Done", TransitionedAt: "bad-timestamp"},
			},
			currentStatus: "Done",
			createdDate:   "also-bad",
			wantZero:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := findCurrentStatusEntryTime(tt.transitions, tt.currentStatus, tt.createdDate)
			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("findCurrentStatusEntryTime() = %v, want zero time", got)
				}
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("findCurrentStatusEntryTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseStatusTransitions_MultipleStatusChangesInSameChangelog(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	createdDate := ts(baseTime)

	// A single changelog entry can contain multiple status changes (rare but possible)
	changelogs := []Changelog{
		{
			ID:      "1",
			Created: ts(baseTime.Add(60 * time.Minute)),
			Author:  &User{DisplayName: "Alice"},
			Items: []ChangelogItem{
				{Field: "status", FromString: "To Do", ToString: "In Progress"},
				{Field: "status", FromString: "In Progress", ToString: "Done"},
			},
		},
	}

	got := parseStatusTransitions(changelogs, createdDate)

	// Both status items share the same timestamp, so the first becomes
	// the "initial" transition and the second gets duration 0 (same timestamp).
	if len(got) != 2 {
		t.Fatalf("expected 2 transitions, got %d: %+v", len(got), got)
	}

	// Both transitions should be present and have duration from creation to changelog time
	// for the first, and 0 for the second (same time).
	if got[0].DurationMinutes == nil {
		t.Fatal("first transition should have duration")
	}
	if *got[0].DurationMinutes != 60 {
		t.Errorf("first transition duration = %d, want 60", *got[0].DurationMinutes)
	}
	if got[1].DurationMinutes == nil {
		t.Fatal("second transition should have duration")
	}
	if *got[1].DurationMinutes != 0 {
		t.Errorf("second transition duration = %d, want 0", *got[1].DurationMinutes)
	}
}

func TestAggregateStatusTimes_ManyVisitsToSameStatus(t *testing.T) {
	t.Parallel()

	// Simulate a ticket bouncing back and forth multiple times
	transitions := []StatusTransition{
		{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: intPtr(10)},
		{FromStatus: "In Progress", ToStatus: "To Do", DurationMinutes: intPtr(20)},
		{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: intPtr(5)},
		{FromStatus: "In Progress", ToStatus: "To Do", DurationMinutes: intPtr(30)},
		{FromStatus: "To Do", ToStatus: "In Progress", DurationMinutes: intPtr(15)},
		{FromStatus: "In Progress", ToStatus: "Done", DurationMinutes: intPtr(60)},
	}

	got := aggregateStatusTimes(transitions)

	wantByStatus := map[string]StatusTimeSummary{
		"In Progress": {
			Status:               "In Progress",
			TotalDurationMinutes: 110, // 20 + 30 + 60
			TotalDurationDisplay: "1h 50m",
			VisitCount:           3,
		},
		"To Do": {
			Status:               "To Do",
			TotalDurationMinutes: 30, // 10 + 5 + 15
			TotalDurationDisplay: "30m",
			VisitCount:           3,
		},
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 status summaries, got %d: %+v", len(got), got)
	}

	for _, s := range got {
		want, ok := wantByStatus[s.Status]
		if !ok {
			t.Errorf("unexpected status in summary: %q", s.Status)
			continue
		}
		if diff := cmp.Diff(want, s); diff != "" {
			t.Errorf("summary for %q mismatch (-want +got):\n%s", s.Status, diff)
		}
	}
}

func TestFormatDuration_BoundaryValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		minutes int
		want    string
	}{
		{name: "one minute before one day", minutes: 24*60 - 1, want: "23h 59m"},
		{name: "exactly one week", minutes: 7 * 24 * 60, want: "7d 0h 0m"},
		{name: "huge value 365 days", minutes: 365 * 24 * 60, want: "365d 0h 0m"},
		{name: "large negative", minutes: -1000, want: "0m"},
		{name: "max int minutes", minutes: 1<<31 - 1, want: formatDuration(1<<31 - 1)}, // just ensure no panic
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatDuration(tt.minutes)
			if got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.minutes, got, tt.want)
			}
		})
	}
}

func TestParseStatusTransitions_LongChain(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	createdDate := ts(baseTime)

	// Build a chain: To Do → In Progress → In Review → QA → Done
	// with each step taking exactly 2 hours
	statuses := []struct{ from, to string }{
		{"To Do", "In Progress"},
		{"In Progress", "In Review"},
		{"In Review", "QA"},
		{"QA", "Done"},
	}
	var changelogs []Changelog
	for i, s := range statuses {
		changelogs = append(changelogs, Changelog{
			ID:      string(rune('1' + i)),
			Created: ts(baseTime.Add(time.Duration(i+1) * 2 * time.Hour)),
			Author:  &User{DisplayName: "Dev"},
			Items:   []ChangelogItem{{Field: "status", FromString: s.from, ToString: s.to}},
		})
	}

	got := parseStatusTransitions(changelogs, createdDate)

	if len(got) != 4 {
		t.Fatalf("expected 4 transitions, got %d", len(got))
	}

	// First transition: created → first change = 2 hours = 120 min
	if *got[0].DurationMinutes != 120 {
		t.Errorf("transition 0 duration = %d, want 120", *got[0].DurationMinutes)
	}
	// Each subsequent transition = 2 hours = 120 min
	for i := 1; i < len(got); i++ {
		if got[i].DurationMinutes == nil {
			t.Fatalf("transition %d has nil duration", i)
		}
		if *got[i].DurationMinutes != 120 {
			t.Errorf("transition %d duration = %d, want 120", i, *got[i].DurationMinutes)
		}
	}
}

func BenchmarkFormatDuration(b *testing.B) {
	cases := []struct {
		name    string
		minutes int
	}{
		{"zero", 0},
		{"minutes_only", 45},
		{"hours_and_minutes", 150},
		{"days_hours_minutes", 3*24*60 + 5*60 + 30},
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				formatDuration(bc.minutes)
			}
		})
	}
}

func BenchmarkParseJiraTimestamp(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"millis_numeric_tz", "2024-01-15T09:30:00.000+0000"},
		{"millis_Z", "2024-01-15T09:30:00.000Z"},
		{"rfc3339", "2024-01-15T09:30:00+00:00"},
		{"last_format", "2024-01-15T09:30:00+00:00"}, // hits RFC3339 (last in list)
	}

	for _, bc := range cases {
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				_, _ = parseJiraTimestamp(bc.input)
			}
		})
	}
}

func BenchmarkParseStatusTransitions(b *testing.B) {
	baseTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	createdDate := ts(baseTime)

	// Build 50 transitions
	changelogs := make([]Changelog, 50)
	statuses := []string{"To Do", "In Progress", "In Review", "Done"}
	for i := range changelogs {
		changelogs[i] = Changelog{
			ID:      fmt.Sprintf("%d", i),
			Created: ts(baseTime.Add(time.Duration(i+1) * time.Hour)),
			Author:  &User{DisplayName: "Dev"},
			Items: []ChangelogItem{{
				Field:      "status",
				FromString: statuses[i%len(statuses)],
				ToString:   statuses[(i+1)%len(statuses)],
			}},
		}
	}

	b.ResetTimer()
	for b.Loop() {
		parseStatusTransitions(changelogs, createdDate)
	}
}
