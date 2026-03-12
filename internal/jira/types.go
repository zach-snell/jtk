package jira

// Types for the Jira REST API.

// User represents a Jira user.
type User struct {
	AccountID   string            `json:"accountId"`
	DisplayName string            `json:"displayName"`
	EmailAddr   string            `json:"emailAddress,omitempty"`
	Active      bool              `json:"active"`
	AvatarURLs  map[string]string `json:"avatarUrls,omitempty"`
	Self        string            `json:"self,omitempty"`
}

// Status represents an issue status.
type Status struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	StatusCategory StatusCategory `json:"statusCategory,omitempty"`
	Self           string         `json:"self,omitempty"`
}

// StatusCategory represents the category of a status.
type StatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Priority represents an issue priority.
type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Self string `json:"self,omitempty"`
}

// IssueType represents the type of an issue.
type IssueType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Subtask     bool   `json:"subtask"`
	Self        string `json:"self,omitempty"`
}

// Project represents a Jira project.
type Project struct {
	ID          string      `json:"id"`
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	ProjectType string      `json:"projectTypeKey,omitempty"`
	Style       string      `json:"style,omitempty"`
	Description string      `json:"description,omitempty"`
	Lead        *User       `json:"lead,omitempty"`
	IssueTypes  []IssueType `json:"issueTypes,omitempty"`
	Self        string      `json:"self,omitempty"`
}

// Component represents a project component.
type Component struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Self string `json:"self,omitempty"`
}

// Sprint represents an agile sprint.
type Sprint struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	State         string `json:"state"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	CompleteDate  string `json:"completeDate,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
	Goal          string `json:"goal,omitempty"`
	Self          string `json:"self,omitempty"`
}

// IssueFields represents the fields of a Jira issue.
type IssueFields struct {
	Summary        string      `json:"summary"`
	Description    interface{} `json:"description,omitempty"` // ADF format
	Status         *Status     `json:"status,omitempty"`
	IssueType      *IssueType  `json:"issuetype,omitempty"`
	Priority       *Priority   `json:"priority,omitempty"`
	Assignee       *User       `json:"assignee,omitempty"`
	Reporter       *User       `json:"reporter,omitempty"`
	Creator        *User       `json:"creator,omitempty"`
	Project        *Project    `json:"project,omitempty"`
	Labels         []string    `json:"labels,omitempty"`
	Components     []Component `json:"components,omitempty"`
	Created        string      `json:"created,omitempty"`
	Updated        string      `json:"updated,omitempty"`
	ResolutionDate string      `json:"resolutiondate,omitempty"`
	Parent         *Issue      `json:"parent,omitempty"`

	// Sprint is typically in a custom field, but we also check the known field
	Sprint *Sprint `json:"sprint,omitempty"`

	// Story points — commonly customfield_10016 but varies by instance
	StoryPoints *float64 `json:"story_points,omitempty"`

	// Comment container
	Comment *CommentPage `json:"comment,omitempty"`
}

// Issue represents a Jira issue.
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self,omitempty"`
	Fields IssueFields `json:"fields"`
	Expand string      `json:"expand,omitempty"`
}

// SearchResult represents the response from a JQL search.
type SearchResult struct {
	StartAt       int     `json:"startAt"`
	MaxResults    int     `json:"maxResults"`
	Total         int     `json:"total"`
	Issues        []Issue `json:"issues"`
	Expand        string  `json:"expand,omitempty"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	IsLast        bool    `json:"isLast,omitempty"`
}

// Transition represents a possible issue state transition.
type Transition struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	To   *Status `json:"to,omitempty"`
}

// TransitionsResponse is the API response for available transitions.
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
	Expand      string       `json:"expand,omitempty"`
}

// Comment represents an issue comment.
type Comment struct {
	ID      string      `json:"id"`
	Author  *User       `json:"author,omitempty"`
	Body    interface{} `json:"body,omitempty"` // ADF format
	Created string      `json:"created,omitempty"`
	Updated string      `json:"updated,omitempty"`
	Self    string      `json:"self,omitempty"`
}

// CommentPage represents a paginated list of comments.
type CommentPage struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// Board represents a Jira agile board.
type Board struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Location *struct {
		ProjectID  int    `json:"projectId,omitempty"`
		ProjectKey string `json:"projectKey,omitempty"`
		Name       string `json:"displayName,omitempty"`
	} `json:"location,omitempty"`
	Self string `json:"self,omitempty"`
}

// BoardsResponse represents the agile boards list response.
type BoardsResponse struct {
	MaxResults int     `json:"maxResults"`
	StartAt    int     `json:"startAt"`
	Total      int     `json:"total"`
	IsLast     bool    `json:"isLast"`
	Values     []Board `json:"values"`
}

// SprintsResponse represents the sprints list response.
type SprintsResponse struct {
	MaxResults int      `json:"maxResults"`
	StartAt    int      `json:"startAt"`
	Total      int      `json:"total"`
	IsLast     bool     `json:"isLast"`
	Values     []Sprint `json:"values"`
}

// DevStatusSummary is the response from /rest/dev-status/latest/issue/summary.
type DevStatusSummary struct {
	Summary map[string]DevStatusDataTypeSummary `json:"summary"`
}

// DevStatusDataTypeSummary contains the byInstanceType map for a data type.
type DevStatusDataTypeSummary struct {
	ByInstanceType map[string]interface{} `json:"byInstanceType"`
}

// DevStatusDetail is the response from /rest/dev-status/latest/issue/detail.
type DevStatusDetail struct {
	Detail []DevStatusDetailItem `json:"detail"`
	Errors []interface{}         `json:"errors"`
}

// DevStatusDetailItem contains development data from a single VCS integration.
type DevStatusDetailItem struct {
	Branches     []DevBranch     `json:"branches"`
	PullRequests []DevPR         `json:"pullRequests"`
	Repositories []DevRepository `json:"repositories"`
	Builds       []DevBuild      `json:"builds"`
}

// DevBranch represents a branch linked to an issue.
type DevBranch struct {
	Name                 string      `json:"name"`
	URL                  string      `json:"url"`
	CreatePullRequestURL string      `json:"createPullRequestUrl,omitempty"`
	Repository           *DevRepoRef `json:"repository,omitempty"`
	LastCommit           *DevCommit  `json:"lastCommit,omitempty"`
}

// DevPR represents a pull request linked to an issue.
type DevPR struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	Status       string        `json:"status"` // OPEN, MERGED, DECLINED
	Author       *DevUser      `json:"author,omitempty"`
	Source       *DevBranchRef `json:"source,omitempty"`
	Destination  *DevBranchRef `json:"destination,omitempty"`
	CommentCount int           `json:"commentCount,omitempty"`
	Reviewers    []DevUser     `json:"reviewers,omitempty"`
	LastUpdate   string        `json:"lastUpdate,omitempty"`
}

// DevRepository represents a Git repository with commits.
type DevRepository struct {
	Name    string      `json:"name"`
	URL     string      `json:"url"`
	Commits []DevCommit `json:"commits"`
}

// DevCommit represents a commit linked to an issue.
type DevCommit struct {
	ID        string   `json:"id"`
	DisplayID string   `json:"displayId,omitempty"`
	Message   string   `json:"message"`
	Author    *DevUser `json:"author,omitempty"`
	URL       string   `json:"url,omitempty"`
	Timestamp string   `json:"authorTimestamp,omitempty"`
	FileCount int      `json:"fileCount,omitempty"`
	Merge     bool     `json:"merge,omitempty"`
}

// DevBuild represents a CI/CD build linked to an issue.
type DevBuild struct {
	State string `json:"state"`
	URL   string `json:"url"`
	Name  string `json:"name,omitempty"`
}

// DevUser represents a user in dev info.
type DevUser struct {
	Name   string `json:"name,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	URL    string `json:"url,omitempty"`
}

// DevRepoRef is a lightweight repository reference.
type DevRepoRef struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// DevBranchRef is a branch reference used in pull requests.
type DevBranchRef struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Permission represents a single Jira permission.
type Permission struct {
	ID             string `json:"id"`
	Key            string `json:"key"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	HavePermission bool   `json:"havePermission"`
}

// PermissionsResponse is the response from the mypermissions endpoint.
type PermissionsResponse struct {
	Permissions map[string]Permission `json:"permissions"`
}

// CreateIssueRequest is the payload for creating an issue.
type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

// CreateIssueFields are the fields required to create an issue.
type CreateIssueFields struct {
	Project     ProjectRef   `json:"project"`
	Summary     string       `json:"summary"`
	IssueType   IssueTypeRef `json:"issuetype"`
	Description interface{}  `json:"description,omitempty"`
	Assignee    *UserRef     `json:"assignee,omitempty"`
	Priority    *PriorityRef `json:"priority,omitempty"`
	Labels      []string     `json:"labels,omitempty"`
	Parent      *IssueRef    `json:"parent,omitempty"`
}

// ProjectRef is a minimal project reference by key.
type ProjectRef struct {
	Key string `json:"key"`
}

// IssueTypeRef is a minimal issue type reference by name.
type IssueTypeRef struct {
	Name string `json:"name"`
}

// UserRef is a minimal user reference by account ID.
type UserRef struct {
	AccountID string `json:"accountId"`
}

// PriorityRef is a minimal priority reference by name.
type PriorityRef struct {
	Name string `json:"name"`
}

// IssueRef is a minimal issue reference by key.
type IssueRef struct {
	Key string `json:"key"`
}

// UpdateIssueRequest is the payload for updating an issue.
type UpdateIssueRequest struct {
	Fields map[string]interface{} `json:"fields"`
}

// TransitionRequest is the payload for transitioning an issue.
type TransitionRequest struct {
	Transition TransitionRef `json:"transition"`
}

// TransitionRef references a transition by ID.
type TransitionRef struct {
	ID string `json:"id"`
}

// AddCommentRequest is the payload for adding a comment.
type AddCommentRequest struct {
	Body interface{} `json:"body"`
}

// CreatedIssue is the response after creating an issue.
type CreatedIssue struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// Worklog represents a time tracking worklog entry.
type Worklog struct {
	ID               string      `json:"id"`
	IssueID          string      `json:"issueId,omitempty"`
	Author           *User       `json:"author,omitempty"`
	UpdateAuthor     *User       `json:"updateAuthor,omitempty"`
	Created          string      `json:"created"`
	Updated          string      `json:"updated"`
	Started          string      `json:"started"`
	TimeSpent        string      `json:"timeSpent"`
	TimeSpentSeconds int         `json:"timeSpentSeconds"`
	Comment          interface{} `json:"comment,omitempty"` // ADF
}

// WorklogPage represents a paginated list of worklogs.
type WorklogPage struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Worklogs   []Worklog `json:"worklogs"`
}

// AddWorklogRequest is the payload for adding a worklog.
type AddWorklogRequest struct {
	TimeSpent string      `json:"timeSpent"`
	Started   string      `json:"started,omitempty"`
	Comment   interface{} `json:"comment,omitempty"` // ADF
}

// Version represents a project version (fixVersion / release).
type Version struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Released    bool   `json:"released"`
	Archived    bool   `json:"archived"`
	ReleaseDate string `json:"releaseDate,omitempty"`
	StartDate   string `json:"startDate,omitempty"`
	ProjectID   int    `json:"projectId,omitempty"`
}

// IssueLink represents a link between two issues.
type IssueLink struct {
	ID           string        `json:"id"`
	Type         IssueLinkType `json:"type"`
	InwardIssue  *Issue        `json:"inwardIssue,omitempty"`
	OutwardIssue *Issue        `json:"outwardIssue,omitempty"`
}

// IssueLinkType describes the relationship type of an issue link.
type IssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

// CreateIssueLinkRequest is the payload for linking two issues.
type CreateIssueLinkRequest struct {
	Type         IssueLinkTypeRef `json:"type"`
	InwardIssue  IssueRef         `json:"inwardIssue"`
	OutwardIssue IssueRef         `json:"outwardIssue"`
	Comment      interface{}      `json:"comment,omitempty"`
}

// IssueLinkTypeRef references a link type by name.
type IssueLinkTypeRef struct {
	Name string `json:"name"`
}

// ChangelogPage represents a paginated list of changelog entries.
type ChangelogPage struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Histories  []Changelog `json:"values"`
}

// Changelog represents a single change event on an issue.
type Changelog struct {
	ID      string          `json:"id"`
	Author  *User           `json:"author,omitempty"`
	Created string          `json:"created"`
	Items   []ChangelogItem `json:"items"`
}

// ChangelogItem represents a single field change in a changelog entry.
type ChangelogItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldtype"`
	From       string `json:"from,omitempty"`
	FromString string `json:"fromString,omitempty"`
	To         string `json:"to,omitempty"`
	ToString   string `json:"toString,omitempty"`
}

// IssueTypeWithStatuses represents an issue type and its available statuses.
type IssueTypeWithStatuses struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Statuses []Status `json:"statuses"`
}

// Attachment represents a file attached to an issue.
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Author   *User  `json:"author,omitempty"`
	Created  string `json:"created"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Content  string `json:"content"` // download URL
}

// UserDetail extends User with additional profile fields.
type UserDetail struct {
	AccountID    string            `json:"accountId"`
	AccountType  string            `json:"accountType,omitempty"`
	DisplayName  string            `json:"displayName"`
	EmailAddress string            `json:"emailAddress,omitempty"`
	Active       bool              `json:"active"`
	TimeZone     string            `json:"timeZone,omitempty"`
	Locale       string            `json:"locale,omitempty"`
	AvatarURLs   map[string]string `json:"avatarUrls,omitempty"`
	Self         string            `json:"self,omitempty"`
}

// CreateSprintRequest is the payload for creating a sprint.
type CreateSprintRequest struct {
	Name          string `json:"name"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	OriginBoardID int    `json:"originBoardId"`
	Goal          string `json:"goal,omitempty"`
}

// MoveIssuesToSprintRequest is the payload for moving issues to a sprint.
type MoveIssuesToSprintRequest struct {
	Issues []string `json:"issues"`
}
