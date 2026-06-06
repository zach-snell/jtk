package jira_test

import (
	"reflect"
	"testing"

	"github.com/zach-snell/jtk/internal/jira"
)

// clientExposure records, for every exported *jira.Client method, where it is
// reachable from: "cli", "mcp", "cli+mcp", or "internal" (transport/helpers or
// used only by other client methods). This is the codified result of the
// capability parity audit. The test below fails if a new exported method is
// added without classifying it here — so a capability can never silently exist
// in the client without a deliberate decision about CLI/MCP exposure (the bug
// class that once hid `jtk issues delete`).
//
// Entries marked "mcp" (no CLI) or "cli" (no MCP) are KNOWN, intentional gaps —
// documented here rather than silently missing. Closing them is tracked work.
var clientExposure = map[string]string{
	// transport / internal helpers
	"Get": "internal", "GetAbsolute": "internal", "GetAgile": "internal",
	"Post": "internal", "PostAgile": "internal", "PostMultipart": "internal",
	"Put": "internal", "Delete": "internal", "GetDevStatus": "internal",
	"Domain": "internal", "TokenType": "internal", "GetMyPermissions": "internal",
	"SearchJQLPaginated": "internal", // used by SearchAllJQL

	// issues
	"GetIssue": "cli+mcp", "CreateIssue": "cli+mcp", "UpdateIssue": "cli+mcp",
	"AssignIssue": "cli+mcp", "DeleteIssue": "cli+mcp", "TransitionIssue": "cli+mcp",
	"GetTransitions": "cli+mcp", "ArchiveIssues": "cli+mcp", "UnarchiveIssues": "cli+mcp",
	"MoveIssue": "cli+mcp", "ReparentIssue": "cli+mcp", "ModifyLabels": "cli+mcp",
	"ListIssueTypes": "mcp", // GAP: no CLI

	// comments
	"AddComment": "cli+mcp", "EditComment": "mcp", "ListComments": "mcp", // GAPs: no CLI

	// links / history
	"GetIssueLinks": "mcp", "GetIssueChangelog": "mcp",
	"CreateIssueLink": "mcp", "GetIssueLinkTypes": "mcp", // GAPs: no CLI

	// watchers
	"GetWatchers": "mcp", "AddWatcher": "mcp", "RemoveWatcher": "mcp", // GAPs: no CLI

	// search
	"SearchJQL": "cli+mcp", "SearchAllJQL": "cli", "QuickSearch": "mcp",

	// boards / sprints
	"ListBoards": "cli+mcp", "GetBoard": "mcp", "SearchSprintByName": "mcp",
	"ListSprints": "cli+mcp", "GetActiveSprint": "mcp", "GetSprintIssues": "mcp",
	"GetBoardBacklog": "mcp", "CreateSprint": "mcp", "UpdateSprint": "mcp",
	"MoveIssuesToSprint": "mcp", // GAPs: sprint CRUD not on CLI

	// projects / versions
	"ListProjects": "cli+mcp", "GetProject": "cli+mcp", "CreateProject": "cli+mcp",
	"GetProjectStatuses": "mcp",
	"ListVersions":       "cli+mcp", "GetVersion": "cli+mcp", "CreateVersion": "mcp",

	// attachments
	"ListAttachments": "cli+mcp", "DownloadAttachment": "cli+mcp",
	"GetAttachment": "internal", "UploadAttachment": "mcp", "DeleteAttachment": "mcp",

	// worklogs / users / metrics / devinfo
	"ListWorklogs": "cli+mcp", "AddWorklog": "cli+mcp",
	"GetCurrentUser": "cli+mcp", "GetUser": "cli+mcp", "SearchUsers": "cli+mcp",
	"GetIssueMetrics": "cli+mcp", "GetIssueDates": "cli+mcp",
	"GetDevelopmentInfo": "cli+mcp",

	// automation
	"ListAutomationRules": "cli+mcp", "GetAutomationRule": "cli+mcp",
	"CreateAutomationRule": "cli+mcp", "SetAutomationRuleState": "cli+mcp",
	"DeleteAutomationRule": "cli+mcp",
}

func TestClientCapabilityParity(t *testing.T) {
	valid := map[string]bool{"cli": true, "mcp": true, "cli+mcp": true, "internal": true}
	for k, v := range clientExposure {
		if !valid[v] {
			t.Errorf("clientExposure[%q] = %q is not a valid classification", k, v)
		}
	}

	typ := reflect.TypeOf(&jira.Client{})
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		if _, ok := clientExposure[name]; !ok {
			t.Errorf("client method %q is not classified in clientExposure — declare its CLI/MCP exposure (cli, mcp, cli+mcp, or internal) so the capability isn't silently unreachable", name)
		}
	}

	// Also catch stale entries (method removed but still listed).
	methods := map[string]bool{}
	for i := 0; i < typ.NumMethod(); i++ {
		methods[typ.Method(i).Name] = true
	}
	for name := range clientExposure {
		if !methods[name] {
			t.Errorf("clientExposure lists %q but it is no longer an exported *Client method — remove the stale entry", name)
		}
	}
}
