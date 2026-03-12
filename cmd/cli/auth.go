package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/jtk/internal/jira"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Jira Cloud",
	Long: `Set up credentials for accessing Jira Cloud.

You will need:
  1. Your Jira domain (e.g., 'mycompany' for mycompany.atlassian.net)
  2. Your Atlassian email address
  3. An API Token from https://id.atlassian.com/manage-profile/security/api-tokens`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := jira.InteractiveLogin(); err != nil {
			fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		runStatus()
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored credentials",
	Run: func(cmd *cobra.Command, args []string) {
		runLogout()
	},
}

func init() {
	RootCmd.AddCommand(authCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(logoutCmd)
}

func runStatus() {
	// Check env vars first
	domain := os.Getenv("JIRA_DOMAIN")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_API_TOKEN")

	if domain != "" && email != "" && token != "" {
		fmt.Println("Authenticated via environment variables")
		fmt.Printf("  Domain: %s.atlassian.net\n", domain)
		fmt.Printf("  Email:  %s\n", email)
		return
	}

	creds, err := jira.LoadCredentials()
	if err != nil {
		fmt.Println("Not authenticated. Run: jtk auth")
		return
	}

	path, _ := jira.CredentialsPath()
	fmt.Println("Authenticated via stored credentials")
	KV("Domain", creds.Domain+".atlassian.net")
	KV("Email", creds.Email)
	if len(creds.APIToken) > 8 {
		KVf("Token", "%s...%s", creds.APIToken[:4], creds.APIToken[len(creds.APIToken)-4:])
	} else {
		KV("Token", "****")
	}
	KV("Saved", creds.SavedAt.Format("2006-01-02 15:04:05"))
	KV("File", path)
}

func runLogout() {
	if err := jira.RemoveCredentials(); err != nil {
		fmt.Fprintf(os.Stderr, "error removing credentials: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Logged out. Credentials removed.")
}
