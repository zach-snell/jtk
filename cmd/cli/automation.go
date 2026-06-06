package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var automationCmd = &cobra.Command{
	Use:   "automation",
	Short: "Manage Jira Automation rules",
	Long: `List, inspect, toggle, create, clone, and delete Jira Automation rules
via the (GA) Automation Rule Management API.

Creating a rule from scratch is fiddly — the rule-component JSON is barely
documented. The practical path is to build a rule once in the UI, then
'jtk automation get <uuid>' it as a template and 'create --from' / 'clone' it.`,
	Aliases: []string{"auto"},
}

var automationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List automation rules",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := getClient().ListAutomationRules()
		automationExit(err)
		printIndentedJSON(data)
	},
}

var automationGetCmd = &cobra.Command{
	Use:   "get <rule-uuid>",
	Short: "Get a rule's full definition (template for create/clone)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := getClient().GetAutomationRule(args[0])
		automationExit(err)
		printIndentedJSON(data)
	},
}

var automationEnableCmd = &cobra.Command{
	Use:   "enable <rule-uuid>",
	Short: "Enable a rule",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		automationExit(getClient().SetAutomationRuleState(args[0], true))
		fmt.Printf("Enabled rule %s\n", args[0])
	},
}

var automationDisableCmd = &cobra.Command{
	Use:   "disable <rule-uuid>",
	Short: "Disable a rule",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		automationExit(getClient().SetAutomationRuleState(args[0], false))
		fmt.Printf("Disabled rule %s\n", args[0])
	},
}

var automationDeleteCmd = &cobra.Command{
	Use:   "delete <rule-uuid>",
	Short: "Delete a rule (must be disabled first)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		automationExit(getClient().DeleteAutomationRule(args[0]))
		fmt.Printf("Deleted rule %s\n", args[0])
	},
}

var automationCreateCmd = &cobra.Command{
	Use:   "create --from <file.json>",
	Short: "Create a rule from a JSON file (POST /rule body)",
	Run: func(cmd *cobra.Command, args []string) {
		from, _ := cmd.Flags().GetString("from")
		if from == "" {
			fmt.Fprintln(os.Stderr, "Error: --from <file.json> is required")
			os.Exit(1)
		}
		body, err := os.ReadFile(from) //nolint:gosec // user-provided rule definition
		automationExit(err)
		data, err := getClient().CreateAutomationRule(body)
		automationExit(err)
		printIndentedJSON(data)
	},
}

var automationCloneCmd = &cobra.Command{
	Use:   "clone <rule-uuid> --name <new-name>",
	Short: "Clone an existing rule into a new one",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		newName, _ := cmd.Flags().GetString("name")
		raw, err := getClient().GetAutomationRule(args[0])
		automationExit(err)

		var rule map[string]interface{}
		if err := json.Unmarshal(raw, &rule); err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not parse source rule: %v\n", err)
			os.Exit(1)
		}
		// GET may return the rule directly or wrapped in {"rule": ...}.
		if inner, ok := rule["rule"].(map[string]interface{}); ok {
			rule = inner
		}
		stripAutomationIDs(rule) // ids/uuids are auto-generated on create
		if newName != "" {
			rule["name"] = newName
		} else if n, ok := rule["name"].(string); ok {
			rule["name"] = n + " (copy)"
		}

		body, err := json.Marshal(map[string]interface{}{"rule": rule, "connections": []interface{}{}})
		automationExit(err)
		data, err := getClient().CreateAutomationRule(body)
		automationExit(err)
		printIndentedJSON(data)
	},
}

// stripAutomationIDs recursively removes server-generated "id" and "uuid" keys
// so a fetched rule can be re-created as a new one.
func stripAutomationIDs(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		delete(t, "id")
		delete(t, "uuid")
		for _, child := range t {
			stripAutomationIDs(child)
		}
	case []interface{}:
		for _, child := range t {
			stripAutomationIDs(child)
		}
	}
}

func printIndentedJSON(data []byte) {
	var out interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		fmt.Println(string(data))
		return
	}
	pretty, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(string(pretty))
}

func automationExit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.AddCommand(automationCmd)
	automationCmd.AddCommand(automationListCmd)
	automationCmd.AddCommand(automationGetCmd)
	automationCmd.AddCommand(automationEnableCmd)
	automationCmd.AddCommand(automationDisableCmd)
	automationCmd.AddCommand(automationDeleteCmd)
	automationCmd.AddCommand(automationCreateCmd)
	automationCmd.AddCommand(automationCloneCmd)

	automationCreateCmd.Flags().String("from", "", "Path to a JSON file with the rule definition (required)")
	automationCloneCmd.Flags().String("name", "", "Name for the cloned rule (defaults to '<original> (copy)')")
}
