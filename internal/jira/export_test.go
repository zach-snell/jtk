package jira

// Export unexported functions for black-box testing in package jira_test.

var ParseInlineFormatting = parseInlineFormatting
var AutoLinkText = autoLinkText
var MakeParagraph = makeParagraph
var MakeListItem = makeListItem
var ParseTable = parseTable
var AdfPlainText = adfPlainText
var AdfMarkedText = adfMarkedText
var AdfLinkText = adfLinkText

// Flattener helpers
var GetStr = getStr
var GetNestedName = getNestedName
var GetNestedDisplayName = getNestedDisplayName
var ApplyMarks = applyMarks
