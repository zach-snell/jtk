package jira

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Credentials holds persisted Jira authentication data.
type Credentials struct {
	Domain   string    `json:"domain"` // e.g., "mycompany" (for mycompany.atlassian.net)
	Email    string    `json:"email"`
	APIToken string    `json:"api_token"`
	SavedAt  time.Time `json:"saved_at"`
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
		return &Credentials{
			Domain:   domain,
			Email:    email,
			APIToken: token,
		}, nil
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

	// Verify credentials
	fmt.Println("\nVerifying credentials...")
	client := NewClient(domain, email, token)
	_, verifyErr := client.Get("/myself")
	if verifyErr != nil {
		return fmt.Errorf("credential verification failed: %w\n\nCheck your domain, email, and API token", verifyErr)
	}

	fmt.Println("Credentials verified successfully!")

	creds := &Credentials{
		Domain:   domain,
		Email:    email,
		APIToken: token,
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
