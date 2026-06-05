package jira

import (
	"net/http"
	"time"
)

// NewTestClient builds a Client pointed at a test server (e.g. httptest) so
// black-box tests in package jira_test can exercise request methods. The rate
// limiter is configured to be effectively instant.
func NewTestClient(baseURL string) *Client {
	return &Client{
		http:        &http.Client{Timeout: 5 * time.Second},
		baseURL:     baseURL,
		agileURL:    baseURL,
		siteURL:     baseURL,
		tokenType:   TokenTypeClassic,
		rateLimiter: NewRateLimiter(1000, time.Millisecond),
	}
}

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
