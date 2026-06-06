package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/jtk/internal/jira"
)

type ManageAutomationArgs struct {
	Action   string `json:"action" jsonschema:"Action: 'list', 'get', 'enable', 'disable', 'delete', 'create'" jsonschema_enum:"list,get,enable,disable,delete,create"`
	RuleUUID string `json:"rule_uuid,omitempty" jsonschema:"Automation rule UUID (for 'get', 'enable', 'disable', 'delete')"`
	RuleJSON string `json:"rule_json,omitempty" jsonschema:"Rule definition JSON for 'create' — shape {\"rule\":{...},\"connections\":[]}. Get an existing rule first to use as a template."`
}

// ManageAutomationHandler manages Jira Automation rules via the GA Automation
// Rule Management API. Creating a rule from scratch is fiddly; the practical
// flow is to build a rule in the UI, 'get' it as a template, then 'create'.
func ManageAutomationHandler(c *jira.Client) func(context.Context, *mcp.CallToolRequest, ManageAutomationArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageAutomationArgs) (*mcp.CallToolResult, any, error) {
		needsUUID := func() bool { return args.RuleUUID != "" }

		switch args.Action {
		case "list":
			data, err := c.ListAutomationRules()
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list rules: %v", err)), nil, nil
			}
			return ToolResultText(string(data)), nil, nil
		case "get":
			if !needsUUID() {
				return ToolResultError("rule_uuid is required for 'get'"), nil, nil
			}
			data, err := c.GetAutomationRule(args.RuleUUID)
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get rule: %v", err)), nil, nil
			}
			return ToolResultText(string(data)), nil, nil
		case "enable", "disable":
			if !needsUUID() {
				return ToolResultError("rule_uuid is required for 'enable'/'disable'"), nil, nil
			}
			if err := c.SetAutomationRuleState(args.RuleUUID, args.Action == "enable"); err != nil {
				return ToolResultError(fmt.Sprintf("failed to %s rule: %v", args.Action, err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("%sd rule %s", args.Action, args.RuleUUID)), nil, nil
		case "delete":
			if !needsUUID() {
				return ToolResultError("rule_uuid is required for 'delete'"), nil, nil
			}
			if err := c.DeleteAutomationRule(args.RuleUUID); err != nil {
				return ToolResultError(fmt.Sprintf("failed to delete rule: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("deleted rule %s", args.RuleUUID)), nil, nil
		case "create":
			if args.RuleJSON == "" {
				return ToolResultError("rule_json is required for 'create'"), nil, nil
			}
			data, err := c.CreateAutomationRule([]byte(args.RuleJSON))
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create rule: %v", err)), nil, nil
			}
			return ToolResultText(string(data)), nil, nil
		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s. Valid: list, get, enable, disable, delete, create", args.Action)), nil, nil
		}
	}
}
