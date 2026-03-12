package jira

import (
	"fmt"
	"sort"
	"time"
)

// StatusTransition represents a single status change on an issue.
type StatusTransition struct {
	FromStatus      string `json:"from_status"`
	ToStatus        string `json:"to_status"`
	TransitionedAt  string `json:"transitioned_at"`
	TransitionedBy  string `json:"transitioned_by,omitempty"`
	DurationMinutes *int   `json:"duration_minutes,omitempty"` // time spent in from_status
	DurationDisplay string `json:"duration_display,omitempty"`
}

// StatusTimeSummary aggregates time spent in a specific status across all visits.
type StatusTimeSummary struct {
	Status               string `json:"status"`
	TotalDurationMinutes int    `json:"total_duration_minutes"`
	TotalDurationDisplay string `json:"total_duration_display"`
	VisitCount           int    `json:"visit_count"`
}

// IssueDates contains raw date information for an issue.
type IssueDates struct {
	IssueKey       string `json:"issue_key"`
	Created        string `json:"created,omitempty"`
	Updated        string `json:"updated,omitempty"`
	DueDate        string `json:"due_date,omitempty"`
	ResolutionDate string `json:"resolution_date,omitempty"`
	CurrentStatus  string `json:"current_status,omitempty"`

	StatusTransitions []StatusTransition  `json:"status_transitions,omitempty"`
	StatusSummary     []StatusTimeSummary `json:"status_summary,omitempty"`
}

// IssueMetrics contains computed metrics for an issue (cycle time, lead time, etc.).
type IssueMetrics struct {
	IssueKey string `json:"issue_key"`

	// Lead time: Created → Done (or current status if not done)
	LeadTimeMinutes *int   `json:"lead_time_minutes,omitempty"`
	LeadTimeDisplay string `json:"lead_time_display,omitempty"`

	// Cycle time: first transition to "In Progress" (or similar) → Done
	CycleTimeMinutes *int   `json:"cycle_time_minutes,omitempty"`
	CycleTimeDisplay string `json:"cycle_time_display,omitempty"`

	// Time in current status
	CurrentStatus        string `json:"current_status,omitempty"`
	TimeInCurrentMinutes *int   `json:"time_in_current_minutes,omitempty"`
	TimeInCurrentDisplay string `json:"time_in_current_display,omitempty"`

	// Status breakdown
	StatusSummary []StatusTimeSummary `json:"status_summary,omitempty"`
}

// GetIssueDates fetches an issue and its changelog, returning raw date info
// with status transition history and time-in-status aggregation.
func (c *Client) GetIssueDates(issueKey string) (*IssueDates, error) {
	// Get issue fields
	issue, err := c.GetIssue(issueKey)
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}

	dates := &IssueDates{
		IssueKey: issue.Key,
		Created:  issue.Fields.Created,
		Updated:  issue.Fields.Updated,
		DueDate:  issue.Fields.DueDate,
	}

	if issue.Fields.Resolution != nil {
		dates.ResolutionDate = issue.Fields.ResolutionDate
	}
	if issue.Fields.Status != nil {
		dates.CurrentStatus = issue.Fields.Status.Name
	}

	// Fetch all changelog pages
	changelogs, err := c.fetchAllChangelogs(issueKey)
	if err != nil {
		return nil, fmt.Errorf("getting changelog: %w", err)
	}

	// Parse status transitions
	dates.StatusTransitions = parseStatusTransitions(changelogs, issue.Fields.Created)
	dates.StatusSummary = aggregateStatusTimes(dates.StatusTransitions)

	return dates, nil
}

// GetIssueMetrics computes cycle time, lead time, and time-in-status for an issue.
func (c *Client) GetIssueMetrics(issueKey string) (*IssueMetrics, error) {
	dates, err := c.GetIssueDates(issueKey)
	if err != nil {
		return nil, err
	}

	metrics := &IssueMetrics{
		IssueKey:      dates.IssueKey,
		CurrentStatus: dates.CurrentStatus,
		StatusSummary: dates.StatusSummary,
	}

	now := time.Now().UTC()

	// Lead time: Created → ResolutionDate (or now if unresolved)
	if dates.Created != "" {
		createdTime, err := parseJiraTimestamp(dates.Created)
		if err == nil {
			var endTime time.Time
			if dates.ResolutionDate != "" {
				endTime, _ = parseJiraTimestamp(dates.ResolutionDate)
			}
			if endTime.IsZero() {
				endTime = now
			}
			leadMins := int(endTime.Sub(createdTime).Minutes())
			metrics.LeadTimeMinutes = &leadMins
			metrics.LeadTimeDisplay = formatDuration(leadMins)
		}
	}

	// Cycle time: first entry into an "in progress" category → resolution (or now)
	inProgressTime := findFirstInProgressTime(dates.StatusTransitions)
	if !inProgressTime.IsZero() && dates.Created != "" {
		var endTime time.Time
		if dates.ResolutionDate != "" {
			endTime, _ = parseJiraTimestamp(dates.ResolutionDate)
		}
		if endTime.IsZero() {
			endTime = now
		}
		cycleMins := int(endTime.Sub(inProgressTime).Minutes())
		metrics.CycleTimeMinutes = &cycleMins
		metrics.CycleTimeDisplay = formatDuration(cycleMins)
	}

	// Time in current status
	currentEnteredAt := findCurrentStatusEntryTime(dates.StatusTransitions, dates.CurrentStatus, dates.Created)
	if !currentEnteredAt.IsZero() {
		mins := int(now.Sub(currentEnteredAt).Minutes())
		metrics.TimeInCurrentMinutes = &mins
		metrics.TimeInCurrentDisplay = formatDuration(mins)
	}

	return metrics, nil
}

// fetchAllChangelogs paginates through all changelog entries for an issue.
func (c *Client) fetchAllChangelogs(issueKey string) ([]Changelog, error) {
	var all []Changelog
	startAt := 0
	for {
		page, err := c.GetIssueChangelog(issueKey, startAt, 100)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Histories...)
		if startAt+len(page.Histories) >= page.Total || len(page.Histories) == 0 {
			break
		}
		startAt += len(page.Histories)
	}
	return all, nil
}

// parseStatusTransitions extracts status changes from changelogs and computes durations.
func parseStatusTransitions(changelogs []Changelog, createdDate string) []StatusTransition {
	// Collect raw status changes
	type rawTransition struct {
		from   string
		to     string
		at     time.Time
		atRaw  string
		author string
	}
	var transitions []rawTransition

	for _, cl := range changelogs {
		clTime, err := parseJiraTimestamp(cl.Created)
		if err != nil {
			continue
		}
		for _, item := range cl.Items {
			if item.Field != "status" {
				continue
			}
			author := ""
			if cl.Author != nil {
				author = cl.Author.DisplayName
			}
			transitions = append(transitions, rawTransition{
				from:   item.FromString,
				to:     item.ToString,
				at:     clTime,
				atRaw:  cl.Created,
				author: author,
			})
		}
	}

	// Sort by time ascending
	sort.Slice(transitions, func(i, j int) bool {
		return transitions[i].at.Before(transitions[j].at)
	})

	var result []StatusTransition

	// Add initial status entry (from creation to first transition)
	if len(transitions) > 0 && createdDate != "" {
		createdTime, err := parseJiraTimestamp(createdDate)
		if err == nil {
			dur := int(transitions[0].at.Sub(createdTime).Minutes())
			result = append(result, StatusTransition{
				FromStatus:      transitions[0].from,
				ToStatus:        transitions[0].to,
				TransitionedAt:  transitions[0].atRaw,
				TransitionedBy:  transitions[0].author,
				DurationMinutes: &dur,
				DurationDisplay: formatDuration(dur),
			})
		}
	} else if len(transitions) > 0 {
		// No created date, still include first transition without duration
		result = append(result, StatusTransition{
			FromStatus:     transitions[0].from,
			ToStatus:       transitions[0].to,
			TransitionedAt: transitions[0].atRaw,
			TransitionedBy: transitions[0].author,
		})
	}

	// Remaining transitions: duration = time from this transition to the next one
	for i := 1; i < len(transitions); i++ {
		dur := int(transitions[i].at.Sub(transitions[i-1].at).Minutes())
		result = append(result, StatusTransition{
			FromStatus:      transitions[i].from,
			ToStatus:        transitions[i].to,
			TransitionedAt:  transitions[i].atRaw,
			TransitionedBy:  transitions[i].author,
			DurationMinutes: &dur,
			DurationDisplay: formatDuration(dur),
		})
	}

	return result
}

// aggregateStatusTimes aggregates time spent in each status across all visits.
func aggregateStatusTimes(transitions []StatusTransition) []StatusTimeSummary {
	type accumulator struct {
		totalMinutes int
		visitCount   int
	}
	byStatus := make(map[string]*accumulator)

	for _, t := range transitions {
		// Count time in the from_status (the status we're leaving)
		if t.DurationMinutes != nil {
			acc, ok := byStatus[t.FromStatus]
			if !ok {
				acc = &accumulator{}
				byStatus[t.FromStatus] = acc
			}
			acc.totalMinutes += *t.DurationMinutes
			acc.visitCount++
		}
	}

	var summaries []StatusTimeSummary
	for status, acc := range byStatus {
		summaries = append(summaries, StatusTimeSummary{
			Status:               status,
			TotalDurationMinutes: acc.totalMinutes,
			TotalDurationDisplay: formatDuration(acc.totalMinutes),
			VisitCount:           acc.visitCount,
		})
	}

	// Sort by total duration descending
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].TotalDurationMinutes > summaries[j].TotalDurationMinutes
	})

	return summaries
}

// inProgressStatuses are status names that indicate "work started" for cycle time.
var inProgressStatuses = map[string]bool{
	"in progress":  true,
	"in review":    true,
	"in testing":   true,
	"in qa":        true,
	"in dev":       true,
	"development":  true,
	"coding":       true,
	"implementing": true,
}

// findFirstInProgressTime returns the time when the issue first entered an "in progress" status.
func findFirstInProgressTime(transitions []StatusTransition) time.Time {
	for _, t := range transitions {
		lowerTo := toLower(t.ToStatus)
		if inProgressStatuses[lowerTo] {
			ts, err := parseJiraTimestamp(t.TransitionedAt)
			if err == nil {
				return ts
			}
		}
	}
	return time.Time{}
}

// findCurrentStatusEntryTime returns when the issue entered its current status.
func findCurrentStatusEntryTime(transitions []StatusTransition, currentStatus, createdDate string) time.Time {
	if currentStatus == "" {
		return time.Time{}
	}

	// Walk backward to find the last transition *to* the current status
	for i := len(transitions) - 1; i >= 0; i-- {
		if transitions[i].ToStatus == currentStatus {
			ts, err := parseJiraTimestamp(transitions[i].TransitionedAt)
			if err == nil {
				return ts
			}
		}
	}

	// If no transition found, it's been in the same status since creation
	if createdDate != "" {
		ts, err := parseJiraTimestamp(createdDate)
		if err == nil {
			return ts
		}
	}
	return time.Time{}
}

// parseJiraTimestamp parses Jira's ISO 8601 timestamp format.
func parseJiraTimestamp(s string) (time.Time, error) {
	// Jira uses formats like "2024-01-15T09:30:00.000+0000" or "2024-01-15T09:30:00.000+00:00"
	formats := []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}

// formatDuration converts minutes to a human-readable string like "1d 2h 30m".
func formatDuration(minutes int) string {
	if minutes <= 0 {
		return "0m"
	}

	days := minutes / (24 * 60)
	remaining := minutes % (24 * 60)
	hours := remaining / 60
	mins := remaining % 60

	parts := make([]string, 0, 3)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	parts = append(parts, fmt.Sprintf("%dm", mins))

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

// toLower is a simple lowercase helper to avoid importing strings.
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
