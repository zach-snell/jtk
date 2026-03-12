package mcp

import "github.com/modelcontextprotocol/go-sdk/mcp"

// ToolResultText creates a strictly typed success *mcp.CallToolResult with text content.
func ToolResultText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// ToolResultError creates a strictly typed error *mcp.CallToolResult.
func ToolResultError(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		IsError: true,
	}
}
