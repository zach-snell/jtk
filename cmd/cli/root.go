package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/jtk/internal/jira"
	"github.com/zach-snell/jtk/internal/version"
)

// RootCmd represents the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:     "jtk",
	Version: version.Version,
	Short:   "A unified CLI and MCP server for Jira Cloud",
	Long: `jtk is a complete command-line interface and Model Context Protocol
server for Jira Cloud.

It allows you to manage issues, sprints, boards, and development info
directly from your terminal, or expose these capabilities to your AI
agents via the MCP protocol.

Try running 'jtk auth' to get started!`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().Bool("json", false, "Output raw JSON instead of formatted tables")
}

// getClient creates a Jira API client from stored credentials or env vars.
func getClient() *jira.Client {
	creds, err := jira.LoadCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'jtk auth' to authenticate, or set JIRA_DOMAIN, JIRA_EMAIL, and JIRA_API_TOKEN env vars.\n")
		os.Exit(1)
	}
	return jira.NewClient(creds.Domain, creds.Email, creds.APIToken)
}
