package jira_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zach-snell/jtk/internal/jira"
)

// reparentServer serves a PUT /issue/{key} that always 204s, and a
// GET /issue/{key} that reports whatever parent key it is told to.
func reparentServer(t *testing.T, readbackParent string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			parent := ""
			if readbackParent != "" {
				parent = fmt.Sprintf(`,"parent":{"key":%q}`, readbackParent)
			}
			fmt.Fprintf(w, `{"key":"KAN-57","fields":{"summary":"x"%s}}`, parent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}

func TestReparentIssue_Verified(t *testing.T) {
	srv := reparentServer(t, "KAN-44") // server confirms the new parent stuck
	defer srv.Close()

	c := jira.NewTestClient(srv.URL)
	if err := c.ReparentIssue("KAN-57", "KAN-44"); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestReparentIssue_SilentNoOp(t *testing.T) {
	srv := reparentServer(t, "KAN-45") // server still reports the OLD parent
	defer srv.Close()

	c := jira.NewTestClient(srv.URL)
	err := c.ReparentIssue("KAN-57", "KAN-44")
	if err == nil {
		t.Fatal("expected a loud error when the parent silently did not change, got nil")
	}
	if !strings.Contains(err.Error(), "silently ignored") {
		t.Fatalf("expected a silent-no-op error, got: %v", err)
	}
}

func TestDo_RetriesOn429(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0") // retry immediately
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, `{"id":"1","key":"KAN-1","fields":{"summary":"ok"}}`)
	}))
	defer srv.Close()

	c := jira.NewTestClient(srv.URL)
	if _, err := c.GetIssue("KAN-1"); err != nil {
		t.Fatalf("expected success after a 429 retry, got: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls (one 429 + one success), got %d", calls)
	}
}
