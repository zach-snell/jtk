package jira

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TokenType indicates the kind of Atlassian API token in use.
type TokenType string

const (
	// TokenTypeClassic is a classic API token — uses Basic Auth against the site URL.
	TokenTypeClassic TokenType = "classic"
	// TokenTypeScoped is a fine-grained/scoped API token — uses Bearer auth against the gateway URL.
	TokenTypeScoped TokenType = "scoped"
)

// Credentials holds persisted Jira authentication data.
type Credentials struct {
	Domain   string    `json:"domain"` // e.g., "mycompany" (for mycompany.atlassian.net)
	Email    string    `json:"email"`
	APIToken string    `json:"api_token"`
	CloudID  string    `json:"cloud_id,omitempty"`   // Atlassian Cloud ID (required for scoped tokens)
	Type     TokenType `json:"token_type,omitempty"` // "classic" or "scoped"
	SavedAt  time.Time `json:"saved_at"`
}

// FetchCloudID retrieves the Atlassian Cloud ID for a given domain.
// This endpoint is public and requires no authentication.
func FetchCloudID(domain string) (string, error) {
	url := fmt.Sprintf("https://%s.atlassian.net/_edge/tenant_info", domain)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // URL is constructed from user-provided domain
	if err != nil {
		return "", fmt.Errorf("fetching tenant info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tenant info returned status %d (is '%s' a valid Atlassian domain?)", resp.StatusCode, domain)
	}

	var info struct {
		CloudID string `json:"cloudId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("parsing tenant info: %w", err)
	}
	if info.CloudID == "" {
		return "", fmt.Errorf("no cloudId found in tenant info for domain '%s'", domain)
	}

	return info.CloudID, nil
}

// ProbeTokenType determines whether a token is classic or scoped by testing both
// auth methods against the Jira API. It tries basic auth first (classic),
// and falls back to Bearer auth via the gateway (scoped).
// Returns the detected token type and cloudID (empty for classic tokens).
func ProbeTokenType(domain, email, token string) (TokenType, string, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// 1. Try classic: Basic Auth against direct site URL
	classicURL := fmt.Sprintf("https://%s.atlassian.net/rest/api/3/myself", domain)
	req, err := http.NewRequest(http.MethodGet, classicURL, http.NoBody)
	if err != nil {
		return "", "", fmt.Errorf("creating probe request: %w", err)
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("probing classic auth: %w", err)
	}
	classicBody := readBodySnippet(resp.Body)
	resp.Body.Close()

	// Only 401 means "auth credentials rejected" — 403/404/etc mean auth worked
	// but the user lacks permissions or the resource doesn't exist. A scope-denial
	// (HTTP 401 "scope does not match") means the token authenticated but lacks the
	// scope for /myself — the credentials are still valid, so treat it as success.
	if resp.StatusCode != http.StatusUnauthorized || isScopeDenied(classicBody) {
		return TokenTypeClassic, "", nil
	}

	// 2. Classic rejected (401) — try scoped: Basic Auth against gateway URL
	//    Scoped tokens use the same Basic Auth as classic, just via the gateway.
	cloudID, err := FetchCloudID(domain)
	if err != nil {
		return "", "", fmt.Errorf("classic auth rejected and could not fetch Cloud ID for scoped fallback: %w", err)
	}

	gatewayURL := fmt.Sprintf("https://api.atlassian.com/ex/jira/%s/rest/api/3/myself", cloudID)
	req, err = http.NewRequest(http.MethodGet, gatewayURL, http.NoBody)
	if err != nil {
		return "", "", fmt.Errorf("creating gateway probe request: %w", err)
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	resp, err = httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("probing scoped auth: %w", err)
	}
	scopedBody := readBodySnippet(resp.Body)
	resp.Body.Close()

	// 200 (or any non-401) means the scoped token works for /myself. A 401
	// "scope does not match" means the token authenticated against the gateway
	// but lacks the read:user:jira scope — the credentials are valid, so accept
	// them as a scoped token. Only a generic 401 ("Client must be authenticated")
	// indicates the token itself was rejected.
	if resp.StatusCode != http.StatusUnauthorized || isScopeDenied(scopedBody) {
		return TokenTypeScoped, cloudID, nil
	}

	return "", "", fmt.Errorf("authentication failed with both classic (Basic Auth) and scoped (Bearer) methods. Verify your domain, email, and API token are correct")
}

// readBodySnippet reads up to 1 KiB of a response body for diagnostic matching.
func readBodySnippet(body io.Reader) string {
	snippet, _ := io.ReadAll(io.LimitReader(body, 1024))
	return string(snippet)
}

// isScopeDenied reports whether an Atlassian 401 body indicates the token
// authenticated successfully but lacks the scope for the requested endpoint
// (gateway returns `{"message":"Unauthorized; scope does not match"}`), as
// opposed to the credentials themselves being rejected.
func isScopeDenied(body string) bool {
	return strings.Contains(body, "scope does not match")
}

// CredentialsPath returns the path to the credentials file.
func CredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".config", "jtk", "credentials.json"), nil
}

// SaveCredentials persists credentials to disk.
func SaveCredentials(creds *Credentials) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing credentials file: %w", err)
	}

	return nil
}

// LoadCredentials loads credentials from disk or environment variables.
// Priority: env vars > stored file.
func LoadCredentials() (*Credentials, error) {
	// 1. Check environment variables first
	domain := os.Getenv("JIRA_DOMAIN")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_API_TOKEN")

	if domain != "" && email != "" && token != "" {
		creds := &Credentials{
			Domain:   domain,
			Email:    email,
			APIToken: token,
			Type:     TokenType(os.Getenv("JIRA_TOKEN_TYPE")), // auto-detected if empty
			CloudID:  os.Getenv("JIRA_CLOUD_ID"),
		}
		return creds, nil
	}

	// 2. Fall back to stored credentials
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading credentials file: %w (run 'jtk auth' to authenticate)", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials file: %w", err)
	}

	return &creds, nil
}

// RemoveCredentials deletes the stored credentials file.
func RemoveCredentials() error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// ScopeReadOnly contains the granular scopes required for read-only access.
var ScopeReadOnly = []string{
	"read:me",
	"read:jql:jira",
	"read:issue-details:jira",
	"read:issue-type:jira",
	"read:issue-link:jira",
	"read:issue-worklog:jira",
	"read:issue.changelog:jira",
	"read:issue.transition:jira",
	"read:comment:jira",
	"read:attachment:jira",
	"read:project:jira",
	"read:project-version:jira",
	"read:status:jira",
	"read:user:jira",
	"read:permission:jira",
	"read:board-scope:jira-software",
	"read:sprint:jira-software",
	"read:dev-info:jira",
}

// ScopeWrite contains the additional granular scopes for write operations.
var ScopeWrite = []string{
	"write:issue:jira",
	"write:comment:jira",
	"write:issue-worklog:jira",
	"write:issue-link:jira",
	"write:attachment:jira",
	"write:sprint:jira-software",
	"delete:issue:jira",
}

// InteractiveLogin prompts the user for Jira credentials and stores them.
func InteractiveLogin() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("Jira Cloud API Token Authentication")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("Create an API Token at:")
	fmt.Println("  https://id.atlassian.com/manage-profile/security/api-tokens")
	fmt.Println()
	fmt.Println("Select \"Jira\" as the app, then add these scopes:")
	fmt.Println()
	fmt.Println("  Read-only (18 scopes):")
	for _, s := range ScopeReadOnly {
		fmt.Printf("    %s\n", s)
	}
	fmt.Println()
	fmt.Println("  Write access (add these 7 for full access):")
	for _, s := range ScopeWrite {
		fmt.Printf("    %s\n", s)
	}
	fmt.Println()
	fmt.Println("Mutation tools are dynamically hidden if your token lacks write scopes.")
	fmt.Println("To explicitly deny tools despite having full scopes, use:")
	fmt.Println("  export JIRA_DISABLED_TOOLS=\"manage_boards,manage_worklogs\"")
	fmt.Println()

	fmt.Print("Jira domain (e.g., 'mycompany' for mycompany.atlassian.net): ")
	domain, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading domain: %w", err)
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	// Strip full URL if provided
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, ".atlassian.net")
	domain = strings.TrimSuffix(domain, "/")

	fmt.Print("Atlassian email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading email: %w", err)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}

	fmt.Print("API Token: ")
	token, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading API token: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("API token is required")
	}

	fmt.Println("\nVerifying credentials (trying classic auth, then scoped)...")

	tokenType, cloudID, probeErr := ProbeTokenType(domain, email, token)
	if probeErr != nil {
		return fmt.Errorf("credential verification failed: %w", probeErr)
	}

	switch tokenType {
	case TokenTypeScoped:
		fmt.Printf("Authenticated via scoped token (Bearer, Cloud ID: %s)\n", cloudID)
	default:
		fmt.Println("Authenticated via classic token (Basic Auth)")
	}

	fmt.Println("Credentials verified successfully!")

	creds := &Credentials{
		Domain:   domain,
		Email:    email,
		APIToken: token,
		CloudID:  cloudID,
		Type:     tokenType,
		SavedAt:  time.Now(),
	}

	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	path, _ := CredentialsPath()
	fmt.Printf("\nCredentials saved to: %s\n", path)
	fmt.Println("You can now use the Jira CLI and MCP server.")
	return nil
}
