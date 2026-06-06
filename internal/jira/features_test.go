package jira_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zach-snell/jtk/internal/jira"
)

// --- SearchAllJQL: pagination + truncation detection (the bulk-bug fix) ---

func pagedSearchServer(t *testing.T) *httptest.Server {
	t.Helper()
	// Two pages of 2 issues each, chained by nextPageToken.
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			NextPageToken string `json:"nextPageToken"`
		}
		_ = json.Unmarshal(body, &req)
		switch req.NextPageToken {
		case "":
			fmt.Fprint(w, `{"issues":[{"key":"A-1"},{"key":"A-2"}],"nextPageToken":"p2","isLast":false}`)
		default:
			fmt.Fprint(w, `{"issues":[{"key":"A-3"},{"key":"A-4"}],"isLast":true}`)
		}
	}))
}

func TestSearchAllJQL_FetchesAllPages(t *testing.T) {
	srv := pagedSearchServer(t)
	defer srv.Close()
	c := jira.NewTestClient(srv.URL)

	issues, truncated, err := c.SearchAllJQL("project = A", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 4 {
		t.Fatalf("expected 4 issues across 2 pages, got %d", len(issues))
	}
	if truncated {
		t.Fatal("did not expect truncation when cap exceeds total")
	}
}

func TestSearchAllJQL_ReportsTruncation(t *testing.T) {
	srv := pagedSearchServer(t)
	defer srv.Close()
	c := jira.NewTestClient(srv.URL)

	issues, truncated, err := c.SearchAllJQL("project = A", 3) // 4 match, cap 3
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("expected cap of 3, got %d", len(issues))
	}
	if !truncated {
		t.Fatal("expected truncated=true when more issues match than the cap")
	}
}

// --- ArchiveIssues: request shape + result parse ---

func TestArchiveIssues(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		fmt.Fprint(w, `{"numberOfIssuesUpdated":2,"errors":{}}`)
	}))
	defer srv.Close()
	c := jira.NewTestClient(srv.URL)

	res, err := c.ArchiveIssues([]string{"A-1", "A-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.NumberOfIssuesUpdated != 2 {
		t.Fatalf("expected 2 updated, got %d", res.NumberOfIssuesUpdated)
	}
	if !strings.HasSuffix(gotPath, "/issue/archive") {
		t.Fatalf("expected /issue/archive, got %s", gotPath)
	}
	if !strings.Contains(gotBody, `"issueIdsOrKeys"`) || !strings.Contains(gotBody, "A-1") {
		t.Fatalf("unexpected archive body: %s", gotBody)
	}
}

// --- ModifyLabels: uses the non-clobbering "update" verb ---

func TestModifyLabels_NonClobbering(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := jira.NewTestClient(srv.URL)

	if err := c.ModifyLabels("A-1", []string{"add-me"}, []string{"drop-me"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must use "update" (incremental), never "fields" (full replace).
	if !strings.Contains(gotBody, `"update"`) || strings.Contains(gotBody, `"fields"`) {
		t.Fatalf("ModifyLabels must use the update verb, not fields: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"add":"add-me"`) || !strings.Contains(gotBody, `"remove":"drop-me"`) {
		t.Fatalf("expected add/remove ops in body: %s", gotBody)
	}
}
