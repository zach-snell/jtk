package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user", "u"},
	Short:   "Manage Jira users",
}

var usersMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Get the currently authenticated user",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		user, err := client.GetCurrentUser()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, user, func() {
			fmt.Printf("%s\n", user.DisplayName)
			KV("Account ID", user.AccountID)
			if user.EmailAddress != "" {
				KV("Email", user.EmailAddress)
			}
			if user.AccountType != "" {
				KV("Account Type", user.AccountType)
			}
			KV("Active", FormatBool(user.Active))
			if user.TimeZone != "" {
				KV("Time Zone", user.TimeZone)
			}
			if user.Locale != "" {
				KV("Locale", user.Locale)
			}
		})
	},
}

var usersSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for users by name or email",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")
		maxResults, _ := cmd.Flags().GetInt("max")

		client := getClient()
		users, err := client.SearchUsers(query, maxResults)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, users, func() {
			if len(users) == 0 {
				fmt.Println("No users found.")
				return
			}
			t := NewTable()
			t.Header("Account ID", "Display Name", "Email", "Active")
			for _, u := range users {
				t.Row(
					u.AccountID,
					u.DisplayName,
					u.EmailAddress,
					FormatBool(u.Active),
				)
			}
			t.Flush()
		})
	},
}

var usersGetCmd = &cobra.Command{
	Use:   "get <account-id>",
	Short: "Get a user by account ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		accountID := args[0]

		client := getClient()
		user, err := client.GetUser(accountID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, user, func() {
			fmt.Printf("%s\n", user.DisplayName)
			KV("Account ID", user.AccountID)
			if user.EmailAddress != "" {
				KV("Email", user.EmailAddress)
			}
			if user.AccountType != "" {
				KV("Account Type", user.AccountType)
			}
			KV("Active", FormatBool(user.Active))
			if user.TimeZone != "" {
				KV("Time Zone", user.TimeZone)
			}
			if user.Locale != "" {
				KV("Locale", user.Locale)
			}
		})
	},
}

func init() {
	RootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(usersMeCmd)
	usersCmd.AddCommand(usersSearchCmd)
	usersCmd.AddCommand(usersGetCmd)

	usersSearchCmd.Flags().Int("max", 20, "Maximum results to return")
}
